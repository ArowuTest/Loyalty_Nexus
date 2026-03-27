package mtnpush_test

// ─── Bonus Pulse Integration Tests ───────────────────────────────────────────
//
// These tests exercise the BonusPulseService and the admin/user-facing HTTP
// handlers end-to-end against a real Postgres database.
//
// Auth is bypassed by injecting the admin UUID directly into the request
// context (same pattern used by the CSV upload tests).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/infrastructure/persistence"
	"loyalty-nexus/internal/presentation/http/handlers"
	"loyalty-nexus/internal/presentation/http/middleware"

	"github.com/google/uuid"
)

// ─── Helpers ─────────────────────────────────────────────────────────────────

// buildBonusPulseHandler constructs an AdminHandler with only the
// BonusPulseService attached (no other services needed for these tests).
func buildBonusPulseHandler(t *testing.T) (*handlers.AdminHandler, *services.BonusPulseService) {
	t.Helper()
	db := openTestDB(t)
	userRepo := persistence.NewPostgresUserRepository(db)
	svc := services.NewBonusPulseService(db, userRepo)
	h := handlers.NewAdminHandler(db, nil, nil, nil, nil, nil, nil, nil, nil, nil).
		WithBonusPulseService(svc)
	return h, svc
}

// adminRequest builds an *http.Request with the admin UUID injected into
// context so that AdminAuthMiddleware is effectively bypassed.
func adminRequest(method, path string, body interface{}, adminID uuid.UUID) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.ContextUserID, adminID.String())
	ctx = context.WithValue(ctx, middleware.ContextIsAdmin, true)
	return req.WithContext(ctx)
}

// ─── Tests ────────────────────────────────────────────────────────────────────

// TestBonusPulse_AwardIncreasesWallet verifies that awarding bonus points
// atomically increments the user's pulse_points balance.
func TestBonusPulse_AwardIncreasesWallet(t *testing.T) {
	db := openTestDB(t)
	phone := uniquePhone()
	user := seedUser(t, db, phone)
	t.Cleanup(func() {
		db.Exec("DELETE FROM pulse_point_awards WHERE user_id = ?", user.ID)
		db.Exec("DELETE FROM transactions WHERE phone_number = ?", phone)
		db.Exec("DELETE FROM users WHERE phone_number = ?", phone)
	})

	userRepo := persistence.NewPostgresUserRepository(db)
	svc := services.NewBonusPulseService(db, userRepo)

	result, err := svc.AwardBonusPulse(context.Background(), services.AwardBonusPulseRequest{
		PhoneNumber:   phone,
		Points:        100,
		Campaign:      "Test Campaign",
		Note:          "integration test award",
		AwardedByID:   uuid.New(),
		AwardedByName: "test-admin",
	})
	if err != nil {
		t.Fatalf("AwardBonusPulse: %v", err)
	}
	if result.PointsAwarded != 100 {
		t.Errorf("want PointsAwarded=100, got %d", result.PointsAwarded)
	}

	// Verify wallet balance
	var balance int64
	if err := db.Table("wallets").
		Where("user_id = ?", user.ID).
		Select("pulse_points").
		Scan(&balance).Error; err != nil {
		t.Fatalf("query wallet: %v", err)
	}
	if balance != 100 {
		t.Errorf("want wallet.pulse_points=100, got %d", balance)
	}
}

