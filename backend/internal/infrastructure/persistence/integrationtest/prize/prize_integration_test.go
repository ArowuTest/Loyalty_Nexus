// Package prize contains real Postgres integration tests for the prize management system.
//
// These tests run against a live loyalty_nexus_test Postgres database (not SQLite).
// They test the full stack: repository SQL → service logic → HTTP handler → JSON response.
//
// Prerequisites:
//
//	Postgres 14 running locally with:
//	  - user:     nexus_test / nexus_test
//	  - database: loyalty_nexus_test  (all migrations applied)
//
// Run with:
//
//	TEST_DATABASE_URL="postgres://nexus_test:nexus_test@localhost:5432/loyalty_nexus_test?sslmode=disable" \
//	JWT_SECRET="test-secret-32-chars-minimum-len!" \
//	AES_256_KEY="0000000000000000000000000000000000000000000000000000000000000000" \
//	go test ./internal/infrastructure/persistence/integrationtest/prize/... -v
package prize_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"loyalty-nexus/internal/application/services"
	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/infrastructure/config"
	"loyalty-nexus/internal/infrastructure/persistence"
	"loyalty-nexus/internal/presentation/http/handlers"
	"loyalty-nexus/internal/presentation/http/middleware"
)

// ─── TestMain: set required env vars so tests run without manual injection ────

func TestMain(m *testing.M) {
	// JWT_SECRET and AES_256_KEY are required by NewAuthService at construction
	// time. Set test-only values here so callers never need to export them.
	if os.Getenv("JWT_SECRET") == "" {
		if err := os.Setenv("JWT_SECRET", "integration-test-jwt-secret-32ch!"); err != nil {
			panic("TestMain: set JWT_SECRET: " + err.Error())
		}
	}
	if os.Getenv("AES_256_KEY") == "" {
		// 64 hex chars = 32 bytes
		if err := os.Setenv("AES_256_KEY", "0000000000000000000000000000000000000000000000000000000000000000"); err != nil {
			panic("TestMain: set AES_256_KEY: " + err.Error())
		}
	}
	os.Exit(m.Run())
}

// ─── Test DB Connection ───────────────────────────────────────────────────────

func testDSN() string {
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	return "postgres://nexus_test:nexus_test@localhost:5432/loyalty_nexus_test?sslmode=disable"
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.Open(testDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		if strings.EqualFold(os.Getenv("CI"), "true") {
			t.Fatalf("required Postgres integration database unavailable in CI: %v", err)
		}
		t.Skipf("Postgres not available (%v) — skipping integration tests", err)
	}
	return db
}

// ─── Test Isolation: each test runs inside a transaction that is rolled back ──

// withTx runs fn inside a Postgres transaction that is always rolled back,
// giving each test a clean slate without truncating tables.
func withTx(t *testing.T, db *gorm.DB, fn func(tx *gorm.DB)) {
	t.Helper()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin tx: %v", tx.Error)
	}
	defer tx.Rollback() //nolint:errcheck
	fn(tx)
}

// ─── Service + Handler factory ────────────────────────────────────────────────

func newSvc(db *gorm.DB) *services.SpinService {
	userRepo := persistence.NewPostgresUserRepository(db)
	txRepo := persistence.NewPostgresTransactionRepository(db)
	prizeRepo := persistence.NewPostgresPrizeRepository(db)
	fulfillCfgRepo := persistence.NewPostgresPrizeFulfillmentConfigRepository(db)
	cfg := config.NewConfigManagerNoRefresh(db)
	return services.NewSpinService(userRepo, txRepo, prizeRepo, nil, nil, cfg, db, fulfillCfgRepo)
}

// newAdminHandler builds the full AdminHandler wired to the given DB.
func newAdminHandler(db *gorm.DB) *handlers.AdminHandler {
	userRepo := persistence.NewPostgresUserRepository(db)
	txRepo := persistence.NewPostgresTransactionRepository(db)
	prizeRepo := persistence.NewPostgresPrizeRepository(db)
	fulfillCfgRepo := persistence.NewPostgresPrizeFulfillmentConfigRepository(db)
	cfg := config.NewConfigManagerNoRefresh(db)
	spinSvc := services.NewSpinService(userRepo, txRepo, prizeRepo, nil, nil, cfg, db, fulfillCfgRepo)
	drawSvc := services.NewDrawService(db)
	fraudSvc := services.NewFraudService(db)
	claimSvc := services.NewAdminClaimService(prizeRepo, nil)
	drawWindowSvc := services.NewDrawWindowService(db)
	return handlers.NewAdminHandler(db, cfg, spinSvc, drawSvc, drawWindowSvc, fraudSvc, nil, nil, claimSvc, nil)
}

// newAdminAuthSvc builds an AdminAuthService using the test JWT_SECRET env var.
func newAdminAuthSvc(db *gorm.DB) *services.AdminAuthService {
	return services.NewAdminAuthService(db)
}

// adminToken mints a valid admin JWT for use in HTTP test requests.
func adminToken(t *testing.T, authSvc *services.AdminAuthService) string {
	t.Helper()
	tok, err := authSvc.MintIntegrationTestToken(uuid.New())
	if err != nil {
		t.Fatalf("MintIntegrationTestToken: %v", err)
	}
	return tok
}

// ─── HTTP test router ─────────────────────────────────────────────────────────
//
// Go 1.22 net/http mux requires exactly ONE space between METHOD and PATH.
// Extra spaces cause the pattern to be treated as a plain path (no method filter)
// which results in 404 for non-matching methods and 405 for wrong methods.

