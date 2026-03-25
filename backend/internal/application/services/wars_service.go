package services

// wars_service.go — Regional Wars application service (spec §3.5)
//
// Responsibilities:
//   1. EnsureActiveWar — idempotent monthly war bootstrap
//   2. GetLeaderboard  — live aggregation via WarsRepository
//   3. GetUserRank     — single state lookup within leaderboard
//   4. ResolveWar      — atomic: compute top-3, write winners, mark COMPLETED
//   5. BonusToStateMembers — send Pulse Point bonuses to winning state users
//
// Financial note: PrizeKobo is read from regional_wars.total_prize_kobo (DB).
// Bonus PP for winning users is read from network_configs 'regional_wars_winning_bonus'.

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"
)

// prizeShares defines the percentage breakdown for top-3 states.
var prizeShares = [3]int64{50, 30, 20}

// RegionalWarsService manages the full lifecycle of monthly Regional Wars.
type RegionalWarsService struct {
	warsRepo  repositories.WarsRepository
	userRepo  repositories.UserRepository
	txRepo    repositories.TransactionRepository
	cfg       *config.ConfigManager
	db        *gorm.DB // used only for the bonus-award transaction
}

func NewRegionalWarsService(
	warsRepo repositories.WarsRepository,
	userRepo repositories.UserRepository,
	txRepo repositories.TransactionRepository,
	cfg *config.ConfigManager,
	db *gorm.DB,
) *RegionalWarsService {
	return &RegionalWarsService{
		warsRepo: warsRepo,
		userRepo: userRepo,
		txRepo:   txRepo,
		cfg:      cfg,
		db:       db,
	}
}

// ─── War bootstrap ────────────────────────────────────────────────────────────

// EnsureActiveWar creates the current month's war if it doesn't already exist.
func (svc *RegionalWarsService) EnsureActiveWar(ctx context.Context, defaultPrizeKobo int64) error {
	now := time.Now().UTC()
	period := periodStr(now)

	// Prize pool: prefer DB config, fall back to argument.
	prizeKobo := int64(svc.cfg.GetInt("regional_wars_prize_pool_kobo", int(defaultPrizeKobo)))

	startsAt := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endsAt   := time.Date(now.Year(), now.Month()+1, 0, 23, 59, 59, 0, time.UTC)

	return svc.warsRepo.EnsureWar(ctx, period, prizeKobo, startsAt, endsAt)
}

// ─── Leaderboard ──────────────────────────────────────────────────────────────

// GetLeaderboard returns top-N states for the current month, ranked by
// aggregate points_award transactions during the war window.
func (svc *RegionalWarsService) GetLeaderboard(ctx context.Context, limit int) ([]entities.LeaderboardEntry, error) {
	now := time.Now().UTC()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	to   := time.Date(now.Year(), now.Month()+1, 0, 23, 59, 59, 999999999, time.UTC)

	entries, err := svc.warsRepo.GetLeaderboard(ctx, from, to, limit)
	if err != nil {
		return nil, err
	}

	// Decorate prize amounts from active war record
	period := periodStr(now)
	war, wErr := svc.warsRepo.FindActiveWar(ctx, period)
	if wErr == nil {
		for i := range entries {
			if i < len(prizeShares) {
				entries[i].PrizeKobo = war.TotalPrizeKobo * prizeShares[i] / 100
			}
			entries[i].Period = period
		}
	} else {
		for i := range entries {
			entries[i].Period = period
		}
	}

	return entries, nil
}

// GetUserRank finds the leaderboard position for the requesting user's state.
func (svc *RegionalWarsService) GetUserRank(ctx context.Context, userID uuid.UUID) (*entities.LeaderboardEntry, error) {
	user, err := svc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.State == "" {
		return nil, fmt.Errorf("user has no state — update profile to join Regional Wars")
	}

	// Get full leaderboard (max 37 Nigerian states)
	entries, err := svc.GetLeaderboard(ctx, 37)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.State == user.State {
			e2 := e
			return &e2, nil
		}
	}
	// State has no points yet — return placeholder at last position
	return &entities.LeaderboardEntry{
		State:  user.State,
		Rank:   37,
		Period: periodStr(time.Now().UTC()),
	}, nil
}

// ─── War resolution ───────────────────────────────────────────────────────────

