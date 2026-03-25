package services

// ai_studio_service.go — 4-tier AI provider orchestration (spec §9 "Nexus Studio")
//
// Provider Tier Priority (spec §9.1):
//   Tier 1 (Free/Freemium) : Google Gemini Flash 2.0, Groq Llama-3.3-70B, HuggingFace
//   Tier 2 (Pay-per-use)   : FAL.AI (image/video), ElevenLabs (voice), Mubert (music)
//   Tier 3 (Overflow/paid) : DeepSeek-V3
//   Tier 4 (Degraded)      : return structured error — user told to retry
//
// Tool → provider mapping (spec §9.2):
//   text (translate, study-guide, quiz, mindmap, research-brief, bizplan, slide-deck):
//       Primary: Groq (Llama-3.3-70B) → Gemini Flash 2.0 → DeepSeek-V3
//   image (ai-photo, bg-remover):
//       Primary: FAL.AI FLUX.1-dev → HuggingFace SDXL
//   video (animate-photo, video-premium):
//       Primary: FAL.AI Kling v1.5 → FAL.AI LTX-Video
//   voice (narrate, transcribe):
//       Primary: ElevenLabs TTS → HuggingFace Bark
//   music (jingle, bg-music):
//       Primary: Mubert API → ElevenLabs sound-generation
//   composite (podcast, infographic):
//       Multi-step: LLM script + voice/image assembly
//
// Financial rule: point costs are NEVER read here — that is the service layer's job.
// This orchestrator only does I/O; all DB writes go through StudioService.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"
)

// ─── Tool category map ────────────────────────────────────────────────────────

type studioToolCat string

const (
	catText      studioToolCat = "text"
	catImage     studioToolCat = "image"
	catVideo     studioToolCat = "video"
	catVoice     studioToolCat = "voice"
	catMusic     studioToolCat = "music"
	catComposite studioToolCat = "composite"
)

var slugCategory = map[string]studioToolCat{
	"translate":      catText,
	"study-guide":    catText,
	"quiz":           catText,
	"mindmap":        catText,
	"research-brief": catText,
	"bizplan":        catText,
	"slide-deck":     catText,
	"ai-photo":       catImage,
	"bg-remover":     catImage,
	"narrate":        catVoice,
	"transcribe":     catVoice,
	"jingle":         catMusic,
	"bg-music":       catMusic,
	"animate-photo":  catVideo,
	"video-premium":  catVideo,
	"podcast":        catComposite,
	"infographic":    catComposite,
}

// ─── Provider result ──────────────────────────────────────────────────────────

type studioProviderResult struct {
	OutputURL  string // CDN / data URI
	OutputText string // for text-generation tools
	Provider   string // e.g. "groq", "fal.ai/flux"
	CostMicros int    // fractional cost in µUSD for accounting
	DurationMs int    // wall-clock time
}

// ─── AIStudioOrchestrator ─────────────────────────────────────────────────────

// AIStudioOrchestrator calls external AI provider APIs and persists results via
// StudioService (which owns the repo + ledger).  It has no direct DB access.
type AIStudioOrchestrator struct {
	cfg        *config.ConfigManager
	studioRepo repositories.StudioRepository
	studioSvc  *StudioService // for CompleteGeneration / FailGeneration
	userRepo   repositories.UserRepository
}

func NewAIStudioOrchestrator(
	cfg *config.ConfigManager,
	sr repositories.StudioRepository,
	ss *StudioService,
	ur repositories.UserRepository,
) *AIStudioOrchestrator {
	return &AIStudioOrchestrator{
		cfg:        cfg,
		studioRepo: sr,
		studioSvc:  ss,
		userRepo:   ur,
	}
}

// ─── Main dispatch ────────────────────────────────────────────────────────────

