package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/infrastructure/external"
	"github.com/google/uuid"
)

type AsyncStudioWorker struct {
	studioService      *services.StudioService
	knowledgeGenerator external.KnowledgeGenerator
}

func NewAsyncStudioWorker(ss *services.StudioService, kg external.KnowledgeGenerator) *AsyncStudioWorker {
	return &AsyncStudioWorker{
		studioService:      ss,
		knowledgeGenerator: kg,
	}
}

// StartJob simulates the background polling loop for an async generation
func (w *AsyncStudioWorker) StartJob(genID uuid.UUID, providerGenID string) {
	go func() {
		ctx := context.Background()
		log.Printf("[AsyncWorker] Starting poll for GenID: %s", genID)

		// Simple polling strategy
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		timeout := time.After(5 * time.Minute)

		for {
			select {
			case <-timeout:
				w.studioService.FailGeneration(ctx, genID, "Generation timed out")
				return
			case <-ticker.C:
				ready, url, err := w.knowledgeGenerator.PollStatus(ctx, providerGenID)
				if err != nil {
					w.studioService.FailGeneration(ctx, genID, err.Error())
					return
				}
				if ready {
					// Mapping cost based on tool/provider logic
					w.studioService.CompleteGeneration(ctx, genID, url, "NOTEBOOK_LM", 10000)
					log.Printf("[AsyncWorker] Completed GenID: %s", genID)
					return
				}
			}
		}
	}()
}
