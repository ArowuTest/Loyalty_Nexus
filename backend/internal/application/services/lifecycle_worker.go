package services

import (
	"context"
	"database/sql"
	"log"
	"time"
)

type LifecycleWorker struct {
	db        *sql.DB
	notifySvc *NotificationService
}

func NewLifecycleWorker(db *sql.DB, ns *NotificationService) *LifecycleWorker {
	return &LifecycleWorker{db: db, notifySvc: ns}
}

func (w *LifecycleWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processExpirations(ctx)
			w.sendExpiryNudges(ctx)
		}
	}
}

func (w *LifecycleWorker) processExpirations(ctx context.Context) {
	// Delete assets older than 30 days
	query := "DELETE FROM ai_generations WHERE expires_at < now()"
	res, err := w.db.ExecContext(ctx, query)
	if err == nil {
		rows, _ := res.RowsAffected()
		log.Printf("[Lifecycle] Expired %d AI assets", rows)
	}
}

func (w *LifecycleWorker) sendExpiryNudges(ctx context.Context) {
	// Nudge users 48h before expiration
	// In production: SELECT generations where expires_at is in 48h and nudge not sent
}
