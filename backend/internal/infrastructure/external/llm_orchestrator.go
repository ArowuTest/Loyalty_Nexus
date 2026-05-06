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
	UserID          string
	SessionID       string   // for session memory lookup
	Prompt          string
	History         []string
	ToolSlug        string   // optional: "web-search-ai" | "code-helper" — routes to Pollinations
	AttachedContext string   // extracted text from uploaded file or URL — injected into system prompt
	AttachedName    string   // display name of the attached file/link (e.g. "business_plan.pdf")
}

type LLMResponse struct {
	Text      string
	Provider  LLMProvider
	Cached    bool   // true if served from Redis cache
	SessionID string // resolved UUID session ID (for frontend to persist)
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
	geminiClient     LLMClient // gemini-2.5-flash
	deepSeekClient   LLMClient
	usageTracker     UsageTracker
	chatRepo         repositories.ChatRepository
	groqDailyLimit   int
	geminiDailyLimit int
	rdb              *redis.Client
	httpClient       *http.Client // shared for Pollinations helper calls
	tavilyKey        string       // Tavily Search API key — live web search grounding
}

// ─── Constructor ─────────────────────────────────────────────────────────────

func NewLLMOrchestrator(
	g, gem, ds LLMClient,
	ut UsageTracker,
	cr repositories.ChatRepository,
	rdb *redis.Client,
	groqLim, gemLim int,
	tavilyKey string,
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
		tavilyKey:        tavilyKey,
	}
}

// ─── buildMemoryBlock constructs the [NEXUS MEMORY] context block ────────────

