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

func setupDrawDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS draws (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'ACTIVE',
		draw_type TEXT NOT NULL DEFAULT 'MONTHLY',
		recurrence TEXT NOT NULL DEFAULT 'none',
		winner_count INTEGER NOT NULL DEFAULT 3,
		runner_ups_count INTEGER NOT NULL DEFAULT 0,
		prize_type TEXT NOT NULL DEFAULT 'MOMO_CASH',
		prize_value_kobo INTEGER NOT NULL DEFAULT 500000,
		prize_pool REAL NOT NULL DEFAULT 0,
		total_entries INTEGER NOT NULL DEFAULT 0,
		total_winners INTEGER NOT NULL DEFAULT 0,
		draw_time DATETIME,
		start_time DATETIME,
		end_time DATETIME,
		next_draw_at DATETIME,
		executed_at DATETIME,
		completed_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS draw_entries (
		id TEXT PRIMARY KEY,
		draw_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		msisdn TEXT NOT NULL DEFAULT '',
		entry_source TEXT NOT NULL DEFAULT 'recharge',
		amount INTEGER NOT NULL DEFAULT 0,
		entries_count INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS draw_winners (
		id TEXT PRIMARY KEY,
		draw_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		msisdn TEXT NOT NULL DEFAULT '',
		position INTEGER NOT NULL,
		prize_name TEXT NOT NULL DEFAULT '',
		prize_value INTEGER NOT NULL DEFAULT 0,
		claim_status TEXT NOT NULL DEFAULT 'PENDING_FULFILLMENT',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS network_configs (
		id TEXT PRIMARY KEY, key TEXT UNIQUE NOT NULL, value TEXT NOT NULL
	)`)
	return db
}

func TestDraw_ListUpcomingDraws_Empty(t *testing.T) {
	db := setupDrawDB(t)
	svc := services.NewDrawService(db)

	draws, err := svc.ListUpcomingDraws(context.Background())
	if err != nil {
		t.Fatalf("ListUpcomingDraws: %v", err)
	}
	if draws == nil {
		t.Fatal("should return empty slice, not nil")
	}
}

func TestDraw_ListUpcomingDraws_ReturnsActive(t *testing.T) {
	db := setupDrawDB(t)
	svc := services.NewDrawService(db)

	db.Exec(`INSERT INTO draws (id, name, status) VALUES (?, ?, 'ACTIVE')`,
		uuid.New().String(), "Test Draw")
	db.Exec(`INSERT INTO draws (id, name, status) VALUES (?, ?, 'COMPLETED')`,
		uuid.New().String(), "Old Draw")

	draws, err := svc.ListUpcomingDraws(context.Background())
	if err != nil {
		t.Fatalf("ListUpcomingDraws: %v", err)
	}
	for _, d := range draws {
		if d.Status == "COMPLETED" {
			t.Fatal("COMPLETED draw should not appear in upcoming list")
		}
	}
}

func TestDraw_ExecuteDraw_NotFound(t *testing.T) {
	db := setupDrawDB(t)
	svc := services.NewDrawService(db)

	err := svc.ExecuteDraw(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent draw")
	}
}

func TestDraw_ExecuteDraw_AlreadyExecuted(t *testing.T) {
	db := setupDrawDB(t)
	svc := services.NewDrawService(db)
	drawID := uuid.New()

	db.Exec(`INSERT INTO draws (id, name, status, executed_at) VALUES (?,?,?,?)`,
		drawID.String(), "Done Draw", "COMPLETED", time.Now().Add(-time.Hour))

	err := svc.ExecuteDraw(context.Background(), drawID)
	if err == nil {
		t.Fatal("expected error for already-executed draw")
	}
}

func TestDraw_ExecuteDraw_NoEntries(t *testing.T) {
	db := setupDrawDB(t)
	svc := services.NewDrawService(db)
	drawID := uuid.New()

	db.Exec(`INSERT INTO draws (id, name, status, winner_count) VALUES (?,?,?,?)`,
		drawID.String(), "Empty Draw", "ACTIVE", 3)

	err := svc.ExecuteDraw(context.Background(), drawID)
	// Should fail gracefully — no entries to pick from
	if err == nil {
		t.Fatal("expected error when draw has zero entries")
	}
}

func TestDraw_ExecuteDraw_SelectsWinners(t *testing.T) {
	db := setupDrawDB(t)
	svc := services.NewDrawService(db)
	drawID := uuid.New()

	db.Exec(`INSERT INTO draws (id, name, status, winner_count, prize_value_kobo) VALUES (?,?,?,?,?)`,
		drawID.String(), "Test Draw", "ACTIVE", 3, 500000)

	// Seed 10 entries (more than winner_count=3)
	for i := 0; i < 10; i++ {
		uid := uuid.New()
		db.Exec(`INSERT INTO draw_entries (id, draw_id, user_id, msisdn, entries_count)
			VALUES (?,?,?,?,?)`,
			uuid.New().String(), drawID.String(), uid.String(),
			"0801234"+uid.String()[:4], 1+i%3)
	}

	err := svc.ExecuteDraw(context.Background(), drawID)
	if err != nil {
		t.Fatalf("ExecuteDraw: %v", err)
	}

	// Verify exactly 3 winners were created
	var count int64
	db.Raw("SELECT COUNT(*) FROM draw_winners WHERE draw_id = ?", drawID.String()).Scan(&count)
	if count != 3 {
		t.Fatalf("expected 3 winners, got %d", count)
	}

	// Verify draw is marked COMPLETED
	var status string
	db.Raw("SELECT status FROM draws WHERE id = ?", drawID.String()).Scan(&status)
	if status != "COMPLETED" {
		t.Fatalf("draw status should be COMPLETED, got %s", status)
	}
}

func TestDraw_ExecuteDraw_NoDuplicateWinners(t *testing.T) {
	db := setupDrawDB(t)
	svc := services.NewDrawService(db)
	drawID := uuid.New()

	db.Exec(`INSERT INTO draws (id, name, status, winner_count) VALUES (?,?,?,?)`,
		drawID.String(), "Dedup Test", "ACTIVE", 5)

	// Seed 5 distinct users
	userIDs := make([]string, 5)
	for i := 0; i < 5; i++ {
		uid := uuid.New()
		userIDs[i] = uid.String()
		db.Exec(`INSERT INTO draw_entries (id, draw_id, user_id, msisdn) VALUES (?,?,?,?)`,
			uuid.New().String(), drawID.String(), uid.String(), "0810000"+uid.String()[:4])
	}

	_ = svc.ExecuteDraw(context.Background(), drawID)

	// Check no duplicate user_id in winners
	var winners []struct{ UserID string }
	db.Raw("SELECT user_id FROM draw_winners WHERE draw_id = ?", drawID.String()).Scan(&winners)
	seen := make(map[string]bool)
	for _, w := range winners {
		if seen[w.UserID] {
			t.Fatalf("duplicate winner: user %s won more than once", w.UserID)
		}
		seen[w.UserID] = true
	}
}

func TestDraw_GetWinners_EmptyBeforeExecution(t *testing.T) {
	db := setupDrawDB(t)
	svc := services.NewDrawService(db)
	drawID := uuid.New()

	db.Exec(`INSERT INTO draws (id, name, status) VALUES (?,?,?)`,
		drawID.String(), "New Draw", "ACTIVE")

	winners, err := svc.GetDrawWinners(context.Background(), drawID)
	if err != nil {
		t.Fatalf("GetDrawWinners: %v", err)
	}
	if len(winners) != 0 {
		t.Fatalf("expected 0 winners before execution, got %d", len(winners))
	}
}
