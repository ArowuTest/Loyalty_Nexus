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
	"video-jingle":   catVideo,
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
}

// ─── Provider result ──────────────────────────────────────────────────────────

type studioProviderResult struct {
	OutputURL  string // CDN URL or data URI for binary outputs
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
	return &AIStudioOrchestrator{
		cfg:        cfg,
		studioRepo: studioRepo,
		studioSvc:  studioSvc,
		userRepo:   userRepo,
		storage:    storage,
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
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
		base := "You are Nexus AI, a helpful assistant for Nigerian users. Be clear, practical, and culturally aware."
		fallbackText, fErr := o.callGeminiFlash(ctx, base, fmt.Sprintf("(Search unavailable) Answer as best you can: %s", prompt))
		if fErr == nil {
			return &studioProviderResult{OutputText: fallbackText, Provider: "gemini-flash/nosearch", CostMicros: 0}, nil
		}
		return nil, fmt.Errorf("web-search-ai: all providers failed: %v / %v", err, fErr)
	}

	// code-helper: primary via Pollinations Qwen3-Coder, fallback to Gemini Flash
	if slug == "code-helper" {
		codeSys := "You are an expert programmer. Write clean, well-commented code. Explain your solution briefly."
		text, err := o.callPollinationsQwenCoder(ctx, codeSys, prompt)
		if err == nil {
			return &studioProviderResult{OutputText: text, Provider: "pollinations/qwen-coder", CostMicros: 0}, nil
		}
		log.Printf("[AIStudio] Pollinations Qwen-Coder failed: %v — falling back", err)
		base := "You are an expert programmer. Write clean, well-commented code. Explain your solution briefly."
		fallbackText, fErr := o.callGeminiFlash(ctx, base, prompt)
		if fErr == nil {
			return &studioProviderResult{OutputText: fallbackText, Provider: "gemini-flash/code", CostMicros: 0}, nil
		}
		return nil, fmt.Errorf("code-helper: all providers failed: %v / %v", err, fErr)
	}

	systemPrompt, userPrompt := buildTextPrompts(slug, prompt)

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
	base := "You are Nexus AI, a helpful assistant for Nigerian users. Be clear, practical, and culturally aware."
	switch slug {
	case "web-search-ai":
		return base, fmt.Sprintf("Search the web and answer: %s", input)
	case "code-helper":
		return "You are an expert programmer. Write clean, well-commented code. Explain your solution briefly.",
			fmt.Sprintf("Help me with the following coding task: %s", input)
	case "study-guide":
		return base, fmt.Sprintf(
			"Create a comprehensive study guide for: %s\n\nInclude:\n- Key concepts with clear definitions\n- Real-world examples relevant to Nigerian context\n- Practice questions with answers\n- Quick revision summary\n\nFormat with clear headings.", input)
	case "quiz":
		return base, fmt.Sprintf(
			"Create 10 quiz questions with answers about: %s\n\nReturn as JSON array:\n[{\"question\": \"...\", \"options\": [\"A) ...\", \"B) ...\", \"C) ...\", \"D) ...\"], \"answer\": \"A\", \"explanation\": \"...\"}]\n\nReturn ONLY the JSON, no extra text.", input)
	case "mindmap":
		return base, fmt.Sprintf(
			"Create a detailed mind map for: %s\n\nReturn as JSON:\n{\"center\": \"...\", \"branches\": [{\"label\": \"...\", \"children\": [{\"label\": \"...\"}]}]}\n\nReturn ONLY the JSON.", input)
	case "research-brief":
		return base, fmt.Sprintf(
			"Write a comprehensive research brief about: %s\n\nSections:\n1. Executive Summary (3 sentences)\n2. Background & Context\n3. Key Findings (5 bullet points)\n4. Market/Industry Data (with specific figures where available)\n5. Recommendations\n6. Conclusion\n\nBe specific and data-driven.", input)
	case "slide-deck":
		return base, fmt.Sprintf(
			"Create a slide deck outline for: %s\n\nReturn as JSON:\n{\"title\": \"...\", \"slides\": [{\"number\": 1, \"title\": \"...\", \"bullets\": [\"...\"], \"speaker_notes\": \"...\"}]}\n\nCreate 10-12 slides. Return ONLY the JSON.", input)
	case "infographic":
		return base, fmt.Sprintf(
			"Create an infographic content structure for: %s\n\nReturn as JSON:\n{\"title\": \"...\", \"subtitle\": \"...\", \"sections\": [{\"heading\": \"...\", \"stats\": [{\"label\": \"...\", \"value\": \"...\"}], \"bullets\": [\"...\"]}]}\n\nReturn ONLY the JSON.", input)
	case "bizplan":
		return base, fmt.Sprintf(
			"Write a complete business plan for: %s\n\nSections:\n1. Executive Summary\n2. Company Description & Vision\n3. Market Analysis (target market, competition, Nigerian market context)\n4. Products/Services\n5. Marketing & Sales Strategy\n6. Operations Plan\n7. Financial Projections (3-year)\n8. Funding Requirements\n\nBe specific and actionable.", input)
	default:
		return base, fmt.Sprintf("Generate comprehensive, well-structured content about: %s", input)
	}
}

