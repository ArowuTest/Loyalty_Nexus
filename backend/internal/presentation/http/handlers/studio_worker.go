package handlers

// studio_worker.go — background AI generation processor
// Dispatched by StudioHandler.Generate(); runs after HTTP response is returned.
// Each tool category calls the appropriate external API adapter.

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
)

// ────────────────────────────────────────────────────────────────────────────
// Entry-point — dispatched by StudioHandler
// ────────────────────────────────────────────────────────────────────────────

func (h *StudioHandler) processGeneration(genID uuid.UUID, toolName, prompt string, extra map[string]interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var (
		resultURL string
		resultText string
		err        error
	)

	tool := strings.ToLower(toolName)
	switch {
	// ── Text / LLM tools ────────────────────────────────────────────────────
	case contains(tool, "business-plan", "business_plan", "business plan"):
		resultText, err = h.runGroqChat(ctx, prompt, "llama3-70b-8192")
	case contains(tool, "story", "creative"):
		resultText, err = h.runGroqChat(ctx, prompt, "llama3-70b-8192")
	case contains(tool, "pitch", "startup"):
		resultText, err = h.runGroqChat(ctx, prompt, "llama3-70b-8192")
	case contains(tool, "recipe"):
		resultText, err = h.runGroqChat(ctx, "Generate a complete Nigerian recipe for: "+prompt, "llama3-70b-8192")
	case contains(tool, "cv", "resume"):
		resultText, err = h.runGroqChat(ctx, "Generate a professional CV for: "+prompt, "llama3-70b-8192")
	case contains(tool, "translate"):
		resultText, err = h.runGeminiText(ctx, prompt)
	case contains(tool, "trivia", "quiz"):
		resultText, err = h.runGroqChat(ctx, "Generate 5 trivia questions about: "+prompt, "llama3-8b-8192")

	// ── Voice / Audio tools ──────────────────────────────────────────────────
	case contains(tool, "voice", "speech", "tts"):
		resultURL, err = h.runVoiceTTS(ctx, prompt)
	case contains(tool, "jingle", "music"):
		resultURL, err = h.runMubertJingle(ctx, prompt, extra)

	// ── Image tools ──────────────────────────────────────────────────────────
	case contains(tool, "avatar", "photo", "image", "portrait"):
		resultURL, err = h.runHuggingFaceImage(ctx, prompt)
	case contains(tool, "background", "remove", "rembg"):
		if src, ok := extra["source_url"].(string); ok {
			resultURL, err = h.runRembg(ctx, src)
		} else {
			err = fmt.Errorf("source_url required for background removal")
		}

	// ── Video tools ──────────────────────────────────────────────────────────
	case contains(tool, "video", "animation"):
		resultURL, err = h.runFALVideo(ctx, prompt, extra)

	// ── Vision tools ─────────────────────────────────────────────────────────
	case contains(tool, "scan", "document", "vision", "ocr"):
		if src, ok := extra["source_url"].(string); ok {
			resultText, err = h.runGeminiVision(ctx, src, prompt)
		} else {
			resultText, err = h.runGeminiText(ctx, prompt)
		}

	// ── Voice-to-plan ─────────────────────────────────────────────────────────
	case contains(tool, "voice-to-plan", "voiceplan"):
		resultText, err = h.runVoiceToPlan(ctx, extra)

	default:
		// Fallback to Groq for unknown text tools
		resultText, err = h.runGroqChat(ctx, prompt, "llama3-8b-8192")
	}

	if err != nil {
		log.Printf("[studio_worker] gen %s failed: %v", genID, err)
		if sErr := h.studioSvc.FailGeneration(ctx, genID, err.Error()); sErr != nil {
			log.Printf("[studio_worker] FailGeneration: %v", sErr)
		}
		return
	}

	out := resultURL
	if out == "" {
		out = resultText
	}
	if err := h.studioSvc.CompleteGeneration(ctx, genID, out, "", 0); err != nil {
		log.Printf("[studio_worker] CompleteGeneration: %v", err)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Groq — fast LLM inference (free tier, 6000 req/min)
// ────────────────────────────────────────────────────────────────────────────

type groqRequest struct {
	Model    string        `json:"model"`
	Messages []groqMessage `json:"messages"`
}
type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type groqResponse struct {
	Choices []struct {
		Message groqMessage `json:"message"`
	} `json:"choices"`
}

func (h *StudioHandler) runGroqChat(ctx context.Context, prompt, model string) (string, error) {
	key := os.Getenv("GROQ_API_KEY")
	if key == "" {
		return "⚠ GROQ_API_KEY not configured. Text generation unavailable.", nil
	}
	body, _ := json.Marshal(groqRequest{
		Model: model,
		Messages: []groqMessage{
			{Role: "system", Content: "You are Nexus, a creative AI assistant for Nigerian users. Be concise and culturally relevant."},
			{Role: "user", Content: prompt},
		},
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("groq request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("groq %d: %s", resp.StatusCode, string(b))
	}
	var gr groqResponse
	json.NewDecoder(resp.Body).Decode(&gr)
	if len(gr.Choices) == 0 {
		return "", fmt.Errorf("groq empty response")
	}
	return gr.Choices[0].Message.Content, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Google Gemini — vision + translation (free tier)
// ────────────────────────────────────────────────────────────────────────────

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}
type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}
type geminiPart struct {
	Text       string          `json:"text,omitempty"`
	InlineData *geminiInline   `json:"inlineData,omitempty"`
}
type geminiInline struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}
type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

func (h *StudioHandler) runGeminiText(ctx context.Context, prompt string) (string, error) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		return h.runGroqChat(ctx, prompt, "llama3-8b-8192")
	}
	body, _ := json.Marshal(geminiRequest{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: prompt}}}},
	})
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + key
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var gr geminiResponse
	json.NewDecoder(resp.Body).Decode(&gr)
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini empty response")
	}
	return gr.Candidates[0].Content.Parts[0].Text, nil
}

