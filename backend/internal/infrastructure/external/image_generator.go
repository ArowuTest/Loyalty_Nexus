package external

import (
	"context"
	"fmt"
)

type ImageGenerator interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type FalAIAdapter struct {
	APIKey string
}

func (a *FalAIAdapter) Generate(ctx context.Context, prompt string) (string, error) {
	// In production: POST to https://fal.run/fal-ai/flux/schnell
	return fmt.Sprintf("https://cdn.loyalty-nexus.ai/generated/%s.webp", "mock-fal-uuid"), nil
}

type HuggingFaceAdapter struct {
	APIKey string
}

func (a *HuggingFaceAdapter) Generate(ctx context.Context, prompt string) (string, error) {
	// In production: POST to HF Inference API
	return fmt.Sprintf("https://cdn.loyalty-nexus.ai/generated/%s.webp", "mock-hf-uuid"), nil
}