func buildRouter(db *gorm.DB, authSvc *services.AdminAuthService) http.Handler {
	mux := http.NewServeMux()
	h := newAdminHandler(db)
	adminAuth := middleware.AdminAuthMiddleware(authSvc)

	// Prize CRUD — static paths must be registered BEFORE wildcard {id} paths
	mux.Handle("GET /api/v1/admin/prizes/summary", adminAuth(http.HandlerFunc(h.GetPrizeSummary)))
	mux.Handle("PUT /api/v1/admin/prizes/config", adminAuth(http.HandlerFunc(h.PublishPrizeConfiguration)))
	mux.Handle("POST /api/v1/admin/prizes/reorder", adminAuth(http.HandlerFunc(h.ReorderPrizes)))
	mux.Handle("GET /api/v1/admin/prizes", adminAuth(http.HandlerFunc(h.GetPrizePool)))
	mux.Handle("POST /api/v1/admin/prizes", adminAuth(http.HandlerFunc(h.CreatePrize)))
	mux.Handle("GET /api/v1/admin/prizes/{id}", adminAuth(http.HandlerFunc(h.GetPrize)))
	mux.Handle("PUT /api/v1/admin/prizes/{id}", adminAuth(http.HandlerFunc(h.UpdatePrize)))
	mux.Handle("DELETE /api/v1/admin/prizes/{id}", adminAuth(http.HandlerFunc(h.DeletePrize)))

	// Spin tiers
	mux.Handle("GET /api/v1/admin/spin/tiers", adminAuth(http.HandlerFunc(h.GetSpinTiers)))
	mux.Handle("POST /api/v1/admin/spin/tiers", adminAuth(http.HandlerFunc(h.CreateSpinTier)))
	mux.Handle("PUT /api/v1/admin/spin/tiers/{id}", adminAuth(http.HandlerFunc(h.UpdateSpinTier)))
	mux.Handle("DELETE /api/v1/admin/spin/tiers/{id}", adminAuth(http.HandlerFunc(h.DeleteSpinTier)))

	// Spin config
	mux.Handle("GET /api/v1/admin/spin/config", adminAuth(http.HandlerFunc(h.GetSpinConfig)))

	// Claims — static paths before wildcard
	mux.Handle("GET /api/v1/admin/spin/claims/pending", adminAuth(http.HandlerFunc(h.GetPendingClaims)))
	mux.Handle("GET /api/v1/admin/spin/claims/statistics", adminAuth(http.HandlerFunc(h.GetClaimStatistics)))
	mux.Handle("GET /api/v1/admin/spin/claims/export", adminAuth(http.HandlerFunc(h.ExportClaims)))
	mux.Handle("GET /api/v1/admin/spin/claims", adminAuth(http.HandlerFunc(h.ListClaims)))

	return mux
}

// do executes an HTTP request against the test server and returns the response.
func do(t *testing.T, srv *httptest.Server, method, path, tok string, body interface{}) *http.Response {
	t.Helper()
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req, err := http.NewRequest(method, srv.URL+path, reqBody)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, dest interface{}) {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		t.Fatalf("decode response JSON: %v", err)
	}
}

// ─── Repo-level tests (direct SQL via real Postgres) ─────────────────────────

// TestPrizeRepo_CreateAndFetch_Postgres tests that CreatePrize persists all
// fields to the real Postgres prize_pool table and ListActivePrizes returns them.
func TestPrizeRepo_CreateAndFetch_Postgres(t *testing.T) {
	db := openTestDB(t)
	withTx(t, db, func(tx *gorm.DB) {
		tx.Exec("DELETE FROM prize_pool") // migrations seed a full 100% wheel; this test authors its own
		svc := newSvc(tx)
		ctx := context.Background()

		prize, err := svc.CreatePrize(ctx, map[string]interface{}{
			"name":                   "Integration Test Prize",
			"prize_type":             "airtime",
			"base_value":             float64(5000),
			"win_probability_weight": float64(100),
			"is_active":              true,
			"color_scheme":           "#FF0000",
			"icon_name":              "phone",
			"terms_and_conditions":   "Valid 24h",
			"prize_code":             "INTTEST",
			"variation_code":         "NG_INT",
			"sort_order":             float64(99),
			"minimum_recharge":       float64(100000),
		})
		if err != nil {
			t.Fatalf("CreatePrize: %v", err)
		}
		if prize.ID == uuid.Nil {
			t.Fatal("expected non-nil UUID from Postgres gen_random_uuid()")
		}

		// Fetch back via ListActivePrizes to confirm the real SQL query works
		prizeRepo := persistence.NewPostgresPrizeRepository(tx)
		prizes, err := prizeRepo.ListActivePrizes(ctx)
		if err != nil {
			t.Fatalf("ListActivePrizes: %v", err)
		}
		found := false
		for _, p := range prizes {
			if p.ID == prize.ID {
				found = true
				if p.Name != "Integration Test Prize" {
					t.Errorf("name mismatch: got %q", p.Name)
				}
				if p.IconName != "phone" {
					t.Errorf("icon_name mismatch: got %q", p.IconName)
				}
				if p.PrizeCode != "INTTEST" {
					t.Errorf("prize_code mismatch: got %q", p.PrizeCode)
				}
				if p.VariationCode != "NG_INT" {
					t.Errorf("variation_code mismatch: got %q", p.VariationCode)
				}
				if p.TermsAndConditions != "Valid 24h" {
					t.Errorf("terms mismatch: got %q", p.TermsAndConditions)
				}
				if p.ColorScheme != "#FF0000" {
					t.Errorf("color_scheme mismatch: got %q", p.ColorScheme)
				}
				if p.MinimumRecharge != 100000 {
					t.Errorf("minimum_recharge mismatch: got %d", p.MinimumRecharge)
				}
			}
		}
		if !found {
			t.Fatal("created prize not found in ListActivePrizes result")
		}
	})
}