func (h *StudioHandler) runGeminiVision(ctx context.Context, imageURL, prompt string) (string, error) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		return "⚠ GEMINI_API_KEY not configured for vision tasks.", nil
	}
	// Download image
	imgResp, err := http.Get(imageURL)
	if err != nil {
		return "", err
	}
	defer imgResp.Body.Close()
	imgData, _ := io.ReadAll(imgResp.Body)

	// base64 encode
	encoded := base64.StdEncoding.EncodeToString(imgData)

	body, _ := json.Marshal(geminiRequest{
		Contents: []geminiContent{{Parts: []geminiPart{
			{InlineData: &geminiInline{MimeType: "image/jpeg", Data: encoded}},
			{Text: prompt},
		}}},
	})
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + key
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var gr geminiResponse
	json.NewDecoder(resp.Body).Decode(&gr)
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini vision empty response")
	}
	return gr.Candidates[0].Content.Parts[0].Text, nil
}

// ────────────────────────────────────────────────────────────────────────────
// HuggingFace Serverless Inference — Stable Diffusion image
// ────────────────────────────────────────────────────────────────────────────

func (h *StudioHandler) runHuggingFaceImage(ctx context.Context, prompt string) (string, error) {
	token := os.Getenv("HF_TOKEN")
	if token == "" {
		return "https://placehold.co/512x512/1c2038/5f72f9?text=HF_TOKEN+required", nil
	}
	model := os.Getenv("HF_IMAGE_MODEL")
	if model == "" {
		model = "black-forest-labs/FLUX.1-schnell"
	}
	url := fmt.Sprintf("https://api-inference.huggingface.co/models/%s", model)
	body, _ := json.Marshal(map[string]string{"inputs": prompt})
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HF %d: %s", resp.StatusCode, string(b))
	}
	// Upload to CDN / storage; for now return a data URI placeholder
	imgBytes, _ := io.ReadAll(resp.Body)
	if len(imgBytes) > 0 {
		// Save to a temp file and return the storage URL
		fname := fmt.Sprintf("/tmp/gen_%d.png", time.Now().UnixNano())
		os.WriteFile(fname, imgBytes, 0644)
		// TODO: upload to Cloudflare R2 / S3 and return public URL
		return fname, nil
	}
	return "", fmt.Errorf("HF returned empty image")
}

// ────────────────────────────────────────────────────────────────────────────
// FAL.AI — video generation
// ────────────────────────────────────────────────────────────────────────────