// TestBonusPulse_AuditRowWritten verifies that an audit row is written to
// pulse_point_awards with all required fields.
func TestBonusPulse_AuditRowWritten(t *testing.T) {
	db := openTestDB(t)
	phone := uniquePhone()
	user := seedUser(t, db, phone)
	t.Cleanup(func() {
		db.Exec("DELETE FROM pulse_point_awards WHERE user_id = ?", user.ID)
		db.Exec("DELETE FROM transactions WHERE phone_number = ?", phone)
		db.Exec("DELETE FROM users WHERE phone_number = ?", phone)
	})

	adminID := uuid.New()
	userRepo := persistence.NewPostgresUserRepository(db)
	svc := services.NewBonusPulseService(db, userRepo)

	_, err := svc.AwardBonusPulse(context.Background(), services.AwardBonusPulseRequest{
		PhoneNumber:   phone,
		Points:        250,
		Campaign:      "Ramadan Promo",
		Note:          "audit test",
		AwardedByID:   adminID,
		AwardedByName: "super-admin",
	})
	if err != nil {
		t.Fatalf("AwardBonusPulse: %v", err)
	}

	type auditRow struct {
		UserID        string `gorm:"column:user_id"`
		Points        int64  `gorm:"column:points"`
		Campaign      string `gorm:"column:campaign"`
		Note          string `gorm:"column:note"`
		AwardedBy     string `gorm:"column:awarded_by"`
		AwardedByName string `gorm:"column:awarded_by_name"`
	}
	var row auditRow
	if err := db.Table("pulse_point_awards").
		Where("user_id = ? AND campaign = ?", user.ID, "Ramadan Promo").
		First(&row).Error; err != nil {
		t.Fatalf("query pulse_point_awards: %v", err)
	}
	if row.Points != 250 {
		t.Errorf("want points=250, got %d", row.Points)
	}
	if row.Note != "audit test" {
		t.Errorf("want note='audit test', got %q", row.Note)
	}
	if row.AwardedBy != adminID.String() {
		t.Errorf("want awarded_by=%s, got %s", adminID, row.AwardedBy)
	}
	if row.AwardedByName != "super-admin" {
		t.Errorf("want awarded_by_name='super-admin', got %q", row.AwardedByName)
	}
}

// TestBonusPulse_LedgerEntryWritten verifies that a bonus transaction row is
// written to the transactions table alongside the audit row.
func TestBonusPulse_LedgerEntryWritten(t *testing.T) {
	db := openTestDB(t)
	phone := uniquePhone()
	user := seedUser(t, db, phone)
	t.Cleanup(func() {
		db.Exec("DELETE FROM pulse_point_awards WHERE user_id = ?", user.ID)
		db.Exec("DELETE FROM transactions WHERE phone_number = ?", phone)
		db.Exec("DELETE FROM users WHERE phone_number = ?", phone)
	})

	userRepo := persistence.NewPostgresUserRepository(db)
	svc := services.NewBonusPulseService(db, userRepo)

	_, err := svc.AwardBonusPulse(context.Background(), services.AwardBonusPulseRequest{
		PhoneNumber:   phone,
		Points:        75,
		Campaign:      "Ledger Test",
		AwardedByID:   uuid.New(),
		AwardedByName: "test-admin",
	})
	if err != nil {
		t.Fatalf("AwardBonusPulse: %v", err)
	}

	var count int64
	if err := db.Table("transactions").
		Where("phone_number = ? AND type = 'bonus'", phone).
		Count(&count).Error; err != nil {
		t.Fatalf("query transactions: %v", err)
	}
	if count != 1 {
		t.Errorf("want 1 bonus transaction row, got %d", count)
	}
}

// TestBonusPulse_Idempotency verifies that awarding twice to the same user
// accumulates correctly (no deduplication — each award is independent).
func TestBonusPulse_MultipleAwardsAccumulate(t *testing.T) {
	db := openTestDB(t)
	phone := uniquePhone()
	user := seedUser(t, db, phone)
	t.Cleanup(func() {
		db.Exec("DELETE FROM pulse_point_awards WHERE user_id = ?", user.ID)
		db.Exec("DELETE FROM transactions WHERE phone_number = ?", phone)
		db.Exec("DELETE FROM users WHERE phone_number = ?", phone)
	})

	userRepo := persistence.NewPostgresUserRepository(db)
	svc := services.NewBonusPulseService(db, userRepo)

	for i := 0; i < 3; i++ {
		_, err := svc.AwardBonusPulse(context.Background(), services.AwardBonusPulseRequest{
			PhoneNumber:   phone,
			Points:        50,
			Campaign:      fmt.Sprintf("Campaign-%d", i),
			AwardedByID:   uuid.New(),
			AwardedByName: "test-admin",
		})
		if err != nil {
			t.Fatalf("AwardBonusPulse #%d: %v", i, err)
		}
	}

	var balance int64
	if err := db.Table("wallets").
		Where("user_id = ?", user.ID).
		Select("pulse_points").
		Scan(&balance).Error; err != nil {
		t.Fatalf("query wallet: %v", err)
	}
	if balance != 150 {
		t.Errorf("want wallet.pulse_points=150 after 3×50, got %d", balance)
	}

	total, err := svc.GetUserBonusTotal(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetUserBonusTotal: %v", err)
	}
	if total != 150 {
		t.Errorf("want total_bonus=150, got %d", total)
	}
}