// ─── Image dispatch ────────────────────────────────────────────────────────────
// Handles: ai-photo, bg-remover, ai-photo-pro, ai-photo-max, ai-photo-dream, photo-editor

func (o *AIStudioOrchestrator) dispatchImage(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	prompt := env.Prompt
	switch slug {
	case "bg-remover":
		return o.dispatchBgRemover(ctx, prompt)

	case "ai-photo-pro":
		// GPT Image (gptimage model) — CostMicros: $0.02
		url, err := o.callPollinationsGPTImage(ctx, prompt, "gptimage")
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "pollinations/gptimage", CostMicros: 20000}, nil
		}
		log.Printf("[AIStudio] GPTImage failed for ai-photo-pro: %v — falling back to FLUX", err)
		url, err = o.callPollinationsImage(ctx, prompt)
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "pollinations/flux", CostMicros: 0}, nil
		}
		return nil, fmt.Errorf("ai-photo-pro: all providers failed")

	case "ai-photo-max":
		// GPT Image Large — CostMicros: $0.03
		url, err := o.callPollinationsGPTImage(ctx, prompt, "gptimage-large")
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "pollinations/gptimage-large", CostMicros: 30000}, nil
		}
		log.Printf("[AIStudio] GPTImage-large failed for ai-photo-max: %v — falling back", err)
		url, err = o.callPollinationsImage(ctx, prompt)
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "pollinations/flux", CostMicros: 0}, nil
		}
		return nil, fmt.Errorf("ai-photo-max: all providers failed")

	case "ai-photo-dream":
		// Seedream (ByteDance) — CostMicros: $0.01
		// Live model ID from GET /v1/models is "seedream5" (not "seedream")
		url, err := o.callPollinationsGPTImage(ctx, prompt, "seedream5")
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "pollinations/seedream5", CostMicros: 10000}, nil
		}
		log.Printf("[AIStudio] Seedream5 failed for ai-photo-dream: %v — falling back", err)
		url, err = o.callPollinationsImage(ctx, prompt)
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "pollinations/flux", CostMicros: 0}, nil
		}
		return nil, fmt.Errorf("ai-photo-dream: all providers failed")

	case "photo-editor":
		// Kontext image-to-image editing — CostMicros: $0.015
		// Frontend sends: { prompt: instruction, image_url: imgURL } via buildEnrichedPrompt
		imgURL := env.ImageURL
		instruction := env.Prompt
		if imgURL == "" {
			return nil, fmt.Errorf("photo-editor: image_url is required")
		}
		url, err := o.callPollinationsKontext(ctx, imgURL, instruction)
		if err != nil {
			return nil, fmt.Errorf("photo-editor requires Pollinations key: %w", err)
		}
		return &studioProviderResult{OutputURL: url, Provider: "pollinations/kontext", CostMicros: 15000}, nil

	default: // ai-photo
		// tier 1: HuggingFace FLUX.1-Schnell (free, uses HF_TOKEN)
		if hfKey := os.Getenv("HF_TOKEN"); hfKey != "" {
			url, err := o.callHFFluxSchnell(ctx, hfKey, prompt)
			if err == nil {
				return &studioProviderResult{OutputURL: url, Provider: "huggingface/flux-schnell", CostMicros: 0}, nil
			}
			log.Printf("[AIStudio] HF FLUX.1-Schnell failed: %v", err)
		}
		// tier 2: Pollinations.ai FLUX (100% free, no key required)
		if url, err := o.callPollinationsImage(ctx, prompt); err == nil {
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

func (o *AIStudioOrchestrator) dispatchVideo(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	// video-cinematic: Seedance (image + motion prompt, paid)
	// Frontend sends: { prompt: motionPrompt, image_url: imgURL } via buildEnrichedPrompt
	if slug == "video-cinematic" {
		imgURL := env.ImageURL
		motionPrompt := env.Prompt
		if imgURL == "" {
			return nil, fmt.Errorf("video-cinematic: image_url is required")
		}
		vidURL, err := o.callPollinationsSeedance(ctx, imgURL, motionPrompt)
		if err == nil {
			return &studioProviderResult{OutputURL: vidURL, Provider: "pollinations/seedance", CostMicros: 200000}, nil
		}
		log.Printf("[AIStudio] Seedance failed for video-cinematic: %v", err)
		return nil, fmt.Errorf("video-cinematic: all providers failed")
	}

	// video-veo: Google Veo text-to-video (paid, highest quality)
	if slug == "video-veo" {
		prompt := env.Prompt
		vidURL, err := o.callPollinationsVeo(ctx, prompt)
		if err == nil {
			return &studioProviderResult{OutputURL: vidURL, Provider: "pollinations/veo2", CostMicros: 400000}, nil
		}
		log.Printf("[AIStudio] Veo failed for video-veo: %v", err)
		return nil, fmt.Errorf("video-veo: all providers failed")
	}

	// Standard video slugs (animate-photo, video-premium, video-jingle)
	// Frontend sends image_url in the envelope; prompt is the motion description
	imageURL := env.ImageURL
	if imageURL == "" {
		imageURL = env.Prompt // legacy fallback: older rows stored imageURL in prompt
	}

	// Tier 1: FAL.AI (Kling v1.5 for premium, LTX for standard)
	if falKey := os.Getenv("FAL_API_KEY"); falKey != "" {
		var model string
		switch slug {
		case "video-premium":
			model = "fal-ai/kling-video/v1.5/standard/image-to-video"
		case "video-jingle":
			model = "fal-ai/kling-video/v1.5/standard/image-to-video"
		default: // animate-photo
			model = "fal-ai/ltx-video"
		}

		videoURL, err := o.callFALVideo(ctx, falKey, model, imageURL)
		if err != nil {
			// Fallback for Kling → LTX within FAL
			if slug == "video-premium" {
				log.Printf("[AIStudio] Kling failed, falling back to LTX: %v", err)
				videoURL, err = o.callFALVideo(ctx, falKey, "fal-ai/ltx-video", imageURL)
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

	// Tier 2: Pollinations.ai seedance (image-to-video, sk_ key required)
	if videoURL, err := o.callPollinationsVideo(ctx, imageURL, "animate this image with subtle cinematic motion"); err == nil {
		return &studioProviderResult{OutputURL: videoURL, Provider: "pollinations/seedance", CostMicros: 50000}, nil
	}

	return nil, fmt.Errorf("video generation unavailable: configure FAL_API_KEY for premium video")
}

// ─── Voice / Translate dispatch ───────────────────────────────────────────────

func (o *AIStudioOrchestrator) dispatchVoiceOrTranslate(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	switch slug {
	case "translate":
		return o.dispatchTranslate(ctx, env)
	case "transcribe":
		return o.dispatchTranscribe(ctx, env.Prompt) // prompt holds audioURL for transcription
	case "transcribe-african":
		return o.dispatchTranscribeAfrican(ctx, env)
	case "narrate-pro":
		return o.dispatchNarratorPro(ctx, env)
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
	prompt := fmt.Sprintf("Translate the following text to %s. Return ONLY the translation, no explanation:\n\n%s", targetLang, text)
	translated, err := o.callGeminiFlash(ctx, "You are a professional translator.", prompt)
	if err == nil {
		return &studioProviderResult{OutputText: translated, Provider: "gemini/translate", CostMicros: 0}, nil
	}

	return nil, fmt.Errorf("translation unavailable: configure GOOGLE_TRANSLATE_API_KEY")
}

func (o *AIStudioOrchestrator) dispatchTTS(ctx context.Context, text string) (*studioProviderResult, error) {
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

func (o *AIStudioOrchestrator) dispatchTranscribe(ctx context.Context, audioURL string) (*studioProviderResult, error) {
	// Primary: AssemblyAI (free $50 credit on signup)
	if aaiKey := os.Getenv("ASSEMBLY_AI_KEY"); aaiKey != "" {
		text, err := o.callAssemblyAI(ctx, aaiKey, audioURL)
		if err == nil {
			return &studioProviderResult{OutputText: text, Provider: "assemblyai", CostMicros: 25}, nil
		}
		log.Printf("[AIStudio] AssemblyAI failed: %v", err)
	}

	// Fallback: Groq Whisper-large-v3 (fast, cheap)
	if groqKey := os.Getenv("GROQ_API_KEY"); groqKey != "" {
		text, err := o.callGroqWhisper(ctx, groqKey, audioURL)
		if err == nil {
			return &studioProviderResult{OutputText: text, Provider: "groq/whisper-large-v3", CostMicros: 10}, nil
		}
		log.Printf("[AIStudio] Groq Whisper failed: %v", err)
	}

	return nil, fmt.Errorf("transcription unavailable: configure ASSEMBLY_AI_KEY or GROQ_API_KEY")
}

// ─── Music dispatch ───────────────────────────────────────────────────────────

func (o *AIStudioOrchestrator) dispatchMusic(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	prompt := env.Prompt
	switch slug {
	case "song-creator":
		// Full song with vocals via Pollinations ElevenMusic
		audioURL, err := o.callPollinationsElevenMusic(ctx, prompt, false)
		if err == nil {
			return &studioProviderResult{OutputURL: audioURL, Provider: "pollinations/elevenmusic", CostMicros: 100000}, nil
		}
		log.Printf("[AIStudio] Pollinations ElevenMusic failed for song-creator: %v — falling back", err)
		// Fallback: ElevenLabs Music
		if el11Key := os.Getenv("ELEVENLABS_API_KEY"); el11Key != "" {
			audioURL, err = o.callElevenLabsMusic(ctx, el11Key, prompt)
			if err == nil {
				return &studioProviderResult{OutputURL: audioURL, Provider: "elevenlabs-music", CostMicros: 45000}, nil
			}
		}
		return nil, fmt.Errorf("song-creator: all providers failed")

	case "instrumental":
		// Instrumental track (no vocals) via Pollinations ElevenMusic
		audioURL, err := o.callPollinationsElevenMusic(ctx, prompt, true)
		if err == nil {
			return &studioProviderResult{OutputURL: audioURL, Provider: "pollinations/elevenmusic-instrumental", CostMicros: 100000}, nil
		}
		log.Printf("[AIStudio] Pollinations ElevenMusic failed for instrumental: %v — falling back", err)
		// Fallback: ElevenLabs Music
		if el11Key := os.Getenv("ELEVENLABS_API_KEY"); el11Key != "" {
			audioURL, err = o.callElevenLabsMusic(ctx, el11Key, prompt)
			if err == nil {
				return &studioProviderResult{OutputURL: audioURL, Provider: "elevenlabs-music", CostMicros: 45000}, nil
			}
		}
		return nil, fmt.Errorf("instrumental: all providers failed")

	case "jingle":
		// Premium: ElevenLabs Music (professional quality, ~₦450/jingle)
		if el11Key := os.Getenv("ELEVENLABS_API_KEY"); el11Key != "" {
			audioURL, err := o.callElevenLabsMusic(ctx, el11Key, prompt)
			if err == nil {
				return &studioProviderResult{OutputURL: audioURL, Provider: "elevenlabs-music", CostMicros: 45000}, nil
			}
			log.Printf("[AIStudio] ElevenLabs Music failed: %v", err)
		}
		return nil, fmt.Errorf("marketing jingle requires ELEVENLABS_API_KEY")

	default: // bg-music
		// Primary: Pollinations ElevenMusic (instrumental mode — no vocals, pure background track)
		// HuggingFace MusicGen was removed from HF serverless (410 Gone) — replaced by Pollinations.
		// Pollinations ElevenMusic: GET /audio/{prompt}?model=elevenmusic&instrumental=true
		// Cost: pollen credits (~$0.005/s). sk_ key required since 2026-03-26.
		if sk := os.Getenv("POLLINATIONS_SECRET_KEY"); sk != "" {
			audioURL, err := o.callPollinationsElevenMusic(ctx, prompt, true)
			if err == nil {
				return &studioProviderResult{OutputURL: audioURL, Provider: "pollinations/elevenmusic", CostMicros: 500}, nil
			}
			log.Printf("[AIStudio] Pollinations ElevenMusic failed for bg-music: %v — trying Mubert", err)
		}
		// Secondary: Mubert (royalty-free, text-to-music, paid plan)
		if mubertKey := os.Getenv("MUBERT_API_KEY"); mubertKey != "" {
			audioURL, err := o.callMubert(ctx, mubertKey, prompt, 30)
			if err == nil {
				return &studioProviderResult{OutputURL: audioURL, Provider: "mubert", CostMicros: 0}, nil
			}
			log.Printf("[AIStudio] Mubert failed: %v — trying ElevenLabs sound", err)
		}
		// Final fallback: ElevenLabs direct (uses existing ELEVENLABS_API_KEY)
		if el11Key := os.Getenv("ELEVENLABS_API_KEY"); el11Key != "" {
			audioURL, err := o.callElevenLabsMusic(ctx, el11Key, prompt)
			if err == nil {
				return &studioProviderResult{OutputURL: audioURL, Provider: "elevenlabs-sound", CostMicros: 500}, nil
			}
		}
		return nil, fmt.Errorf("background music unavailable: configure POLLINATIONS_SECRET_KEY (primary) or MUBERT_API_KEY")
	}
}

// ─── Composite dispatch (podcast, video-jingle) ───────────────────────────────

func (o *AIStudioOrchestrator) dispatchComposite(ctx context.Context, slug string, env promptEnvelope) (*studioProviderResult, error) {
	switch slug {
	case "podcast":
		return o.assemblePodcast(ctx, env.Prompt)
	default:
		return o.dispatchText(ctx, slug, env)
	}
}

func (o *AIStudioOrchestrator) assemblePodcast(ctx context.Context, topic string) (*studioProviderResult, error) {
	// Step 1: Generate podcast script via Gemini
	scriptPrompt := fmt.Sprintf(`Create a podcast script with two hosts (Nexus and Ade) discussing: %s

Format:
NEXUS: [intro greeting, introduce topic]
ADE: [react, add context relevant to Nigeria]
NEXUS: [key point 1 with explanation]
ADE: [question or real-world Nigerian example]
NEXUS: [key point 2]
ADE: [personal story or practical application]
NEXUS: [key point 3 + actionable takeaway]
ADE: [closing thoughts]
NEXUS: [outro, mention Loyalty Nexus]

Make it conversational, engaging, and relevant to Nigerian users. Total length: 400-600 words.`, topic)

	script, err := o.callGeminiFlash(ctx,
		"You are a podcast script writer for a Nigerian audience. Create engaging, educational content.",
		scriptPrompt)
	if err != nil {
		// Fallback to Groq
		script, err = o.callGroqLlama4(ctx,
			"You are a podcast script writer for a Nigerian audience.",
			scriptPrompt)
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
		return "", fmt.Errorf("Gemini request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
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
		return "", fmt.Errorf("Gemini parse: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("Gemini API error: %s", result.Error.Message)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini: no content returned")
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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	w.Close()

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
	defer resp.Body.Close()

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
func (o *AIStudioOrchestrator) callFALVideo(ctx context.Context, falKey, model, imageURL string) (string, error) {
	payload := map[string]interface{}{
		"image_url": imageURL,
		"prompt":    "animate this photo naturally with smooth motion",
		"duration":  "5",
	}
	if strings.Contains(model, "kling") {
		payload["aspect_ratio"] = "9:16"
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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Google TTS %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	var result struct {
		AudioContent string `json:"audioContent"` // base64-encoded MP3
	}
	if err := json.Unmarshal(raw, &result); err != nil || result.AudioContent == "" {
		return "", fmt.Errorf("Google TTS parse failed")
	}

	audioData, err := base64.StdEncoding.DecodeString(result.AudioContent)
	if err != nil {
		return "", fmt.Errorf("Google TTS decode: %w", err)
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
	defer resp.Body.Close()

	audio, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ElevenLabs TTS %d: %s", resp.StatusCode, truncateStr(string(audio), 200))
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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
		pollResp.Body.Close()

		switch result.Status {
		case "completed":
			return result.Text, nil
		case "error":
			return "", fmt.Errorf("AssemblyAI error: %s", result.Error)
		}
	}
	return "", fmt.Errorf("AssemblyAI: transcription timed out after 5 minutes")
}

// callGroqWhisper uses Groq's Whisper for transcription (fast fallback).
func (o *AIStudioOrchestrator) callGroqWhisper(ctx context.Context, apiKey, audioURL string) (string, error) {
	// Groq Whisper requires multipart/form-data with a binary file upload.
	// It does NOT accept a JSON body with a "url" field.
	// Step 1: Download the audio file from the URL.
	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, audioURL, nil)
	if err != nil {
		return "", fmt.Errorf("Groq Whisper download request: %w", err)
	}
	dlResp, err := o.httpClient.Do(dlReq)
	if err != nil {
		return "", fmt.Errorf("Groq Whisper download: %w", err)
	}
	defer dlResp.Body.Close()
	if dlResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Groq Whisper: audio download failed with status %d", dlResp.StatusCode)
	}
	audioBytes, err := io.ReadAll(dlResp.Body)
	if err != nil {
		return "", fmt.Errorf("Groq Whisper: read audio bytes: %w", err)
	}

	// Step 2: POST as multipart/form-data with the file bytes.
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("model", "whisper-large-v3")
	_ = w.WriteField("language", "en")
	_ = w.WriteField("response_format", "json")
	fw, err := w.CreateFormFile("file", "audio.mp3")
	if err != nil {
		return "", fmt.Errorf("Groq Whisper form: %w", err)
	}
	if _, err = fw.Write(audioBytes); err != nil {
		return "", fmt.Errorf("Groq Whisper write form: %w", err)
	}
	w.Close()

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
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Groq Whisper %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}
	var result struct {
		Text string `json:"text"`
	}
	if decErr := json.Unmarshal(raw, &result); decErr != nil {
		return "", fmt.Errorf("Groq Whisper decode: %w", decErr)
	}
	if result.Text == "" {
		return "", fmt.Errorf("Groq Whisper: no transcription returned")
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
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Google Translate %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	var result struct {
		Data struct {
			Translations []struct {
				TranslatedText string `json:"translatedText"`
			} `json:"translations"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("Google Translate parse: %w", err)
	}
	if len(result.Data.Translations) == 0 {
		return "", fmt.Errorf("Google Translate: no translation returned")
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
	defer resp.Body.Close()

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
func (o *AIStudioOrchestrator) callPollinationsImage(ctx context.Context, prompt string) (string, error) {
	encoded := url.PathEscape(prompt)
	seed := time.Now().UnixNano() % 999983
	apiURL := fmt.Sprintf(
		"https://gen.pollinations.ai/image/%s?model=flux&width=1024&height=1024&nologo=true&seed=%d&enhance=false",
		encoded, seed,
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
		return "", fmt.Errorf("Pollinations image request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Pollinations image %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	imgBytes, err := io.ReadAll(resp.Body)
	if err != nil || len(imgBytes) < 1000 {
		return "", fmt.Errorf("Pollinations image: response too small (%d bytes)", len(imgBytes))
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
		return "", fmt.Errorf("Pollinations TTS request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Pollinations TTS %d: %s", resp.StatusCode, truncateStr(string(raw), 100))
	}

	audioBytes, err := io.ReadAll(resp.Body)
	if err != nil || len(audioBytes) < 500 {
		return "", fmt.Errorf("Pollinations TTS: response too small")
	}

	fileName := fmt.Sprintf("studio/narrate/pollinations_%d.mp3", time.Now().UnixNano())
	publicURL, err := o.storage.Upload(ctx, fileName, audioBytes, "audio/mpeg")
	if err != nil {
		encoded64 := base64.StdEncoding.EncodeToString(audioBytes)
		return "data:audio/mpeg;base64," + encoded64, nil
	}
	return publicURL, nil
}

// callPollinationsVideo generates a short video using Pollinations seedance.
// NOTE (2026-03-26): wan-fast has been REMOVED from Pollinations video models.
// Current video models: seedance (image-to-video + text-to-video), veo (text-to-video only).
// imageURL is passed as the `image` query param for image-to-video; empty = text-to-video.
func (o *AIStudioOrchestrator) callPollinationsVideo(ctx context.Context, imageURL, prompt string) (string, error) {
	return o.callPollinationsVideoModel(ctx, "seedance", imageURL, prompt, 180)
}

// callPollinationsVideoModel is the shared GET-based video caller for any video model.
func (o *AIStudioOrchestrator) callPollinationsVideoModel(ctx context.Context, model, imageURL, prompt string, timeoutSecs int) (string, error) {
	sk := os.Getenv("POLLINATIONS_SECRET_KEY")
	if sk == "" {
		return "", fmt.Errorf("POLLINATIONS_SECRET_KEY not configured")
	}

	encoded := url.PathEscape(prompt)
	apiURL := fmt.Sprintf("https://gen.pollinations.ai/image/%s?model=%s&duration=5&aspectRatio=16:9",
		encoded, model)
	if imageURL != "" {
		apiURL += "&image=" + url.QueryEscape(imageURL)
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
		return "", fmt.Errorf("Pollinations %s video request: %w", model, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Pollinations %s video %d: %s", model, resp.StatusCode, truncateStr(string(raw), 200))
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil || len(raw) < 1000 {
		return "", fmt.Errorf("Pollinations %s video: response too small (%d bytes)", model, len(raw))
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
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Mubert %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
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
		return "", fmt.Errorf("Mubert parse: %w", err)
	}
	if result.Status != 1 {
		errText := "unknown error"
		if result.Error != nil {
			errText = fmt.Sprintf("code %d: %s", result.Error.Code, result.Error.Text)
		}
		return "", fmt.Errorf("Mubert API error — %s", errText)
	}
	if len(result.Data.Tasks) == 0 || result.Data.Tasks[0].MusicURL == "" {
		return "", fmt.Errorf("Mubert: no track URL in response")
	}
	return result.Data.Tasks[0].MusicURL, nil
}

// callElevenLabsMusic calls ElevenLabs for music/sound generation.
func (o *AIStudioOrchestrator) callElevenLabsMusic(ctx context.Context, apiKey, prompt string) (string, error) {
	payload := map[string]interface{}{
		"text":     prompt,
		"duration_seconds": 30,
		"prompt_influence": 0.3,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.elevenlabs.io/v1/sound-generation", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	audio, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ElevenLabs music %d: %s", resp.StatusCode, truncateStr(string(audio), 200))
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
	text, err = o.callGeminiFlash(ctx, "You are a helpful image analysis assistant.", fallbackQ)
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

	// Use callPollinationsTTS — already implemented, uses POLLINATIONS_SECRET_KEY when set
	audioURL, err := o.callPollinationsTTS(ctx, text, voice)
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
		return "", fmt.Errorf("Pollinations chat request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Pollinations chat %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
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

// callPollinationsGPTImage generates a premium image via Pollinations (gptimage / gptimage-large / seedream).
func (o *AIStudioOrchestrator) callPollinationsGPTImage(ctx context.Context, prompt, model string) (string, error) {
	sk := os.Getenv("POLLINATIONS_SECRET_KEY")
	if sk == "" {
		return "", fmt.Errorf("POLLINATIONS_SECRET_KEY not configured")
	}
	payload := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"n":      1,
		"size":   "1024x1024",
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
		return "", fmt.Errorf("Pollinations GPTImage request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Pollinations GPTImage %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}

	var parsed struct {
		Data []struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Data) == 0 {
		return "", fmt.Errorf("Pollinations GPTImage parse: empty data")
	}
	item := parsed.Data[0]
	if item.URL != "" {
		return item.URL, nil
	}
	if item.B64JSON != "" {
		imgBytes, err := base64.StdEncoding.DecodeString(item.B64JSON)
		if err != nil {
			return "", fmt.Errorf("Pollinations GPTImage base64 decode: %w", err)
		}
		key := fmt.Sprintf("studio/ai-photo/%s_%d.png", model, time.Now().UnixNano())
		return o.uploadOrDataURI(ctx, imgBytes, "image/png", key), nil
	}
	return "", fmt.Errorf("Pollinations GPTImage: no url or b64_json in response")
}

// callPollinationsKontext performs image-to-image editing via Pollinations Kontext.
// Downloads the source image, then sends it as multipart to the edits endpoint.
func (o *AIStudioOrchestrator) callPollinationsKontext(ctx context.Context, imageURL, instruction string) (string, error) {
	sk := os.Getenv("POLLINATIONS_SECRET_KEY")
	if sk == "" {
		return "", fmt.Errorf("POLLINATIONS_SECRET_KEY not configured")
	}

	// Step 1: Download source image
	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("Kontext: build download request: %w", err)
	}
	dlResp, err := o.httpClient.Do(dlReq)
	if err != nil {
		return "", fmt.Errorf("Kontext: download image: %w", err)
	}
	defer dlResp.Body.Close()
	imgBytes, err := io.ReadAll(dlResp.Body)
	if err != nil || len(imgBytes) < 500 {
		return "", fmt.Errorf("Kontext: image download failed or too small")
	}

	// Step 2: Build multipart body
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("model", "kontext")
	_ = mw.WriteField("prompt", instruction)
	fw, err := mw.CreateFormFile("image", "source.png")
	if err != nil {
		return "", err
	}
	if _, err = fw.Write(imgBytes); err != nil {
		return "", err
	}
	mw.Close()

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
		return "", fmt.Errorf("Pollinations Kontext request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Pollinations Kontext %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}

	var parsed struct {
		Data []struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Data) == 0 {
		return "", fmt.Errorf("Pollinations Kontext parse: empty data")
	}
	item := parsed.Data[0]
	if item.URL != "" {
		return item.URL, nil
	}
	if item.B64JSON != "" {
		outBytes, err := base64.StdEncoding.DecodeString(item.B64JSON)
		if err != nil {
			return "", fmt.Errorf("Pollinations Kontext b64 decode: %w", err)
		}
		key := fmt.Sprintf("studio/photo-editor/kontext_%d.png", time.Now().UnixNano())
		return o.uploadOrDataURI(ctx, outBytes, "image/png", key), nil
	}
	return "", fmt.Errorf("Pollinations Kontext: no url or b64_json in response")
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
		return "", fmt.Errorf("Whisper African: build download request: %w", err)
	}
	dlResp, err := o.httpClient.Do(dlReq)
	if err != nil {
		return "", fmt.Errorf("Whisper African: download audio: %w", err)
	}
	defer dlResp.Body.Close()
	audioBytes, err := io.ReadAll(dlResp.Body)
	if err != nil || len(audioBytes) < 100 {
		return "", fmt.Errorf("Whisper African: audio download failed or too small")
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
	mw.Close()

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
		return "", fmt.Errorf("Pollinations Whisper African request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Pollinations Whisper African %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || result.Text == "" {
		return "", fmt.Errorf("Pollinations Whisper African: no transcription returned")
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
		return "", fmt.Errorf("Pollinations ElevenMusic request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Pollinations ElevenMusic %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}

	// GET /audio returns raw MP3 bytes directly
	raw, err := io.ReadAll(resp.Body)
	if err != nil || len(raw) < 1000 {
		return "", fmt.Errorf("Pollinations ElevenMusic: response too small (%d bytes)", len(raw))
	}

	suffix := "song"
	if instrumental {
		suffix = "instrumental"
	}
	key := fmt.Sprintf("studio/music/%s_%d.mp3", suffix, time.Now().UnixNano())
	return o.uploadOrDataURI(ctx, raw, "audio/mpeg", key), nil
}

// callPollinationsSeedance generates a cinematic image-to-video using Pollinations Seedance.
// Official documented endpoint: GET gen.pollinations.ai/image/{prompt}?model=seedance&image={srcURL}
// Paid model (~$0.20/video). Uses sk_ key via Bearer header. Timeout: 180s.
func (o *AIStudioOrchestrator) callPollinationsSeedance(ctx context.Context, imageURL, prompt string) (string, error) {
	return o.callPollinationsVideoModel(ctx, "seedance", imageURL, prompt, 180)
}

// callPollinationsVeo generates a premium text-to-video using Google Veo via Pollinations.
// Official documented endpoint: GET gen.pollinations.ai/image/{prompt}?model=veo
// Paid model (~$0.40-0.50/video). Uses sk_ key via Bearer header. Timeout: 180s.
func (o *AIStudioOrchestrator) callPollinationsVeo(ctx context.Context, prompt string) (string, error) {
	return o.callPollinationsVideoModel(ctx, "veo", "", prompt, 180)
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
	return o.studioSvc.CompleteGeneration(ctx, gen.ID, r.OutputURL, r.OutputText, r.Provider, r.CostMicros, r.DurationMs)
}

// ─── Utility ─────────────────────────────────────────────────────────────────

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
