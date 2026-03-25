package external

import (
	"context"
	"fmt"
)

type DocumentGenerator interface {
	GenerateSlideDeck(ctx context.Context, topic string) (string, error)
	GenerateBusinessPlan(ctx context.Context, description string) (string, error)
}

type AudioTranscriber interface {
	Transcribe(ctx context.Context, audioURL string) (string, error)
}

type AssemblyAIAdapter struct {
	APIKey string
}

func (a *AssemblyAIAdapter) Transcribe(ctx context.Context, audioURL string) (string, error) {
	// In production: POST to AssemblyAI
	return "Transcribed business idea about solar energy...", nil
}

type DocumentAutomationAdapter struct {
	// Wrapper for Gemini/NotebookLM structured output
}

func (a *DocumentAutomationAdapter) GenerateSlideDeck(ctx context.Context, topic string) (string, error) {
	return fmt.Sprintf("https://cdn.loyalty-nexus.ai/build/%s.pptx", "deck-uuid"), nil
}

func (a *DocumentAutomationAdapter) GenerateBusinessPlan(ctx context.Context, description string) (string, error) {
	return fmt.Sprintf("https://cdn.loyalty-nexus.ai/build/%s.pdf", "plan-uuid"), nil
}
