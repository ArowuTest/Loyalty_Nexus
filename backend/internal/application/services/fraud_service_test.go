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

// setupFraudDB creates an in-memory SQLite DB with the minimal schema needed.
func setupFraudDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS transactions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		type TEXT NOT NULL,
		reference TEXT,
		points_earned INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS spin_results (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS fraud_events (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		event_type TEXT NOT NULL,
		severity TEXT NOT NULL,
		details TEXT,
		resolved INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		is_active INTEGER DEFAULT 1
	)`)
	return db
}

func TestFraudService_CheckRecharge_DuplicateReference(t *testing.T) {
	db := setupFraudDB(t)
	svc := services.NewFraudService(db)
	ctx := context.Background()
	userID := uuid.New()
	ref := "REF-DUPLICATE-001"

	// Seed an existing transaction with same reference
	db.Exec(`INSERT INTO transactions (id, user_id, type, reference) VALUES (?,?,?,?)`,
		uuid.New().String(), userID.String(), "CREDIT", ref)

	err := svc.CheckRecharge(ctx, userID, 50000, ref)
	if err == nil {
		t.Fatal("expected duplicate error, got nil")
	}
}

func TestFraudService_CheckRecharge_VelocityLimit(t *testing.T) {
	db := setupFraudDB(t)
	svc := services.NewFraudService(db)
	ctx := context.Background()
	userID := uuid.New()

	// Seed 20 recharge transactions in the last 24h
	for i := 0; i < 20; i++ {
		db.Exec(`INSERT INTO transactions (id, user_id, type, reference, created_at) VALUES (?,?,?,?,?)`,
			uuid.New().String(), userID.String(), "CREDIT",
			"REF-VEL-"+uuid.New().String(), time.Now().Add(-time.Hour))
	}

	err := svc.CheckRecharge(ctx, userID, 200000, "REF-NEW-999")
	if err == nil {
		t.Fatal("expected velocity limit error, got nil")
	}
}

func TestFraudService_CheckRecharge_Clean(t *testing.T) {
	db := setupFraudDB(t)
	svc := services.NewFraudService(db)
	ctx := context.Background()
	userID := uuid.New()

	err := svc.CheckRecharge(ctx, userID, 500000, "REF-CLEAN-001")
	if err != nil {
		t.Fatalf("clean recharge should pass, got: %v", err)
	}
}

func TestFraudService_CheckSpin_Velocity(t *testing.T) {
	db := setupFraudDB(t)
	svc := services.NewFraudService(db)
	ctx := context.Background()
	userID := uuid.New()

	// Seed 10 spins in last 24h
	for i := 0; i < 10; i++ {
		db.Exec(`INSERT INTO spin_results (id, user_id, created_at) VALUES (?,?,?)`,
			uuid.New().String(), userID.String(), time.Now().Add(-time.Hour))
	}

	err := svc.CheckSpin(ctx, userID)
	if err == nil {
		t.Fatal("expected spin velocity error, got nil")
	}
}

func TestFraudService_CheckSpin_Clean(t *testing.T) {
	db := setupFraudDB(t)
	svc := services.NewFraudService(db)
	ctx := context.Background()
	userID := uuid.New()

	err := svc.CheckSpin(ctx, userID)
	if err != nil {
		t.Fatalf("clean spin should pass, got: %v", err)
	}
}

func TestFraudService_ListOpenEvents_Empty(t *testing.T) {
	db := setupFraudDB(t)
	svc := services.NewFraudService(db)
	ctx := context.Background()

	events, err := svc.ListOpenEvents(ctx, 50)
	if err != nil {
		t.Fatalf("ListOpenEvents: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestFraudService_SuspendUser(t *testing.T) {
	db := setupFraudDB(t)
	svc := services.NewFraudService(db)
	ctx := context.Background()
	userID := uuid.New()

	db.Exec(`INSERT INTO users (id, is_active) VALUES (?, 1)`, userID.String())

	err := svc.SuspendUser(ctx, userID, "test suspension")
	if err != nil {
		t.Fatalf("SuspendUser: %v", err)
	}

	var active bool
	db.Raw("SELECT is_active FROM users WHERE id = ?", userID.String()).Scan(&active)
	if active {
		t.Fatal("user should be suspended (is_active=false)")
	}
}
