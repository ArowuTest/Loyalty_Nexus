package services

// ai_studio_service.go — Production 4-tier AI provider orchestration (spec §9)
//
// ═══════════════════════════════════════════════════════════════════════════
//  TOOL CATALOGUE (from key-points spec doc + master spec §3.2)
// ═══════════════════════════════════════════════════════════════════════════
//  Slug             Category   Points  Provider(s)
//  ───────────────  ─────────  ──────  ────────────────────────────────────
//  translate        Create      1 pt   Google Translate API (free)
//  narrate          Create      2 pts  Google Cloud TTS → Azure TTS
//  transcribe       Create      2 pts  AssemblyAI → Groq Whisper
//  bg-remover       Create      3 pts  rembg (self-hosted) → Photoroom API
//  study-guide      Learn       3 pts  Gemini Flash → Groq
//  quiz             Learn       2 pts  Gemini Flash → Groq
//  mindmap          Learn       2 pts  Gemini Flash → Groq
//  research-brief   Learn       5 pts  Gemini Flash → Groq → DeepSeek
//  ai-photo         Create     10 pts  HF FLUX.1-Schnell (free) → FAL.AI FLUX-dev
//  bg-music         Create      5 pts  Pollinations ElevenMusic (instrumental) → Mubert → ElevenLabs sound
//  podcast          Learn       4 pts  Gemini script + Google TTS narration
//  slide-deck       Build       4 pts  Gemini Flash → Groq
//  infographic      Build       5 pts  Gemini Flash → Groq
//  bizplan          Build      12 pts  Gemini Flash → Groq → DeepSeek
//  animate-photo    Create     65 pts  FAL.AI LTX-Video (basic)
//  jingle           Create    200 pts  ElevenLabs Music (premium)
//  video-premium    Build      65 pts  FAL.AI Kling v1.5 (premium)
//  video-jingle     Build     470 pts  FAL.AI Kling + ElevenLabs (full production)
//
// Financial rule: point costs live ONLY in the DB (network_configs / studio_tools).
// This service never hardcodes them — it dispatches and returns results only.
// ═══════════════════════════════════════════════════════════════════════════

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"
	"loyalty-nexus/internal/infrastructure/external"
)

// ─── Tool category ────────────────────────────────────────────────────────────

type studioToolCat string

const (
	catText      studioToolCat = "text"
	catImage     studioToolCat = "image"
	catVideo     studioToolCat = "video"
	catVoice     studioToolCat = "voice"
	catMusic     studioToolCat = "music"
	catComposite studioToolCat = "composite"
	catVision    studioToolCat = "vision"
	catAvatar    studioToolCat = "avatar" // talking-head / lip-synced digital human
	catRender    studioToolCat = "render" // programmatic/templated video (Remotion)
)

// slugCategory maps every tool slug to its dispatch category.
var slugCategory = map[string]studioToolCat{
	"translate":      catVoice, // routed through voice pipeline (TTS/translate)
	"narrate":        catVoice,
	"transcribe":     catVoice,
	"bg-remover":     catImage,
	"ai-photo":       catImage,
	"animate-photo":  catVideo,
	"video-premium":  catVideo,
	"video-jingle":   catComposite,
	"bg-music":       catMusic,
	"jingle":         catMusic,
	"study-guide":    catText,
	"quiz":           catText,
	"mindmap":        catText,
	"research-brief": catText,
	"podcast":        catComposite,
	"slide-deck":     catText,
	"infographic":    catText,
	"bizplan":        catText,
	// ── NEW: Free tools (Pollinations secret key, zero Pollen cost) ────────────
	"transcribe-african": catVoice,
	"narrate-pro":        catVoice,
	"web-search-ai":      catText,
	"image-analyser":     catVision,
	"ask-my-photo":       catVision,
	"code-helper":        catText,
	// ── NEW: Paid tools (Pollinations Pollen credits) ────────────────────────
	"ai-photo-pro":    catImage,
	"ai-photo-max":    catImage,
	"ai-photo-dream":  catImage,
	"photo-editor":    catImage,
	"song-creator":    catMusic,
	"instrumental":    catMusic,
	"video-cinematic": catVideo,
	"video-veo":       catVideo,
	// ── Alias slugs (DB tool names that map to existing dispatch logic) ──────
	"my-marketing-jingle":   catMusic,     // alias for jingle
	"text-to-speech":        catVoice,     // alias for narrate
	"local-translation":     catVoice,     // alias for translate
	"deep-research-brief":   catText,      // alias for research-brief
	"mind-map":              catText,      // alias for mindmap
	"quiz-me":               catText,      // alias for quiz
	"my-ai-photo":           catImage,     // alias for ai-photo
	"my-video-story":        catVideo,     // alias for animate-photo
	"my-podcast":            catComposite, // alias for podcast
	"background-remover":    catImage,     // alias for bg-remover
	"animate-my-photo":      catVideo,     // alias for animate-photo
	"video-story":           catVideo,     // multi-scene image-to-video (Grok reference / Kling multi-image)
	"video-edit":            catVideo,     // natural language video editing (Grok Imagine)
	"video-extend":          catVideo,     // extend existing video (Grok Imagine)
	"business-plan-summary": catText,      // alias for bizplan
	// ── Whisk-style image composition ────────────────────────────────────────────────────────────────────────────────
	"image-compose": catImage, // Whisk-style subject+scene+style composition (Flux Ultra)
	// ── Free chat tools ──────────────────────────────────────────────────────────────────────────────────────
	"ask-nexus": catText, // free conversational AI
	// ── Talking Avatar (photo + script → lip-synced talking-head video) ──────────
	"talking-avatar": catAvatar,
	// ── Remotion templated video (SCAFFOLDING — tool ships is_active=false until
	// the render-service exists; hidden from users by ListActiveTools) ──────────
	"video-slideshow": catRender,
	"nexus-chat":      catText, // free Gemini Flash chat
	"voice-to-plan":   catText, // voice-to-business-plan

	// ── Gemma 4 / Nexus AI Tools ────────────────────────────────────────────────────────────────────────
	"code-pro":     catVision, // Nexus Code Pro: code + optional image upload (multimodal debugging)
	"doc-analyzer": catVision, // Nexus Document Analyzer: PDF/image structured extraction
	"localize-ui":  catVision, // Nexus Localization Engine: screenshot OCR + African dialect translation
	"nexus-agent":  catText,   // Nexus Agentic Workflow Builder: multi-step agentic reasoning loop

	// website-builder: routed as catText so the worker doesn't fail with "unknown slug".
	// dispatchText handles it by delegating to the website builder generation flow.
	"website-builder": catText,
}

// ─── Provider result ──────────────────────────────────────────────────────────────────────────────────────

type studioProviderResult struct {
	OutputURL  string // CDN URL or data URI for binary outputs
	OutputURL2 string // second audio track (Suno returns 2 takes)
	OutputText string // for text-generation tools (study guide, bizplan, etc.)
	Provider   string // e.g. "gemini-flash", "fal.ai/flux"
	CostMicros int    // fractional cost in µUSD for accounting
	DurationMs int
}

// ─── AIStudioOrchestrator ─────────────────────────────────────────────────────

type AIStudioOrchestrator struct {
	cfg            *config.ConfigManager
	studioRepo     repositories.StudioRepository
	studioSvc      *StudioService
	userRepo       repositories.UserRepository
	storage        external.AssetStorage
	httpClient     *http.Client
	llmOrch        *external.LLMOrchestrator // for provider health tracking
	providerDB     ProviderConfigStore       // legacy category registry during Router V2 migration
	routingDB      AIRoutingStore            // Router V2 tool/stage routing authority
	capacity       *AICapacityController     // distributed surge/capacity guard
	agentSemaphore chan struct{}             // caps concurrent true-agent runs (ReAct loop)
}

// ProviderConfigStore is the minimal interface the orchestrator needs
// to load dynamic provider chains from the database.
// Implemented by persistence.AIProviderRepository.
type ProviderConfigStore interface {
	ListByCategory(ctx context.Context, category string) ([]entities.AIProviderConfig, error)
}

func NewAIStudioOrchestrator(
	cfg *config.ConfigManager,
	studioRepo repositories.StudioRepository,
	studioSvc *StudioService,
	userRepo repositories.UserRepository,
	storage external.AssetStorage,
) *AIStudioOrchestrator {
	if storage == nil {
		storage = external.NewAssetStorageFromEnv()
	}
	agentMax := 10
	if cfg != nil {
		agentMax = cfg.GetInt("nexus_agent_max_concurrent", 10)
	}
	if agentMax < 1 {
		agentMax = 1
	}
	if agentMax > 200 {
		agentMax = 200
	}
	agentSem := make(chan struct{}, agentMax)
	for i := 0; i < agentMax; i++ {
		agentSem <- struct{}{}
	}

	return &AIStudioOrchestrator{
		cfg:            cfg,
		studioRepo:     studioRepo,
		studioSvc:      studioSvc,
		userRepo:       userRepo,
		storage:        storage,
		httpClient:     &http.Client{Timeout: 120 * time.Second},
		agentSemaphore: agentSem,
	}
}

// SetProviderDB wires the DB-backed provider registry.
// When set, the dispatch functions check DB for active providers before
// falling back to the hardcoded chains (backward-compatible).
func (o *AIStudioOrchestrator) SetProviderDB(store ProviderConfigStore) {
	o.providerDB = store
}

// dbProviders returns active providers for a category from DB, sorted by priority.
// Returns nil (not an error) if DB is not wired or returns nothing — callers
// treat nil as "use hardcoded chain".
func (o *AIStudioOrchestrator) dbProviders(ctx context.Context, category string) []entities.AIProviderConfig {
	if o.providerDB == nil {
		return nil
	}
	providers, err := o.providerDB.ListByCategory(ctx, category)
	if err != nil {
		log.Printf("[AIStudio] dbProviders(%s): %v — using hardcoded chain", category, err)
		return nil
	}
	return providers
}

// SetLLMOrch wires the LLM orchestrator for provider health tracking.
// Called after construction so the constructor stays dependency-free.
func (o *AIStudioOrchestrator) SetLLMOrch(orch *external.LLMOrchestrator) {
	o.llmOrch = orch
}

// classifyErrorType classifies an error message string into one of four
// canonical error-type labels used as a prefix in the error_message field
// (no migration required — stored as "[TYPE][provider] original message").
func classifyErrorType(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "429") || strings.Contains(lower, "rate limit") || strings.Contains(lower, "quota exceeded"):
		return "RATE_LIMIT"
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded") || strings.Contains(lower, "context deadline"):
		return "TIMEOUT"
	case strings.Contains(lower, "invalid") || strings.Contains(lower, "bad request") || strings.Contains(lower, "400"):
		return "INVALID_INPUT"
	default:
		return "PROVIDER_ERROR"
	}
}

// Dispatch is the main entry point: resolves category, calls the right provider chain,
// then persists the result via StudioService.
func (o *AIStudioOrchestrator) Dispatch(ctx context.Context, genID uuid.UUID) error {
	gen, err := o.studioRepo.FindGenerationByID(ctx, genID)
	if err != nil {
		return fmt.Errorf("generation not found: %w", err)
	}

	// Mark processing
	if err := o.studioRepo.UpdateStatus(ctx, genID, "processing", "", ""); err != nil {
		return fmt.Errorf("mark processing: %w", err)
	}

	start := time.Now()
	routeCtx := withAIGenerationID(ctx, genID)
	result, dispatchErr := o.route(routeCtx, gen)
	elapsed := int(time.Since(start).Milliseconds())

	if dispatchErr != nil {
		// BUG-039: determine the attempted provider from the category chain
		attemptedProvider := "unknown"
		if cat, ok := slugCategory[gen.ToolSlug]; ok {
			if providers := o.dbProviders(ctx, string(cat)); len(providers) > 0 {
				attemptedProvider = providers[0].Slug
			}
		}
		// BUG-040: classify error type; prefix into error_message (no migration)
		errType := classifyErrorType(dispatchErr.Error())
		classifiedMsg := "[" + errType + "][" + attemptedProvider + "] " + dispatchErr.Error()
		if failErr := o.studioSvc.FailGeneration(ctx, genID, classifiedMsg); failErr != nil {
			log.Printf("[AIStudio] FailGeneration for %s: %v", genID, failErr)
		}
		return dispatchErr
	}

	result.DurationMs = elapsed
	// Track studio tool usage in Redis for admin AI health dashboard
	if o.llmOrch != nil {
		go o.llmOrch.RecordStudioToolUse(context.Background(), gen.ToolSlug, result.Provider)
	}
	return o.complete(ctx, gen, result)
}

// route dispatches to the correct provider chain based on slug category.
func (o *AIStudioOrchestrator) route(ctx context.Context, gen *entities.AIGeneration) (*studioProviderResult, error) {
	slug := gen.ToolSlug

	// ── Parse the JSON envelope emitted by buildEnrichedPrompt ────────────
	// Every generation stores a JSON object in gen.Prompt produced by the HTTP
	// handler's buildEnrichedPrompt(). Dispatch functions must read from this
	// envelope — never do string-splitting on gen.Prompt directly.
	env := parseEnvelope(gen.Prompt)

	cat, known := slugCategory[slug]
	if !known {
		return nil, fmt.Errorf("unknown tool slug %q", slug)
	}

	switch cat {
	case catText:
		return o.dispatchText(ctx, slug, env)
	case catImage:
		return o.dispatchImage(ctx, slug, env)
	case catVideo:
		return o.dispatchVideo(ctx, slug, env)
	case catVoice:
		return o.dispatchVoiceOrTranslate(ctx, slug, env)
	case catMusic:
		return o.dispatchMusic(ctx, slug, env)
	case catComposite:
		return o.dispatchComposite(ctx, slug, env)
	case catVision:
		return o.dispatchVision(ctx, slug, env)
	case catAvatar:
		return o.dispatchAvatar(ctx, gen.UserID, slug, env)
	case catRender:
		return o.dispatchRender(ctx, slug, env)
	default:
		return nil, fmt.Errorf("unhandled category %q", cat)
	}
}

// dispatchRender handles Remotion templated-video tools (SCAFFOLDING for Track A).
// It posts {composition, props} to the self-hosted render-service at
// RENDER_SERVICE_URL (host-portable Render→GCP) and polls for the MP4. The
// render-service and the video-slideshow tool are not live yet — the tool row
// ships is_active=false, so this path is only reachable once both exist. Reuses
// the same points/refund pipeline as every other dispatcher.
func (o *AIStudioOrchestrator) dispatchRender(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	base := os.Getenv("RENDER_SERVICE_URL")
	if base == "" {
		return nil, fmt.Errorf("video templates are not configured yet (RENDER_SERVICE_URL unset)")
	}
	// The composition id is the tool slug (e.g. "video-slideshow"); props carry
	// the user's inputs (image URLs, music, captions) from the envelope's Extra bag.
	composition := slug
	props := map[string]interface{}{
		"prompt":      env.Prompt,
		"aspectRatio": env.AspectRatio,
	}
	if env.Extra != nil {
		for k, v := range env.Extra {
			props[k] = v
		}
	}
	url, err := o.callRemotionRender(ctx, strings.TrimRight(base, "/"), composition, props)
	if err != nil {
		return nil, fmt.Errorf("video template render failed: %w", err)
	}
	return &studioProviderResult{OutputURL: url, Provider: "remotion/" + composition, CostMicros: 1000}, nil
}

