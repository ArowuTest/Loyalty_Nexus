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
//  bg-music         Create      5 pts  HF MusicGen-small (free, uses HF_TOKEN) → Mubert → ElevenLabs sound
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
	return o.complete(ctx, gen, result)
}

// route dispatches to the correct provider chain based on slug category.
func (o *AIStudioOrchestrator) route(ctx context.Context, gen *entities.AIGeneration) (*studioProviderResult, error) {
	slug := gen.ToolSlug
	prompt := gen.Prompt

	cat, known := slugCategory[slug]
	if !known {
		return nil, fmt.Errorf("unknown tool slug %q", slug)
	}

	switch cat {
	case catText:
		return o.dispatchText(ctx, slug, prompt)
	case catImage:
		return o.dispatchImage(ctx, slug, prompt)
	case catVideo:
		return o.dispatchVideo(ctx, slug, prompt)
	case catVoice:
		return o.dispatchVoiceOrTranslate(ctx, slug, prompt)
	case catMusic:
		return o.dispatchMusic(ctx, slug, prompt)
	case catComposite:
		return o.dispatchComposite(ctx, slug, prompt)
	default:
		return nil, fmt.Errorf("unhandled category %q", cat)
	}
}

// ─── Text dispatch (study guide, quiz, mindmap, research-brief, bizplan, slide-deck, infographic) ──

func (o *AIStudioOrchestrator) dispatchText(ctx context.Context, slug, prompt string) (*studioProviderResult, error) {
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

// ─── Image dispatch (ai-photo → HF FLUX.1-Schnell primary, FAL.AI fallback; bg-remover → rembg/Photoroom) ──

func (o *AIStudioOrchestrator) dispatchImage(ctx context.Context, slug, prompt string) (*studioProviderResult, error) {
	if slug == "bg-remover" {
		return o.dispatchBgRemover(ctx, prompt)
	}
	// ai-photo: HuggingFace FLUX.1-Schnell (free) → FAL.AI FLUX-dev (paid fallback)
	if hfKey := os.Getenv("HF_TOKEN"); hfKey != "" {
		url, err := o.callHFFluxSchnell(ctx, hfKey, prompt)
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "huggingface/flux-schnell", CostMicros: 0}, nil
		}
		log.Printf("[AIStudio] HF FLUX.1-Schnell failed: %v", err)
	}

	if falKey := os.Getenv("FAL_API_KEY"); falKey != "" {
		url, err := o.callFALFlux(ctx, falKey, prompt)
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "fal.ai/flux-dev", CostMicros: 6500}, nil
		}
		log.Printf("[AIStudio] FAL FLUX failed: %v", err)
	}

	return nil, fmt.Errorf("image generation unavailable: configure HF_TOKEN or FAL_API_KEY")
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

// ─── Video dispatch (animate-photo → FAL.AI LTX basic; video-premium → FAL.AI Kling v1.5) ──

func (o *AIStudioOrchestrator) dispatchVideo(ctx context.Context, slug, imageURL string) (*studioProviderResult, error) {
	falKey := os.Getenv("FAL_API_KEY")
	if falKey == "" {
		return nil, fmt.Errorf("video generation requires FAL_API_KEY")
	}

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
		// Fallback for Kling → LTX
		if slug == "video-premium" {
			log.Printf("[AIStudio] Kling failed, falling back to LTX: %v", err)
			videoURL, err = o.callFALVideo(ctx, falKey, "fal-ai/ltx-video", imageURL)
		}
		if err != nil {
			return nil, fmt.Errorf("video generation failed: %w", err)
		}
	}

	costMicros := 14500 // ~₦145 for LTX
	if slug == "video-premium" {
		costMicros = 56000 // ~₦560 for Kling
	}
	return &studioProviderResult{OutputURL: videoURL, Provider: "fal.ai/" + model, CostMicros: costMicros}, nil
}

// ─── Voice / Translate dispatch ───────────────────────────────────────────────

func (o *AIStudioOrchestrator) dispatchVoiceOrTranslate(ctx context.Context, slug, input string) (*studioProviderResult, error) {
	switch slug {
	case "translate":
		return o.dispatchTranslate(ctx, input)
	case "transcribe":
		return o.dispatchTranscribe(ctx, input)
	default: // narrate
		return o.dispatchTTS(ctx, input)
	}
}

