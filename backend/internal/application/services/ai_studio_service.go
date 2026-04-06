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
)

// slugCategory maps every tool slug to its dispatch category.
var slugCategory = map[string]studioToolCat{
	"translate":      catVoice,   // routed through voice pipeline (TTS/translate)
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
	"my-marketing-jingle":  catMusic,  // alias for jingle
	"text-to-speech":       catVoice,  // alias for narrate
	"local-translation":    catVoice,  // alias for translate
	"deep-research-brief":  catText,   // alias for research-brief
	"mind-map":             catText,   // alias for mindmap
	"quiz-me":              catText,   // alias for quiz
	"my-ai-photo":          catImage,  // alias for ai-photo
	"my-video-story":       catVideo,  // alias for animate-photo
	"my-podcast":           catComposite, // alias for podcast
	"background-remover":   catImage,  // alias for bg-remover
	"animate-my-photo":     catVideo,  // alias for animate-photo
	"video-story":           catVideo,  // multi-scene image-to-video (Grok reference / Kling multi-image)
	"video-edit":            catVideo,  // natural language video editing (Grok Imagine)
	"video-extend":          catVideo,  // extend existing video (Grok Imagine)
	"business-plan-summary": catText,   // alias for bizplan
	// ── Whisk-style image composition ────────────────────────────────────────────────────────────────────────────────
	"image-compose":         catImage,  // Whisk-style subject+scene+style composition (Flux Ultra)
	// ── Free chat tools ──────────────────────────────────────────────────────────────────────────────────────
	"ask-nexus":             catText,   // free conversational AI
	"nexus-chat":            catText,   // free Gemini Flash chat
	"voice-to-plan":         catText,   // voice-to-business-plan
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
	cfg        *config.ConfigManager
	studioRepo repositories.StudioRepository
	studioSvc  *StudioService
	userRepo   repositories.UserRepository
	storage    external.AssetStorage
	httpClient *http.Client
	llmOrch    *external.LLMOrchestrator // for provider health tracking
	providerDB ProviderConfigStore       // optional DB-backed provider registry
	grokClient *external.GrokAdapter     // xAI Grok for premium image/video generation
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
	grokKey := os.Getenv("XAI_API_KEY")
	var grokClient *external.GrokAdapter
	if grokKey != "" {
		grokClient = external.NewGrokAdapter(grokKey)
		log.Printf("[AIStudio] Grok (xAI) adapter initialised — premium image/video enabled")
	} else {
		log.Printf("[AIStudio] XAI_API_KEY not set — Grok premium image/video disabled")
	}
	return &AIStudioOrchestrator{
		cfg:        cfg,
		studioRepo: studioRepo,
		studioSvc:  studioSvc,
		userRepo:   userRepo,
		storage:    storage,
		httpClient: &http.Client{Timeout: 120 * time.Second},
		grokClient: grokClient,
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
	result, dispatchErr := o.route(ctx, gen)
	elapsed := int(time.Since(start).Milliseconds())

	if dispatchErr != nil {
		failErr := o.studioSvc.FailGeneration(ctx, genID, dispatchErr.Error())
		if failErr != nil {
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
	default:
		return nil, fmt.Errorf("unhandled category %q", cat)
	}
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
	// web-search-ai: primary via Pollinations gemini-search, fallback to Gemini Flash
	if slug == "web-search-ai" {
		text, err := o.callPollinationsWebSearch(ctx, prompt)
		if err == nil {
			return &studioProviderResult{OutputText: text, Provider: "pollinations/gemini-search", CostMicros: 0}, nil
		}
		log.Printf("[AIStudio] Pollinations web-search failed: %v — falling back", err)
		webSys := "You are Nexus AI, a world-class intelligent assistant with comprehensive global knowledge. " +
			"Web search is temporarily unavailable, but answer from your knowledge with depth and accuracy. " +
			"Structure your answer clearly: direct answer first, then supporting details. " +
			"Be specific with facts, numbers, and examples. Include local context naturally when the query suggests it."
		fallbackText, fErr := o.callGeminiFlash(ctx, webSys, fmt.Sprintf("(Web search unavailable — answer from knowledge) %s", prompt))
		if fErr == nil {
			return &studioProviderResult{OutputText: fallbackText, Provider: "gemini-flash/nosearch", CostMicros: 0}, nil
		}
		return nil, fmt.Errorf("web-search-ai: all providers failed: %v / %v", err, fErr)
	}

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

	// code-helper: primary via Pollinations Qwen3-Coder, fallback to Gemini Flash
	if slug == "code-helper" {
		codeSys := "You are Nexus Code, a world-class software engineer and programming mentor. " +
			"Write production-quality, clean, well-commented code. " +
			"Always wrap code in fenced code blocks with the correct language tag (e.g. ```python, ```javascript). " +
			"Explain the key logic in 2-4 bullet points after the code. " +
			"Include error handling in all examples. " +
			"If debugging, quote the problematic line, explain why it's wrong, then show the fix."
		text, err := o.callPollinationsQwenCoder(ctx, codeSys, prompt)
		if err == nil {
			return &studioProviderResult{OutputText: text, Provider: "pollinations/qwen-coder", CostMicros: 0}, nil
		}
		log.Printf("[AIStudio] Pollinations Qwen-Coder failed: %v — falling back", err)
		fallbackText, fErr := o.callGeminiFlash(ctx, codeSys, prompt)
		if fErr == nil {
			return &studioProviderResult{OutputText: fallbackText, Provider: "gemini-flash/code", CostMicros: 0}, nil
		}
		return nil, fmt.Errorf("code-helper: all providers failed: %v / %v", err, fErr)
	}

	systemPrompt, userPrompt := buildTextPrompts(slug, prompt)

	// FEAT-01: if a document was uploaded, route through Gemini multimodal (PDF/TXT analysis)
	// Applies to the 8 knowledge tools: study-guide, quiz, mindmap, research-brief,
	// bizplan, slide-deck, infographic, podcast
	knowledgeSlugs := map[string]bool{
		"study-guide": true, "quiz": true, "mindmap": true, "research-brief": true,
		"bizplan": true, "slide-deck": true, "infographic": true, "podcast": true,
	}
	if env.DocumentURL != "" && knowledgeSlugs[slug] {
		// Enrich the user prompt to instruct Gemini to use the uploaded document
		docPrompt := fmt.Sprintf("%s\n\n[The user has uploaded a document. Use its content as the primary source material for the above task. Analyse the document thoroughly and base your response on it.]", userPrompt)
		text, err := o.callGeminiWithDocument(ctx, systemPrompt, docPrompt, env.DocumentURL)
		if err == nil {
			return &studioProviderResult{OutputText: text, Provider: "gemini-flash/document", CostMicros: 0}, nil
		}
		log.Printf("[AIStudio] callGeminiWithDocument failed for %s: %v — falling back to text-only", slug, err)
		// Fall through to standard text chain below
	}

	// ── DB-first: admin can override/reorder providers per category ──────
	in := providerInput{SystemPrompt: systemPrompt, UserPrompt: userPrompt}
	if url, text, cost, usedSlug, err := o.runProviderChain(ctx, entities.ProviderCategoryText, in); err == nil {
		return &studioProviderResult{OutputText: text, OutputURL: url, Provider: "db/" + usedSlug, CostMicros: cost}, nil
	}

	// ── Hardcoded fallback chain (active when DB has no text providers) ──────
	providers := []struct {
		name string
		fn   func(ctx context.Context, sys, user string) (string, error)
	}{
		{"gemini-flash", o.callGeminiFlash},
		{"groq-llama4", o.callGroqLlama4},
		{"deepseek-v3", o.callDeepSeek},
	}

	for _, p := range providers {
		text, err := p.fn(ctx, systemPrompt, userPrompt)
		if err != nil {
			log.Printf("[AIStudio] %s failed for %s: %v", p.name, slug, err)
			continue
		}
		return &studioProviderResult{
			OutputText: text,
			Provider:   p.name,
			CostMicros: 0,
		}, nil
	}
	return nil, fmt.Errorf("all text providers failed for slug %q", slug)
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

	enhanced, err := o.callGeminiFlash(ctx, sys, userMsg)
	if err != nil || len(enhanced) < 10 {
		log.Printf("[AIStudio] enhanceImagePrompt failed: %v — using original", err)
		return userPrompt
	}
	return enhanced
}

// enhanceVideoPrompt takes a user's simple video description and expands it into
// a rich, cinematic prompt using Runway/Pika/Veo best practices.
func (o *AIStudioOrchestrator) enhanceVideoPrompt(ctx context.Context, userPrompt string) string {
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

	enhanced, err := o.callGeminiFlash(ctx, sys, userMsg)
	if err != nil || len(enhanced) < 10 {
		log.Printf("[AIStudio] enhanceVideoPrompt failed: %v — using original", err)
		return userPrompt
	}
	return enhanced
}

// ─── Image dispatch ────────────────────────────────────────────────────────────
// Handles: ai-photo, bg-remover, ai-photo-pro, ai-photo-max, ai-photo-dream, photo-editor, image-compose

func (o *AIStudioOrchestrator) dispatchImage(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	prompt := o.enhanceImagePrompt(ctx, slug, env.Prompt)
	// Remap alias slugs to canonical ones
	switch slug {
	case "my-ai-photo":
		slug = "ai-photo"
	case "background-remover":
		slug = "bg-remover"
	}

	// image-compose: Whisk-style multi-reference composition
	// Tier 1: Grok Aurora (grok-imagine-image) — true multi-image editing, up to 5 reference images
	//         Sends subject + scene + style images as image_urls array. $0.02/image.
	// Tier 2: FAL Flux Pro 1.1 Ultra — single image_url reference fallback
	if slug == "image-compose" {
		subjectURL := env.ImageURL
		if subjectURL == "" {
			return nil, fmt.Errorf("image-compose: subject image_url is required")
		}
		// Collect all reference images: subject first, then scene, then style
		imageURLs := []string{subjectURL}
		if sceneURL, ok := env.Extra["scene_image_url"].(string); ok && sceneURL != "" {
			imageURLs = append(imageURLs, sceneURL)
		}
		if styleURL, ok := env.Extra["style_image_url"].(string); ok && styleURL != "" {
			imageURLs = append(imageURLs, styleURL)
		}

		// Tier 1: Grok Aurora — true multi-image composition
		if o.grokClient != nil {
			if imgURL, err := o.grokClient.ComposeImages(ctx, prompt, imageURLs, env.AspectRatio); err == nil {
				return &studioProviderResult{OutputURL: imgURL, Provider: "grok/aurora-compose", CostMicros: 2000 * len(imageURLs)}, nil
			} else {
				log.Printf("[AIStudio] Grok Aurora compose failed for image-compose: %v — falling back to FAL Flux Ultra", err)
			}
		}

		// Tier 2: FAL Flux Pro 1.1 Ultra fallback (subject image only)
		falKey := os.Getenv("FAL_API_KEY")
		if falKey == "" {
			return nil, fmt.Errorf("image-compose: both Grok and FAL unavailable (no FAL_API_KEY)")
		}
		numImages := 1
		if n, ok := env.Extra["num_images"].(float64); ok && n >= 1 && n <= 4 {
			numImages = int(n)
		}
		imgStrength := 0.35
		if s, ok := env.Extra["image_prompt_strength"].(float64); ok && s > 0 && s <= 1.0 {
			imgStrength = s
		}
		urls, err := o.callFALFluxUltra(ctx, falKey, prompt, subjectURL, imgStrength, numImages, env.AspectRatio)
		if err != nil {
			return nil, fmt.Errorf("image-compose: %w", err)
		}
		return &studioProviderResult{OutputURL: urls[0], Provider: "fal/flux-pro-ultra", CostMicros: 40000 * numImages}, nil
	}
	switch slug {
	case "bg-remover":
		// bg-remover needs the source image URL, not the text prompt
		bgImgURL := env.ImageURL
		if bgImgURL == "" {
			bgImgURL = env.Prompt // legacy fallback: older rows stored imageURL in prompt
		}
		return o.dispatchBgRemover(ctx, bgImgURL)

	case "ai-photo-pro":
		// Tier 1: Grok Aurora (xAI) — #1 ranked image quality, $0.07/image
		if o.grokClient != nil {
			url, err := o.grokClient.GenerateImage(ctx, prompt, "2k") // 2k = $0.07/image, highest quality
			if err == nil {
				return &studioProviderResult{OutputURL: url, Provider: "grok/aurora-2k", CostMicros: 70000}, nil
			}
			log.Printf("[AIStudio] Grok Aurora 2k failed for ai-photo-pro: %v — falling back", err)
		}
		// Tier 2: GPT Image (gptimage model) — CostMicros: $0.02
		url, err := o.callPollinationsGPTImage(ctx, prompt, "gptimage", env.AspectRatio)
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "pollinations/gptimage", CostMicros: 20000}, nil
		}
		log.Printf("[AIStudio] GPTImage failed for ai-photo-pro: %v — falling back to FLUX", err)
		// Tier 3: Pollinations FLUX (free fallback)
		url, err = o.callPollinationsImage(ctx, prompt, env.AspectRatio)
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "pollinations/flux", CostMicros: 0}, nil
		}
		return nil, fmt.Errorf("ai-photo-pro: all providers failed")

	case "ai-photo-max":
		// Tier 1: Grok Aurora Pro (xAI) — #1 ranked image quality, $0.07/image
		if o.grokClient != nil {
			url, err := o.grokClient.GenerateImage(ctx, prompt, "2k") // 2k = $0.07/image, highest quality
			if err == nil {
				return &studioProviderResult{OutputURL: url, Provider: "grok/aurora-2k", CostMicros: 70000}, nil
			}
			log.Printf("[AIStudio] Grok Aurora 2k failed for ai-photo-max: %v — falling back", err)
		}
		// Tier 2: GPT Image Large — CostMicros: $0.03
		quality := "standard"
		if q, ok := env.Extra["quality"].(string); ok && q == "hd" {
			quality = "hd"
		}
		url, err := o.callPollinationsGPTImage(ctx, prompt, "gptimage-large", env.AspectRatio, quality)
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "pollinations/gptimage-large", CostMicros: 30000}, nil
		}
		log.Printf("[AIStudio] GPTImage-large failed for ai-photo-max: %v — falling back", err)
		// Tier 3: Pollinations FLUX (free fallback)
		url, err = o.callPollinationsImage(ctx, prompt, env.AspectRatio)
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "pollinations/flux", CostMicros: 0}, nil
		}
		return nil, fmt.Errorf("ai-photo-max: all providers failed")

	case "ai-photo-dream":
		// Seedream (ByteDance) — CostMicros: $0.01
		// Live model ID from GET /v1/models is "seedream5" (not "seedream")
		url, err := o.callPollinationsGPTImage(ctx, prompt, "seedream5", env.AspectRatio)
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "pollinations/seedream5", CostMicros: 10000}, nil
		}
		log.Printf("[AIStudio] Seedream5 failed for ai-photo-dream: %v — falling back", err)
		url, err = o.callPollinationsImage(ctx, prompt, env.AspectRatio)
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "pollinations/flux", CostMicros: 0}, nil
		}
		return nil, fmt.Errorf("ai-photo-dream: all providers failed")

	case "photo-editor":
		// Image-to-image editing — upgraded provider chain (2025-04)
		// Tier 1: p-image-edit (Pruna/Pollinations, 98.1% success) — fast, reliable
		// Tier 2: FAL.ai FLUX.1 Kontext Pro — highest quality image editing
		// Tier 3: GPT-Image-Large — generate from instruction prompt (100% success, no source image)
		// Frontend sends: { prompt: instruction, image_url: imgURL }
		imgURL := env.ImageURL
		instruction := env.Prompt
		if imgURL == "" {
			return nil, fmt.Errorf("photo-editor: image_url is required")
		}
		// Read strength from extra_params (0.0–1.0, default 0.75)
		strength := 0.75
		if env.Extra != nil {
			if s, ok := env.Extra["strength"].(float64); ok && s > 0 && s <= 1.0 {
				strength = s
			}
		}
		// Append strength hint to instruction for providers that use text guidance
		if strength < 0.5 {
			instruction += " (subtle change, preserve most of the original)"
		} else if strength > 0.85 {
			instruction += " (strong transformation)"
		}
		// Tier 1: p-image-edit (Pruna) — image-to-image editing, 98.1% success
		url, err := o.callPollinationsKontextAlt(ctx, imgURL, instruction)
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "pollinations/p-image-edit", CostMicros: 10000}, nil
		}
		log.Printf("[AIStudio] p-image-edit failed for photo-editor: %v — trying FAL kontext", err)
		// Tier 2: FAL.ai FLUX.1 Kontext Pro — best quality, paid
		falKey := os.Getenv("FAL_API_KEY")
		if falKey != "" {
			url, err = o.callFALImageEdit(ctx, falKey, imgURL, instruction)
			if err == nil {
				return &studioProviderResult{OutputURL: url, Provider: "fal/flux-kontext", CostMicros: 20000}, nil
			}
			log.Printf("[AIStudio] FAL kontext failed for photo-editor: %v — falling back to gptimage-large", err)
		}
		// Tier 3: gptimage-large — generate from instruction prompt (no source image, 100% success)
		url, err = o.callPollinationsGPTImage(ctx, instruction, "gptimage-large")
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "pollinations/gptimage-large", CostMicros: 30000}, nil
		}
		return nil, fmt.Errorf("photo-editor: all providers failed: %w", err)

	default: // ai-photo — also handles ai-photo with reference image (Whisk-style)
		// If a reference image is provided, route to Flux Pro 1.1 Ultra for image-guided generation
		if env.ImageURL != "" {
			if falKey := os.Getenv("FAL_API_KEY"); falKey != "" {
				numImages := 1
				if n, ok := env.Extra["num_images"].(float64); ok && n >= 1 && n <= 4 {
					numImages = int(n)
				}
				imgStrength := 0.3
				if s, ok := env.Extra["image_prompt_strength"].(float64); ok && s > 0 && s <= 1.0 {
					imgStrength = s
				}
				urls, err := o.callFALFluxUltra(ctx, falKey, prompt, env.ImageURL, imgStrength, numImages, env.AspectRatio)
				if err == nil {
					return &studioProviderResult{OutputURL: urls[0], Provider: "fal/flux-pro-ultra", CostMicros: 40000 * numImages}, nil
				}
				log.Printf("[AIStudio] FAL Flux Ultra (reference) failed: %v — falling back to standard generation", err)
			}
		}
		// ── DB-first ────────────────────────────────────────────────────────
		in := providerInput{Prompt: prompt}
		if url, _, cost, usedSlug, err := o.runProviderChain(ctx, entities.ProviderCategoryImage, in); err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "db/" + usedSlug, CostMicros: cost}, nil
		}
		// ── Hardcoded fallback ───────────────────────────────────────────────
		// Extract user-supplied seed from extra_params (sent by ImageCreator Advanced Settings)
		var userSeed int64
		if env.Extra != nil {
			if s, ok := env.Extra["seed"].(float64); ok && s > 0 {
				userSeed = int64(s)
			}
		}
		// tier 1: HuggingFace FLUX.1-Schnell (free, uses HF_TOKEN)
		if hfKey := os.Getenv("HF_TOKEN"); hfKey != "" {
			url, err := o.callHFFluxSchnell(ctx, hfKey, prompt)
			if err == nil {
				return &studioProviderResult{OutputURL: url, Provider: "huggingface/flux-schnell", CostMicros: 0}, nil
			}
			log.Printf("[AIStudio] HF FLUX.1-Schnell failed: %v", err)
		}
		// tier 2: Pollinations.ai FLUX — pass user seed for reproducible generation
		if url, err := o.callPollinationsImageWithSeed(ctx, prompt, userSeed, env.AspectRatio); err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "pollinations/flux", CostMicros: 0}, nil
		}
		// tier 3: FAL.AI FLUX-dev (paid fallback)
		if falKey := os.Getenv("FAL_API_KEY"); falKey != "" {
			url, err := o.callFALFlux(ctx, falKey, prompt)
			if err == nil {
				return &studioProviderResult{OutputURL: url, Provider: "fal.ai/flux-dev", CostMicros: 6500}, nil
			}
			log.Printf("[AIStudio] FAL FLUX failed: %v", err)
		}
		return nil, fmt.Errorf("image generation unavailable: configure HF_TOKEN or FAL_API_KEY")
	}
}

