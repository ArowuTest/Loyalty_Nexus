package services

import (
	"context"
	"encoding/base64"
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
	// Shared Gemini call: key in the x-goog-api-key header, structural decode so
	// a blocked or content-stopped answer is a refusal rather than "no content".
	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)
	return o.callGeminiEndpoint(ctx, endpoint, apiKey, payload)
}