// TestPrizeRepo_SortOrder_Postgres verifies that ListActivePrizesSorted returns
// prizes in sort_order ASC order from the real Postgres index.
func TestPrizeRepo_SortOrder_Postgres(t *testing.T) {
	db := openTestDB(t)
	withTx(t, db, func(tx *gorm.DB) {
		// Wipe any existing prizes in this tx so we control the full set
		tx.Exec("DELETE FROM prize_pool")

		svc := newSvc(tx)
		ctx := context.Background()

		for _, item := range []struct {
			name  string
			order int
		}{
			{"Third", 3},
			{"First", 1},
			{"Second", 2},
		} {
			_, err := svc.CreatePrize(ctx, map[string]interface{}{
				"name":                   item.name,
				"prize_type":             "try_again",
				"win_probability_weight": float64(10),
				"sort_order":             float64(item.order),
			})
			if err != nil {
				t.Fatalf("CreatePrize %s: %v", item.name, err)
			}
		}

		prizeRepo := persistence.NewPostgresPrizeRepository(tx)
		prizes, err := prizeRepo.ListActivePrizesSorted(ctx)
		if err != nil {
			t.Fatalf("ListActivePrizesSorted: %v", err)
		}
		if len(prizes) != 3 {
			t.Fatalf("expected 3 prizes, got %d", len(prizes))
		}
		if prizes[0].Name != "First" || prizes[1].Name != "Second" || prizes[2].Name != "Third" {
			t.Errorf("wrong sort order: %v %v %v", prizes[0].Name, prizes[1].Name, prizes[2].Name)
		}
	})
}