func (o *AIStudioOrchestrator) dispatchBgRemover(ctx context.Context, imageURL string) (*studioProviderResult, error) {
	// ── DB-first ─────────────────────────────────────────────────────────────
	in := providerInput{ImageURL: imageURL}
	if url, _, cost, usedSlug, err := o.runProviderChain(ctx, entities.ProviderCategoryBGRemove, in); err == nil {
		return &studioProviderResult{OutputURL: url, Provider: "db/" + usedSlug, CostMicros: cost}, nil
	}

	// ── Hardcoded fallback chain ──────────────────────────────────────────────
	// Primary: self-hosted rembg microservice
	if rembgURL := os.Getenv("REMBG_SERVICE_URL"); rembgURL != "" {
		result, err := o.callRembgService(ctx, rembgURL, imageURL)
		if err == nil {
			return &studioProviderResult{OutputURL: result, Provider: "rembg/self-hosted", CostMicros: 0}, nil
		}
		log.Printf("[AIStudio] rembg failed: %v", err)
	}

	// Fallback: FAL.AI BiRefNet (accurate background removal)
	if falKey := os.Getenv("FAL_API_KEY"); falKey != "" {
		result, err := o.callFALBgRemover(ctx, falKey, imageURL)
		if err == nil {
			return &studioProviderResult{OutputURL: result, Provider: "fal.ai/birefnet", CostMicros: 2000}, nil
		}
		log.Printf("[AIStudio] FAL BiRefNet failed: %v", err)
	}

	// Last resort: remove.bg API
	if rbgKey := os.Getenv("REMOVEBG_API_KEY"); rbgKey != "" {
		result, err := o.callRemoveBg(ctx, rbgKey, imageURL)
		if err == nil {
			return &studioProviderResult{OutputURL: result, Provider: "remove.bg", CostMicros: 1000}, nil
		}
		log.Printf("[AIStudio] remove.bg failed: %v", err)
	}

	return nil, fmt.Errorf("background removal unavailable: configure REMBG_SERVICE_URL, FAL_API_KEY, or REMOVEBG_API_KEY")
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
	// Remap alias slugs to canonical ones
	switch slug {
	case "animate-my-photo":
		// Single-image animation — maps to the animate-photo path
		slug = "animate-photo"
	case "my-video-story":
		// Script-driven animation — handled separately below (accepts 1+ images).
		// Do NOT remap to video-story which requires 2+ images.
	}

	// ── video-edit: Natural language video editing via Grok Imagine ───────────────────────────
	if slug == "video-edit" {
		if o.grokClient == nil {
			return nil, fmt.Errorf("video-edit: XAI_API_KEY not configured")
		}
		videoURL := env.ImageURL // frontend sends video_url in image_url field for simplicity
		if v, ok := env.Extra["video_url"].(string); ok && v != "" {
			videoURL = v
		}
		if videoURL == "" {
			return nil, fmt.Errorf("video-edit: video_url is required")
		}
		prompt := env.Prompt
		if prompt == "" {
			return nil, fmt.Errorf("video-edit: edit instruction (prompt) is required")
		}
		grokReq := external.GrokVideoRequest{
			Prompt:   prompt,
			VideoURL: videoURL,
		}
		url, err := o.grokClient.GenerateVideo(ctx, grokReq)
		if err != nil {
			return nil, fmt.Errorf("video-edit: %w", err)
		}
		// Cost: $0.05/sec × estimated 6s output = $0.30 = 300000 µUSD
		return &studioProviderResult{OutputURL: url, Provider: "grok/imagine-video-edit", CostMicros: 300000}, nil
	}

	// ── video-extend: Extend an existing video via Grok Imagine ───────────────────────────
	if slug == "video-extend" {
		if o.grokClient == nil {
			return nil, fmt.Errorf("video-extend: XAI_API_KEY not configured")
		}
		videoURL := env.ImageURL
		if v, ok := env.Extra["video_url"].(string); ok && v != "" {
			videoURL = v
		}
		if videoURL == "" {
			return nil, fmt.Errorf("video-extend: video_url is required")
		}
		duration := env.Duration
		if duration <= 0 {
			duration = 6
		}
		grokReq := external.GrokVideoRequest{
			Prompt:      env.Prompt,
			VideoURL:    videoURL,
			Extend:      true,
			Duration:    duration,
			AspectRatio: env.AspectRatio,
			Resolution:  "720p",
		}
		url, err := o.grokClient.GenerateVideo(ctx, grokReq)
		if err != nil {
			return nil, fmt.Errorf("video-extend: %w", err)
		}
		costMicros := 50000 * duration // $0.05/sec
		return &studioProviderResult{OutputURL: url, Provider: "grok/imagine-video-extend", CostMicros: costMicros}, nil
	}

	// ── my-video-story: script-driven animation (1+ images) ──────────────────────────────────────
	// Same provider chain as video-story but requires only 1 image.
	if slug == "my-video-story" {
		imageURLsRaw := env.Extra["image_urls"]
		var imageURLs []string
		if arr, ok := imageURLsRaw.([]interface{}); ok {
			for _, v := range arr {
				if s, ok2 := v.(string); ok2 && s != "" {
					imageURLs = append(imageURLs, s)
				}
			}
		}
		// Also accept a single image_url field for single-scene stories
		if len(imageURLs) == 0 && env.ImageURL != "" {
			imageURLs = []string{env.ImageURL}
		}
		if len(imageURLs) < 1 {
			return nil, fmt.Errorf("my-video-story: at least 1 image is required")
		}
		prompt := o.enhanceVideoPrompt(ctx, env.Prompt)
		// Tier 1: Grok reference-image mode (supports up to 7 images)
		if o.grokClient != nil {
			duration := env.Duration
			if duration <= 0 {
				duration = 10
			}
			grokReq := external.GrokVideoRequest{
				Prompt:             prompt,
				ReferenceImageURLs: imageURLs,
				Duration:           duration,
				AspectRatio:        env.AspectRatio,
				Resolution:         "720p",
			}
			if url, err := o.grokClient.GenerateVideo(ctx, grokReq); err == nil {
				costMicros := 50000 * duration
				return &studioProviderResult{OutputURL: url, Provider: "grok/imagine-reference", CostMicros: costMicros}, nil
			} else {
				log.Printf("[AIStudio] Grok reference-image failed for my-video-story: %v — falling back to Kling", err)
			}
		}
		// Tier 2: FAL Kling v1.6 multi-image fallback
		if len(imageURLs) > 4 {
			imageURLs = imageURLs[:4]
		}
		falKey := os.Getenv("FAL_API_KEY")
		if falKey == "" {
			return nil, fmt.Errorf("my-video-story: FAL_API_KEY not configured")
		}
		vidURL, err := o.callFALMultiImageVideo(ctx, falKey, imageURLs, prompt, env)
		if err != nil {
			return nil, fmt.Errorf("my-video-story: %w", err)
		}
		return &studioProviderResult{OutputURL: vidURL, Provider: "fal.ai/kling-multi-image", CostMicros: 56000}, nil
	}

	// ── video-story: multi-scene image-to-video ────────────────────────────────────────────────────
	// Tier 1: Grok reference-image mode (up to 7 images, $0.05/sec)
	// Tier 2: FAL Kling v1.6 multi-image fallback
	if slug == "video-story" {
		imageURLsRaw := env.Extra["image_urls"]
		var imageURLs []string
		if arr, ok := imageURLsRaw.([]interface{}); ok {
			for _, v := range arr {
				if s, ok2 := v.(string); ok2 && s != "" {
					imageURLs = append(imageURLs, s)
				}
			}
		}
		if len(imageURLs) < 2 {
			return nil, fmt.Errorf("video-story: at least 2 images required, got %d", len(imageURLs))
		}
		prompt := o.enhanceVideoPrompt(ctx, env.Prompt)
		// Tier 1: Grok reference-image mode (supports up to 7 images)
		if o.grokClient != nil {
			duration := env.Duration
			if duration <= 0 {
				duration = 10
			}
			grokReq := external.GrokVideoRequest{
				Prompt:             prompt,
				ReferenceImageURLs: imageURLs,
				Duration:           duration,
				AspectRatio:        env.AspectRatio,
				Resolution:         "720p",
			}
			if url, err := o.grokClient.GenerateVideo(ctx, grokReq); err == nil {
				costMicros := 50000 * duration
				return &studioProviderResult{OutputURL: url, Provider: "grok/imagine-reference", CostMicros: costMicros}, nil
			} else {
				log.Printf("[AIStudio] Grok reference-image failed for video-story: %v — falling back to Kling", err)
			}
		}
		// Tier 2: FAL Kling v1.6 multi-image fallback
		if len(imageURLs) > 4 {
			imageURLs = imageURLs[:4]
		}
		falKey := os.Getenv("FAL_API_KEY")
		if falKey == "" {
			return nil, fmt.Errorf("video-story: FAL_API_KEY not configured")
		}
		vidURL, err := o.callFALMultiImageVideo(ctx, falKey, imageURLs, prompt, env)
		if err != nil {
			return nil, fmt.Errorf("video-story: %w", err)
		}
		return &studioProviderResult{OutputURL: vidURL, Provider: "fal.ai/kling-multi-image", CostMicros: 56000}, nil
	}

	// video-cinematic: high-quality cinematic image-to-video
	// Primary: wan-fast (Wan 2.2) — FREE, 15 pollen input, ~50s, image-to-video
	// Fallback: ltx-2 (LTX-2)   — FREE, 15 pollen input, NEW model
	if slug == "video-cinematic" {
		imgURL := env.ImageURL
		motionPrompt := o.enhanceVideoPrompt(ctx, env.Prompt)
		// Inject motion intensity hint into prompt (sent by VideoCreator slider)
		if mi, ok := env.Extra["motion_intensity"].(float64); ok && mi > 0 {
			motionHints := map[int]string{
				1: "very subtle motion, minimal movement",
				2: "gentle motion, slow and smooth",
				3: "balanced motion, natural movement",
				4: "dynamic motion, expressive movement",
				5: "extreme motion, high energy, dramatic movement",
			}
			if hint, ok2 := motionHints[int(mi)]; ok2 {
				motionPrompt = motionPrompt + ". Motion style: " + hint
			}
		}
		if imgURL == "" {
			return nil, fmt.Errorf("video-cinematic: image_url is required")
		}
		// Tier 1: wan-fast (Wan 2.2) — FREE, 91.4% success; audio=true enables ambient sound
		vidURL, err := o.callPollinationsVideoModel(ctx, "wan-fast", imgURL, motionPrompt, 180, env.AspectRatio, fmt.Sprintf("%d", env.Duration), "true")
		if err == nil {
			return &studioProviderResult{OutputURL: vidURL, Provider: "pollinations/wan-fast", CostMicros: 0}, nil
		}
		log.Printf("[AIStudio] wan-fast failed for video-cinematic: %v — trying p-video", err)
		// Tier 2: p-video (Pruna p-video) — FREE, 100% success; audio=true enables ambient sound
		vidURL, err = o.callPollinationsVideoModel(ctx, "p-video", imgURL, motionPrompt, 180, env.AspectRatio, fmt.Sprintf("%d", env.Duration), "true")
		if err == nil {
			return &studioProviderResult{OutputURL: vidURL, Provider: "pollinations/p-video", CostMicros: 0}, nil
		}
		log.Printf("[AIStudio] p-video failed for video-cinematic: %v", err)
		return nil, fmt.Errorf("video-cinematic: all providers failed")
	}

	// video-veo: Premium text-to-video with Grok (xAI) as Tier 1
	// Tier 1: Grok Imagine Video (xAI) — $0.05/sec, top-tier quality, native audio
	// Tier 2: Google Veo 3.1 (Pollinations) — $0.150/sec
	if slug == "video-veo" {
		prompt := o.enhanceVideoPrompt(ctx, env.Prompt)
		// Inject motion intensity hint into prompt (sent by VideoCreator slider)
		if mi, ok := env.Extra["motion_intensity"].(float64); ok && mi > 0 {
			motionHints := map[int]string{
				1: "very subtle motion, minimal movement",
				2: "gentle motion, slow and smooth",
				3: "balanced motion, natural movement",
				4: "dynamic motion, expressive movement",
				5: "extreme motion, high energy, dramatic movement",
			}
			if hint, ok2 := motionHints[int(mi)]; ok2 {
				prompt = prompt + ". Motion style: " + hint
			}
		}
		// Tier 1: Grok Imagine Video (xAI) — $0.05/sec
		if o.grokClient != nil {
			duration := env.Duration
			if duration <= 0 {
				duration = 6
			}
			ar := env.AspectRatio
			if ar == "" {
				ar = "16:9"
			}
			grokReq := external.GrokVideoRequest{
				Prompt:      prompt,
				Duration:    duration,
				AspectRatio: ar,
				Resolution:  "720p",
			}
			vidURL, err := o.grokClient.GenerateVideo(ctx, grokReq)
			if err == nil {
				costMicros := 50000 * duration
				return &studioProviderResult{OutputURL: vidURL, Provider: "grok/imagine-video", CostMicros: costMicros}, nil
			}
			log.Printf("[AIStudio] Grok Imagine Video failed for video-veo: %v — trying Veo", err)
		}
		// Tier 2: Google Veo 3.1 (Pollinations) — $0.150/sec
		// Always enable audio for Veo; audio_direction is appended to prompt as a hint
		if audioHint, ok := env.Extra["audio_direction"]; ok {
			if hint := fmt.Sprintf("%v", audioHint); hint != "" && hint != "<nil>" {
				prompt = prompt + ". Audio: " + hint
			}
		}
		vidURL, err := o.callPollinationsVeo(ctx, prompt, env.AspectRatio, true)
		if err == nil {
			return &studioProviderResult{OutputURL: vidURL, Provider: "pollinations/veo", CostMicros: 400000}, nil
		}
		log.Printf("[AIStudio] Veo failed for video-veo: %v — falling back to wan-fast (FREE)", err)
		// Fallback 1: wan-fast text-to-video (FREE) — NOT seedance (also paid); audio=true
		vidURL, err = o.callPollinationsVideoModel(ctx, "wan-fast", "", prompt, 180, env.AspectRatio, fmt.Sprintf("%d", env.Duration), "true")
		if err == nil {
			return &studioProviderResult{OutputURL: vidURL, Provider: "pollinations/wan-fast", CostMicros: 0}, nil
		}
		log.Printf("[AIStudio] wan-fast fallback failed for video-veo: %v — trying p-video", err)
		// Fallback 2: p-video (Pruna p-video) — FREE, 100% success; audio=true
		vidURL, err = o.callPollinationsVideoModel(ctx, "p-video", "", prompt, 180, env.AspectRatio, fmt.Sprintf("%d", env.Duration), "true")
		if err == nil {
			return &studioProviderResult{OutputURL: vidURL, Provider: "pollinations/p-video", CostMicros: 0}, nil
		}
		log.Printf("[AIStudio] p-video fallback failed for video-veo: %v", err)
		return nil, fmt.Errorf("video-veo: all providers failed")
	}

	// Standard video slugs (animate-photo, video-premium, video-jingle)
	// Frontend sends image_url in the envelope; prompt is the motion description
	imageURL := env.ImageURL
	if imageURL == "" {
		imageURL = env.Prompt // legacy fallback: older rows stored imageURL in prompt
	}

	// ── Tier 0: Grok image-to-video (for video-premium and animate-photo) ───────────────────────────
	if o.grokClient != nil && imageURL != "" && (slug == "video-premium" || slug == "animate-photo") {
		duration := env.Duration
		if duration <= 0 {
			duration = 6
		}
		ar := env.AspectRatio
		if ar == "" {
			ar = "16:9"
		}
		grokReq := external.GrokVideoRequest{
			Prompt:      o.enhanceVideoPrompt(ctx, env.Prompt),
			ImageURL:    imageURL,
			Duration:    duration,
			AspectRatio: ar,
			Resolution:  "720p",
		}
		if vidURL, err := o.grokClient.GenerateVideo(ctx, grokReq); err == nil {
			costMicros := 50000 * duration
			return &studioProviderResult{OutputURL: vidURL, Provider: "grok/imagine-i2v", CostMicros: costMicros}, nil
		} else {
			log.Printf("[AIStudio] Grok image-to-video failed for %s: %v — falling back", slug, err)
		}
	}

	// ── DB-first ─────────────────────────────────────────────────────────
	vidIn := providerInput{Prompt: env.Prompt, ImageURL: imageURL}
	if url, _, cost, usedSlug, err := o.runProviderChain(ctx, entities.ProviderCategoryVideo, vidIn); err == nil {
		return &studioProviderResult{OutputURL: url, Provider: "db/" + usedSlug, CostMicros: cost}, nil
	}

	// ── Hardcoded fallback chain ────────────────────────────────────────────
	// Tier 1: FAL.AI (Kling v1.5 for premium, LTX for standard)
	if falKey := os.Getenv("FAL_API_KEY"); falKey != "" {
		var model string
		switch slug {
		case "video-premium":
			model = "fal-ai/kling-video/v2.6/pro/image-to-video"
		case "video-jingle":
			model = "fal-ai/kling-video/v2.6/pro/image-to-video"
		default: // animate-photo
			model = "fal-ai/ltx-video"
		}

		videoURL, err := o.callFALVideo(ctx, falKey, model, imageURL, o.enhanceVideoPrompt(ctx, env.Prompt), env)
		if err != nil {
			// Fallback for Kling → LTX within FAL (LTX doesn't support end_image_url, pass empty env)
			if slug == "video-premium" {
				log.Printf("[AIStudio] Kling failed, falling back to LTX: %v", err)
				videoURL, err = o.callFALVideo(ctx, falKey, "fal-ai/ltx-video", imageURL, o.enhanceVideoPrompt(ctx, env.Prompt))
			}
		}
		if err == nil {
			costMicros := 14500
			if slug == "video-premium" {
				costMicros = 56000
			}
			return &studioProviderResult{OutputURL: videoURL, Provider: "fal.ai/" + model, CostMicros: costMicros}, nil
		}
		log.Printf("[AIStudio] FAL video failed: %v", err)
	}

	// Tier 2: Pollinations wan-fast / Wan 2.2 — FREE, 91.4% success; audio=true for ambient sound
	// NOTE: seedance is PAID (1.8 pollen/M) — never use as a free fallback
	motionDesc := o.enhanceVideoPrompt(ctx, env.Prompt)
	if motionDesc == "" {
		motionDesc = "animate this image with subtle cinematic motion, smooth camera movement, natural lighting"
	}
	if videoURL, err := o.callPollinationsVideoModel(ctx, "wan-fast", imageURL, motionDesc, 180, env.AspectRatio, fmt.Sprintf("%d", env.Duration), "true"); err == nil {
		return &studioProviderResult{OutputURL: videoURL, Provider: "pollinations/wan-fast", CostMicros: 0}, nil
	} else {
		log.Printf("[AIStudio] Pollinations wan-fast failed: %v — trying p-video", err)
	}

	// Tier 3: Pollinations p-video (Pruna) — FREE, 100% success; audio=true for ambient sound
	if videoURL, err := o.callPollinationsVideoModel(ctx, "p-video", imageURL, motionDesc, 180, env.AspectRatio, fmt.Sprintf("%d", env.Duration), "true"); err == nil {
		return &studioProviderResult{OutputURL: videoURL, Provider: "pollinations/p-video", CostMicros: 0}, nil
	} else {
		log.Printf("[AIStudio] Pollinations p-video failed: %v", err)
	}

	return nil, fmt.Errorf("video generation unavailable: all providers failed")
}

