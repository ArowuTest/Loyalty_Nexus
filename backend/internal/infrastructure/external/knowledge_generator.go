package external

import (
	"context"
	"fmt"
	"time"
)

type KnowledgeGenerator interface {
	TriggerGeneration(ctx context.Context, topic string, toolType string) (string, error)
	PollStatus(ctx context.Context, genID string) (bool, string, error) // ready, output_url, error
}

type NotebookLMAdapter struct {
	APIKey string
}

func (a *NotebookLMAdapter) TriggerGeneration(ctx context.Context, topic string, toolType string) (string, error) {
	// In production: POST to NotebookLM API
	return fmt.Sprintf("nblm_%d", time.Now().Unix()), nil
}

func (a *NotebookLMAdapter) PollStatus(ctx context.Context, genID string) (bool, string, error) {
	// In production: GET status from NotebookLM
	// Simulation: always ready after first poll
	return true, fmt.Sprintf("https://cdn.loyalty-nexus.ai/learning/%s.pdf", genID), nil
}
