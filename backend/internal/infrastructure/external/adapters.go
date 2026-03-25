package external

import (
	"context"
	"fmt"
)

// GroqAdapter implements the LLMClient interface for Groq API
type GroqAdapter struct {
	APIKey string
}

func (a *GroqAdapter) Complete(ctx context.Context, prompt string) (string, error) {
	// In production, make HTTP call to Groq API
	return fmt.Sprintf("[Groq] Response to: %s", prompt), nil
}

// GeminiAdapter implements the LLMClient interface for Google Gemini API
type GeminiAdapter struct {
	APIKey string
}

func (a *GeminiAdapter) Complete(ctx context.Context, prompt string) (string, error) {
	// In production, make HTTP call to Gemini API
	return fmt.Sprintf("[Gemini] Response to: %s", prompt), nil
}

// DeepSeekAdapter implements the LLMClient interface for DeepSeek API
type DeepSeekAdapter struct {
	APIKey string
}

func (a *DeepSeekAdapter) Complete(ctx context.Context, prompt string) (string, error) {
	// In production, make HTTP call to DeepSeek API
	return fmt.Sprintf("[DeepSeek] Response to: %s", prompt), nil
}