// ─── Voice / Translate dispatch ───────────────────────────────────────────────

func (o *AIStudioOrchestrator) dispatchVoiceOrTranslate(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	switch slug {
	case "translate", "local-translation":
		return o.dispatchTranslate(ctx, env)
	case "transcribe":
		return o.dispatchTranscribe(ctx, env) // env.Prompt = audioURL, env.Language = language code
	case "transcribe-african":
		return o.dispatchTranscribeAfrican(ctx, env)
	case "narrate-pro":
		return o.dispatchNarratorPro(ctx, env)
	case "text-to-speech":
		// text-to-speech is an alias for narrate — use voice_id if set
		if env.VoiceID != "" {
			return o.dispatchNarratorPro(ctx, env)
		}
		return o.dispatchTTS(ctx, env.Prompt)
	default: // narrate
		// Use voice_id from envelope if set, else fall back to generic TTS
		if env.VoiceID != "" {
			return o.dispatchNarratorPro(ctx, env)
		}
		return o.dispatchTTS(ctx, env.Prompt)
	}
}

func (o *AIStudioOrchestrator) dispatchTranslate(ctx context.Context, env promptEnvelope) (*studioProviderResult, error) {
	// Frontend sends: { prompt: textToTranslate, language: "yo" } via buildEnrichedPrompt
	// Legacy fallback: if language is empty, default to Yoruba
	targetLang := strings.ToLower(strings.TrimSpace(env.Language))
	text := env.Prompt
	if targetLang == "" || targetLang == "auto" {
		targetLang = "yo" // default to Yoruba
	}
	// Validate supported languages
	supportedLangs := map[string]bool{"yo": true, "ha": true, "ig": true, "fr": true, "en": true}
	if !supportedLangs[targetLang] {
		targetLang = "yo"
	}

	// ── DB-first ─────────────────────────────────────────────────────────────
	dbIn := providerInput{UserPrompt: text, TargetLang: targetLang}
	if _, translated, cost, usedSlug, err := o.runProviderChain(ctx, entities.ProviderCategoryTranslate, dbIn); err == nil {
		return &studioProviderResult{OutputText: translated, Provider: "db/" + usedSlug, CostMicros: cost}, nil
	}

	// ── Hardcoded fallback chain ──────────────────────────────────────────────
	if apiKey := os.Getenv("GOOGLE_TRANSLATE_API_KEY"); apiKey != "" {
		translated, err := o.callGoogleTranslate(ctx, apiKey, text, targetLang)
		if err == nil {
			return &studioProviderResult{
				OutputText: translated,
				Provider:   "google-translate",
				CostMicros: 0,
			}, nil
		}
		log.Printf("[AIStudio] Google Translate failed: %v", err)
	}

	// Fallback: use Gemini for translation
	prompt := fmt.Sprintf("Translate the following text to %s. Return ONLY the translation, no explanation, no commentary, no quotation marks around the result:\n\n%s", targetLang, text)
	transSys := "You are a professional translator with native-level fluency in all major world languages, including French, Spanish, Arabic, Chinese, German, Portuguese, Swahili, Yoruba, Igbo, Hausa, and Nigerian Pidgin English. " +
		"Preserve the original tone, style, register, and meaning precisely — do not paraphrase or summarise. " +
		"For idiomatic expressions, find the natural equivalent in the target language rather than a literal translation. " +
		"Return ONLY the translated text — no explanations, no notes, no quotation marks."
	translated, err := o.callGeminiFlash(ctx, transSys, prompt)
	if err == nil {
		return &studioProviderResult{OutputText: translated, Provider: "gemini/translate", CostMicros: 0}, nil
	}

	return nil, fmt.Errorf("translation unavailable: configure GOOGLE_TRANSLATE_API_KEY")
}

