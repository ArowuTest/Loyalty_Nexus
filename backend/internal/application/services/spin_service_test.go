package services_test

import (
	"context"
	"fmt"
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

// ─── DB Setup ────────────────────────────────────────────────────────────────

var spinTestCounter int

func setupSpinDB(t *testing.T) *gorm.DB {
	t.Helper()
	spinTestCounter++
	dsn := fmt.Sprintf("file:spin_test_%d?mode=memory&cache=shared", spinTestCounter)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
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
		momo_verified INTEGER NOT NULL DEFAULT 0,
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
		amount INTEGER NOT NULL DEFAULT 0,
		balance_after INTEGER NOT NULL DEFAULT 0,
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
		fulfillment_status TEXT NOT NULL DEFAULT 'NA',
		claim_status TEXT NOT NULL DEFAULT 'PENDING',
		fulfillment_ref TEXT NOT NULL DEFAULT '',
		momo_number TEXT NOT NULL DEFAULT '',
		momo_claim_number TEXT NOT NULL DEFAULT '',
		bank_account_number TEXT NOT NULL DEFAULT '',
		bank_account_name TEXT NOT NULL DEFAULT '',
		bank_name TEXT NOT NULL DEFAULT '',
		admin_notes TEXT NOT NULL DEFAULT '',
		rejection_reason TEXT NOT NULL DEFAULT '',
		payment_reference TEXT NOT NULL DEFAULT '',
		error_message TEXT NOT NULL DEFAULT '',
		retry_count INTEGER NOT NULL DEFAULT 0,
		reviewed_by TEXT,
		reviewed_at DATETIME,
		expires_at DATETIME,
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
		minimum_recharge INTEGER NOT NULL DEFAULT 0,
		is_active INTEGER NOT NULL DEFAULT 1,
		is_no_win INTEGER NOT NULL DEFAULT 0,
		no_win_message TEXT NOT NULL DEFAULT '',
		color_scheme TEXT NOT NULL DEFAULT '',
		icon_name TEXT NOT NULL DEFAULT '',
		sort_order INTEGER NOT NULL DEFAULT 0,
		terms_and_conditions TEXT NOT NULL DEFAULT '',
		prize_code TEXT NOT NULL DEFAULT '',
		variation_code TEXT NOT NULL DEFAULT ''
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS network_configs (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT NOT NULL
	)`)
	// Spin tiers table (RechargeMax lift-and-shift)
	db.Exec(`CREATE TABLE IF NOT EXISTS spin_tiers (
		id TEXT PRIMARY KEY,
		tier_name TEXT NOT NULL,
		tier_display_name TEXT NOT NULL DEFAULT '',
		min_daily_amount INTEGER NOT NULL DEFAULT 0,
		max_daily_amount INTEGER NOT NULL DEFAULT 0,
		spins_per_day INTEGER NOT NULL DEFAULT 1,
		tier_color TEXT NOT NULL DEFAULT '',
		tier_icon TEXT NOT NULL DEFAULT '',
		tier_badge TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		sort_order INTEGER NOT NULL DEFAULT 0,
		is_active INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	// Seed default 4 tiers — column names match SpinTier entity GORM tags
	db.Exec(`INSERT OR IGNORE INTO spin_tiers (id, tier_name, tier_display_name, min_daily_amount, max_daily_amount, spins_per_day, sort_order) VALUES
		(?,?,?,?,?,?,?)`, uuid.New().String(), "Starter", "Starter", 100000, 199999, 1, 1)
	db.Exec(`INSERT OR IGNORE INTO spin_tiers (id, tier_name, tier_display_name, min_daily_amount, max_daily_amount, spins_per_day, sort_order) VALUES
		(?,?,?,?,?,?,?)`, uuid.New().String(), "Bronze", "Bronze", 200000, 499999, 2, 2)
	db.Exec(`INSERT OR IGNORE INTO spin_tiers (id, tier_name, tier_display_name, min_daily_amount, max_daily_amount, spins_per_day, sort_order) VALUES
		(?,?,?,?,?,?,?)`, uuid.New().String(), "Silver", "Silver", 500000, 999999, 3, 3)
	db.Exec(`INSERT OR IGNORE INTO spin_tiers (id, tier_name, tier_display_name, min_daily_amount, max_daily_amount, spins_per_day, sort_order) VALUES
		(?,?,?,?,?,?,?)`, uuid.New().String(), "Gold", "Gold", 1000000, 9999999999, 5, 4)

	// Seed a TRY_AGAIN prize (always safe — no fulfillment needed)
	db.Exec(`INSERT INTO prize_pool (id, name, prize_type, base_value, win_probability_weight, is_active, is_no_win, no_win_message) VALUES (?,?,?,?,?,1,1,?)`,
		uuid.New().String(), "Try Again", "try_again", 0.0, 10, "Better luck next time!")
	return db
}

// seedWalletUser creates a user + wallet with the given spin credits, and seeds
// a qualifying recharge transaction for today so the tier-based daily cap is unlocked.
func seedWalletUser(db *gorm.DB, credits int) uuid.UUID {
	id := uuid.New()
	phone := "0801" + id.String()[:7]
	db.Exec(`INSERT INTO users (id, phone_number, spin_credits) VALUES (?,?,?)`,
		id.String(), phone, credits)
	db.Exec(`INSERT INTO wallets (id, user_id, spin_credits) VALUES (?,?,?)`,
		uuid.New().String(), id.String(), credits)
	// Seed a ₦2,000 recharge today (200,000 kobo) → unlocks Bronze tier (2 spins/day)
	// Use ISO8601 format so SQLite's date() function can parse it correctly.
	db.Exec(`INSERT INTO transactions (id, user_id, phone_number, type, amount, reference, created_at) VALUES (?,?,?,?,?,?,?)`,
		uuid.New().String(), id.String(), phone, "recharge", 200000, "test_recharge_"+id.String(), time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	return id
}

func newSpinSvc(db *gorm.DB) *services.SpinService {
	userRepo := persistence.NewPostgresUserRepository(db)
	txRepo := persistence.NewPostgresTransactionRepository(db)
	prizeRepo := persistence.NewPostgresPrizeRepository(db)
	cfg := config.NewConfigManager(db)
	return services.NewSpinService(userRepo, txRepo, prizeRepo, nil, nil, cfg, db)
}

// ─── Spin Tests ───────────────────────────────────────────────────────────────

func TestSpin_NoCredits_Fails(t *testing.T) {
	db := setupSpinDB(t)
	svc := newSpinSvc(db)
	userID := seedWalletUser(db, 0) // zero credits — tier unlocked but wallet empty

	_, err := svc.PlaySpin(context.Background(), userID)
	if err == nil {
		t.Fatal("expected 'no spin credits' error, got nil")
	}
	if err.Error() == "" {
		t.Fatal("error message must not be empty")
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
	svc := newSpinSvc(db)
	// 10 credits + Bronze tier (2 spins from recharge) = cap of 12
	userID := seedWalletUser(db, 10)

	// Pre-seed 12 spin_results today to exactly hit the cap
	for i := 0; i < 12; i++ {
		db.Exec(`INSERT INTO spin_results (id, user_id, prize_type, fulfillment_status, claim_status, created_at)
			VALUES (?,?,?,?,?,?)`,
			uuid.New().String(), userID.String(), "try_again", "na", "claimed",
			time.Now().Add(-time.Minute))
	}

	_, err := svc.PlaySpin(context.Background(), userID)
	if err == nil {
		t.Fatal("expected daily limit error, got nil")
	}
}

func TestSpin_NoRechargeToday_SpinCreditsAllow(t *testing.T) {
	db := setupSpinDB(t)
	svc := newSpinSvc(db)

	// User with spin credits but NO recharge today.
	// New behaviour: credits act as direct passes — spin must succeed.
	id := uuid.New()
	phone := "0802" + id.String()[:7]
	db.Exec(`INSERT INTO users (id, phone_number, spin_credits) VALUES (?,?,?)`, id.String(), phone, 3)
	db.Exec(`INSERT INTO wallets (id, user_id, spin_credits) VALUES (?,?,?)`, uuid.New().String(), id.String(), 3)
	// No recharge transaction → tier cap = 0, but credits = 3 → total cap = 3

	_, err := svc.PlaySpin(context.Background(), id)
	if err != nil {
		t.Fatalf("user with spin credits should be able to spin without today recharge: %v", err)
	}

	// After 3 spins total the cap (= credits) should be exhausted
	for i := 0; i < 2; i++ {
		_, _ = svc.PlaySpin(context.Background(), id)
	}
	_, err = svc.PlaySpin(context.Background(), id)
	if err == nil {
		t.Fatal("expected daily limit error after all credits consumed, got nil")
	}
}

func TestSpin_GoldTier_AllowsFiveSpins(t *testing.T) {
	db := setupSpinDB(t)
	svc := newSpinSvc(db)

	// 2 credits + Gold tier recharge (5 spins/day) = cap of 7
	id := uuid.New()
	phone := "0803" + id.String()[:7]
	db.Exec(`INSERT INTO users (id, phone_number, spin_credits) VALUES (?,?,?)`, id.String(), phone, 2)
	db.Exec(`INSERT INTO wallets (id, user_id, spin_credits) VALUES (?,?,?)`, uuid.New().String(), id.String(), 2)
	db.Exec(`INSERT INTO transactions (id, user_id, phone_number, type, amount, reference, created_at) VALUES (?,?,?,?,?,?,?)`,
		uuid.New().String(), id.String(), phone, "recharge", 1000000, "gold_recharge_"+id.String(), time.Now().UTC().Format("2006-01-02T15:04:05Z"))

	// Should be able to spin 7 times (2 credits + 5 tier spins)
	for i := 0; i < 7; i++ {
		_, err := svc.PlaySpin(context.Background(), id)
		if err != nil {
			t.Fatalf("spin %d of 7 failed unexpectedly: %v", i+1, err)
		}
	}

	// 8th spin should be blocked
	_, err := svc.PlaySpin(context.Background(), id)
	if err == nil {
		t.Fatal("expected daily limit error on 8th spin (cap=7), got nil")
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
		entities.PrizeTryAgain:    true,
		entities.PrizeMoMoCash:    true,
		entities.PrizeAirtime:     true,
		entities.PrizeDataBundle:  true,
		entities.PrizePulsePoints: true,
	}
	if !validTypes[result.SpinResult.PrizeType] {
		t.Fatalf("unexpected prize type: %s", result.SpinResult.PrizeType)
	}
}

func TestSpin_IsNoWin_NoWinMessage(t *testing.T) {
	db := setupSpinDB(t)
	svc := newSpinSvc(db)
	userID := seedWalletUser(db, 1)

	// The only prize seeded is is_no_win=1 (Try Again), so the outcome must be try_again
	result, err := svc.PlaySpin(context.Background(), userID)
	if err != nil {
		t.Fatalf("PlaySpin: %v", err)
	}
	if result.SpinResult.PrizeType != entities.PrizeTryAgain {
		t.Fatalf("expected try_again prize type, got %s", result.SpinResult.PrizeType)
	}
	// Spin credit must have been deducted
	var credits int
	db.Raw("SELECT spin_credits FROM wallets WHERE user_id = ?", userID.String()).Scan(&credits)
	if credits != 0 {
		t.Fatalf("expected 0 credits after no-win spin, got %d", credits)
	}
}

// ─── Claim Flow Tests ─────────────────────────────────────────────────────────

func TestClaim_MoMoCash_PendingAdmin(t *testing.T) {
	db := setupSpinDB(t)
	// Seed a MoMo cash prize
	db.Exec(`INSERT INTO prize_pool (id, name, prize_type, base_value, win_probability_weight, is_active, is_no_win) VALUES (?,?,?,?,?,1,0)`,
		uuid.New().String(), "₦500 MoMo Cash", "momo_cash", 50000, 90)

	userID := seedWalletUser(db, 1)

	// Seed a pending spin result directly to test the claim path deterministically
	prizeRepo := persistence.NewPostgresPrizeRepository(db)
	claimSvc := services.NewClaimService(prizeRepo, persistence.NewPostgresUserRepository(db), nil, nil)

	spinID := uuid.New()
	db.Exec(`INSERT INTO spin_results (id, user_id, prize_type, prize_value, fulfillment_status, claim_status, expires_at, created_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		spinID.String(), userID.String(), "momo_cash", 50000, "PENDING_CLAIM", "PENDING",
		time.Now().Add(30*24*time.Hour), time.Now())

	result, err := claimSvc.ClaimPrize(context.Background(), userID, spinID, services.ClaimRequest{
		MoMoNumber: "08012345678",
	})
	if err != nil {
		t.Fatalf("ClaimPrize: %v", err)
	}
	if result.ClaimStatus != entities.ClaimPendingAdmin {
		t.Fatalf("expected claim_status=pending_admin_review, got %s", result.ClaimStatus)
	}
}

func TestClaim_Airtime_AutoClaimed(t *testing.T) {
	db := setupSpinDB(t)
	prizeRepo := persistence.NewPostgresPrizeRepository(db)
	userRepo := persistence.NewPostgresUserRepository(db)
	claimSvc := services.NewClaimService(prizeRepo, userRepo, nil, nil)

	userID := seedWalletUser(db, 1)
	spinID := uuid.New()
	db.Exec(`INSERT INTO spin_results (id, user_id, prize_type, prize_value, fulfillment_status, claim_status, expires_at, created_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		spinID.String(), userID.String(), "airtime", 50000, "PENDING_CLAIM", "PENDING",
		time.Now().Add(30*24*time.Hour), time.Now())

	result, err := claimSvc.ClaimPrize(context.Background(), userID, spinID, services.ClaimRequest{})
	if err != nil {
		t.Fatalf("ClaimPrize airtime: %v", err)
	}
	if result.ClaimStatus != entities.ClaimClaimed {
		t.Fatalf("expected claim_status=claimed for airtime, got %s", result.ClaimStatus)
	}
}

func TestClaim_Expired_Rejected(t *testing.T) {
	db := setupSpinDB(t)
	prizeRepo := persistence.NewPostgresPrizeRepository(db)
	userRepo := persistence.NewPostgresUserRepository(db)
	claimSvc := services.NewClaimService(prizeRepo, userRepo, nil, nil)

	userID := seedWalletUser(db, 1)
	spinID := uuid.New()
	// expires_at in the past
	db.Exec(`INSERT INTO spin_results (id, user_id, prize_type, prize_value, fulfillment_status, claim_status, expires_at, created_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		spinID.String(), userID.String(), "momo_cash", 50000, "PENDING_CLAIM", "PENDING",
		time.Now().Add(-1*time.Hour), time.Now().Add(-31*24*time.Hour))

	_, err := claimSvc.ClaimPrize(context.Background(), userID, spinID, services.ClaimRequest{
		MoMoNumber: "08012345678",
	})
	if err == nil {
		t.Fatal("expected expiry error, got nil")
	}
}

func TestClaim_UnauthorizedUser_Rejected(t *testing.T) {
	db := setupSpinDB(t)
	prizeRepo := persistence.NewPostgresPrizeRepository(db)
	userRepo := persistence.NewPostgresUserRepository(db)
	claimSvc := services.NewClaimService(prizeRepo, userRepo, nil, nil)

	ownerID := seedWalletUser(db, 1)
	otherID := seedWalletUser(db, 1)
	spinID := uuid.New()
	db.Exec(`INSERT INTO spin_results (id, user_id, prize_type, prize_value, fulfillment_status, claim_status, expires_at, created_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		spinID.String(), ownerID.String(), "momo_cash", 50000, "PENDING_CLAIM", "PENDING",
		time.Now().Add(30*24*time.Hour), time.Now())

	_, err := claimSvc.ClaimPrize(context.Background(), otherID, spinID, services.ClaimRequest{
		MoMoNumber: "08012345678",
	})
	if err == nil {
		t.Fatal("expected unauthorized error, got nil")
	}
}

func TestAdminClaim_ApproveCash_UpdatesStatus(t *testing.T) {
	db := setupSpinDB(t)
	prizeRepo := persistence.NewPostgresPrizeRepository(db)
	adminSvc := services.NewAdminClaimService(prizeRepo, nil)

	userID := seedWalletUser(db, 1)
	adminID := uuid.New()
	spinID := uuid.New()
	db.Exec(`INSERT INTO spin_results (id, user_id, prize_type, prize_value, fulfillment_status, claim_status, momo_claim_number, expires_at, created_at)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		spinID.String(), userID.String(), "momo_cash", 50000, "PENDING_CLAIM", "PENDING_ADMIN_REVIEW", "08012345678",
		time.Now().Add(30*24*time.Hour), time.Now())

	result, err := adminSvc.ApproveClaim(context.Background(), spinID, adminID, services.ApproveClaimRequest{
		AdminNotes:       "Verified and paid",
		PaymentReference: "PAY_REF_001",
	})
	if err != nil {
		t.Fatalf("ApproveClaim: %v", err)
	}
	if result.ClaimStatus != entities.ClaimApproved {
		t.Fatalf("expected claim_status=approved, got %s", result.ClaimStatus)
	}
}

func TestAdminClaim_RejectClaim_RequiresReason(t *testing.T) {
	db := setupSpinDB(t)
	prizeRepo := persistence.NewPostgresPrizeRepository(db)
	adminSvc := services.NewAdminClaimService(prizeRepo, nil)

	userID := seedWalletUser(db, 1)
	adminID := uuid.New()
	spinID := uuid.New()
	db.Exec(`INSERT INTO spin_results (id, user_id, prize_type, prize_value, fulfillment_status, claim_status, expires_at, created_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		spinID.String(), userID.String(), "momo_cash", 50000, "PENDING_CLAIM", "PENDING_ADMIN_REVIEW",
		time.Now().Add(30*24*time.Hour), time.Now())

	// Missing rejection reason — should fail
	_, err := adminSvc.RejectClaim(context.Background(), spinID, adminID, services.RejectClaimRequest{
		RejectionReason: "",
	})
	if err == nil {
		t.Fatal("expected error when rejection reason is empty, got nil")
	}

	// With reason — should succeed
	result, err := adminSvc.RejectClaim(context.Background(), spinID, adminID, services.RejectClaimRequest{
		RejectionReason: "Invalid MoMo number",
		AdminNotes:      "User provided wrong number",
	})
	if err != nil {
		t.Fatalf("RejectClaim: %v", err)
	}
	if result.ClaimStatus != entities.ClaimRejected {
		t.Fatalf("expected claim_status=rejected, got %s", result.ClaimStatus)
	}
}

func TestAdminClaim_GetStatistics(t *testing.T) {
	db := setupSpinDB(t)
	prizeRepo := persistence.NewPostgresPrizeRepository(db)
	adminSvc := services.NewAdminClaimService(prizeRepo, nil)

	userID := seedWalletUser(db, 1)
	// Seed 2 pending, 1 approved
	for i := 0; i < 2; i++ {
		db.Exec(`INSERT INTO spin_results (id, user_id, prize_type, prize_value, fulfillment_status, claim_status, expires_at, created_at)
			VALUES (?,?,?,?,?,?,?,?)`,
			uuid.New().String(), userID.String(), "momo_cash", 50000, "PENDING_CLAIM", "PENDING",
			time.Now().Add(30*24*time.Hour), time.Now())
	}
	db.Exec(`INSERT INTO spin_results (id, user_id, prize_type, prize_value, fulfillment_status, claim_status, expires_at, created_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		uuid.New().String(), userID.String(), "momo_cash", 50000, "COMPLETED", "APPROVED",
		time.Now().Add(30*24*time.Hour), time.Now())

	stats, err := adminSvc.GetStatistics(context.Background())
	if err != nil {
		t.Fatalf("GetStatistics: %v", err)
	}
	if stats.TotalClaims < 3 {
		t.Fatalf("expected at least 3 total claims, got %d", stats.TotalClaims)
	}
	if stats.ApprovedClaims != 1 {
		t.Fatalf("expected 1 approved claim, got %d", stats.ApprovedClaims)
	}
}
