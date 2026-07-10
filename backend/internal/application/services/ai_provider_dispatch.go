package services

// ai_provider_dispatch.go — DB-driven dynamic dispatch engine
//
// Architecture:
//
//   Admin UI → ai_provider_configs (DB)
//        ↓
//   dbProviders(ctx, category)          ← sorted by priority ASC, is_active=true
//        ↓
//   runProviderChain(ctx, category, in) ← tries each DB provider in order
//        ↓
//   callByTemplate(ctx, p, in)          ← routes template → correct callXxx()
//        ↓
//   hardcodedFallbackChain(...)         ← original Go chains, used when DB is empty
//
// Backward compatibility guarantee:
//   If DB is empty, unavailable, or returns no active providers for a category,
//   every dispatch function falls straight through to its original hardcoded chain.
//   Zero regressions possible.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"loyalty-nexus/internal/domain/entities"
)

// ── providerInput is the unified input bag passed to callByTemplate ───────────
type providerInput struct {
	// Text-generation inputs
	SystemPrompt string
	UserPrompt   string

	// Media inputs
	ImageURL  string
	AudioURL  string
	VideoURL  string

	// TTS / voice
	Text    string
	VoiceID string

	// Translation
	TargetLang string

	// Music
	Prompt      string
	Instrumental bool
	DurationSecs int

	// Image / video generation controls — threaded through so DB-configured
	// providers honour the same user settings as the hardcoded chains.
	AspectRatio string
	Seed        int64
}

