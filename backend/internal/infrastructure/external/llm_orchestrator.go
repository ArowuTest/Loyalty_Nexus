package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"loyalty-nexus/internal/domain/repositories"
	"github.com/google/uuid"
)

type LLMProvider string
const (
	ProviderGroq     LLMProvider = "GROQ"
	ProviderGemini   LLMProvider = "GEMINI"
	ProviderDeepSeek LLMProvider = "DEEPSEEK"
)

type LLMRequest struct {
	UserID  string
	Prompt  string
	History []string
}

type LLMResponse struct {
	Text     string
	Provider LLMProvider
}

type LLMClient interface {
	Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

type UsageTracker interface {
	GetDailyCount(ctx context.Context, userID string) (int, error)
	Increment(ctx context.Context, userID string) error
}

type LLMOrchestrator struct {
	groqClient     LLMClient
	geminiClient   LLMClient
	deepSeekClient LLMClient
	usageTracker   UsageTracker
	chatRepo       repositories.ChatRepository
	groqLimit      int
	geminiLimit    int
}

func NewLLMOrchestrator(g, gem, ds LLMClient, ut UsageTracker, cr repositories.ChatRepository, gLim, gemLim int) *LLMOrchestrator {
	return &LLMOrchestrator{
		groqClient: g, geminiClient: gem, deepSeekClient: ds,
		usageTracker: ut, chatRepo: cr,
		groqLimit: gLim, geminiLimit: gemLim,
	}
}

func (o *LLMOrchestrator) Chat(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	dailyCount, _ := o.usageTracker.GetDailyCount(ctx, req.UserID)

	// Build memory context from session summaries
	uid, _ := uuid.Parse(req.UserID)
	summaries, _ := o.chatRepo.GetLastSummaries(ctx, uid, 3)
	systemPrompt := "You are Nexus, a helpful AI assistant for Loyalty Nexus subscribers in Nigeria. Be concise, friendly and locally aware."
	if len(summaries) > 0 {
		systemPrompt += "\n\nPrevious conversation context:\n" + strings.Join(summaries, "\n")
	}

	// Phase 1: Groq (Primary)
	if dailyCount < o.groqLimit {
		if resp, err := o.groqClient.Complete(ctx, systemPrompt, req.Prompt); err == nil {
			_ = o.usageTracker.Increment(ctx, req.UserID)
			return &LLMResponse{Text: resp, Provider: ProviderGroq}, nil
		} else {
			log.Printf("[LLM] Groq failed → Gemini: %v", err)
		}
	}

	// Phase 2: Gemini Flash (Secondary)
	if dailyCount < o.geminiLimit {
		if resp, err := o.geminiClient.Complete(ctx, systemPrompt, req.Prompt); err == nil {
			_ = o.usageTracker.Increment(ctx, req.UserID)
			return &LLMResponse{Text: resp, Provider: ProviderGemini}, nil
		} else {
			log.Printf("[LLM] Gemini failed → DeepSeek: %v", err)
		}
	}

	// Phase 3: DeepSeek (Paid overflow — last resort)
	if resp, err := o.deepSeekClient.Complete(ctx, systemPrompt, req.Prompt); err == nil {
		_ = o.usageTracker.Increment(ctx, req.UserID)
		return &LLMResponse{Text: resp, Provider: ProviderDeepSeek}, nil
	}

	return nil, fmt.Errorf("all LLM providers exhausted")
}

func (o *LLMOrchestrator) Summarize(ctx context.Context, transcript string) (string, error) {
	prompt := "Summarize this conversation in 2-3 sentences, preserving the user's key topics and preferences:\n\n" + transcript
	return o.groqClient.Complete(ctx, "You are a helpful assistant that creates concise conversation summaries.", prompt)
}

// ─── GroqAdapter ────────────────────────────────────────────────
type GroqAdapter struct {
	apiKey string
	client *http.Client
}

func NewGroqAdapter(apiKey string) *GroqAdapter {
	return &GroqAdapter{apiKey: apiKey, client: &http.Client{Timeout: 30 * time.Second}}
}

func (a *GroqAdapter) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	payload := map[string]interface{}{
		"model": "llama-4-scout-17b-16e-instruct",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"max_tokens": 1024,
		"temperature": 0.7,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != nil {
		return "", fmt.Errorf("Groq API: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("Groq returned no choices")
	}
	return result.Choices[0].Message.Content, nil
}

// ─── GeminiAdapter ──────────────────────────────────────────────
type GeminiAdapter struct {
	apiKey string
	client *http.Client
}

func NewGeminiAdapter(apiKey string) *GeminiAdapter {
	return &GeminiAdapter{apiKey: apiKey, client: &http.Client{Timeout: 30 * time.Second}}
}

func (a *GeminiAdapter) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash-lite:generateContent?key=%s", a.apiKey)
	payload := map[string]interface{}{
		"system_instruction": map[string]interface{}{"parts": []map[string]string{{"text": systemPrompt}}},
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": userPrompt}}},
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

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
		return "", err
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini returned no content")
	}
	return result.Candidates[0].Content.Parts[0].Text, nil
}

// ─── DeepSeekAdapter ────────────────────────────────────────────
type DeepSeekAdapter struct {
	apiKey string
	client *http.Client
}

func NewDeepSeekAdapter(apiKey string) *DeepSeekAdapter {
	return &DeepSeekAdapter{apiKey: apiKey, client: &http.Client{Timeout: 60 * time.Second}}
}

func (a *DeepSeekAdapter) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	payload := map[string]interface{}{
		"model": "deepseek-chat",
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.deepseek.com/chat/completions", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("DeepSeek returned no choices")
	}
	return result.Choices[0].Message.Content, nil
}
