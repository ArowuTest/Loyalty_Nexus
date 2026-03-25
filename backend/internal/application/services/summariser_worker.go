package services

import (
	"context"
	"database/sql"
	"log"
	"time"
)

type SummariserWorker struct {
	db           *sql.DB
	llmOrchestrator *LLMOrchestrator // to call a 'Summarize' method
}

func (w *SummariserWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.ProcessExpiredSessions(ctx)
		}
	}
}

func (w *SummariserWorker) ProcessExpiredSessions(ctx context.Context) {
	// Query chats with status 'active' and no activity for 30m
	// Call LLM to summarize
	// Store in session_summaries table
	log.Printf("[Summariser] Scanning for expired chat sessions...")
}
