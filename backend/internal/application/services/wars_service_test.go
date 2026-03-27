package services_test

// wars_service_test.go — unit tests for RegionalWarsService using mock repos + SQLite.
//
// The WarsRepository, UserRepository and TransactionRepository are implemented
// here as lightweight in-memory mocks so the tests do not require Postgres.

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/domain/repositories"
	"loyalty-nexus/internal/infrastructure/config"
)

// ─── in-memory WarsRepository ────────────────────────────────────────────────

type mockWarsRepo struct {
	mu      sync.Mutex
	wars    map[string]*entities.RegionalWar   // keyed by period
	winners []entities.RegionalWarWinner
}

func newMockWarsRepo() *mockWarsRepo { return &mockWarsRepo{wars: map[string]*entities.RegionalWar{}} }

func (r *mockWarsRepo) FindActiveWar(_ context.Context, period string) (*entities.RegionalWar, error) {
	r.mu.Lock(); defer r.mu.Unlock()
	w, ok := r.wars[period]
	if !ok || w.Status != entities.WarStatusActive {
		return nil, errors.New("not found")
	}
	cp := *w; return &cp, nil
}

func (r *mockWarsRepo) EnsureWar(_ context.Context, period string, prizeKobo int64, startsAt, endsAt time.Time) error {
	r.mu.Lock(); defer r.mu.Unlock()
	if _, exists := r.wars[period]; exists { return nil }
	r.wars[period] = &entities.RegionalWar{
		ID: uuid.New(), Period: period, Status: entities.WarStatusActive,
		TotalPrizeKobo: prizeKobo, StartsAt: startsAt, EndsAt: endsAt,
	}
	return nil
}

func (r *mockWarsRepo) MarkResolved(_ context.Context, warID uuid.UUID) error {
	r.mu.Lock(); defer r.mu.Unlock()
	for _, w := range r.wars {
		if w.ID == warID { w.Status = entities.WarStatusCompleted; return nil }
	}
	return errors.New("war not found")
}

func (r *mockWarsRepo) ListWars(_ context.Context, limit int) ([]entities.RegionalWar, error) {
	r.mu.Lock(); defer r.mu.Unlock()
	out := make([]entities.RegionalWar, 0, len(r.wars))
	for _, w := range r.wars { out = append(out, *w) }
	return out, nil
}

func (r *mockWarsRepo) GetLeaderboard(_ context.Context, _, _ time.Time, limit int) ([]entities.LeaderboardEntry, error) {
	// Return synthetic entries for tests that need resolution to proceed
	return []entities.LeaderboardEntry{
		{State: "Lagos", TotalPoints: 5000, ActiveMembers: 1, Rank: 1},
		{State: "Abuja", TotalPoints: 3000, ActiveMembers: 1, Rank: 2},
		{State: "Kano",  TotalPoints: 1000, ActiveMembers: 1, Rank: 3},
	}, nil
}

func (r *mockWarsRepo) GetStateTotal(_ context.Context, state string, _, _ time.Time) (int64, error) {
	return 0, nil
}

func (r *mockWarsRepo) CreateWinners(_ context.Context, winners []entities.RegionalWarWinner) error {
	r.mu.Lock(); defer r.mu.Unlock()
	r.winners = append(r.winners, winners...)
	return nil
}

func (r *mockWarsRepo) GetWinnersForWar(_ context.Context, warID uuid.UUID) ([]entities.RegionalWarWinner, error) {
	r.mu.Lock(); defer r.mu.Unlock()
	var out []entities.RegionalWarWinner
	for _, w := range r.winners {
		if w.WarID == warID { out = append(out, w) }
	}
	return out, nil
}

func (r *mockWarsRepo) MarkWinnerPaid(_ context.Context, _ uuid.UUID) error { return nil }


func (r *mockWarsRepo) CreateSecondaryDraw(_ context.Context, draw *entities.WarSecondaryDraw, _ []entities.WarSecondaryDrawWinner) error {
	return nil
}

func (r *mockWarsRepo) GetSecondaryDrawsForWar(_ context.Context, _ uuid.UUID) ([]entities.WarSecondaryDraw, error) {
	return nil, nil
}

func (r *mockWarsRepo) GetSecondaryDrawByID(_ context.Context, _ uuid.UUID) (*entities.WarSecondaryDraw, error) {
	return nil, nil
}

func (r *mockWarsRepo) MarkSecondaryWinnerPaid(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID) error {
	return nil
}

func (r *mockWarsRepo) ListActiveUsersInState(_ context.Context, _ string, _, _ time.Time) ([]entities.UserRef, error) {
	return []entities.UserRef{
		{ID: uuid.New(), PhoneNumber: "08012345678"},
		{ID: uuid.New(), PhoneNumber: "08098765432"},
	}, nil
}

// verify interface compliance at compile time
var _ repositories.WarsRepository = (*mockWarsRepo)(nil)

// ─── in-memory UserRepository (wars-relevant subset) ────────────────────────

type mockWarsUserRepo struct{ user *entities.User }

