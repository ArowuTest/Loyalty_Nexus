package handlers_test

// ussd_handler_test.go — End-to-end tests for the USSD menu state machine.
//
// Each test uses a unique phone number to prevent cross-test DB interference
// caused by the background rollbackExpiredSessions goroutine.
//
// Coverage:
//   - Root menu (all 8 options visible, shortcode shown)
//   - Option 0: Exit
//   - Option 1: My Balance — Pulse Points, Spin Credits, Tier, Streak
//   - Option 2: Spin & Win — sub-menu, no-credits guard, last-3-results, invalid option
//   - Option 3: Monthly Draw — graceful degradation when drawSvc is nil
//   - Option 4: Redeem Points — sub-menu, confirm, success, insufficient points, data bundle
//   - Option 5: My Streak — active streak, zero streak, Month Master (30 days)
//   - Option 6: My Passport — sub-menu, view passport, no-badges, leaderboard
//   - Option 7: AI Knowledge Tools — SMS instruction, point cost, topic prompt, invalid, back
//   - Session persistence — session row is created on CON responses
//   - Unknown user — graceful END response
//   - Invalid top-level option — END with shortcode hint

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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
	"loyalty-nexus/internal/presentation/http/handlers"
)

// ─── Test DB setup ────────────────────────────────────────────────────────────

var dbCounter int64

// newTestDB creates a fresh in-memory SQLite DB with all required tables and seed data.
// Each call uses a unique named in-memory DB to prevent SQLite connection-pool sharing.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Use a unique name per DB to ensure complete isolation between tests.
	dbID := atomic.AddInt64(&dbCounter, 1)
	dsn := fmt.Sprintf("file:testdb_%d?mode=memory&cache=shared", dbID)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	// AutoMigrate entities that have gorm: tags or well-formed struct fields.
	if err := db.AutoMigrate(
		&entities.User{},
		&entities.Wallet{},
		&entities.USSDSession{},
		&entities.StudioTool{},
		&entities.SpinResult{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}

	// lifetime_points lives in users via raw migration — not a User struct field.
	db.Exec(`ALTER TABLE users ADD COLUMN lifetime_points INTEGER NOT NULL DEFAULT 0`)

	// user_badges is not an entity struct.
	db.Exec(`CREATE TABLE IF NOT EXISTS user_badges (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		badge_key TEXT NOT NULL,
		earned_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, badge_key)
	)`)

	// network_configs is not an entity struct.
	db.Exec(`CREATE TABLE IF NOT EXISTS network_configs (key TEXT PRIMARY KEY, value TEXT NOT NULL)`)
	db.Exec(`INSERT OR IGNORE INTO network_configs (key, value) VALUES ('ussd_shortcode', '*384#')`)
	db.Exec(`INSERT OR IGNORE INTO network_configs (key, value) VALUES ('ussd_sms_number', '08012345678')`)
	db.Exec(`INSERT OR IGNORE INTO network_configs (key, value) VALUES ('spin_trigger_naira', '1000')`)
	db.Exec(`INSERT OR IGNORE INTO network_configs (key, value) VALUES ('ussd_session_timeout_seconds', '120')`)
	db.Exec(`INSERT OR IGNORE INTO network_configs (key, value) VALUES ('spin_max_per_user_per_day', '3')`)
	db.Exec(`INSERT OR IGNORE INTO network_configs (key, value) VALUES ('daily_prize_liability_cap_naira', '500000')`)

	// Seed a Knowledge Tool (Study Guide, 5 pts).
	db.Exec(`INSERT OR IGNORE INTO studio_tools (id, slug, name, point_cost, is_active) VALUES (?, 'study-guide', 'Study Guide', 5, 1)`, uuid.New())

	return db
}