func (o *AIStudioOrchestrator) dispatchTTS(ctx context.Context, text string) (*studioProviderResult, error) {
	// ── DB-first ─────────────────────────────────────────────────────────────
	in := providerInput{Text: text}
	if url, _, cost, usedSlug, err := o.runProviderChain(ctx, entities.ProviderCategoryTTS, in); err == nil {
		return &studioProviderResult{OutputURL: url, Provider: "db/" + usedSlug, CostMicros: cost}, nil
	}

	// ── Hardcoded fallback chain ──────────────────────────────────────────────
	// Primary: Google Cloud TTS (free tier: 1M chars/month standard)
	if gcpKey := os.Getenv("GOOGLE_CLOUD_TTS_KEY"); gcpKey != "" {
		audioURL, err := o.callGoogleCloudTTS(ctx, gcpKey, text)
		if err == nil {
			return &studioProviderResult{OutputURL: audioURL, Provider: "google-cloud-tts", CostMicros: 0}, nil
		}
		log.Printf("[AIStudio] Google Cloud TTS failed: %v", err)
	}

	// Secondary: ElevenLabs TTS (premium quality)
	if el11Key := os.Getenv("ELEVENLABS_API_KEY"); el11Key != "" {
		voiceID := os.Getenv("ELEVENLABS_VOICE_ID")
		if voiceID == "" {
			voiceID = "EXAVITQu4vr4xnSDxMaL" // Sarah - premade voice, accessible on free tier
		}
		audioURL, err := o.callElevenLabsTTS(ctx, el11Key, voiceID, text)
		if err == nil {
			return &studioProviderResult{OutputURL: audioURL, Provider: "elevenlabs-tts", CostMicros: 2000}, nil
		}
		log.Printf("[AIStudio] ElevenLabs TTS failed: %v", err)
	}

	// Fallback: HuggingFace Bark (free, lower quality)
	if hfKey := os.Getenv("HF_TOKEN"); hfKey != "" {
		audioURL, err := o.callHuggingFaceTTS(ctx, hfKey, text)
		if err == nil {
			return &studioProviderResult{OutputURL: audioURL, Provider: "huggingface/bark", CostMicros: 0}, nil
		}
	}

	// Last resort: Pollinations.ai TTS (free, OpenAI-compatible, 30+ voices, no key)
	if audioURL, err := o.callPollinationsTTS(ctx, text, "nova"); err == nil {
		return &studioProviderResult{OutputURL: audioURL, Provider: "pollinations/tts", CostMicros: 0}, nil
	}

	return nil, fmt.Errorf("TTS unavailable: configure GOOGLE_CLOUD_TTS_KEY or ELEVENLABS_API_KEY")
}

