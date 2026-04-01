package handlers

import (
	"log"
	"net/http"
	"time"

	"gorm.io/gorm"
)

// PublicStats represents the aggregated platform statistics for the landing page.
type PublicStats struct {
	TotalUsers       int64 `json:"total_users"`
	TotalGenerations int64 `json:"total_generations"`
	TotalPrizesWon   int64 `json:"total_prizes_won"`
	TotalPointsEarned int64 `json:"total_points_earned"`
}

// GetPublicStats returns a handler that serves cached platform statistics.
// To avoid hitting the DB on every landing page load, stats are cached for 5 minutes.
func GetPublicStats(db *gorm.DB) http.HandlerFunc {
	var cachedStats PublicStats
	var lastFetch time.Time

	return func(w http.ResponseWriter, r *http.Request) {
		// Serve from cache if less than 5 minutes old
		if time.Since(lastFetch) < 5*time.Minute && cachedStats.TotalUsers > 0 {
			writeJSON(w, http.StatusOK, cachedStats)
			return
		}

		var stats PublicStats

		// Count total users
		if err := db.WithContext(r.Context()).Table("users").Count(&stats.TotalUsers).Error; err != nil {
			log.Printf("[Stats] Error counting users: %v", err)
		}

		// Count total AI generations
		if err := db.WithContext(r.Context()).Table("studio_generations").Count(&stats.TotalGenerations).Error; err != nil {
			log.Printf("[Stats] Error counting generations: %v", err)
		}

		// Count total prizes won (cash, data, airtime)
		if err := db.WithContext(r.Context()).Table("spin_results").Where("prize_type != 'no_win'").Count(&stats.TotalPrizesWon).Error; err != nil {
			log.Printf("[Stats] Error counting prizes: %v", err)
		}

		// Sum total lifetime points earned
		if err := db.WithContext(r.Context()).Table("users").Select("COALESCE(SUM(lifetime_points), 0)").Scan(&stats.TotalPointsEarned).Error; err != nil {
			log.Printf("[Stats] Error summing points: %v", err)
		}

		// Update cache
		cachedStats = stats
		lastFetch = time.Now()

		writeJSON(w, http.StatusOK, stats)
	}
}