// newHandler builds a fully-wired USSDHandler backed by the given DB.
func newHandler(t *testing.T, db *gorm.DB) *handlers.USSDHandler {
	t.Helper()
	cfg := config.NewConfigManager(db)
	userRepo := persistence.NewPostgresUserRepository(db)
	sessionRepo := persistence.NewPostgresUSSDSessionRepository(db)
	studioRepo := persistence.NewPostgresStudioRepository(db)
	prizeRepo := persistence.NewPostgresPrizeRepository(db)
	txRepo := persistence.NewPostgresTransactionRepository(db)
	studioSvc := services.NewStudioService(studioRepo, userRepo, nil, nil, nil, db)
	knowledgeSvc := services.NewUSSDKnowledgeService(studioSvc, nil, nil, cfg)
	passportSvc := services.NewPassportService(db)
	spinSvc := services.NewSpinService(userRepo, txRepo, prizeRepo, nil, nil, cfg, db)

	h := handlers.NewUSSDHandler(spinSvc, nil, userRepo, sessionRepo, cfg)
	h.SetKnowledgeService(knowledgeSvc)
	h.SetPassportService(passportSvc)
	return h
}

// seedUser inserts a user + wallet. Uses a unique phone per call to prevent cross-test
// interference from the background rollbackExpiredSessions goroutine.
func seedUser(db *gorm.DB, phone, tier string, streak int, pulsePoints, spinCredits int64) uuid.UUID {
	userID := uuid.New()
	db.Exec(
		`INSERT INTO users (id, phone_number, tier, streak_count, lifetime_points, is_active) VALUES (?, ?, ?, ?, ?, 1)`,
		userID, phone, tier, streak, pulsePoints*10,
	)
	db.Exec(
		`INSERT INTO wallets (id, user_id, pulse_points, spin_credits, recharge_counter) VALUES (?, ?, ?, ?, 0)`,
		uuid.New(), userID, pulsePoints, spinCredits,
	)
	return userID
}