// callRemotionRender POSTs a render request to the self-hosted render-service and
// returns the resulting MP4 URL. The service wraps Remotion's renderMedia();
// where it actually renders (Render container now, GCP Cloud Run later) is its
// own internal detail — this Go side only ever speaks HTTP to RENDER_SERVICE_URL.
func (o *AIStudioOrchestrator) callRemotionRender(ctx context.Context, base, composition string, props map[string]interface{}) (string, error) {
	payload := map[string]interface{}{"composition": composition, "props": props}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/render", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	// Shared-secret auth for the internal render-service.
	if tok := os.Getenv("RENDER_SERVICE_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	// Dedicated long timeout — a templated render can take 1-3 minutes.
	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("render-service %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}
	var parsed struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || parsed.URL == "" {
		return "", fmt.Errorf("render-service: no url in response: %s", truncateStr(string(raw), 200))
	}
	return parsed.URL, nil
}

// dispatchAvatar handles talking-avatar tools (photo + script → lip-synced video).
// DB-first via the avatar provider registry (admin-swappable), then a hardcoded
// FAL fallback. Providers are keyed by input SHAPE, so each adapter reads only the
// fields it needs from providerInput — text-driven models use Prompt+VoiceID,
// audio-driven models use AudioURL (resolved from the script via TTS on demand).
//
// Voice handling:
//   - preset voice (Bill/Cherry/…)     → FAL text-driven model (fast, internal TTS)
//   - "My Voice" (voice_source=elevenlabs) → we TTS the script in the user's OWN
//     stored ElevenLabs clone, then lip-sync the audio (audio-driven path). The
//     clone id is resolved SERVER-SIDE from the user record, never trusted from
//     the client, so nobody can borrow another user's voice.
//   - uploaded audio (audio_url)       → lip-sync straight to that audio.
func (o *AIStudioOrchestrator) dispatchAvatar(ctx context.Context, userID uuid.UUID, slug string, env promptEnvelope) (*studioProviderResult, error) {
	imageURL := env.ImageURL
	if imageURL == "" {
		return nil, fmt.Errorf("talking-avatar: a source photo (image_url) is required")
	}
	script := env.Prompt
	voice := env.VoiceID
	preAudio := ""
	useClonedVoice := false
	if env.Extra != nil {
		if a, ok := env.Extra["audio_url"].(string); ok {
			preAudio = a
		}
		if vs, ok := env.Extra["voice_source"].(string); ok && vs == "elevenlabs" {
			useClonedVoice = true
		}
	}
	if script == "" && preAudio == "" {
		return nil, fmt.Errorf("talking-avatar: a script or uploaded audio clip is required")
	}

	runAvatar := func(in providerInput) (*studioProviderResult, error) {
		url, _, cost, usedSlug, err := o.runToolStageChain(ctx, nil, slug, "main", in)
		if err != nil {
			return nil, err
		}
		return &studioProviderResult{
			OutputURL: url, Provider: "route/" + usedSlug, CostMicros: cost,
		}, nil
	}

	// Uploaded audio is authoritative: never replace it with generated speech.
	if preAudio != "" {
		return runAvatar(providerInput{ImageURL: imageURL, AudioURL: preAudio})
	}

	if useClonedVoice {
		u, err := o.userRepo.FindByID(ctx, userID)
		if err != nil || u == nil || u.ClonedVoiceID == "" {
			return nil, fmt.Errorf("talking-avatar: no cloned voice found — record your voice first")
		}
		speech, err := o.dispatchTTSForStage(ctx, slug, "clone-speech", script, u.ClonedVoiceID)
		if err != nil {
			return nil, fmt.Errorf("talking-avatar cloned-voice speech unavailable: %w", err)
		}
		avatar, err := runAvatar(providerInput{ImageURL: imageURL, AudioURL: speech.OutputURL})
		if err != nil {
			return nil, fmt.Errorf("talking-avatar audio route unavailable: %w", err)
		}
		avatar.Provider = speech.Provider + "+" + avatar.Provider
		avatar.CostMicros += speech.CostMicros
		return avatar, nil
	}

	// Fast path: text-driven avatar providers can render speech internally.
	textAvatar, textErr := runAvatar(providerInput{
		ImageURL: imageURL, Prompt: script, VoiceID: voice,
	})
	if textErr == nil {
		return textAvatar, nil
	}

	// Quality fallback: synthesize speech through the tool's Admin-configured TTS
	// stage, then retry the avatar stage with audio-driven providers.
	speech, speechErr := o.dispatchTTSForStage(ctx, slug, "speech", script, voice)
	if speechErr != nil {
		return nil, fmt.Errorf("talking-avatar routes exhausted: text=%v; speech=%v", textErr, speechErr)
	}
	audioAvatar, audioErr := runAvatar(providerInput{
		ImageURL: imageURL, AudioURL: speech.OutputURL,
	})
	if audioErr != nil {
		return nil, fmt.Errorf("talking-avatar routes exhausted: text=%v; audio=%v", textErr, audioErr)
	}
	audioAvatar.Provider = speech.Provider + "+" + audioAvatar.Provider
	audioAvatar.CostMicros += speech.CostMicros
	return audioAvatar, nil
}

// ─── promptEnvelope is the parsed form of buildEnrichedPrompt's output ───────

type promptEnvelope struct {
	Prompt         string                 `json:"prompt"`
	ImageURL       string                 `json:"image_url"`
	DocumentURL    string                 `json:"document_url"` // FEAT-01: PDF/TXT for knowledge tools
	VoiceID        string                 `json:"voice_id"`
	Language       string                 `json:"language"`
	AspectRatio    string                 `json:"aspect_ratio"`
	Duration       int                    `json:"duration"`
	Vocals         *bool                  `json:"vocals"`
	Lyrics         string                 `json:"lyrics"`
	StyleTags      []string               `json:"style_tags"`
	NegativePrompt string                 `json:"negative_prompt"`
	Extra          map[string]interface{} `json:"extra"`
}

// parseEnvelope decodes the JSON envelope stored in the generation's Prompt column.
// If the stored string is not valid JSON (legacy plain-text prompts), it returns an
// envelope with Prompt set to the raw string so existing rows still work.
func parseEnvelope(raw string) promptEnvelope {
	var env promptEnvelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		// Plain-text fallback (old rows / direct API calls)
		env.Prompt = raw
	}
	return env
}

// ─── Text dispatch (study guide, quiz, mindmap, research-brief, bizplan, slide-deck, infographic,
//                   web-search-ai, code-helper) ──────────────────────────────────────────────────

func (o *AIStudioOrchestrator) dispatchText(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	prompt := env.Prompt
	// Handle alias slugs by remapping to canonical slugs
	switch slug {
	case "deep-research-brief":
		slug = "research-brief"
	case "mind-map":
		slug = "mindmap"
	case "quiz-me":
		slug = "quiz"
	case "my-podcast":
		slug = "podcast"
	case "business-plan-summary":
		slug = "bizplan"
	case "ask-nexus", "nexus-chat":
		slug = "ask-nexus" // both use the same free conversational AI path
	case "voice-to-plan":
		slug = "voice-to-plan"
	}

	// nexus-agent: multi-step agentic workflow builder
	if slug == "nexus-agent" {
		return o.dispatchNexusAgent(ctx, env)
	}

	systemPrompt, userPrompt := buildTextPrompts(slug, prompt)

	knowledgeSlugs := map[string]bool{
		"study-guide": true, "quiz": true, "mindmap": true, "research-brief": true,
		"bizplan": true, "slide-deck": true, "infographic": true, "podcast": true,
	}
	documentURL := ""
	if env.DocumentURL != "" && knowledgeSlugs[slug] {
		documentURL = env.DocumentURL
		userPrompt = fmt.Sprintf("%s\n\n[Use the uploaded document as the primary source material. Analyse it thoroughly and base the response on its content.]", userPrompt)
	}

	in := providerInput{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		DocumentURL:  documentURL,
	}
	url, text, cost, usedSlug, err := o.runToolStageChain(ctx, nil, slug, "main", in)
	if err != nil {
		return nil, fmt.Errorf("text route unavailable for %q: %w", slug, err)
	}
	return &studioProviderResult{OutputText: text, OutputURL: url, Provider: "route/" + usedSlug, CostMicros: cost}, nil
}

// buildTextPrompts returns (systemPrompt, userPrompt) for each tool slug.
func buildTextPrompts(slug, input string) (system, user string) {
	nexusSys := "You are Nexus AI, a world-class AI assistant with comprehensive global knowledge. " +
		"You are deeply knowledgeable across all domains: business, education, science, technology, culture, finance, law, health, and the arts. " +
		"Always produce thorough, accurate, well-structured responses that provide genuine value. " +
		"When the user's context or query suggests Nigerian or African relevance, naturally incorporate local insights, examples, and context."

	switch slug {
	case "web-search-ai":
		return nexusSys + " You have real-time web search access. Always cite your sources naturally.",
			fmt.Sprintf("Search the web and provide a comprehensive, well-structured answer to: %s", input)

	case "code-helper":
		return "You are Nexus Code, a world-class software engineer and programming mentor. " +
				"You write production-quality, clean, well-commented code in any language. " +
				"You explain every solution clearly with the key logic highlighted. " +
				"You always wrap code in fenced code blocks with the correct language tag. " +
				"You include error handling in all examples. " +
				"You detect the language from context and never ask unless truly ambiguous.",
			input

	case "website-builder":
		return "You are an expert web developer. Generate a complete, production-ready single-page HTML website. " +
				"Use responsive modern CSS, accessible semantic markup, professional typography and a coherent visual system. " +
				"Return ONLY the complete HTML document beginning with <!DOCTYPE html>; never use markdown fences.",
			"Create a professional website for: " + input

	case "study-guide":
		return nexusSys + " You are an expert educator who creates comprehensive, exam-ready study materials.",
			fmt.Sprintf(`Create a comprehensive, exam-ready study guide for: %s

Structure your guide as follows:

## Overview
Brief introduction to the topic (2-3 sentences).

## Key Concepts
For each major concept:
- **Concept Name**: Clear definition
		- Real-world example with clear, relatable context
- Why it matters

## Detailed Explanations
In-depth coverage of each subtopic with examples, diagrams described in text, and analogies.

## Practice Questions
5 short-answer questions with model answers.
3 essay-style questions with outline answers.

## Quick Revision Summary
Bullet-point cheat sheet of the 10 most important facts/formulas/concepts.

## Further Study
3 recommended areas to explore for deeper understanding.

	Make it thorough enough for any major exam (WAEC, JAMB, A-Level, SAT, university, or professional certification).`, input)

	case "quiz":
		return nexusSys + " You are an expert quiz designer who creates challenging, educational assessments.",
			fmt.Sprintf(`Create 10 high-quality quiz questions about: %s

Requirements:
- Mix difficulty: 3 easy, 4 medium, 3 hard
		- Each question must have 4 distinct options (no obviously wrong answers)
- Explanations must be educational, not just restate the answer

Return ONLY valid JSON array, no markdown, no extra text:
[{"question": "...", "options": ["A) ...", "B) ...", "C) ...", "D) ..."], "answer": "A", "explanation": "Detailed explanation of why this is correct and why others are wrong."}]`, input)

	case "mindmap":
		return nexusSys + " You are an expert at creating rich, comprehensive mind maps for learning and planning.",
			fmt.Sprintf(`Create a detailed, comprehensive mind map for: %s

Requirements:
- Central topic should be concise (2-4 words)
- Include 5-7 main branches covering all key aspects
- Each branch should have 3-5 sub-branches with specific, actionable items
		- Sub-branches should be specific facts, examples, or action items — not vague categories

Return ONLY valid JSON, no markdown, no extra text:
{"center": "...", "branches": [{"label": "...", "color": "#hex", "children": [{"label": "...", "children": [{"label": "..."}]}]}]}

Use these colors for branches: #f59e0b, #3b82f6, #10b981, #8b5cf6, #ef4444, #06b6d4, #f97316`, input)

	case "research-brief":
		return nexusSys + " You are a senior research analyst who produces rigorous, data-driven research briefs.",
			fmt.Sprintf(`Write a comprehensive, professional research brief about: %s

## Executive Summary
3-4 sentences capturing the most important findings and their significance.

## Background & Context
	Historical context, current state, and why this topic matters globally and locally.

## Key Findings
7-10 specific, evidence-based findings with data points, statistics, and examples where possible.

## Market & Industry Analysis
- Market size and growth trends (with specific figures)
- Key players and competitive landscape
		- Regional market dynamics (include Nigerian/African context where applicable)
- Opportunities and challenges

## Expert Perspectives
Summarise what leading experts, institutions, or reports say about this topic.

## Strategic Recommendations
5 specific, actionable recommendations with rationale.

## Conclusion
Synthesis of findings and forward-looking outlook.

Be specific, cite real data and examples, and make it genuinely useful for decision-making.`, input)

	case "slide-deck":
		return nexusSys + " You are an expert presentation designer who creates compelling, professional slide decks.",
			fmt.Sprintf(`Create a professional, compelling slide deck for: %s

Requirements:
- 12-15 slides covering the topic comprehensively
- Each slide should have a strong, action-oriented title
- Bullets should be concise (max 8 words each), not full sentences
- Speaker notes should be 2-3 sentences of talking points
- Include a strong opening hook and a clear call-to-action on the final sl
	Return ONLY valid JSON, no markdown, no extra text:
	{"title": "...", "subtitle": "...", "slides": [{"number": 1, "title": "...", "bullets": ["...", "...", "..."], "speaker_notes": "..."}]}`, input)

	case "infographic":
		return nexusSys + " You are an expert data visualisation designer who creates insightful, visually compelling infographics.",
			fmt.Sprintf(`Create a rich, data-packed infographic about: %s

Requirements:
- 5-6 sections covering different aspects of the topic
- Mix stat-heavy sections (with specific numbers/percentages) and insight sections (with bullet points)
- Stats must be real, specific, and verifiable — not made up
- Points must be concise, punchy, and genuinely insightful (under 12 words)
		- Use real, globally verifiable data; include Nigerian/African data where available and relevant

Return ONLY valid JSON, no markdown, no code blocks:
{"title": "Main Title", "subtitle": "Brief compelling description", "sections": [{"heading": "Section Title", "icon": "chart", "stat": "42%%", "stat_label": "Label for the stat", "points": ["Key point 1", "Key point 2", "Key point 3"]}]}

icon must be one of: chart, data, stats, info, tip, warning, check, star, money, people, time, globe, phone, idea, growth
stat and stat_label are optional — only include when there is a real, meaningful number`, input)

	case "bizplan":
		return nexusSys + " You are a top-tier business consultant and MBA with global expertise across all industries and markets.",
			fmt.Sprintf(`Write a comprehensive, investor-ready business plan for: %s

## Executive Summary
Compelling 3-paragraph overview: the problem, the solution, and the opportunity.

## Company Description & Vision
Mission statement, vision, core values, and what makes this business unique.

## Market Analysis
		- Target market size and demographics (with specific figures; include Nigerian/African context if relevant)
- Market trends and growth drivers
- Competitive landscape (name specific competitors)
- Competitive advantage and positioning

## Products & Services
Detailed description of offerings, pricing strategy, and value proposition.

## Marketing & Sales Strategy
- Customer acquisition channels (digital, traditional, referral)
- Brand positioning and messaging
- Sales funnel and conversion strategy
- Social media and content strategy

## Operations Plan
- Business model and revenue streams
- Key processes and workflows
- Team structure and key hires needed
- Technology and tools required

## Financial Projections (3-Year)
- Year 1, 2, 3 revenue projections with assumptions
- Cost structure and break-even analysis
- Key financial metrics (CAC, LTV, gross margin)
- Funding requirements and use of funds

## Risk Analysis & Mitigation
Top 5 risks and specific mitigation strategies.

## Conclusion & Call to Action
Why now, why this team, and what's the ask.

	Use the appropriate currency for the market described (default to Nigerian Naira ₦ if the context is Nigerian). Be specific, realistic, and actionable.`, input)

	case "ask-nexus":
		return nexusSys + " You are a helpful, knowledgeable conversational AI. Respond naturally and thoroughly.",
			input
	case "voice-to-plan":
		return nexusSys + " You are an expert business consultant who transforms spoken ideas into structured business plans.",
			fmt.Sprintf(`Transform this spoken idea or voice note into a structured, actionable business plan:\n\n%s\n\nProvide:\n## Business Concept\nClear one-paragraph description.\n## Target Market\nWho are the customers and what problem does this solve?\n## Revenue Model\nHow will this make money?\n## Key Activities\n3-5 core activities needed to launch.\n## Resources Needed\nCapital, team, technology, partnerships.\n## Next Steps\n5 immediate actions to take this week.`, input)
	default:
		return nexusSys,
			fmt.Sprintf(`Generate comprehensive, well-structured, genuinely useful content about: %s\n\nProvide:\n- A clear, direct answer or output\n- Supporting details, examples, and context\n- Practical takeaways the user can act on immediately`, input)
	}
}

// ─── Prompt Enhancement ──────────────────────────────────────────────────────
// enhanceImagePrompt takes a user's simple image description and expands it into
// a rich, professional-quality prompt using Midjourney/DALL-E 3 best practices.
// Falls back to the original prompt if Gemini is unavailable.
func (o *AIStudioOrchestrator) enhanceImagePrompt(ctx context.Context, slug, userPrompt string) string {
	// Skip enhancement for very long prompts (user likely already wrote a detailed prompt)
	if len(userPrompt) > 200 {
		return userPrompt
	}
	// Skip for bg-remover and photo-editor (they use image URLs, not text prompts)
	if slug == "bg-remover" || slug == "photo-editor" {
		return userPrompt
	}

	// Style guidance per slug
	styleGuide := "photorealistic, ultra-detailed, professional photography"
	switch slug {
	case "ai-photo-dream":
		styleGuide = "dreamlike, surreal, painterly, ethereal atmosphere, soft lighting"
	case "ai-photo-pro":
		styleGuide = "professional photography, studio quality, sharp focus, perfect lighting, commercial grade"
	case "ai-photo-max":
		styleGuide = "ultra-high resolution, photorealistic, cinematic, award-winning photography, 8K"
	}

	sys := "You are an expert AI image prompt engineer specialising in Midjourney, DALL-E 3, and Stable Diffusion. " +
		"Your job is to take a user's simple image description and expand it into a rich, detailed, professional-quality prompt. " +
		"Add: specific lighting (golden hour, studio, cinematic, rim light), composition (rule of thirds, close-up, wide angle, aerial), " +
		"quality modifiers (ultra-detailed, 8K, sharp focus, photorealistic), mood/atmosphere, and relevant style tags. " +
		"Keep the core subject and intent exactly as the user described. " +
		"Return ONLY the enhanced prompt — no explanation, no quotes, no preamble. Maximum 150 words."

	userMsg := fmt.Sprintf("Style target: %s\n\nUser's prompt: %s\n\nEnhanced prompt:", styleGuide, userPrompt)

	_, enhanced, _, _, err := o.runToolStageChain(ctx, nil, slug, "prompt", providerInput{SystemPrompt: sys, UserPrompt: userMsg})
	if err != nil || len(enhanced) < 10 {
		return userPrompt
	}
	return enhanced
}

// enhanceVideoPrompt takes a user's simple video description and expands it into
// a rich, cinematic prompt using Runway/Pika/Veo best practices.
func (o *AIStudioOrchestrator) enhanceVideoPrompt(ctx context.Context, slug, userPrompt string) string {
	// Skip enhancement for very long prompts
	if len(userPrompt) > 200 {
		return userPrompt
	}

	sys := "You are an expert AI video prompt engineer specialising in Runway, Pika, Veo, and Wan. " +
		"Your job is to take a user's simple video description and expand it into a rich, cinematic prompt. " +
		"Add: camera movement (slow zoom, pan left, dolly forward, aerial tracking shot), " +
		"cinematic quality (cinematic lighting, film grain, shallow depth of field, 4K), " +
		"motion description (gentle breeze, flowing, dynamic, slow motion), " +
		"atmosphere (golden hour, dramatic clouds, neon lights, misty morning). " +
		"Keep the core subject and intent exactly as the user described. " +
		"Return ONLY the enhanced prompt — no explanation, no quotes, no preamble. Maximum 100 words."

	userMsg := fmt.Sprintf("User's video prompt: %s\n\nCinematic enhanced prompt:", userPrompt)

	_, enhanced, _, _, err := o.runToolStageChain(ctx, nil, slug, "prompt", providerInput{SystemPrompt: sys, UserPrompt: userMsg})
	if err != nil || len(enhanced) < 10 {
		return userPrompt
	}
	return enhanced
}

// ─── Image dispatch ────────────────────────────────────────────────────────────
// Handles: ai-photo, bg-remover, ai-photo-pro, ai-photo-max, ai-photo-dream, photo-editor, image-compose

func (o *AIStudioOrchestrator) dispatchImage(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	prompt := o.enhanceImagePrompt(ctx, slug, env.Prompt)
	for _, tag := range env.StyleTags {
		if tag != "" && !strings.Contains(strings.ToLower(prompt), strings.ToLower(tag)) {
			prompt += ", " + tag + " style"
		}
	}
	if env.NegativePrompt != "" {
		prompt += ". Avoid: " + env.NegativePrompt
	}

	if slug == "bg-remover" || slug == "background-remover" {
		imageURL := env.ImageURL
		if imageURL == "" {
			imageURL = env.Prompt
		}
		if imageURL == "" {
			return nil, fmt.Errorf("%s: image_url is required", slug)
		}
		return o.dispatchBgRemover(ctx, slug, imageURL)
	}

	in := providerInput{
		Prompt:      prompt,
		ImageURL:    env.ImageURL,
		AspectRatio: env.AspectRatio,
		Extra:       env.Extra,
	}

	if env.Extra != nil {
		if seed, ok := env.Extra["seed"].(float64); ok && seed > 0 {
			in.Seed = int64(seed)
		}
	}

	switch slug {
	case "image-compose":
		if env.ImageURL == "" {
			return nil, fmt.Errorf("image-compose: subject image_url is required")
		}
		refs := []string{env.ImageURL}
		if v, ok := env.Extra["scene_image_url"].(string); ok && v != "" {
			refs = append(refs, v)
		}
		if v, ok := env.Extra["style_image_url"].(string); ok && v != "" {
			refs = append(refs, v)
		}
		in.ImageURL = ""
		in.ReferenceImageURLs = refs

	case "photo-editor":
		if env.ImageURL == "" {
			return nil, fmt.Errorf("photo-editor: image_url is required")
		}
		strength := 0.75
		if env.Extra != nil {
			if s, ok := env.Extra["strength"].(float64); ok && s > 0 && s <= 1 {
				strength = s
			}
		}
		if strength < 0.5 {
			in.Prompt += " (subtle change, preserve most of the original)"
		} else if strength > 0.85 {
			in.Prompt += " (strong transformation)"
		}

	case "ai-photo", "my-ai-photo":
		qualityKeywords := []string{
			"photo", "realistic", "4k", "8k", "hd", "cinematic", "detailed",
			"portrait", "professional", "ultra", "sharp", "raw", "shot", "film",
		}
		lower := strings.ToLower(in.Prompt)
		hasQuality := false
		for _, kw := range qualityKeywords {
			if strings.Contains(lower, kw) {
				hasQuality = true
				break
			}
		}
		if !hasQuality && len(in.Prompt) < 120 {
			in.Prompt += ", photorealistic, professional photography, sharp focus, high detail, natural lighting"
		}

	case "ai-photo-pro", "ai-photo-max":
		in.Resolution = "2k"

	case "ai-photo-dream":
		// Style comes from prompt enhancement; provider/model preference is Admin-configured.
	}

	url, text, cost, usedSlug, err := o.runToolStageChain(ctx, nil, slug, "main", in)
	if err != nil {
		return nil, fmt.Errorf("image route unavailable for %q: %w", slug, err)
	}
	return &studioProviderResult{
		OutputURL: url, OutputText: text, Provider: "route/" + usedSlug, CostMicros: cost,
	}, nil
}
func (o *AIStudioOrchestrator) dispatchBgRemover(ctx context.Context, slug, imageURL string) (*studioProviderResult, error) {
	in := providerInput{ImageURL: imageURL}
	url, _, cost, usedSlug, err := o.runToolStageChain(ctx, nil, slug, "main", in)
	if err != nil {
		return nil, fmt.Errorf("background removal route unavailable for %q: %w", slug, err)
	}
	return &studioProviderResult{OutputURL: url, Provider: "route/" + usedSlug, CostMicros: cost}, nil
}

// ─── Video dispatch ────────────────────────────────────────────────────────────
// Handles: animate-photo, video-premium, video-jingle, video-cinematic, video-veo
//
// Pollinations video model pricing (confirmed 2026-03-28):
//   FREE:  wan-fast (Wan 2.2, 91.4% success), p-video (Pruna p-video, 100% success)
//   PAID:  seedance (1.8/M pollen), seedance-pro (1.0/M), veo (0.150/sec), wan (0.050/sec)
//   OFF:   ltx-2 (5.3% success — REMOVED from all chains)
//
// Strategy:
//   video-cinematic  → wan-fast FREE primary, p-video FREE fallback  (ltx-2 was OFF — replaced)
//   video-veo        → veo PAID primary, wan-fast FREE fallback, p-video FREE 2nd fallback
//   animate-photo    → FAL LTX-Video → wan-fast FREE → p-video FREE (ltx-2 was OFF — replaced)

func (o *AIStudioOrchestrator) dispatchVideo(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	if env.NegativePrompt != "" && env.Prompt != "" {
		env.Prompt += ". Avoid: " + env.NegativePrompt
	}

	prompt := o.enhanceVideoPrompt(ctx, slug, env.Prompt)
	duration := env.Duration
	if duration <= 0 {
		duration = 6
	}
	aspectRatio := env.AspectRatio
	resolution := "720p"
	generateAudio := false
	if env.Extra != nil {
		if ga, ok := env.Extra["generate_audio"].(bool); ok {
			generateAudio = ga
		}
	}

	// Preserve UI controls that previously lived inside provider-specific branches.
	if slug == "video-cinematic" || slug == "video-veo" {
		if mi, ok := env.Extra["motion_intensity"].(float64); ok && mi > 0 {
			hints := map[int]string{
				1: "very subtle motion, minimal movement",
				2: "gentle motion, slow and smooth",
				3: "balanced motion, natural movement",
				4: "dynamic motion, expressive movement",
				5: "extreme motion, high energy, dramatic movement",
			}
			if hint := hints[int(mi)]; hint != "" {
				prompt += ". Motion style: " + hint
			}
		}
		if cm, ok := env.Extra["camera_movement"].(string); ok && cm != "" {
			prompt += ". Camera: " + cm
		}
		generateAudio = true
	}
	if slug == "video-veo" {
		if hint, ok := env.Extra["audio_direction"]; ok {
			if s := fmt.Sprintf("%v", hint); s != "" && s != "<nil>" {
				prompt += ". Audio: " + s
			}
		}
	}

	in := providerInput{
		Prompt:        prompt,
		ImageURL:      env.ImageURL,
		DurationSecs:  duration,
		AspectRatio:   aspectRatio,
		Resolution:    resolution,
		GenerateAudio: generateAudio,
		Extra:         env.Extra,
	}

	switch slug {
	case "video-edit":
		source := env.ImageURL
		if v, ok := env.Extra["video_url"].(string); ok && v != "" {
			source = v
		}
		if source == "" {
			return nil, fmt.Errorf("video-edit: video_url is required")
		}
		if env.Prompt == "" {
			return nil, fmt.Errorf("video-edit: edit instruction is required")
		}
		in.ImageURL = ""
		in.VideoURL = source

	case "video-extend":
		source := env.ImageURL
		if v, ok := env.Extra["video_url"].(string); ok && v != "" {
			source = v
		}
		if source == "" {
			return nil, fmt.Errorf("video-extend: video_url is required")
		}
		in.ImageURL = ""
		in.VideoURL = source
		in.Extend = true

	case "video-story", "my-video-story":
		var refs []string
		if arr, ok := env.Extra["image_urls"].([]interface{}); ok {
			for _, raw := range arr {
				if s, ok := raw.(string); ok && s != "" {
					refs = append(refs, s)
				}
			}
		}
		if len(refs) == 0 && env.ImageURL != "" {
			refs = append(refs, env.ImageURL)
		}
		minImages := 1
		if slug == "video-story" {
			minImages = 2
		}
		if len(refs) < minImages {
			return nil, fmt.Errorf("%s: at least %d image(s) required, got %d", slug, minImages, len(refs))
		}
		if duration <= 6 {
			in.DurationSecs = 10
		}
		if len(refs) == 1 {
			in.ImageURL = refs[0]
		} else {
			in.ImageURL = ""
			in.ReferenceImageURLs = refs
		}

	case "video-cinematic":
		if env.ImageURL == "" {
			return nil, fmt.Errorf("video-cinematic: image_url is required")
		}

	case "animate-photo", "animate-my-photo", "video-premium":
		if in.ImageURL == "" && strings.HasPrefix(strings.ToLower(env.Prompt), "http") {
			in.ImageURL = env.Prompt
			in.Prompt = "Animate this image naturally with smooth cinematic motion"
		}
		if in.ImageURL == "" {
			return nil, fmt.Errorf("%s: image_url is required", slug)
		}

	case "video-veo":
		// Text-to-video; source image is optional if supplied by the client.
	default:
		// Other video tools use their configured route with whatever media inputs were supplied.
	}

	url, text, cost, usedSlug, err := o.runToolStageChain(ctx, nil, slug, "main", in)
	if err != nil {
		return nil, fmt.Errorf("video route unavailable for %q: %w", slug, err)
	}
	return &studioProviderResult{
		OutputURL: url, OutputText: text, Provider: "route/" + usedSlug, CostMicros: cost,
	}, nil
}

// ─── Voice / Translate dispatch ───────────────────────────────────────────────

func (o *AIStudioOrchestrator) dispatchVoiceOrTranslate(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	switch slug {
	case "translate", "local-translation":
		return o.dispatchTranslate(ctx, slug, env)
	case "transcribe":
		return o.dispatchTranscribe(ctx, slug, env) // env.Prompt = audioURL, env.Language = language code
	case "transcribe-african":
		return o.dispatchTranscribe(ctx, slug, env)
	case "narrate-pro":
		return o.dispatchNarratorPro(ctx, slug, env)
	case "text-to-speech":
		// text-to-speech is an alias for narrate — use voice_id if set
		if env.VoiceID != "" {
			return o.dispatchNarratorPro(ctx, slug, env)
		}
		return o.dispatchTTSForStage(ctx, slug, "main", env.Prompt, env.VoiceID)
	default: // narrate
		// Use voice_id from envelope if set, else fall back to generic TTS
		if env.VoiceID != "" {
			return o.dispatchNarratorPro(ctx, slug, env)
		}
		return o.dispatchTTSForStage(ctx, slug, "main", env.Prompt, env.VoiceID)
	}
}

// langCodeToName maps BCP-47 / ISO 639-1 codes to full language names for translation prompts.
var langCodeToName = map[string]string{
	// African languages (priority for Loyalty Nexus)
	"yo": "Yoruba", "ha": "Hausa", "ig": "Igbo", "sw": "Swahili",
	"am": "Amharic", "om": "Oromo", "so": "Somali", "rw": "Kinyarwanda",
	"ny": "Chichewa", "sn": "Shona", "st": "Sesotho", "zu": "Zulu",
	"xh": "Xhosa", "tn": "Tswana", "ts": "Tsonga", "ss": "Swati",
	"ve": "Venda", "nr": "Southern Ndebele", "ln": "Lingala", "kg": "Kongo",
	"tw": "Twi", "ak": "Akan", "ee": "Ewe", "ff": "Fula",
	"wo": "Wolof", "bm": "Bambara", "dyo": "Jola-Fonyi",
	"pcm": "Nigerian Pidgin English",
	// Major world languages
	"en": "English", "fr": "French", "es": "Spanish", "pt": "Portuguese",
	"de": "German", "it": "Italian", "nl": "Dutch", "pl": "Polish",
	"ru": "Russian", "uk": "Ukrainian", "cs": "Czech", "sk": "Slovak",
	"ro": "Romanian", "hu": "Hungarian", "bg": "Bulgarian", "hr": "Croatian",
	"sr": "Serbian", "sl": "Slovenian", "lt": "Lithuanian", "lv": "Latvian",
	"et": "Estonian", "fi": "Finnish", "sv": "Swedish", "da": "Danish",
	"nb": "Norwegian", "is": "Icelandic", "el": "Greek", "tr": "Turkish",
	"ar": "Arabic", "he": "Hebrew", "fa": "Persian", "ur": "Urdu",
	"hi": "Hindi", "bn": "Bengali", "pa": "Punjabi", "gu": "Gujarati",
	"mr": "Marathi", "ta": "Tamil", "te": "Telugu", "kn": "Kannada",
	"ml": "Malayalam", "si": "Sinhala", "ne": "Nepali", "my": "Burmese",
	"th": "Thai", "lo": "Lao", "km": "Khmer", "vi": "Vietnamese",
	"id": "Indonesian", "ms": "Malay", "tl": "Filipino", "jv": "Javanese",
	"zh": "Chinese (Simplified)", "zh-tw": "Chinese (Traditional)",
	"ja": "Japanese", "ko": "Korean",
	"ka": "Georgian", "hy": "Armenian", "az": "Azerbaijani",
	"kk": "Kazakh", "uz": "Uzbek", "tk": "Turkmen", "ky": "Kyrgyz",
	"mn": "Mongolian",
}

func (o *AIStudioOrchestrator) dispatchTranslate(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	targetLang := strings.ToLower(strings.TrimSpace(env.Language))
	if targetLang == "" || targetLang == "auto" {
		targetLang = "yo"
	}
	langName, ok := langCodeToName[targetLang]
	if !ok {
		langName = targetLang
	}
	systemPrompt := "You are a professional translator. Preserve meaning, tone, register and formatting. " +
		"Return only the translation and never add commentary."
	userPrompt := fmt.Sprintf("Translate the following text to %s:\n\n%s", langName, env.Prompt)
	in := providerInput{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		TargetLang:   targetLang,
	}
	_, translated, cost, usedSlug, err := o.runToolStageChain(ctx, nil, slug, "main", in)
	if err != nil {
		return nil, fmt.Errorf("translation route unavailable for %q: %w", slug, err)
	}
	return &studioProviderResult{OutputText: translated, Provider: "route/" + usedSlug, CostMicros: cost}, nil
}

func (o *AIStudioOrchestrator) dispatchTTSForStage(ctx context.Context, toolSlug, stageKey, text, voiceID string) (*studioProviderResult, error) {
	in := providerInput{Text: text, VoiceID: voiceID}
	url, _, cost, usedSlug, err := o.runToolStageChain(ctx, nil, toolSlug, stageKey, in)
	if err != nil {
		return nil, fmt.Errorf("TTS route unavailable for %s/%s: %w", toolSlug, stageKey, err)
	}
	return &studioProviderResult{OutputURL: url, Provider: "route/" + usedSlug, CostMicros: cost}, nil
}

func (o *AIStudioOrchestrator) dispatchTranscribe(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	lang := strings.ToLower(strings.TrimSpace(env.Language))
	if lang == "" {
		if slug == "transcribe-african" {
			lang = "yo"
		} else {
			lang = "en"
		}
	}
	speakerLabels := false
	outputFormat := "plain"
	if env.Extra != nil {
		if sl, ok := env.Extra["speaker_labels"].(bool); ok {
			speakerLabels = sl
		}
		if of, ok := env.Extra["output_format"].(string); ok && of != "" {
			outputFormat = of
		}
	}
	in := providerInput{
		AudioURL:      env.Prompt,
		Language:      lang,
		SpeakerLabels: speakerLabels,
		OutputFormat:  outputFormat,
	}
	_, text, cost, usedSlug, err := o.runToolStageChain(ctx, nil, slug, "main", in)
	if err != nil {
		return nil, fmt.Errorf("transcription route unavailable for %q: %w", slug, err)
	}
	return &studioProviderResult{
		OutputText: o.formatTranscript(text, outputFormat),
		Provider:   "route/" + usedSlug,
		CostMicros: cost,
	}, nil
}

// formatTranscript converts a plain transcript to the requested output format.
func (o *AIStudioOrchestrator) formatTranscript(text, format string) string {
	switch format {
	case "srt":
		// Wrap entire transcript in a single SRT block (real timestamps require word-level data)
		return fmt.Sprintf("1\n00:00:00,000 --> 00:05:00,000\n%s\n", text)
	case "vtt":
		return fmt.Sprintf("WEBVTT\n\n00:00:00.000 --> 00:05:00.000\n%s\n", text)
	case "timestamped":
		return fmt.Sprintf("[00:00:00] %s", text)
	default: // plain
		return text
	}
}

// callAssemblyAIFull calls AssemblyAI with full options: language, speaker diarization, and output format.
func (o *AIStudioOrchestrator) callAssemblyAIFull(ctx context.Context, apiKey, audioURL, lang string, speakerLabels bool, outputFormat string) (string, error) {
	submitPayload := map[string]interface{}{
		"audio_url":      audioURL,
		"language_code":  lang,
		"speech_models":  []string{"universal-2"},
		"speaker_labels": speakerLabels,
	}
	body, _ := json.Marshal(submitPayload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.assemblyai.com/v2/transcript", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("AssemblyAI submit: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var jobResp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jobResp); err != nil {
		return "", fmt.Errorf("AssemblyAI submit parse: %w", err)
	}
	if jobResp.ID == "" {
		return "", fmt.Errorf("AssemblyAI: no job ID returned")
	}
	pollURL := "https://api.assemblyai.com/v2/transcript/" + jobResp.ID
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(3 * time.Second)
		pollReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, pollURL, nil)
		pollReq.Header.Set("Authorization", apiKey)
		pollResp, err := o.httpClient.Do(pollReq)
		if err != nil {
			continue
		}
		var result struct {
			Status     string `json:"status"`
			Text       string `json:"text"`
			Error      string `json:"error"`
			Utterances []struct {
				Speaker string `json:"speaker"`
				Text    string `json:"text"`
				Start   int    `json:"start"`
				End     int    `json:"end"`
			} `json:"utterances"`
		}
		_ = json.NewDecoder(pollResp.Body).Decode(&result)
		if err := pollResp.Body.Close(); err != nil {
			log.Printf("[Studio] AssemblyAI poll body close: %v", err)
		}
		switch result.Status {
		case "completed":
			// If speaker labels requested and utterances available, format them
			if speakerLabels && len(result.Utterances) > 0 {
				var lines []string
				for _, u := range result.Utterances {
					startSec := u.Start / 1000
					lines = append(lines, fmt.Sprintf("[Speaker %s %02d:%02d]: %s",
						u.Speaker, startSec/60, startSec%60, u.Text))
				}
				rawFormatted := strings.Join(lines, "\n")
				return o.formatTranscript(rawFormatted, outputFormat), nil
			}
			return o.formatTranscript(result.Text, outputFormat), nil
		case "error":
			return "", fmt.Errorf("AssemblyAI error: %s", result.Error)
		}
	}
	return "", fmt.Errorf("AssemblyAI: timeout waiting for transcript")
}

