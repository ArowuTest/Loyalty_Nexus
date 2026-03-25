package external

import (
	"context"
	"fmt"
	"log"
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

// Client Interfaces for abstraction
type LLMClient interface {
	Complete(ctx context.Context, prompt string) (string, error)
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
	// Thresholds from Cockpit Config
	groqLimit   int
	geminiLimit int
}

func NewLLMOrchestrator(g, gem, ds LLMClient, ut UsageTracker, gLim, gemLim int) *LLMOrchestrator {
	return &LLMOrchestrator{
		groqClient:     g,
		geminiClient:   gem,
		deepSeekClient: ds,
		usageTracker:   ut,
		groqLimit:      gLim,
		geminiLimit:    gemLim,
	}
}

func (o *LLMOrchestrator) Chat(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	dailyCount, _ := o.usageTracker.GetDailyCount(ctx, req.UserID)

	// Phase 1: Groq (Primary Free Workhorse)
	if dailyCount < o.groqLimit {
		resp, err := o.groqClient.Complete(ctx, req.Prompt)
		if err == nil {
			o.usageTracker.Increment(ctx, req.UserID)
			return &LLMResponse{Text: resp, Provider: ProviderGroq}, nil
		}
		log.Printf("Groq failed, falling back to Gemini: %v", err)
	}

	// Phase 2: Gemini Flash-Lite (Secondary Free Pool)
	if dailyCount < o.geminiLimit {
		resp, err := o.geminiClient.Complete(ctx, req.Prompt)
		if err == nil {
			o.usageTracker.Increment(ctx, req.UserID)
			return &LLMResponse{Text: resp, Provider: ProviderGemini}, nil
		}
		log.Printf("Gemini failed, falling back to DeepSeek: %v", err)
	}

	// Phase 3: DeepSeek V3.2 (Paid Overflow)
	resp, err := o.deepSeekClient.Complete(ctx, req.Prompt)
	if err != nil {
		return nil, fmt.Errorf("all LLM providers exhausted: %w", err)
	}

	o.usageTracker.Increment(ctx, req.UserID)
	return &LLMResponse{Text: resp, Provider: ProviderDeepSeek}, nil
}
