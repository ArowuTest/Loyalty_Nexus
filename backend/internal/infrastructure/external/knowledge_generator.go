package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ─── GeminiKnowledgeAdapter ───────────────────────────────────────────────

// GeminiKnowledgeAdapter implements KnowledgeGenerator using Gemini 2.0 Flash.
// Generation is synchronous; results are cached in-memory under a jobID so
// callers can use the same Generate → PollStatus contract regardless.
type GeminiKnowledgeAdapter struct {
	APIKey string
	client *http.Client
	store  sync.Map // jobID → string (output text)
}

// NewGeminiKnowledgeAdapter returns a ready-to-use adapter. If apiKey is empty,
// os.Getenv("GEMINI_API_KEY") should be set by the caller before constructing.
func NewGeminiKnowledgeAdapter(apiKey string) *GeminiKnowledgeAdapter {
	return &GeminiKnowledgeAdapter{
		APIKey: apiKey,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// Generate calls Gemini synchronously, stores the output in memory, and returns
// a jobID immediately. Sources are concatenated into the prompt when provided.
func (a *GeminiKnowledgeAdapter) Generate(ctx context.Context, topic, toolType string, sources []string) (string, error) {
	prompt := buildKnowledgePrompt(topic, toolType, sources)
	text, err := a.callGeminiKnowledge(ctx, prompt)
	if err != nil {
		return "", err
	}

	jobID := uuid.New().String()
	a.store.Store(jobID, text)
	return jobID, nil
}

// PollStatus checks the in-memory store. Since generation is synchronous the
// result is always ready after Generate returns.
func (a *GeminiKnowledgeAdapter) PollStatus(ctx context.Context, jobID string) (bool, string, error) {
	val, ok := a.store.Load(jobID)
	if !ok {
		return false, "", nil
	}
	text, _ := val.(string)
	return true, text, nil
}

// callGeminiKnowledge sends a single-turn message to Gemini and returns the
// generated text.
func (a *GeminiKnowledgeAdapter) callGeminiKnowledge(ctx context.Context, prompt string) (string, error) {
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

	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=" + a.APIKey
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini knowledge request: %w", err)
	}
	defer resp.Body.Close()

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
	text := strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text), nil
}

// buildKnowledgePrompt returns the appropriate Gemini prompt for each toolType.
func buildKnowledgePrompt(topic, toolType string, sources []string) string {
	sourceNote := ""
	if len(sources) > 0 {
		sourceNote = fmt.Sprintf(" Use the following reference sources where relevant: %s.", strings.Join(sources, ", "))
	}

	var prompt string
	switch toolType {
	case "study-guide":
		prompt = fmt.Sprintf(
			"Create a comprehensive study guide for: %s. "+
				"Include key concepts, definitions, examples, and practice questions. "+
				"Format with clear sections.",
			topic)
	case "quiz":
		prompt = fmt.Sprintf(
			"Create 10 quiz questions with answers about: %s. "+
				"Format as JSON: [{\"question\": \"\", \"options\": [\"a\",\"b\",\"c\",\"d\"], \"answer\": \"\", \"explanation\": \"\"}]",
			topic)
	case "mindmap":
		prompt = fmt.Sprintf(
			"Create a detailed mind map structure for: %s. "+
				"Format as JSON: {\"center\": \"\", \"branches\": [{\"label\": \"\", \"children\": [{\"label\": \"\"}]}]}",
			topic)
	case "research-brief":
		prompt = fmt.Sprintf(
			"Write a comprehensive research brief about: %s. "+
				"Include executive summary, key findings, methodology notes, and recommendations.",
			topic)
	case "podcast":
		prompt = fmt.Sprintf(
			"Create a podcast script (two hosts: Nexus and Ade) discussing: %s. "+
				"Include intro, 3 main discussion points, and outro. "+
				"Make it engaging and Nigerian-context aware.",
			topic)
	case "slide-deck":
		prompt = fmt.Sprintf(
			"Create a slide deck outline for: %s. "+
				"Return JSON: {\"title\": \"\", \"slides\": [{\"title\": \"\", \"content\": \"\", \"notes\": \"\"}]}",
			topic)
	case "infographic":
		prompt = fmt.Sprintf(
			"Create an infographic content structure for: %s. "+
				"Return JSON: {\"title\": \"\", \"sections\": [{\"heading\": \"\", \"stats\": [{\"label\": \"\", \"value\": \"\"}], \"bullets\": []}]}",
			topic)
	case "bizplan":
		prompt = fmt.Sprintf(
			"Write a complete business plan for: %s. "+
				"Sections: Executive Summary, Market Analysis, Products/Services, "+
				"Marketing Strategy, Financial Projections, Operations Plan.",
			topic)
	default:
		prompt = fmt.Sprintf("Generate comprehensive content about: %s", topic)
	}

	return prompt + sourceNote
}

// ─── NotebookLMAdapter (deprecated — wraps GeminiKnowledgeAdapter) ────────

// NotebookLMAdapter is a deprecated type alias kept for backwards compatibility.
// New code should use GeminiKnowledgeAdapter directly.
type NotebookLMAdapter struct {
	*GeminiKnowledgeAdapter
}

// NewNotebookLMAdapter returns a NotebookLMAdapter backed by Gemini.
func NewNotebookLMAdapter(apiKey string) *NotebookLMAdapter {
	return &NotebookLMAdapter{GeminiKnowledgeAdapter: NewGeminiKnowledgeAdapter(apiKey)}
}

// TriggerGeneration is a legacy method. Use Generate instead.
func (a *NotebookLMAdapter) TriggerGeneration(ctx context.Context, topic, toolType string) (string, error) {
	return a.Generate(ctx, topic, toolType, nil)
}
