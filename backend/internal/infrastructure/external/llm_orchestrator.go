package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"loyalty-nexus/internal/domain/repositories"
)

// ─── Redis key helpers ────────────────────────────────────────────────────────

func providerStatusKey(p LLMProvider) string    { return "nexus:ai:provider:" + string(p) + ":status" }
func providerLastUsedKey(p LLMProvider) string  { return "nexus:ai:provider:" + string(p) + ":last_used_at" }
func providerLastErrKey(p LLMProvider) string   { return "nexus:ai:provider:" + string(p) + ":last_error" }
func providerReqTodayKey(p LLMProvider) string  { return "nexus:ai:provider:" + string(p) + ":requests_today" }
const activeProviderKey    = "nexus:ai:active_chat_provider"
const providerSwitchLogKey = "nexus:ai:provider_switch_log"

// secondsUntilMidnightUTC returns the number of seconds until the next UTC midnight.
func secondsUntilMidnightUTC() time.Duration {
	now := time.Now().UTC()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	return time.Until(midnight)
}

// ─── Provider constants ──────────────────────────────────────────────────────

type LLMProvider string

const (
	ProviderGroq       LLMProvider = "GROQ"
	ProviderGeminiLite LLMProvider = "GEMINI_LITE"
	ProviderDeepSeek   LLMProvider = "DEEPSEEK"
)

// ─── Request / Response ──────────────────────────────────────────────────────

type LLMRequest struct {
	UserID    string
	SessionID string   // for session memory lookup
	Prompt    string
	History   []string
	ToolSlug  string   // optional: "web-search-ai" | "code-helper" — routes to Pollinations
}

type LLMResponse struct {
	Text     string
	Provider LLMProvider
	Cached   bool // true if served from Redis cache
}

// ─── Interfaces ──────────────────────────────────────────────────────────────