func (r *mockWarsUserRepo) FindByID(_ context.Context, _ uuid.UUID) (*entities.User, error) {
	if r.user != nil { return r.user, nil }
	return nil, errors.New("not found")
}
func (r *mockWarsUserRepo) FindByPhoneNumber(_ context.Context, _ string) (*entities.User, error) { return nil, nil }
func (r *mockWarsUserRepo) FindByReferralCode(_ context.Context, _ string) (*entities.User, error) { return nil, nil }
func (r *mockWarsUserRepo) ExistsByPhoneNumber(_ context.Context, _ string) (bool, error)          { return false, nil }
func (r *mockWarsUserRepo) Create(_ context.Context, _ *entities.User) error                       { return nil }
func (r *mockWarsUserRepo) Update(_ context.Context, _ *entities.User) error                       { return nil }
func (r *mockWarsUserRepo) UpdateStreak(_ context.Context, _ uuid.UUID, _ int, _ interface{}) error { return nil }
func (r *mockWarsUserRepo) UpdateMoMo(_ context.Context, _ uuid.UUID, _ string, _ bool) error      { return nil }
func (r *mockWarsUserRepo) UpdateTier(_ context.Context, _ uuid.UUID, _ string) error              { return nil }
func (r *mockWarsUserRepo) UpdateWalletPassID(_ context.Context, _ uuid.UUID, _ string) error      { return nil }
func (r *mockWarsUserRepo) SetPointsExpiry(_ context.Context, _ uuid.UUID, _ interface{}) error    { return nil }
func (r *mockWarsUserRepo) GetWallet(_ context.Context, _ uuid.UUID) (*entities.Wallet, error)     { return &entities.Wallet{}, nil }
func (r *mockWarsUserRepo) GetWalletForUpdate(_ context.Context, _ uuid.UUID) (*entities.Wallet, error) { return &entities.Wallet{}, nil }
func (r *mockWarsUserRepo) UpdateWallet(_ context.Context, _ *entities.Wallet) error              { return nil }
func (r *mockWarsUserRepo) FindInactiveUsers(_ context.Context, _, _ int) ([]entities.User, error) { return nil, nil }
func (r *mockWarsUserRepo) FindUsersWithExpiringPoints(_ context.Context, _, _ int) ([]entities.User, error) { return nil, nil }
func (r *mockWarsUserRepo) CountByState(_ context.Context, _ string) (int64, error)               { return 0, nil }
func (r *mockWarsUserRepo) UpdateState(_ context.Context, _ uuid.UUID, _ string) error            { return nil }

var _ repositories.UserRepository = (*mockWarsUserRepo)(nil)

// ─── in-memory TransactionRepository (minimal stub) ─────────────────────────

type mockWarsTxRepo struct{}

func (r *mockWarsTxRepo) Save(_ context.Context, _ *entities.Transaction) error { return nil }
func (r *mockWarsTxRepo) SaveTx(_ context.Context, _ *gorm.DB, _ *entities.Transaction) error { return nil }
func (r *mockWarsTxRepo) FindByID(_ context.Context, _ uuid.UUID) (*entities.Transaction, error) { return nil, nil }
func (r *mockWarsTxRepo) FindByReference(_ context.Context, _ string) (*entities.Transaction, error) { return nil, nil }
func (r *mockWarsTxRepo) ListByUser(_ context.Context, _ uuid.UUID, _, _ int) ([]entities.Transaction, error) { return nil, nil }
func (r *mockWarsTxRepo) CountByUserAndType(_ context.Context, _ uuid.UUID, _ entities.TransactionType, _ int64) (int64, error) { return 0, nil }
func (r *mockWarsTxRepo) CountByPhoneAndTypeSince(_ context.Context, _ string, _ entities.TransactionType, _ int64) (int64, error) { return 0, nil }
func (r *mockWarsTxRepo) SumAmountByUserSince(_ context.Context, _ uuid.UUID, _ int64) (int64, error) { return 0, nil }
func (r *mockWarsTxRepo) DailyLiabilityTotal(_ context.Context) (int64, error) { return 0, nil }

var _ repositories.TransactionRepository = (*mockWarsTxRepo)(nil)

// ─── test helper: build service with mocks + minimal SQLite DB ───────────────

