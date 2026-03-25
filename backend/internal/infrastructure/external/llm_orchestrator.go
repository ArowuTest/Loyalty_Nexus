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
	// ... existing logic ...
}

func (o *LLMOrchestrator) Summarize(ctx context.Context, transcript string) (string, error) {
	// Use Groq or Gemini to summarize the transcript into a short paragraph
	return o.groqClient.Complete(ctx, "Summarize this chat transcript: " + transcript)
}
