package services

import (
	"context"
	"fmt"
	"log"
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"loyalty-nexus/internal/infrastructure/external"
)

type SummariserWorker struct {
	db              *gorm.DB
	llmOrchestrator *external.LLMOrchestrator
}

func NewSummariserWorker(db *gorm.DB, llm *external.LLMOrchestrator) *SummariserWorker {
	return &SummariserWorker{db: db, llmOrchestrator: llm}
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
	var sessions []struct {
		ID     uuid.UUID
		UserID uuid.UUID
	}
	// REQ-4.3.4: Sessions expire after 30 minutes of inactivity
	query := "status = 'active' AND last_activity_at < now() - interval '30 minutes'"
	if err := w.db.WithContext(ctx).Table("chat_sessions").Where(query).Find(&sessions).Error; err != nil {
		return
	}

	for _, s := range sessions {
		transcript, _ := w.getTranscript(ctx, s.ID)
		if transcript == "" {
			continue
		}

		summary, err := w.llmOrchestrator.Summarize(ctx, transcript)
		if err == nil {
			w.storeSummary(ctx, s.UserID, s.ID, summary)
			log.Printf("[Summariser] Session %s compressed into memory", s.ID)
		}
	}
}

func (w *SummariserWorker) getTranscript(ctx context.Context, sessionID uuid.UUID) (string, error) {
	var messages []struct {
		Role    string
		Content string
	}
	w.db.WithContext(ctx).Table("chat_messages").
		Where("session_id = ?", sessionID).
		Order("created_at asc").
		Find(&messages)

	var transcript string
	for _, m := range messages {
		transcript += fmt.Sprintf("%s: %s\n", m.Role, m.Content)
	}
	return transcript, nil
}

func (w *SummariserWorker) storeSummary(ctx context.Context, userID, sessionID uuid.UUID, summary string) {
	w.db.Transaction(func(tx *gorm.DB) error {
		tx.Table("session_summaries").Create(map[string]interface{}{
			"user_id": userID,
			"summary": summary,
		})
		tx.Table("chat_sessions").Where("id = ?", sessionID).Update("status", "summarized")
		return nil
	})
}
