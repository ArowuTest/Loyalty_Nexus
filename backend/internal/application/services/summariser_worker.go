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
	// 1. Find sessions inactive for > 30m
	query := `
		SELECT id, user_id FROM chat_sessions 
		WHERE status = 'active' AND last_activity_at < now() - interval '30 minutes'
	`
	rows, err := w.db.QueryContext(ctx, query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var sessionID, userID uuid.UUID
		rows.Scan(&sessionID, &userID)

		// 2. Aggregate Transcript
		transcript, _ := w.getTranscript(ctx, sessionID)

		// 3. Generate Summary via LLM
		summary, err := w.llmOrchestrator.Summarize(ctx, transcript)
		if err == nil {
			// 4. Store Summary and update status
			w.storeSummary(ctx, userID, sessionID, summary)
		}
	}
}

func (w *SummariserWorker) getTranscript(ctx context.Context, sessionID uuid.UUID) (string, error) {
	var transcript string
	rows, _ := w.db.QueryContext(ctx, "SELECT role, content FROM chat_messages WHERE session_id = $1 ORDER BY created_at ASC", sessionID)
	defer rows.Close()
	for rows.Next() {
		var role, content string
		rows.Scan(&role, &content)
		transcript += fmt.Sprintf("%s: %s\n", role, content)
	}
	return transcript, nil
}

func (w *SummariserWorker) storeSummary(ctx context.Context, userID, sessionID uuid.UUID, summary string) {
	tx, _ := w.db.BeginTx(ctx, nil)
	defer tx.Rollback()

	tx.ExecContext(ctx, "INSERT INTO session_summaries (user_id, summary) VALUES ($1, $2)", userID, summary)
	tx.ExecContext(ctx, "UPDATE chat_sessions SET status = 'summarized' WHERE id = $1", sessionID)
	tx.Commit()
}