func (o *AIStudioOrchestrator) dispatchTranscribe(ctx context.Context, env promptEnvelope) (*studioProviderResult, error) {
	audioURL := env.Prompt // audioURL is stored in the prompt field for transcribe
	lang := strings.ToLower(strings.TrimSpace(env.Language))
	if lang == "" {
		lang = "en"
	}
	// Read speaker_labels and output_format from extra_params
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
	// ── DB-first ─────────────────────────────────────────────────────────────────────────────
	in := providerInput{AudioURL: audioURL}
	if _, text, cost, usedSlug, err := o.runProviderChain(ctx, entities.ProviderCategoryTranscribe, in); err == nil {
		formatted := o.formatTranscript(text, outputFormat)
		return &studioProviderResult{OutputText: formatted, Provider: "db/" + usedSlug, CostMicros: cost}, nil
	}
	// ── Hardcoded fallback chain ─────────────────────────────────────────────────────────────────────────────
	// Primary: AssemblyAI (free $50 credit on signup) — supports language_code + speaker diarization
	if aaiKey := os.Getenv("ASSEMBLY_AI_KEY"); aaiKey != "" {
		text, err := o.callAssemblyAIFull(ctx, aaiKey, audioURL, lang, speakerLabels, outputFormat)
		if err == nil {
			return &studioProviderResult{OutputText: text, Provider: "assemblyai", CostMicros: 25}, nil
		}
		log.Printf("[AIStudio] AssemblyAI failed: %v", err)
	}
	// Fallback: Groq Whisper-large-v3 (fast, cheap)
	if groqKey := os.Getenv("GROQ_API_KEY"); groqKey != "" {
		text, err := o.callGroqWhisper(ctx, groqKey, audioURL)
		if err == nil {
			formatted := o.formatTranscript(text, outputFormat)
			return &studioProviderResult{OutputText: formatted, Provider: "groq/whisper-large-v3", CostMicros: 10}, nil
		}
		log.Printf("[AIStudio] Groq Whisper failed: %v", err)
	}
	return nil, fmt.Errorf("transcription unavailable: configure ASSEMBLY_AI_KEY or GROQ_API_KEY")
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
		"audio_url":          audioURL,
		"language_code":      lang,
		"speech_models":      []string{"universal-2"},
		"speaker_labels":     speakerLabels,
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
			Status    string `json:"status"`
			Text      string `json:"text"`
			Error     string `json:"error"`
			Utterances []struct {
				Speaker string  `json:"speaker"`
				Text    string  `json:"text"`
				Start   int     `json:"start"`
				End     int     `json:"end"`
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
	// ── Enrich prompt with all music-specific envelope fields ──────────────────
	// This matches Suno/Udio behaviour: key, structure, instruments, negative
	// prompt, and user lyrics are all forwarded to the AI provider.
	prompt := env.Prompt

	// Musical key (e.g. "C major", "F# minor")
	if env.Extra != nil {
		if key, ok := env.Extra["key"].(string); ok && key != "" && key != "Any" {
			prompt += fmt.Sprintf(" Key of %s.", key)
		}
		// Song structure preset (e.g. "Verse-Chorus-Verse-Chorus-Bridge-Chorus")
		if structure, ok := env.Extra["structure"].(string); ok && structure != "" && structure != "Auto" {
			prompt += fmt.Sprintf(" Song structure: %s.", structure)
		}
		// Instrument focus (e.g. "guitar, piano, light percussion")
		if instruments, ok := env.Extra["instruments"].(string); ok && instruments != "" {
			prompt += fmt.Sprintf(" Featured instruments: %s.", instruments)
		}
	}

	// Negative prompt — what to avoid (e.g. "no drums", "no distortion")
	if env.NegativePrompt != "" {
		prompt += fmt.Sprintf(" Avoid: %s.", env.NegativePrompt)
	}

	// User-supplied lyrics — appended last so the AI treats them as lyric content.
	// Supports Suno-style section tags: [Verse], [Chorus], [Bridge], [Outro].
	if env.Lyrics != "" {
		prompt += fmt.Sprintf("\n\nLyrics:\n%s", env.Lyrics)
	}

	switch slug {
	case "song-creator":
		// Full song with vocals.
		// Tier 1: Suno AI (best-in-class quality, 2 takes per generation)
		if sunoKey := os.Getenv("SUNO_API_KEY"); sunoKey != "" {
			// Extract style and title from envelope extras for richer Suno output
			sunoStyle := ""
			sunoTitle := "My Song"
			if env.Extra != nil {
				if s, ok := env.Extra["genre"].(string); ok && s != "" {
					sunoStyle = s
				}
				if t, ok := env.Extra["title"].(string); ok && t != "" {
					sunoTitle = t
				}
			}
			if sunoStyle == "" {
				sunoStyle = "Pop"
			}
			// Extract vocal gender from envelope extras (female/male/mixed)
			sunoVocalGender := "female" // default
			if env.Extra != nil {
				if vg, ok := env.Extra["vocal_gender"].(string); ok && vg != "" {
					sunoVocalGender = vg
				}
			}
			audioURL1, audioURL2, err := o.callSunoMusic(ctx, sunoKey, prompt, sunoStyle, sunoTitle, sunoVocalGender, false)
			if err == nil {
				return &studioProviderResult{OutputURL: audioURL1, OutputURL2: audioURL2, Provider: "suno-ai", CostMicros: 50000}, nil
			}
			log.Printf("[AIStudio] Suno failed for song-creator: %v — trying ElevenLabs", err)
		}
		// Tier 2: ElevenLabs Music direct API (professional quality, ~$0.045/song)
		if el11Key := os.Getenv("ELEVENLABS_API_KEY"); el11Key != "" {
			audioURL, err := o.callElevenLabsMusic(ctx, el11Key, prompt)
			if err == nil {
				return &studioProviderResult{OutputURL: audioURL, Provider: "elevenlabs-music", CostMicros: 45000}, nil
			}
			log.Printf("[AIStudio] ElevenLabs Music failed for song-creator: %v — trying Pollinations", err)
		}
		// Tier 3: Pollinations ElevenMusic (kept as secondary in case it recovers)
		if audioURL, err := o.callPollinationsElevenMusic(ctx, prompt, false); err == nil {
			return &studioProviderResult{OutputURL: audioURL, Provider: "pollinations/elevenmusic", CostMicros: 100000}, nil
		}
		// Tier 4: HuggingFace MusicGen-small (FREE — uses HF_TOKEN already required for image gen)
		if hfKey := os.Getenv("HF_TOKEN"); hfKey != "" {
			if audioURL, err := o.callHFMusicGen(ctx, hfKey, prompt, 30); err == nil {
				return &studioProviderResult{OutputURL: audioURL, Provider: "huggingface/musicgen-small", CostMicros: 0}, nil
			}
			log.Printf("[AIStudio] HF MusicGen failed for song-creator: skipping")
		}
		return nil, fmt.Errorf("song-creator: all providers failed — configure SUNO_API_KEY, ELEVENLABS_API_KEY, POLLINATIONS_SECRET_KEY, or HF_TOKEN")

	case "instrumental":
		// Instrumental track (no vocals).
		// Tier 1: Suno AI (best-in-class instrumental quality)
		if sunoKey := os.Getenv("SUNO_API_KEY"); sunoKey != "" {
			sunoStyle := "Instrumental"
			sunoTitle := "Instrumental Track"
			if env.Extra != nil {
				if s, ok := env.Extra["genre"].(string); ok && s != "" {
					sunoStyle = s + " Instrumental"
				}
			}
			audioURL1, audioURL2, err := o.callSunoMusic(ctx, sunoKey, prompt, sunoStyle, sunoTitle, "", true)
			if err == nil {
				return &studioProviderResult{OutputURL: audioURL1, OutputURL2: audioURL2, Provider: "suno-ai", CostMicros: 50000}, nil
			}
			log.Printf("[AIStudio] Suno failed for instrumental: %v — trying ElevenLabs", err)
		}
		// Tier 2: ElevenLabs Music direct API
		if el11Key := os.Getenv("ELEVENLABS_API_KEY"); el11Key != "" {
			audioURL, err := o.callElevenLabsMusic(ctx, el11Key, prompt)
			if err == nil {
				return &studioProviderResult{OutputURL: audioURL, Provider: "elevenlabs-music", CostMicros: 45000}, nil
			}
			log.Printf("[AIStudio] ElevenLabs Music failed for instrumental: %v — trying Pollinations", err)
		}
		// Tier 3: Pollinations ElevenMusic (kept as secondary in case it recovers)
		if audioURL, err := o.callPollinationsElevenMusic(ctx, prompt, true); err == nil {
			return &studioProviderResult{OutputURL: audioURL, Provider: "pollinations/elevenmusic-instrumental", CostMicros: 100000}, nil
		}
		// Tier 4: HuggingFace MusicGen-small (FREE — uses HF_TOKEN already required for image gen)
		if hfKey := os.Getenv("HF_TOKEN"); hfKey != "" {
			if audioURL, err := o.callHFMusicGen(ctx, hfKey, prompt+" instrumental only, no vocals", 30); err == nil {
				return &studioProviderResult{OutputURL: audioURL, Provider: "huggingface/musicgen-small", CostMicros: 0}, nil
			}
			log.Printf("[AIStudio] HF MusicGen failed for instrumental: skipping")
		}
		return nil, fmt.Errorf("instrumental: all providers failed — configure SUNO_API_KEY, ELEVENLABS_API_KEY, POLLINATIONS_SECRET_KEY, or HF_TOKEN")

	case "jingle", "my-marketing-jingle":
		// Tier 1: Suno AI (best for branded jingles with lyrics)
		if sunoKey := os.Getenv("SUNO_API_KEY"); sunoKey != "" {
			sunoStyle := "Jingle, Catchy, Upbeat"
			sunoTitle := "Brand Jingle"
			if env.Extra != nil {
				if s, ok := env.Extra["genre"].(string); ok && s != "" {
					sunoStyle = s + ", Jingle, Catchy"
				}
			}
			audioURL1, audioURL2, err := o.callSunoMusic(ctx, sunoKey, prompt, sunoStyle, sunoTitle, "", false)
			if err == nil {
				return &studioProviderResult{OutputURL: audioURL1, OutputURL2: audioURL2, Provider: "suno-ai", CostMicros: 50000}, nil
			}
			log.Printf("[AIStudio] Suno failed for jingle: %v — trying ElevenLabs", err)
		}
		// Tier 2: ElevenLabs Music
		if el11Key := os.Getenv("ELEVENLABS_API_KEY"); el11Key != "" {
			audioURL, err := o.callElevenLabsMusic(ctx, el11Key, prompt)
			if err == nil {
				return &studioProviderResult{OutputURL: audioURL, Provider: "elevenlabs-music", CostMicros: 45000}, nil
			}
			log.Printf("[AIStudio] ElevenLabs Music failed for jingle: %v — trying Pollinations TTS", err)
		}
		// Tier 3: Pollinations TTS with upbeat voice for jingle-style output
		if audioURL, err := o.callPollinationsTTS(ctx, prompt, "onyx"); err == nil {
			return &studioProviderResult{OutputURL: audioURL, Provider: "pollinations/tts-jingle", CostMicros: 0}, nil
		}
		return nil, fmt.Errorf("jingle generation failed: all providers unavailable")

	default: // bg-music
		// ── DB-first ────────────────────────────────────────────────────────────────────────
		dbIn := providerInput{Prompt: prompt, Instrumental: true, DurationSecs: 30}
		if url, _, cost, usedSlug, err := o.runProviderChain(ctx, entities.ProviderCategoryMusic, dbIn); err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "db/" + usedSlug, CostMicros: cost}, nil
		}
		// ── Hardcoded fallback chain ─────────────────────────────────────────────────────
		// NOTE: Pollinations elevenmusic is currently OFF (45.9% success).
		// Reordered: ElevenLabs direct → Mubert → Pollinations ElevenMusic (as last resort)
		// Primary: ElevenLabs direct (reliable, uses existing ELEVENLABS_API_KEY)
		if el11Key := os.Getenv("ELEVENLABS_API_KEY"); el11Key != "" {
			audioURL, err := o.callElevenLabsMusic(ctx, el11Key, prompt)
			if err == nil {
				return &studioProviderResult{OutputURL: audioURL, Provider: "elevenlabs-sound", CostMicros: 500}, nil
			}
			log.Printf("[AIStudio] ElevenLabs Music failed for bg-music: %v — trying Mubert", err)
		}
		// Secondary: Mubert (royalty-free, text-to-music, paid plan)
		if mubertKey := os.Getenv("MUBERT_API_KEY"); mubertKey != "" {
			audioURL, err := o.callMubert(ctx, mubertKey, prompt, 30)
			if err == nil {
				return &studioProviderResult{OutputURL: audioURL, Provider: "mubert", CostMicros: 0}, nil
			}
			log.Printf("[AIStudio] Mubert failed: %v — trying Pollinations ElevenMusic", err)
		}
		// Tertiary: Pollinations ElevenMusic (currently OFF — kept as last resort in case it recovers)
		if sk := os.Getenv("POLLINATIONS_SECRET_KEY"); sk != "" {
			audioURL, err := o.callPollinationsElevenMusic(ctx, prompt, true)
			if err == nil {
				return &studioProviderResult{OutputURL: audioURL, Provider: "pollinations/elevenmusic", CostMicros: 500}, nil
			}
			log.Printf("[AIStudio] Pollinations ElevenMusic also failed for bg-music: %v", err)
		}
		// Last resort: HuggingFace MusicGen-small (FREE — uses HF_TOKEN already required for image gen)
		if hfKey := os.Getenv("HF_TOKEN"); hfKey != "" {
			if audioURL, err := o.callHFMusicGen(ctx, hfKey, prompt, 30); err == nil {
				return &studioProviderResult{OutputURL: audioURL, Provider: "huggingface/musicgen-small", CostMicros: 0}, nil
			}
			log.Printf("[AIStudio] HF MusicGen also failed for bg-music")
		}
		return nil, fmt.Errorf("background music unavailable: configure ELEVENLABS_API_KEY, MUBERT_API_KEY, POLLINATIONS_SECRET_KEY, or HF_TOKEN")
	}
}