// post sends a USSD POST request and returns the response body.
// A brief sleep is added after each call to allow the background
// rollbackExpiredSessions goroutine to complete before the test ends.
func post(t *testing.T, h *handlers.USSDHandler, sessionID, phone, text string) string {
	t.Helper()
	body := "sessionId=" + sessionID + "&phoneNumber=" + phone + "&text=" + text
	req := httptest.NewRequest(http.MethodPost, "/ussd", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.Handle(w, req)
	// Give the background rollbackExpiredSessions goroutine time to finish.
	time.Sleep(5 * time.Millisecond)
	return w.Body.String()
}

// ─── Root menu ────────────────────────────────────────────────────────────────

func TestUSSDHandler_RootMenu(t *testing.T) {
	db := newTestDB(t)
	h := newHandler(t, db)

	res := post(t, h, "s1", "2340000000001", "")
	if !strings.HasPrefix(res, "CON Welcome to Loyalty Nexus!") {
		t.Errorf("expected root menu, got: %s", res)
	}
	if !strings.Contains(res, "*384#") {
		t.Errorf("expected shortcode *384# in root menu, got: %s", res)
	}
	for _, opt := range []string{
		"1. My Balance", "2. Spin & Win", "3. Monthly Draw",
		"4. Redeem Points", "5. My Streak", "6. My Passport", "7. AI Knowledge Tools", "0. Exit",
	} {
		if !strings.Contains(res, opt) {
			t.Errorf("missing menu option %q in: %s", opt, res)
		}
	}
}

func TestUSSDHandler_Exit(t *testing.T) {
	db := newTestDB(t)
	h := newHandler(t, db)

	res := post(t, h, "s2", "2340000000002", "0")
	if !strings.HasPrefix(res, "END Thank you") {
		t.Errorf("expected exit message, got: %s", res)
	}
}

func TestUSSDHandler_UnknownUser(t *testing.T) {
	db := newTestDB(t)
	h := newHandler(t, db)

	res := post(t, h, "s3", "0000000000003", "1")
	if !strings.HasPrefix(res, "END Account not found") {
		t.Errorf("expected account-not-found, got: %s", res)
	}
}

// ─── Option 1: My Balance ─────────────────────────────────────────────────────

func TestUSSDHandler_MyBalance(t *testing.T) {
	db := newTestDB(t)
	seedUser(db, "2340000000010", "GOLD", 12, 150, 2)
	h := newHandler(t, db)

	res := post(t, h, "s10", "2340000000010", "1")
	if !strings.HasPrefix(res, "END 📊 Loyalty Nexus Balance") {
		t.Errorf("expected balance screen, got: %s", res)
	}
	for _, want := range []string{"150 pts", "Spin Credits:  2", "GOLD", "Day 12"} {
		if !strings.Contains(res, want) {
			t.Errorf("missing %q in balance screen: %s", want, res)
		}
	}
}

// ─── Option 2: Spin & Win ─────────────────────────────────────────────────────

func TestUSSDHandler_SpinSubMenu(t *testing.T) {
	db := newTestDB(t)
	h := newHandler(t, db)

	res := post(t, h, "s20", "2340000000020", "2")
	if !strings.HasPrefix(res, "CON 🎡 Spin & Win") {
		t.Errorf("expected spin sub-menu, got: %s", res)
	}
	if !strings.Contains(res, "1. Play a Spin") || !strings.Contains(res, "2. Last 3 Results") {
		t.Errorf("missing spin sub-menu options: %s", res)
	}
}

func TestUSSDHandler_SpinNoCredits(t *testing.T) {
	db := newTestDB(t)
	seedUser(db, "2340000000021", "BRONZE", 1, 0, 0) // 0 spin credits
	h := newHandler(t, db)

	res := post(t, h, "s21", "2340000000021", "2*1")
	if !strings.HasPrefix(res, "END ❌") {
		t.Errorf("expected spin error, got: %s", res)
	}
	if !strings.Contains(res, "No spin credits") {
		t.Errorf("expected 'No spin credits' message, got: %s", res)
	}
}

func TestUSSDHandler_SpinHistory_NoHistory(t *testing.T) {
	db := newTestDB(t)
	seedUser(db, "2340000000022", "BRONZE", 1, 0, 0)
	h := newHandler(t, db)

	res := post(t, h, "s22", "2340000000022", "2*2")
	if !strings.HasPrefix(res, "END No spin history") {
		t.Errorf("expected no-history message, got: %s", res)
	}
}

func TestUSSDHandler_SpinInvalidOption(t *testing.T) {
	db := newTestDB(t)
	h := newHandler(t, db)

	res := post(t, h, "s23", "2340000000023", "2*9")
	if !strings.HasPrefix(res, "END Invalid option") {
		t.Errorf("expected invalid option, got: %s", res)
	}
}

// ─── Option 3: Monthly Draw ───────────────────────────────────────────────────

func TestUSSDHandler_DrawSubMenu(t *testing.T) {
	db := newTestDB(t)
	h := newHandler(t, db) // drawSvc is nil by default

	res := post(t, h, "s30", "2340000000030", "3")
	if !strings.HasPrefix(res, "CON 🏆 Monthly Draw") {
		t.Errorf("expected draw sub-menu, got: %s", res)
	}
}

func TestUSSDHandler_DrawMyEntries_NilDrawSvc(t *testing.T) {
	db := newTestDB(t)
	seedUser(db, "2340000000031", "BRONZE", 0, 0, 0)
	h := newHandler(t, db)

	res := post(t, h, "s31", "2340000000031", "3*1")
	if !strings.HasPrefix(res, "END Draw service unavailable") {
		t.Errorf("expected draw service unavailable, got: %s", res)
	}
}

// ─── Option 4: Redeem Points ──────────────────────────────────────────────────

func TestUSSDHandler_RedeemMenu(t *testing.T) {
	db := newTestDB(t)
	seedUser(db, "2340000000040", "BRONZE", 0, 50, 0)
	h := newHandler(t, db)

	res := post(t, h, "s40", "2340000000040", "4")
	if !strings.HasPrefix(res, "CON 💳 Redeem Points") {
		t.Errorf("expected redeem menu, got: %s", res)
	}
	if !strings.Contains(res, "50 pts") {
		t.Errorf("expected '50 pts' in redeem menu: %s", res)
	}
}

func TestUSSDHandler_RedeemAirtimeConfirm(t *testing.T) {
	db := newTestDB(t)
	seedUser(db, "2340000000041", "BRONZE", 0, 50, 0)
	h := newHandler(t, db)

	res := post(t, h, "s41", "2340000000041", "4*1")
	if !strings.HasPrefix(res, "CON Redeem 10 pts") {
		t.Errorf("expected airtime confirm prompt, got: %s", res)
	}
}

func TestUSSDHandler_RedeemAirtimeSuccess(t *testing.T) {
	db := newTestDB(t)
	seedUser(db, "2340000000042", "BRONZE", 0, 50, 0)
	h := newHandler(t, db)

	res := post(t, h, "s42", "2340000000042", "4*1*1")
	if !strings.HasPrefix(res, "END ✅ Redemption queued") {
		t.Errorf("expected redemption success, got: %s", res)
	}
}

func TestUSSDHandler_RedeemAirtimeInsufficientPoints(t *testing.T) {
	db := newTestDB(t)
	seedUser(db, "2340000000043", "BRONZE", 0, 5, 0) // only 5 pts, need 10
	h := newHandler(t, db)

	res := post(t, h, "s43", "2340000000043", "4*1*1")
	if !strings.HasPrefix(res, "END ❌ Insufficient points") {
		t.Errorf("expected insufficient points, got: %s", res)
	}
}

func TestUSSDHandler_RedeemDataBundle(t *testing.T) {
	db := newTestDB(t)
	seedUser(db, "2340000000044", "BRONZE", 0, 50, 0)
	h := newHandler(t, db)

	res := post(t, h, "s44", "2340000000044", "4*2*1")
	if !strings.HasPrefix(res, "END ✅ Redemption queued") {
		t.Errorf("expected data redemption success, got: %s", res)
	}
}

func TestUSSDHandler_RedeemMoreOnApp(t *testing.T) {
	db := newTestDB(t)
	h := newHandler(t, db)

	res := post(t, h, "s45", "2340000000045", "4*3")
	if !strings.Contains(res, "Download the Loyalty Nexus app") {
		t.Errorf("expected app download message, got: %s", res)
	}
}

// ─── Option 5: My Streak ─────────────────────────────────────────────────────

func TestUSSDHandler_MyStreak_Active(t *testing.T) {
	db := newTestDB(t)
	seedUser(db, "2340000000050", "BRONZE", 8, 100, 0)
	h := newHandler(t, db)

	res := post(t, h, "s50", "2340000000050", "5")
	if !strings.HasPrefix(res, "END 🔥 Streak: 8 days") {
		t.Errorf("expected streak message, got: %s", res)
	}
	if !strings.Contains(res, "Week Warrior") {
		t.Errorf("expected Week Warrior badge mention: %s", res)
	}
}

func TestUSSDHandler_MyStreak_Zero(t *testing.T) {
	db := newTestDB(t)
	seedUser(db, "2340000000051", "BRONZE", 0, 0, 0)
	h := newHandler(t, db)

	res := post(t, h, "s51", "2340000000051", "5")
	if !strings.Contains(res, "No active streak") {
		t.Errorf("expected no-streak message, got: %s", res)
	}
}

func TestUSSDHandler_MyStreak_MonthMaster(t *testing.T) {
	db := newTestDB(t)
	seedUser(db, "2340000000052", "GOLD", 30, 500, 0)
	h := newHandler(t, db)

	res := post(t, h, "s52", "2340000000052", "5")
	if !strings.Contains(res, "Month Master") {
		t.Errorf("expected Month Master mention: %s", res)
	}
}

// ─── Option 6: My Digital Passport ───────────────────────────────────────────

func TestUSSDHandler_PassportSubMenu(t *testing.T) {
	db := newTestDB(t)
	seedUser(db, "2340000000060", "SILVER", 5, 100, 1)
	h := newHandler(t, db)

	res := post(t, h, "s60", "2340000000060", "6")
	if !strings.HasPrefix(res, "CON 🪪 My Digital Passport") {
		t.Errorf("expected passport sub-menu, got: %s", res)
	}
	if !strings.Contains(res, "SILVER") {
		t.Errorf("expected tier in passport sub-menu: %s", res)
	}
}

func TestUSSDHandler_PassportView(t *testing.T) {
	db := newTestDB(t)
	seedUser(db, "2340000000061", "GOLD", 12, 500, 1)
	h := newHandler(t, db)

	res := post(t, h, "s61", "2340000000061", "6*1")
	if !strings.HasPrefix(res, "END 🪪 Your Digital Passport") {
		t.Errorf("expected passport view, got: %s", res)
	}
	if !strings.Contains(res, "GOLD") || !strings.Contains(res, "12 days") {
		t.Errorf("expected tier and streak in passport view: %s", res)
	}
}

func TestUSSDHandler_PassportBadges_None(t *testing.T) {
	db := newTestDB(t)
	seedUser(db, "2340000000062", "BRONZE", 0, 0, 0)
	h := newHandler(t, db)

	res := post(t, h, "s62", "2340000000062", "6*2")
	if !strings.Contains(res, "No badges yet") {
		t.Errorf("expected no-badges message, got: %s", res)
	}
}

func TestUSSDHandler_PassportLeaderboard(t *testing.T) {
	db := newTestDB(t)
	seedUser(db, "2340000000063", "BRONZE", 0, 0, 0)
	h := newHandler(t, db)

	res := post(t, h, "s63", "2340000000063", "6*3")
	if !strings.Contains(res, "Leaderboard") {
		t.Errorf("expected leaderboard message, got: %s", res)
	}
}

// ─── Option 7: AI Knowledge Tools ────────────────────────────────────────────

func TestUSSDHandler_KnowledgeToolsMenu(t *testing.T) {
	db := newTestDB(t)
	h := newHandler(t, db)

	res := post(t, h, "s70", "2340000000070", "7")
	if !strings.HasPrefix(res, "CON 🤖 AI Knowledge Tools") {
		t.Errorf("expected knowledge tools menu, got: %s", res)
	}
	if !strings.Contains(res, "Send topic via SMS to 08012345678") {
		t.Errorf("missing SMS instruction line: %s", res)
	}
	if !strings.Contains(res, "Study Guide (5 pts)") {
		t.Errorf("missing tool with point cost: %s", res)
	}
}

func TestUSSDHandler_KnowledgeToolsTopicPrompt(t *testing.T) {
	db := newTestDB(t)
	h := newHandler(t, db)

	res := post(t, h, "s71", "2340000000071", "7*1")
	if !strings.HasPrefix(res, "CON 📝 Study Guide") {
		t.Errorf("expected topic prompt, got: %s", res)
	}
	if !strings.Contains(res, "Type your topic") {
		t.Errorf("expected topic instruction: %s", res)
	}
}

func TestUSSDHandler_KnowledgeToolsInvalidOption(t *testing.T) {
	db := newTestDB(t)
	h := newHandler(t, db)

	res := post(t, h, "s72", "2340000000072", "7*9")
	if !strings.HasPrefix(res, "END Invalid option") {
		t.Errorf("expected invalid option, got: %s", res)
	}
}

func TestUSSDHandler_KnowledgeToolsBack(t *testing.T) {
	db := newTestDB(t)
	h := newHandler(t, db)

	res := post(t, h, "s73", "2340000000073", "7*0")
	if !strings.HasPrefix(res, "CON Welcome to Loyalty Nexus!") {
		t.Errorf("expected root menu on back, got: %s", res)
	}
}

// ─── Session persistence ──────────────────────────────────────────────────────

func TestUSSDHandler_SessionPersisted(t *testing.T) {
	db := newTestDB(t)
	h := newHandler(t, db)

	// CON response (spin sub-menu) should upsert a session row.
	post(t, h, "sess-persist-80", "2340000000080", "2")

	var count int64
	db.Table("ussd_sessions").Where("session_id = ?", "sess-persist-80").Count(&count)
	if count != 1 {
		t.Errorf("expected session to be persisted, got count=%d", count)
	}
}

// ─── Invalid top-level option ─────────────────────────────────────────────────

func TestUSSDHandler_InvalidTopLevelOption(t *testing.T) {
	db := newTestDB(t)
	h := newHandler(t, db)

	res := post(t, h, "s90", "2340000000090", "99")
	if !strings.HasPrefix(res, "END Invalid option") {
		t.Errorf("expected invalid option, got: %s", res)
	}
	if !strings.Contains(res, "*384#") {
		t.Errorf("expected shortcode hint in invalid option: %s", res)
	}
}