// Dispatch routes a generation job to the correct provider tier.
// Called from the async worker goroutine after the HTTP handler has returned.
func (o *AIStudioOrchestrator) Dispatch(ctx context.Context, genID uuid.UUID) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	gen, err := o.studioRepo.FindGenerationByID(ctx, genID)
	if err != nil {
		log.Printf("[AIStudio] Dispatch: gen %s not found: %v", genID, err)
		return
	}

	// Mark as processing
	if err := o.studioRepo.UpdateStatus(ctx, genID, "processing", "", ""); err != nil {
		log.Printf("[AIStudio] UpdateStatus processing: %v", err)
	}

	tool, err := o.studioRepo.FindToolByID(ctx, gen.ToolID)
	if err != nil {
		o.fail(ctx, gen, "tool configuration error: "+err.Error())
		return
	}

	// Resolve slug: prefer stored ToolSlug, fall back to normalising tool.Name
	slug := gen.ToolSlug
	if slug == "" {
		slug = normaliseSlug(tool.Name)
	}

	cat, ok := slugCategory[slug]
	if !ok {
		// Best-effort: treat as generic text tool
		cat = catText
	}

	log.Printf("[AIStudio] dispatching gen=%s tool=%s slug=%s cat=%s", genID, tool.Name, slug, cat)
	start := time.Now()

	var result *studioProviderResult

	switch cat {
	case catText:
		result, err = o.dispatchText(ctx, slug, gen.Prompt)
	case catImage:
		result, err = o.dispatchImage(ctx, slug, gen.Prompt)
	case catVoice:
		result, err = o.dispatchVoice(ctx, slug, gen.Prompt)
	case catMusic:
		result, err = o.dispatchMusic(ctx, slug, gen.Prompt)
	case catVideo:
		result, err = o.dispatchVideo(ctx, slug, gen.Prompt)
	case catComposite:
		result, err = o.dispatchComposite(ctx, slug, gen.Prompt)
	default:
		err = fmt.Errorf("unknown category %q for slug %q", cat, slug)
	}

	if err != nil {
		log.Printf("[AIStudio] gen %s failed: %v", genID, err)
		o.fail(ctx, gen, err.Error())
		return
	}

	result.DurationMs = int(time.Since(start).Milliseconds())
	o.complete(ctx, gen, result)
}

// ─── Text (Groq → Gemini → DeepSeek) ─────────────────────────────────────────

func (o *AIStudioOrchestrator) dispatchText(ctx context.Context, slug, prompt string) (*studioProviderResult, error) {
	sys := textSystemPrompt(slug)

	type llmProvider struct {
		name, key, endpoint, model string
	}
	providers := []llmProvider{
		{
			name:     "groq",
			key:      os.Getenv("GROQ_API_KEY"),
			endpoint: "https://api.groq.com/openai/v1/chat/completions",
			model:    "llama-3.3-70b-versatile",
		},
		{
			name:     "gemini",
			key:      os.Getenv("GEMINI_API_KEY"),
			endpoint: fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=%s", os.Getenv("GEMINI_API_KEY")),
			model:    "gemini-2.0-flash",
		},
		{
			name:     "deepseek",
			key:      os.Getenv("DEEPSEEK_API_KEY"),
			endpoint: "https://api.deepseek.com/chat/completions",
			model:    "deepseek-chat",
		},
	}

	for _, p := range providers {
		if p.key == "" {
			continue
		}
		text, err := callOpenAICompat(ctx, p.endpoint, p.key, p.model, sys, prompt)
		if err != nil {
			log.Printf("[AIStudio] %s text failed (slug=%s): %v", p.name, slug, err)
			continue
		}
		return &studioProviderResult{
			OutputText: text,
			Provider:   p.name,
			CostMicros: estimateLLMCost(len(text)),
		}, nil
	}
	return nil, fmt.Errorf("all LLM providers failed for tool %q", slug)
}