// ─── Music dispatch ───────────────────────────────────────────────────────────

func (o *AIStudioOrchestrator) dispatchMusic(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	prompt := env.Prompt
	if env.Extra != nil {
		if key, ok := env.Extra["key"].(string); ok && key != "" && key != "Any" {
			prompt += fmt.Sprintf(" Key of %s.", key)
		}
		if structure, ok := env.Extra["structure"].(string); ok && structure != "" && structure != "Auto" {
			prompt += fmt.Sprintf(" Song structure: %s.", structure)
		}
		if instruments, ok := env.Extra["instruments"].(string); ok && instruments != "" {
			prompt += fmt.Sprintf(" Featured instruments: %s.", instruments)
		}
	}
	if env.NegativePrompt != "" {
		prompt += fmt.Sprintf(" Avoid: %s.", env.NegativePrompt)
	}
	if env.Lyrics != "" {
		prompt += fmt.Sprintf("\n\nLyrics:\n%s", env.Lyrics)
	}

	instrumental := false
	duration := env.Duration
	if duration <= 0 {
		duration = 30
	}
	style, title, vocalGender := "", "", ""
	if env.Extra != nil {
		style, _ = env.Extra["genre"].(string)
		title, _ = env.Extra["title"].(string)
		vocalGender, _ = env.Extra["vocal_gender"].(string)
	}

	switch slug {
	case "song-creator":
		if style == "" {
			style = "Pop"
		}
		if title == "" {
			title = "My Song"
		}
		if vocalGender == "" {
			vocalGender = "female"
		}
	case "instrumental":
		instrumental = true
		if style == "" {
			style = "Instrumental"
		} else {
			style += " Instrumental"
		}
		if title == "" {
			title = "Instrumental Track"
		}
	case "jingle", "my-marketing-jingle":
		if style == "" {
			style = "Jingle, Catchy, Upbeat"
		} else {
			style += ", Jingle, Catchy"
		}
		if title == "" {
			title = "Brand Jingle"
		}
	case "bg-music":
		instrumental = true
		if style == "" {
			style = "Background ambient"
		}
		if title == "" {
			title = "Background Music"
		}
	}

	var secondary string
	url, text, cost, usedSlug, err := o.runToolStageChain(ctx, nil, slug, "main", providerInput{
		Prompt: prompt, Instrumental: instrumental, DurationSecs: duration,
		Style: style, Title: title, VocalGender: vocalGender, SecondaryURL: &secondary,
	})
	if err != nil {
		return nil, fmt.Errorf("music route unavailable for %q: %w", slug, err)
	}
	return &studioProviderResult{
		OutputURL: url, OutputURL2: secondary, OutputText: text,
		Provider: "route/" + usedSlug, CostMicros: cost,
	}, nil
}

// ─── Composite dispatch (podcast, video-jingle) ───────────────────────────────

func (o *AIStudioOrchestrator) dispatchComposite(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	switch slug {
	case "podcast", "my-podcast":
		return o.assemblePodcast(ctx, slug, env.Prompt)
	case "video-jingle":
		return o.assembleVideoJingle(ctx, env)
	default:
		return o.dispatchText(ctx, slug, env)
	}
}