func (h *StudioHandler) runFALVideo(ctx context.Context, prompt string, extra map[string]interface{}) (string, error) {
	key := os.Getenv("FAL_AI_KEY")
	if key == "" {
		return "https://placehold.co/640x360/1c2038/5f72f9?text=FAL_AI_KEY+required", nil
	}
	payload := map[string]interface{}{
		"prompt":    prompt,
		"num_frames": 24,
		"fps":        8,
	}
	if src, ok := extra["source_image_url"].(string); ok {
		payload["image_url"] = src
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://fal.run/fal-ai/stable-video-diffusion", bytes.NewReader(body))
	req.Header.Set("Authorization", "Key "+key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if url, ok := result["video"].(map[string]interface{})["url"].(string); ok {
		return url, nil
	}
	return "", fmt.Errorf("FAL video no URL in response")
}

// ────────────────────────────────────────────────────────────────────────────
// Mubert — AI jingle generation
// ────────────────────────────────────────────────────────────────────────────

func (h *StudioHandler) runMubertJingle(ctx context.Context, prompt string, extra map[string]interface{}) (string, error) {
	key := os.Getenv("MUBERT_API_KEY")
	if key == "" {
		return "https://placehold.co/audio?text=MUBERT_API_KEY+required", nil
	}
	duration := 15
	if d, ok := extra["duration_seconds"].(float64); ok {
		duration = int(d)
	}
	body, _ := json.Marshal(map[string]interface{}{
		"method": "RecordTrackTTM",
		"params": map[string]interface{}{
			"pat":      key,
			"prompt":   prompt,
			"duration": duration,
			"format":   "mp3",
			"intensity": "medium",
		},
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api-b2b.mubert.com/v2/RecordTrackTTM", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if data, ok := result["data"].(map[string]interface{}); ok {
		if tracks, ok := data["tasks"].([]interface{}); ok && len(tracks) > 0 {
			if t, ok := tracks[0].(map[string]interface{}); ok {
				if url, ok := t["download_link"].(string); ok {
					return url, nil
				}
			}
		}
	}
	return "", fmt.Errorf("mubert no download_link in response")
}

// ────────────────────────────────────────────────────────────────────────────
// Voice TTS — ElevenLabs or Google Cloud TTS
// ────────────────────────────────────────────────────────────────────────────

func (h *StudioHandler) runVoiceTTS(ctx context.Context, text string) (string, error) {
	key := os.Getenv("ELEVENLABS_API_KEY")
	if key == "" {
		return "https://placehold.co/audio?text=ELEVENLABS_API_KEY+required", nil
	}
	voiceID := os.Getenv("ELEVENLABS_VOICE_ID")
	if voiceID == "" {
		voiceID = "21m00Tcm4TlvDq8ikWAM" // Rachel (default)
	}
	body, _ := json.Marshal(map[string]interface{}{
		"text":     text,
		"model_id": "eleven_turbo_v2_5",
		"voice_settings": map[string]float64{"stability": 0.5, "similarity_boost": 0.75},
	})
	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s", voiceID)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("xi-api-key", key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("elevenlabs %d: %s", resp.StatusCode, string(b))
	}
	audioData, _ := io.ReadAll(resp.Body)
	fname := fmt.Sprintf("/tmp/tts_%d.mp3", time.Now().UnixNano())
	os.WriteFile(fname, audioData, 0644)
	// TODO: upload to object storage
	return fname, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Voice-to-Plan — AssemblyAI transcription → Groq business plan
// ────────────────────────────────────────────────────────────────────────────

func (h *StudioHandler) runVoiceToPlan(ctx context.Context, extra map[string]interface{}) (string, error) {
	audioURL, _ := extra["audio_url"].(string)
	if audioURL == "" {
		return "", fmt.Errorf("audio_url required for voice-to-plan")
	}
	aaiKey := os.Getenv("ASSEMBLY_AI_KEY")
	if aaiKey == "" {
		return "⚠ ASSEMBLY_AI_KEY not configured for transcription.", nil
	}
	// Submit transcription
	body, _ := json.Marshal(map[string]string{"audio_url": audioURL})
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.assemblyai.com/v2/transcript", bytes.NewReader(body))
	req.Header.Set("Authorization", aaiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var sub map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&sub)
	txID, _ := sub["id"].(string)
	if txID == "" {
		return "", fmt.Errorf("assemblyai no transcript id")
	}
	// Poll until completed
	for i := 0; i < 30; i++ {
		time.Sleep(3 * time.Second)
		pollReq, _ := http.NewRequestWithContext(ctx, "GET", "https://api.assemblyai.com/v2/transcript/"+txID, nil)
		pollReq.Header.Set("Authorization", aaiKey)
		pollResp, err := http.DefaultClient.Do(pollReq)
		if err != nil {
			return "", err
		}
		var pollData map[string]interface{}
		json.NewDecoder(pollResp.Body).Decode(&pollData)
		pollResp.Body.Close()
		status, _ := pollData["status"].(string)
		if status == "completed" {
			transcript, _ := pollData["text"].(string)
			planPrompt := fmt.Sprintf("Based on this voice note, generate a structured business plan: %s", transcript)
			return h.runGroqChat(ctx, planPrompt, "llama3-70b-8192")
		}
		if status == "error" {
			return "", fmt.Errorf("assemblyai error: %v", pollData["error"])
		}
	}
	return "", fmt.Errorf("assemblyai transcription timeout")
}

// ────────────────────────────────────────────────────────────────────────────
// rembg — background removal (self-hosted or Remove.bg API)
// ────────────────────────────────────────────────────────────────────────────

func (h *StudioHandler) runRembg(ctx context.Context, imageURL string) (string, error) {
	// Try Remove.bg API first
	key := os.Getenv("REMOVEBG_API_KEY")
	if key != "" {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		mw.WriteField("image_url", imageURL)
		mw.WriteField("size", "auto")
		mw.Close()
		req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.remove.bg/v1.0/removebg", &buf)
		req.Header.Set("X-Api-Key", key)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			imgBytes, _ := io.ReadAll(resp.Body)
			fname := fmt.Sprintf("/tmp/rembg_%d.png", time.Now().UnixNano())
			os.WriteFile(fname, imgBytes, 0644)
			return fname, nil
		}
	}
	// Fallback: self-hosted rembg container
	rembgURL := os.Getenv("REMBG_ENDPOINT")
	if rembgURL == "" {
		rembgURL = "http://rembg:7000"
	}
	body, _ := json.Marshal(map[string]string{"url": imageURL})
	req, _ := http.NewRequestWithContext(ctx, "POST", rembgURL+"/api/remove", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("rembg not available: %w", err)
	}
	defer resp.Body.Close()
	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if url, ok := result["url"]; ok {
		return url, nil
	}
	return "", fmt.Errorf("rembg no url in response")
}

// ────────────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────────────

func contains(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// ────────────────────────────────────────────────────────────────────────────
// AsyncStudioWorker — wraps background dispatch logic
// ────────────────────────────────────────────────────────────────────────────

// AsyncStudioWorker dispatches generations to background goroutines.
// The handler parameter is set after construction to avoid circular init.
type AsyncStudioWorker struct {
	handler    *StudioHandler
	studioSvc  interface{ FindGenerationByID(ctx context.Context, id uuid.UUID) (interface{}, error) }
}

// NewAsyncStudioWorker creates a worker.  handler is linked later via LinkHandler().
func NewAsyncStudioWorker(studioSvc interface{}, _ interface{}) *AsyncStudioWorker {
	return &AsyncStudioWorker{}
}

// LinkHandler connects the worker to the concrete StudioHandler.
func (w *AsyncStudioWorker) LinkHandler(h *StudioHandler) { w.handler = h }

// DispatchGeneration runs the AI pipeline asynchronously.
// gen must have ID, ToolName and Prompt fields accessible.
func (w *AsyncStudioWorker) DispatchGeneration(gen interface{}, sources []string) {
	if w.handler == nil {
		log.Println("[AsyncStudioWorker] handler not linked — skipping dispatch")
		return
	}
	type genMinimal interface {
		GetID() uuid.UUID
		GetToolName() string
		GetPrompt() string
	}

	// Use reflection-free approach: accept the entities.AIGeneration struct
	// by wrapping in a generic container
	type genLike struct {
		ID       uuid.UUID
		ToolName string
		Prompt   string
	}

	// Build extra context from sources
	extra := map[string]interface{}{}
	if len(sources) > 0 {
		extra["sources"] = sources
	}

	// Type-assert to map (as returned by service layer)
	if m, ok := gen.(map[string]interface{}); ok {
		id, _ := uuid.Parse(fmt.Sprintf("%v", m["id"]))
		tool, _ := m["tool_name"].(string)
		prompt, _ := m["prompt"].(string)
		go w.handler.processGeneration(id, tool, prompt, extra)
		return
	}

	// Struct type from entities package (used via interface{})
	type typedGen struct {
		ID       uuid.UUID `json:"id"`
		ToolName string    `json:"tool_name"`
		Prompt   string    `json:"prompt"`
	}
	b, _ := json.Marshal(gen)
	var tg typedGen
	if err := json.Unmarshal(b, &tg); err == nil && tg.ID != uuid.Nil {
		go w.handler.processGeneration(tg.ID, tg.ToolName, tg.Prompt, extra)
		return
	}

	log.Printf("[AsyncStudioWorker] could not extract generation fields from type %T", gen)
}