type LLMClient interface {
	Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

type UsageTracker interface {
	GetDailyCount(ctx context.Context, userID string) (int, error)
	Increment(ctx context.Context, userID string) error
}

// ─── LLMOrchestrator struct ─────────────────────────────────────────────────

type LLMOrchestrator struct {
	groqClient       LLMClient
	geminiClient     LLMClient // gemini-2.0-flash-lite
	deepSeekClient   LLMClient
	usageTracker     UsageTracker
	chatRepo         repositories.ChatRepository
	groqDailyLimit   int
	geminiDailyLimit int
	rdb              *redis.Client
	httpClient       *http.Client // shared for Pollinations helper calls
}

// ─── Constructor ─────────────────────────────────────────────────────────────

func NewLLMOrchestrator(
	g, gem, ds LLMClient,
	ut UsageTracker,
	cr repositories.ChatRepository,
	rdb *redis.Client,
	groqLim, gemLim int,
) *LLMOrchestrator {
	return &LLMOrchestrator{
		groqClient:       g,
		geminiClient:     gem,
		deepSeekClient:   ds,
		usageTracker:     ut,
		chatRepo:         cr,
		rdb:              rdb,
		groqDailyLimit:   groqLim,
		geminiDailyLimit: gemLim,
		httpClient:       &http.Client{Timeout: 60 * time.Second},
	}
}

// ─── buildMemoryBlock constructs the [NEXUS MEMORY] context block ────────────

func (o *LLMOrchestrator) buildMemoryBlock(ctx context.Context, uid uuid.UUID, sessionID string) string {
	// 1. Fetch up to 3 past session summaries
	summaries, _ := o.chatRepo.GetLastSummaries(ctx, uid, 3)

	// 2. Fetch last 5 raw messages from the current session (if sessionID given)
	var recentMsgs []repositories.ChatMessage
	if sessionID != "" {
		sid, err := uuid.Parse(sessionID)
		if err == nil {
			all, err := o.chatRepo.GetSessionMessages(ctx, sid)
			if err == nil && len(all) > 0 {
				// Take the last 5
				start := len(all) - 5
				if start < 0 {
					start = 0
				}
				recentMsgs = all[start:]
			}
		}
	}

	if len(summaries) == 0 && len(recentMsgs) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("[NEXUS MEMORY]\n")

	if len(summaries) > 0 {
		sb.WriteString("Previous sessions summary:\n")
		labels := []string{
			"Session (older)",
			"Session (recent)",
			"Session (latest)",
		}
		// summaries come oldest-first from GetLastSummaries; assign labels accordingly
		for i, s := range summaries {
			label := labels[0]
			if i < len(labels) {
				label = labels[i]
			}
			sb.WriteString(fmt.Sprintf("- %s: %s\n", label, s))
		}
	}

	if len(recentMsgs) > 0 {
		sb.WriteString("Last messages:\n")
		for _, m := range recentMsgs {
			switch strings.ToLower(m.Role) {
			case "user":
				sb.WriteString(fmt.Sprintf("User: %q\n", m.Content))
			case "assistant":
				sb.WriteString(fmt.Sprintf("Nexus: %q\n", m.Content))
			default:
				sb.WriteString(fmt.Sprintf("%s: %q\n", m.Role, m.Content))
			}
		}
	}

	sb.WriteString("[END NEXUS MEMORY]")
	return sb.String()
}

// ─── Provider health helpers ──────────────────────────────────────────────────

// recordProviderUse writes health-tracking keys to Redis after each LLM call.
// It runs synchronously but with a short-circuit timeout so it never blocks Chat.
func (o *LLMOrchestrator) recordProviderUse(ctx context.Context, provider LLMProvider, success bool, errMsg string) {
	rCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	now := time.Now().UTC()
	ts := fmt.Sprintf("%d", now.Unix())

	// Determine status
	status := "ok"
	if !success {
		lower := strings.ToLower(errMsg)
		if strings.Contains(lower, "429") || strings.Contains(lower, "rate") || strings.Contains(lower, "limit") {
			status = "limit_reached"
		} else {
			status = "error"
		}
	}

	pipe := o.rdb.Pipeline()
	pipe.Set(rCtx, providerStatusKey(provider), status, 0)
	pipe.Set(rCtx, providerLastUsedKey(provider), ts, 0)

	if !success && errMsg != "" {
		pipe.Set(rCtx, providerLastErrKey(provider), errMsg, 0)
	}

	if success {
		pipe.Incr(rCtx, providerReqTodayKey(provider))
		pipe.ExpireAt(rCtx, providerReqTodayKey(provider), time.Now().UTC().Add(secondsUntilMidnightUTC()))
	}

	pipe.Set(rCtx, activeProviderKey, string(provider), 0)
	_, _ = pipe.Exec(rCtx)

	// If provider changed, push a switch log entry
	prev, err := o.rdb.Get(rCtx, activeProviderKey).Result()
	if err == nil && prev != string(provider) {
		type switchEntry struct {
			From   string `json:"from"`
			To     string `json:"to"`
			Reason string `json:"reason"`
			TS     int64  `json:"ts"`
		}
		reason := "provider_change"
		if status == "limit_reached" {
			reason = "rate_limit"
		} else if status == "error" {
			reason = "error"
		}
		entry := switchEntry{From: prev, To: string(provider), Reason: reason, TS: now.Unix()}
		entryBytes, jsonErr := json.Marshal(entry)
		if jsonErr == nil {
			pipe2 := o.rdb.Pipeline()
			pipe2.LPush(rCtx, providerSwitchLogKey, string(entryBytes))
			pipe2.LTrim(rCtx, providerSwitchLogKey, 0, 49) // keep last 50
			_, _ = pipe2.Exec(rCtx)
		}
	}
}

// RecordStudioToolUse records per-tool usage stats in Redis.
// Intended to be called as a fire-and-forget goroutine from AIStudioOrchestrator.
func (o *LLMOrchestrator) RecordStudioToolUse(ctx context.Context, toolSlug, provider string) {
	rCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	base := "nexus:ai:studio:" + toolSlug
	now := time.Now().UTC()
	ts := fmt.Sprintf("%d", now.Unix())

	pipe := o.rdb.Pipeline()
	pipe.Incr(rCtx, base+":requests_today")
	pipe.ExpireAt(rCtx, base+":requests_today", time.Now().UTC().Add(secondsUntilMidnightUTC()))
	pipe.Set(rCtx, base+":last_provider", provider, 0)
	pipe.Set(rCtx, base+":last_used_at", ts, 0)
	_, _ = pipe.Exec(rCtx)
}

// ─── Chat ────────────────────────────────────────────────────────────────────

func (o *LLMOrchestrator) Chat(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	// 1. Get current daily usage count
	dailyCount, _ := o.usageTracker.GetDailyCount(ctx, req.UserID)

	// 2. Build session memory context
	uid, _ := uuid.Parse(req.UserID)
	memoryBlock := o.buildMemoryBlock(ctx, uid, req.SessionID)

	// 3. Build system prompt
	systemPrompt := "You are Nexus, a helpful AI assistant for Loyalty Nexus subscribers in Nigeria. " +
		"Be concise, practical and locally aware. You understand Nigerian English, culture, and context."
	if memoryBlock != "" {
		systemPrompt += "\n\n" + memoryBlock
	}

	// 4. Route: Groq → Gemini Flash-Lite → DeepSeek
	var (
		text     string
		provider LLMProvider
		err      error
	)

	switch {
	case dailyCount < o.groqDailyLimit:
		text, err = o.groqClient.Complete(ctx, systemPrompt, req.Prompt)
		if err != nil {
			log.Printf("[LLM] Groq failed (count=%d) → falling through to Gemini: %v", dailyCount, err)
			go o.recordProviderUse(context.Background(), ProviderGroq, false, err.Error())
			// Fall through to Gemini
			text, err = o.geminiClient.Complete(ctx, systemPrompt, req.Prompt)
			if err != nil {
				log.Printf("[LLM] Gemini failed → DeepSeek: %v", err)
				go o.recordProviderUse(context.Background(), ProviderGeminiLite, false, err.Error())
				text, err = o.deepSeekClient.Complete(ctx, systemPrompt, req.Prompt)
				provider = ProviderDeepSeek
			} else {
				provider = ProviderGeminiLite
			}
		} else {
			provider = ProviderGroq
		}

	case dailyCount < o.geminiDailyLimit:
		text, err = o.geminiClient.Complete(ctx, systemPrompt, req.Prompt)
		if err != nil {
			log.Printf("[LLM] Gemini failed (count=%d) → DeepSeek: %v", dailyCount, err)
			go o.recordProviderUse(context.Background(), ProviderGeminiLite, false, err.Error())
			text, err = o.deepSeekClient.Complete(ctx, systemPrompt, req.Prompt)
			provider = ProviderDeepSeek
		} else {
			provider = ProviderGeminiLite
		}

	default:
		text, err = o.deepSeekClient.Complete(ctx, systemPrompt, req.Prompt)
		provider = ProviderDeepSeek
	}

	if err != nil {
		// Record the failure for the last-attempted provider
		go o.recordProviderUse(context.Background(), provider, false, err.Error())
		return nil, fmt.Errorf("all LLM providers exhausted: %w", err)
	}

	// 5. Post-success: increment usage counter & persist messages
	_ = o.usageTracker.Increment(ctx, req.UserID)

	if req.SessionID != "" {
		sid, parseErr := uuid.Parse(req.SessionID)
		if parseErr == nil {
			_ = o.chatRepo.AppendMessage(ctx, sid, "user", req.Prompt)
			_ = o.chatRepo.AppendMessage(ctx, sid, "assistant", text)
		}
	}

	// Record successful provider use (non-blocking)
	go o.recordProviderUse(context.Background(), provider, true, "")

	return &LLMResponse{
		Text:     text,
		Provider: provider,
		Cached:   false,
	}, nil
}

// ─── Summarize ───────────────────────────────────────────────────────────────

// Summarize sends a full conversation transcript to Gemini Flash-Lite and returns
// a structured memory paragraph the AI can use to continue the conversation.
func (o *LLMOrchestrator) Summarize(ctx context.Context, transcript string) (string, error) {
	systemPrompt := "You are a helpful assistant that creates structured conversation summaries for an AI memory system."

	userPrompt := `Summarise this conversation. Extract:
1. INTENT: What was the user trying to achieve?
2. PERSONAL CONTEXT: Any personal details shared (name, business, location, preferences)
3. TOPICS: Key subjects discussed and their outcomes
4. NEXT STEPS: Did the user mention plans to continue or return?
Write as a structured paragraph the AI can use to seamlessly continue the conversation.

Conversation:
` + transcript

	return o.geminiClient.Complete(ctx, systemPrompt, userPrompt)
}

// ─── ChatWithTool ─────────────────────────────────────────────────────────────

// ChatWithTool routes a chat message to a specific Pollinations-backed tool
// (web-search-ai → gemini-search, code-helper → qwen-coder) and persists the
// exchange to the session just like a normal Chat() call.
func (o *LLMOrchestrator) ChatWithTool(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	sk := os.Getenv("POLLINATIONS_SECRET_KEY")
	if sk == "" {
		// No Pollinations key — graceful fallback to standard chat
		return o.Chat(ctx, req)
	}

	var (
		payload map[string]interface{}
		providerName LLMProvider
	)

	switch req.ToolSlug {
	case "web-search-ai":
		payload = map[string]interface{}{
			"model": "openai",
			"messages": []map[string]interface{}{
				{"role": "system", "content": "You are Nexus AI with real-time web search access. Provide current, accurate information with sources when relevant. Be concise and locally aware of Nigerian context."},
				{"role": "user", "content": req.Prompt},
			},
			"search": true,
		}
		providerName = "POLLINATIONS_SEARCH"
	case "code-helper":
		payload = map[string]interface{}{
			"model": "qwen-coder",
			"messages": []map[string]interface{}{
				{"role": "system", "content": "You are an expert programmer and coding assistant. Write clean, well-commented code. Explain your solution clearly. Format code blocks with proper markdown."},
				{"role": "user", "content": req.Prompt},
			},
		}
		providerName = "POLLINATIONS_QWEN"
	default:
		return o.Chat(ctx, req)
	}

	text, err := o.callPollinationsChat(ctx, sk, payload)
	if err != nil {
		log.Printf("[LLM] ChatWithTool %s failed: %v — falling back to general chat", req.ToolSlug, err)
		go o.recordProviderUse(context.Background(), providerName, false, err.Error())
		return o.Chat(ctx, req)
	}

	// Persist messages to session
	if req.SessionID != "" {
		sid, parseErr := uuid.Parse(req.SessionID)
		if parseErr == nil {
			_ = o.chatRepo.AppendMessage(ctx, sid, "user", req.Prompt)
			_ = o.chatRepo.AppendMessage(ctx, sid, "assistant", text)
		}
	}
	go o.recordProviderUse(context.Background(), providerName, true, "")

	return &LLMResponse{Text: text, Provider: providerName}, nil
}

// callPollinationsChat is a shared helper for Pollinations OpenAI-compatible chat.
func (o *LLMOrchestrator) callPollinationsChat(ctx context.Context, sk string, payload interface{}) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://gen.pollinations.ai/v1/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+sk)
	req.Header.Set("User-Agent", "NexusAI/1.0")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Pollinations chat request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Pollinations chat %d: %s", resp.StatusCode, string(raw[:min(300, len(raw))]))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct{ Message string `json:"message"` } `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("Pollinations chat parse: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("Pollinations chat API error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("Pollinations chat: no choices returned")
	}
	return parsed.Choices[0].Message.Content, nil
}

