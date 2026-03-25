package external

import (
	"context"
	"fmt"
	"log"
	"strings"
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
	chatRepo       repositories.ChatRepository
	// Thresholds from Cockpit Config
	groqLimit   int
	geminiLimit int
}

func NewLLMOrchestrator(g, gem, ds LLMClient, ut UsageTracker, cr repositories.ChatRepository, gLim, gemLim int) *LLMOrchestrator {
	return &LLMOrchestrator{
		groqClient:     g,
		geminiClient:   gem,
		deepSeekClient: ds,
		usageTracker:   ut,
		chatRepo:       cr,
		groqLimit:      gLim,
		geminiLimit:    gemLim,
	}
}

func (o *LLMOrchestrator) Chat(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
	dailyCount, _ := o.usageTracker.GetDailyCount(ctx, req.UserID)

	// 1. Fetch Memory (REQ-4.3.4)
	uid, _ := uuid.Parse(req.UserID)
	summaries, _ := o.chatRepo.GetLastSummaries(ctx, uid, 3)
	
	systemPrompt := "You are Nexus, a helpful AI assistant for MTN Pulse subscribers."
	if len(summaries) > 0 {
		memoryContext := "\nPrevious context: " + strings.Join(summaries, " ")
		systemPrompt += memoryContext
	}

	fullPrompt := fmt.Sprintf("%s\nUser: %s", systemPrompt, req.Prompt)

	// Phase 1: Groq (Primary Free Workhorse)
	if dailyCount < o.groqLimit {
		resp, err := o.groqClient.Complete(ctx, fullPrompt)
		if err == nil {
			o.usageTracker.Increment(ctx, req.UserID)
			return &LLMResponse{Text: resp, Provider: ProviderGroq}, nil
		}
		log.Printf("Groq failed, falling back to Gemini: %v", err)
	}

	// Phase 2: Gemini Flash-Lite (Secondary Free Pool)
	if dailyCount < o.geminiLimit {
		resp, err := o.geminiClient.Complete(ctx, fullPrompt)
		if err == nil {
			o.usageTracker.Increment(ctx, req.UserID)
			return &LLMResponse{Text: resp, Provider: ProviderGemini}, nil
		}
		log.Printf("Gemini failed, falling back to DeepSeek: %v", err)
	}

	// Phase 3: DeepSeek V3.2 (Paid Overflow)
	resp, err := o.deepSeekClient.Complete(ctx, fullPrompt)
	if err != nil {
		return nil, fmt.Errorf("all LLM providers exhausted: %w", err)
	}

	o.usageTracker.Increment(ctx, req.UserID)
	return &LLMResponse{Text: resp, Provider: ProviderDeepSeek}, nil
}

func (o *LLMOrchestrator) Summarize(ctx context.Context, transcript string) (string, error) {
	// Use Groq or Gemini to summarize the transcript into a short paragraph
	return o.groqClient.Complete(ctx, "Summarize this chat transcript in one concise paragraph: " + transcript)
}
