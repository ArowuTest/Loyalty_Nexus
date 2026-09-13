package services

// ai_provider_dispatch.go — Admin-configured provider dispatch (AI Routing V2)
//
//	Admin UI → ai_provider_configs + ai_tool_stages + ai_tool_provider_bindings
//	     ↓
//	runToolStageChain / streamToolStageChain (ai_router_v2.go, ai_streaming.go)
//	     ↓  ordered candidates, circuit + capacity + budget reserved per attempt
//	callByTemplate(ctx, p, in)   ← routes provider.template → the matching adapter
//
// There is deliberately no fallback outside the configured route: an empty or
// exhausted route is ErrNoConfiguredAIRoute, an unknown template is an error,
// and no adapter reads a provider key from the environment or hard-codes a
// model. A tool that must always answer is configured with a last-resort
// binding in the Admin UI, where it is visible and auditable.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
			err = errModelIDRequired(p)
			break
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
			err = errModelIDRequired(p) // silently picking the paid Kling Pro model here is exactly the hidden fallback V2 forbids
			break
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
			err = errModelIDRequired(p)
			break
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
		// Single fixed FAL endpoint; there is no model to choose.
		outputURL, err = o.callFALBgRemover(ctx, key, in.ImageURL)

	case entities.TemplateRemoveBG:
		outputURL, err = o.callRemoveBg(ctx, key, in.ImageURL)

	default:
		err = fmt.Errorf("unknown template %q for provider %q", p.Template, p.Slug)
	}
	return
}

// errModelIDRequired is the answer to an empty model_id on a template where the
// model matters: a configuration error surfaced to the Admin, never a model
// picked in code (that would be a hidden fallback — invisible, unaudited, and
// on some templates a paid one).
func errModelIDRequired(p entities.AIProviderConfig) error {
	return fmt.Errorf("%s (%s): model_id not configured — set it on the provider", p.Slug, p.Template)
}

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

// callGeminiFlashWithModel calls Gemini with the admin-configured model ID and
// the resolved key. An empty model_id is a configuration error, not an excuse
// to pick a model in code: the same rule the streaming path enforces.
func (o *AIStudioOrchestrator) callGeminiFlashWithModel(
	ctx context.Context, model, apiKey, systemPrompt, userPrompt string,
) (string, error) {
	if model == "" {
		return "", fmt.Errorf("gemini: model_id not configured for this provider")
	}
	if apiKey == "" {
		return "", fmt.Errorf("gemini: provider API key not configured")
	}

	endpoint := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent",
		model,
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
	return o.callGeminiEndpoint(ctx, endpoint, apiKey, payload)
}

// callGeminiEndpoint POSTs a generateContent payload to any Gemini model
// endpoint. The key travels in the x-goog-api-key header, never the query
// string (URLs are logged by proxies and error text), and the response is
// decoded structurally so blocks and content stops surface as refusals.
func (o *AIStudioOrchestrator) callGeminiEndpoint(
	ctx context.Context, endpoint, apiKey string, payload map[string]interface{},
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
	req.Header.Set("x-goog-api-key", apiKey)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", fmt.Errorf("gemini read: %w", err)
	}
	return decodeGeminiGenerateContent(resp.StatusCode, raw)
}

// callDeepSeekWithKey calls DeepSeek with an explicit API key.
func (o *AIStudioOrchestrator) callDeepSeekWithKey(
	ctx context.Context, apiKey, model, systemPrompt, userPrompt string,
) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("deepseek: provider API key not configured")
	}
	if model == "" {
		return "", fmt.Errorf("deepseek: model_id not configured for this provider")
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
	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)
	return o.callGeminiEndpoint(ctx, endpoint, apiKey, payload)
}
