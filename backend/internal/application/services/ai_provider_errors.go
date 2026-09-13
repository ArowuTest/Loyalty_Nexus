package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// ProviderRefusalError marks a response the provider declined on content
// grounds — a safety block, a content filter, a recitation stop. It is a
// verdict about the prompt, not about the provider, so the router must never
// treat it as a reason to try the next candidate: classifyRoutingError maps it
// to POLICY_REJECTION with no failover. It is detected structurally from the
// provider's response (block reasons, finish reasons, error codes), never by
// sniffing error text. Before this existed every such outcome collapsed into
// "no content returned", which was classified as a retryable provider fault
// and shopped the same prompt to the next provider (review M7).
type ProviderRefusalError struct {
	Provider string // which adapter refused: "gemini", "openai-compatible"
	Reason   string // the provider's own code, e.g. SAFETY, content_filter
}

func (e *ProviderRefusalError) Error() string {
	return fmt.Sprintf("%s: content refused by provider (%s)", e.Provider, e.Reason)
}

func isProviderRefusal(err error) bool {
	var r *ProviderRefusalError
	return errors.As(err, &r)
}

// geminiRefusalFinishReasons are the finishReason values with which Gemini
// stops a candidate on content grounds. Anything else (STOP, MAX_TOKENS,
// MALFORMED_FUNCTION_CALL, ...) is not a refusal.
var geminiRefusalFinishReasons = map[string]bool{
	"SAFETY": true, "RECITATION": true, "BLOCKLIST": true,
	"PROHIBITED_CONTENT": true, "SPII": true, "IMAGE_SAFETY": true,
}

type geminiCandidate struct {
	FinishReason string `json:"finishReason"`
	Content      struct {
		Parts []struct {
			Text    string `json:"text"`
			Thought bool   `json:"thought"`
		} `json:"parts"`
	} `json:"content"`
}

type geminiGenerateContentResponse struct {
	Candidates     []geminiCandidate `json:"candidates"`
	PromptFeedback *struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// geminiCandidateText concatenates a candidate's visible text parts; thought
// parts (reasoning models) are never user-facing output.
func geminiCandidateText(c geminiCandidate) string {
	var b strings.Builder
	for _, part := range c.Content.Parts {
		if part.Thought {
			continue
		}
		b.WriteString(part.Text)
	}
	return b.String()
}

// decodeGeminiGenerateContent turns a generateContent HTTP response into text
// or a precise error: API error envelopes keep their status code (so 401/403/
// 429 classify as before), a prompt-level block or a content-grounded
// finishReason becomes a ProviderRefusalError, and only a genuinely empty
// successful response is reported as "no content".
func decodeGeminiGenerateContent(statusCode int, raw []byte) (string, error) {
	var parsed geminiGenerateContentResponse
	jsonErr := json.Unmarshal(raw, &parsed)
	if jsonErr == nil && parsed.Error != nil {
		return "", fmt.Errorf("gemini API error %d (%s): %s", parsed.Error.Code, parsed.Error.Status, parsed.Error.Message)
	}
	if statusCode != http.StatusOK {
		return "", fmt.Errorf("gemini http %d: %s", statusCode, truncateStr(string(raw), 300))
	}
	if jsonErr != nil {
		return "", fmt.Errorf("gemini decode: %w", jsonErr)
	}
	if parsed.PromptFeedback != nil && parsed.PromptFeedback.BlockReason != "" {
		return "", &ProviderRefusalError{Provider: "gemini", Reason: "prompt blocked: " + parsed.PromptFeedback.BlockReason}
	}
	if len(parsed.Candidates) == 0 {
		return "", fmt.Errorf("gemini: no candidates returned")
	}
	cand := parsed.Candidates[0]
	if geminiRefusalFinishReasons[cand.FinishReason] {
		// A partial answer cut off on content grounds is still a refusal.
		return "", &ProviderRefusalError{Provider: "gemini", Reason: "candidate finished: " + cand.FinishReason}
	}
	text := geminiCandidateText(cand)
	if text == "" {
		if cand.FinishReason != "" && cand.FinishReason != "STOP" {
			return "", fmt.Errorf("gemini: no content returned (finishReason=%s)", cand.FinishReason)
		}
		return "", fmt.Errorf("gemini: no content returned")
	}
	return text, nil
}

// openAIErrorRefusal recognises the OpenAI-compatible error envelope for a
// content-policy rejection — {"error":{"code":"content_policy_violation"}},
// a content_filter code, or a moderation type — which providers send with a
// 400 status even though it is a verdict about the prompt, not a bad request.
func openAIErrorRefusal(raw []byte) (*ProviderRefusalError, bool) {
	var env struct {
		Error *struct {
			Code    interface{} `json:"code"` // string on OpenAI, integer on some clones
			Type    string      `json:"type"`
			Message string      `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &env) != nil || env.Error == nil {
		return nil, false
	}
	signal := strings.ToLower(fmt.Sprint(env.Error.Code) + " " + env.Error.Type)
	for _, marker := range []string{"content_policy", "content_filter", "content_management_policy", "moderation"} {
		if strings.Contains(signal, marker) {
			return &ProviderRefusalError{Provider: "openai-compatible", Reason: fmt.Sprint(env.Error.Code)}, true
		}
	}
	return nil, false
}
