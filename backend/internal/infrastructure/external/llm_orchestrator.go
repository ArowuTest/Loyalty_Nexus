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
	// Daily count retained for usage tracking; no longer used for routing.
	_, _ = o.usageTracker.GetDailyCount(ctx, req.UserID)

	// 2. Build session memory context
	uid, _ := uuid.Parse(req.UserID)
	memoryBlock := o.buildMemoryBlock(ctx, uid, req.SessionID, req.ToolSlug)

	// 3. Build system prompt
	// Determine system prompt based on tool slug (chat mode)
	var basePrompt string
	today := time.Now().UTC().Format("Monday, January 2, 2006")
	switch req.ToolSlug {
	case "web-search-ai":
		basePrompt = "You are Nexus AI, a world-class web search assistant with real-time access to the internet. " +
			"Today's date is " + today + ".\n\n" +
			"Your capabilities:\n" +
			"- You have real-time web search access. Use it to provide current, accurate, up-to-date information on any topic worldwide.\n" +
			"- You can answer questions about global news, technology, science, business, politics, culture, sports, finance, and any other domain.\n" +
			"- When the user's context suggests Nigerian or African relevance, naturally incorporate local insights (e.g., Nigerian regulations, Naira exchange rates, local market data, African perspectives).\n\n" +
			"Response format rules (MUST follow every time):\n" +
			"1. Start with a **bold one-sentence direct answer** to the question.\n" +
			"2. Follow with 2-4 short paragraphs or bullet points of supporting detail.\n" +
			"3. For factual questions: include specific numbers, dates, and names — never be vague.\n" +
			"4. For news/current events: summarise key facts + their implications.\n" +
			"5. Cite sources naturally inline (e.g., 'According to Reuters...', 'Per the World Bank...').\n" +
			"6. End with a **Sources** section listing 2-4 sources used (publication name + brief description).\n" +
			"7. Keep total response under 450 words unless the user explicitly asks for more.\n" +
			"8. If search results are unclear or outdated, say so honestly and provide your best knowledge with a caveat."
	case "code-helper":
		basePrompt = "You are Nexus Code, a world-class programming assistant and senior software engineer. " +
			"You have deep expertise in all major programming languages, frameworks, and software engineering best practices globally.\n\n" +
			"Your capabilities:\n" +
			"- Write production-quality, clean, well-commented, fully functional code in any language.\n" +
			"- Debug errors with clear explanations of the root cause and fix.\n" +
			"- Explain complex concepts in simple terms with practical examples.\n" +
			"- Suggest better approaches, patterns, and optimisations based on industry best practices.\n" +
			"- Handle web dev (React, Next.js, Node, Go, Python, SQL), mobile (Flutter, React Native, Swift, Kotlin), backend (Go, Python, Java, C#), data science (Python, R), and DevOps (Docker, CI/CD, Kubernetes).\n" +
			"- For multi-file projects, clearly label each file with a comment like `# File: main.py` or `// File: server.js`.\n\n" +
			"Response rules (MUST follow every time):\n" +
			"- ALWAYS wrap code in fenced code blocks with the language name (e.g., ```python, ```javascript, ```go).\n" +
			"- For every code block, add a brief comment at the top explaining what it does.\n" +
			"- After the code, explain the key logic in 3-5 numbered bullet points.\n" +
			"- If the user's code has a bug, quote the problematic line, explain why it's wrong, then show the corrected version.\n" +
			"- Detect the programming language from context — never ask the user to specify it unless truly ambiguous.\n" +
			"- Always include proper error handling, input validation, and edge case handling.\n" +
			"- Write complete, runnable code — never use placeholder comments like '// TODO' or '// rest of code here'."
	default:
		basePrompt = "You are Nexus AI, a brilliant and versatile personal AI assistant with world-class capabilities. " +
			"Today's date is " + today + ".\n\n" +
			"Your personality:\n" +
			"- Warm, intelligent, and direct — like a brilliant friend who gives real, specific advice rather than generic answers.\n" +
			"- You have deep knowledge across all domains: science, history, technology, business, culture, health, law, finance, education, and the arts.\n" +
			"- When the user's context suggests Nigerian or African relevance (e.g., mentions Lagos, Naira, CBN, JAMB, WAEC, NYSC), naturally incorporate local insights, examples, and context.\n\n" +
			"Your capabilities:\n" +
			"- Answer any question with depth, accuracy, and global or local context as appropriate.\n" +
			"- Write and improve: business plans, emails, CVs, cover letters, proposals, essays, social media posts, and creative content.\n" +
			"- Analyse, summarise, and explain complex documents, concepts, or situations.\n" +
			"- Give financial, legal, and health information (always recommend consulting a professional for critical decisions).\n" +
			"- Brainstorm ideas, help with decision-making, and provide strategic recommendations.\n\n" +
			"Response rules (MUST follow every time):\n" +
			"- Give complete, thorough answers — never cut off mid-thought or refuse to help unless the request is truly harmful.\n" +
			"- Use **bold** for key terms, important points, and section headers.\n" +
			"- Use bullet points for lists, numbered lists for steps, and paragraphs for explanations and analysis.\n" +
			"- Match the user's tone: casual for casual messages, formal for formal requests.\n" +
			"- Never give one-line answers to substantive questions — always provide genuine value and depth.\n" +
			"- For writing tasks: produce the full, complete draft — never write a partial version or say 'here's a template'.\n" +
			"- For analysis tasks: go beyond surface-level observations to provide genuine insight and actionable conclusions."
	}
	systemPrompt := basePrompt

	// Inject attached file/link context if provided
	if req.AttachedContext != "" {
		name := req.AttachedName
		if name == "" {
			name = "attached document"
		}
		systemPrompt += "\n\n[ATTACHED DOCUMENT: " + name + "]\n" +
			"The user has attached the following document for you to read and reason over. " +
			"Use its content to answer their question accurately and specifically.\n\n" +
			req.AttachedContext + "\n[END ATTACHED DOCUMENT]"
	}

	if memoryBlock != "" {
		systemPrompt += "\n\n" + memoryBlock + "\n\n" +
			"[MEMORY USAGE RULES]\n" +
			"- The [NEXUS MEMORY] block above contains SUMMARIES of past sessions — NOT the full content of previous responses.\n" +
			"- Do NOT claim to have the full text of any previous response. You only have a summary.\n" +
			"- Do NOT say 'I already drafted this for you' or 'I created this in our previous session' — you cannot re-share something you only have a summary of.\n" +
			"- DO use the memory context to personalise your response (e.g., 'I see you've been working on a food delivery startup — here's a fresh detailed plan:').\n" +
			"- ALWAYS generate a complete, fresh, full-length answer to the user's current request.\n" +
			"[END MEMORY USAGE RULES]"
	}

	// 4. Route: Gemini 2.5 Flash (primary) → DeepSeek V3 (fallback)
	// Groq removed: model availability and response accuracy were unreliable.
	var (
		text     string
		provider LLMProvider
		err      error
	)

	// Primary: Gemini 2.5 Flash
	text, err = o.geminiClient.Complete(ctx, systemPrompt, req.Prompt)
	if err != nil {
		log.Printf("[LLM] Gemini failed → DeepSeek: %v", err)
		go o.recordProviderUse(context.Background(), ProviderGeminiLite, false, err.Error())
		// Fallback: DeepSeek V3
		text, err = o.deepSeekClient.Complete(ctx, systemPrompt, req.Prompt)
		provider = ProviderDeepSeek
	} else {
		provider = ProviderGeminiLite
	}

	if err != nil {
		// Record the failure for the last-attempted provider
		go o.recordProviderUse(context.Background(), provider, false, err.Error())
		return nil, fmt.Errorf("all LLM providers exhausted: %w", err)
	}

	// 5. Post-success: increment usage counter & persist messages
	_ = o.usageTracker.Increment(ctx, req.UserID)

	// Resolve session UUID: if req.SessionID is already a valid UUID, use it;
	// otherwise get-or-create a session in chat_sessions by (userID, toolSlug).
	// This ensures messages are always persisted even when the frontend sends
	// non-UUID session IDs like "sess_general_1234567890".
	var resolvedSessionID string
	if req.SessionID != "" {
		if sid, parseErr := uuid.Parse(req.SessionID); parseErr == nil {
			// Already a valid UUID — use it directly
			_ = o.chatRepo.AppendMessage(ctx, sid, "user", req.Prompt)
			_ = o.chatRepo.AppendMessage(ctx, sid, "assistant", text)
			resolvedSessionID = sid.String()
		} else {
			// Not a UUID — resolve via chat_sessions table
			toolSlug := req.ToolSlug
			if toolSlug == "" {
				toolSlug = "general"
			}
			sess, sessErr := o.chatRepo.GetActiveSession(ctx, uid, toolSlug)
			if sessErr != nil {
				// No active session — create one
				sess, sessErr = o.chatRepo.CreateSession(ctx, uid, toolSlug)
			}
			if sessErr == nil && sess != nil {
				_ = o.chatRepo.AppendMessage(ctx, sess.ID, "user", req.Prompt)
				_ = o.chatRepo.AppendMessage(ctx, sess.ID, "assistant", text)
				resolvedSessionID = sess.ID.String()
			}
		}
	}

	// Record successful provider use (non-blocking)
	go o.recordProviderUse(context.Background(), provider, true, "")

	return &LLMResponse{
		Text:      text,
		Provider:  provider,
		Cached:    false,
		SessionID: resolvedSessionID,
	}, nil
}

