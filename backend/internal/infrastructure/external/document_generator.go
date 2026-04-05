package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ─── Interfaces re-declared locally (already in document_generator.go) ───
// DocumentGenerator and AudioTranscriber are kept here so this file stays
// self-contained, but the types below satisfy them.

// DocumentGenerator generates structured documents using an AI backend.
type DocumentGenerator interface {
	GenerateSlideDeck(ctx context.Context, topic string) (string, error)
	GenerateBusinessPlan(ctx context.Context, description string) (string, error)
}

// AudioTranscriber converts audio to text.
type AudioTranscriber interface {
	Transcribe(ctx context.Context, audioURL string) (string, error)
}

// ─── AssemblyAI ───────────────────────────────────────────────────────────

// AssemblyAIAdapter transcribes audio via the AssemblyAI REST API.
type AssemblyAIAdapter struct {
	APIKey string
	client *http.Client
}

// NewAssemblyAIAdapter returns an adapter backed by the given API key.
func NewAssemblyAIAdapter(apiKey string) *AssemblyAIAdapter {
	return &AssemblyAIAdapter{
		APIKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Transcribe submits audioURL for transcription and polls until the job
// completes or fails. Maximum wait time is 5 minutes.
func (a *AssemblyAIAdapter) Transcribe(ctx context.Context, audioURL string) (string, error) {
	// Submit transcript job
	jobID, err := a.submitTranscript(ctx, audioURL)
	if err != nil {
		return "", err
	}

	// Poll every 2 seconds for up to 5 minutes
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(2 * time.Second):
		}

		text, done, err := a.pollTranscript(ctx, jobID)
		if err != nil {
			return "", err
		}
		if done {
			return text, nil
		}
	}
	return "", fmt.Errorf("assemblyai: transcription timed out after 5 minutes")
}

func (a *AssemblyAIAdapter) submitTranscript(ctx context.Context, audioURL string) (string, error) {
	payload := map[string]string{
		"audio_url":     audioURL,
		"language_code": "en",
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.assemblyai.com/v2/transcript", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", a.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("assemblyai submit: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("assemblyai submit returned %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("assemblyai: parse submit response: %w", err)
	}
	if result.ID == "" {
		return "", fmt.Errorf("assemblyai: empty job id")
	}
	return result.ID, nil
}

func (a *AssemblyAIAdapter) pollTranscript(ctx context.Context, jobID string) (text string, done bool, err error) {
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.assemblyai.com/v2/transcript/"+jobID, nil)
	if reqErr != nil {
		return "", false, reqErr
	}
	req.Header.Set("Authorization", a.APIKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return "", false, fmt.Errorf("assemblyai poll: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", false, fmt.Errorf("assemblyai poll returned %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Status string `json:"status"`
		Text   string `json:"text"`
		Error  string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", false, fmt.Errorf("assemblyai: parse poll response: %w", err)
	}

	switch result.Status {
	case "completed":
		return result.Text, true, nil
	case "error":
		return "", false, fmt.Errorf("assemblyai: transcription error: %s", result.Error)
	default:
		return "", false, nil
	}
}

// ─── GoogleTranslateAdapter ───────────────────────────────────────────────

// GoogleTranslateAdapter translates text via the Google Cloud Translation API.
// Supported target languages: "ha" (Hausa), "yo" (Yoruba), "ig" (Igbo),
// "en" (English), "fr" (French).
type GoogleTranslateAdapter struct {
	APIKey string
	client *http.Client
}

// NewGoogleTranslateAdapter returns an adapter using the given API key.
// If apiKey is empty the GOOGLE_TRANSLATE_API_KEY env var is used.
func NewGoogleTranslateAdapter(apiKey string) *GoogleTranslateAdapter {
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_TRANSLATE_API_KEY")
	}
	return &GoogleTranslateAdapter{
		APIKey: apiKey,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Translate translates text into targetLang using the Google Translate REST API.
func (a *GoogleTranslateAdapter) Translate(ctx context.Context, text, targetLang string) (string, error) {
	payload := map[string]interface{}{
		"q":      text,
		"target": targetLang,
		"format": "text",
	}
	body, _ := json.Marshal(payload)

	url := "https://translation.googleapis.com/language/translate/v2?key=" + a.APIKey
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("google translate request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("google translate returned %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Data struct {
			Translations []struct {
				TranslatedText string `json:"translatedText"`
			} `json:"translations"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("google translate: parse response: %w", err)
	}
	if len(result.Data.Translations) == 0 {
		return "", fmt.Errorf("google translate: no translations returned")
	}
	return result.Data.Translations[0].TranslatedText, nil
}

// ─── DocumentAutomationAdapter ────────────────────────────────────────────

// DocumentAutomationAdapter implements DocumentGenerator via Gemini 2.0 Flash.
type DocumentAutomationAdapter struct {
	GeminiAPIKey string
	client       *http.Client
}

// NewDocumentAutomationAdapter returns an adapter. If geminiAPIKey is empty,
// GEMINI_API_KEY env var is used.
func NewDocumentAutomationAdapter(geminiAPIKey string) *DocumentAutomationAdapter {
	if geminiAPIKey == "" {
		geminiAPIKey = os.Getenv("GEMINI_API_KEY")
	}
	return &DocumentAutomationAdapter{
		GeminiAPIKey: geminiAPIKey,
		client:       &http.Client{Timeout: 60 * time.Second},
	}
}

// GenerateSlideDeck calls Gemini to produce a structured slide deck outline.
// The result is a JSON string stored as output_text, not a file URL.
func (a *DocumentAutomationAdapter) GenerateSlideDeck(ctx context.Context, topic string) (string, error) {
	prompt := fmt.Sprintf(
		"Generate a detailed slide deck outline for: %s. "+
			"Format as JSON with slides array containing {title, bullets[]}. "+
			"Return ONLY the JSON.",
		topic)
	return a.callGemini(ctx, prompt)
}

// GenerateBusinessPlan calls Gemini to produce a complete business plan text.
func (a *DocumentAutomationAdapter) GenerateBusinessPlan(ctx context.Context, description string) (string, error) {
	prompt := fmt.Sprintf(
		"Write a detailed and comprehensive business plan for the following:\n\n%s\n\n"+
			"Include these sections: Executive Summary, Market Analysis, Products/Services, "+
			"Marketing Strategy, Financial Projections, Operations Plan. "+
			"Be thorough and professional.",
		description)
	return a.callGemini(ctx, prompt)
}

// callGemini sends a single-turn user message to Gemini 2.0 Flash and returns
// the generated text.
func (a *DocumentAutomationAdapter) callGemini(ctx context.Context, prompt string) (string, error) {
	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
	}
	body, _ := json.Marshal(payload)

	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=" + a.GeminiAPIKey
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("gemini returned %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("gemini: parse response: %w", err)
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: empty response")
	}

	text := result.Candidates[0].Content.Parts[0].Text
	// Trim markdown code fences if Gemini wraps JSON in ```json ... ```
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text), nil
}
