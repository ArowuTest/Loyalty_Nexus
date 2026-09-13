package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (o *AIStudioOrchestrator) callGeminiConfiguredDocument(
	ctx context.Context,
	model, apiKey, systemPrompt, userPrompt, documentURL string,
) (string, error) {
	if model == "" || apiKey == "" {
		return "", fmt.Errorf("Gemini document model/key not configured")
	}
	docReq, err := http.NewRequestWithContext(ctx, http.MethodGet, documentURL, nil)
	if err != nil {
		return "", err
	}
	docResp, err := o.httpClient.Do(docReq)
	if err != nil {
		return "", fmt.Errorf("document fetch: %w", err)
	}
	defer docResp.Body.Close()
	if docResp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(docResp.Body, 2048))
		return "", fmt.Errorf("document fetch HTTP %d: %s", docResp.StatusCode, truncateStr(string(raw), 300))
	}
	docBytes, err := io.ReadAll(io.LimitReader(docResp.Body, 50<<20))
	if err != nil {
		return "", err
	}
	mimeType := docResp.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		lower := strings.ToLower(documentURL)
		switch {
		case strings.HasSuffix(lower, ".pdf"):
			mimeType = "application/pdf"
		case strings.HasSuffix(lower, ".md"):
			mimeType = "text/markdown"
		case strings.HasSuffix(lower, ".csv"):
			mimeType = "text/csv"
		default:
			mimeType = "text/plain"
		}
	}
	if idx := strings.Index(mimeType, ";"); idx >= 0 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}
	allowed := map[string]bool{
		"application/pdf": true,
		"text/plain":      true,
		"text/markdown":   true,
		"text/html":       true,
		"text/csv":        true,
	}
	if !allowed[mimeType] {
		return "", fmt.Errorf("unsupported document MIME %q", mimeType)
	}
	payload := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]string{{"text": systemPrompt}},
		},
		"contents": []map[string]interface{}{
			{"parts": []map[string]interface{}{
				{"inline_data": map[string]string{
					"mime_type": mimeType,
					"data":      base64.StdEncoding.EncodeToString(docBytes),
				}},
				{"text": userPrompt},
			}},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.7,
			"maxOutputTokens": 8192,
		},
	}
	body, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		model, apiKey,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Gemini document request: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini document HTTP %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}
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
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("Gemini document API error: %s", parsed.Error.Message)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini document returned no content")
	}
	return parsed.Candidates[0].Content.Parts[0].Text, nil
}