func (o *AIStudioOrchestrator) dispatchTranslate(ctx context.Context, input string) (*studioProviderResult, error) {
	// Parse input: expected format "LANG:text" e.g. "yo:Hello world"
	parts := strings.SplitN(input, ":", 2)
	targetLang, text := "yo", input
	if len(parts) == 2 {
		targetLang = strings.ToLower(strings.TrimSpace(parts[0]))
		text = parts[1]
	}
	// Validate supported languages
	supportedLangs := map[string]bool{"yo": true, "ha": true, "ig": true, "fr": true, "en": true}
	if !supportedLangs[targetLang] {
		targetLang = "yo" // default to Yoruba
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
			voiceID = "21m00Tcm4TlvDq8ikWAM" // Rachel
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

func (o *AIStudioOrchestrator) dispatchMusic(ctx context.Context, slug, prompt string) (*studioProviderResult, error) {
	switch slug {
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
		// Primary: HuggingFace MusicGen-small (FREE — uses existing HF_TOKEN, no extra signup)
		// Model: facebook/musicgen-small — 5-30 second ambient/background music clips
		if hfToken := os.Getenv("HF_TOKEN"); hfToken != "" {
			audioURL, err := o.callHFMusicGen(ctx, hfToken, prompt, 15)
			if err == nil {
				return &studioProviderResult{OutputURL: audioURL, Provider: "hf-musicgen", CostMicros: 0}, nil
			}
			log.Printf("[AIStudio] HF MusicGen failed: %v — trying Mubert", err)
		}
		// Secondary: Mubert (if user has configured a paid plan key)
		if mubertKey := os.Getenv("MUBERT_API_KEY"); mubertKey != "" {
			audioURL, err := o.callMubert(ctx, mubertKey, prompt, 30)
			if err == nil {
				return &studioProviderResult{OutputURL: audioURL, Provider: "mubert", CostMicros: 0}, nil
			}
			log.Printf("[AIStudio] Mubert failed: %v — trying ElevenLabs sound", err)
		}
		// Final fallback: ElevenLabs sound-generation (uses existing ELEVENLABS_API_KEY)
		if el11Key := os.Getenv("ELEVENLABS_API_KEY"); el11Key != "" {
			audioURL, err := o.callElevenLabsMusic(ctx, el11Key, prompt)
			if err == nil {
				return &studioProviderResult{OutputURL: audioURL, Provider: "elevenlabs-sound", CostMicros: 500}, nil
			}
		}
		return nil, fmt.Errorf("background music unavailable: configure HF_TOKEN (free) to enable this feature")
	}
}

// ─── Composite dispatch (podcast, video-jingle) ───────────────────────────────

func (o *AIStudioOrchestrator) dispatchComposite(ctx context.Context, slug, prompt string) (*studioProviderResult, error) {
	switch slug {
	case "podcast":
		return o.assemblePodcast(ctx, prompt)
	default:
		return o.dispatchText(ctx, slug, prompt)
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

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=%s", apiKey)
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
		"model": "llama-4-scout-17b-16e-instruct",
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
	endpoint := "https://api-inference.huggingface.co/models/" + model

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
		"https://api-inference.huggingface.co/models/suno/bark",
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
		json.NewDecoder(pollResp.Body).Decode(&result)
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
	// Groq Whisper accepts a URL for audio files
	payload := map[string]interface{}{
		"model":     "whisper-large-v3",
		"url":       audioURL,
		"language":  "en",
		"response_format": "json",
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.groq.com/openai/v1/audio/transcriptions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Text string `json:"text"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
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
	// HF Inference API for audio generation
	apiURL := "https://api-inference.huggingface.co/models/facebook/musicgen-small"
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
		Data struct {
			Tasks []struct {
				MusicURL string `json:"music_url"`
			} `json:"tasks"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || len(result.Data.Tasks) == 0 || result.Data.Tasks[0].MusicURL == "" {
		return "", fmt.Errorf("Mubert: no track URL returned — check API key and plan")
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

// encodeBase64 encodes bytes to standard base64 string.
func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