func min(a, b int) int {
	if a < b { return a }
	return b
}

// SaveMessage persists a single message (role: "user" or "assistant") to the chat repo.
func (o *LLMOrchestrator) SaveMessage(ctx context.Context, sessionID uuid.UUID, role, content string) error {
	return o.chatRepo.AppendMessage(ctx, sessionID, role, content)
}

// ─── GroqAdapter ─────────────────────────────────────────────────────────────

type GroqAdapter struct {
	apiKey string
	client *http.Client
}

func NewGroqAdapter(apiKey string) *GroqAdapter {
	return &GroqAdapter{apiKey: apiKey, client: &http.Client{Timeout: 30 * time.Second}}
}

func (a *GroqAdapter) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	payload := map[string]interface{}{
		"model": "llama-4-scout-17b-16e-instruct",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"max_tokens":  2048,
		"temperature": 0.7,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("groq marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("groq new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("groq http: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("groq decode: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("groq API error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("groq: no choices returned")
	}
	return result.Choices[0].Message.Content, nil
}

// ─── GeminiAdapter (gemini-2.0-flash-lite) ───────────────────────────────────

type GeminiAdapter struct {
	apiKey string
	client *http.Client
}

func NewGeminiAdapter(apiKey string) *GeminiAdapter {
	return &GeminiAdapter{apiKey: apiKey, client: &http.Client{Timeout: 30 * time.Second}}
}

func (a *GeminiAdapter) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash-lite:generateContent?key=%s",
		a.apiKey,
	)

	payload := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]string{{"text": systemPrompt}},
		},
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": userPrompt}}},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("gemini marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("gemini new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini http: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("gemini decode: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("gemini API error %d: %s", result.Error.Code, result.Error.Message)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: no content returned")
	}
	return result.Candidates[0].Content.Parts[0].Text, nil
}

// ─── DeepSeekAdapter ─────────────────────────────────────────────────────────

type DeepSeekAdapter struct {
	apiKey string
	client *http.Client
}

func NewDeepSeekAdapter(apiKey string) *DeepSeekAdapter {
	return &DeepSeekAdapter{apiKey: apiKey, client: &http.Client{Timeout: 60 * time.Second}}
}

func (a *DeepSeekAdapter) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	payload := map[string]interface{}{
		"model": "deepseek-chat",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"max_tokens":  2048,
		"temperature": 0.7,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("deepseek marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.deepseek.com/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("deepseek new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("deepseek http: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("deepseek decode: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("deepseek API error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("deepseek: no choices returned")
	}
	return result.Choices[0].Message.Content, nil
}