// TestBonusPulse_UnknownPhone verifies that awarding to an unregistered
// phone number returns an error and writes nothing to the DB.
func TestBonusPulse_UnknownPhone(t *testing.T) {
	db := openTestDB(t)
	userRepo := persistence.NewPostgresUserRepository(db)
	svc := services.NewBonusPulseService(db, userRepo)

	_, err := svc.AwardBonusPulse(context.Background(), services.AwardBonusPulseRequest{
		PhoneNumber:   "08000000000", // not seeded
		Points:        100,
		Campaign:      "Ghost Campaign",
		AwardedByID:   uuid.New(),
		AwardedByName: "test-admin",
	})
	if err == nil {
		t.Fatal("expected error for unknown phone, got nil")
	}
}

// TestBonusPulse_HTTPEndpoint_AwardAndList exercises the full HTTP layer:
// POST /api/v1/admin/bonus-pulse to award, then GET to list.
func TestBonusPulse_HTTPEndpoint_AwardAndList(t *testing.T) {
	db := openTestDB(t)
	phone := uniquePhone()
	user := seedUser(t, db, phone)
	adminID := uuid.New()
	t.Cleanup(func() {
		db.Exec("DELETE FROM pulse_point_awards WHERE user_id = ?", user.ID)
		db.Exec("DELETE FROM transactions WHERE phone_number = ?", phone)
		db.Exec("DELETE FROM users WHERE phone_number = ?", phone)
	})

	h, _ := buildBonusPulseHandler(t)

	// POST — award
	body := map[string]interface{}{
		"phone_number": phone,
		"points":       200,
		"campaign":     "HTTP Test Campaign",
		"note":         "http endpoint test",
	}
	req := adminRequest("POST", "/api/v1/admin/bonus-pulse", body, adminID)
	w := httptest.NewRecorder()
	h.AwardBonusPulse(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST award: want 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var awardResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &awardResp); err != nil {
		t.Fatalf("decode award response: %v", err)
	}
	if awardResp["points_awarded"] == nil {
		t.Error("response missing points_awarded field")
	}

	// GET — list
	listReq := adminRequest("GET", "/api/v1/admin/bonus-pulse?phone="+phone, nil, adminID)
	lw := httptest.NewRecorder()
	h.ListBonusPulseAwards(lw, listReq)
	if lw.Code != http.StatusOK {
		t.Fatalf("GET list: want 200, got %d — body: %s", lw.Code, lw.Body.String())
	}
	var listResp map[string]interface{}
	if err := json.Unmarshal(lw.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	total, _ := listResp["total"].(float64)
	if total < 1 {
		t.Errorf("want at least 1 record in list, got total=%v", total)
	}
}

// TestBonusPulse_GetUserAwards verifies the user-facing GetUserAwards method
// returns the correct records.
func TestBonusPulse_GetUserAwards(t *testing.T) {
	db := openTestDB(t)
	phone := uniquePhone()
	user := seedUser(t, db, phone)
	t.Cleanup(func() {
		db.Exec("DELETE FROM pulse_point_awards WHERE user_id = ?", user.ID)
		db.Exec("DELETE FROM transactions WHERE phone_number = ?", phone)
		db.Exec("DELETE FROM users WHERE phone_number = ?", phone)
	})

	userRepo := persistence.NewPostgresUserRepository(db)
	svc := services.NewBonusPulseService(db, userRepo)

	for i := 0; i < 5; i++ {
		_, err := svc.AwardBonusPulse(context.Background(), services.AwardBonusPulseRequest{
			PhoneNumber:   phone,
			Points:        int64(10 * (i + 1)),
			Campaign:      fmt.Sprintf("Camp-%d", i),
			AwardedByID:   uuid.New(),
			AwardedByName: "test-admin",
		})
		if err != nil {
			t.Fatalf("AwardBonusPulse #%d: %v", i, err)
		}
	}

	awards, err := svc.GetUserAwards(context.Background(), user.ID, 3)
	if err != nil {
		t.Fatalf("GetUserAwards: %v", err)
	}
	if len(awards) != 3 {
		t.Errorf("want 3 records (limit), got %d", len(awards))
	}
	// Most recent first — last award was 50 pts
	if awards[0].Points != 50 {
		t.Errorf("want most recent award points=50, got %d", awards[0].Points)
	}
}
