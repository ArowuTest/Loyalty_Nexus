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
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/infrastructure/config"
	"loyalty-nexus/internal/infrastructure/persistence"
)

func setupSpinDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		phone_number TEXT NOT NULL UNIQUE,
		full_name TEXT DEFAULT '',
		subscription_status TEXT NOT NULL DEFAULT 'FREE',
		is_active INTEGER NOT NULL DEFAULT 1,
		is_suspended INTEGER NOT NULL DEFAULT 0,
		spin_credits INTEGER NOT NULL DEFAULT 0,
		pulse_points INTEGER NOT NULL DEFAULT 0,
		streak_count INTEGER NOT NULL DEFAULT 0,
		tier TEXT NOT NULL DEFAULT 'BRONZE',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS wallets (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL UNIQUE,
		balance_kobo INTEGER NOT NULL DEFAULT 0,
		pulse_points INTEGER NOT NULL DEFAULT 0,
		spin_credits INTEGER NOT NULL DEFAULT 0,
		lifetime_points INTEGER NOT NULL DEFAULT 0,
		recharge_counter INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS transactions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		phone_number TEXT NOT NULL DEFAULT '',
		type TEXT NOT NULL,
		points_delta INTEGER NOT NULL DEFAULT 0,
		spin_delta INTEGER NOT NULL DEFAULT 0,
		amount REAL NOT NULL DEFAULT 0,
		balance_after REAL NOT NULL DEFAULT 0,
		reference TEXT NOT NULL DEFAULT '',
		metadata TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS spin_results (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		prize_type TEXT NOT NULL DEFAULT 'try_again',
		prize_value REAL NOT NULL DEFAULT 0,
		slot_index INTEGER NOT NULL DEFAULT 0,
		fulfillment_status TEXT NOT NULL DEFAULT 'na',
		fulfillment_ref TEXT NOT NULL DEFAULT '',
		mo_mo_number TEXT NOT NULL DEFAULT '',
		error_message TEXT NOT NULL DEFAULT '',
		retry_count INTEGER NOT NULL DEFAULT 0,
		claimed_at DATETIME,
		fulfilled_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS prize_pool (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT 'Try Again',
		prize_type TEXT NOT NULL DEFAULT 'try_again',
		base_value REAL NOT NULL DEFAULT 0,
		win_probability_weight INTEGER NOT NULL DEFAULT 10,
		daily_inventory_cap INTEGER,
		is_active INTEGER NOT NULL DEFAULT 1
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS network_configs (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT NOT NULL
	)`)
	// Seed a TRY_AGAIN prize (always safe - no fulfillment needed)
	db.Exec(`INSERT INTO prize_pool (id, name, prize_type, base_value, win_probability_weight, is_active) VALUES (?,?,?,?,?,1)`,
		uuid.New().String(), "Try Again", "try_again", 0.0, 10)
	return db
}

func seedWalletUser(db *gorm.DB, credits int) uuid.UUID {
	id := uuid.New()
	db.Exec(`INSERT INTO users (id, phone_number, spin_credits) VALUES (?,?,?)`,
		id.String(), "0801"+id.String()[:7], credits)
	db.Exec(`INSERT INTO wallets (id, user_id, spin_credits) VALUES (?,?,?)`,
		uuid.New().String(), id.String(), credits)
	return id
}

func newSpinSvc(db *gorm.DB) *services.SpinService {
	userRepo := persistence.NewPostgresUserRepository(db)
	txRepo := persistence.NewPostgresTransactionRepository(db)
	prizeRepo := persistence.NewPostgresPrizeRepository(db)
	cfg := config.NewConfigManager(db)
	return services.NewSpinService(userRepo, txRepo, prizeRepo, nil, nil, cfg, db)
}

func TestSpin_NoCredits_Fails(t *testing.T) {
	db := setupSpinDB(t)
	svc := newSpinSvc(db)
	userID := seedWalletUser(db, 0) // zero credits

	_, err := svc.PlaySpin(context.Background(), userID)
	if err == nil {
		t.Fatal("expected 'no spin credits' error, got nil")
	}
}

func TestSpin_WithCredits_Succeeds(t *testing.T) {
	db := setupSpinDB(t)
	svc := newSpinSvc(db)
	userID := seedWalletUser(db, 5)

	result, err := svc.PlaySpin(context.Background(), userID)
	if err != nil {
		t.Fatalf("PlaySpin with credits should succeed: %v", err)
	}
	if result == nil {
		t.Fatal("result must not be nil")
	}

	var credits int
	db.Raw("SELECT spin_credits FROM wallets WHERE user_id = ?", userID.String()).Scan(&credits)
	if credits != 4 {
		t.Fatalf("expected 4 credits remaining, got %d", credits)
	}
}

func TestSpin_DailyLimit_Enforced(t *testing.T) {
	db := setupSpinDB(t)
	db.Exec(`CREATE TABLE IF NOT EXISTS prize_pool (
		id TEXT PRIMARY KEY, name TEXT, prize_type TEXT, base_value REAL, win_probability_weight INTEGER, is_active INTEGER DEFAULT 1
	)`)
	db.Exec(`INSERT INTO network_configs (id, key, value) VALUES (?,?,?)`,
		uuid.New().String(), "spin_max_per_user_per_day", "2")
	db.Exec(`INSERT INTO prize_pool (id, name, prize_type, base_value, win_probability_weight, is_active) VALUES (?,?,?,?,?,1)\`,
		uuid.New().String(), "Try Again", "try_again", 0.0, 10)
	svc := newSpinSvc(db)
	userID := seedWalletUser(db, 10)

	// Pre-seed 2 spin_results today
	for i := 0; i < 2; i++ {
		db.Exec(`INSERT INTO spin_results (id, user_id, prize_type, fulfillment_status, created_at)
			VALUES (?,?,?,?,?)`,
			uuid.New().String(), userID.String(), "TRY_AGAIN", "N_A",
			time.Now().Add(-time.Minute))
	}

	_, err := svc.PlaySpin(context.Background(), userID)
	if err == nil {
		t.Fatal("expected daily limit error, got nil")
	}
}

func TestSpin_Result_PrizeType_Valid(t *testing.T) {
	db := setupSpinDB(t)
	svc := newSpinSvc(db)
	userID := seedWalletUser(db, 3)

	result, err := svc.PlaySpin(context.Background(), userID)
	if err != nil {
		t.Fatalf("PlaySpin: %v", err)
	}

	validTypes := map[entities.PrizeType]bool{
		entities.PrizeTryAgain:  true,
		entities.PrizeMoMoCash:  true,
		entities.PrizeAirtime:   true,
		entities.PrizeDataBundle:      true,
		entities.PrizePulsePoints: true,
		// entities.PrizeJackpot: (not defined)   true,
	}
	if !validTypes[result.SpinResult.PrizeType] {
		t.Fatalf("unexpected prize type: %s", result.SpinResult.PrizeType)
	}
}
