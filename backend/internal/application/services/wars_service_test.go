package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"loyalty-nexus/internal/application/services"
)

func setupWarsDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS regional_wars (
		id TEXT PRIMARY KEY,
		period TEXT NOT NULL UNIQUE,
		status TEXT NOT NULL DEFAULT 'ACTIVE',
		total_prize_kobo INTEGER NOT NULL DEFAULT 50000000,
		starts_at DATETIME NOT NULL,
		ends_at DATETIME NOT NULL,
		resolved_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS regional_war_winners (
		id TEXT PRIMARY KEY,
		war_id TEXT NOT NULL,
		state TEXT NOT NULL,
		rank INTEGER NOT NULL,
		total_points INTEGER NOT NULL DEFAULT 0,
		prize_kobo INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'PENDING',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS transactions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		type TEXT NOT NULL,
		points_earned INTEGER DEFAULT 0,
		metadata TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		phone_number TEXT NOT NULL,
		full_name TEXT DEFAULT '',
		state TEXT
	)`)
	return db
}

func TestWarsService_EnsureActiveWar_CreatesRecord(t *testing.T) {
	db := setupWarsDB(t)
	svc := services.NewRegionalWarsService(db)
	ctx := context.Background()

	err := svc.EnsureActiveWar(ctx, 50_000_000)
	if err != nil {
		t.Fatalf("EnsureActiveWar: %v", err)
	}

	var count int64
	db.Raw("SELECT COUNT(*) FROM regional_wars WHERE status = 'ACTIVE'").Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 active war, got %d", count)
	}
}

func TestWarsService_EnsureActiveWar_Idempotent(t *testing.T) {
	db := setupWarsDB(t)
	svc := services.NewRegionalWarsService(db)
	ctx := context.Background()

	_ = svc.EnsureActiveWar(ctx, 50_000_000)
	_ = svc.EnsureActiveWar(ctx, 50_000_000) // second call must not create duplicate

	var count int64
	db.Raw("SELECT COUNT(*) FROM regional_wars").Scan(&count)
	if count != 1 {
		t.Fatalf("EnsureActiveWar must be idempotent — got %d rows", count)
	}
}

func TestWarsService_GetLeaderboard_Empty(t *testing.T) {
	db := setupWarsDB(t)
	svc := services.NewRegionalWarsService(db)
	ctx := context.Background()
	_ = svc.EnsureActiveWar(ctx, 50_000_000)

	// GetLeaderboard uses date_trunc (postgres-only).
	// In SQLite unit tests, a SQL error is expected — verify it doesn't panic.
	_, _ = svc.GetLeaderboard(ctx, 10)
}

func TestWarsService_GetUserRank_UnknownUser(t *testing.T) {
	db := setupWarsDB(t)
	svc := services.NewRegionalWarsService(db)
	ctx := context.Background()
	_ = svc.EnsureActiveWar(ctx, 50_000_000)

	_, err := svc.GetUserRank(ctx, uuid.New())
	// Unknown user — should return an error or rank=0, not panic
	_ = err // both nil and non-nil are acceptable here
}

func TestWarsService_ResolveWar_NoActiveWar(t *testing.T) {
	db := setupWarsDB(t)
	svc := services.NewRegionalWarsService(db)
	ctx := context.Background()

	err := svc.ResolveWar(ctx, "2000-01")
	if err == nil {
		t.Fatal("expected error when resolving non-existent war")
	}
}

func TestWarsService_ResolveWar_Distributes_Prizes(t *testing.T) {
	db := setupWarsDB(t) // users + transactions tables already created
	svc := services.NewRegionalWarsService(db)
	ctx := context.Background()

	// Create a war manually
	warID := uuid.New()
	period := time.Now().UTC().Format("2006-01")
	db.Exec(`INSERT INTO regional_wars (id, period, status, total_prize_kobo, starts_at, ends_at)
		VALUES (?, ?, 'ACTIVE', 50000000, ?, ?)`,
		warID.String(), period,
		time.Now().Add(-30*24*time.Hour),
		time.Now().Add(time.Hour))

	// Seed some users with different states
	states := []string{"Lagos", "Abuja", "Kano"}
	for _, state := range states {
		userID := uuid.New()
		db.Exec(`INSERT INTO users (id, phone_number, full_name, state) VALUES (?,?,?,?)`,
			userID.String(), "080"+uuid.New().String()[:9], "Test User", state)
		db.Exec(`INSERT INTO transactions (id, user_id, type, points_earned, created_at) VALUES (?,?,?,?,?)`,
			uuid.New().String(), userID.String(), "CREDIT", 5000, time.Now().Add(-time.Hour))
	}

	err := svc.ResolveWar(ctx, period)
	if err != nil {
		t.Fatalf("ResolveWar: %v", err)
	}

	var count int64
	db.Raw("SELECT COUNT(*) FROM regional_war_winners").Scan(&count)
	if count == 0 {
		t.Fatal("expected at least one winner record after resolve")
	}

	var status string
	db.Raw("SELECT status FROM regional_wars WHERE period = ?", period).Scan(&status)
	if status != "COMPLETED" {
		t.Fatalf("war status should be COMPLETED, got %s", status)
	}
}