func callOpenAICompat(ctx context.Context, endpoint, key, model, sys, user string) (string, error) {
	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": sys},
			{"role": "user", "content": user},
		},
		"max_tokens":  2048,
		"temperature": 0.7,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}
	var parsed struct {
		Choices []struct {
			Message struct{ Content string `json:"content"` } `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Choices) == 0 {
		return "", fmt.Errorf("parse OpenAI-compat: %w", err)
	}
	return parsed.Choices[0].Message.Content, nil
}

// ─── Image (FAL.AI FLUX → HuggingFace SDXL) ──────────────────────────────────

func (o *AIStudioOrchestrator) dispatchImage(ctx context.Context, slug, prompt string) (*studioProviderResult, error) {
	if key := os.Getenv("FAL_API_KEY"); key != "" {
		falModel := "fal-ai/flux/dev"
		if slug == "bg-remover" {
			falModel = "fal-ai/birefnet"
		}
		url, err := callFALImage(ctx, key, falModel, prompt)
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "fal.ai/" + falModel, CostMicros: 3000}, nil
		}
		log.Printf("[AIStudio] FAL image failed: %v", err)
	}

	if key := os.Getenv("HUGGINGFACE_API_KEY"); key != "" {
		url, err := callHuggingFaceImage(ctx, key, "stabilityai/stable-diffusion-xl-base-1.0", prompt)
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "huggingface/sdxl", CostMicros: 0}, nil
		}
		log.Printf("[AIStudio] HF image failed: %v", err)
	}

	return nil, fmt.Errorf("all image providers failed")
}

func callFALImage(ctx context.Context, key, model, prompt string) (string, error) {
	payload := map[string]interface{}{
		"prompt":        prompt,
		"image_size":    "square_hd",
		"num_images":    1,
		"output_format": "jpeg",
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://fal.run/"+model, bytes.NewReader(body))
	req.Header.Set("Authorization", "Key "+key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("FAL %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}
	var parsed struct {
		Images []struct{ URL string `json:"url"` } `json:"images"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Images) == 0 {
		return "", fmt.Errorf("fal parse: %w", err)
	}
	return parsed.Images[0].URL, nil
}

func callHuggingFaceImage(ctx context.Context, key, model, prompt string) (string, error) {
	endpoint := "https://api-inference.huggingface.co/models/" + model
	body, _ := json.Marshal(map[string]string{"inputs": prompt})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusServiceUnavailable {
		return "", fmt.Errorf("HF model loading")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HF %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}
	// HF returns raw binary; encode as data URI (prod: upload to CDN first)
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw), nil
}

// ─── Voice (ElevenLabs → HuggingFace Bark) ───────────────────────────────────

func (o *AIStudioOrchestrator) dispatchVoice(ctx context.Context, slug, prompt string) (*studioProviderResult, error) {
	if slug == "transcribe" {
		return o.dispatchTranscribe(ctx, prompt)
	}

	if key := os.Getenv("ELEVENLABS_API_KEY"); key != "" {
		voiceID := os.Getenv("ELEVENLABS_VOICE_ID")
		if voiceID == "" {
			voiceID = "21m00Tcm4TlvDq8ikWAM" // Rachel (default)
		}
		url, err := callElevenLabsTTS(ctx, key, voiceID, prompt)
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "elevenlabs", CostMicros: 2000}, nil
		}
		log.Printf("[AIStudio] ElevenLabs TTS failed: %v", err)
	}

	if key := os.Getenv("HUGGINGFACE_API_KEY"); key != "" {
		url, err := callHuggingFaceTTS(ctx, key, prompt)
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "huggingface/bark", CostMicros: 0}, nil
		}
		log.Printf("[AIStudio] HF Bark failed: %v", err)
	}

	return nil, fmt.Errorf("all TTS providers failed")
}

