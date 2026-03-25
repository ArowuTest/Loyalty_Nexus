package persistence

// wars_repo_postgres.go — GORM implementation of repositories.WarsRepository
//
// Key correctness notes:
//   - Leaderboard aggregates transactions.points_delta WHERE type = 'points_award'
//     (positive delta only).  This correctly uses the immutable ledger.
//   - users.state is the Nigerian state string ("Lagos", "Abuja", etc.).
//   - EnsureWar uses ON CONFLICT DO NOTHING for idempotency.

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
)

type postgresWarsRepository struct{ db *gorm.DB }

func NewPostgresWarsRepository(db *gorm.DB) repositories.WarsRepository {
	return &postgresWarsRepository{db: db}
}

// ─── War lifecycle ────────────────────────────────────────────────────────────

func (r *postgresWarsRepository) FindActiveWar(ctx context.Context, period string) (*entities.RegionalWar, error) {
	var war entities.RegionalWar
	err := r.db.WithContext(ctx).
		Where("period = ? AND status = ?", period, entities.WarStatusActive).
		First(&war).Error
	if err != nil {
		return nil, err
	}
	return &war, nil
}

func (r *postgresWarsRepository) EnsureWar(ctx context.Context, period string, prizeKobo int64, startsAt, endsAt time.Time) error {
	war := entities.RegionalWar{
		ID:             uuid.New(),
		Period:         period,
		Status:         entities.WarStatusActive,
		TotalPrizeKobo: prizeKobo,
		StartsAt:       startsAt,
		EndsAt:         endsAt,
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:  []clause.Column{{Name: "period"}},
			DoNothing: true,
		}).
		Create(&war).Error
}

func (r *postgresWarsRepository) MarkResolved(ctx context.Context, warID uuid.UUID) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&entities.RegionalWar{}).
		Where("id = ?", warID).
		Updates(map[string]interface{}{
			"status":      entities.WarStatusCompleted,
			"resolved_at": &now,
			"updated_at":  now,
		}).Error
}

func (r *postgresWarsRepository) ListWars(ctx context.Context, limit int) ([]entities.RegionalWar, error) {
	if limit <= 0 {
		limit = 12
	}
	var wars []entities.RegionalWar
	err := r.db.WithContext(ctx).
		Order("period DESC").
		Limit(limit).
		Find(&wars).Error
	return wars, err
}

// ─── Leaderboard ──────────────────────────────────────────────────────────────

// GetLeaderboard computes live per-state Pulse Point totals from the immutable
// transaction ledger.  Only 'points_award' transactions with positive delta are
// counted so that studio spends, refunds etc. don't pollute the leaderboard.
func (r *postgresWarsRepository) GetLeaderboard(ctx context.Context, from, to time.Time, limit int) ([]entities.LeaderboardEntry, error) {
	if limit <= 0 || limit > 37 {
		limit = 10
	}

	type row struct {
		State         string `gorm:"column:state"`
		TotalPoints   int64  `gorm:"column:total_points"`
		ActiveMembers int    `gorm:"column:active_members"`
	}
	var rows []row

	err := r.db.WithContext(ctx).Raw(`
		SELECT
			u.state                                     AS state,
			COALESCE(SUM(t.points_delta), 0)            AS total_points,
			COUNT(DISTINCT u.id)                        AS active_members
		FROM users u
		INNER JOIN transactions t
			ON  t.user_id     = u.id
			AND t.type        = ?
			AND t.points_delta > 0
			AND t.created_at BETWEEN ? AND ?
		WHERE u.state IS NOT NULL
		  AND u.state <> ''
		  AND u.is_active = true
		GROUP BY u.state
		ORDER BY total_points DESC
		LIMIT ?
	`, string(entities.TxTypePointsAward), from, to, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	entries := make([]entities.LeaderboardEntry, len(rows))
	for i, row := range rows {
		entries[i] = entities.LeaderboardEntry{
			State:         row.State,
			TotalPoints:   row.TotalPoints,
			ActiveMembers: row.ActiveMembers,
			Rank:          i + 1, // client-side rank (no window function needed)
		}
	}
	return entries, nil
}

func (r *postgresWarsRepository) GetStateTotal(ctx context.Context, state string, from, to time.Time) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Raw(`
		SELECT COALESCE(SUM(t.points_delta), 0)
		FROM transactions t
		INNER JOIN users u ON u.id = t.user_id
		WHERE u.state      = ?
		  AND t.type       = ?
		  AND t.points_delta > 0
		  AND t.created_at BETWEEN ? AND ?
	`, state, string(entities.TxTypePointsAward), from, to).Scan(&total).Error
	return total, err
}

// ─── Winners ──────────────────────────────────────────────────────────────────

func (r *postgresWarsRepository) CreateWinners(ctx context.Context, winners []entities.RegionalWarWinner) error {
	if len(winners) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&winners).Error
}

func (r *postgresWarsRepository) GetWinnersForWar(ctx context.Context, warID uuid.UUID) ([]entities.RegionalWarWinner, error) {
	var winners []entities.RegionalWarWinner
	err := r.db.WithContext(ctx).
		Where("war_id = ?", warID).
		Order("rank ASC").
		Find(&winners).Error
	return winners, err
}

func (r *postgresWarsRepository) MarkWinnerPaid(ctx context.Context, winnerID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&entities.RegionalWarWinner{}).
		Where("id = ?", winnerID).
		Updates(map[string]interface{}{
			"status":     "PAID",
			"updated_at": time.Now(),
		}).Error
}