// ─── Composite dispatch (podcast, video-jingle) ───────────────────────────────

func (o *AIStudioOrchestrator) dispatchComposite(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	switch slug {
	case "podcast", "my-podcast":
		return o.assemblePodcast(ctx, env.Prompt)
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
	// ── Step 1: Generate jingle audio ────────────────────────────────────────
	// Use the music style from extra_params if provided, else derive from prompt
	musicStyle := env.Extra["music_style"]
	if musicStyle == "" {
		musicStyle = "upbeat, catchy, commercial jingle"
	}
	musicPrompt := fmt.Sprintf("Short 15-second jingle: %s. Style: %s. Energetic, memorable, brand-friendly.", env.Prompt, musicStyle)
	// Try ElevenLabs Music first, then Mubert, then Pollinations ElevenMusic
	var musicURL string
	if el11Key := os.Getenv("ELEVENLABS_API_KEY"); el11Key != "" {
		if url, err := o.callElevenLabsMusic(ctx, el11Key, musicPrompt); err == nil {
			musicURL = url
		} else {
			log.Printf("[AIStudio] video-jingle: ElevenLabs music failed: %v", err)
		}
	}
	if musicURL == "" {
		if mubertKey := os.Getenv("MUBERT_API_KEY"); mubertKey != "" {
			if url, err := o.callMubert(ctx, mubertKey, musicPrompt, 15); err == nil {
				musicURL = url
			} else {
				log.Printf("[AIStudio] video-jingle: Mubert failed: %v", err)
			}
		}
	}
	if musicURL == "" {
		// Pollinations ElevenMusic fallback
		polKey := os.Getenv("POLLINATIONS_SECRET_KEY")
		if polKey != "" {
			musicAPIURL := fmt.Sprintf("https://gen.pollinations.ai/audio/%s?model=elevenmusic&duration=15",
				url.PathEscape(musicPrompt))
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, musicAPIURL, nil)
			req.Header.Set("Authorization", "Bearer "+polKey)
			if resp, err := o.httpClient.Do(req); err == nil {
				defer func() { _ = resp.Body.Close() }()
				if resp.StatusCode == http.StatusOK {
					raw, _ := io.ReadAll(resp.Body)
					if len(raw) > 1000 {
						key := fmt.Sprintf("studio/audio/jingle_%d.mp3", time.Now().UnixNano())
						musicURL = o.uploadOrDataURI(ctx, raw, "audio/mpeg", key)
					}
				}
			}
		}
	}
	// ── Step 2: Generate video clip ──────────────────────────────────────────
	videoPrompt := o.enhanceVideoPrompt(ctx, env.Prompt)
	var videoURL string
	// If user uploaded an image, animate it; otherwise text-to-video
	if env.ImageURL != "" {
		// audio=true: video-jingle always wants sound (the music track is merged in later)
		if url, err := o.callPollinationsVideoModel(ctx, "wan-fast", env.ImageURL, videoPrompt, 180, env.AspectRatio, "15", "true"); err == nil {
			videoURL = url
		} else {
			log.Printf("[AIStudio] video-jingle: wan-fast image-to-video failed: %v", err)
		}
	}
	if videoURL == "" {
		if url, err := o.callPollinationsVideoModel(ctx, "wan-fast", "", videoPrompt, 180, env.AspectRatio, "15", "true"); err == nil {
			videoURL = url
		} else {
			log.Printf("[AIStudio] video-jingle: wan-fast text-to-video failed: %v", err)
		}
	}
	if videoURL == "" {
		if url, err := o.callPollinationsVideoModel(ctx, "p-video", "", videoPrompt, 180, env.AspectRatio, "15", "true"); err == nil {
			videoURL = url
		} else {
			log.Printf("[AIStudio] video-jingle: p-video fallback failed: %v", err)
		}
	}
	// ── Return composite result ───────────────────────────────────────────────
	if videoURL == "" && musicURL == "" {
		return nil, fmt.Errorf("video-jingle: all providers failed")
	}
	outText := ""
	if musicURL == "" {
		outText = "Note: Music generation failed — video only."
	} else if videoURL == "" {
		outText = "Note: Video generation failed — music track only."
	}
	return &studioProviderResult{
		OutputURL:  videoURL,
		OutputURL2: musicURL,
		OutputText: outText,
		Provider:   "composite/video-jingle",
		CostMicros: 0,
	}, nil
}