func callElevenLabsTTS(ctx context.Context, key, voiceID, text string) (string, error) {
	payload := map[string]interface{}{
		"text":     text,
		"model_id": "eleven_turbo_v2",
		"voice_settings": map[string]float64{
			"stability": 0.5, "similarity_boost": 0.75,
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.elevenlabs.io/v1/text-to-speech/"+voiceID, bytes.NewReader(body))
	req.Header.Set("xi-api-key", key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	audio, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ElevenLabs %d: %s", resp.StatusCode, truncateStr(string(audio), 200))
	}
	// Prod: upload to CDN; dev: data URI
	return "data:audio/mpeg;base64," + base64.StdEncoding.EncodeToString(audio), nil
}

func callHuggingFaceTTS(ctx context.Context, key, text string) (string, error) {
	body, _ := json.Marshal(map[string]string{"inputs": text})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api-inference.huggingface.co/models/suno/bark", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	audio, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HF Bark %d", resp.StatusCode)
	}
	return "data:audio/wav;base64," + base64.StdEncoding.EncodeToString(audio), nil
}

func (o *AIStudioOrchestrator) dispatchTranscribe(ctx context.Context, audioURL string) (*studioProviderResult, error) {
	// Groq Whisper-large-v3 — fastest + cheapest
	if key := os.Getenv("GROQ_API_KEY"); key != "" {
		text, err := callGroqWhisper(ctx, key, audioURL)
		if err == nil {
			return &studioProviderResult{OutputText: text, Provider: "groq/whisper", CostMicros: 100}, nil
		}
		log.Printf("[AIStudio] Groq Whisper failed: %v", err)
	}
	return nil, fmt.Errorf("transcription unavailable — no GROQ_API_KEY configured")
}

func callGroqWhisper(ctx context.Context, key, audioURL string) (string, error) {
	payload := map[string]string{"url": audioURL, "model": "whisper-large-v3"}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.groq.com/openai/v1/audio/transcriptions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var parsed struct{ Text string `json:"text"` }
	json.NewDecoder(resp.Body).Decode(&parsed)
	if parsed.Text == "" {
		return "", fmt.Errorf("Groq Whisper returned no text")
	}
	return parsed.Text, nil
}

// ─── Music (Mubert → ElevenLabs sound-generation) ────────────────────────────

func (o *AIStudioOrchestrator) dispatchMusic(ctx context.Context, slug, prompt string) (*studioProviderResult, error) {
	if key := os.Getenv("MUBERT_API_KEY"); key != "" {
		url, err := callMubert(ctx, key, prompt, slug)
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "mubert", CostMicros: 5000}, nil
		}
		log.Printf("[AIStudio] Mubert failed: %v", err)
	}

	if key := os.Getenv("ELEVENLABS_API_KEY"); key != "" {
		url, err := callElevenLabsMusic(ctx, key, prompt)
		if err == nil {
			return &studioProviderResult{OutputURL: url, Provider: "elevenlabs/music", CostMicros: 8000}, nil
		}
		log.Printf("[AIStudio] ElevenLabs music failed: %v", err)
	}

	return nil, fmt.Errorf("all music providers failed")
}

