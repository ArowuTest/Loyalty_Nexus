package repositories

// wars_repository.go — Persistence port for Regional Wars.
//
// All leaderboard computation uses correct column names from the actual schema:
//   transactions.type       = 'points_award'   (TxTypePointsAward)
//   transactions.points_delta > 0              (positive = earn, negative = spend)
//   users.state             — Nigerian state name (e.g. "Lagos")

import (
	"context"
	"time"

	"github.com/google/uuid"

	"loyalty-nexus/internal/domain/entities"
)

type WarsRepository interface {

	// ─── War lifecycle ────────────────────────────────────────────────────────

	// FindActivWar returns the ACTIVE war for a given period.
	FindActiveWar(ctx context.Context, period string) (*entities.RegionalWar, error)

	// EnsureWar creates a war for the period if none exists (idempotent).
	EnsureWar(ctx context.Context, period string, prizeKobo int64, startsAt, endsAt time.Time) error

	// MarkResolved sets status=COMPLETED and resolved_at.
	MarkResolved(ctx context.Context, warID uuid.UUID) error

	// ListWars returns wars ordered by period desc (for admin panel).
	ListWars(ctx context.Context, limit int) ([]entities.RegionalWar, error)

	// ─── Leaderboard ──────────────────────────────────────────────────────────

	// GetLeaderboard computes live point totals per state for the given time window.
	// Aggregates transactions WHERE type='points_award' AND points_delta > 0.
	GetLeaderboard(ctx context.Context, from, to time.Time, limit int) ([]entities.LeaderboardEntry, error)

	// GetStateTotal returns aggregate points for a single state.
	GetStateTotal(ctx context.Context, state string, from, to time.Time) (int64, error)

	// ─── Winners ──────────────────────────────────────────────────────────────

	// CreateWinners bulk-inserts top-3 winner rows (called inside ResolveWar txn).
	CreateWinners(ctx context.Context, winners []entities.RegionalWarWinner) error

	// GetWinnersForWar returns winners for a given war_id.
	GetWinnersForWar(ctx context.Context, warID uuid.UUID) ([]entities.RegionalWarWinner, error)

	// MarkWinnerPaid updates a winner row status to PAID.
	MarkWinnerPaid(ctx context.Context, winnerID uuid.UUID) error
}