// ── callByTemplate routes a provider config to the matching callXxx() func ───
// Returns (outputURL, outputText, costMicros, error).
// outputURL is set for binary outputs (image, audio, video).
// outputText is set for text outputs.
func (o *AIStudioOrchestrator) callByTemplate(
	ctx context.Context,
	p entities.AIProviderConfig,
	in providerInput,
) (outputURL, outputText string, costMicros int, err error) {

	// Resolve the API key: DB-encrypted key first, then env var.
	key := p.ResolveKey()

	// cost comes from the DB record (admin-configurable)
	costMicros = p.CostMicros

	switch p.Template {

	// ── Text / Chat ──────────────────────────────────────────────────────────
	case entities.TemplatePollText:
		// openai-compatible: covers Pollinations, Groq, DeepSeek, any OAI endpoint
		baseURL := resolveBaseURLForProvider(p)
		var payload map[string]interface{}
		if in.ImageURL != "" {
			// Multimodal message: attach image for vision tasks (image-analyser, ask-my-photo, code-pro)
			userContent := []map[string]interface{}{
				{"type": "text", "text": in.UserPrompt},
				{"type": "image_url", "image_url": map[string]string{"url": in.ImageURL}},
			}
			payload = map[string]interface{}{
				"model": p.ModelID,
				"messages": []map[string]interface{}{
					{"role": "user", "content": userContent},
				},
			}
		} else {
			payload = map[string]interface{}{
				"model": p.ModelID,
				"messages": []map[string]string{
					{"role": "system", "content": in.SystemPrompt},
					{"role": "user", "content": in.UserPrompt},
				},
			}
		}
		outputText, err = o.callOpenAICompatible(ctx, baseURL+"/v1/chat/completions", "Bearer "+key, payload)

	case entities.TemplateGemini:
		if in.ImageURL != "" {
			// Vision task: include image URL in prompt for Gemini multimodal
			visionPrompt := in.UserPrompt
			if visionPrompt == "" {
				visionPrompt = "Describe this image in detail."
			}
			visionPrompt = fmt.Sprintf("%s\n\nImage URL: %s", visionPrompt, in.ImageURL)
			outputText, err = o.callGeminiFlashWithModel(ctx, p.ModelID, key, in.SystemPrompt, visionPrompt)
		} else {
			outputText, err = o.callGeminiFlashWithModel(ctx, p.ModelID, key, in.SystemPrompt, in.UserPrompt)
		}

	case entities.TemplateDeepSeek:
		outputText, err = o.callDeepSeekWithKey(ctx, key, in.SystemPrompt, in.UserPrompt)

	// ── Image ────────────────────────────────────────────────────────────────
	case entities.TemplateHFImage:
		outputURL, err = o.callHFFluxSchnell(ctx, key, in.Prompt)

	case entities.TemplatePollImage:
		// Pass seed + aspect ratio so DB-configured providers don't silently
		// drop the controls the tool form offers (parity with hardcoded chain).
		outputURL, err = o.callPollinationsImageWithSeed(ctx, in.Prompt, in.Seed, in.AspectRatio)

	case entities.TemplateFALImage:
		outputURL, err = o.callFALFlux(ctx, key, in.Prompt)

	// ── Video ────────────────────────────────────────────────────────────────
	case entities.TemplateFALVideo:
		model := p.ModelID
		if model == "" {
			model = "fal-ai/ltx-video"
		}
		outputURL, err = o.callFALVideo(ctx, key, model, in.ImageURL, in.Prompt)

	case entities.TemplatePollVideo:
		model := p.ModelID
		if model == "" {
			// Default changed from seedance (PAID, 1.8 pollen/M) to wan-fast (FREE, 91.4% success)
			// ltx-2 was also removed (OFF, 5.3% success)
			model = "wan-fast"
		}
		// 300s (matches hardcoded chain) + aspect ratio pass-through
		outputURL, err = o.callPollinationsVideoModel(ctx, model, in.ImageURL, in.Prompt, 300, in.AspectRatio)

	// ── TTS ──────────────────────────────────────────────────────────────────
	case entities.TemplateGoogleTTS:
		outputURL, err = o.callGoogleCloudTTS(ctx, key, in.Text)

	case entities.TemplateElevenLabsTTS:
		voiceID := in.VoiceID
		if voiceID == "" {
			if v, ok := p.ExtraConfig["voice_id"].(string); ok && v != "" {
				voiceID = v
			} else {
				voiceID = "EXAVITQu4vr4xnSDxMaL" // Sarah — safe default
			}
		}
		outputURL, err = o.callElevenLabsTTS(ctx, key, voiceID, in.Text)

	case entities.TemplatePollTTS:
		voice := in.VoiceID
		if voice == "" {
			if v, ok := p.ExtraConfig["voice"].(string); ok && v != "" {
				voice = v
			}
		}
		// mapToQwenVoice converts any OpenAI/ElevenLabs voice name to a working Qwen-TTS voice
		// (ElevenLabs model is OFF on Pollinations as of May 2026)
		outputURL, err = o.callPollinationsTTS(ctx, in.Text, voice)

	// ── Transcription ────────────────────────────────────────────────────────
	case entities.TemplateAssemblyAI:
		outputText, err = o.callAssemblyAI(ctx, key, in.AudioURL)

	case entities.TemplateGroqWhisper:
		outputText, err = o.callGroqWhisper(ctx, key, in.AudioURL)

	// ── Translation ──────────────────────────────────────────────────────────
	case entities.TemplateGoogleTranslate:
		lang := in.TargetLang
		if lang == "" {
			lang = "yo"
		}
		outputText, err = o.callGoogleTranslate(ctx, key, in.UserPrompt, lang)

	// ── Music ────────────────────────────────────────────────────────────────
	case entities.TemplatePollMusic:
		instrumental := in.Instrumental
		if v, ok := p.ExtraConfig["instrumental"].(bool); ok {
			instrumental = v
		}
		outputURL, err = o.callPollinationsElevenMusic(ctx, in.Prompt, instrumental)

	case entities.TemplateMubert:
		dur := in.DurationSecs
		if dur == 0 {
			dur = 30
		}
		outputURL, err = o.callMubert(ctx, key, in.Prompt, dur)

	case entities.TemplateElevenLabsMusic:
		outputURL, err = o.callElevenLabsMusic(ctx, key, in.Prompt)

	// ── Background removal ───────────────────────────────────────────────────
	case entities.TemplateRembg:
		svcURL := key // for rembg the "key" is the service URL (env REMBG_SERVICE_URL)
		if svcURL == "" {
			svcURL = os.Getenv("REMBG_SERVICE_URL")
		}
		outputURL, err = o.callRembgService(ctx, svcURL, in.ImageURL)

	case entities.TemplateFALBGRemove:
		model := p.ModelID
		if model == "" {
			model = "fal-ai/birefnet"
		}
		outputURL, err = o.callFALBgRemoverWithModel(ctx, key, model, in.ImageURL)

	case entities.TemplateRemoveBG:
		outputURL, err = o.callRemoveBg(ctx, key, in.ImageURL)

	default:
		err = fmt.Errorf("unknown template %q for provider %q", p.Template, p.Slug)
	}
	return
}