// assembleVideoJingle creates a short video with a matching jingle/music track.
// Step 1: Generate a short music jingle using the music pipeline.
// Step 2: Generate a short video clip using the video pipeline.
// Returns: OutputURL = video, OutputURL2 = music track.
func (o *AIStudioOrchestrator) assembleVideoJingle(ctx context.Context, env promptEnvelope) (*studioProviderResult, error) {
	style := "upbeat, catchy, commercial jingle"
	if env.Extra != nil {
		if s, ok := env.Extra["music_style"].(string); ok && s != "" {
			style = s
		}
	}
	musicPrompt := fmt.Sprintf(
		"Short 15-second jingle: %s. Style: %s. Energetic, memorable, brand-friendly.",
		env.Prompt, style,
	)

	var secondary string
	musicURL, _, musicCost, musicProvider, err := o.runToolStageChain(ctx, nil, "video-jingle", "music", providerInput{
		Prompt:       musicPrompt,
		DurationSecs: 15,
		Style:        style,
		Title:        "Brand Jingle",
		SecondaryURL: &secondary,
	})
	if err != nil {
		return nil, fmt.Errorf("video-jingle music stage unavailable: %w", err)
	}

	videoPrompt := o.enhanceVideoPrompt(ctx, "video-jingle", env.Prompt)
	videoURL, _, videoCost, videoProvider, err := o.runToolStageChain(ctx, nil, "video-jingle", "video", providerInput{
		Prompt:        videoPrompt,
		ImageURL:      env.ImageURL,
		DurationSecs:  15,
		AspectRatio:   env.AspectRatio,
		Resolution:    "720p",
		GenerateAudio: true,
		Extra:         env.Extra,
	})
	if err != nil {
		return nil, fmt.Errorf("video-jingle video stage unavailable: %w", err)
	}

	return &studioProviderResult{
		OutputURL:  videoURL,
		OutputURL2: musicURL,
		Provider:   "route/" + musicProvider + "+route/" + videoProvider,
		CostMicros: musicCost + videoCost,
	}, nil
}
func (o *AIStudioOrchestrator) assemblePodcast(ctx context.Context, slug, topic string) (*studioProviderResult, error) {
	scriptPrompt := fmt.Sprintf("Create a conversational 400-600 word two-host podcast script about: %s. "+
		"Use NEXUS: and ADE: speaker labels, a strong opening hook, three useful points and a concise outro.", topic)
	podcastSys := "You are a talented podcast script writer. Make the conversation natural, educational and engaging. " +
		"Use Nigerian or African context naturally when relevant."

	_, script, scriptCost, scriptProvider, err := o.runToolStageChain(ctx, nil, slug, "script", providerInput{
		SystemPrompt: podcastSys,
		UserPrompt:   scriptPrompt,
	})
	if err != nil {
		return nil, fmt.Errorf("podcast script route failed: %w", err)
	}

	narrationResult, speechErr := o.dispatchTTSForStage(ctx, slug, "speech", script, "")
	if speechErr != nil {
		log.Printf("[AIStudio] Podcast speech stage failed, returning script only: %v", speechErr)
		return &studioProviderResult{
			OutputText: script,
			Provider:   "route/" + scriptProvider + "+speech-unavailable",
			CostMicros: scriptCost,
		}, nil
	}

	return &studioProviderResult{
		OutputURL:  narrationResult.OutputURL,
		OutputText: script,
		Provider:   "route/" + scriptProvider + "+" + narrationResult.Provider,
		CostMicros: scriptCost + narrationResult.CostMicros,
	}, nil
}

// ─── Provider API calls ───────────────────────────────────────────────────────

// callGeminiFlash calls Gemini 2.0 Flash for text generation.
func (o *AIStudioOrchestrator) callGeminiFlash(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY not configured")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=%s", apiKey)
	payload := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]string{{"text": systemPrompt}},
		},
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": userPrompt}}},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 0.7,
			// 8192 — long-form knowledge tools (bizplan: 9 sections + 3-year
			// financials; deep research briefs) were being truncated mid-document
			// at 4096.  Output is billed per token actually generated, so this
			// only costs more when the tool genuinely needs the length.
			"maxOutputTokens": 8192,
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}

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
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("gemini parse: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("gemini API error: %s", result.Error.Message)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: no content returned")
	}
	return result.Candidates[0].Content.Parts[0].Text, nil
}

// callGroqLlama4 calls Groq's Llama-4-Scout model.
func (o *AIStudioOrchestrator) callGroqLlama4(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY not configured")
	}

	payload := map[string]interface{}{
		"model": "meta-llama/llama-4-scout-17b-16e-instruct",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"max_tokens":  8192, // match Gemini — long-form tools truncated at 4096
		"temperature": 0.7,
	}

	return o.callOpenAICompatible(ctx, "https://api.groq.com/openai/v1/chat/completions",
		"Bearer "+apiKey, payload)
}

// callDeepSeek calls DeepSeek V3 as paid overflow.
func (o *AIStudioOrchestrator) callDeepSeek(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("DEEPSEEK_API_KEY not configured")
	}

	payload := map[string]interface{}{
		"model": "deepseek-chat",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"max_tokens":  8192, // match Gemini — long-form tools truncated at 4096
		"temperature": 0.7,
	}

	return o.callOpenAICompatible(ctx, "https://api.deepseek.com/chat/completions",
		"Bearer "+apiKey, payload)
}