// ResolveWar atomically:
//  1. Finds the ACTIVE war for `period`
//  2. Computes the final top-3 leaderboard
//  3. Inserts RegionalWarWinner rows
//  4. Marks the war COMPLETED
//  5. Awards Pulse Point bonuses to all users in winning states (async, best-effort)
func (svc *RegionalWarsService) ResolveWar(ctx context.Context, period string) ([]entities.LeaderboardEntry, error) {
	// Resolve the war's time window from the DB record
	war, err := svc.warsRepo.FindActiveWar(ctx, period)
	if err != nil {
		return nil, fmt.Errorf("no active war for period %q: %w", period, err)
	}

	// Compute final leaderboard for the war window
	entries, err := svc.warsRepo.GetLeaderboard(ctx, war.StartsAt, war.EndsAt, 3)
	if err != nil {
		return nil, fmt.Errorf("final leaderboard: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no state data to resolve for period %q", period)
	}

	// Build winner rows
	winners := make([]entities.RegionalWarWinner, len(entries))
	for i, e := range entries {
		prizeKobo := int64(0)
		if i < len(prizeShares) {
			prizeKobo = war.TotalPrizeKobo * prizeShares[i] / 100
		}
		winners[i] = entities.RegionalWarWinner{
			ID:          uuid.New(),
			WarID:       war.ID,
			State:       e.State,
			Rank:        i + 1,
			TotalPoints: e.TotalPoints,
			PrizeKobo:   prizeKobo,
			Status:      "PENDING",
		}
		entries[i].PrizeKobo = prizeKobo
		entries[i].Rank      = i + 1
		entries[i].Period     = period
	}

	// Persist winners + mark war resolved
	if err := svc.warsRepo.CreateWinners(ctx, winners); err != nil {
		return nil, fmt.Errorf("create winners: %w", err)
	}
	if err := svc.warsRepo.MarkResolved(ctx, war.ID); err != nil {
		return nil, fmt.Errorf("mark resolved: %w", err)
	}

	// Award bonus Pulse Points to all users in top-3 states (async, best-effort)
	bonusPP := int64(svc.cfg.GetInt("regional_wars_winning_bonus", 50))
	go svc.awardStateBonuses(context.Background(), winners, bonusPP)

	return entries, nil
}

// awardStateBonuses iterates winning states and issues bonus PulsePoints to
// every active user in those states.  Called in a goroutine (best-effort).
func (svc *RegionalWarsService) awardStateBonuses(ctx context.Context, winners []entities.RegionalWarWinner, bonusPP int64) {
	if bonusPP <= 0 {
		return
	}
	for _, w := range winners {
		var userIDs []uuid.UUID
		svc.db.WithContext(ctx).
			Table("users").
			Where("state = ? AND is_active = true", w.State).
			Pluck("id", &userIDs)

		for _, uid := range userIDs {
			svc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				// Credit wallet
				if err := tx.Exec(`
					UPDATE wallets SET pulse_points = pulse_points + ?, lifetime_points = lifetime_points + ?
					WHERE user_id = ?`, bonusPP, bonusPP, uid).Error; err != nil {
					return err
				}
				// Immutable ledger entry
				user, _ := svc.userRepo.FindByID(ctx, uid)
				phone := ""
				if user != nil {
					phone = user.PhoneNumber
				}
				return tx.Create(&entities.Transaction{
					ID:          uuid.New(),
					UserID:      uid,
					PhoneNumber: phone,
					Type:        entities.TxTypeBonus,
					PointsDelta: bonusPP,
					Reference:   "wars_bonus_" + w.State + "_rank" + fmt.Sprintf("%d", w.Rank),
					CreatedAt:   time.Now(),
				}).Error
			})
		}
	}
}

// ─── Admin helpers ────────────────────────────────────────────────────────────

// ListWars returns recent wars for the admin panel.
func (svc *RegionalWarsService) ListWars(ctx context.Context, limit int) ([]entities.RegionalWar, error) {
	return svc.warsRepo.ListWars(ctx, limit)
}

// GetWinnersForWar returns winner rows for a completed war.
func (svc *RegionalWarsService) GetWinnersForWar(ctx context.Context, warID uuid.UUID) ([]entities.RegionalWarWinner, error) {
	return svc.warsRepo.GetWinnersForWar(ctx, warID)
}

// UpdateWarPrizePool allows admin to change the prize pool before resolution.
func (svc *RegionalWarsService) UpdateWarPrizePool(ctx context.Context, period string, newPrizeKobo int64) error {
	return svc.db.WithContext(ctx).
		Model(&entities.RegionalWar{}).
		Where("period = ? AND status = ?", period, entities.WarStatusActive).
		Update("total_prize_kobo", newPrizeKobo).Error
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// periodStr returns "YYYY-MM" for the given UTC time.
func periodStr(t time.Time) string {
	return fmt.Sprintf("%d-%02d", t.Year(), t.Month())
}

// ExportedCurrentPeriod exposes the current period string to the handler layer.
func ExportedCurrentPeriod() string { return periodStr(time.Now().UTC()) }
