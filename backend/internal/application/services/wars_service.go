package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RegionalWarsService manages state-level leaderboards and monthly prize pools.
// Architecture: points are aggregated from transactions; this service reads the
// materialised leaderboard and handles prize declarations.
type RegionalWarsService struct {
	db *gorm.DB
}

func NewRegionalWarsService(db *gorm.DB) *RegionalWarsService {
	return &RegionalWarsService{db: db}
}

// RegionalEntry represents one state's standing in the current war.
type RegionalEntry struct {
	State         string    `json:"state"          gorm:"column:state"`
	TotalPoints   int64     `json:"total_points"   gorm:"column:total_points"`
	ActiveMembers int       `json:"active_members" gorm:"column:active_members"`
	Rank          int       `json:"rank"           gorm:"column:rank"`
	PrizeKobo     int64     `json:"prize_kobo"     gorm:"column:prize_kobo"`
	Period        string    `json:"period"         gorm:"column:period"`
}

// RegionalWar metadata row.
type RegionalWar struct {
	ID             uuid.UUID  `gorm:"column:id;primaryKey"                  json:"id"`
	Period         string     `gorm:"column:period"                         json:"period"`  // e.g. "2026-03"
	Status         string     `gorm:"column:status"                         json:"status"`  // ACTIVE|COMPLETED
	TotalPrizeKobo int64      `gorm:"column:total_prize_kobo"               json:"total_prize_kobo"`
	StartsAt       time.Time  `gorm:"column:starts_at"                      json:"starts_at"`
	EndsAt         time.Time  `gorm:"column:ends_at"                        json:"ends_at"`
	ResolvedAt     *time.Time `gorm:"column:resolved_at"                    json:"resolved_at,omitempty"`
}

func (RegionalWar) TableName() string { return "regional_wars" }

// GetLeaderboard returns top-N states for the current active war, ranked by
// total pulse points contributed by members whose billing state is set.
func (svc *RegionalWarsService) GetLeaderboard(ctx context.Context, limit int) ([]RegionalEntry, error) {
	if limit <= 0 || limit > 37 {
		limit = 10
	}
	period := currentPeriod()

	// Cross-DB compatible query (Postgres + SQLite) — no window functions, no date_trunc.
	var rows []RegionalEntry
	err := svc.db.WithContext(ctx).Raw(`
		SELECT
			u.state                              AS state,
			COALESCE(SUM(t.points_earned), 0)    AS total_points,
			COUNT(DISTINCT u.id)                 AS active_members,
			0                                    AS prize_kobo,
			? AS period
		FROM users u
		LEFT JOIN transactions t
			ON t.user_id = u.id
			AND t.type = 'CREDIT'
			AND t.points_earned > 0
		WHERE u.state IS NOT NULL AND u.state != ''
		GROUP BY u.state
		ORDER BY total_points DESC
		LIMIT ?
	`, period, limit).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("leaderboard query: %w", err)
	}

	// Assign ranks client-side (avoids ROW_NUMBER OVER which is Postgres/SQLite 3.25+)
	for i := range rows {
		rows[i].Rank = i + 1
	}

	// Decorate prize_kobo for top 3 from active war
	var war RegionalWar
	if svc.db.WithContext(ctx).Where("status = 'ACTIVE' AND period = ?", period).
		First(&war).Error == nil {
		prizeShares := []int64{50, 30, 20}
		for i := range rows {
			if i < len(prizeShares) {
				rows[i].PrizeKobo = war.TotalPrizeKobo * prizeShares[i] / 100
			}
		}
	}
	return rows, nil
}

// GetUserRank returns the rank and stats for a single user's state.
func (svc *RegionalWarsService) GetUserRank(ctx context.Context, userID uuid.UUID) (*RegionalEntry, error) {
	var state string
	if err := svc.db.WithContext(ctx).Table("users").
		Where("id = ?", userID).Pluck("state", &state).Error; err != nil {
		return nil, err
	}
	if state == "" {
		return nil, fmt.Errorf("user has no state set")
	}
	rows, err := svc.GetLeaderboard(ctx, 37)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.State == state {
			r := r
			return &r, nil
		}
	}
	return &RegionalEntry{State: state, Rank: 37, Period: currentPeriod()}, nil
}

// EnsureActiveWar creates a war record for the current month if none exists.
func (svc *RegionalWarsService) EnsureActiveWar(ctx context.Context, prizePoolKobo int64) error {
	period := currentPeriod()
	now := time.Now().UTC()
	war := RegionalWar{
		ID:             uuid.New(),
		Period:         period,
		Status:         "ACTIVE",
		TotalPrizeKobo: prizePoolKobo,
		StartsAt:       time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
		EndsAt:         time.Date(now.Year(), now.Month()+1, 0, 23, 59, 59, 0, time.UTC),
	}
	return svc.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "period"}}, DoNothing: true}).
		Create(&war).Error
}

// ResolveWar distributes prizes to top-3 states and marks war COMPLETED.
func (svc *RegionalWarsService) ResolveWar(ctx context.Context, period string) error {
	return svc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var war RegionalWar
		if err := tx.Where("period = ? AND status = 'ACTIVE'", period).First(&war).Error; err != nil {
			return fmt.Errorf("active war for period %s not found", period)
		}
		// Inline leaderboard using tx (avoids nested transaction on SQLite)
		var rows []RegionalEntry
		qErr := tx.Raw(`
			SELECT
				u.state                              AS state,
				COALESCE(SUM(t.points_earned), 0)    AS total_points,
				COUNT(DISTINCT u.id)                 AS active_members,
				0                                    AS prize_kobo,
				? AS period
			FROM users u
			LEFT JOIN transactions t
				ON t.user_id = u.id
				AND t.type = 'CREDIT'
				AND t.points_earned > 0
			WHERE u.state IS NOT NULL AND u.state != ''
			GROUP BY u.state
			ORDER BY total_points DESC
			LIMIT 3
		`, period).Scan(&rows).Error
		if qErr != nil {
			return fmt.Errorf("leaderboard query: %w", qErr)
		}
		for i := range rows { rows[i].Rank = i + 1 }
		prizeShares := []int64{50, 30, 20}
		for i := range rows {
			if i < len(prizeShares) {
				rows[i].PrizeKobo = war.TotalPrizeKobo * prizeShares[i] / 100
			}
		}
		now := time.Now().UTC()
		for _, r := range rows {
			winner := map[string]interface{}{
				"id":          uuid.New(),
				"war_id":      war.ID,
				"state":       r.State,
				"rank":        r.Rank,
				"total_points": r.TotalPoints,
				"prize_kobo":  r.PrizeKobo,
				"status":      "PENDING",
				"created_at":  now,
				"updated_at":  now,
			}
			tx.Table("regional_war_winners").Create(winner)
		}
		resolved := now
		return tx.Model(&war).Updates(map[string]interface{}{
			"status":      "COMPLETED",
			"resolved_at": &resolved,
		}).Error
	})
}

func currentPeriod() string {
	t := time.Now().UTC()
	return fmt.Sprintf("%d-%02d", t.Year(), t.Month())
}