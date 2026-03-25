package services

import (
	"context"
	"log"
	"time"
	"gorm.io/gorm"
)

type TournamentWorker struct {
	db *gorm.DB
}

func NewTournamentWorker(db *gorm.DB) *TournamentWorker {
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
	// Rank regions based on total recharge volume
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
	err := w.db.WithContext(ctx).Exec(query).Error
	if err != nil {
		log.Printf("[TournamentWorker] Rank aggregation failed: %v", err)
	}
}

func (w *TournamentWorker) CheckGoldenHour(ctx context.Context) {
	// Innovation: Regional Wars (Strategy Doc Section 4)
	// Every Friday, enable Golden Hour for the region with the highest weekly growth.
	if time.Now().Weekday() == time.Friday {
		w.db.Transaction(func(tx *gorm.DB) error {
			// 1. Reset all golden hours
			tx.Table("regional_settings").Update("is_golden_hour", false)

			// 2. Find top region
			var topRegion string
			tx.Table("regional_stats").Order("total_recharge_kobo DESC").Limit(1).Pluck("region_code", &topRegion)

			// 3. Activate
			if topRegion != "" {
				tx.Table("regional_settings").Where("region_code = ?", topRegion).Update("is_golden_hour", true)
				log.Printf("[TournamentWorker] Golden Hour Activated for %s", topRegion)
			}
			return nil
		})
	}
}