func setupWarsSvc(t *testing.T) (*services.RegionalWarsService, *mockWarsRepo) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Minimal schema for awardStateBonuses goroutine (best-effort, won't panic if tables missing)
	db.Exec(`CREATE TABLE IF NOT EXISTS users (id TEXT PRIMARY KEY, phone_number TEXT, state TEXT, is_active INTEGER DEFAULT 1)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS wallets (user_id TEXT PRIMARY KEY, pulse_points INTEGER DEFAULT 0, lifetime_points INTEGER DEFAULT 0)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS transactions (id TEXT PRIMARY KEY, user_id TEXT, phone_number TEXT, type TEXT, points_delta INTEGER, reference TEXT, created_at DATETIME)`)

	warsRepo := newMockWarsRepo()
	userRepo := &mockWarsUserRepo{}
	txRepo   := &mockWarsTxRepo{}
	cfg      := config.NewConfigManager(db)

	svc := services.NewRegionalWarsService(warsRepo, userRepo, txRepo, cfg, db)
	return svc, warsRepo
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestWarsService_EnsureActiveWar_CreatesRecord(t *testing.T) {
	svc, repo := setupWarsSvc(t)
	ctx := context.Background()

	if err := svc.EnsureActiveWar(ctx, 50_000_000); err != nil {
		t.Fatalf("EnsureActiveWar: %v", err)
	}
	if len(repo.wars) != 1 {
		t.Fatalf("expected 1 war, got %d", len(repo.wars))
	}
}

func TestWarsService_EnsureActiveWar_Idempotent(t *testing.T) {
	svc, repo := setupWarsSvc(t)
	ctx := context.Background()

	_ = svc.EnsureActiveWar(ctx, 50_000_000)
	_ = svc.EnsureActiveWar(ctx, 50_000_000)

	if len(repo.wars) != 1 {
		t.Fatalf("EnsureActiveWar must be idempotent, got %d wars", len(repo.wars))
	}
}

func TestWarsService_GetLeaderboard_ReturnsEntries(t *testing.T) {
	svc, _ := setupWarsSvc(t)
	ctx := context.Background()

	_ = svc.EnsureActiveWar(ctx, 50_000_000)
	entries, err := svc.GetLeaderboard(ctx, 10)
	if err != nil {
		t.Fatalf("GetLeaderboard: %v", err)
	}
	// Mock returns 3 entries; we expect at least 1
	if len(entries) == 0 {
		t.Fatal("expected at least 1 leaderboard entry")
	}
}

func TestWarsService_GetLeaderboard_PrizeDecorated(t *testing.T) {
	svc, _ := setupWarsSvc(t)
	ctx := context.Background()
	_ = svc.EnsureActiveWar(ctx, 100_000) // 1 kobo × 100,000

	entries, err := svc.GetLeaderboard(ctx, 10)
	if err != nil {
		t.Fatalf("GetLeaderboard: %v", err)
	}
	if len(entries) > 0 && entries[0].PrizeKobo == 0 {
		t.Fatal("top entry should have prize decorated")
	}
}

func TestWarsService_GetUserRank_UserWithState(t *testing.T) {
	svc, _ := setupWarsSvc(t)
	ctx := context.Background()
	_ = svc.EnsureActiveWar(ctx, 50_000_000)

	// Inject a user with state "Lagos" into the mock user repo
	// (the mock always returns nil for FindByID — so the error path returns gracefully)
	_, err := svc.GetUserRank(ctx, uuid.New())
	// Expected: error "not found" from mock user repo
	if err == nil {
		t.Fatal("expected error for unknown user")
	}
}

func TestWarsService_ResolveWar_NoActiveWar(t *testing.T) {
	svc, _ := setupWarsSvc(t)
	ctx := context.Background()

	_, err := svc.ResolveWar(ctx, "2000-01")
	if err == nil {
		t.Fatal("expected error resolving non-existent war")
	}
}

func TestWarsService_ResolveWar_DistributesPrizes(t *testing.T) {
	svc, repo := setupWarsSvc(t)
	ctx := context.Background()

	// Create a war
	period := time.Now().UTC().Format("2006-01")
	_ = svc.EnsureActiveWar(ctx, 50_000_000)

	_, err := svc.ResolveWar(ctx, period)
	if err != nil {
		t.Fatalf("ResolveWar: %v", err)
	}

	// Should have 3 winner rows (mock leaderboard returns 3 states)
	if len(repo.winners) == 0 {
		t.Fatal("expected winner rows after resolve")
	}

	// War should be COMPLETED
	w, ok := repo.wars[period]
	if !ok || w.Status != entities.WarStatusCompleted {
		t.Fatalf("war status should be COMPLETED, got %v", w)
	}
}

func TestWarsService_ResolveWar_PrizeSharesSumTo100(t *testing.T) {
	svc, repo := setupWarsSvc(t)
	ctx := context.Background()

	period := time.Now().UTC().Format("2006-01")
	_ = svc.EnsureActiveWar(ctx, 100_000_000) // ₦1m in kobo

	winners, err := svc.ResolveWar(ctx, period)
	if err != nil {
		t.Fatalf("ResolveWar: %v", err)
	}

	var total int64
	for _, e := range winners {
		total += e.PrizeKobo
	}
	// 50 + 30 + 20 = 100% of 100_000_000 = 100_000_000
	if total != 100_000_000 {
		t.Fatalf("prize shares should sum to 100%% of pool (100000000 kobo), got %d. winners: %+v, repo: %+v",
			total, winners, repo.winners)
	}
}

func TestWarsService_ListWars(t *testing.T) {
	svc, _ := setupWarsSvc(t)
	ctx := context.Background()

	_ = svc.EnsureActiveWar(ctx, 50_000_000)
	wars, err := svc.ListWars(ctx, 12)
	if err != nil {
		t.Fatalf("ListWars: %v", err)
	}
	if len(wars) == 0 {
		t.Fatal("expected at least one war in ListWars")
	}
}