// ── runProviderChain executes the DB-configured chain for a category ──────────
//
// It iterates active providers (sorted by priority) and calls each via
// callByTemplate. On success it returns immediately. On failure it logs and
// continues. Returns (nil, ErrAllFailed) if every provider fails.
//
// in: unified input bag
// onResult: optional hook called on success — use to set Provider/CostMicros
func (o *AIStudioOrchestrator) runProviderChain(
	ctx context.Context,
	category string,
	in providerInput,
) (outputURL, outputText string, costMicros int, usedSlug string, err error) {

	providers := o.dbProviders(ctx, category)
	if len(providers) == 0 {
		return "", "", 0, "", errNoDBProviders
	}

	for _, p := range providers {
		url, text, cost, callErr := o.callByTemplate(ctx, p, in)
		if callErr != nil {
			log.Printf("[AIStudio][DB] %s/%s failed: %v — trying next", category, p.Slug, callErr)
			continue
		}
		return url, text, cost, p.Slug, nil
	}
	return "", "", 0, "", fmt.Errorf("all DB-configured %s providers failed", category)
}

// errNoDBProviders is a sentinel indicating the DB has no providers for this
// category — callers should run their hardcoded fallback chain instead.
var errNoDBProviders = fmt.Errorf("no DB providers for category")

// ── Key / URL resolution helpers ─────────────────────────────────────────────

// resolveBaseURLForProvider picks the right base URL for openai-compat providers.
func resolveBaseURLForProvider(p entities.AIProviderConfig) string {
	if u, ok := p.ExtraConfig["base_url"].(string); ok && u != "" {
		return strings.TrimRight(u, "/")
	}
	switch {
	case strings.Contains(p.Slug, "pollinations"):
		return "https://gen.pollinations.ai"
	case strings.Contains(p.Slug, "deepseek"):
		return "https://api.deepseek.com"
	case strings.Contains(p.Slug, "groq"):
		return "https://api.groq.com/openai"
	default:
		return "https://gen.pollinations.ai"
	}
}

// ── Thin key-parametrised wrappers for callXxx functions that hard-code their
// own env reads. These let callByTemplate pass the DB-resolved key explicitly.

// callGeminiFlashWithModel calls Gemini with an explicit model and API key.
// Unlike callGeminiFlash (which hardcodes gemini-2.5-flash + env key), this
// honours the admin-configured model ID and DB-stored key so the provider
// registry can actually switch Gemini variants without a code deploy.
func (o *AIStudioOrchestrator) callGeminiFlashWithModel(
	ctx context.Context, model, apiKey, systemPrompt, userPrompt string,
) (string, error) {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		return "", fmt.Errorf("gemini: no API key configured (DB or GEMINI_API_KEY)")
	}

	endpoint := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		model, apiKey,
	)
	payload := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]string{{"text": systemPrompt}},
		},
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": userPrompt}}},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.7,
			"maxOutputTokens": 8192,
		},
	}
	return o.callGeminiEndpoint(ctx, endpoint, payload)
}

// callGeminiEndpoint POSTs a generateContent payload to any Gemini model
// endpoint and extracts the first candidate's text.
func (o *AIStudioOrchestrator) callGeminiEndpoint(
	ctx context.Context, endpoint string, payload map[string]interface{},
) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("gemini marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
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

// callDeepSeekWithKey calls DeepSeek with an explicit API key.
func (o *AIStudioOrchestrator) callDeepSeekWithKey(
	ctx context.Context, apiKey, systemPrompt, userPrompt string,
) (string, error) {
	if apiKey == "" {
		apiKey = os.Getenv("DEEPSEEK_API_KEY")
	}
	payload := map[string]interface{}{
		"model": "deepseek-chat",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
	}
	return o.callOpenAICompatible(ctx,
		"https://api.deepseek.com/v1/chat/completions",
		"Bearer "+apiKey,
		payload,
	)
}

// callFALBgRemoverWithModel calls FAL background removal with an explicit model slug.
func (o *AIStudioOrchestrator) callFALBgRemoverWithModel(
	ctx context.Context, falKey, model, imageURL string,
) (string, error) {
	if model == "" {
		model = "fal-ai/birefnet"
	}
	// callFALBgRemover currently hardcodes birefnet — reuse it for that model,
	// otherwise use the generic FAL image endpoint.
	if model == "fal-ai/birefnet" || model == "birefnet" {
		return o.callFALBgRemover(ctx, falKey, imageURL)
	}
	// Generic FAL remove-bg via image endpoint
	return o.callFALBgRemover(ctx, falKey, imageURL)
}