// ─── Summarize ───────────────────────────────────────────────────────────────

// Summarize sends a full conversation transcript to Gemini Flash-Lite and returns
// a structured memory paragraph the AI can use to continue the conversation.
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

	// Build attached context block (shared across tool modes)
	attachedBlock := ""
	if req.AttachedContext != "" {
		name := req.AttachedName
		if name == "" {
			name = "attached document"
		}
		attachedBlock = "\n\n[ATTACHED DOCUMENT: " + name + "]\n" +
			"The user has attached the following document. Read it carefully and use its content to answer their question.\n\n" +
			req.AttachedContext + "\n[END ATTACHED DOCUMENT]"
	}

	switch req.ToolSlug {
	case "web-search-ai":
			today := time.Now().UTC().Format("Monday, January 2, 2006")
			payload = map[string]interface{}{
				"model": "openai",
				"messages": []map[string]interface{}{
					{"role": "system", "content": "You are Nexus AI, a world-class web search assistant with real-time access to the internet. " +
						"Today's date is " + today + ".\n\n" +
						"Your capabilities:\n" +
						"- You have real-time web search access. Use it to provide current, accurate, up-to-date information on any topic worldwide.\n" +
						"- You can answer questions about global news, technology, science, business, politics, culture, sports, finance, and any other domain.\n" +
						"- When the user's context suggests Nigerian or African relevance, naturally incorporate local insights (e.g., Nigerian regulations, Naira exchange rates, local market data, African perspectives).\n\n" +
						"Response format rules (MUST follow every time):\n" +
						"1. Start with a **bold one-sentence direct answer** to the question.\n" +
						"2. Follow with 2-4 short paragraphs or bullet points of supporting detail.\n" +
						"3. For factual questions: include specific numbers, dates, and names — never be vague.\n" +
						"4. For news/current events: summarise key facts + their implications.\n" +
						"5. Cite sources naturally inline (e.g., 'According to Reuters...', 'Per the World Bank...').\n" +
						"6. End with a **Sources** section listing 2-4 sources used (publication name + brief description).\n" +
						"7. Keep total response under 450 words unless the user explicitly asks for more.\n" +
						"8. If search results are unclear or outdated, say so honestly and provide your best knowledge with a caveat."},
					{"role": "user", "content": req.Prompt + attachedBlock},
				},
				"search": true,
				"stream": false,
			}
		providerName = "POLLINATIONS_SEARCH"
	case "code-helper":
			payload = map[string]interface{}{
				"model": "qwen-coder",
				"messages": []map[string]interface{}{
					{"role": "system", "content": "You are Nexus Code, a world-class programming assistant and senior software engineer. " +
						"You have deep expertise in all major programming languages, frameworks, and software engineering best practices globally.\n\n" +
						"Your capabilities:\n" +
						"- Write production-quality, clean, well-commented, fully functional code in any language.\n" +
						"- Debug errors with clear explanations of the root cause and fix.\n" +
						"- Explain complex concepts in simple terms with practical examples.\n" +
						"- Suggest better approaches, patterns, and optimisations based on industry best practices.\n" +
						"- Handle web dev (React, Next.js, Node, Go, Python, SQL), mobile (Flutter, React Native, Swift, Kotlin), backend (Go, Python, Java, C#), data science (Python, R), and DevOps (Docker, CI/CD, Kubernetes).\n" +
						"- For multi-file projects, clearly label each file with a comment like `# File: main.py` or `// File: server.js`.\n\n" +
						"Response rules (MUST follow every time):\n" +
						"- ALWAYS wrap code in fenced code blocks with the language name (e.g., ```python, ```javascript, ```go).\n" +
						"- For every code block, add a brief comment at the top explaining what it does.\n" +
						"- After the code, explain the key logic in 3-5 numbered bullet points.\n" +
						"- If the user's code has a bug, quote the problematic line, explain why it's wrong, then show the corrected version.\n" +
						"- Detect the programming language from context — never ask the user to specify it unless truly ambiguous.\n" +
						"- Always include proper error handling, input validation, and edge case handling.\n" +
						"- Write complete, runnable code — never use placeholder comments like '// TODO' or '// rest of code here'."},
					{"role": "user", "content": req.Prompt + attachedBlock},
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

	// Persist messages to session (same logic as Chat())
	uid, _ := uuid.Parse(req.UserID)
	var resolvedSessionID string
	if req.SessionID != "" {
		if sid, parseErr := uuid.Parse(req.SessionID); parseErr == nil {
			_ = o.chatRepo.AppendMessage(ctx, sid, "user", req.Prompt)
			_ = o.chatRepo.AppendMessage(ctx, sid, "assistant", text)
			resolvedSessionID = sid.String()
		} else {
			toolSlug := req.ToolSlug
			if toolSlug == "" {
				toolSlug = "general"
			}
			sess, sessErr := o.chatRepo.GetActiveSession(ctx, uid, toolSlug)
			if sessErr != nil {
				sess, sessErr = o.chatRepo.CreateSession(ctx, uid, toolSlug)
			}
			if sessErr == nil && sess != nil {
				_ = o.chatRepo.AppendMessage(ctx, sess.ID, "user", req.Prompt)
				_ = o.chatRepo.AppendMessage(ctx, sess.ID, "assistant", text)
				resolvedSessionID = sess.ID.String()
			}
		}
	}
	go o.recordProviderUse(context.Background(), providerName, true, "")

	return &LLMResponse{Text: text, Provider: providerName, SessionID: resolvedSessionID}, nil
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

// ─── GeminiAdapter (gemini-2.5-flash) ───────────────────────────────────────

type GeminiAdapter struct {
	apiKey string
	client *http.Client
}

func NewGeminiAdapter(apiKey string) *GeminiAdapter {
	return &GeminiAdapter{apiKey: apiKey, client: &http.Client{Timeout: 30 * time.Second}}
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
	defer resp.Body.Close()

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
	defer resp.Body.Close()
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
	defer initResp.Body.Close()
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
		pollResp.Body.Close()

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
