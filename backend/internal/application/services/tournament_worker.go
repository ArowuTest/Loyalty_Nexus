package services

import (
	"context"
	"database/sql"
	"log"
	"time"
)

type TournamentWorker struct {
	db *sql.DB
}

func NewTournamentWorker(db *sql.DB) *TournamentWorker {
	return &TournamentWorker{db: db}
}

func (w *TournamentWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.AggregateRanks(ctx)
			w.CheckGoldenHour(ctx)
		}
	}
}

func (w *TournamentWorker) AggregateRanks(ctx context.Context) {
	// Rank regions based on total recharge volume in the current window
	query := `
		WITH ranks AS (
			SELECT region_code, RANK() OVER (ORDER BY total_recharge_kobo DESC) as new_rank
			FROM regional_stats
		)
		UPDATE regional_stats s
		SET rank = r.new_rank, updated_at = now()
		FROM ranks r
		WHERE s.region_code = r.region_code
	`
	_, err := w.db.ExecContext(ctx, query)
	if err != nil {
		log.Printf("[TournamentWorker] Rank aggregation failed: %v", err)
	}
}

func (w *TournamentWorker) CheckGoldenHour(ctx context.Context) {
	// Logic to enable golden hour for the top region on Fridays, etc.
	// This is a business-rule placeholder.
}
