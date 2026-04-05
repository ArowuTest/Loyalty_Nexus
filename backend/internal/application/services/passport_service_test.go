package services_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/infrastructure/config"
)

func setupPassportDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	// Create required tables
	db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		phone_number TEXT NOT NULL UNIQUE,
		tier TEXT NOT NULL DEFAULT 'BRONZE',
		streak_count INTEGER NOT NULL DEFAULT 0,
		lifetime_points INTEGER NOT NULL DEFAULT 0,
		total_spins INTEGER NOT NULL DEFAULT 0,
		studio_use_count INTEGER NOT NULL DEFAULT 0,
		total_referrals INTEGER NOT NULL DEFAULT 0,
		is_active INTEGER NOT NULL DEFAULT 1,
		last_recharge_at DATETIME,
		google_wallet_object_id TEXT,
		apple_pass_serial TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS wallets (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL UNIQUE,
		pulse_points INTEGER NOT NULL DEFAULT 0,
		spin_credits INTEGER NOT NULL DEFAULT 0,
		recharge_counter INTEGER NOT NULL DEFAULT 0
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS user_badges (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		badge_key TEXT NOT NULL,
		earned_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, badge_key)
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS network_configs (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS ghost_nudge_log (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL UNIQUE,
		nudged_at DATETIME
	)`)

	db.Exec(`CREATE TABLE IF NOT EXISTS passport_events (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		event_type TEXT NOT NULL,
		details TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	// Seed configs
	db.Exec(`INSERT INTO network_configs (key, value) VALUES ('spin_trigger_naira', '1000')`)
	db.Exec(`INSERT INTO network_configs (key, value) VALUES ('streak_expiry_hours', '24')`)

	return db
}

func TestPassportService_GetPassport(t *testing.T) {
	db := setupPassportDB(t)
	cfg := config.NewConfigManagerNoRefresh(db)
	svc := services.NewPassportService(db, cfg)
	ctx := context.Background()
	userID := uuid.New()
	db.Exec(`INSERT INTO users (id, phone_number, tier, streak_count, lifetime_points) VALUES (?, '2348012345678', 'SILVER', 5, 2500)`, userID)
	db.Exec(`INSERT INTO wallets (id, user_id, pulse_points, spin_credits, recharge_counter) VALUES (?, ?, 150, 2, 1650)`, uuid.New(), userID)
	db.Exec(`INSERT INTO user_badges (id, user_id, badge_key) VALUES (?, ?, 'first_recharge')`, uuid.New(), userID)

	passport, err := svc.GetPassport(ctx, userID)
	if err != nil {
		t.Fatalf("GetPassport failed: %v", err)
	}

	if passport.Tier != "SILVER" {
		t.Errorf("expected tier SILVER, got %s", passport.Tier)
	}
	if passport.StreakCount != 5 {
		t.Errorf("expected streak 5, got %d", passport.StreakCount)
	}
	if passport.LifetimePoints != 2500 {
		t.Errorf("expected lifetime points 2500, got %d", passport.LifetimePoints)
	}
	if passport.PulsePoints != 150 {
		t.Errorf("expected pulse points 150, got %d", passport.PulsePoints)
	}
	if passport.SpinCredits != 2 {
		t.Errorf("expected spin credits 2, got %d", passport.SpinCredits)
	}
	if len(passport.Badges) != 1 || passport.Badges[0].Key != "first_recharge" {
		t.Errorf("expected 1 badge 'first_recharge', got %v", passport.Badges)
	}
	if passport.NextTier != "GOLD" {
		t.Errorf("expected next tier GOLD, got %s", passport.NextTier)
	}
	if passport.PointsToNext != 7500 { // 10000 - 2500
		t.Errorf("expected points to next 7500, got %d", passport.PointsToNext)
	}
	// recharge_counter is 1650. spin_trigger_naira is 1000.
	// mod = 1650 % 1000 = 650. amountToNextSpin = 1000 - 650 = 350.
	if passport.AmountToNextSpin != 350 {
		t.Errorf("expected amount to next spin 350, got %d", passport.AmountToNextSpin)
	}
}

func TestPassportService_EvaluateBadges(t *testing.T) {
	db := setupPassportDB(t)
	cfg := config.NewConfigManagerNoRefresh(db)
	svc := services.NewPassportService(db, cfg)
	ctx := context.Background()

	userID := uuid.New()
	db.Exec(`INSERT INTO users (id, phone_number, streak_count, tier) VALUES (?, '2348012345678', 7, 'GOLD')`, userID)

	err := svc.EvaluateBadges(ctx, userID, "recharge", nil)
	if err != nil {
		t.Fatalf("EvaluateBadges failed: %v", err)
	}

	var count int64
	db.Table("user_badges").Where("user_id = ?", userID).Count(&count)
	// Expected badges: first_recharge, streak_7, silver_tier, gold_tier = 4 badges
	if count != 4 {
		t.Errorf("expected 4 badges, got %d", count)
	}
}

func TestPassportService_QR(t *testing.T) {
	db := setupPassportDB(t)
	cfg := config.NewConfigManagerNoRefresh(db)
	svc := services.NewPassportService(db, cfg)

	if err := os.Setenv("PASSPORT_QR_SECRET", "test-secret"); err != nil {
		t.Fatalf("set PASSPORT_QR_SECRET: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("PASSPORT_QR_SECRET"); err != nil {
			t.Logf("unset PASSPORT_QR_SECRET: %v", err)
		}
	}()

	userID := uuid.New()
	qr, err := svc.GenerateQRPayload(userID)
	if err != nil {
		t.Fatalf("GenerateQRPayload failed: %v", err)
	}

	verifiedID, err := svc.VerifyQRPayload(qr)
	if err != nil {
		t.Fatalf("VerifyQRPayload failed: %v", err)
	}

	if verifiedID != userID {
		t.Errorf("expected user ID %s, got %s", userID, verifiedID)
	}
}

func TestPassportService_BuildPKPass(t *testing.T) {
	db := setupPassportDB(t)
	cfg := config.NewConfigManagerNoRefresh(db)
	svc := services.NewPassportService(db, cfg)
	ctx := context.Background()
	userID := uuid.New()
	// Insert a GOLD-tier user (lifetime_points >= 10000 threshold)
	db.Exec(`INSERT INTO users (id, phone_number, tier, streak_count, lifetime_points, total_spins) VALUES (?, '2348012345678', 'GOLD', 3, 12000, 5)`, userID)
	db.Exec(`INSERT INTO wallets (id, user_id, pulse_points, spin_credits, recharge_counter) VALUES (?, ?, 500, 1, 0)`, uuid.New(), userID)

	pass, err := svc.BuildPKPass(ctx, userID)
	if err != nil {
		t.Fatalf("BuildPKPass failed: %v", err)
	}

	if pass.SerialNumber != userID.String() {
		t.Errorf("expected serial number %s, got %s", userID, pass.SerialNumber)
	}
	if pass.BackgroundColor != "rgb(180,140,0)" { // GOLD color
		t.Errorf("expected background color rgb(180,140,0), got %s", pass.BackgroundColor)
	}
	if len(pass.Generic.PrimaryFields) == 0 || pass.Generic.PrimaryFields[0].Value != "GOLD" {
		t.Errorf("expected primary field GOLD, got %v", pass.Generic.PrimaryFields)
	}
}

// Postgres-specific raw SQL tests (GhostNudgeCandidates, LogPassportEvent) are omitted from SQLite suite.