func callMubert(ctx context.Context, key, prompt, slug string) (string, error) {
	duration := 30
	if slug == "bg-music" {
		duration = 60
	}
	payload := map[string]interface{}{
		"method": "RecordTrackTTM",
		"params": map[string]interface{}{
			"pat": key, "text": prompt, "duration": duration, "format": "mp3", "bitrate": 128,
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api-b2b.mubert.com/v2/RecordTrackTTM", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		Data struct {
			Tasks []struct{ DownloadLink string `json:"download_link"` } `json:"tasks"`
		} `json:"data"`
		Error *struct{ Text string `json:"text"` } `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Error != nil {
		return "", fmt.Errorf("mubert: %s", result.Error.Text)
	}
	if len(result.Data.Tasks) == 0 || result.Data.Tasks[0].DownloadLink == "" {
		return "", fmt.Errorf("mubert: no download_link")
	}
	return result.Data.Tasks[0].DownloadLink, nil
}

func callElevenLabsMusic(ctx context.Context, key, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]interface{}{"prompt": prompt, "duration": 30})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.elevenlabs.io/v1/sound-generation", bytes.NewReader(body))
	req.Header.Set("xi-api-key", key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	audio, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ElevenLabs music %d", resp.StatusCode)
	}
	return "data:audio/mpeg;base64," + base64.StdEncoding.EncodeToString(audio), nil
}

// ─── Video (FAL.AI Kling v1.5 → LTX-Video) ───────────────────────────────────

func (o *AIStudioOrchestrator) dispatchVideo(ctx context.Context, slug, prompt string) (*studioProviderResult, error) {
	key := os.Getenv("FAL_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("FAL_API_KEY not configured for video generation")
	}

	primaryModel := "fal-ai/kling-video/v1.5/standard/image-to-video"
	if slug == "video-premium" {
		primaryModel = "fal-ai/kling-video/v1.5/pro/text-to-video"
	}

	url, err := callFALVideo(ctx, key, primaryModel, prompt)
	if err == nil {
		return &studioProviderResult{OutputURL: url, Provider: "fal.ai/kling", CostMicros: 25000}, nil
	}
	log.Printf("[AIStudio] FAL Kling failed: %v", err)

	// Fallback: LTX-Video (faster, cheaper)
	url, err = callFALVideo(ctx, key, "fal-ai/ltx-video", prompt)
	if err == nil {
		return &studioProviderResult{OutputURL: url, Provider: "fal.ai/ltx-video", CostMicros: 8000}, nil
	}
	log.Printf("[AIStudio] FAL LTX-Video failed: %v", err)

	return nil, fmt.Errorf("all video providers failed")
}

func callFALVideo(ctx context.Context, key, model, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]interface{}{"prompt": prompt, "duration": "5"})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://fal.run/"+model, bytes.NewReader(body))
	req.Header.Set("Authorization", "Key "+key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("FAL video %d: %s", resp.StatusCode, truncateStr(string(raw), 200))
	}

	// FAL.AI returns either {"video":{"url":"..."}} or {"url":"..."}
	var parsed struct {
		Video struct{ URL string `json:"url"` } `json:"video"`
		URL   string                             `json:"url"`
	}
	json.Unmarshal(raw, &parsed)
	if parsed.Video.URL != "" {
		return parsed.Video.URL, nil
	}
	if parsed.URL != "" {
		return parsed.URL, nil
	}
	return "", fmt.Errorf("FAL video: no URL in response")
}

// ─── Composite tools ──────────────────────────────────────────────────────────

func (o *AIStudioOrchestrator) dispatchComposite(ctx context.Context, slug, prompt string) (*studioProviderResult, error) {
	switch slug {
	case "podcast":
		return o.assemblePodcast(ctx, prompt)
	case "infographic":
		return o.assembleInfographic(ctx, prompt)
	default:
		// Treat unknown composites as text
		return o.dispatchText(ctx, slug, prompt)
	}
}

func (o *AIStudioOrchestrator) assemblePodcast(ctx context.Context, prompt string) (*studioProviderResult, error) {
	// Step 1: LLM generates 2-host script
	script, err := o.dispatchText(ctx, "podcast", prompt)
	if err != nil {
		return nil, fmt.Errorf("podcast script: %w", err)
	}

	// Step 2: Narrate script (best-effort — return text-only if voice fails)
	voice, err := o.dispatchVoice(ctx, "narrate", script.OutputText)
	if err != nil {
		log.Printf("[AIStudio] podcast voice step failed (returning text-only): %v", err)
		return &studioProviderResult{
			OutputText: script.OutputText,
			Provider:   script.Provider + "/text-only",
			CostMicros: script.CostMicros,
		}, nil
	}
	return &studioProviderResult{
		OutputURL:  voice.OutputURL,
		OutputText: script.OutputText,
		Provider:   script.Provider + "+" + voice.Provider,
		CostMicros: script.CostMicros + voice.CostMicros,
	}, nil
}

func (o *AIStudioOrchestrator) assembleInfographic(ctx context.Context, prompt string) (*studioProviderResult, error) {
	// Step 1: LLM generates JSON data layout
	outline, err := o.dispatchText(ctx, "infographic", prompt)
	if err != nil {
		return nil, err
	}

	// Step 2: Image model renders a visual (best-effort)
	visualPrompt := fmt.Sprintf(
		"Professional infographic design about: %s. Modern, data-rich, colourful, clean typography. %s",
		prompt, truncateStr(outline.OutputText, 200),
	)
	img, err := o.dispatchImage(ctx, "ai-photo", visualPrompt)
	if err != nil {
		log.Printf("[AIStudio] infographic image step failed (returning text-only): %v", err)
		return &studioProviderResult{
			OutputText: outline.OutputText,
			Provider:   outline.Provider + "/text-only",
			CostMicros: outline.CostMicros,
		}, nil
	}
	return &studioProviderResult{
		OutputURL:  img.OutputURL,
		OutputText: outline.OutputText,
		Provider:   outline.Provider + "+" + img.Provider,
		CostMicros: outline.CostMicros + img.CostMicros,
	}, nil
}

// ─── Persist helpers ──────────────────────────────────────────────────────────

// complete persists result via StudioService (which owns the ledger / notifications).
func (o *AIStudioOrchestrator) complete(ctx context.Context, gen *entities.AIGeneration, r *studioProviderResult) {
	if err := o.studioSvc.CompleteGeneration(
		ctx, gen.ID,
		r.OutputURL, r.OutputText, r.Provider,
		r.CostMicros, r.DurationMs,
	); err != nil {
		log.Printf("[AIStudio] complete gen %s: %v", gen.ID, err)
	}
}

// fail persists failure and issues refund via StudioService.
func (o *AIStudioOrchestrator) fail(ctx context.Context, gen *entities.AIGeneration, reason string) {
	if err := o.studioSvc.FailGeneration(ctx, gen.ID, reason); err != nil {
		log.Printf("[AIStudio] fail gen %s: %v", gen.ID, err)
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// textSystemPrompt returns a context-appropriate system prompt for each tool slug.
func textSystemPrompt(slug string) string {
	m := map[string]string{
		"translate":      "You are an expert translator. Detect the language of the input and translate it to English (or to the language requested). Return only the translated text.",
		"study-guide":    "You are an expert educator. Create a comprehensive study guide: Overview, Key Concepts, Summary Points, 3 Practice Questions. Use clear headings.",
		"quiz":           "You are a quiz generator. Create 5 multiple-choice questions. Format each as:\nQ1. [question]\nA) [opt]\nB) [opt]\nC) [opt]\nD) [opt]\nAnswer: [letter]",
		"mindmap":        `You are a mind-map expert. Return ONLY valid JSON: {"central":"topic","branches":[{"name":"branch","sub":["item1","item2"]}]}`,
		"research-brief": "You are a research analyst. Write a concise 500–800 word research brief: Executive Summary, Key Findings (3–5), Analysis, Recommendations. Nigerian business context.",
		"bizplan":        "You are a startup advisor. Write a 1-page business plan: Description, Target Market, Revenue Model, Competitive Advantage, Marketing, Year-1 Projections, Action Steps. Nigerian market focus.",
		"slide-deck":     `You are a presentation expert. Return ONLY valid JSON: {"title":"...","slides":[{"title":"...","bullets":["..."],"speaker_notes":"..."}]} — 10 slides.`,
		"infographic":    `You are a data visualisation expert. Return ONLY valid JSON: {"title":"...","sections":[{"heading":"...","stat":"...","icon":"emoji","description":"..."}],"source":"..."}`,
		"podcast":        "You are a podcast scriptwriter for a Nigerian audience. Write an engaging 2-host script (HOST_A / HOST_B alternating). Under 500 words. Conversational, culturally relevant.",
	}
	if p, ok := m[slug]; ok {
		return p
	}
	return "You are a helpful AI assistant. Answer the user's request concisely and accurately."
}

func normaliseSlug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	return strings.ReplaceAll(s, "_", "-")
}

func estimateLLMCost(chars int) int {
	tokens := chars / 4
	return (tokens * 100) / 1000 // ~100 µUSD per 1k tokens
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