func (o *LLMOrchestrator) buildMemoryBlock(ctx context.Context, uid uuid.UUID, sessionID, toolSlug string) string {
	if toolSlug == "" {
		toolSlug = "general"
	}
	// 1. Fetch up to 3 past session summaries scoped to this chat mode
	summaries, _ := o.chatRepo.GetLastSummaries(ctx, uid, toolSlug, 3)

	// 2. Fetch last 5 raw messages from the current session (if sessionID given)
	var recentMsgs []repositories.ChatMessage
	if sessionID != "" {
		sid, err := uuid.Parse(sessionID)
		if err == nil {
			msgs, err := o.chatRepo.GetRecentMessages(ctx, sid, 5)
			if err == nil {
				recentMsgs = msgs
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
		switch status {
		case "limit_reached":
			reason = "rate_limit"
		case "error":
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


// ─── searchTavily calls the Tavily Search API and returns formatted results ───
// Returns an empty string (not an error) when the key is absent so callers
// can always fall back to training-data responses without crashing.
func (o *LLMOrchestrator) searchTavily(ctx context.Context, query string, maxResults int) string {
	if o.tavilyKey == "" {
		return ""
	}
	if maxResults <= 0 {
		maxResults = 5
	}
	payload := map[string]interface{}{
		"api_key":              o.tavilyKey,
		"query":                query,
		"max_results":          maxResults,
		"include_answer":       true,
		"include_raw_content":  false,
		"search_depth":         "advanced",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.tavily.com/search", bytes.NewBuffer(body))
	if err != nil {
		return ""
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		log.Printf("[Tavily] request failed: %v", err)
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)

	var result struct {
		Answer  string `json:"answer"`
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
			Score   float64 `json:"score"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		log.Printf("[Tavily] parse failed: %v", err)
		return ""
	}

	var sb strings.Builder
	sb.WriteString("[LIVE SEARCH RESULTS]\n")
	if result.Answer != "" {
		sb.WriteString("Quick answer: " + result.Answer + "\n\n")
	}
	for i, r := range result.Results {
		if i >= maxResults {
			break
		}
		sb.WriteString(fmt.Sprintf("Source %d: %s\n", i+1, r.Title))
		sb.WriteString(fmt.Sprintf("URL: %s\n", r.URL))
		content := r.Content
		if len(content) > 600 {
			content = content[:600] + "..."
		}
		sb.WriteString(fmt.Sprintf("Excerpt: %s\n\n", content))
	}
	sb.WriteString("[END SEARCH RESULTS]\n")
	sb.WriteString("Today's date: " + time.Now().UTC().Format("Monday, January 2, 2006") + "\n")
	return sb.String()
}

// ─── Chat handles general Nexus AI chat (ask-nexus, nexus-agent, research-brief, etc.) ─
func (o *LLMOrchestrator) Chat(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	_, _ = o.usageTracker.GetDailyCount(ctx, req.UserID)

	uid, _ := uuid.Parse(req.UserID)
	memoryBlock := o.buildMemoryBlock(ctx, uid, req.SessionID, req.ToolSlug)
	today := time.Now().UTC().Format("Monday, January 2, 2006")

	// ── Per-tool system prompts ────────────────────────────────────────────────
	var basePrompt string
	switch req.ToolSlug {

	case "web-search-ai":
		basePrompt = `You are Nexus Search AI. You receive live web search results and synthesise them into clear, accurate answers.

RULES:
- Answer using ONLY the [LIVE SEARCH RESULTS] provided. Do not add information from your training data unless no results were found.
- Start with a direct, confident answer in 1-2 sentences.
- Use bullet points or short paragraphs for supporting detail.
- Cite sources inline: "According to [Source Name]..." or "(Source: [Title])".
- End with a Sources section listing the URLs used.
- If the search results do not contain enough information, say so clearly and state what you do know.
- Keep responses focused and under 400 words unless the user asks for more.
- Today is ` + today + `.`

	case "code-helper", "code-pro":
		basePrompt = `You are Nexus Code — a senior software engineer and expert coding assistant.

RULES:
- Always wrap code in fenced blocks with the language name (e.g. ` + "```" + `go, ` + "```" + `python).
- Write complete, runnable code. Never use placeholder comments like // TODO.
- Include proper error handling and edge cases.
- After each code block, explain the key logic in 3-5 numbered points.
- If debugging: quote the exact broken line, explain why it fails, then show the fix.
- Match response length to question complexity. Simple questions get concise answers.`

	case "research-brief":
		basePrompt = `You are Nexus Research — a professional analyst producing structured research briefs.
Today is ` + today + `.

RULES:
- If [LIVE SEARCH RESULTS] are provided, base your answer primarily on those sources and cite them.
- If no search results are available, answer from training data and clearly state "Based on available knowledge (not live data):" at the start.
- Structure output as: Executive Summary → Key Findings → Data Points → Conclusion.
- Be specific with numbers, dates, and names. If a figure is uncertain, say so.
- Recommend 2-3 sources for the user to verify live data.
- Keep to 500 words unless depth is explicitly requested.`

	case "deep-research-brief":
		basePrompt = `You are Nexus Deep Research — producing comprehensive, multi-perspective research reports.
Today is ` + today + `.

RULES:
- If [LIVE SEARCH RESULTS] are provided, cite them extensively. Cross-reference multiple sources.
- If no results are available, state this clearly and answer from training knowledge with caveats.
- Structure: Executive Summary → Background → Key Findings (multiple perspectives) → Data & Evidence → Analysis → Conclusion → Sources.
- Include specific statistics, quotes, and named sources where available.
- Flag any information that may be outdated or requires live verification.`

	case "nexus-agent":
		basePrompt = `You are Nexus Agent — an advanced AI assistant that reasons step-by-step through complex, multi-part tasks.
Today is ` + today + `.

RULES:
- Break complex requests into clear numbered steps and execute each one.
- If [LIVE SEARCH RESULTS] are provided, use them as your primary source of facts.
- Show your reasoning process: "Step 1: ...", "Step 2: ...", etc.
- For research tasks: gather facts first, then analyse, then conclude.
- Be direct and decisive. Give recommendations, not just information.
- Acknowledge uncertainty clearly rather than guessing.`

	default: // ask-nexus and all other general tools
		basePrompt = `You are Nexus AI — a brilliant, direct personal assistant. Today is ` + today + `.

Your strengths: writing, analysis, business advice, education, general knowledge, Nigerian and African context.

RULES:
- Match response length to the question. Short questions get concise answers. Complex questions get structured detail.
- Use **bold** for key terms. Use bullet points for lists. Use paragraphs for explanations.
- For current events, live prices, recent news, or anything that changes day-to-day: clearly state your knowledge has a cutoff and recommend a live source. Do NOT invent specific numbers or dates.
- For writing tasks: produce the full draft immediately — no templates, no "here's an example".
- For factual questions you are confident about: answer directly without excessive caveats.
- Naturally incorporate Nigerian/African context when relevant (Naira, CBN, Lagos, JAMB, etc.).`
	}

	systemPrompt := basePrompt

	// Inject attached file/link context
	if req.AttachedContext != "" {
		name := req.AttachedName
		if name == "" {
			name = "attached document"
		}
		systemPrompt += "\n\n[ATTACHED DOCUMENT: " + name + "]\n" +
			"The user has attached the following. Use its content to answer accurately.\n\n" +
			req.AttachedContext + "\n[END ATTACHED DOCUMENT]"
	}

	// Inject session memory
	if memoryBlock != "" {
		systemPrompt += "\n\n" + memoryBlock + "\n\n" +
			"[MEMORY RULES]\n" +
			"- Use memory context to personalise responses (e.g. recall their name, business, or prior goals).\n" +
			"- Do NOT claim to have the full text of previous responses — you only have summaries.\n" +
			"- Always generate a complete fresh answer to the current request.\n" +
			"[END MEMORY RULES]"
	}

	// Primary: Gemini 2.5 Flash → Fallback: DeepSeek V3
	var (
		text     string
		provider LLMProvider
		err      error
	)
	text, err = o.geminiClient.Complete(ctx, systemPrompt, req.Prompt)
	if err != nil {
		log.Printf("[LLM] Gemini failed → DeepSeek: %v", err)
		go o.recordProviderUse(context.Background(), ProviderGeminiLite, false, err.Error())
		text, err = o.deepSeekClient.Complete(ctx, systemPrompt, req.Prompt)
		provider = ProviderDeepSeek
	} else {
		provider = ProviderGeminiLite
	}
	if err != nil {
		go o.recordProviderUse(context.Background(), provider, false, err.Error())
		return nil, fmt.Errorf("all LLM providers exhausted: %w", err)
	}

	_ = o.usageTracker.Increment(ctx, req.UserID)

	resolvedSessionID := o.persistMessages(ctx, uid, req.SessionID, req.ToolSlug, req.Prompt, text)
	go o.recordProviderUse(context.Background(), provider, true, "")

	return &LLMResponse{
		Text:      text,
		Provider:  provider,
		Cached:    false,
		SessionID: resolvedSessionID,
	}, nil
}

// persistMessages saves user+assistant messages to the session, creating one if needed.
// Returns the resolved session UUID string.
func (o *LLMOrchestrator) persistMessages(ctx context.Context, uid uuid.UUID, sessionID, toolSlug, userMsg, assistantMsg string) string {
	if sessionID == "" {
		return ""
	}
	if sid, err := uuid.Parse(sessionID); err == nil {
		_ = o.chatRepo.AppendMessage(ctx, sid, "user", userMsg)
		_ = o.chatRepo.AppendMessage(ctx, sid, "assistant", assistantMsg)
		return sid.String()
	}
	// Non-UUID session ID — resolve or create via chat_sessions table
	slug := toolSlug
	if slug == "" {
		slug = "general"
	}
	sess, sessErr := o.chatRepo.GetActiveSession(ctx, uid, slug)
	if sessErr != nil {
		sess, sessErr = o.chatRepo.CreateSession(ctx, uid, slug)
	}
	if sessErr == nil && sess != nil {
		_ = o.chatRepo.AppendMessage(ctx, sess.ID, "user", userMsg)
		_ = o.chatRepo.AppendMessage(ctx, sess.ID, "assistant", assistantMsg)
		return sess.ID.String()
	}
	return ""
}

func (o *LLMOrchestrator) Summarize(ctx context.Context, transcript string) (string, error) {
	systemPrompt := "You are a precision memory extraction system for an AI assistant. " +
		"Your job is to extract and compress the most important context from a conversation into a structured memory block. " +
		"The output will be injected into future conversations so the AI can seamlessly continue without the user repeating themselves. " +
		"Be specific, factual, and concise — capture names, numbers, preferences, and decisions made."

	userPrompt := "Analyse this conversation and extract a structured memory block.\n\n" +
		"Extract:\n" +
		"1. USER PROFILE: Name, location, occupation, business type (if mentioned)\n" +
		"2. GOALS & INTENT: What the user is trying to achieve (be specific)\n" +
		"3. KEY DECISIONS: Any decisions made, options chosen, or conclusions reached\n" +
		"4. IMPORTANT CONTEXT: Specific facts, numbers, preferences, constraints mentioned\n" +
		"5. OPEN THREADS: Topics started but not finished, questions asked but not fully answered\n" +
		"6. TONE & STYLE: How the user communicates (formal/casual, technical/non-technical)\n\n" +
		"Write as a concise structured paragraph (max 150 words) that gives the AI everything it needs to continue naturally.\n\n" +
		"Conversation:\n" + transcript

	return o.geminiClient.Complete(ctx, systemPrompt, userPrompt)
}

// ─── ChatWithTool ─────────────────────────────────────────────────────────────

// ChatWithTool routes a chat message to a specific Pollinations-backed tool
// (web-search-ai → gemini-search, code-helper → qwen-coder) and persists the
// exchange to the session just like a normal Chat() call.


// ─── ChatWithTool routes search-backed tools through Tavily → Gemini ─────────
//
// Routing logic:
//   web-search-ai, research-brief, deep-research-brief, nexus-agent
//     → searchTavily() for live grounding → Gemini synthesises results
//   code-helper, code-pro
//     → Pollinations Qwen-Coder (specialised code model)
//   everything else
//     → Chat() (standard Gemini)
func (o *LLMOrchestrator) ChatWithTool(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	uid, _ := uuid.Parse(req.UserID)

	switch req.ToolSlug {

	// ── Search-grounded tools: Tavily → Gemini ─────────────────────────────
	case "web-search-ai", "research-brief", "deep-research-brief", "nexus-agent":

		// Determine how many search results to fetch based on depth
		numResults := 5
		if req.ToolSlug == "deep-research-brief" {
			numResults = 8
		}

		// Step 1: Live search (returns "" gracefully if key not set)
		searchResults := o.searchTavily(ctx, req.Prompt, numResults)

		// Step 2: Build augmented prompt — inject search results or honest fallback note
		var augmentedPrompt string
		if searchResults != "" {
			augmentedPrompt = searchResults + "\n\nUser question: " + req.Prompt
		} else {
			// No Tavily key or search failed — prepend honest context
			today := time.Now().UTC().Format("Monday, January 2, 2006")
			augmentedPrompt = "[NOTE: Live web search is unavailable. Answer from training knowledge only. " +
				"Today is " + today + ". If the question requires current data, say so clearly.]\n\n" + req.Prompt
		}

		// Reuse Chat() with augmented prompt — it picks the right system prompt for this slug
		augReq := req
		augReq.Prompt = augmentedPrompt
		resp, err := o.Chat(ctx, augReq)
		if err != nil {
			return nil, err
		}
		// Label provider to show search was used
		if searchResults != "" {
			resp.Provider = "TAVILY+GEMINI"
		}
		return resp, nil

	// ── Code tools: Pollinations Qwen-Coder ───────────────────────────────
	case "code-helper", "code-pro":
		sk := os.Getenv("POLLINATIONS_SECRET_KEY")
		if sk == "" {
			// No Pollinations key — fall back to Gemini with code prompt
			return o.Chat(ctx, req)
		}

		attachedBlock := ""
		if req.AttachedContext != "" {
			name := req.AttachedName
			if name == "" {
				name = "attached document"
			}
			attachedBlock = "\n\n[ATTACHED DOCUMENT: " + name + "]\n" +
				req.AttachedContext + "\n[END ATTACHED DOCUMENT]"
		}

		payload := map[string]interface{}{
			"model": "qwen-coder",
			"messages": []map[string]interface{}{
				{"role": "system", "content": `You are Nexus Code — a senior software engineer and expert coding assistant.
RULES:
- Always wrap code in fenced blocks with the language name.
- Write complete, runnable code. Never use placeholder comments.
- Include proper error handling and edge cases.
- After each code block, explain the key logic in 3-5 numbered points.
- If debugging: quote the broken line, explain why it fails, then show the fix.`},
				{"role": "user", "content": req.Prompt + attachedBlock},
			},
		}

		text, err := o.callPollinationsChat(ctx, sk, payload)
		if err != nil {
			log.Printf("[LLM] Qwen-Coder failed → Gemini fallback: %v", err)
			go o.recordProviderUse(context.Background(), "POLLINATIONS_QWEN", false, err.Error())
			return o.Chat(ctx, req)
		}

		resolvedSessionID := o.persistMessages(ctx, uid, req.SessionID, req.ToolSlug, req.Prompt, text)
		go o.recordProviderUse(context.Background(), "POLLINATIONS_QWEN", true, "")

		return &LLMResponse{
			Text:      text,
			Provider:  "POLLINATIONS_QWEN",
			SessionID: resolvedSessionID,
		}, nil

	// ── All other slugs ────────────────────────────────────────────────────
	default:
		return o.Chat(ctx, req)
	}
}


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
		return "", fmt.Errorf("pollinations chat request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("pollinations chat %d: %s", resp.StatusCode, string(raw[:min(300, len(raw))]))
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
		return "", fmt.Errorf("pollinations chat parse: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("pollinations chat API error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("pollinations chat: no choices returned")
	}
	return parsed.Choices[0].Message.Content, nil
}

func min(a, b int) int {
	if a < b { return a }
	return b
}

// ─── Daily chat usage counter (Redis) ────────────────────────────────────────

// chatDailyKey returns the Redis key for the user's daily chat message counter.
// Key expires at midnight UTC so the count resets each day.
func chatDailyKey(uid string) string {
	return fmt.Sprintf("nexus:chat:daily:%s:%s", uid, time.Now().UTC().Format("2006-01-02"))
}

// IncrDailyChatCount atomically increments the user's daily message counter
// and sets TTL to 48 h (so counts survive midnight by one day).
// Returns the new count after increment.
func (o *LLMOrchestrator) IncrDailyChatCount(ctx context.Context, uid string) int {
	if o.rdb == nil {
		return 0
	}
	key := chatDailyKey(uid)
	count, err := o.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0
	}
	// Set TTL on first increment; ignore error (key may already have TTL)
	if count == 1 {
		_ = o.rdb.Expire(ctx, key, 48*time.Hour)
	}
	return int(count)
}

// GetDailyChatCount returns the current daily message count for the user.
func (o *LLMOrchestrator) GetDailyChatCount(ctx context.Context, uid string) int {
	if o.rdb == nil {
		return 0
	}
	key := chatDailyKey(uid)
	val, err := o.rdb.Get(ctx, key).Int()
	if err != nil {
		return 0
	}
	return val
}

// ─── SaveMessage ─────────────────────────────────────────────────────────────
func (o *LLMOrchestrator) SaveMessage(ctx context.Context, sessionID uuid.UUID, role, content string) error {
	return o.chatRepo.AppendMessage(ctx, sessionID, role, content)
}

// ─── GetChatHistory ──────────────────────────────────────────────────────────
// GetChatHistory returns the active session ID and all messages for the given
// user + toolSlug, so the frontend can restore the chat UI on page load.
func (o *LLMOrchestrator) GetChatHistory(ctx context.Context, userID, toolSlug string) (sessionID string, messages []repositories.ChatMessage, err error) {
	uid, parseErr := uuid.Parse(userID)
	if parseErr != nil {
		return "", nil, fmt.Errorf("invalid user id")
	}
	if toolSlug == "" {
		toolSlug = "general"
	}
	sess, sessErr := o.chatRepo.GetActiveSession(ctx, uid, toolSlug)
	if sessErr != nil {
		// No active session — return empty history (not an error)
		return "", nil, nil
	}
	msgs, msgErr := o.chatRepo.GetSessionMessages(ctx, sess.ID)
	if msgErr != nil {
		return sess.ID.String(), nil, nil
	}
	return sess.ID.String(), msgs, nil
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
		"model": "llama-3.3-70b-versatile",
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
	defer func() { _ = resp.Body.Close() }()

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

// ─── GeminiAdapter (gemini-2.5-flash) ───────────────────────────────────────

type GeminiAdapter struct {
	apiKey string
	client *http.Client
}

func NewGeminiAdapter(apiKey string) *GeminiAdapter {
	// 120s timeout — website generation via Gemini 2.5 Flash can take 45-90s for large HTML
	return &GeminiAdapter{apiKey: apiKey, client: &http.Client{Timeout: 120 * time.Second}}
}

func (a *GeminiAdapter) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=%s",
		a.apiKey,
	)

	payload := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]string{{"text": systemPrompt}},
		},
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": userPrompt}}},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": 65536,
			"temperature":     0.85,
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
	defer func() { _ = resp.Body.Close() }()

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

// CompleteWithImages sends a multimodal prompt (text + base64 images) to Gemini.
// Used by the website builder to pass product photos alongside the prompt.
func (a *GeminiAdapter) CompleteWithImages(ctx context.Context, systemPrompt, userPrompt string, base64Images []string) (string, error) {
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=%s",
		a.apiKey,
	)
	// Build parts: text prompt + inline image data
	parts := []map[string]interface{}{
		{"text": userPrompt},
	}
	for _, b64 := range base64Images {
		// Each image is a base64-encoded JPEG
		parts = append(parts, map[string]interface{}{
			"inlineData": map[string]string{
				"mimeType": "image/jpeg",
				"data":     b64,
			},
		})
	}
	payload := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]string{{"text": systemPrompt}},
		},
		"contents": []map[string]interface{}{
			{"parts": parts},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": 65536,
			"temperature":     0.85,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("gemini multimodal marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("gemini multimodal request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini multimodal http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
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
		return "", fmt.Errorf("gemini multimodal decode: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("gemini multimodal API error %d: %s", result.Error.Code, result.Error.Message)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini multimodal: no content returned")
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
	defer func() { _ = resp.Body.Close() }()

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

// ─── GrokAdapter (xAI Grok for image/video generation) ──────────────────────

type GrokAdapter struct {
	apiKey string
	client *http.Client
}

func NewGrokAdapter(apiKey string) *GrokAdapter {
	return &GrokAdapter{apiKey: apiKey, client: &http.Client{Timeout: 90 * time.Second}}
}

// GenerateImage calls Grok Aurora image generation API.
// Official docs: https://docs.x.ai/developers/model-capabilities/images/generation
// The only valid model is "grok-imagine-image".
// resolution: "2k" = $0.07/image (high quality), "1k" = $0.02/image (standard).
// The second parameter accepts "2k", "1k", or legacy model-name strings which are mapped to resolution.
func (a *GrokAdapter) GenerateImage(ctx context.Context, prompt string, resolution string) (string, error) {
	// Normalise legacy model-name strings to resolution values
	switch resolution {
	case "", "grok-imagine-image-pro":
		resolution = "2k" // high quality, $0.07/image
	case "grok-imagine-image":
		resolution = "1k" // standard, $0.02/image
	case "1k", "2k":
		// already correct
	default:
		resolution = "2k" // safe default
	}

	payload := map[string]interface{}{
		"model":      "grok-imagine-image",
		"prompt":     prompt,
		"n":          1, // generate 1 image
		"resolution": resolution,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("grok image marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.x.ai/v1/images/generations", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("grok image new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("grok image http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Data []struct {
			URL string `json:"url"`
		} `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("grok image decode: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("grok image API error: %s", result.Error.Message)
	}
	if len(result.Data) == 0 {
		return "", fmt.Errorf("grok image: no images returned")
	}
	return result.Data[0].URL, nil
}

// ComposeImages calls Grok Aurora (grok-imagine-image) with up to 5 reference images
// for Whisk-style subject+scene+style composition.
// API: POST /v1/images/generations with image_urls array (up to 5 images).
// Docs: https://docs.x.ai/developers/model-capabilities/images/generation#editing-with-multiple-images
func (a *GrokAdapter) ComposeImages(ctx context.Context, prompt string, imageURLs []string, aspectRatio string) (string, error) {
	if len(imageURLs) == 0 {
		return "", fmt.Errorf("grok compose: at least one image URL required")
	}
	// Cap at 5 (API limit for multi-image editing)
	if len(imageURLs) > 5 {
		imageURLs = imageURLs[:5]
	}
	if prompt == "" {
		prompt = "Compose these reference images into a single cohesive image"
	}
	payload := map[string]interface{}{
		"model":      "grok-imagine-image",
		"prompt":     prompt,
		"image_urls": imageURLs,
		"n":          1,
	}
	if aspectRatio != "" {
		payload["aspect_ratio"] = aspectRatio
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("grok compose marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.x.ai/v1/images/generations", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("grok compose new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("grok compose http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("grok compose %d: %s", resp.StatusCode, truncateGrokStr(string(raw), 300))
	}
	var result struct {
		Data []struct {
			URL string `json:"url"`
		} `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("grok compose decode: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("grok compose API error: %s", result.Error.Message)
	}
	if len(result.Data) == 0 || result.Data[0].URL == "" {
		return "", fmt.Errorf("grok compose: no image URL in response")
	}
	return result.Data[0].URL, nil
}

// GrokVideoRequest holds all parameters for a Grok Imagine video generation call.
// Exactly one of the mode fields should be set:
//   - TextToVideo:      prompt only (no image/video URL)
//   - ImageToVideo:     ImageURL set  → image becomes first frame
//   - ReferenceVideo:   ReferenceImageURLs set (1-7) → reference-guided generation
//   - VideoEdit:        VideoURL set + prompt → edit existing video
//   - VideoExtend:      VideoURL set + Extend=true → extend existing video
type GrokVideoRequest struct {
	Prompt              string
	ImageURL            string   // image-to-video: source image
	VideoURL            string   // video-edit / video-extend: source video
	ReferenceImageURLs  []string // reference-image mode: 1-7 reference images
	Duration            int      // seconds (2-15 for generation, 2-10 for extension)
	AspectRatio         string   // "16:9", "9:16", "1:1"
	Resolution          string   // "480p" or "720p"
	Extend              bool     // true = video extension mode
}

// GenerateVideo submits a Grok Imagine video request and polls until done.
// It supports all five modes: text-to-video, image-to-video, reference-image,
// video-edit, and video-extend. The ctx deadline controls the total wait time.
func (a *GrokAdapter) GenerateVideo(ctx context.Context, req GrokVideoRequest) (string, error) {
	if req.Prompt == "" && req.VideoURL == "" {
		return "", fmt.Errorf("grok video: prompt is required")
	}

	// ── Build payload ─────────────────────────────────────────────────────────
	payload := map[string]interface{}{
		"model":  "grok-imagine-video",
		"prompt": req.Prompt,
	}

	// Duration (only for generation modes, not video editing)
	if req.VideoURL == "" || req.Extend {
		dur := req.Duration
		if dur <= 0 {
			dur = 6 // API default
		}
		payload["duration"] = dur
	}

	// Aspect ratio and resolution (not supported in video editing mode)
	if req.VideoURL == "" || req.Extend {
		ar := req.AspectRatio
		if ar == "" {
			ar = "16:9"
		}
		payload["aspect_ratio"] = ar
		res := req.Resolution
		if res == "" {
			res = "720p"
		}
		payload["resolution"] = res
	}

	// Mode-specific fields
	switch {
	case req.Extend && req.VideoURL != "":
		// Video extension mode
		payload["video_url"] = req.VideoURL
	case req.VideoURL != "":
		// Video editing mode
		payload["video_url"] = req.VideoURL
	case len(req.ReferenceImageURLs) > 0:
		// Reference-image mode (1-7 images)
		refs := req.ReferenceImageURLs
		if len(refs) > 7 {
			refs = refs[:7]
		}
		payload["reference_image_urls"] = refs
	case req.ImageURL != "":
		// Image-to-video mode
		payload["image_url"] = req.ImageURL
	}

	// ── Step 1: Submit generation request ────────────────────────────────────
	// Select the correct endpoint based on mode:
	//   /v1/videos/extensions  → video extension (Extend=true)
	//   /v1/videos/generations → all other modes (text, image, reference, edit)
	submitEndpoint := "https://api.x.ai/v1/videos/generations"
	if req.Extend && req.VideoURL != "" {
		submitEndpoint = "https://api.x.ai/v1/videos/extensions"
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("grok video marshal: %w", err)
	}
	// Use a long-lived client for the initial request (up to 30s)
	initClient := &http.Client{Timeout: 30 * time.Second}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		submitEndpoint, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("grok video new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	initResp, err := initClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("grok video submit: %w", err)
	}
	defer func() {
		if err := initResp.Body.Close(); err != nil {
			log.Printf("[GrokAdapter] initResp body close: %v", err)
		}
	}()
	initBody, _ := io.ReadAll(initResp.Body)
	if initResp.StatusCode != http.StatusOK && initResp.StatusCode != http.StatusCreated && initResp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("grok video submit %d: %s", initResp.StatusCode, truncateGrokStr(string(initBody), 300))
	}

	var submitResult struct {
		RequestID string `json:"request_id"`
		// Some responses return the video directly if already done
		Status string `json:"status"`
		Video  struct {
			URL string `json:"url"`
		} `json:"video"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(initBody, &submitResult); err != nil {
		return "", fmt.Errorf("grok video submit parse: %w", err)
	}
	if submitResult.Error != nil {
		return "", fmt.Errorf("grok video API error: %s", submitResult.Error.Message)
	}
	// If already done (rare but possible)
	if submitResult.Status == "done" && submitResult.Video.URL != "" {
		return submitResult.Video.URL, nil
	}
	if submitResult.RequestID == "" {
		return "", fmt.Errorf("grok video: no request_id in response: %s", truncateGrokStr(string(initBody), 200))
	}

	// ── Step 2: Poll for completion ───────────────────────────────────────────
	// Grok videos take up to several minutes. We poll every 5s for up to 8 minutes.
	pollURL := fmt.Sprintf("https://api.x.ai/v1/videos/%s", submitResult.RequestID)
	pollClient := &http.Client{Timeout: 15 * time.Second}
	const maxAttempts = 96 // 96 × 5s = 8 minutes
	for attempt := 0; attempt < maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("grok video: context cancelled while polling")
		case <-time.After(5 * time.Second):
		}

		pollReq, err := http.NewRequestWithContext(ctx, http.MethodGet, pollURL, nil)
		if err != nil {
			log.Printf("[GrokAdapter] poll request build error: %v", err)
			continue
		}
		pollReq.Header.Set("Authorization", "Bearer "+a.apiKey)

		pollResp, err := pollClient.Do(pollReq)
		if err != nil {
			log.Printf("[GrokAdapter] poll attempt %d error: %v", attempt+1, err)
			continue
		}
		pollBody, _ := io.ReadAll(pollResp.Body)
		if err := pollResp.Body.Close(); err != nil {
			log.Printf("[GrokAdapter] pollResp body close: %v", err)
		}

		var pollResult struct {
			Status string `json:"status"`
			Video  struct {
				URL string `json:"url"`
			} `json:"video"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(pollBody, &pollResult); err != nil {
			log.Printf("[GrokAdapter] poll parse error attempt %d: %v", attempt+1, err)
			continue
		}
		if pollResult.Error != nil {
			return "", fmt.Errorf("grok video poll error: %s", pollResult.Error.Message)
		}

		switch pollResult.Status {
		case "done":
			if pollResult.Video.URL == "" {
				return "", fmt.Errorf("grok video: done but no URL")
			}
			log.Printf("[GrokAdapter] video ready after %d polls: %s", attempt+1, pollResult.Video.URL)
			return pollResult.Video.URL, nil
		case "failed":
			return "", fmt.Errorf("grok video: generation failed")
		case "expired":
			return "", fmt.Errorf("grok video: request expired")
		default: // "pending" or unknown
			log.Printf("[GrokAdapter] poll attempt %d: status=%s", attempt+1, pollResult.Status)
		}
	}
	return "", fmt.Errorf("grok video: timed out after 8 minutes (request_id=%s)", submitResult.RequestID)
}

// truncateGrokStr truncates a string for error messages.
func truncateGrokStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