func (o *AIStudioOrchestrator) assemblePodcast(ctx context.Context, topic string) (*studioProviderResult, error) {
	// Step 1: Generate podcast script via Gemini
	scriptPrompt := fmt.Sprintf(`Create a podcast script with two hosts (Nexus and Ade) discussing: %s

Format:
NEXUS: [intro greeting, introduce topic with a compelling hook]
ADE: [react with curiosity, add a relatable angle or question]
NEXUS: [key point 1 with clear explanation and example]
ADE: [follow-up question or real-world application]
NEXUS: [key point 2 with deeper insight]
ADE: [personal perspective or practical takeaway]
NEXUS: [key point 3 + actionable advice]
ADE: [closing thoughts and reflection]
NEXUS: [outro, mention Loyalty Nexus]

Make it conversational, engaging, and genuinely educational. Total length: 400-600 words.`, topic)

	podcastSys := "You are a talented podcast script writer and storyteller. " +
		"You write in a warm, conversational, and engaging style that feels natural when spoken aloud. " +
		"Nexus is the knowledgeable, enthusiastic host. Ade is the relatable, curious co-host who asks great questions. " +
		"Write in clear, accessible English that any global listener can enjoy. When the topic has Nigerian or African relevance, naturally weave in local context and examples. " +
		"Make the content educational, entertaining, and genuinely useful."
	script, err := o.callGeminiFlash(ctx, podcastSys, scriptPrompt)
	if err != nil {
		// Fallback to Groq
		script, err = o.callGroqLlama4(ctx, podcastSys, scriptPrompt)
		if err != nil {
			return nil, fmt.Errorf("podcast script generation failed: %w", err)
		}
	}

	// Step 2: Narrate the script (Google Cloud TTS preferred)
	narrationResult, err := o.dispatchTTS(ctx, script)
	if err != nil {
		// Return text-only if TTS fails — still useful
		log.Printf("[AIStudio] Podcast TTS failed, returning script only: %v", err)
		return &studioProviderResult{
			OutputText: script,
			Provider:   "gemini-script-only",
			CostMicros: 0,
		}, nil
	}

	return &studioProviderResult{
		OutputURL:  narrationResult.OutputURL,
		OutputText: script,
		Provider:   "gemini+" + narrationResult.Provider,
		CostMicros: narrationResult.CostMicros,
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
			"temperature":     0.7,
			"maxOutputTokens": 4096,
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
		"max_tokens":  4096,
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
		"max_tokens":  4096,
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
		return "", fmt.Errorf("API %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
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
		return "", fmt.Errorf("parse: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("API error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}
	return parsed.Choices[0].Message.Content, nil
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
			"num_inference_steps": 4,
			"guidance_scale":      0,
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
		"prompt":        prompt,
		"image_size":    "square_hd",
		"num_images":    1,
		"output_format": "jpeg",
		"num_inference_steps": 28,
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
	if numImages < 1 {
		numImages = 1
	}
	if numImages > 4 {
		numImages = 4
	}
	payload := map[string]interface{}{
		"prompt":                prompt,
		"num_images":            numImages,
		"output_format":         "jpeg",
		"safety_tolerance":      "2",
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
		"https://fal.run/fal-ai/flux-pro/v1.1-ultra", bytes.NewReader(body))
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
	payload := map[string]interface{}{
		"prompt":        instruction,
		"image_url":     imageURL,
		"num_images":    1,
		"output_format": "jpeg",
		"guidance_scale": 3.5,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://fal.run/fal-ai/flux-pro/kontext", bytes.NewReader(body))
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
		Image struct{ URL string `json:"url"` } `json:"image"`
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

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("FAL video %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	var parsed struct {
		Video struct{ URL string `json:"url"` } `json:"video"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || parsed.Video.URL == "" {
		return "", fmt.Errorf("FAL video parse failed: %s", truncateStr(string(raw), 200))
	}
	return parsed.Video.URL, nil
}

// callFALMultiImageVideo calls the FAL Kling v1.6 multi-image-to-video endpoint.
// Accepts 2-4 image URLs and a story prompt, returns a video URL.
func (o *AIStudioOrchestrator) callFALMultiImageVideo(ctx context.Context, falKey string, imageURLs []string, prompt string, env promptEnvelope) (string, error) {
	// Upgraded from v1.6-standard to v2.6-pro for native audio generation support
	const model = "fal-ai/kling-video/v2.6/pro/multi-image-to-video"

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
		if env.Extra != nil {
			key := fmt.Sprintf("scene_%d_caption", i+1)
			if cap, ok := env.Extra[key].(string); ok && cap != "" {
				entry.Caption = cap
			}
		}
		images = append(images, entry)
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

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("FAL multi-image video %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	var parsed struct {
		Video struct{ URL string `json:"url"` } `json:"video"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || parsed.Video.URL == "" {
		return "", fmt.Errorf("FAL multi-image video parse failed: %s", truncateStr(string(raw), 200))
	}
	return parsed.Video.URL, nil
}

// callGoogleCloudTTS calls Google Cloud Text-to-Speech API.
func (o *AIStudioOrchestrator) callGoogleCloudTTS(ctx context.Context, apiKey, text string) (string, error) {
	url := fmt.Sprintf("https://texttospeech.googleapis.com/v1/text:synthesize?key=%s", apiKey)
	payload := map[string]interface{}{
		"input": map[string]string{"text": text},
		"voice": map[string]interface{}{
			"languageCode": "en-NG", // Nigerian English
			"name":         "en-GB-Neural2-A",
			"ssmlGender":   "NEUTRAL",
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
	return o.callPollinationsImageWithSeed(ctx, prompt, 0, aspectRatio...)
}

// callPollinationsImageWithSeed is like callPollinationsImage but accepts an explicit seed.
// Pass seed=0 to use a random seed (default behaviour).
func (o *AIStudioOrchestrator) callPollinationsImageWithSeed(ctx context.Context, prompt string, userSeed int64, aspectRatio ...string) (string, error) {
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
	apiURL := fmt.Sprintf(
		"https://gen.pollinations.ai/image/%s?model=flux&width=%d&height=%d&nologo=true&seed=%d&enhance=false",
		encoded, width, height, seed,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "NexusAI/1.0")
	sk := os.Getenv("POLLINATIONS_SECRET_KEY")
	if sk == "" {
		return "", fmt.Errorf("POLLINATIONS_SECRET_KEY not configured (required since 2026-03-26)")
	}
	req.Header.Set("Authorization", "Bearer "+sk)

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
		return "", fmt.Errorf("pollinations image: response too small (%d bytes)", len(imgBytes))
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
	publicURL, err := o.storage.Upload(ctx, fileName, imgBytes, ct)
	if err != nil {
		encoded64 := base64.StdEncoding.EncodeToString(imgBytes)
		return "data:" + ct + ";base64," + encoded64, nil
	}
	return publicURL, nil
}

// callPollinationsTTS generates speech using Pollinations TTS (OpenAI-compatible).
// sk_ key is now REQUIRED (anonymous access removed 2026-03-26).
// Endpoint: POST https://gen.pollinations.ai/v1/audio/speech
func (o *AIStudioOrchestrator) callPollinationsTTS(ctx context.Context, text, voice string) (string, error) {
	if voice == "" {
		voice = "nova" // natural, clear English voice
	}
	payload := map[string]interface{}{
		"model": "tts-1",
		"input": text,
		"voice": voice,
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

	fileName := fmt.Sprintf("studio/narrate/pollinations_%d.mp3", time.Now().UnixNano())
	publicURL, err := o.storage.Upload(ctx, fileName, audioBytes, "audio/mpeg")
	if err != nil {
		encoded64 := base64.StdEncoding.EncodeToString(audioBytes)
		return "data:audio/mpeg;base64," + encoded64, nil
	}
	return publicURL, nil
}

// callPollinationsTTSWithSpeed is like callPollinationsTTS but forwards the speed parameter.
// Speed 0.25–4.0; 1.0 = normal. Falls back to callPollinationsTTS if speed is default.
// callPollinationsTTSFull generates speech with speed, format, and language support.
func (o *AIStudioOrchestrator) callPollinationsTTSFull(ctx context.Context, text, voice string, speed float64, format, lang string) (string, error) {
	sk := os.Getenv("POLLINATIONS_SECRET_KEY")
	if sk == "" {
		return "", fmt.Errorf("POLLINATIONS_SECRET_KEY not configured")
	}
	if voice == "" {
		voice = "nova"
	}
	if format == "" {
		format = "mp3"
	}
	payload := map[string]interface{}{
		"model":           "tts-1",
		"input":           text,
		"voice":           voice,
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
	req.Header.Set("Authorization", "Bearer "+sk)
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
	mimeType := "audio/mpeg"
	ext := "mp3"
	switch format {
	case "wav":
		mimeType, ext = "audio/wav", "wav"
	case "opus":
		mimeType, ext = "audio/ogg", "opus"
	case "aac":
		mimeType, ext = "audio/aac", "aac"
	case "flac":
		mimeType, ext = "audio/flac", "flac"
	}
	fileName := fmt.Sprintf("studio/narrate/pollinations_%d.%s", time.Now().UnixNano(), ext)
	publicURL, err := o.storage.Upload(ctx, fileName, audioBytes, mimeType)
	if err != nil {
		encoded64 := base64.StdEncoding.EncodeToString(audioBytes)
		return "data:" + mimeType + ";base64," + encoded64, nil
	}
	return publicURL, nil
}

// callPollinationsVideo generates a short video using wan-fast (FREE).
// Pollinations video pricing (2026-03-26):
//   FREE: wan-fast (Wan 2.2), ltx-2 (LTX-2)
//   PAID: seedance, seedance-pro, veo, wan
// Using wan-fast as the default free option.
func (o *AIStudioOrchestrator) callPollinationsVideo(ctx context.Context, imageURL, prompt string) (string, error) { //nolint:unused
	return o.callPollinationsVideoModel(ctx, "wan-fast", imageURL, prompt, 180)
}

// callPollinationsVideoModel is the shared GET-based video caller for any video model.
func (o *AIStudioOrchestrator) callPollinationsVideoModel(ctx context.Context, model, imageURL, prompt string, timeoutSecs int, opts ...string) (string, error) {
	sk := os.Getenv("POLLINATIONS_SECRET_KEY")
	if sk == "" {
		return "", fmt.Errorf("POLLINATIONS_SECRET_KEY not configured")
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
	req.Header.Set("Authorization", "Bearer "+sk)
	req.Header.Set("User-Agent", "NexusAI/1.0")

	vidCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSecs)*time.Second)
	defer cancel()
	req = req.WithContext(vidCtx)

	resp, err := o.httpClient.Do(req)
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
			"pat":        apiKey,
			"prompt":     prompt,
			"mode":       "track",
			"duration":   durationSecs,
			"format":     "mp3",
			"bitrate":    128,
			"intensity":  "medium",
			"copyright":  true,
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
		Status int    `json:"status"` // 1 = success, 0 = error
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
				TaskID string `json:"taskId"`
				Status string `json:"status"`
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
	default: // image-analyser
		// Frontend sends image_url or puts imageURL in prompt for legacy compatibility
		imageURL = env.ImageURL
		if imageURL == "" {
			imageURL = env.Prompt
		}
		question = ""
	}

	// ── DB-first ─────────────────────────────────────────────────────────────
	vIn := providerInput{ImageURL: imageURL, UserPrompt: question}
	if _, text, cost, usedSlug, err := o.runProviderChain(ctx, entities.ProviderCategoryVision, vIn); err == nil {
		return &studioProviderResult{OutputText: text, Provider: "db/" + usedSlug, CostMicros: cost}, nil
	}

	// ── Hardcoded fallback chain ──────────────────────────────────────────────
	// Primary: Pollinations Vision (OpenAI-compatible multimodal)
	text, err := o.callPollinationsVision(ctx, imageURL, question)
	if err == nil {
		return &studioProviderResult{OutputText: text, Provider: "pollinations/vision", CostMicros: 0}, nil
	}
	log.Printf("[AIStudio] Pollinations Vision failed for %s: %v — falling back to Gemini", slug, err)

	// Fallback: Gemini Flash with URL in prompt
	fallbackQ := question
	if fallbackQ == "" {
		fallbackQ = fmt.Sprintf("Describe this image: %s", imageURL)
	} else {
		fallbackQ = fmt.Sprintf("Regarding this image at %s — %s", imageURL, question)
	}
	visionSys := "You are Nexus Vision, an expert image analyst with deep knowledge of visual content, photography, design, art, and global cultural contexts. " +
		"Analyse images with precision and depth. Describe what you see comprehensively: objects, people, text, colours, composition, mood, and context. " +
		"For documents or text in images, extract and transcribe the text accurately. " +
		"For products or items, identify them and provide relevant information including brand, model, or category. " +
		"For scenes or places, identify the location type, architectural style, and cultural context where possible. " +
		"For artworks, identify the style, period, and likely influence. " +
		"Always structure your response clearly and provide genuinely useful, actionable insights."
	text, err = o.callGeminiFlash(ctx, visionSys, fallbackQ)
	if err == nil {
		return &studioProviderResult{OutputText: text, Provider: "gemini-flash/vision", CostMicros: 0}, nil
	}
	return nil, fmt.Errorf("vision analysis failed: all providers unavailable")
}

// ─── NEW: dispatchTranscribeAfrican ──────────────────────────────────────────

func (o *AIStudioOrchestrator) dispatchTranscribeAfrican(ctx context.Context, env promptEnvelope) (*studioProviderResult, error) {
	// Frontend sends: { prompt: audioURL, language: "yo" } via buildEnrichedPrompt
	audioURL := env.Prompt
	lang := strings.ToLower(strings.TrimSpace(env.Language))
	if lang == "" {
		lang = "en"
	}
	// Validate language
	validLangs := map[string]bool{"yo": true, "ha": true, "ig": true, "en": true, "fr": true}
	if !validLangs[lang] {
		lang = "en"
	}

	// Primary: Pollinations Whisper with African language selector
	if sk := os.Getenv("POLLINATIONS_SECRET_KEY"); sk != "" {
		text, err := o.callPollinationsWhisperAfrican(ctx, audioURL, lang)
		if err == nil {
			return &studioProviderResult{OutputText: text, Provider: "pollinations/whisper-african", CostMicros: 0}, nil
		}
		log.Printf("[AIStudio] Pollinations Whisper African failed: %v — falling back to Groq", err)
	}

	// Fallback: Groq Whisper
	if groqKey := os.Getenv("GROQ_API_KEY"); groqKey != "" {
		text, err := o.callGroqWhisper(ctx, groqKey, audioURL)
		if err == nil {
			return &studioProviderResult{OutputText: text, Provider: "groq/whisper-large-v3", CostMicros: 10}, nil
		}
		log.Printf("[AIStudio] Groq Whisper fallback failed: %v", err)
	}

	return nil, fmt.Errorf("transcribe-african: all providers unavailable — configure POLLINATIONS_SECRET_KEY or GROQ_API_KEY")
}

// ─── NEW: dispatchNarratorPro ─────────────────────────────────────────────────

func (o *AIStudioOrchestrator) dispatchNarratorPro(ctx context.Context, env promptEnvelope) (*studioProviderResult, error) {
	// Frontend sends: { prompt: text, voice_id: "coral" } via buildEnrichedPrompt
	validVoices := map[string]bool{
		"alloy": true, "echo": true, "fable": true, "onyx": true, "nova": true,
		"shimmer": true, "coral": true, "verse": true, "ballad": true, "ash": true,
		"sage": true, "amuch": true, "dan": true,
	}
	voice := strings.ToLower(strings.TrimSpace(env.VoiceID))
	if voice == "" || !validVoices[voice] {
		voice = "nova" // safe default
	}
	text := env.Prompt

	// Read speed and format from extra_params (sent by VoiceStudio)
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
	// Pass language for multilingual TTS (e.g. "fr", "es", "de")
	lang := strings.ToLower(strings.TrimSpace(env.Language))
	// Use callPollinationsTTSFull — supports speed, format, and language
	audioURL, err := o.callPollinationsTTSFull(ctx, text, voice, speed, audioFormat, lang)
	if err == nil {
		return &studioProviderResult{OutputURL: audioURL, Provider: "pollinations/tts-" + voice, CostMicros: 0}, nil
	}
	log.Printf("[AIStudio] Pollinations TTS Pro failed: %v", err)

	// Fallback: Google Cloud TTS
	if gcpKey := os.Getenv("GOOGLE_CLOUD_TTS_KEY"); gcpKey != "" {
		audioURL, err = o.callGoogleCloudTTS(ctx, gcpKey, text)
		if err == nil {
			return &studioProviderResult{OutputURL: audioURL, Provider: "google-cloud-tts", CostMicros: 0}, nil
		}
	}
	return nil, fmt.Errorf("narrate-pro: all TTS providers failed")
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

// callPollinationsGPTImage generates a premium image via Pollinations (gptimage / gptimage-large / seedream).
func (o *AIStudioOrchestrator) callPollinationsGPTImage(ctx context.Context, prompt, model string, opts ...string) (string, error) {
	sk := os.Getenv("POLLINATIONS_SECRET_KEY")
	if sk == "" {
		return "", fmt.Errorf("POLLINATIONS_SECRET_KEY not configured")
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
		"model":   model,
		"prompt":  prompt,
		"n":       1,
		"size":    gptSize,
		"quality": gptQuality,
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
	req.Header.Set("Authorization", "Bearer "+sk)
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
	if item.URL != "" {
		return item.URL, nil
	}
	if item.B64JSON != "" {
		imgBytes, err := base64.StdEncoding.DecodeString(item.B64JSON)
		if err != nil {
			return "", fmt.Errorf("pollinations GPTImage base64 decode: %w", err)
		}
		key := fmt.Sprintf("studio/ai-photo/%s_%d.png", model, time.Now().UnixNano())
		return o.uploadOrDataURI(ctx, imgBytes, "image/png", key), nil
	}
	return "", fmt.Errorf("pollinations GPTImage: no url or b64_json in response")
}

// callPollinationsKontextAlt performs image-to-image editing via Pollinations p-image-edit (Pruna).
// This is the fallback for when kontext is OFF/degraded. Uses the same edits endpoint but model=p-image-edit.
func (o *AIStudioOrchestrator) callPollinationsKontextAlt(ctx context.Context, imageURL, instruction string) (string, error) {
	sk := os.Getenv("POLLINATIONS_SECRET_KEY")
	if sk == "" {
		return "", fmt.Errorf("POLLINATIONS_SECRET_KEY not configured")
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
	_ = mw.WriteField("model", "p-image-edit")
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
	req.Header.Set("Authorization", "Bearer "+sk)
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
func (o *AIStudioOrchestrator) callPollinationsElevenMusic(ctx context.Context, prompt string, instrumental bool) (string, error) {
	sk := os.Getenv("POLLINATIONS_SECRET_KEY")
	if sk == "" {
		return "", fmt.Errorf("POLLINATIONS_SECRET_KEY not configured")
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
	req.Header.Set("Authorization", "Bearer "+sk)
	req.Header.Set("User-Agent", "NexusAI/1.0")

	// Music generation can take up to 3 minutes
	musicCtx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()
	req = req.WithContext(musicCtx)

	resp, err := o.httpClient.Do(req)
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
	return o.callPollinationsVideoModel(ctx, "veo", "", prompt, 180, aspectRatio, "5", audioOpt)
}

// ─── S3 upload helper ─────────────────────────────────────────────────────────

// uploadOrDataURI uploads binary data via the configured AssetStorage backend
// (S3, GCS, or local). Falls back to a base64 data URI only if the storage
// backend itself returns an error (e.g. no credentials in dev mode).
func (o *AIStudioOrchestrator) uploadOrDataURI(ctx context.Context, data []byte, contentType, key string) string {
	url, err := o.storage.Upload(ctx, key, data, contentType)
	if err != nil {
		log.Printf("[AIStudio] asset upload failed for %s (backend=%s): %v — using data URI",
			key, o.storage.Provider(), err)
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

	geminiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=%s", apiKey)
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