// TestPrizeRepo_ProbabilityBudget_Postgres verifies that the probability budget
// guard (total active weight > 100.00%) is enforced by the service layer against
// the real Postgres prize_pool table.
func TestPrizeRepo_ProbabilityBudget_Postgres(t *testing.T) {
	db := openTestDB(t)
	withTx(t, db, func(tx *gorm.DB) {
		tx.Exec("DELETE FROM prize_pool")

		svc := newSvc(tx)
		ctx := context.Background()

		// Seed 98.00% weight
		_, err := svc.CreatePrize(ctx, map[string]interface{}{
			"name":                   "Big Prize",
			"prize_type":             "try_again",
			"win_probability_weight": float64(98),
		})
		if err != nil {
			t.Fatalf("seed CreatePrize: %v", err)
		}

		// Attempt to add 3% more (98 + 3 = 101 > 100) — must fail
		_, err = svc.CreatePrize(ctx, map[string]interface{}{
			"name":                   "Overflow Prize",
			"prize_type":             "airtime",
			"win_probability_weight": float64(3),
		})
		if err == nil {
			t.Fatal("expected probability budget error, got nil")
		}
		if !strings.Contains(err.Error(), "probability") && !strings.Contains(err.Error(), "budget") && !strings.Contains(err.Error(), "100") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

// TestPrizeRepo_UpdatePrize_Postgres verifies that UpdatePrize persists all
// new fields to the real Postgres table and the updated row is readable.
func TestPrizeRepo_UpdatePrize_Postgres(t *testing.T) {
	db := openTestDB(t)
	withTx(t, db, func(tx *gorm.DB) {
		tx.Exec("DELETE FROM prize_pool") // migrations seed a full 100% wheel; this test authors its own
		svc := newSvc(tx)
		ctx := context.Background()

		original, err := svc.CreatePrize(ctx, map[string]interface{}{
			"name":                   "Original Name",
			"prize_type":             "airtime",
			"win_probability_weight": float64(50),
		})
		if err != nil {
			t.Fatalf("CreatePrize: %v", err)
		}

		updated, err := svc.UpdatePrize(ctx, original.ID, map[string]interface{}{
			"name":                 "Updated Name",
			"icon_name":            "trophy",
			"terms_and_conditions": "Updated terms",
			"prize_code":           "UPD001",
			"variation_code":       "NG_UPD",
			"color_scheme":         "#00FF00",
		})
		if err != nil {
			t.Fatalf("UpdatePrize: %v", err)
		}
		if updated.Name != "Updated Name" {
			t.Errorf("name: got %q, want 'Updated Name'", updated.Name)
		}
		if updated.IconName != "trophy" {
			t.Errorf("icon_name: got %q, want 'trophy'", updated.IconName)
		}
		if updated.PrizeCode != "UPD001" {
			t.Errorf("prize_code: got %q, want 'UPD001'", updated.PrizeCode)
		}
		if updated.VariationCode != "NG_UPD" {
			t.Errorf("variation_code: got %q, want 'NG_UPD'", updated.VariationCode)
		}
		if updated.TermsAndConditions != "Updated terms" {
			t.Errorf("terms: got %q", updated.TermsAndConditions)
		}

		// Confirm the update is readable via GetPrize (real SELECT)
		fetched, err := svc.GetPrize(ctx, original.ID)
		if err != nil {
			t.Fatalf("GetPrize after update: %v", err)
		}
		if fetched.Name != "Updated Name" {
			t.Errorf("GetPrize name: got %q", fetched.Name)
		}
	})
}

// TestPrizeRepo_SoftDelete_Postgres verifies that DeletePrize sets is_active=false
// in the real Postgres table and the prize disappears from ListActivePrizes.
func TestPrizeRepo_SoftDelete_Postgres(t *testing.T) {
	db := openTestDB(t)
	withTx(t, db, func(tx *gorm.DB) {
		tx.Exec("DELETE FROM prize_pool") // migrations seed a full 100% wheel; this test authors its own
		svc := newSvc(tx)
		ctx := context.Background()

		prize, err := svc.CreatePrize(ctx, map[string]interface{}{
			"name":                   "To Delete",
			"prize_type":             "try_again",
			"win_probability_weight": float64(50),
		})
		if err != nil {
			t.Fatalf("CreatePrize: %v", err)
		}

		if err := svc.DeletePrize(ctx, prize.ID); err != nil {
			t.Fatalf("DeletePrize: %v", err)
		}

		// Must NOT appear in active list
		prizeRepo := persistence.NewPostgresPrizeRepository(tx)
		active, _ := prizeRepo.ListActivePrizes(ctx)
		for _, p := range active {
			if p.ID == prize.ID {
				t.Fatal("soft-deleted prize still appears in ListActivePrizes")
			}
		}

		// Must still exist in DB with is_active=false
		var isActive bool
		tx.Raw("SELECT is_active FROM prize_pool WHERE id = ?", prize.ID).Scan(&isActive)
		if isActive {
			t.Fatal("is_active should be false after DeletePrize")
		}
	})
}

// TestPrizeRepo_ReorderPrizes_Postgres verifies that ReorderPrizes updates
// sort_order in the real Postgres table atomically.
func TestPrizeRepo_ReorderPrizes_Postgres(t *testing.T) {
	db := openTestDB(t)
	withTx(t, db, func(tx *gorm.DB) {
		tx.Exec("DELETE FROM prize_pool")

		svc := newSvc(tx)
		ctx := context.Background()

		var ids []uuid.UUID
		for _, name := range []string{"Alpha", "Beta", "Gamma"} {
			p, err := svc.CreatePrize(ctx, map[string]interface{}{
				"name":                   name,
				"prize_type":             "try_again",
				"win_probability_weight": float64(10),
			})
			if err != nil {
				t.Fatalf("CreatePrize %s: %v", name, err)
			}
			ids = append(ids, p.ID)
		}

		// Reorder: Gamma(2), Alpha(0), Beta(1)
		reordered := []uuid.UUID{ids[2], ids[0], ids[1]}
		if err := svc.ReorderPrizes(ctx, reordered); err != nil {
			t.Fatalf("ReorderPrizes: %v", err)
		}

		// Verify via direct SQL on the real Postgres table
		type row struct {
			ID        uuid.UUID
			SortOrder int
		}
		var rows []row
		tx.Raw("SELECT id, sort_order FROM prize_pool ORDER BY sort_order ASC").Scan(&rows)
		if len(rows) != 3 {
			t.Fatalf("expected 3 rows, got %d", len(rows))
		}
		if rows[0].ID != ids[2] {
			t.Errorf("position 0: expected Gamma (%v), got %v", ids[2], rows[0].ID)
		}
		if rows[1].ID != ids[0] {
			t.Errorf("position 1: expected Alpha (%v), got %v", ids[0], rows[1].ID)
		}
		if rows[2].ID != ids[1] {
			t.Errorf("position 2: expected Beta (%v), got %v", ids[1], rows[2].ID)
		}
	})
}

// TestPrizeRepo_ProbabilitySummary_Postgres verifies that GetPrizeProbabilitySummary
// computes correct totals from the real Postgres prize_pool table.
func TestPrizeRepo_ProbabilitySummary_Postgres(t *testing.T) {
	db := openTestDB(t)
	withTx(t, db, func(tx *gorm.DB) {
		tx.Exec("DELETE FROM prize_pool")

		svc := newSvc(tx)
		ctx := context.Background()

		// Active: 40.00 + 30.00 = 70.00%
		for _, w := range []float64{40.0, 30.0} {
			_, err := svc.CreatePrize(ctx, map[string]interface{}{
				"name":                   fmt.Sprintf("Prize %.0f", w),
				"prize_type":             "try_again",
				"win_probability_weight": w,
				"is_active":              true,
			})
			if err != nil {
				t.Fatalf("CreatePrize: %v", err)
			}
		}
		// Inactive: 10.00% (should NOT count toward total)
		p, _ := svc.CreatePrize(ctx, map[string]interface{}{
			"name":                   "Inactive",
			"prize_type":             "try_again",
			"win_probability_weight": float64(10),
			"is_active":              true,
		})
		tx.Exec("UPDATE prize_pool SET is_active = false WHERE id = ?", p.ID)

		summary, err := svc.GetPrizeProbabilitySummary(ctx)
		if err != nil {
			t.Fatalf("GetPrizeProbabilitySummary: %v", err)
		}
		if summary.TotalWeight != 70.0 {
			t.Errorf("TotalWeight: got %v, want 70.0", summary.TotalWeight)
		}
		if summary.RemainingBudget != 30.0 {
			t.Errorf("RemainingBudget: got %v, want 30.0", summary.RemainingBudget)
		}
		if summary.PercentUsed != 70.0 {
			t.Errorf("PercentUsed: got %.2f, want 70.00", summary.PercentUsed)
		}
	})
}

// ─── HTTP integration tests (real Postgres + real HTTP handler) ───────────────

// TestHTTP_GetPrizePool_ReturnsJSON tests GET /admin/prizes returns a JSON object
// with a "prizes" array (the production handler wraps the array in an envelope).
func TestHTTP_GetPrizePool_ReturnsJSON(t *testing.T) {
	db := openTestDB(t)
	authSvc := newAdminAuthSvc(db)
	tok := adminToken(t, authSvc)
	srv := httptest.NewServer(buildRouter(db, authSvc))
	defer srv.Close()

	resp := do(t, srv, "GET", "/api/v1/admin/prizes", tok, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	decodeJSON(t, resp, &body)
	if body["prizes"] == nil {
		t.Fatal("expected 'prizes' key in GET /admin/prizes response")
	}
}

// TestHTTP_GetPrize_Postgres tests GET /admin/prizes/{id} returns the correct prize.
func TestHTTP_GetPrize_Postgres(t *testing.T) {
	db := openTestDB(t)
	withTx(t, db, func(tx *gorm.DB) {
		tx.Exec("DELETE FROM prize_pool")
		svc := newSvc(tx)
		ctx := context.Background()

		prize, err := svc.CreatePrize(ctx, map[string]interface{}{
			"name":                   "Fetchable Prize",
			"prize_type":             "airtime",
			"win_probability_weight": float64(50),
			"icon_name":              "star",
		})
		if err != nil {
			t.Fatalf("CreatePrize: %v", err)
		}

		authSvc := newAdminAuthSvc(tx)
		tok := adminToken(t, authSvc)
		srv := httptest.NewServer(buildRouter(tx, authSvc))
		defer srv.Close()

		resp := do(t, srv, "GET", "/api/v1/admin/prizes/"+prize.ID.String(), tok, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var body map[string]interface{}
		decodeJSON(t, resp, &body)
		if body["id"] != prize.ID.String() {
			t.Errorf("id mismatch: got %v", body["id"])
		}
		if body["name"] != "Fetchable Prize" {
			t.Errorf("name mismatch: got %v", body["name"])
		}
		if body["icon_name"] != "star" {
			t.Errorf("icon_name mismatch: got %v", body["icon_name"])
		}
	})
}

// TestHTTP_GetPrize_NotFound_Returns404 tests that a non-existent prize returns 404.
func TestHTTP_GetPrize_NotFound_Returns404(t *testing.T) {
	db := openTestDB(t)
	authSvc := newAdminAuthSvc(db)
	tok := adminToken(t, authSvc)
	srv := httptest.NewServer(buildRouter(db, authSvc))
	defer srv.Close()

	resp := do(t, srv, "GET", "/api/v1/admin/prizes/"+uuid.New().String(), tok, nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// TestHTTP_GetPrizeSummary_Postgres tests GET /admin/prizes/summary returns
// correct probability totals from the real Postgres table.
func TestHTTP_GetPrizeSummary_Postgres(t *testing.T) {
	db := openTestDB(t)
	withTx(t, db, func(tx *gorm.DB) {
		tx.Exec("DELETE FROM prize_pool")

		svc := newSvc(tx)
		ctx := context.Background()

		// 30% + 20% = 50% used
		for _, w := range []float64{30.0, 20.0} {
			_, err := svc.CreatePrize(ctx, map[string]interface{}{
				"name":                   fmt.Sprintf("P%.0f", w),
				"prize_type":             "try_again",
				"win_probability_weight": w,
			})
			if err != nil {
				t.Fatalf("CreatePrize: %v", err)
			}
		}

		authSvc := newAdminAuthSvc(tx)
		tok := adminToken(t, authSvc)
		srv := httptest.NewServer(buildRouter(tx, authSvc))
		defer srv.Close()

		resp := do(t, srv, "GET", "/api/v1/admin/prizes/summary", tok, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var body map[string]interface{}
		decodeJSON(t, resp, &body)

		totalWeight, _ := body["total_weight"].(float64)
		if totalWeight != 50.0 {
			t.Errorf("total_weight: got %.2f, want 50.00", totalWeight)
		}
		remaining, _ := body["remaining_budget"].(float64)
		if remaining != 50.0 {
			t.Errorf("remaining_budget: got %.2f, want 50.00", remaining)
		}
		pct, _ := body["percent_used"].(float64)
		if pct != 50.0 {
			t.Errorf("percent_used: got %.2f, want 50.00", pct)
		}
	})
}

// TestHTTP_GetSpinConfig_Postgres tests GET /admin/spin/config returns the full
// config payload including prizes, tiers, and probability summary.
func TestHTTP_GetSpinConfig_Postgres(t *testing.T) {
	db := openTestDB(t)
	authSvc := newAdminAuthSvc(db)
	tok := adminToken(t, authSvc)
	srv := httptest.NewServer(buildRouter(db, authSvc))
	defer srv.Close()

	resp := do(t, srv, "GET", "/api/v1/admin/spin/config", tok, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	decodeJSON(t, resp, &body)

	if body["prizes"] == nil {
		t.Error("expected 'prizes' key in spin config response")
	}
	if body["tiers"] == nil {
		t.Error("expected 'tiers' key in spin config response")
	}
	if body["probability_summary"] == nil {
		t.Error("expected 'probability_summary' key in spin config response")
	}
}

// TestHTTP_GetSpinTiers_Postgres tests GET /admin/spin/tiers returns the seeded tiers.
func TestHTTP_GetSpinTiers_Postgres(t *testing.T) {
	db := openTestDB(t)
	authSvc := newAdminAuthSvc(db)
	tok := adminToken(t, authSvc)
	srv := httptest.NewServer(buildRouter(db, authSvc))
	defer srv.Close()

	resp := do(t, srv, "GET", "/api/v1/admin/spin/tiers", tok, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var tiers []interface{}
	decodeJSON(t, resp, &tiers)
	if len(tiers) == 0 {
		t.Error("expected at least one spin tier from seeded data")
	}
}

// TestHTTP_AdminAuth_Rejected_Without_Token tests that all admin routes
// return 401 when no Authorization header is provided.
func TestHTTP_AdminAuth_Rejected_Without_Token(t *testing.T) {
	db := openTestDB(t)
	authSvc := newAdminAuthSvc(db)
	srv := httptest.NewServer(buildRouter(db, authSvc))
	defer srv.Close()

	paths := []string{
		"/api/v1/admin/prizes",
		"/api/v1/admin/prizes/summary",
		"/api/v1/admin/spin/tiers",
		"/api/v1/admin/spin/config",
	}
	for _, path := range paths {
		resp := do(t, srv, "GET", path, "", nil) // no token
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("path %s: expected 401, got %d", path, resp.StatusCode)
		}
	}
}

// TestHTTP_AdminAuth_Rejected_With_Invalid_Token tests that a tampered/invalid
// JWT returns 401 Unauthorized on admin routes.
func TestHTTP_AdminAuth_Rejected_With_Invalid_Token(t *testing.T) {
	db := openTestDB(t)
	authSvc := newAdminAuthSvc(db)
	srv := httptest.NewServer(buildRouter(db, authSvc))
	defer srv.Close()

	// A syntactically valid-looking JWT signed with a different secret
	resp := do(t, srv, "GET", "/api/v1/admin/prizes",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOiJ0ZXN0IiwiaXNfYWRtaW4iOnRydWV9.INVALIDSIG",
		nil)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid token, got %d", resp.StatusCode)
	}
}

// TestHTTP_ClaimsList_Postgres tests GET /admin/spin/claims returns a valid response.
func TestHTTP_ClaimsList_Postgres(t *testing.T) {
	db := openTestDB(t)
	authSvc := newAdminAuthSvc(db)
	tok := adminToken(t, authSvc)
	srv := httptest.NewServer(buildRouter(db, authSvc))
	defer srv.Close()

	resp := do(t, srv, "GET", "/api/v1/admin/spin/claims", tok, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body interface{}
	decodeJSON(t, resp, &body)
}

// TestHTTP_PendingClaims_Postgres tests GET /admin/spin/claims/pending.
func TestHTTP_PendingClaims_Postgres(t *testing.T) {
	db := openTestDB(t)
	authSvc := newAdminAuthSvc(db)
	tok := adminToken(t, authSvc)
	srv := httptest.NewServer(buildRouter(db, authSvc))
	defer srv.Close()

	resp := do(t, srv, "GET", "/api/v1/admin/spin/claims/pending", tok, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	decodeJSON(t, resp, &body)
	if body["data"] == nil {
		t.Error("expected 'data' key in pending claims response")
	}
	if body["total"] == nil {
		t.Error("expected 'total' key in pending claims response")
	}
}

// TestHTTP_ClaimStatistics_Postgres tests GET /admin/spin/claims/statistics.
func TestHTTP_ClaimStatistics_Postgres(t *testing.T) {
	db := openTestDB(t)
	authSvc := newAdminAuthSvc(db)
	tok := adminToken(t, authSvc)
	srv := httptest.NewServer(buildRouter(db, authSvc))
	defer srv.Close()

	resp := do(t, srv, "GET", "/api/v1/admin/spin/claims/statistics", tok, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body interface{}
	decodeJSON(t, resp, &body)
}

// TestHTTP_ExportClaims_Postgres tests GET /admin/spin/claims/export returns CSV.
func TestHTTP_ExportClaims_Postgres(t *testing.T) {
	db := openTestDB(t)
	authSvc := newAdminAuthSvc(db)
	tok := adminToken(t, authSvc)
	srv := httptest.NewServer(buildRouter(db, authSvc))
	defer srv.Close()

	resp := do(t, srv, "GET", "/api/v1/admin/spin/claims/export", tok, nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/csv") {
		t.Errorf("expected Content-Type text/csv, got %q", ct)
	}
	cd := resp.Header.Get("Content-Disposition")
	if !strings.Contains(cd, "claims_export.csv") {
		t.Errorf("expected Content-Disposition with filename, got %q", cd)
	}
}

// TestHTTP_GetPrizePool_IncludesInactive_Postgres tests that GET /admin/prizes
// returns inactive prizes (admin view) while ?active_only=true filters them out.
// The handler returns {"prizes": [...]} — tests unwrap the envelope.
func TestHTTP_GetPrizePool_IncludesInactive_Postgres(t *testing.T) {
	db := openTestDB(t)
	withTx(t, db, func(tx *gorm.DB) {
		tx.Exec("DELETE FROM prize_pool")

		svc := newSvc(tx)
		ctx := context.Background()

		// Active prize: 60%
		_, err := svc.CreatePrize(ctx, map[string]interface{}{
			"name":                   "Active",
			"prize_type":             "try_again",
			"win_probability_weight": float64(60),
			"is_active":              true,
		})
		if err != nil {
			t.Fatalf("CreatePrize (active): %v", err)
		}

		// Create a second prize at 30%, then soft-delete it
		inactive, err := svc.CreatePrize(ctx, map[string]interface{}{
			"name":                   "Inactive",
			"prize_type":             "try_again",
			"win_probability_weight": float64(30),
			"is_active":              true,
		})
		if err != nil {
			t.Fatalf("CreatePrize (to-be-inactive): %v", err)
		}
		tx.Exec("UPDATE prize_pool SET is_active = false WHERE id = ?", inactive.ID)

		authSvc := newAdminAuthSvc(tx)
		tok := adminToken(t, authSvc)
		srv := httptest.NewServer(buildRouter(tx, authSvc))
		defer srv.Close()

		// Default admin view: should include both active and inactive
		resp := do(t, srv, "GET", "/api/v1/admin/prizes", tok, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var allEnvelope map[string]interface{}
		decodeJSON(t, resp, &allEnvelope)
		all, _ := allEnvelope["prizes"].([]interface{})
		if len(all) != 2 {
			t.Errorf("default admin view: expected 2 prizes, got %d", len(all))
		}

		// ?active_only=true: only active prize
		resp2 := do(t, srv, "GET", "/api/v1/admin/prizes?active_only=true", tok, nil)
		if resp2.StatusCode != http.StatusOK {
			t.Fatalf("active_only: expected 200, got %d", resp2.StatusCode)
		}
		var activeEnvelope map[string]interface{}
		decodeJSON(t, resp2, &activeEnvelope)
		activeOnly, _ := activeEnvelope["prizes"].([]interface{})
		if len(activeOnly) != 1 {
			t.Errorf("active_only view: expected 1 prize, got %d", len(activeOnly))
		}
	})
}

// ─── Timestamp sanity check ───────────────────────────────────────────────────

// TestPrizeRepo_UpdatedAt_IsSet_Postgres verifies that updated_at is populated
// by the Postgres DEFAULT NOW() when a prize row is created.
// Note: prize_pool has no created_at column — only updated_at.
func TestPrizeRepo_UpdatedAt_IsSet_Postgres(t *testing.T) {
	db := openTestDB(t)
	withTx(t, db, func(tx *gorm.DB) {
		tx.Exec("DELETE FROM prize_pool")
		svc := newSvc(tx)
		ctx := context.Background()

		before := time.Now().Add(-2 * time.Second)
		prize, err := svc.CreatePrize(ctx, map[string]interface{}{
			"name":                   "Timestamp Test",
			"prize_type":             "try_again",
			"win_probability_weight": float64(10),
		})
		if err != nil {
			t.Fatalf("CreatePrize: %v", err)
		}
		after := time.Now().Add(2 * time.Second)

		// updated_at is a Postgres DEFAULT NOW() column — read it directly
		var updatedAt time.Time
		result := tx.Raw("SELECT updated_at FROM prize_pool WHERE id = ?", prize.ID).Scan(&updatedAt)
		if result.Error != nil {
			t.Fatalf("SELECT updated_at: %v", result.Error)
		}
		if updatedAt.IsZero() {
			t.Fatal("updated_at is zero — Postgres DEFAULT NOW() not applied or row not found")
		}
		if updatedAt.Before(before) || updatedAt.After(after) {
			t.Errorf("updated_at %v is outside expected range [%v, %v]", updatedAt, before, after)
		}
	})
}

func TestPublishPrizeConfiguration_DynamicWheelTenToFour_Postgres(t *testing.T) {
	db := openTestDB(t)
	withTx(t, db, func(tx *gorm.DB) {
		if err := tx.Table("prize_pool").Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
			t.Fatalf("deactivate existing prize rows: %v", err)
		}
		svc := newSvc(tx)

		prizes := make([]entities.PrizePoolEntry, 10)
		for i := range prizes {
			prizes[i] = entities.PrizePoolEntry{
				ID:          uuid.New(),
				Name:        fmt.Sprintf("Dynamic Prize %02d", i+1),
				PrizeType:   entities.PrizePulsePoints,
				BaseValue:   float64(i + 1),
				IsActive:    true,
				ProbWeight:  10.00,
				ColorScheme: fmt.Sprintf("#%06X", 0x110000+i*0x001111),
				SortOrder:   i,
			}
		}

		if _, err := svc.PublishPrizeConfiguration(context.Background(), prizes); err != nil {
			t.Fatalf("publish 10-prize wheel: %v", err)
		}
		wheel10, err := svc.GetWheelConfig(context.Background())
		if err != nil {
			t.Fatalf("get 10-prize wheel: %v", err)
		}
		if len(wheel10.Slots) != 10 {
			t.Fatalf("expected 10 live wheel slots, got %d", len(wheel10.Slots))
		}
		total10 := 0.0
		for i, slot := range wheel10.Slots {
			if slot.Index != i {
				t.Fatalf("slot %d returned index %d", i, slot.Index)
			}
			if slot.Probability != 10.00 {
				t.Fatalf("slot %d probability = %.2f, want 10.00", i, slot.Probability)
			}
			total10 += slot.Probability
		}
		if math.Abs(total10-100.00) > 0.001 {
			t.Fatalf("10-prize wheel total = %.2f, want 100.00", total10)
		}

		weights := []float64{40, 30, 20, 10}
		for i := range prizes {
			if i < 4 {
				prizes[i].IsActive = true
				prizes[i].ProbWeight = weights[i]
			} else {
				prizes[i].IsActive = false
			}
		}

		if _, err := svc.PublishPrizeConfiguration(context.Background(), prizes); err != nil {
			t.Fatalf("publish reduced 4-prize wheel: %v", err)
		}
		wheel4, err := svc.GetWheelConfig(context.Background())
		if err != nil {
			t.Fatalf("get 4-prize wheel: %v", err)
		}
		if len(wheel4.Slots) != 4 {
			t.Fatalf("expected 4 live wheel slots, got %d", len(wheel4.Slots))
		}
		total4 := 0.0
		for i, want := range weights {
			got := wheel4.Slots[i]
			if got.PrizeID != prizes[i].ID {
				t.Fatalf("slot %d prize id = %s, want %s", i, got.PrizeID, prizes[i].ID)
			}
			if math.Abs(got.Probability-want) > 0.001 {
				t.Fatalf("slot %d probability = %.2f, want %.2f", i, got.Probability, want)
			}
			total4 += got.Probability
		}
		if math.Abs(total4-100.00) > 0.001 {
			t.Fatalf("4-prize wheel total = %.2f, want 100.00", total4)
		}

		var activeCount int64
		if err := tx.Table("prize_pool").Where("is_active = ?", true).Count(&activeCount).Error; err != nil {
			t.Fatalf("count active prizes: %v", err)
		}
		if activeCount != 4 {
			t.Fatalf("database active prize count = %d, want 4", activeCount)
		}
	})
}

func TestPublishPrizeConfiguration_InvalidDraftRollsBack_Postgres(t *testing.T) {
	db := openTestDB(t)
	withTx(t, db, func(tx *gorm.DB) {
		if err := tx.Table("prize_pool").Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
			t.Fatalf("deactivate existing prize rows: %v", err)
		}
		svc := newSvc(tx)

		prizes := []entities.PrizePoolEntry{
			{ID: uuid.New(), Name: "Prize A", PrizeType: entities.PrizePulsePoints, BaseValue: 10, IsActive: true, ProbWeight: 40, SortOrder: 0},
			{ID: uuid.New(), Name: "Prize B", PrizeType: entities.PrizePulsePoints, BaseValue: 20, IsActive: true, ProbWeight: 30, SortOrder: 1},
			{ID: uuid.New(), Name: "Prize C", PrizeType: entities.PrizePulsePoints, BaseValue: 30, IsActive: true, ProbWeight: 20, SortOrder: 2},
			{ID: uuid.New(), Name: "Prize D", PrizeType: entities.PrizePulsePoints, BaseValue: 40, IsActive: true, ProbWeight: 10, SortOrder: 3},
		}
		if _, err := svc.PublishPrizeConfiguration(context.Background(), prizes); err != nil {
			t.Fatalf("publish valid baseline: %v", err)
		}

		invalid := append([]entities.PrizePoolEntry(nil), prizes...)
		invalid[0].ProbWeight = 30 // final total would be 90%

		if _, err := svc.PublishPrizeConfiguration(context.Background(), invalid); err == nil {
			t.Fatal("expected invalid 90% draft to be rejected")
		}

		wheel, err := svc.GetWheelConfig(context.Background())
		if err != nil {
			t.Fatalf("get live wheel after rejected draft: %v", err)
		}
		if len(wheel.Slots) != 4 {
			t.Fatalf("live wheel changed after rejected draft: got %d slots", len(wheel.Slots))
		}
		want := []float64{40, 30, 20, 10}
		for i := range want {
			if math.Abs(wheel.Slots[i].Probability-want[i]) > 0.001 {
				t.Fatalf("slot %d changed after rejected draft: got %.2f want %.2f", i, wheel.Slots[i].Probability, want[i])
			}
		}
	})
}

// TestHTTP_LegacyPrizeMutations_AreRefused pins the prize-wheel contract: the
// wheel is published atomically through PUT /api/v1/admin/prizes/config (see
// TestPublishPrizeConfiguration_*), so the per-prize create/update/delete/
// reorder endpoints must refuse with 409 and leave the table untouched — a
// partial mutation could put a live wheel at anything but exactly 100%.
func TestHTTP_LegacyPrizeMutations_AreRefused(t *testing.T) {
	db := openTestDB(t)
	withTx(t, db, func(tx *gorm.DB) {
		tx.Exec("DELETE FROM prize_pool")
		svc := newSvc(tx)
		existing, err := svc.CreatePrize(context.Background(), map[string]interface{}{
			"name": "Existing", "prize_type": "try_again", "win_probability_weight": float64(100),
		})
		if err != nil {
			t.Fatalf("seed prize: %v", err)
		}
		authSvc := newAdminAuthSvc(tx)
		tok := adminToken(t, authSvc)
		srv := httptest.NewServer(buildRouter(tx, authSvc))
		defer srv.Close()

		calls := []struct {
			name, method, path string
			body               interface{}
		}{
			{"create", "POST", "/api/v1/admin/prizes", map[string]interface{}{"name": "New", "prize_type": "airtime", "win_probability_weight": float64(10)}},
			{"update", "PUT", "/api/v1/admin/prizes/" + existing.ID.String(), map[string]interface{}{"name": "Renamed"}},
			{"delete", "DELETE", "/api/v1/admin/prizes/" + existing.ID.String(), nil},
			{"reorder", "POST", "/api/v1/admin/prizes/reorder", map[string]interface{}{"ordered_ids": []string{existing.ID.String()}}},
		}
		for _, c := range calls {
			resp := do(t, srv, c.method, c.path, tok, c.body)
			var body map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&body) //nolint:errcheck
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusConflict {
				t.Fatalf("%s: got %d, want 409 (%v)", c.name, resp.StatusCode, body)
			}
			if msg, _ := body["error"].(string); !strings.Contains(msg, "/api/v1/admin/prizes/config") {
				t.Fatalf("%s: refusal should point at the atomic endpoint, got %q", c.name, msg)
			}
		}

		var count int64
		var name string
		var active bool
		tx.Raw("SELECT count(*) FROM prize_pool").Scan(&count)
		tx.Raw("SELECT name, is_active FROM prize_pool WHERE id = ?", existing.ID).Row().Scan(&name, &active) //nolint:errcheck
		if count != 1 || name != "Existing" || !active {
			t.Fatalf("refused mutations must leave the table untouched: count=%d name=%q active=%v", count, name, active)
		}
	})
}
