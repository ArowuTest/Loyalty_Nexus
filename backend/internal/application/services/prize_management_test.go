package services_test

// ─── Prize Management & Probability Tests ────────────────────────────────────
//
// Covers:
//   1. GetAllPrizes — active-only vs include-inactive
//   2. CreatePrize — happy path with all fields
//   3. CreatePrize — probability budget guard (total > 10000 rejected)
//   4. UpdatePrize — partial field update
//   5. UpdatePrize — probability budget guard (update would exceed 10000)
//   6. DeletePrize — soft-delete (is_active = false)
//   7. GetPrizeProbabilitySummary — totals, remaining budget, per-prize percent
//   8. ReorderPrizes — sort_order updated in bulk
//   9. CreatePrize — all new fields (icon_name, terms, prize_code, variation_code)
//  10. GetAllPrizes — sort order is respected (sort_order ASC)

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/infrastructure/config"
	"loyalty-nexus/internal/infrastructure/persistence"
)

// ─── DB Setup ────────────────────────────────────────────────────────────────

var prizeTestCounter int

func setupPrizeDB(t *testing.T) *gorm.DB {
	t.Helper()
	prizeTestCounter++
	dsn := fmt.Sprintf("file:prize_test_%d?mode=memory&cache=shared", prizeTestCounter)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS prize_pool (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT 'Try Again',
		prize_code TEXT NOT NULL DEFAULT '',
		variation_code TEXT NOT NULL DEFAULT '',
		prize_type TEXT NOT NULL DEFAULT 'try_again',
		base_value REAL NOT NULL DEFAULT 0,
		win_probability_weight INTEGER NOT NULL DEFAULT 0,
		daily_inventory_cap INTEGER,
		is_active INTEGER NOT NULL DEFAULT 1,
		is_no_win INTEGER NOT NULL DEFAULT 0,
		no_win_message TEXT NOT NULL DEFAULT '',
		color_scheme TEXT NOT NULL DEFAULT '',
		icon_name TEXT NOT NULL DEFAULT '',
		sort_order INTEGER NOT NULL DEFAULT 0,
		minimum_recharge INTEGER NOT NULL DEFAULT 0,
		terms_and_conditions TEXT NOT NULL DEFAULT ''
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS network_configs (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		value TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS spin_tiers (
		id TEXT PRIMARY KEY,
		tier_name TEXT NOT NULL DEFAULT '',
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
	return db
}

func newPrizeSpinSvc(db *gorm.DB) *services.SpinService {
	userRepo  := persistence.NewPostgresUserRepository(db)
	txRepo    := persistence.NewPostgresTransactionRepository(db)
	prizeRepo := persistence.NewPostgresPrizeRepository(db)
	cfg       := config.NewConfigManager(db)
	return services.NewSpinService(userRepo, txRepo, prizeRepo, nil, nil, cfg, db)
}

// ─── Helper: seed a prize directly ───────────────────────────────────────────

func seedPrize(db *gorm.DB, name, prizeType string, weight int, isActive bool) uuid.UUID {
	id := uuid.New()
	activeInt := 0
	if isActive {
		activeInt = 1
	}
	db.Exec(`INSERT INTO prize_pool (id, name, prize_type, base_value, win_probability_weight, is_active) VALUES (?,?,?,?,?,?)`,
		id.String(), name, prizeType, 0.0, weight, activeInt)
	return id
}

// ─── Tests ───────────────────────────────────────────────────────────────────

// 1. GetAllPrizes — active-only by default, include-inactive when flag is set
func TestGetAllPrizes_ActiveOnly(t *testing.T) {
	db := setupPrizeDB(t)
	svc := newPrizeSpinSvc(db)

	seedPrize(db, "Active Prize", "try_again", 500, true)
	seedPrize(db, "Inactive Prize", "try_again", 200, false)

	prizes, err := svc.GetAllPrizes(context.Background()) // default: active only
	if err != nil {
		t.Fatalf("GetAllPrizes: %v", err)
	}
	if len(prizes) != 1 {
		t.Fatalf("expected 1 active prize, got %d", len(prizes))
	}
	if prizes[0].Name != "Active Prize" {
		t.Fatalf("expected 'Active Prize', got %q", prizes[0].Name)
	}
}

func TestGetAllPrizes_IncludeInactive(t *testing.T) {
	db := setupPrizeDB(t)
	svc := newPrizeSpinSvc(db)

	seedPrize(db, "Active Prize", "try_again", 500, true)
	seedPrize(db, "Inactive Prize", "try_again", 200, false)

	prizes, err := svc.GetAllPrizes(context.Background(), true) // include inactive
	if err != nil {
		t.Fatalf("GetAllPrizes (include inactive): %v", err)
	}
	if len(prizes) != 2 {
		t.Fatalf("expected 2 prizes (active+inactive), got %d", len(prizes))
	}
}

// 2. CreatePrize — happy path with all fields
func TestCreatePrize_AllFields(t *testing.T) {
	db := setupPrizeDB(t)
	svc := newPrizeSpinSvc(db)

	data := map[string]interface{}{
		"name":                   "MoMo Cash ₦500",
		"prize_type":             "momo_cash",
		"base_value":             float64(500),
		"win_probability_weight": float64(300),
		"is_active":              true,
		"is_no_win":              false,
		"color_scheme":           "#FFD700",
		"icon_name":              "momo_icon",
		"sort_order":             float64(1),
		"minimum_recharge":       float64(100000),
		"terms_and_conditions":   "Valid for 30 days",
		"prize_code":             "MOMO500",
		"variation_code":         "NG_MOMO",
		"no_win_message":         "",
	}

	prize, err := svc.CreatePrize(context.Background(), data)
	if err != nil {
		t.Fatalf("CreatePrize: %v", err)
	}
	if prize.Name != "MoMo Cash ₦500" {
		t.Errorf("expected name 'MoMo Cash ₦500', got %q", prize.Name)
	}
	if prize.ProbWeight != 300 {
		t.Errorf("expected weight 300, got %v", prize.ProbWeight)
	}
	if prize.ColorScheme != "#FFD700" {
		t.Errorf("expected color '#FFD700', got %q", prize.ColorScheme)
	}
	if prize.IconName != "momo_icon" {
		t.Errorf("expected icon_name 'momo_icon', got %q", prize.IconName)
	}
	if prize.TermsAndConditions != "Valid for 30 days" {
		t.Errorf("expected terms 'Valid for 30 days', got %q", prize.TermsAndConditions)
	}
	if prize.PrizeCode != "MOMO500" {
		t.Errorf("expected prize_code 'MOMO500', got %q", prize.PrizeCode)
	}
	if prize.VariationCode != "NG_MOMO" {
		t.Errorf("expected variation_code 'NG_MOMO', got %q", prize.VariationCode)
	}
	if prize.MinimumRecharge != 100000 {
		t.Errorf("expected minimum_recharge 100000, got %d", prize.MinimumRecharge)
	}
}

// 3. CreatePrize — probability budget guard: total > 10000 must be rejected
func TestCreatePrize_ExceedsBudget_Rejected(t *testing.T) {
	db := setupPrizeDB(t)
	svc := newPrizeSpinSvc(db)

	// Seed 9800 weight already used
	seedPrize(db, "Big Prize", "momo_cash", 9800, true)

	// Attempt to add 300 more (9800 + 300 = 10100 > 10000)
	_, err := svc.CreatePrize(context.Background(), map[string]interface{}{
		"name":                   "Overflow Prize",
		"prize_type":             "airtime",
		"win_probability_weight": float64(300),
	})
	if err == nil {
		t.Fatal("expected budget-exceeded error, got nil")
	}
}

// 4. UpdatePrize — partial field update (only name and color)
func TestUpdatePrize_PartialFields(t *testing.T) {
	db := setupPrizeDB(t)
	svc := newPrizeSpinSvc(db)

	id := seedPrize(db, "Old Name", "try_again", 500, true)

	updated, err := svc.UpdatePrize(context.Background(), id, map[string]interface{}{
		"name":         "New Name",
		"color_scheme": "#123456",
	})
	if err != nil {
		t.Fatalf("UpdatePrize: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("expected name 'New Name', got %q", updated.Name)
	}
	if updated.ColorScheme != "#123456" {
		t.Errorf("expected color '#123456', got %q", updated.ColorScheme)
	}
	// Weight should be unchanged
	if updated.ProbWeight != 500 {
		t.Errorf("expected weight 500 unchanged, got %v", updated.ProbWeight)
	}
}

// 5. UpdatePrize — probability budget guard
func TestUpdatePrize_ExceedsBudget_Rejected(t *testing.T) {
	db := setupPrizeDB(t)
	svc := newPrizeSpinSvc(db)

	// Prize A: 9000 weight
	seedPrize(db, "Prize A", "momo_cash", 9000, true)
	// Prize B: 500 weight (we will try to update it to 1500, making total 10500)
	idB := seedPrize(db, "Prize B", "airtime", 500, true)

	_, err := svc.UpdatePrize(context.Background(), idB, map[string]interface{}{
		"win_probability_weight": float64(1500), // 9000 + 1500 = 10500 > 10000
	})
	if err == nil {
		t.Fatal("expected budget-exceeded error on update, got nil")
	}
}

// 6. DeletePrize — soft-delete sets is_active = false
func TestDeletePrize_SoftDelete(t *testing.T) {
	db := setupPrizeDB(t)
	svc := newPrizeSpinSvc(db)

	id := seedPrize(db, "To Delete", "try_again", 100, true)

	if err := svc.DeletePrize(context.Background(), id); err != nil {
		t.Fatalf("DeletePrize: %v", err)
	}

	// Should not appear in active list
	prizes, _ := svc.GetAllPrizes(context.Background())
	for _, p := range prizes {
		if p.ID == id {
			t.Fatal("deleted prize still appears in active list")
		}
	}

	// Should appear in include-inactive list
	allPrizes, _ := svc.GetAllPrizes(context.Background(), true)
	found := false
	for _, p := range allPrizes {
		if p.ID == id {
			found = true
			if p.IsActive {
				t.Fatal("deleted prize still has is_active=true")
			}
		}
	}
	if !found {
		t.Fatal("deleted prize not found in include-inactive list")
	}
}

// 7. GetPrizeProbabilitySummary — totals, remaining budget, per-prize percent
func TestGetPrizeProbabilitySummary(t *testing.T) {
	db := setupPrizeDB(t)
	svc := newPrizeSpinSvc(db)

	seedPrize(db, "Prize A", "momo_cash", 4000, true)
	seedPrize(db, "Prize B", "airtime", 3000, true)
	seedPrize(db, "Prize C (inactive)", "data", 1000, false)

	summary, err := svc.GetPrizeProbabilitySummary(context.Background())
	if err != nil {
		t.Fatalf("GetPrizeProbabilitySummary: %v", err)
	}

	// Only active prizes count toward total
	if summary.TotalWeight != 7000 {
		t.Errorf("expected TotalWeight=7000, got %v", summary.TotalWeight)
	}
	if summary.RemainingBudget != 3000 {
		t.Errorf("expected RemainingBudget=3000, got %v", summary.RemainingBudget)
	}
	if summary.PercentUsed != 70.0 {
		t.Errorf("expected PercentUsed=70.0, got %.2f", summary.PercentUsed)
	}
	// All 3 prizes (including inactive) should appear in the items list
	if len(summary.Prizes) != 3 {
		t.Errorf("expected 3 prize items in summary, got %d", len(summary.Prizes))
	}
	// Verify Prize A percent
	for _, item := range summary.Prizes {
		if item.Name == "Prize A" {
			if item.Percent != 40.0 {
				t.Errorf("Prize A: expected percent=40.0, got %.2f", item.Percent)
			}
		}
	}
}

// 8. ReorderPrizes — sort_order updated in bulk
func TestReorderPrizes(t *testing.T) {
	db := setupPrizeDB(t)
	svc := newPrizeSpinSvc(db)

	// Seed 3 prizes
	idA := seedPrize(db, "Prize A", "try_again", 100, true)
	idB := seedPrize(db, "Prize B", "airtime", 200, true)
	idC := seedPrize(db, "Prize C", "momo_cash", 300, true)

	// Reorder: C first, A second, B third
	err := svc.ReorderPrizes(context.Background(), []uuid.UUID{idC, idA, idB})
	if err != nil {
		t.Fatalf("ReorderPrizes: %v", err)
	}

	// Fetch and verify sort_order
	prizes, _ := svc.GetAllPrizes(context.Background())
	orderMap := map[string]int{}
	for _, p := range prizes {
		orderMap[p.Name] = p.SortOrder
	}
	if orderMap["Prize C"] != 0 {
		t.Errorf("Prize C: expected sort_order=0, got %d", orderMap["Prize C"])
	}
	if orderMap["Prize A"] != 1 {
		t.Errorf("Prize A: expected sort_order=1, got %d", orderMap["Prize A"])
	}
	if orderMap["Prize B"] != 2 {
		t.Errorf("Prize B: expected sort_order=2, got %d", orderMap["Prize B"])
	}
}

// 9. UpdatePrize — new fields (icon_name, terms, prize_code, variation_code)
func TestUpdatePrize_NewFields(t *testing.T) {
	db := setupPrizeDB(t)
	svc := newPrizeSpinSvc(db)

	id := seedPrize(db, "Data Bundle 1GB", "data", 400, true)

	updated, err := svc.UpdatePrize(context.Background(), id, map[string]interface{}{
		"icon_name":            "data_bundle",
		"terms_and_conditions": "One-time use, expires in 24h",
		"prize_code":           "DATA1GB",
		"variation_code":       "NG_DATA_1GB",
	})
	if err != nil {
		t.Fatalf("UpdatePrize new fields: %v", err)
	}
	if updated.IconName != "data_bundle" {
		t.Errorf("expected icon_name 'data_bundle', got %q", updated.IconName)
	}
	if updated.TermsAndConditions != "One-time use, expires in 24h" {
		t.Errorf("expected terms, got %q", updated.TermsAndConditions)
	}
	if updated.PrizeCode != "DATA1GB" {
		t.Errorf("expected prize_code 'DATA1GB', got %q", updated.PrizeCode)
	}
	if updated.VariationCode != "NG_DATA_1GB" {
		t.Errorf("expected variation_code 'NG_DATA_1GB', got %q", updated.VariationCode)
	}
}

// 10. GetAllPrizes — sort_order ASC is respected
func TestGetAllPrizes_SortOrderRespected(t *testing.T) {
	db := setupPrizeDB(t)
	svc := newPrizeSpinSvc(db)

	// Insert in reverse order intentionally
	db.Exec(`INSERT INTO prize_pool (id, name, prize_type, win_probability_weight, is_active, sort_order) VALUES (?,?,?,?,1,?)`,
		uuid.New().String(), "Third", "try_again", 100, 3)
	db.Exec(`INSERT INTO prize_pool (id, name, prize_type, win_probability_weight, is_active, sort_order) VALUES (?,?,?,?,1,?)`,
		uuid.New().String(), "First", "try_again", 200, 1)
	db.Exec(`INSERT INTO prize_pool (id, name, prize_type, win_probability_weight, is_active, sort_order) VALUES (?,?,?,?,1,?)`,
		uuid.New().String(), "Second", "try_again", 150, 2)

	prizes, err := svc.GetAllPrizes(context.Background())
	if err != nil {
		t.Fatalf("GetAllPrizes: %v", err)
	}
	if len(prizes) != 3 {
		t.Fatalf("expected 3 prizes, got %d", len(prizes))
	}
	if prizes[0].Name != "First" {
		t.Errorf("expected first prize to be 'First', got %q", prizes[0].Name)
	}
	if prizes[1].Name != "Second" {
		t.Errorf("expected second prize to be 'Second', got %q", prizes[1].Name)
	}
	if prizes[2].Name != "Third" {
		t.Errorf("expected third prize to be 'Third', got %q", prizes[2].Name)
	}
}

// 11. CreatePrize — name required validation
func TestCreatePrize_NameRequired(t *testing.T) {
	db := setupPrizeDB(t)
	svc := newPrizeSpinSvc(db)

	_, err := svc.CreatePrize(context.Background(), map[string]interface{}{
		"prize_type":             "airtime",
		"win_probability_weight": float64(100),
	})
	if err == nil {
		t.Fatal("expected 'name is required' error, got nil")
	}
}

// 12. CreatePrize — prize_type required validation
func TestCreatePrize_PrizeTypeRequired(t *testing.T) {
	db := setupPrizeDB(t)
	svc := newPrizeSpinSvc(db)

	_, err := svc.CreatePrize(context.Background(), map[string]interface{}{
		"name":                   "Some Prize",
		"win_probability_weight": float64(100),
	})
	if err == nil {
		t.Fatal("expected 'prize_type is required' error, got nil")
	}
}

// 13. GetPrize — not found returns error
func TestGetPrize_NotFound(t *testing.T) {
	db := setupPrizeDB(t)
	svc := newPrizeSpinSvc(db)

	_, err := svc.GetPrize(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected 'prize not found' error, got nil")
	}
}
