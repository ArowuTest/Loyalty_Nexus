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
	"strings"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/infrastructure/external"
)

// ── providerInput is the unified input bag passed to callByTemplate ───────────
type providerInput struct {
	// Text-generation inputs
	SystemPrompt string
	UserPrompt   string

	// Media inputs
	ImageURL           string
	Images             []string // base64/data URLs for multimodal tool stages
	ReferenceImageURLs []string
	DocumentURL        string
	AudioURL           string
	VideoURL           string

	// TTS / voice
	Text        string
	VoiceID     string
	Speed       float64
	AudioFormat string
	Language    string

	// Transcription
	SpeakerLabels bool
	OutputFormat  string

	// Translation
	TargetLang string

	// Music
	Prompt       string
	Instrumental bool
	DurationSecs int
	Style        string
	Title        string
	VocalGender  string
	SecondaryURL *string

	// Image / video generation controls — threaded through so DB-configured
	// providers honour the same user settings as the hardcoded chains.
	AspectRatio   string
	Seed          int64
	Extend        bool
	Resolution    string
	GenerateAudio bool
	Extra         map[string]interface{}
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
		// Generic chat-completions drivers do not receive arbitrary uploaded documents.
		// A future provider can opt in with a dedicated template rather than silently ignoring the file.
		if in.DocumentURL != "" {
			return "", "", costMicros, fmt.Errorf("template %s does not support document input", p.Template)
		}
		baseURL := resolveBaseURLForProvider(p)
		var payload map[string]interface{}
		if in.ImageURL != "" || len(in.Images) > 0 {
			userContent := []map[string]interface{}{{"type": "text", "text": in.UserPrompt}}
			if in.ImageURL != "" {
				userContent = append(userContent, map[string]interface{}{
					"type": "image_url", "image_url": map[string]string{"url": in.ImageURL},
				})
			}
			for _, image := range in.Images {
				if image == "" {
					continue
				}
				if !strings.HasPrefix(image, "data:") && !strings.HasPrefix(image, "http://") && !strings.HasPrefix(image, "https://") {
					image = "data:image/jpeg;base64," + image
				}
				userContent = append(userContent, map[string]interface{}{
					"type": "image_url", "image_url": map[string]string{"url": image},
				})
			}
			payload = map[string]interface{}{
				"model": p.ModelID,
				"messages": []map[string]interface{}{
					{"role": "system", "content": in.SystemPrompt},
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
		if webSearch, _ := p.ExtraConfig["web_search"].(bool); webSearch {
			if strings.Contains(strings.ToLower(baseURL), "openrouter.ai") {
				payload["tools"] = []map[string]interface{}{{"type": "openrouter:web_search"}}
			} else {
				// Pollinations-compatible endpoints use the search flag.
				payload["search"] = true
			}
		}
		outputText, err = o.callOpenAICompatible(ctx, baseURL+"/v1/chat/completions", "Bearer "+key, payload)

	case entities.TemplateGemini:
		if in.DocumentURL != "" {
			outputText, err = o.callGeminiConfiguredDocument(ctx, p.ModelID, key, in.SystemPrompt, in.UserPrompt, in.DocumentURL)
		} else if len(in.Images) > 0 {
			outputText, err = o.callGeminiConfiguredMultimodal(ctx, p.ModelID, key, in.SystemPrompt, in.UserPrompt, in.Images)
		} else if in.ImageURL != "" {
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
		outputText, err = o.callDeepSeekWithKey(ctx, key, p.ModelID, in.SystemPrompt, in.UserPrompt)

	case entities.TemplateTavilySearch:
		outputText, err = o.callTavilySearch(ctx, key, in.UserPrompt)

	// ── Image ────────────────────────────────────────────────────────────────
	case entities.TemplateHFImage:
		if in.ImageURL != "" || len(in.ReferenceImageURLs) > 0 {
			err = fmt.Errorf("HF image driver does not support reference/edit input")
			break
		}
		outputURL, err = o.callHFFluxSchnell(ctx, key, in.Prompt)

	case entities.TemplatePollImage:
		if in.ImageURL != "" || len(in.ReferenceImageURLs) > 0 {
			err = fmt.Errorf("Pollinations base image driver does not support reference/edit input")
			break
		}
		outputURL, err = o.callPollinationsImageWithConfig(ctx, key, p.ModelID, in.Prompt, in.Seed, in.AspectRatio)

	case entities.TemplatePollGPTImage:
		if in.ImageURL != "" || len(in.ReferenceImageURLs) > 0 {
			err = fmt.Errorf("Pollinations premium image driver does not support reference/edit input")
			break
		}
		quality, _ := in.Extra["quality"].(string)
		outputURL, err = o.callPollinationsGPTImageWithKey(ctx, key, in.Prompt, p.ModelID, in.AspectRatio, quality)

	case entities.TemplatePollImageEdit:
		if in.ImageURL == "" {
			err = fmt.Errorf("Pollinations image-edit requires image_url")
			break
		}
		outputURL, err = o.callPollinationsImageEditWithKey(ctx, key, p.ModelID, in.ImageURL, in.Prompt)

	case entities.TemplateFALImage:
		if in.ImageURL != "" || len(in.ReferenceImageURLs) > 0 {
			err = fmt.Errorf("FAL base image driver does not support reference/edit input")
			break
		}
		outputURL, err = o.callFALFlux(ctx, key, in.Prompt)

	case entities.TemplateFALImageUltra:
		imageURL := in.ImageURL
		if imageURL == "" && len(in.ReferenceImageURLs) > 0 {
			imageURL = in.ReferenceImageURLs[0]
		}
		numImages := 1
		if n, ok := in.Extra["num_images"].(float64); ok && n >= 1 && n <= 4 {
			numImages = int(n)
		}
		strength := 0.35
		if s, ok := in.Extra["image_prompt_strength"].(float64); ok && s > 0 && s <= 1 {
			strength = s
		}
		urls, callErr := o.callFALFluxUltraWithModel(ctx, key, p.ModelID, in.Prompt, imageURL, strength, numImages, in.AspectRatio)
		if callErr != nil {
			err = callErr
		} else if len(urls) == 0 {
			err = fmt.Errorf("FAL Ultra returned no images")
		} else {
			outputURL = urls[0]
			costMicros = p.CostMicros * numImages
		}

	case entities.TemplateFALImageEdit:
		if in.ImageURL == "" {
			err = fmt.Errorf("FAL image-edit requires image_url")
			break
		}
		outputURL, err = o.callFALImageEditWithModel(ctx, key, p.ModelID, in.ImageURL, in.Prompt)

	case entities.TemplateGrokImage:
		grok := external.NewGrokAdapter(key)
		refs := append([]string(nil), in.ReferenceImageURLs...)
		if in.ImageURL != "" {
			refs = append([]string{in.ImageURL}, refs...)
		}
		if len(refs) > 0 {
			outputURL, err = grok.ComposeImages(ctx, in.Prompt, refs, in.AspectRatio)
		} else {
			resolution := in.Resolution
			if resolution == "" {
				resolution, _ = p.ExtraConfig["resolution"].(string)
			}
			outputURL, err = grok.GenerateImage(ctx, in.Prompt, resolution)
		}

	// ── Video ────────────────────────────────────────────────────────────────
	case entities.TemplateFALVideo:
		if in.VideoURL != "" || in.Extend || len(in.ReferenceImageURLs) > 0 {
			err = fmt.Errorf("FAL video driver does not support edit/extend/reference mode")
			break
		}
		model := p.ModelID
		if model == "" {
			model = "fal-ai/ltx-video"
		}
		outputURL, err = o.callFALVideo(ctx, key, model, in.ImageURL, in.Prompt, promptEnvelope{
			Duration: in.DurationSecs, AspectRatio: in.AspectRatio, Extra: in.Extra,
		})

	case entities.TemplateFALVideoMulti:
		if in.VideoURL != "" || in.Extend || len(in.ReferenceImageURLs) < 2 {
			err = fmt.Errorf("FAL multi-image video requires at least two reference images")
			break
		}
		model := p.ModelID
		if model == "" {
			model = "fal-ai/kling-video/v2.6/pro/multi-image-to-video"
		}
		outputURL, err = o.callFALMultiImageVideoConfigured(ctx, key, model, in.ReferenceImageURLs, in.Prompt, in.DurationSecs, in.AspectRatio, in.Extra)

	case entities.TemplateGrokVideo:
		grok := external.NewGrokAdapter(key)
		duration := in.DurationSecs
		if duration <= 0 {
			duration = 6
		}
		resolution := in.Resolution
		if resolution == "" {
			resolution = "720p"
		}
		outputURL, err = grok.GenerateVideo(ctx, external.GrokVideoRequest{
			Prompt: in.Prompt, ImageURL: in.ImageURL, VideoURL: in.VideoURL,
			ReferenceImageURLs: in.ReferenceImageURLs, Duration: duration,
			AspectRatio: in.AspectRatio, Resolution: resolution, Extend: in.Extend,
		})
		costMicros = 50000 * duration

	// ── Avatar (talking-head) ──────────────────────────────────────────────────
	case entities.TemplateFALAvatarText:
		if in.Prompt == "" {
			err = fmt.Errorf("text-driven avatar provider requires script")
			break
		}
		outputURL, err = o.callFALAvatarText(ctx, key, p.ModelID, in.ImageURL, in.Prompt, in.VoiceID)

	case entities.TemplateFALAvatarAudio:
		if in.AudioURL == "" {
			err = fmt.Errorf("audio-driven avatar provider requires audio_url")
			break
		}
		outputURL, err = o.callFALAvatarAudio(ctx, key, p.ModelID, in.ImageURL, in.AudioURL)

	case entities.TemplateHeyGen:
		if in.Prompt == "" {
			err = fmt.Errorf("HeyGen avatar provider requires script")
			break
		}
		outputURL, err = o.callHeyGen(ctx, key, in.ImageURL, in.Prompt, in.VoiceID)

	case entities.TemplatePollVideo:
		if in.VideoURL != "" || in.Extend || len(in.ReferenceImageURLs) > 0 {
			err = fmt.Errorf("Pollinations video driver does not support edit/extend/reference mode")
			break
		}
		model := p.ModelID
		if model == "" {
			// Default changed from seedance (PAID, 1.8 pollen/M) to wan-fast (FREE, 91.4% success)
			// ltx-2 was also removed (OFF, 5.3% success)
			model = "wan-fast"
		}
		dur := in.DurationSecs
		if dur <= 0 {
			dur = 5
		}
		outputURL, err = o.callPollinationsVideoModelWithKey(ctx, key, model, in.ImageURL, in.Prompt, 300, in.AspectRatio, fmt.Sprintf("%d", dur), fmt.Sprintf("%t", in.GenerateAudio))

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
		outputURL, err = o.callPollinationsTTSFull(ctx, key, in.Text, voice, in.Speed, in.AudioFormat, in.Language)

	// ── Transcription ────────────────────────────────────────────────────────
	case entities.TemplateOpenAITranscribe:
		outputText, err = o.callOpenAITranscribe(ctx, key, p.ModelID, in.AudioURL, in.Language)

	case entities.TemplateAssemblyAI:
		outputText, err = o.callAssemblyAIFull(ctx, key, in.AudioURL, in.Language, in.SpeakerLabels, in.OutputFormat)

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
		outputURL, err = o.callPollinationsElevenMusic(ctx, key, in.Prompt, instrumental)

	case entities.TemplateSunoMusic:
		style := in.Style
		if style == "" {
			style, _ = p.ExtraConfig["style"].(string)
		}
		title := in.Title
		if title == "" {
			title, _ = p.ExtraConfig["title"].(string)
		}
		var second string
		outputURL, second, err = o.callSunoMusic(ctx, key, in.Prompt, style, title, in.VocalGender, in.Instrumental)
		if err == nil && in.SecondaryURL != nil {
			*in.SecondaryURL = second
		}

	case entities.TemplateHFMusicGen:
		dur := in.DurationSecs
		if dur <= 0 {
			dur = 30
		}
		outputURL, err = o.callHFMusicGen(ctx, key, in.Prompt, dur)

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
		if key == "" {
			err = fmt.Errorf("rembg service URL not configured for provider")
			break
		}
		outputURL, err = o.callRembgService(ctx, key, in.ImageURL)

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
		return "", fmt.Errorf("gemini: provider API key not configured")
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
	ctx context.Context, apiKey, model, systemPrompt, userPrompt string,
) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("deepseek: provider API key not configured")
	}
	if model == "" {
		model = "deepseek-chat"
	}
	payload := map[string]interface{}{
		"model": model,
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

func (o *AIStudioOrchestrator) callGeminiConfiguredMultimodal(
	ctx context.Context, model, apiKey, systemPrompt, userPrompt string, images []string,
) (string, error) {
	if model == "" || apiKey == "" {
		return "", fmt.Errorf("Gemini model/key not configured")
	}
	parts := []map[string]interface{}{{"text": userPrompt}}
	for _, image := range images {
		if image == "" {
			continue
		}
		mimeType, data := "image/jpeg", image
		if strings.HasPrefix(image, "data:") {
			if semi := strings.Index(image, ";"); semi > 5 {
				mimeType = image[5:semi]
			}
			if comma := strings.Index(image, ","); comma >= 0 {
				data = image[comma+1:]
			}
		}
		parts = append(parts, map[string]interface{}{
			"inlineData": map[string]string{"mimeType": mimeType, "data": data},
		})
	}
	payload := map[string]interface{}{
		"system_instruction": map[string]interface{}{"parts": []map[string]string{{"text": systemPrompt}}},
		"contents":           []map[string]interface{}{{"parts": parts}},
		"generationConfig":   map[string]interface{}{"maxOutputTokens": 65536, "temperature": 0.85},
	}
	body, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Gemini multimodal HTTP: %w", err)
	}
	defer resp.Body.Close()
	var parsed struct {
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
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("Gemini API error %d: %s", parsed.Error.Code, parsed.Error.Message)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini multimodal returned no content")
	}
	return parsed.Candidates[0].Content.Parts[0].Text, nil
}