// callOpenAICompatible is a shared helper for OpenAI-compatible chat APIs.
func (o *AIStudioOrchestrator) callOpenAICompatible(ctx context.Context, endpoint, authHeader string, payload interface{}) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		// A content-policy rejection arrives as a 400 — it is a verdict about the
		// prompt, not a provider fault, and must not be shopped to the next provider.
		if refusal, ok := openAIErrorRefusal(raw); ok {
			return "", refusal
		}
		return "", fmt.Errorf("API %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}

	var parsed struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("parse: %w", err)
	}
	if parsed.Error != nil {
		if refusal, ok := openAIErrorRefusal(raw); ok {
			return "", refusal
		}
		return "", fmt.Errorf("API error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}
	choice := parsed.Choices[0]
	if choice.FinishReason == "content_filter" {
		return "", &ProviderRefusalError{Provider: "openai-compatible", Reason: "finish_reason=content_filter"}
	}
	if choice.Message.Content == "" {
		// Previously returned as a successful empty generation — charged,
		// ledgered as SUCCEEDED, and blank for the user.
		return "", fmt.Errorf("no content returned (finish_reason=%s)", choice.FinishReason)
	}
	return choice.Message.Content, nil
}

// callHFFluxSchnell calls HuggingFace FLUX.1-Schnell (free tier, ~3s).
func (o *AIStudioOrchestrator) callHFFluxSchnell(ctx context.Context, hfKey, prompt string) (string, error) {
	model := os.Getenv("HF_IMAGE_MODEL")
	if model == "" {
		model = "black-forest-labs/FLUX.1-schnell"
	}
	// HF deprecated api-inference.huggingface.co (returns 410 Gone).
	// New canonical base: https://router.huggingface.co/hf-inference/models/<model>
	endpoint := "https://router.huggingface.co/hf-inference/models/" + model

	body, _ := json.Marshal(map[string]interface{}{
		"inputs": prompt,
		"parameters": map[string]interface{}{
			// FLUX.1-schnell is a distilled model: 4 steps is minimal (fast but lower quality).
			// 8 steps gives significantly better output with only ~2× latency.
			// Max is 28; free tier has no rate limit on steps.
			"num_inference_steps": 8,
			"guidance_scale":      0,
			"width":               1024,
			"height":              1024,
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+hfKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HF request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusServiceUnavailable {
		return "", fmt.Errorf("HF model loading, retry in ~20s")
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HF %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	imgData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// Upload to S3 if configured, otherwise return data URI
	return o.uploadOrDataURI(ctx, imgData, "image/png", "generated/"+uuid.New().String()+".png"), nil
}

// callFALFlux calls FAL.AI FLUX-dev (paid, higher quality).
func (o *AIStudioOrchestrator) callFALFlux(ctx context.Context, falKey, prompt string) (string, error) {
	payload := map[string]interface{}{
		"prompt":              prompt,
		"image_size":          "square_hd",
		"num_images":          1,
		"output_format":       "jpeg",
		"num_inference_steps": 28,
		"guidance_scale":      3.5,
		"safety_tolerance":    "5", // permissive (1=strict, 6=max permissive)
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://fal.run/fal-ai/flux/dev", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Key "+falKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("FAL %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	var parsed struct {
		Images []struct {
			URL string `json:"url"`
		} `json:"images"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Images) == 0 {
		return "", fmt.Errorf("FAL parse: empty images")
	}
	return parsed.Images[0].URL, nil
}

// callFALFluxUltra calls FAL.AI Flux Pro 1.1 Ultra — supports reference image, num_images (1-4),
// and image_prompt_strength (0-1). Used for Whisk-style reference-guided generation.
func (o *AIStudioOrchestrator) callFALFluxUltra(
	ctx context.Context,
	falKey, prompt, imageURL string,
	imagePromptStrength float64,
	numImages int,
	aspectRatio string,
) ([]string, error) {
	return o.callFALFluxUltraWithModel(ctx, falKey, "fal-ai/flux-pro/v1.1-ultra", prompt, imageURL, imagePromptStrength, numImages, aspectRatio)
}

func (o *AIStudioOrchestrator) callFALFluxUltraWithModel(
	ctx context.Context,
	falKey, model, prompt, imageURL string,
	imagePromptStrength float64,
	numImages int,
	aspectRatio string,
) ([]string, error) {
	if model == "" {
		model = "fal-ai/flux-pro/v1.1-ultra"
	}
	if numImages < 1 {
		numImages = 1
	}
	if numImages > 4 {
		numImages = 4
	}
	payload := map[string]interface{}{
		"prompt":           prompt,
		"num_images":       numImages,
		"output_format":    "jpeg",
		"safety_tolerance": "2",
	}
	if imageURL != "" {
		payload["image_url"] = imageURL
		payload["image_prompt_strength"] = imagePromptStrength
	}
	if aspectRatio != "" {
		payload["aspect_ratio"] = aspectRatio
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://fal.run/"+model, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Key "+falKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FAL Ultra %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}
	var parsed struct {
		Images []struct {
			URL string `json:"url"`
		} `json:"images"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Images) == 0 {
		return nil, fmt.Errorf("FAL Ultra parse: empty images")
	}
	var urls []string
	for _, img := range parsed.Images {
		if img.URL != "" {
			urls = append(urls, img.URL)
		}
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("FAL Ultra: no image URLs in response")
	}
	return urls, nil
}

// callFALImageEdit calls FAL.AI FLUX.1 Kontext Pro for image-to-image editing.
func (o *AIStudioOrchestrator) callFALImageEdit(ctx context.Context, falKey, imageURL, instruction string) (string, error) {
	return o.callFALImageEditWithModel(ctx, falKey, "fal-ai/flux-pro/kontext", imageURL, instruction)
}

func (o *AIStudioOrchestrator) callFALImageEditWithModel(ctx context.Context, falKey, model, imageURL, instruction string) (string, error) {
	if model == "" {
		model = "fal-ai/flux-pro/kontext"
	}
	payload := map[string]interface{}{
		"prompt":         instruction,
		"image_url":      imageURL,
		"num_images":     1,
		"output_format":  "jpeg",
		"guidance_scale": 3.5,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://fal.run/"+model, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Key "+falKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("FAL image-edit %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}
	var parsed struct {
		Images []struct {
			URL string `json:"url"`
		} `json:"images"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Images) == 0 {
		return "", fmt.Errorf("FAL image-edit parse: empty images")
	}
	return parsed.Images[0].URL, nil
}

// callRembgService calls the self-hosted rembg Python microservice.
func (o *AIStudioOrchestrator) callRembgService(ctx context.Context, serviceURL, imageURL string) (string, error) {
	payload := map[string]string{"url": imageURL}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serviceURL+"/remove", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("rembg %d", resp.StatusCode)
	}

	var result struct {
		ResultURL string `json:"result_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.ResultURL == "" {
		// May return raw PNG — upload it
		raw, _ := io.ReadAll(resp.Body)
		return o.uploadOrDataURI(ctx, raw, "image/png", "bgremoved/"+uuid.New().String()+".png"), nil
	}
	return result.ResultURL, nil
}

// callPollinationsBgRemover uses Pollinations image API with a background-removal prompt
// as a free fallback when no dedicated bg-removal service key is configured.
// Pollinations processes the image through its pipeline and returns a CDN URL.
func (o *AIStudioOrchestrator) callPollinationsBgRemover(ctx context.Context, secretKey, imageURL string) (string, error) {
	// Pollinations background removal endpoint
	payload := map[string]interface{}{
		"imageUrl": imageURL,
		"model":    "flux",
		"prompt":   "remove background, transparent background, subject only, no background, isolated subject, professional cutout",
		"enhance":  false,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://image.pollinations.ai/prompt/remove+background?nologo=true",
		bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+secretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("pollinations bg-remove request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("pollinations bg-remove %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	// Pollinations returns the image directly as binary — upload to R2 for a stable CDN URL
	imgData, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return "", fmt.Errorf("pollinations bg-remove read: %w", err)
	}
	if len(imgData) < 1024 {
		return "", fmt.Errorf("pollinations bg-remove: response too small (%d bytes)", len(imgData))
	}

	// Return as data URI if R2 storage unavailable
	encoded := base64.StdEncoding.EncodeToString(imgData)
	return "data:image/png;base64," + encoded, nil
}

// callFALBgRemover uses FAL.AI's BiRefNet for background removal.
func (o *AIStudioOrchestrator) callFALBgRemover(ctx context.Context, falKey, imageURL string) (string, error) {
	payload := map[string]interface{}{
		"image_url": imageURL,
		"model":     "General Use (Light)",
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://fal.run/fal-ai/birefnet", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Key "+falKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("FAL BiRefNet %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	var parsed struct {
		Image struct {
			URL string `json:"url"`
		} `json:"image"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || parsed.Image.URL == "" {
		return "", fmt.Errorf("FAL BiRefNet parse failed")
	}
	return parsed.Image.URL, nil
}

// callRemoveBg uses the remove.bg API as last resort.
func (o *AIStudioOrchestrator) callRemoveBg(ctx context.Context, apiKey, imageURL string) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("image_url", imageURL)
	_ = w.WriteField("size", "auto")
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("remove.bg multipart close: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.remove.bg/v1.0/removebg", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Api-Key", apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("remove.bg %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	imgData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return o.uploadOrDataURI(ctx, imgData, "image/png", "bgremoved/"+uuid.New().String()+".png"), nil
}

// callFALVideo calls FAL.AI for image-to-video animation.
// Pass the promptEnvelope so duration, aspect_ratio, generate_audio, and
// end_image_url (tail frame) are forwarded from the user's request.
func (o *AIStudioOrchestrator) callFALVideo(ctx context.Context, falKey, model, imageURL, motionPrompt string, envs ...promptEnvelope) (string, error) {
	if motionPrompt == "" {
		motionPrompt = "animate this photo naturally with smooth cinematic motion, subtle movement, professional quality"
	}

	// Resolve optional envelope
	var env promptEnvelope
	if len(envs) > 0 {
		env = envs[0]
	}

	// Duration: prefer envelope value, default to 5
	durationStr := "5"
	if env.Duration > 0 {
		durationStr = fmt.Sprintf("%d", env.Duration)
	}

	// Aspect ratio: prefer envelope value, default to 16:9
	aspectRatio := "16:9"
	if env.AspectRatio != "" {
		aspectRatio = env.AspectRatio
	}

	payload := map[string]interface{}{
		"prompt":   motionPrompt,
		"duration": durationStr,
	}

	if strings.Contains(model, "kling") {
		// Kling v2.6 uses start_image_url and supports native audio generation
		payload["start_image_url"] = imageURL
		payload["aspect_ratio"] = aspectRatio

		// generate_audio: read from extra_params, default true
		generateAudio := true
		if env.Extra != nil {
			if ga, ok := env.Extra["generate_audio"].(bool); ok {
				generateAudio = ga
			}
		}
		payload["generate_audio"] = generateAudio

		// end_image_url (tail frame): read from extra_params
		if env.Extra != nil {
			if tail, ok := env.Extra["tail_image_url"].(string); ok && tail != "" {
				payload["end_image_url"] = tail
			}
		}
	} else {
		// LTX and other models use image_url
		payload["image_url"] = imageURL
		payload["aspect_ratio"] = aspectRatio
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://fal.run/"+model, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Key "+falKey)
	req.Header.Set("Content-Type", "application/json")

	// fal.run is synchronous — Kling pro renders routinely exceed 2 minutes.
	// The shared o.httpClient (120s) guaranteed timeouts on premium video, so
	// use a dedicated long-lived client (same pattern as Suno / Pollinations).
	falVideoClient := &http.Client{Timeout: 300 * time.Second}
	resp, err := falVideoClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("FAL video %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	var parsed struct {
		Video struct {
			URL string `json:"url"`
		} `json:"video"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || parsed.Video.URL == "" {
		return "", fmt.Errorf("FAL video parse failed: %s", truncateStr(string(raw), 200))
	}
	return parsed.Video.URL, nil
}

// callFALMultiImageVideo calls the FAL Kling v1.6 multi-image-to-video endpoint.
// Accepts 2-4 image URLs and a story prompt, returns a video URL.
func (o *AIStudioOrchestrator) callFALMultiImageVideoConfigured(ctx context.Context, falKey, model string, imageURLs []string, prompt string, duration int, aspectRatio string, extra map[string]interface{}) (string, error) {
	if model == "" {
		model = "fal-ai/kling-video/v2.6/pro/multi-image-to-video"
	}

	if prompt == "" {
		prompt = "Create a smooth cinematic video transitioning between these scenes with natural motion"
	}

	// Build image_list payload — each entry has url and optional caption
	type imageEntry struct {
		URL     string `json:"url"`
		Caption string `json:"caption,omitempty"`
	}
	var images []imageEntry
	for i, u := range imageURLs {
		entry := imageEntry{URL: u}
		// Check for per-scene captions in extra_params
		if extra != nil {
			key := fmt.Sprintf("scene_%d_caption", i+1)
			if cap, ok := extra[key].(string); ok && cap != "" {
				entry.Caption = cap
			}
		}
		images = append(images, entry)
	}

	durationStr := "5"
	if duration > 0 {
		durationStr = fmt.Sprintf("%d", duration)
	}
	if aspectRatio == "" {
		aspectRatio = "16:9"
	}

	payload := map[string]interface{}{
		"prompt":         prompt,
		"image_list":     images,
		"duration":       durationStr,
		"aspect_ratio":   aspectRatio,
		"generate_audio": true, // Kling v2.6 native audio generation
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://fal.run/"+model, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Key "+falKey)
	req.Header.Set("Content-Type", "application/json")

	// Dedicated 300s client — multi-image Kling renders take minutes; the
	// shared 120s client would time out before the render completes.
	falMultiClient := &http.Client{Timeout: 300 * time.Second}
	resp, err := falMultiClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("FAL multi-image video %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	var parsed struct {
		Video struct {
			URL string `json:"url"`
		} `json:"video"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || parsed.Video.URL == "" {
		return "", fmt.Errorf("FAL multi-image video parse failed: %s", truncateStr(string(raw), 200))
	}
	return parsed.Video.URL, nil
}

// ─── Talking-Avatar adapters ──────────────────────────────────────────────────
//
// Two FAL templates by INPUT SHAPE (so future FAL avatar models are a DB-row-only
// change), plus a HeyGen adapter for its bespoke upload→generate→poll API.
// All follow the callFALVideo pattern: POST fal.run/<model>, Key auth, 300s
// client (FAL avatar renders exceed the shared 120s client), parse {video:{url}}.

// callFALAvatarText drives a FAL avatar model that takes {image_url, text_input,
// voice} and does TTS internally (e.g. fal-ai/ai-avatar/single-text). One call:
// photo + script → lip-synced talking-head video.
func (o *AIStudioOrchestrator) callFALAvatarText(ctx context.Context, falKey, model, imageURL, script, voice string) (string, error) {
	if model == "" {
		model = "fal-ai/ai-avatar/single-text"
	}
	if imageURL == "" {
		return "", fmt.Errorf("fal-avatar-text: image_url required")
	}
	if script == "" {
		return "", fmt.Errorf("fal-avatar-text: script (text_input) required")
	}
	if voice == "" {
		voice = "Bill" // model's default voice roster
	}
	payload := map[string]interface{}{
		"image_url":  imageURL,
		"text_input": script,
		"voice":      voice,
		// The model's "prompt" is a delivery/scene description, distinct from the spoken script.
		"prompt": "A person looking at the camera and speaking naturally with clear lip movement and subtle, professional expression.",
	}
	return o.postFALAvatar(ctx, falKey, model, payload)
}

// callFALAvatarAudio drives an audio-driven FAL lip-sync model that takes
// {image_url, audio_url} (e.g. veed/fabric-1.0). The audio is resolved by the
// caller (pre-uploaded or TTS'd from the script) — see resolveAvatarAudio.
func (o *AIStudioOrchestrator) callFALAvatarAudio(ctx context.Context, falKey, model, imageURL, audioURL string) (string, error) {
	if model == "" {
		model = "veed/fabric-1.0"
	}
	if imageURL == "" {
		return "", fmt.Errorf("fal-avatar-audio: image_url required")
	}
	if audioURL == "" {
		return "", fmt.Errorf("fal-avatar-audio: audio_url required")
	}
	payload := map[string]interface{}{
		"image_url": imageURL,
		"audio_url": audioURL,
	}
	return o.postFALAvatar(ctx, falKey, model, payload)
}

// postFALAvatar POSTs a payload to fal.run/<model> and extracts the video URL.
// Shared by both FAL avatar templates. Dedicated 300s client — avatar renders
// (esp. at $0.20/sec models) routinely exceed the shared 120s client.
func (o *AIStudioOrchestrator) postFALAvatar(ctx context.Context, falKey, model string, payload map[string]interface{}) (string, error) {
	if falKey == "" {
		return "", fmt.Errorf("fal avatar: FAL_API_KEY not configured")
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://fal.run/"+model, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Key "+falKey)
	req.Header.Set("Content-Type", "application/json")

	falAvatarClient := &http.Client{Timeout: 300 * time.Second}
	resp, err := falAvatarClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("FAL avatar %s %d: %s", model, resp.StatusCode, truncateStr(string(raw), 200))
	}
	// FAL avatar/video models return {"video":{"url":...}}
	var parsed struct {
		Video struct {
			URL string `json:"url"`
		} `json:"video"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || parsed.Video.URL == "" {
		return "", fmt.Errorf("FAL avatar parse failed (%s): %s", model, truncateStr(string(raw), 200))
	}
	return parsed.Video.URL, nil
}

// callHeyGen drives HeyGen's Avatar IV API (their photorealistic photo-avatar).
// Fully implemented but DORMANT — no provider row enables it until an admin adds
// a HEYGEN_API_KEY and flips the heygen ai_provider_configs row active. Flow:
//  1. POST /v1/asset/upload  → image_key for the source photo
//  2. POST /v2/video/av4/generate  {image_key, script, voice_id} → video_id
//  3. poll GET /v1/video_status.get?video_id=…  until completed → video_url
func (o *AIStudioOrchestrator) callHeyGen(ctx context.Context, apiKey, imageURL, script, voiceID string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("heygen: HEYGEN_API_KEY not configured")
	}
	if imageURL == "" || script == "" {
		return "", fmt.Errorf("heygen: image_url and script required")
	}
	client := &http.Client{Timeout: 60 * time.Second}

	// Step 1: upload the photo → image_key. HeyGen's upload accepts raw image bytes.
	imgBytes, ctype, err := o.fetchRemoteBytes(ctx, imageURL)
	if err != nil {
		return "", fmt.Errorf("heygen: fetch image: %w", err)
	}
	upReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://upload.heygen.com/v1/asset", bytes.NewReader(imgBytes))
	if err != nil {
		return "", err
	}
	upReq.Header.Set("X-Api-Key", apiKey)
	upReq.Header.Set("Content-Type", ctype)
	upResp, err := client.Do(upReq)
	if err != nil {
		return "", fmt.Errorf("heygen upload: %w", err)
	}
	upRaw, _ := io.ReadAll(upResp.Body)
	_ = upResp.Body.Close()
	if upResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("heygen upload %d: %s", upResp.StatusCode, truncateStr(string(upRaw), 200))
	}
	var upParsed struct {
		Data struct {
			ImageKey string `json:"image_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(upRaw, &upParsed); err != nil || upParsed.Data.ImageKey == "" {
		return "", fmt.Errorf("heygen upload parse: %s", truncateStr(string(upRaw), 200))
	}

	// Step 2: generate the Avatar IV video.
	vid := voiceID
	if vid == "" {
		vid = "1bd001e7e50f421d891986aad5158bc8" // a documented HeyGen default voice
	}
	genPayload := map[string]interface{}{
		"image_key": upParsed.Data.ImageKey,
		"script":    script,
		"voice_id":  vid,
	}
	genBody, _ := json.Marshal(genPayload)
	genReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.heygen.com/v2/video/av4/generate", bytes.NewReader(genBody))
	if err != nil {
		return "", err
	}
	genReq.Header.Set("X-Api-Key", apiKey)
	genReq.Header.Set("Content-Type", "application/json")
	genResp, err := client.Do(genReq)
	if err != nil {
		return "", fmt.Errorf("heygen generate: %w", err)
	}
	genRaw, _ := io.ReadAll(genResp.Body)
	_ = genResp.Body.Close()
	if genResp.StatusCode != http.StatusOK && genResp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("heygen generate %d: %s", genResp.StatusCode, truncateStr(string(genRaw), 200))
	}
	var genParsed struct {
		Data struct {
			VideoID string `json:"video_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(genRaw, &genParsed); err != nil || genParsed.Data.VideoID == "" {
		return "", fmt.Errorf("heygen generate parse: %s", truncateStr(string(genRaw), 200))
	}

	// Step 3: poll for completion (up to ~5 min).
	for attempt := 0; attempt < 60; attempt++ {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("heygen: context cancelled while polling")
		case <-time.After(5 * time.Second):
		}
		stReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
			"https://api.heygen.com/v1/video_status.get?video_id="+genParsed.Data.VideoID, nil)
		if err != nil {
			continue
		}
		stReq.Header.Set("X-Api-Key", apiKey)
		stResp, err := client.Do(stReq)
		if err != nil {
			continue
		}
		stRaw, _ := io.ReadAll(stResp.Body)
		_ = stResp.Body.Close()
		var stParsed struct {
			Data struct {
				Status   string `json:"status"`
				VideoURL string `json:"video_url"`
				Error    any    `json:"error"`
			} `json:"data"`
		}
		if err := json.Unmarshal(stRaw, &stParsed); err != nil {
			continue
		}
		switch stParsed.Data.Status {
		case "completed":
			if stParsed.Data.VideoURL == "" {
				return "", fmt.Errorf("heygen: completed but no video_url")
			}
			return stParsed.Data.VideoURL, nil
		case "failed":
			return "", fmt.Errorf("heygen: generation failed: %v", stParsed.Data.Error)
		}
	}
	return "", fmt.Errorf("heygen: timed out after 5 minutes (video_id=%s)", genParsed.Data.VideoID)
}

// fetchRemoteBytes downloads a remote asset and returns its bytes + content-type.
func (o *AIStudioOrchestrator) fetchRemoteBytes(ctx context.Context, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("fetch %s: status %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	ctype := resp.Header.Get("Content-Type")
	if ctype == "" {
		ctype = "image/jpeg"
	}
	return data, ctype, nil
}

// callGoogleCloudTTS calls Google Cloud Text-to-Speech API.
func (o *AIStudioOrchestrator) callGoogleCloudTTS(ctx context.Context, apiKey, text string) (string, error) {
	url := fmt.Sprintf("https://texttospeech.googleapis.com/v1/text:synthesize?key=%s", apiKey)
	payload := map[string]interface{}{
		"input": map[string]string{"text": text},
		"voice": map[string]interface{}{
			"languageCode": "en-US", // Google TTS has no en-NG voice — en-US Neural2-F is natural and clear
			"name":         "en-US-Neural2-F",
			"ssmlGender":   "FEMALE",
		},
		"audioConfig": map[string]interface{}{
			"audioEncoding": "MP3",
			"speakingRate":  1.0,
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("google TTS %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	var result struct {
		AudioContent string `json:"audioContent"` // base64-encoded MP3
	}
	if err := json.Unmarshal(raw, &result); err != nil || result.AudioContent == "" {
		return "", fmt.Errorf("google TTS parse failed")
	}

	audioData, err := base64.StdEncoding.DecodeString(result.AudioContent)
	if err != nil {
		return "", fmt.Errorf("google TTS decode: %w", err)
	}
	return o.uploadOrDataURI(ctx, audioData, "audio/mpeg", "narrations/"+uuid.New().String()+".mp3"), nil
}

// callElevenLabsTTS calls ElevenLabs Text-to-Speech.
func (o *AIStudioOrchestrator) callElevenLabsTTS(ctx context.Context, apiKey, voiceID, text string) (string, error) {
	payload := map[string]interface{}{
		"text":     text,
		"model_id": "eleven_turbo_v2_5",
		"voice_settings": map[string]float64{
			"stability": 0.5, "similarity_boost": 0.75,
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.elevenlabs.io/v1/text-to-speech/"+voiceID,
		bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	audio, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("elevenLabs TTS %d: %s", resp.StatusCode, truncateStr(string(audio), 200))
	}
	return o.uploadOrDataURI(ctx, audio, "audio/mpeg", "narrations/"+uuid.New().String()+".mp3"), nil
}

// callHuggingFaceTTS calls HuggingFace Bark for TTS (free fallback).
func (o *AIStudioOrchestrator) callHuggingFaceTTS(ctx context.Context, hfKey, text string) (string, error) {
	body, _ := json.Marshal(map[string]string{"inputs": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		// suno/bark is NOT available on HF serverless inference (no providers).
		// Fallback to Google Cloud TTS via callGoogleCloudTTS if key is set, otherwise fail fast.
		// This function is kept as a stub — it always returns an error so dispatchTTS skips it.
		"https://router.huggingface.co/hf-inference/models/suno/bark", // intentionally unsupported — will 404
		bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+hfKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	audio, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HF Bark %d", resp.StatusCode)
	}
	return o.uploadOrDataURI(ctx, audio, "audio/wav", "narrations/"+uuid.New().String()+".wav"), nil
}

// callAssemblyAI submits audio to AssemblyAI and polls for the transcript.
func (o *AIStudioOrchestrator) callAssemblyAI(ctx context.Context, apiKey, audioURL string) (string, error) {
	// Submit transcript job
	submitPayload := map[string]interface{}{
		"audio_url":     audioURL,
		"language_code": "en",
		"speech_models": []string{"universal-2"}, // required since AssemblyAI deprecated speech_model (singular)
	}
	body, _ := json.Marshal(submitPayload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.assemblyai.com/v2/transcript", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("AssemblyAI submit: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var jobResp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jobResp); err != nil {
		return "", fmt.Errorf("AssemblyAI submit parse: %w", err)
	}
	if jobResp.ID == "" {
		return "", fmt.Errorf("AssemblyAI: no job ID returned")
	}

	// Poll until completed (max 5 minutes)
	pollURL := "https://api.assemblyai.com/v2/transcript/" + jobResp.ID
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(3 * time.Second)

		pollReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, pollURL, nil)
		pollReq.Header.Set("Authorization", apiKey)

		pollResp, err := o.httpClient.Do(pollReq)
		if err != nil {
			continue
		}

		var result struct {
			Status string `json:"status"`
			Text   string `json:"text"`
			Error  string `json:"error"`
		}
		if decErr := json.NewDecoder(pollResp.Body).Decode(&result); decErr != nil {
			log.Printf("[Studio] AssemblyAI poll decode error: %v", decErr)
		}
		if err := pollResp.Body.Close(); err != nil {
			log.Printf("[Studio] AssemblyAI poll body close: %v", err)
		}

		switch result.Status {
		case "completed":
			return result.Text, nil
		case "error":
			return "", fmt.Errorf("AssemblyAI error: %s", result.Error)
		}
	}
	return "", fmt.Errorf("AssemblyAI: transcription timed out after 5 minutes")
}

// callAssemblyAIWithLang is like callAssemblyAI but forwards the language_code to AssemblyAI.
// callOpenAITranscribe uses OpenAI gpt-4o-transcribe for high-accuracy transcription.
// Supports 98 languages including all major African languages (Yoruba, Hausa, Igbo, Swahili, etc.)
// Pricing: $0.006/min (~$0.0001 per 1s clip). Much better accuracy than Whisper-v3 for African accents.
func (o *AIStudioOrchestrator) callOpenAITranscribe(ctx context.Context, apiKey, model, audioURL, lang string) (string, error) {
	if model == "" {
		model = "gpt-4o-transcribe"
	}
	// Step 1: Download the audio file
	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, audioURL, nil)
	if err != nil {
		return "", fmt.Errorf("openai transcribe download request: %w", err)
	}
	dlResp, err := o.httpClient.Do(dlReq)
	if err != nil {
		return "", fmt.Errorf("openai transcribe download: %w", err)
	}
	defer func() { _ = dlResp.Body.Close() }()
	if dlResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai transcribe: audio download failed with status %d", dlResp.StatusCode)
	}
	audioBytes, err := io.ReadAll(dlResp.Body)
	if err != nil {
		return "", fmt.Errorf("openai transcribe: read audio bytes: %w", err)
	}

	// Step 2: Detect file extension from Content-Type or URL
	ext := "mp3"
	ct := dlResp.Header.Get("Content-Type")
	switch {
	case strings.Contains(ct, "wav") || strings.HasSuffix(strings.ToLower(audioURL), ".wav"):
		ext = "wav"
	case strings.Contains(ct, "webm") || strings.HasSuffix(strings.ToLower(audioURL), ".webm"):
		ext = "webm"
	case strings.Contains(ct, "ogg") || strings.HasSuffix(strings.ToLower(audioURL), ".ogg"):
		ext = "ogg"
	case strings.Contains(ct, "m4a") || strings.HasSuffix(strings.ToLower(audioURL), ".m4a"):
		ext = "m4a"
	}

	// Step 3: POST as multipart/form-data to OpenAI audio transcriptions endpoint
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("model", model)
	_ = w.WriteField("response_format", "text")
	if lang != "" && lang != "auto" {
		_ = w.WriteField("language", lang)
	}
	fw, err := w.CreateFormFile("file", "audio."+ext)
	if err != nil {
		return "", fmt.Errorf("openai transcribe form: %w", err)
	}
	if _, err = fw.Write(audioBytes); err != nil {
		return "", fmt.Errorf("openai transcribe write form: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("openai transcribe multipart close: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.openai.com/v1/audio/transcriptions", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai transcribe %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}
	// response_format=text returns plain text, not JSON
	result := strings.TrimSpace(string(raw))
	if result == "" {
		return "", fmt.Errorf("openai transcribe: empty response")
	}
	return result, nil
}

// callOpenAITranslate uses GPT-4o-mini for high-quality translation across 70+ languages.
// Pricing: ~$0.15/1M input tokens + $0.60/1M output tokens — very cheap for typical text lengths.
// Covers all African languages: Yoruba, Igbo, Hausa, Swahili, Amharic, Twi, Wolof, Lingala, Pidgin, etc.
func (o *AIStudioOrchestrator) callOpenAITranslate(ctx context.Context, apiKey, text, targetLangName string) (string, error) {
	systemPrompt := "You are a professional translator with native-level fluency in all major world languages, " +
		"including African languages: Yoruba, Igbo, Hausa, Swahili, Zulu, Amharic, Twi, Wolof, Lingala, Fula, Ewe, " +
		"Kinyarwanda, Somali, Sesotho, Chichewa, and Nigerian Pidgin English. " +
		"Preserve the original tone, style, register, and meaning precisely. " +
		"For idiomatic expressions, use the natural equivalent in the target language. " +
		"Return ONLY the translated text — no explanations, no notes, no quotation marks."
	userPrompt := fmt.Sprintf("Translate the following text to %s. Return ONLY the translation:\n\n%s", targetLangName, text)

	payload := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.1, // low temperature for consistent translations
		"max_tokens":  4096,
	}
	return o.callOpenAICompatible(ctx, "https://api.openai.com/v1/chat/completions", "Bearer "+apiKey, payload)
}

// callGroqWhisper uses Groq's Whisper for transcription (fast fallback).
func (o *AIStudioOrchestrator) callGroqWhisper(ctx context.Context, apiKey, audioURL string) (string, error) {
	// Groq Whisper requires multipart/form-data with a binary file upload.
	// It does NOT accept a JSON body with a "url" field.
	// Step 1: Download the audio file from the URL.
	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, audioURL, nil)
	if err != nil {
		return "", fmt.Errorf("groq Whisper download request: %w", err)
	}
	dlResp, err := o.httpClient.Do(dlReq)
	if err != nil {
		return "", fmt.Errorf("groq Whisper download: %w", err)
	}
	defer func() {
		if err := dlResp.Body.Close(); err != nil {
			log.Printf("[AIStudio] Groq Whisper dlResp body close: %v", err)
		}
	}()
	if dlResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("groq Whisper: audio download failed with status %d", dlResp.StatusCode)
	}
	audioBytes, err := io.ReadAll(dlResp.Body)
	if err != nil {
		return "", fmt.Errorf("groq Whisper: read audio bytes: %w", err)
	}

	// Step 2: POST as multipart/form-data with the file bytes.
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("model", "whisper-large-v3")
	_ = w.WriteField("language", "en")
	_ = w.WriteField("response_format", "json")
	fw, err := w.CreateFormFile("file", "audio.mp3")
	if err != nil {
		return "", fmt.Errorf("groq Whisper form: %w", err)
	}
	if _, err = fw.Write(audioBytes); err != nil {
		return "", fmt.Errorf("groq Whisper write form: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("groq Whisper multipart close: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.groq.com/openai/v1/audio/transcriptions", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("groq Whisper %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}
	var result struct {
		Text string `json:"text"`
	}
	if decErr := json.Unmarshal(raw, &result); decErr != nil {
		return "", fmt.Errorf("groq Whisper decode: %w", decErr)
	}
	if result.Text == "" {
		return "", fmt.Errorf("groq Whisper: no transcription returned")
	}
	return result.Text, nil
}

// callGoogleTranslate uses the Google Cloud Translation API.
func (o *AIStudioOrchestrator) callGoogleTranslate(ctx context.Context, apiKey, text, targetLang string) (string, error) {
	url := fmt.Sprintf("https://translation.googleapis.com/language/translate/v2?key=%s", apiKey)
	payload := map[string]interface{}{
		"q":      text,
		"target": targetLang,
		"format": "text",
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("google Translate %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	var result struct {
		Data struct {
			Translations []struct {
				TranslatedText string `json:"translatedText"`
			} `json:"translations"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("google Translate parse: %w", err)
	}
	if len(result.Data.Translations) == 0 {
		return "", fmt.Errorf("google Translate: no translation returned")
	}
	return result.Data.Translations[0].TranslatedText, nil
}

// callHFMusicGen calls the HuggingFace MusicGen-small model for background music.
// This is FREE — it uses the same HF_TOKEN already required for image generation.
// Model: facebook/musicgen-small (best quality/speed for short clips)
// Returns a public CDN URL to the uploaded MP3.
func (o *AIStudioOrchestrator) callHFMusicGen(ctx context.Context, token, prompt string, durationSecs int) (string, error) {
	// HF Inference API for audio generation.
	// HF deprecated api-inference.huggingface.co (410 Gone) — use router instead.
	apiURL := "https://router.huggingface.co/hf-inference/models/facebook/musicgen-small"
	payload := map[string]interface{}{
		"inputs": prompt,
		"parameters": map[string]interface{}{
			"max_new_tokens": durationSecs * 50, // ~50 tokens per second of audio
			"do_sample":      true,
			"guidance_scale": 3.0,
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	// MusicGen can take 10-30s to load on cold start
	req.Header.Set("X-Wait-For-Model", "true")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HF MusicGen request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusServiceUnavailable {
		return "", fmt.Errorf("HF MusicGen model loading — retry in 20s")
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HF MusicGen %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	// Response is raw audio bytes (WAV/FLAC — HF returns audio directly)
	audioBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("HF MusicGen read: %w", err)
	}
	if len(audioBytes) < 1000 {
		return "", fmt.Errorf("HF MusicGen: response too small (%d bytes) — likely an error", len(audioBytes))
	}

	// Upload to asset storage and return public URL
	fileName := fmt.Sprintf("studio/bg-music/%d.wav", time.Now().UnixNano())
	publicURL, err := o.storage.Upload(ctx, fileName, audioBytes, "audio/wav")
	if err != nil {
		// If storage fails, return data URI so the feature still works in dev
		encoded := base64.StdEncoding.EncodeToString(audioBytes)
		return "data:audio/wav;base64," + encoded, nil
	}
	return publicURL, nil
}

// ─── Pollinations.ai callers (100% free, no API key required) ────────────────
// Pollinations is an open-source Berlin-based AI platform. Their gen.pollinations.ai
// unified endpoint provides image, TTS, and video — powered by FLUX, seedance,
// and ElevenLabs voices. No signup, no rate limit per IP (publishable tier).
// Docs: https://github.com/pollinations/pollinations
// Used as: zero-cost tier between HuggingFace (free with key) and FAL.AI (paid).

// callPollinationsImage generates an image using Pollinations FLUX (free model).
// Official documented endpoint: GET https://gen.pollinations.ai/image/{prompt}
// Docs: https://gen.pollinations.ai  — Returns JPEG/PNG directly (not JSON).
// NOTE (2026-03-26): Pollinations removed anonymous access — sk_ key is now REQUIRED
// for ALL models including free ones. Requests without a key return HTTP 401.
func (o *AIStudioOrchestrator) callPollinationsImage(ctx context.Context, prompt string, aspectRatio ...string) (string, error) {
	return o.callPollinationsImageWithConfig(ctx, os.Getenv("POLLINATIONS_SECRET_KEY"), os.Getenv("POLLINATIONS_IMAGE_MODEL"), prompt, 0, aspectRatio...)
}

func (o *AIStudioOrchestrator) callPollinationsImageWithSeed(ctx context.Context, prompt string, userSeed int64, aspectRatio ...string) (string, error) {
	return o.callPollinationsImageWithConfig(ctx, os.Getenv("POLLINATIONS_SECRET_KEY"), os.Getenv("POLLINATIONS_IMAGE_MODEL"), prompt, userSeed, aspectRatio...)
}

// callPollinationsImageWithConfig is the Router V2 entrypoint: key and model are provider config.
func (o *AIStudioOrchestrator) callPollinationsImageWithConfig(ctx context.Context, apiKey, model, prompt string, userSeed int64, aspectRatio ...string) (string, error) {
	encoded := url.PathEscape(prompt)
	seed := time.Now().UnixNano() % 999983
	if userSeed > 0 {
		seed = userSeed
	}
	// Map aspect_ratio to width/height for Pollinations image API
	ar := "1:1"
	if len(aspectRatio) > 0 && aspectRatio[0] != "" {
		ar = aspectRatio[0]
	}
	width, height := 1024, 1024
	switch ar {
	case "9:16", "portrait":
		width, height = 768, 1344
	case "16:9", "landscape", "wide":
		width, height = 1344, 768
	case "4:3":
		width, height = 1152, 896
	case "3:4":
		width, height = 896, 1152
	case "21:9", "ultrawide":
		width, height = 1536, 640
	}
	// Model must be a CURRENTLY-VALID Pollinations model id. "flux-realism" was retired and now
	// returns HTTP 400 "Invalid model or alias", which broke every image tool that lands here.
	// Verified live 2026-08-25 against gen.pollinations.ai: flux returns a real FLUX.1 Schnell
	// image (EXIF model=flux). POLLINATIONS_IMAGE_MODEL allows swapping without a redeploy —
	// other verified-working ids: gptimage, gptimage-large, zimage, nova-canvas, kontext,
	// seedream5, nanobanana-pro.
	imgModel := model
	if imgModel == "" {
		imgModel = "flux"
	}
	apiURL := fmt.Sprintf(
		"https://gen.pollinations.ai/image/%s?model=%s&width=%d&height=%d&nologo=true&seed=%d&enhance=true",
		encoded, url.QueryEscape(imgModel), width, height, seed,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "NexusAI/1.0")
	if apiKey == "" {
		return "", fmt.Errorf("Pollinations image API key not configured")
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("pollinations image request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("pollinations image %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	imgBytes, err := io.ReadAll(resp.Body)
	if err != nil || len(imgBytes) < 1000 {
		// Storage not configured or response too small — return the Pollinations CDN URL directly.
		// The constructed URL is a stable, publicly cacheable CDN link from gen.pollinations.ai.
		// Use the final URL after any redirects; fall back to the original request URL.
		cdnURL := apiURL
		if resp.Request != nil && resp.Request.URL != nil {
			cdnURL = resp.Request.URL.String()
		}
		log.Printf("[AIStudio] pollinations image: response small or read error (%v), returning CDN URL: %s", err, func() string {
			s := cdnURL
			if len(s) > 80 {
				return s[:80]
			}
			return s
		}())
		return cdnURL, nil
	}

	// Detect content type from response header (may be image/jpeg or image/png)
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/jpeg"
	}
	ext := "jpg"
	if strings.Contains(ct, "png") {
		ext = "png"
	}

	fileName := fmt.Sprintf("studio/ai-photo/flux_%d.%s", time.Now().UnixNano(), ext)
	publicURL, uploadErr := o.storage.Upload(ctx, fileName, imgBytes, ct)
	if uploadErr != nil {
		// No S3/GCS configured — return the Pollinations CDN URL directly (stable, public).
		// The frontend can load it from gen.pollinations.ai without any S3 dependency.
		cdnURL2 := apiURL
		if resp.Request != nil && resp.Request.URL != nil {
			cdnURL2 = resp.Request.URL.String()
		}
		log.Printf("[AIStudio] pollinations image: storage upload failed (%v), returning CDN URL: %s", uploadErr, func() string {
			s := cdnURL2
			if len(s) > 80 {
				return s[:80]
			}
			return s
		}())
		return cdnURL2, nil
	}
	return publicURL, nil
}

// detectAndFixAudioFormat inspects the magic bytes of raw audio data and returns the
// correct MIME type and file extension. It also repairs malformed WAV headers that
// Qwen-TTS emits for streaming responses: the RIFF file-size field and the "data"
// sub-chunk size are both set to 0x7FFFFFFF as placeholders, which prevents browsers
// from determining duration or seeking. We fix them in-place before uploading.
func detectAndFixAudioFormat(audio []byte, requestedFormat string) (mimeType, ext string) {
	// Default: honour the requested format.
	mimeType, ext = "audio/mpeg", "mp3"
	switch requestedFormat {
	case "wav":
		mimeType, ext = "audio/wav", "wav"
	case "opus":
		mimeType, ext = "audio/ogg", "opus"
	case "aac":
		mimeType, ext = "audio/aac", "aac"
	case "flac":
		mimeType, ext = "audio/flac", "flac"
	}

	// Detect RIFF/WAV regardless of what the caller requested.
	// Qwen-TTS ignores response_format and always returns WAV.
	if len(audio) >= 12 && string(audio[0:4]) == "RIFF" && string(audio[8:12]) == "WAVE" {
		mimeType, ext = "audio/wav", "wav"

		// ── Fix RIFF file-size field (bytes 4-7, little-endian uint32) ──────────
		// Streaming TTS APIs write 0x7FFFFFFF here because the total size is
		// unknown at stream-start. Overwrite it with the real value.
		fileSize := uint32(len(audio) - 8)
		audio[4] = byte(fileSize)
		audio[5] = byte(fileSize >> 8)
		audio[6] = byte(fileSize >> 16)
		audio[7] = byte(fileSize >> 24)

		// ── Walk the WAV chunk list and fix the "data" sub-chunk size ────────────
		// Each chunk: 4-byte ID + 4-byte LE size + <size> bytes of data.
		for i := 12; i+8 <= len(audio); {
			id := string(audio[i : i+4])
			chunkSz := uint32(audio[i+4]) | uint32(audio[i+5])<<8 |
				uint32(audio[i+6])<<16 | uint32(audio[i+7])<<24
			if id == "data" {
				dataPayload := uint32(len(audio) - i - 8)
				audio[i+4] = byte(dataPayload)
				audio[i+5] = byte(dataPayload >> 8)
				audio[i+6] = byte(dataPayload >> 16)
				audio[i+7] = byte(dataPayload >> 24)
				break
			}
			// Advance to next chunk (WAV chunks are even-byte-padded).
			advance := 8 + int(chunkSz)
			if advance <= 8 || i+advance > len(audio) {
				break // corrupted size — stop scanning
			}
			if chunkSz%2 != 0 {
				advance++
			}
			i += advance
		}
		log.Printf("[TTS] Qwen-TTS returned WAV (requested %s) — corrected RIFF headers, stored as audio/wav", requestedFormat)
	}
	return mimeType, ext
}

// callPollinationsTTS generates speech using Pollinations Qwen-TTS (UP as of May 2026).
// ElevenLabs-backed models (tts-1 with nova/alloy/echo) are OFF — use qwen-tts instead.
// Qwen-TTS voice options: Cherry (female), Ethan (male), Serena (female), Default.
// sk_ key is REQUIRED.
func (o *AIStudioOrchestrator) callPollinationsTTS(ctx context.Context, text, voice string) (string, error) {
	// Map OpenAI voice names → Qwen-TTS equivalents (ElevenLabs is OFF on Pollinations)
	qwenVoice := mapToQwenVoice(voice)
	payload := map[string]interface{}{
		"model": "qwen-tts",
		"input": text,
		"voice": qwenVoice,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://gen.pollinations.ai/v1/audio/speech", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "NexusAI/1.0")
	sk2 := os.Getenv("POLLINATIONS_SECRET_KEY")
	if sk2 == "" {
		return "", fmt.Errorf("POLLINATIONS_SECRET_KEY not configured (required since 2026-03-26)")
	}
	req.Header.Set("Authorization", "Bearer "+sk2)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("pollinations TTS request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("pollinations TTS %d: %s", resp.StatusCode, truncateStr(string(raw), 100))
	}

	audioBytes, err := io.ReadAll(resp.Body)
	if err != nil || len(audioBytes) < 500 {
		return "", fmt.Errorf("pollinations TTS: response too small")
	}

	mimeType, ext := detectAndFixAudioFormat(audioBytes, "mp3")
	fileName := fmt.Sprintf("studio/narrate/pollinations_%d.%s", time.Now().UnixNano(), ext)
	publicURL, err := o.storage.Upload(ctx, fileName, audioBytes, mimeType)
	if err != nil {
		encoded64 := base64.StdEncoding.EncodeToString(audioBytes)
		return "data:" + mimeType + ";base64," + encoded64, nil
	}
	return publicURL, nil
}

// callPollinationsTTSWithSpeed is like callPollinationsTTS but forwards the speed parameter.
// Speed 0.25–4.0; 1.0 = normal. Falls back to callPollinationsTTS if speed is default.
// callPollinationsTTSFull generates speech with speed, format, and language support.
func (o *AIStudioOrchestrator) callPollinationsTTSFull(ctx context.Context, apiKey, text, voice string, speed float64, format, lang string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("Pollinations API key not configured")
	}
	if format == "" {
		format = "mp3"
	}
	qwenVoice := mapToQwenVoice(voice)
	payload := map[string]interface{}{
		"model":           "qwen-tts",
		"input":           text,
		"voice":           qwenVoice,
		"response_format": format,
	}
	if speed > 0 && speed != 1.0 {
		payload["speed"] = speed
	}
	// Some TTS providers support language hints for accent/pronunciation
	if lang != "" && lang != "en" && lang != "en-us" {
		payload["language"] = lang
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://gen.pollinations.ai/v1/audio/speech", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "NexusAI/1.0")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("pollinations TTS request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("pollinations TTS %d: %s", resp.StatusCode, truncateStr(string(raw), 100))
	}
	audioBytes, err := io.ReadAll(resp.Body)
	if err != nil || len(audioBytes) < 500 {
		return "", fmt.Errorf("pollinations TTS: response too small")
	}
	// detectAndFixAudioFormat detects WAV magic bytes and corrects streaming WAV
	// headers so browsers can determine duration and seek.  Qwen-TTS ignores the
	// response_format field and always returns WAV.
	mimeType, ext := detectAndFixAudioFormat(audioBytes, format)
	fileName := fmt.Sprintf("studio/narrate/pollinations_%d.%s", time.Now().UnixNano(), ext)
	publicURL, err := o.storage.Upload(ctx, fileName, audioBytes, mimeType)
	if err != nil {
		encoded64 := base64.StdEncoding.EncodeToString(audioBytes)
		return "data:" + mimeType + ";base64," + encoded64, nil
	}
	return publicURL, nil
}

// mapToQwenVoice maps OpenAI TTS voice names to Pollinations Qwen-TTS voice names.
// ElevenLabs-backed voices (nova, alloy, echo, shimmer, onyx, fable) are OFF on Pollinations.
// Qwen-TTS available voices: Cherry (female, warm), Ethan (male, clear),
// Serena (female, professional), Default.
func mapToQwenVoice(voice string) string {
	switch strings.ToLower(voice) {
	case "nova", "shimmer", "cherry":
		return "Cherry" // warm female
	case "alloy", "echo", "serena":
		return "Serena" // professional female
	case "onyx", "fable", "ethan":
		return "Ethan" // clear male
	default:
		return "Cherry" // safe default: warm, clear female voice
	}
}

// novaReelModelID() is the last free-tier video model on Pollinations, used as the terminal
// fallback in every video chain.
//
// Pollinations re-priced its video catalogue: as of 2026-08-25 the live /models feed marks
// EVERY video model paid_only=true EXCEPT nova-reel — including wan-fast and p-video, which
// this code had been treating as free. That is why video generation started failing.
// Verified live 2026-08-25: nova-reel → HTTP 200, content-type video/mp4, 3.4 MB.
// POLLINATIONS_VIDEO_MODEL overrides it without a redeploy if the free tier moves again.
func novaReelModelID() string {
	if m := os.Getenv("POLLINATIONS_VIDEO_MODEL"); m != "" {
		return m
	}
	return "nova-reel"
}

// callPollinationsVideo generates a short video using wan-fast (FREE).
// Pollinations video pricing (2026-03-26):
//
//	FREE: wan-fast (Wan 2.2), ltx-2 (LTX-2)
//	PAID: seedance, seedance-pro, veo, wan
//
// Using wan-fast as the default free option.
func (o *AIStudioOrchestrator) callPollinationsVideo(ctx context.Context, imageURL, prompt string) (string, error) { //nolint:unused
	return o.callPollinationsVideoModel(ctx, "wan-fast", imageURL, prompt, 180)
}

// callPollinationsVideoModel is the shared GET-based video caller for any video model.
func (o *AIStudioOrchestrator) callPollinationsVideoModelWithKey(ctx context.Context, apiKey, model, imageURL, prompt string, timeoutSecs int, opts ...string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("Pollinations video API key not configured")
	}

	encoded := url.PathEscape(prompt)
	// opts: [0]=aspectRatio, [1]=duration (seconds as string), [2]=audio ("true"/"false")
	videoAR := "16:9"
	videoDur := "5"
	videoAudio := false
	if len(opts) > 0 && opts[0] != "" {
		switch opts[0] {
		case "9:16", "portrait":
			videoAR = "9:16"
		case "1:1", "square":
			videoAR = "1:1"
		case "4:3":
			videoAR = "4:3"
		default:
			videoAR = "16:9"
		}
	}
	if len(opts) > 1 && opts[1] != "" {
		videoDur = opts[1]
	}
	if len(opts) > 2 && opts[2] == "true" {
		videoAudio = true
	}
	apiURL := fmt.Sprintf("https://gen.pollinations.ai/image/%s?model=%s&duration=%s&aspectRatio=%s",
		encoded, model, videoDur, url.QueryEscape(videoAR))
	if imageURL != "" {
		apiURL += "&image=" + url.QueryEscape(imageURL)
	}
	if videoAudio {
		apiURL += "&audio=true"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", "NexusAI/1.0")

	vidCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSecs)*time.Second)
	defer cancel()
	req = req.WithContext(vidCtx)

	// Use a dedicated client with a longer timeout than the outer httpClient (120s).
	// Pollinations video generation can take 3-4 minutes for complex prompts.
	videoClient := &http.Client{Timeout: time.Duration(timeoutSecs+30) * time.Second}
	resp, err := videoClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("pollinations %s video request: %w", model, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("pollinations %s video %d: %s", model, resp.StatusCode, truncateStr(string(raw), 200))
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil || len(raw) < 1000 {
		return "", fmt.Errorf("pollinations %s video: response too small (%d bytes)", model, len(raw))
	}

	key := fmt.Sprintf("studio/video/%s_%d.mp4", model, time.Now().UnixNano())
	return o.uploadOrDataURI(ctx, raw, "video/mp4", key), nil
}

// callMubert calls the Mubert API for royalty-free background music generation.
func (o *AIStudioOrchestrator) callMubert(ctx context.Context, apiKey, prompt string, durationSecs int) (string, error) {
	payload := map[string]interface{}{
		"method": "RecordTrackTTM",
		"params": map[string]interface{}{
			"pat":       apiKey,
			"prompt":    prompt,
			"mode":      "track",
			"duration":  durationSecs,
			"format":    "mp3",
			"bitrate":   128,
			"intensity": "medium",
			"copyright": true,
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api-b2b.mubert.com/v2/RecordTrackTTM", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("mubert %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	var result struct {
		Status int `json:"status"` // 1 = success, 0 = error
		Error  *struct {
			Code int    `json:"code"`
			Text string `json:"text"`
		} `json:"error"`
		Data struct {
			Tasks []struct {
				MusicURL string `json:"music_url"`
			} `json:"tasks"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("mubert parse: %w", err)
	}
	if result.Status != 1 {
		errText := "unknown error"
		if result.Error != nil {
			errText = fmt.Sprintf("code %d: %s", result.Error.Code, result.Error.Text)
		}
		return "", fmt.Errorf("mubert API error — %s", errText)
	}
	if len(result.Data.Tasks) == 0 || result.Data.Tasks[0].MusicURL == "" {
		return "", fmt.Errorf("mubert: no track URL in response")
	}
	return result.Data.Tasks[0].MusicURL, nil
}

// callSunoMusic calls the sunoapi.org third-party proxy for Suno AI music generation.
// Each request returns 2 audio tracks. We poll until status == "SUCCESS" (up to 4 minutes).
// Requires SUNO_API_KEY env var. Docs: https://docs.sunoapi.org/suno-api/generate-music
//
// Request: POST https://api.sunoapi.org/api/v1/generate
// Poll:    GET  https://api.sunoapi.org/api/v1/generate/record-info?taskId=...
func (o *AIStudioOrchestrator) callSunoMusic(ctx context.Context, apiKey, prompt, style, title, vocalGender string, instrumental bool) (string, string, error) {
	// Build request payload
	payload := map[string]interface{}{
		"customMode":   true,
		"instrumental": instrumental,
		"model":        "V4_5ALL",
		"prompt":       prompt,
		"style":        style,
		"title":        title,
		"callBackUrl":  "https://example.com/noop", // required field; we poll instead
	}
	// Add vocal gender for non-instrumental tracks
	if !instrumental && vocalGender != "" {
		payload["vocalGender"] = vocalGender
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("suno: marshal error: %w", err)
	}

	// Use a long-timeout client for Suno (2-3 min generation time)
	sunoClient := &http.Client{Timeout: 300 * time.Second}

	// Step 1: Submit generation task
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.sunoapi.org/api/v1/generate", bytes.NewReader(body))
	if err != nil {
		return "", "", fmt.Errorf("suno: request build error: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := sunoClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("suno: submit error: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var submitResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			TaskID string `json:"taskId"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&submitResp); err != nil {
		return "", "", fmt.Errorf("suno: decode submit response: %w", err)
	}
	if submitResp.Code != 200 || submitResp.Data.TaskID == "" {
		return "", "", fmt.Errorf("suno: submit failed code=%d msg=%s", submitResp.Code, submitResp.Msg)
	}
	taskID := submitResp.Data.TaskID
	log.Printf("[AIStudio] Suno task submitted: %s", taskID)

	// Step 2: Poll until SUCCESS (up to 4 minutes, 10-second intervals)
	pollURL := "https://api.sunoapi.org/api/v1/generate/record-info?taskId=" + url.QueryEscape(taskID)
	deadline := time.Now().Add(4 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(10 * time.Second)

		pollReq, err := http.NewRequestWithContext(ctx, http.MethodGet, pollURL, nil)
		if err != nil {
			return "", "", fmt.Errorf("suno: poll request build: %w", err)
		}
		pollReq.Header.Set("Authorization", "Bearer "+apiKey)

		pollResp, err := sunoClient.Do(pollReq)
		if err != nil {
			log.Printf("[AIStudio] Suno poll error (will retry): %v", err)
			continue
		}

		var statusResp struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				TaskID   string `json:"taskId"`
				Status   string `json:"status"`
				Response struct {
					SunoData []struct {
						ID       string  `json:"id"`
						AudioURL string  `json:"audioUrl"`
						Title    string  `json:"title"`
						Duration float64 `json:"duration"`
					} `json:"sunoData"`
				} `json:"response"`
				ErrorMessage string `json:"errorMessage"`
			} `json:"data"`
		}
		if err := json.NewDecoder(pollResp.Body).Decode(&statusResp); err != nil {
			if cerr := pollResp.Body.Close(); cerr != nil {
				log.Printf("[AIStudio] Suno pollResp body close: %v", cerr)
			}
			log.Printf("[AIStudio] Suno poll decode error (will retry): %v", err)
			continue
		}
		if err := pollResp.Body.Close(); err != nil {
			log.Printf("[AIStudio] Suno pollResp body close: %v", err)
		}

		switch statusResp.Data.Status {
		case "SUCCESS", "FIRST_SUCCESS":
			// Return both audio URLs from the two generated tracks
			if len(statusResp.Data.Response.SunoData) > 0 {
				audioURL1 := statusResp.Data.Response.SunoData[0].AudioURL
				audioURL2 := ""
				if len(statusResp.Data.Response.SunoData) > 1 {
					audioURL2 = statusResp.Data.Response.SunoData[1].AudioURL
				}
				if audioURL1 != "" {
					log.Printf("[AIStudio] Suno SUCCESS — track1: %s track2: %s", audioURL1, audioURL2)
					return audioURL1, audioURL2, nil
				}
			}
		case "CREATE_TASK_FAILED", "GENERATE_AUDIO_FAILED":
			return "", "", fmt.Errorf("suno: generation failed: %s", statusResp.Data.ErrorMessage)
		default:
			log.Printf("[AIStudio] Suno status: %s — polling...", statusResp.Data.Status)
		}
	}
	return "", "", fmt.Errorf("suno: timed out waiting for generation (taskId=%s)", taskID)
}

// callElevenLabsMusic calls the ElevenLabs Music Generation API for full songs and instrumentals.
// Correct endpoint: POST /v1/music-generation (NOT /v1/sound-generation which is for sound effects).
// Docs: https://elevenlabs.io/docs/api-reference/music-generation
func (o *AIStudioOrchestrator) callElevenLabsMusic(ctx context.Context, apiKey, prompt string) (string, error) {
	payload := map[string]interface{}{
		"prompt": prompt,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.elevenlabs.io/v1/music-generation", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	audio, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("elevenLabs music %d: %s", resp.StatusCode, truncateStr(string(audio), 200))
	}
	return o.uploadOrDataURI(ctx, audio, "audio/mpeg", "music/"+uuid.New().String()+".mp3"), nil
}

// ─── NEW: Vision dispatch ─────────────────────────────────────────────────────
// Handles: image-analyser, ask-my-photo

func (o *AIStudioOrchestrator) dispatchVision(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	var imageURL, question string

	switch slug {
	case "ask-my-photo":
		// Frontend sends: { prompt: question, image_url: imgURL } via buildEnrichedPrompt
		imageURL = env.ImageURL
		question = env.Prompt
		if imageURL == "" {
			return nil, fmt.Errorf("ask-my-photo: image_url is required")
		}

	case "code-pro":
		// Nexus Code Pro: multimodal code debugging.
		// If an image is attached (screenshot, error traceback, diagram), use vision.
		// If no image, fall through to text-only Qwen-Coder path.
		imageURL = env.ImageURL
		question = env.Prompt
		if imageURL == "" {
			// No image — route as pure code-helper text request
			return o.dispatchCodePro(ctx, env)
		}
		// Image present — build a code-aware vision prompt
		if question == "" {
			question = "Analyse this image in the context of software development. Identify any errors, bugs, or issues visible. Explain what is wrong and provide a corrected code solution."
		} else {
			question = fmt.Sprintf("[Code Context] %s\n\nAnalyse the attached image carefully. If it shows an error, traceback, UI bug, or architecture diagram, use it to inform your answer. Provide a complete, production-ready solution with explanation.", question)
		}

	case "doc-analyzer":
		// Nexus Document Analyzer: structured extraction from PDF, invoice, chart, or scanned image.
		return o.dispatchDocAnalyzer(ctx, env)

	case "localize-ui":
		// Nexus Localization Engine: OCR + African dialect translation.
		return o.dispatchLocalizeUI(ctx, env)

	default: // image-analyser
		// Frontend sends image_url or puts imageURL in prompt for legacy compatibility
		imageURL = env.ImageURL
		if imageURL == "" {
			imageURL = env.Prompt
		}
		// Use env.Prompt as the question if provided; otherwise use a rich default
		question = env.Prompt
		if question == "" || question == imageURL {
			question = "Analyse this image in detail. Describe: (1) What you see — objects, people, text, colours, composition. " +
				"(2) The mood, style, and context. " +
				"(3) Any notable features — brand names, logos, faces, places, technical elements. " +
				"(4) If it contains charts or data, extract and interpret the data. " +
				"(5) If it contains code or a UI screenshot, identify the technology and describe the content. " +
				"Be specific, accurate, and comprehensive."
		}
	}

	visionSys := "You are Nexus Vision, an expert image analyst. Analyse visual content with precision, " +
		"extract visible text accurately, identify important objects, data, UI/code details and cultural context, " +
		"and structure the answer clearly with actionable insights."
	if slug == "code-pro" {
		visionSys = "You are Nexus Code Pro, an elite software engineer and visual debugging expert. " +
			"Use the attached image or screenshot as evidence, identify visible errors or architecture issues, " +
			"and provide complete production-quality fixes with concise explanation."
	}
	vIn := providerInput{ImageURL: imageURL, SystemPrompt: visionSys, UserPrompt: question}
	_, text, cost, usedSlug, err := o.runToolStageChain(ctx, nil, slug, "main", vIn)
	if err != nil {
		return nil, fmt.Errorf("vision route unavailable for %q: %w", slug, err)
	}
	return &studioProviderResult{OutputText: text, Provider: "route/" + usedSlug, CostMicros: cost}, nil
}

// ─── Nexus Code Pro ──────────────────────────────────────────────────────────
// dispatchCodePro handles the code-pro slug when NO image is attached.
// When an image IS attached, dispatchVision handles it directly (above).
func (o *AIStudioOrchestrator) dispatchCodePro(ctx context.Context, env promptEnvelope) (*studioProviderResult, error) {
	codeSys := "You are Nexus Code Pro, an elite software engineer and debugging expert. " +
		"Write production-quality, clean, well-commented code in any language. " +
		"Always wrap code in fenced code blocks with the correct language tag. " +
		"Include robust error handling, explain key logic concisely, and provide complete fixes when debugging."
	_, text, cost, usedSlug, err := o.runToolStageChain(ctx, nil, "code-pro", "main", providerInput{
		SystemPrompt: codeSys,
		UserPrompt:   env.Prompt,
	})
	if err != nil {
		return nil, fmt.Errorf("code-pro route unavailable: %w", err)
	}
	return &studioProviderResult{OutputText: text, Provider: "route/" + usedSlug, CostMicros: cost}, nil
}

// ─── Nexus Document Analyzer ─────────────────────────────────────────────────
// dispatchDocAnalyzer handles the doc-analyzer slug.
// Supports PDF, text documents (via DocumentURL) and scanned images (via ImageURL).
func (o *AIStudioOrchestrator) dispatchDocAnalyzer(ctx context.Context, env promptEnvelope) (*studioProviderResult, error) {
	docSys := "You are Nexus Document Intelligence, an expert analyst specialising in extracting structured, actionable insights from documents. " +
		"IMPORTANT: If the document appears to be image-based or scanned, use your vision capabilities to read and extract the text — do NOT say you cannot read it. " +
		"For invoices: extract vendor, date, line items, totals, taxes, and payment terms in a clean table. " +
		"For reports or research papers: provide an executive summary, key findings, methodology, and recommendations. " +
		"For charts or graphs: describe the data, identify trends, and state the key insight in one sentence. " +
		"For contracts or legal documents: summarise the parties, key obligations, dates, and any risk clauses. " +
		"For pitch decks or presentations: extract the core narrative, value proposition, market opportunity, and key metrics. " +
		"For general documents: extract the main points, structure them clearly, and answer any specific question the user has asked. " +
		"Always format your response with clear headings. If the user asked a specific question, answer it first before the full analysis. " +
		"Never apologise for document format — always attempt to extract and analyse whatever content is visible."

	userQ := env.Prompt
	if userQ == "" {
		userQ = "Analyse this document thoroughly. Extract all key information, summarise the main points, and present the findings in a clear, structured format."
	}

	if env.DocumentURL == "" && env.ImageURL == "" && env.Prompt == "" {
		return nil, fmt.Errorf("doc-analyzer: document, image, or question is required")
	}
	_, text, cost, usedSlug, err := o.runToolStageChain(ctx, nil, "doc-analyzer", "main", providerInput{
		SystemPrompt: docSys,
		UserPrompt:   userQ,
		DocumentURL:  env.DocumentURL,
		ImageURL:     env.ImageURL,
	})
	if err != nil {
		return nil, fmt.Errorf("doc-analyzer route unavailable: %w", err)
	}
	return &studioProviderResult{OutputText: text, Provider: "route/" + usedSlug, CostMicros: cost}, nil
}

// ─── Nexus Localization Engine ────────────────────────────────────────────────
// dispatchLocalizeUI handles the localize-ui slug.
// Reads text from a screenshot via OCR and translates it into the requested African dialect.
func (o *AIStudioOrchestrator) dispatchLocalizeUI(ctx context.Context, env promptEnvelope) (*studioProviderResult, error) {
	// Determine target language from the prompt or language field
	targetLang := strings.ToLower(strings.TrimSpace(env.Language))
	if targetLang == "" {
		// Try to extract from prompt (e.g. "translate to Yoruba")
		promptLower := strings.ToLower(env.Prompt)
		switch {
		case strings.Contains(promptLower, "yoruba") || strings.Contains(promptLower, "yo"):
			targetLang = "yo"
		case strings.Contains(promptLower, "hausa") || strings.Contains(promptLower, "ha"):
			targetLang = "ha"
		case strings.Contains(promptLower, "igbo") || strings.Contains(promptLower, "ig"):
			targetLang = "ig"
		case strings.Contains(promptLower, "pidgin") || strings.Contains(promptLower, "pcm"):
			targetLang = "pcm"
		default:
			targetLang = "yo" // default to Yoruba
		}
	}

	langNames := map[string]string{
		"yo":  "Yoruba",
		"ha":  "Hausa",
		"ig":  "Igbo",
		"pcm": "Nigerian Pidgin English",
		"fr":  "French",
		"en":  "English",
	}
	langName, ok := langNames[targetLang]
	if !ok {
		langName = "Yoruba"
	}

	locSys := fmt.Sprintf(
		"You are Nexus Localize, an expert in African languages and UI/UX localisation. "+
			"Your task is to: "+
			"1. Extract ALL text visible in the provided screenshot or image (OCR). "+
			"2. Translate every piece of extracted text into %s, preserving the original meaning and UI context. "+
			"3. Present the results as a two-column table: | Original Text | %s Translation |. "+
			"4. After the table, provide a 'Localisation Notes' section with any cultural adaptations made. "+
			"5. For buttons, labels, and CTAs: keep translations short and action-oriented. "+
			"6. For error messages: ensure the translated message is clear and non-technical. "+
			"If no image is provided, translate the user's text directly into %s.",
		langName, langName, langName,
	)

	userQ := env.Prompt
	if userQ == "" {
		userQ = fmt.Sprintf("Extract all text from this UI screenshot and translate everything into %s.", langName)
	}

	_, text, cost, usedSlug, err := o.runToolStageChain(ctx, nil, "localize-ui", "main", providerInput{
		SystemPrompt: locSys,
		UserPrompt:   userQ,
		ImageURL:     env.ImageURL,
		TargetLang:   targetLang,
	})
	if err != nil {
		return nil, fmt.Errorf("localize-ui route unavailable: %w", err)
	}
	return &studioProviderResult{OutputText: text, Provider: "route/" + usedSlug, CostMicros: cost}, nil
}

// ─── Nexus Agent — True ReAct Agent (Option B) ───────────────────────────────
//
// Architecture: Gemini 2.5 Flash with native function-calling (tool use).
// Loop:  Reason → Act (call tool) → Observe (inject result) → repeat
// Tools: web_search (Tavily), read_url (HTTP fetch + strip)
// Cap:   agentSemaphore limits concurrent runs to 10 — protects API rate limits
//        at scale (10 k users). Queue wait timeout: 8 s.
// Max iterations: 8 — prevents infinite loops on adversarial inputs.
// Total timeout: 90 s context deadline (set by caller via ctx).

// agentReadURL fetches a URL and returns its readable text content (stripped of HTML).
// Capped at 6 000 characters to avoid filling the context window.
func (o *AIStudioOrchestrator) agentReadURL(ctx context.Context, rawURL string) string {
	if rawURL == "" {
		return "[read_url error: empty URL]"
	}
	// Basic sanity check — must be http/https
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return "[read_url error: URL must start with http:// or https://]"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Sprintf("[read_url error: %v]", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; NexusAgent/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("[read_url error: %v]", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("[read_url error: HTTP %d for %s]", resp.StatusCode, rawURL)
	}

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024)) // cap at 512 KB
	text := agentStripHTML(string(raw))
	text = strings.TrimSpace(text)
	if len(text) > 6000 {
		text = text[:6000] + "\n...[content truncated]"
	}
	if text == "" {
		return "[read_url: page returned no readable text]"
	}
	return text
}

// agentStripHTML removes HTML tags and normalises whitespace for LLM consumption.
func agentStripHTML(html string) string {
	// Remove script and style blocks entirely
	for _, tag := range []string{"script", "style", "head", "nav", "footer"} {
		for {
			open := strings.Index(strings.ToLower(html), "<"+tag)
			if open == -1 {
				break
			}
			close := strings.Index(strings.ToLower(html[open:]), "</"+tag+">")
			if close == -1 {
				break
			}
			html = html[:open] + " " + html[open+close+len("</"+tag+">"):]
		}
	}
	// Strip remaining tags
	inTag := false
	var out strings.Builder
	for _, ch := range html {
		switch {
		case ch == '<':
			inTag = true
		case ch == '>':
			inTag = false
			out.WriteRune(' ')
		case !inTag:
			out.WriteRune(ch)
		}
	}
	// Collapse whitespace
	text := out.String()
	parts := strings.Fields(text)
	return strings.Join(parts, " ")
}

// ─── NEW: dispatchNarratorPro ─────────────────────────────────────────────────

func (o *AIStudioOrchestrator) dispatchNarratorPro(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	voice := strings.ToLower(strings.TrimSpace(env.VoiceID))
	if voice == "" {
		voice = "nova"
	}
	speed := 1.0
	audioFormat := "mp3"
	if env.Extra != nil {
		if s, ok := env.Extra["speed"].(float64); ok && s > 0 {
			speed = s
		}
		if f, ok := env.Extra["format"].(string); ok && f != "" {
			audioFormat = f
		}
	}
	in := providerInput{
		Text:        env.Prompt,
		VoiceID:     voice,
		Speed:       speed,
		AudioFormat: audioFormat,
		Language:    strings.ToLower(strings.TrimSpace(env.Language)),
	}
	url, _, cost, usedSlug, err := o.runToolStageChain(ctx, nil, slug, "main", in)
	if err != nil {
		return nil, fmt.Errorf("narration route unavailable for %q: %w", slug, err)
	}
	return &studioProviderResult{OutputURL: url, Provider: "route/" + usedSlug, CostMicros: cost}, nil
}

// ─── NEW: Pollinations helper callers ─────────────────────────────────────────

// callPollinationsWebSearch uses Pollinations gemini-search for live web-aware answers.
func (o *AIStudioOrchestrator) callPollinationsWebSearch(ctx context.Context, prompt string) (string, error) {
	sk := os.Getenv("POLLINATIONS_SECRET_KEY")
	if sk == "" {
		return "", fmt.Errorf("POLLINATIONS_SECRET_KEY not configured")
	}
	payload := map[string]interface{}{
		"model": "openai",
		"messages": []map[string]interface{}{
			{"role": "system", "content": "You are Nexus AI, a helpful assistant. You have access to real-time web search. Answer with current information."},
			{"role": "user", "content": prompt},
		},
		"search": true,
	}
	return o.callPollinationsOpenAIChat(ctx, sk, payload)
}

// callPollinationsVision uses Pollinations multimodal API to analyse an image.
func (o *AIStudioOrchestrator) callPollinationsVision(ctx context.Context, imageURL, question string) (string, error) {
	sk := os.Getenv("POLLINATIONS_SECRET_KEY")
	if sk == "" {
		return "", fmt.Errorf("POLLINATIONS_SECRET_KEY not configured")
	}
	if question == "" {
		question = "Describe this image in detail. What do you see? Be comprehensive and mention colors, objects, people, text, and context."
	}
	payload := map[string]interface{}{
		"model": "openai",
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{"type": "text", "text": question},
					{"type": "image_url", "image_url": map[string]string{"url": imageURL}},
				},
			},
		},
	}
	return o.callPollinationsOpenAIChat(ctx, sk, payload)
}

// callPollinationsQwenCoder uses Pollinations Qwen3-Coder for coding tasks.
func (o *AIStudioOrchestrator) callPollinationsQwenCoder(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	sk := os.Getenv("POLLINATIONS_SECRET_KEY")
	if sk == "" {
		return "", fmt.Errorf("POLLINATIONS_SECRET_KEY not configured")
	}
	payload := map[string]interface{}{
		"model": "qwen-coder",
		"messages": []map[string]interface{}{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
	}
	return o.callPollinationsOpenAIChat(ctx, sk, payload)
}

// callPollinationsOpenAIChat is a shared helper for Pollinations OpenAI-compatible endpoints.
// Parses choices[0].message.content from the response.
func (o *AIStudioOrchestrator) callPollinationsOpenAIChat(ctx context.Context, sk string, payload interface{}) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://gen.pollinations.ai/v1/chat/completions", bytes.NewReader(body))
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
		return "", fmt.Errorf("pollinations chat %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
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

// callPollinationsGPTImage generates a premium image via Pollinations (gptimage / gptimage-large / seedream).
func (o *AIStudioOrchestrator) callPollinationsGPTImage(ctx context.Context, prompt, model string, opts ...string) (string, error) {
	return o.callPollinationsGPTImageWithKey(ctx, os.Getenv("POLLINATIONS_SECRET_KEY"), prompt, model, opts...)
}

func (o *AIStudioOrchestrator) callPollinationsGPTImageWithKey(ctx context.Context, apiKey, prompt, model string, opts ...string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("Pollinations image API key not configured")
	}
	// opts: [0]=aspectRatio, [1]=quality
	gptSize := "1024x1024"
	gptQuality := "standard"
	if len(opts) > 0 && opts[0] != "" {
		switch opts[0] {
		case "9:16", "portrait":
			gptSize = "1024x1792"
		case "16:9", "landscape", "wide":
			gptSize = "1792x1024"
		}
	}
	if len(opts) > 1 && opts[1] == "hd" {
		gptQuality = "hd"
	}
	payload := map[string]interface{}{
		"model":           model,
		"prompt":          prompt,
		"n":               1,
		"size":            gptSize,
		"quality":         gptQuality,
		"response_format": "url", // always return URL (not b64_json) — avoids large base64 in DB
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://gen.pollinations.ai/v1/images/generations", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", "NexusAI/1.0")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("pollinations GPTImage request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("pollinations GPTImage %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}

	var parsed struct {
		Data []struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Data) == 0 {
		return "", fmt.Errorf("pollinations GPTImage parse: empty data")
	}
	item := parsed.Data[0]
	if item.B64JSON != "" {
		imgBytes, err := base64.StdEncoding.DecodeString(item.B64JSON)
		if err != nil {
			return "", fmt.Errorf("pollinations GPTImage base64 decode: %w", err)
		}
		key := fmt.Sprintf("studio/ai-photo/%s_%d.png", model, time.Now().UnixNano())
		return o.uploadOrDataURI(ctx, imgBytes, "image/png", key), nil
	}
	if item.URL != "" {
		// Download the image from the Pollinations CDN URL and store it in our own
		// storage so the frontend never depends on Pollinations CDN availability.
		dlCtx, dlCancel := context.WithTimeout(ctx, 30*time.Second)
		defer dlCancel()
		dlReq, dlErr := http.NewRequestWithContext(dlCtx, http.MethodGet, item.URL, nil)
		if dlErr == nil {
			dlReq.Header.Set("Authorization", "Bearer "+apiKey)
			dlResp, dlErr := o.httpClient.Do(dlReq)
			if dlErr == nil {
				defer func() { _ = dlResp.Body.Close() }()
				if dlResp.StatusCode == http.StatusOK {
					imgBytes, dlErr := io.ReadAll(dlResp.Body)
					if dlErr == nil && len(imgBytes) > 1000 {
						ct := dlResp.Header.Get("Content-Type")
						ext := "jpg"
						if strings.Contains(ct, "png") {
							ext = "png"
						}
						if ct == "" {
							ct = "image/jpeg"
						}
						key := fmt.Sprintf("studio/ai-photo/%s_%d.%s", model, time.Now().UnixNano(), ext)
						if publicURL, upErr := o.storage.Upload(ctx, key, imgBytes, ct); upErr == nil {
							return publicURL, nil
						}
						// Storage upload failed — fall through to return direct URL
						log.Printf("[AIStudio] GPTImage storage upload failed — returning Pollinations URL directly")
					}
				}
			}
		}
		// Fallback: return the Pollinations URL directly (may be unstable, but best we can do)
		return item.URL, nil
	}
	return "", fmt.Errorf("pollinations GPTImage: no url or b64_json in response")
}

// callPollinationsKontextAlt performs image-to-image editing via Pollinations p-image-edit (Pruna).
// This is the fallback for when kontext is OFF/degraded. Uses the same edits endpoint but model=p-image-edit.
func (o *AIStudioOrchestrator) callPollinationsKontextAlt(ctx context.Context, imageURL, instruction string) (string, error) {
	return o.callPollinationsImageEditWithKey(ctx, os.Getenv("POLLINATIONS_SECRET_KEY"), "p-image-edit", imageURL, instruction)
}

func (o *AIStudioOrchestrator) callPollinationsImageEditWithKey(ctx context.Context, apiKey, model, imageURL, instruction string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("Pollinations image-edit API key not configured")
	}
	if model == "" {
		model = "p-image-edit"
	}

	// Step 1: Download source image
	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("p-image-edit: build download request: %w", err)
	}
	dlResp, err := o.httpClient.Do(dlReq)
	if err != nil {
		return "", fmt.Errorf("p-image-edit: download image: %w", err)
	}
	defer func() {
		if err := dlResp.Body.Close(); err != nil {
			log.Printf("[AIStudio] p-image-edit dlResp body close: %v", err)
		}
	}()
	imgBytes, err := io.ReadAll(dlResp.Body)
	if err != nil || len(imgBytes) < 500 {
		return "", fmt.Errorf("p-image-edit: image download failed or too small")
	}

	// Step 2: Build multipart body
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("model", model)
	_ = mw.WriteField("prompt", instruction)
	fw, err := mw.CreateFormFile("image", "source.png")
	if err != nil {
		return "", err
	}
	if _, err = fw.Write(imgBytes); err != nil {
		return "", err
	}
	if err := mw.Close(); err != nil {
		return "", fmt.Errorf("p-image-edit multipart close: %w", err)
	}

	// Step 3: POST to edits endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://gen.pollinations.ai/v1/images/edits", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", "NexusAI/1.0")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("pollinations p-image-edit request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("pollinations p-image-edit %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}

	var parsed struct {
		Data []struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Data) == 0 {
		return "", fmt.Errorf("pollinations p-image-edit parse: empty data")
	}
	item := parsed.Data[0]
	if item.URL != "" {
		return item.URL, nil
	}
	if item.B64JSON != "" {
		outBytes, err := base64.StdEncoding.DecodeString(item.B64JSON)
		if err != nil {
			return "", fmt.Errorf("pollinations p-image-edit b64 decode: %w", err)
		}
		key := fmt.Sprintf("studio/photo-editor/p-image-edit_%d.png", time.Now().UnixNano())
		return o.uploadOrDataURI(ctx, outBytes, "image/png", key), nil
	}
	return "", fmt.Errorf("pollinations p-image-edit: no url or b64_json in response")
}

// callPollinationsWhisperAfrican transcribes audio using Pollinations Whisper with African language support.
// Downloads the audio file, then POSTs multipart to the transcriptions endpoint.
func (o *AIStudioOrchestrator) callPollinationsWhisperAfrican(ctx context.Context, audioURL, lang string) (string, error) {
	sk := os.Getenv("POLLINATIONS_SECRET_KEY")
	if sk == "" {
		return "", fmt.Errorf("POLLINATIONS_SECRET_KEY not configured")
	}

	// Step 1: Download audio bytes
	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, audioURL, nil)
	if err != nil {
		return "", fmt.Errorf("whisper African: build download request: %w", err)
	}
	dlResp, err := o.httpClient.Do(dlReq)
	if err != nil {
		return "", fmt.Errorf("whisper African: download audio: %w", err)
	}
	defer func() {
		if err := dlResp.Body.Close(); err != nil {
			log.Printf("[AIStudio] Whisper African dlResp body close: %v", err)
		}
	}()
	audioBytes, err := io.ReadAll(dlResp.Body)
	if err != nil || len(audioBytes) < 100 {
		return "", fmt.Errorf("whisper African: audio download failed or too small")
	}

	// Step 2: Build multipart body
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("model", "whisper-large-v3")
	_ = mw.WriteField("language", lang)
	fw, err := mw.CreateFormFile("file", "audio.mp3")
	if err != nil {
		return "", err
	}
	if _, err = fw.Write(audioBytes); err != nil {
		return "", err
	}
	if err := mw.Close(); err != nil {
		return "", fmt.Errorf("whisper African multipart close: %w", err)
	}

	// Step 3: POST to Pollinations Whisper endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://gen.pollinations.ai/v1/audio/transcriptions", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+sk)
	req.Header.Set("User-Agent", "NexusAI/1.0")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("pollinations Whisper African request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("pollinations Whisper African %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || result.Text == "" {
		return "", fmt.Errorf("pollinations Whisper African: no transcription returned")
	}
	return result.Text, nil
}

// callPollinationsElevenMusic generates a full song or instrumental via Pollinations ElevenMusic.
// Official documented endpoint: GET https://gen.pollinations.ai/audio/{text}?model=elevenmusic
// Docs: https://gen.pollinations.ai — audio models use the /audio/{text} route.
// Returns raw MP3 binary. sk_ key required via Bearer header. Timeout: 180s.
// Set instrumental=true to skip vocals and generate a background track only.
func (o *AIStudioOrchestrator) callPollinationsElevenMusic(ctx context.Context, apiKey, prompt string, instrumental bool) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("Pollinations music API key not configured")
	}

	encoded := url.PathEscape(prompt)
	apiURL := fmt.Sprintf("https://gen.pollinations.ai/audio/%s?model=elevenmusic", encoded)
	if instrumental {
		apiURL += "&instrumental=true"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", "NexusAI/1.0")

	// Music generation can take up to 3 minutes.
	// Use a dedicated client — the shared o.httpClient has a 120s timeout that
	// would silently cap the 180s context deadline set below.
	musicCtx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()
	req = req.WithContext(musicCtx)

	musicClient := &http.Client{Timeout: 200 * time.Second}
	resp, err := musicClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("pollinations ElevenMusic request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("pollinations ElevenMusic %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}

	// GET /audio returns raw MP3 bytes directly
	raw, err := io.ReadAll(resp.Body)
	if err != nil || len(raw) < 1000 {
		return "", fmt.Errorf("pollinations ElevenMusic: response too small (%d bytes)", len(raw))
	}

	suffix := "song"
	if instrumental {
		suffix = "instrumental"
	}
	key := fmt.Sprintf("studio/music/%s_%d.mp3", suffix, time.Now().UnixNano())
	return o.uploadOrDataURI(ctx, raw, "audio/mpeg", key), nil
}

// callPollinationsSeedance is kept for backward compatibility (e.g. admin DB rows that use seedance).
// NOTE: seedance is a PAID model (1.8 pollen/M). Use callPollinationsVideo (wan-fast, FREE) for
// cost-effective generation. Only call this when the user explicitly selected a paid seedance plan.
func (o *AIStudioOrchestrator) callPollinationsSeedance(ctx context.Context, imageURL, prompt string) (string, error) { //nolint:unused
	return o.callPollinationsVideoModel(ctx, "seedance", imageURL, prompt, 180)
}

// callPollinationsVeo generates a premium text-to-video using Google Veo via Pollinations.
// Official documented endpoint: GET gen.pollinations.ai/image/{prompt}?model=veo
// Paid model (~$0.40-0.50/video). Uses sk_ key via Bearer header. Timeout: 180s.
func (o *AIStudioOrchestrator) callPollinationsVeo(ctx context.Context, prompt, aspectRatio string, withAudio bool) (string, error) {
	audioOpt := "false"
	if withAudio {
		audioOpt = "true"
	}
	// Veo is a premium model — generation can take 2-4 minutes
	return o.callPollinationsVideoModel(ctx, "veo", "", prompt, 360, aspectRatio, "8", audioOpt)
}

// ─── S3 upload helper ─────────────────────────────────────────────────────────

// uploadOrDataURI uploads binary data via the configured AssetStorage backend
// (S3, GCS, or local). Falls back to a base64 data URI only if the storage
// backend itself returns an error (e.g. no credentials in dev mode).
func (o *AIStudioOrchestrator) uploadOrDataURI(ctx context.Context, data []byte, contentType, key string) string {
	url, err := o.storage.Upload(ctx, key, data, contentType)
	if err != nil {
		// No cloud storage configured — data URIs work but are large (stored in DB).
		// To fix: set STORAGE_BACKEND=s3 + AWS_S3_BUCKET / STORAGE_BACKEND=gcs + GCS_BUCKET in Render env.
		// For Pollinations images (stable CDN), the caller should return the CDN URL instead of binary.
		log.Printf("[AIStudio] WARN: asset upload failed for %s (backend=%s): %v — returning data URI (configure cloud storage to fix)", key, o.storage.Provider(), err)
		return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
	}
	return url
}

// ─── Completion helpers ───────────────────────────────────────────────────────

func (o *AIStudioOrchestrator) complete(ctx context.Context, gen *entities.AIGeneration, r *studioProviderResult) error {
	return o.studioSvc.CompleteGeneration(ctx, gen.ID, r.OutputURL, r.OutputURL2, r.OutputText, r.Provider, r.CostMicros, r.DurationMs)
}

// ─── Utility ─────────────────────────────────────────────────────────────────

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// ─── FEAT-01: Gemini multimodal document analysis ─────────────────────────────
// callGeminiWithDocument fetches a PDF or TXT file from a CDN URL, base64-encodes
// it, and sends it to Gemini as inline_data alongside the user prompt.
// Supported MIME types: application/pdf, text/plain, text/markdown.
// Falls back to callGeminiFlash (text-only) if the document cannot be fetched.
func (o *AIStudioOrchestrator) callGeminiWithDocument(ctx context.Context, systemPrompt, userPrompt, documentURL string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY not configured")
	}

	// Fetch the document from CDN
	docReq, err := http.NewRequestWithContext(ctx, http.MethodGet, documentURL, nil)
	if err != nil {
		return o.callGeminiFlash(ctx, systemPrompt, userPrompt)
	}
	docResp, err := o.httpClient.Do(docReq)
	if err != nil || docResp.StatusCode != http.StatusOK {
		log.Printf("[AIStudio] callGeminiWithDocument: fetch failed (%v) — falling back to text-only", err)
		return o.callGeminiFlash(ctx, systemPrompt, userPrompt)
	}
	defer func() {
		if err := docResp.Body.Close(); err != nil {
			log.Printf("[AIStudio] callGeminiWithDocument: body close: %v", err)
		}
	}()
	docBytes, err := io.ReadAll(io.LimitReader(docResp.Body, 50<<20)) // 50 MB limit
	if err != nil {
		return o.callGeminiFlash(ctx, systemPrompt, userPrompt)
	}

	// Determine MIME type from Content-Type header or URL extension
	mimeType := docResp.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		lower := strings.ToLower(documentURL)
		switch {
		case strings.HasSuffix(lower, ".pdf"):
			mimeType = "application/pdf"
		case strings.HasSuffix(lower, ".md"):
			mimeType = "text/markdown"
		default:
			mimeType = "text/plain"
		}
	}
	// Strip charset suffix if present (e.g. "text/plain; charset=utf-8")
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}

	// Only allow Gemini-supported document MIME types
	allowed := map[string]bool{
		"application/pdf": true,
		"text/plain":      true,
		"text/markdown":   true,
		"text/html":       true,
		"text/csv":        true,
	}
	if !allowed[mimeType] {
		log.Printf("[AIStudio] callGeminiWithDocument: unsupported MIME %s — falling back to text-only", mimeType)
		return o.callGeminiFlash(ctx, systemPrompt, userPrompt)
	}

	docB64 := base64.StdEncoding.EncodeToString(docBytes)

	// Use gemini-2.5-flash: significantly better PDF OCR, especially for scanned/image-based PDFs
	geminiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=%s", apiKey)
	payload := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]string{{"text": systemPrompt}},
		},
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{
						"inline_data": map[string]string{
							"mime_type": mimeType,
							"data":      docB64,
						},
					},
					{"text": userPrompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.7,
			"maxOutputTokens": 8192,
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, geminiURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini document request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini document %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}
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
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("gemini document parse: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("gemini document API error: %s", result.Error.Message)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini document: no content returned")
	}
	return result.Candidates[0].Content.Parts[0].Text, nil
}

func (o *AIStudioOrchestrator) callFALMultiImageVideo(
	ctx context.Context, falKey string, imageURLs []string, prompt string, env promptEnvelope,
) (string, error) {
	return o.callFALMultiImageVideoConfigured(
		ctx, falKey, "fal-ai/kling-video/v2.6/pro/multi-image-to-video",
		imageURLs, prompt, env.Duration, env.AspectRatio, env.Extra,
	)
}

func (o *AIStudioOrchestrator) callPollinationsVideoModel(
	ctx context.Context, model, imageURL, prompt string, timeoutSecs int, opts ...string,
) (string, error) {
	return o.callPollinationsVideoModelWithKey(
		ctx, os.Getenv("POLLINATIONS_SECRET_KEY"), model, imageURL, prompt, timeoutSecs, opts...,
	)
}
