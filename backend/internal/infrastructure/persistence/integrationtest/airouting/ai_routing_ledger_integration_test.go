// Package airouting contains real Postgres integration tests for the AI routing
// V2 attempt ledger and routing change-log (migrations 131 + 133).
//
// They pin the invariants the V2 review found broken:
//
//   - B2: deleting a generation (the lifecycle worker's retention path) must NOT
//     erase its attempt rows — generation_id is nulled, cost/outcome survive.
//   - M1: a STARTED attempt that never gets finalized is closed by
//     ReconcileStrandedAttempts; recent in-flight and terminal rows are untouched.
//   - M2: the ledger is append-only — UpdateAttempt refuses to touch a terminal
//     row (ErrAttemptNotFinalizable), and the DB trigger rejects any raw UPDATE.
//   - M3: RecordRoutingChange persists for every state shape (before/after land
//     as queryable jsonb). Pre-fix, a nil state became SQL NULL, violated NOT NULL,
//     and the row vanished because every caller discards the error.
//
// Prerequisites: a Postgres database with ALL migrations applied (>= 133).
//
// Run with:
//
//	TEST_DATABASE_URL="postgres://nexus_test:nexus_test@localhost:5432/loyalty_nexus_test?sslmode=disable" \
//	go test ./internal/infrastructure/persistence/integrationtest/airouting/... -v
package airouting_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"loyalty-nexus/internal/domain/entities"
	"loyalty-nexus/internal/infrastructure/persistence"
)

// ledgerMigration is the first schema version carrying the SET NULL foreign key
// and the append-only trigger these tests assert.
const ledgerMigration = 133

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
	// A reachable but under-migrated database is a setup error, not a skip:
	// skipping here would turn a missing migration into a green run.
	var version int
	if err := db.Raw("SELECT version FROM schema_migrations").Scan(&version).Error; err != nil {
		t.Fatalf("read schema_migrations: %v", err)
	}
	if version < ledgerMigration {
		t.Fatalf("test database is at migration %d; these tests need >= %d (apply migrations first)", version, ledgerMigration)
	}
	return db
}

// withTx runs fn inside a transaction that is always rolled back, so tests
// leave no rows behind and never see each other's fixtures.
func withTx(t *testing.T, db *gorm.DB, fn func(tx *gorm.DB)) {
	t.Helper()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin tx: %v", tx.Error)
	}
	defer tx.Rollback() //nolint:errcheck
	fn(tx)
}

// seedGeneration creates the user → tool → generation chain a ledger row hangs
// off, exactly as the production retention sweep would later find it (completed
// and already expired).
func seedGeneration(t *testing.T, tx *gorm.DB) (genID, toolID uuid.UUID) {
	t.Helper()
	userID, toolID, genID := uuid.New(), uuid.New(), uuid.New()
	nonce := fmt.Sprintf("%d%04d", time.Now().UnixNano()%1_000_000_000, rand.Intn(10_000))
	if err := tx.Exec(`
		INSERT INTO users (id, phone_number, user_code, tier, is_active, created_at, updated_at)
		VALUES (?, ?, ?, 'BRONZE', true, NOW(), NOW())`,
		userID, "2348"+nonce[:9], "LEDGER"+nonce[:6]).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := tx.Exec(`
		INSERT INTO studio_tools (id, name, slug, provider, is_active)
		VALUES (?, ?, ?, '', true)`,
		toolID, "Ledger Test Tool "+nonce, "ledger-test-"+nonce).Error; err != nil {
		t.Fatalf("seed tool: %v", err)
	}
	if err := tx.Exec(`
		INSERT INTO ai_generations (id, user_id, tool_id, prompt, status, points_deducted, expires_at)
		VALUES (?, ?, ?, 'ledger integration test', 'completed', 0, NOW() - INTERVAL '1 day')`,
		genID, userID, toolID).Error; err != nil {
		t.Fatalf("seed generation: %v", err)
	}
	return genID, toolID
}

type attemptRow struct {
	Outcome      string
	ErrorClass   string
	CostMicros   int
	GenerationID *uuid.UUID
	CompletedAt  *time.Time
}

func loadAttempt(t *testing.T, tx *gorm.DB, id uuid.UUID) attemptRow {
	t.Helper()
	var row attemptRow
	err := tx.Raw(`SELECT outcome, error_class, cost_micros, generation_id, completed_at
	               FROM ai_generation_attempts WHERE id = ?`, id).Scan(&row).Error
	if err != nil {
		t.Fatalf("load attempt %s: %v", id, err)
	}
	return row
}

// ─── B2: attempts survive generation deletion ────────────────────────────────

func TestLedger_AttemptSurvivesGenerationDeletion(t *testing.T) {
	db := openTestDB(t)
	withTx(t, db, func(tx *gorm.DB) {
		ctx := context.Background()
		repo := persistence.NewAIRoutingRepository(tx)
		genID, toolID := seedGeneration(t, tx)

		attempt := &entities.AIGenerationAttempt{
			GenerationID: &genID, ToolID: &toolID, StageKey: "main", AttemptNo: 1,
			ProviderSlug: "groq", ModelID: "llama-4-scout", Outcome: "STARTED", StartedAt: time.Now(),
		}
		if err := repo.RecordAttempt(ctx, attempt); err != nil {
			t.Fatalf("RecordAttempt: %v", err)
		}
		done := time.Now()
		if err := repo.UpdateAttempt(ctx, attempt.ID, map[string]interface{}{
			"outcome": "SUCCEEDED", "cost_micros": 7, "completed_at": done,
		}); err != nil {
			t.Fatalf("UpdateAttempt: %v", err)
		}

		// The production retention path: lifecycle_worker → StudioRepository.DeleteGeneration.
		if err := persistence.NewPostgresStudioRepository(tx).DeleteGeneration(ctx, genID); err != nil {
			t.Fatalf("DeleteGeneration: %v", err)
		}

		var gens int64
		tx.Raw("SELECT count(*) FROM ai_generations WHERE id = ?", genID).Scan(&gens)
		if gens != 0 {
			t.Fatalf("generation should be deleted, %d rows remain", gens)
		}
		var attempts int64
		tx.Raw("SELECT count(*) FROM ai_generation_attempts WHERE id = ?", attempt.ID).Scan(&attempts)
		if attempts != 1 {
			t.Fatalf("B2 regression: attempt ledger row was cascade-deleted with its generation (rows=%d)", attempts)
		}
		row := loadAttempt(t, tx, attempt.ID)
		if row.GenerationID != nil {
			t.Fatalf("generation_id should be nulled after deletion, got %v", *row.GenerationID)
		}
		if row.Outcome != "SUCCEEDED" || row.CostMicros != 7 {
			t.Fatalf("attempt telemetry must survive: outcome=%q cost=%d", row.Outcome, row.CostMicros)
		}
	})
}

// ─── M2: append-only ─────────────────────────────────────────────────────────

func TestLedger_TerminalAttemptIsImmutable(t *testing.T) {
	db := openTestDB(t)
	withTx(t, db, func(tx *gorm.DB) {
		ctx := context.Background()
		repo := persistence.NewAIRoutingRepository(tx)

		attempt := &entities.AIGenerationAttempt{
			StageKey: "main", AttemptNo: 1, ProviderSlug: "groq", Outcome: "STARTED", StartedAt: time.Now(),
		}
		if err := repo.RecordAttempt(ctx, attempt); err != nil {
			t.Fatalf("RecordAttempt: %v", err)
		}
		done := time.Now()
		if err := repo.UpdateAttempt(ctx, attempt.ID, map[string]interface{}{
			"outcome": "SUCCEEDED", "cost_micros": 42, "completed_at": done,
		}); err != nil {
			t.Fatalf("first finalize must succeed: %v", err)
		}

		// Repo guard: a second finalize is refused, loudly.
		err := repo.UpdateAttempt(ctx, attempt.ID, map[string]interface{}{"outcome": "FAILED", "error_class": "X"})
		if !errors.Is(err, persistence.ErrAttemptNotFinalizable) {
			t.Fatalf("second finalize: want ErrAttemptNotFinalizable, got %v", err)
		}
		// ...and so is finalizing a row that was never recorded.
		err = repo.UpdateAttempt(ctx, uuid.New(), map[string]interface{}{"outcome": "FAILED"})
		if !errors.Is(err, persistence.ErrAttemptNotFinalizable) {
			t.Fatalf("finalize of unknown attempt: want ErrAttemptNotFinalizable, got %v", err)
		}

		// DB guard: even a raw UPDATE that bypasses the repository is rejected.
		// The failing statement aborts the transaction, so fence it with a savepoint.
		if err := tx.SavePoint("before_raw_update").Error; err != nil {
			t.Fatalf("savepoint: %v", err)
		}
		rawErr := tx.Exec("UPDATE ai_generation_attempts SET cost_micros = 999 WHERE id = ?", attempt.ID).Error
		if rawErr == nil || !strings.Contains(rawErr.Error(), "append-only") {
			t.Fatalf("raw UPDATE of a terminal attempt must be rejected by the trigger, got: %v", rawErr)
		}
		if err := tx.RollbackTo("before_raw_update").Error; err != nil {
			t.Fatalf("rollback to savepoint: %v", err)
		}

		row := loadAttempt(t, tx, attempt.ID)
		if row.Outcome != "SUCCEEDED" || row.CostMicros != 42 {
			t.Fatalf("terminal row mutated: outcome=%q cost=%d", row.Outcome, row.CostMicros)
		}

		// The trigger's single exemption is the FK ON DELETE SET NULL shape: an
		// unlink that changes generation_id to NULL and nothing else. Pin that it is
		// exactly that narrow — an unlink smuggling another change is still rejected.
		genID, _ := seedGeneration(t, tx)
		linked := &entities.AIGenerationAttempt{
			GenerationID: &genID, StageKey: "main", AttemptNo: 1, ProviderSlug: "groq",
			Outcome: "STARTED", StartedAt: time.Now(),
		}
		if err := repo.RecordAttempt(ctx, linked); err != nil {
			t.Fatalf("RecordAttempt(linked): %v", err)
		}
		if err := repo.UpdateAttempt(ctx, linked.ID, map[string]interface{}{"outcome": "FAILED", "error_class": "PROVIDER_ERROR"}); err != nil {
			t.Fatalf("finalize linked: %v", err)
		}
		if err := tx.SavePoint("before_smuggled_update").Error; err != nil {
			t.Fatalf("savepoint: %v", err)
		}
		smuggled := tx.Exec("UPDATE ai_generation_attempts SET generation_id = NULL, cost_micros = 999 WHERE id = ?", linked.ID).Error
		if smuggled == nil || !strings.Contains(smuggled.Error(), "append-only") {
			t.Fatalf("unlink bundled with another change must be rejected, got: %v", smuggled)
		}
		if err := tx.RollbackTo("before_smuggled_update").Error; err != nil {
			t.Fatalf("rollback to savepoint: %v", err)
		}
		if err := tx.Exec("UPDATE ai_generation_attempts SET generation_id = NULL WHERE id = ?", linked.ID).Error; err != nil {
			t.Fatalf("bare unlink (the SET NULL cascade shape) must be permitted on a terminal row: %v", err)
		}
		if row := loadAttempt(t, tx, linked.ID); row.GenerationID != nil || row.Outcome != "FAILED" || row.CostMicros != 0 {
			t.Fatalf("bare unlink should only null generation_id: %+v", row)
		}
	})
}

// ─── M1: stranded STARTED rows are reconciled ────────────────────────────────

func TestLedger_ReconcileStrandedAttempts(t *testing.T) {
	db := openTestDB(t)
	withTx(t, db, func(tx *gorm.DB) {
		ctx := context.Background()
		repo := persistence.NewAIRoutingRepository(tx)
		mk := func(outcome string, startedAgo time.Duration) uuid.UUID {
			a := &entities.AIGenerationAttempt{
				StageKey: "main", AttemptNo: 1, ProviderSlug: "groq",
				Outcome: outcome, StartedAt: time.Now().Add(-startedAgo),
			}
			if outcome != "STARTED" {
				done := a.StartedAt.Add(time.Second)
				a.CompletedAt = &done
			}
			if err := repo.RecordAttempt(ctx, a); err != nil {
				t.Fatalf("RecordAttempt(%s): %v", outcome, err)
			}
			return a.ID
		}
		stranded := mk("STARTED", 20*time.Minute)   // crashed mid-call: must be closed
		inFlight := mk("STARTED", 1*time.Minute)    // genuinely running: must be left alone
		terminal := mk("SUCCEEDED", 30*time.Minute) // already final: must be untouched

		closed, err := repo.ReconcileStrandedAttempts(ctx, 15*time.Minute)
		if err != nil {
			t.Fatalf("ReconcileStrandedAttempts: %v", err)
		}
		if closed < 1 {
			t.Fatalf("expected at least the stranded row to be closed, got %d", closed)
		}

		if row := loadAttempt(t, tx, stranded); row.Outcome != "FAILED" || row.ErrorClass != "STRANDED" || row.CompletedAt == nil {
			t.Fatalf("stranded row not reconciled: %+v", row)
		}
		if row := loadAttempt(t, tx, inFlight); row.Outcome != "STARTED" || row.CompletedAt != nil {
			t.Fatalf("in-flight row inside the grace window must stay STARTED: %+v", row)
		}
		if row := loadAttempt(t, tx, terminal); row.Outcome != "SUCCEEDED" || row.ErrorClass != "" {
			t.Fatalf("terminal row must be untouched: %+v", row)
		}
	})
}

// ─── M3: the routing change-log actually persists ────────────────────────────

func TestChangeLog_RecordRoutingChangePersistsJSONB(t *testing.T) {
	db := openTestDB(t)
	withTx(t, db, func(tx *gorm.DB) {
		ctx := context.Background()
		repo := persistence.NewAIRoutingRepository(tx)
		entityID := uuid.New()

		type binding struct {
			Priority int    `json:"priority"`
			CostTier string `json:"cost_tier"`
		}
		before := map[string]interface{}{"priority": 1, "cost_tier": "FREE", "is_active": true}
		after := binding{Priority: 2, CostTier: "PREMIUM"}

		if err := repo.RecordRoutingChange(ctx, "binding", entityID, "update", before, after, "admin@test"); err != nil {
			t.Fatalf("RecordRoutingChange: %v", err)
		}

		var got struct {
			N              int64
			BeforePriority string
			AfterTier      string
			BeforeType     string
		}
		err := tx.Raw(`
			SELECT count(*) AS n,
			       max(before_state->>'priority')    AS before_priority,
			       max(after_state->>'cost_tier')    AS after_tier,
			       max(jsonb_typeof(before_state))   AS before_type
			FROM ai_routing_change_log
			WHERE entity_id = ? AND entity_type = 'binding' AND action = 'update' AND changed_by = 'admin@test'`,
			entityID).Scan(&got).Error
		if err != nil {
			t.Fatalf("query change log: %v", err)
		}
		if got.N != 1 {
			t.Fatalf("M3 regression: expected exactly 1 change-log row, got %d", got.N)
		}
		if got.BeforeType != "object" || got.BeforePriority != "1" || got.AfterTier != "PREMIUM" {
			t.Fatalf("jsonb payloads not queryable: type=%q before.priority=%q after.cost_tier=%q",
				got.BeforeType, got.BeforePriority, got.AfterTier)
		}

		// nil states (e.g. a create has no "before") must still produce a row:
		// JSON null is a valid jsonb value and satisfies NOT NULL.
		createID := uuid.New()
		if err := repo.RecordRoutingChange(ctx, "binding", createID, "create", nil, after, "admin@test"); err != nil {
			t.Fatalf("RecordRoutingChange(nil before): %v", err)
		}
		var beforeType string
		tx.Raw("SELECT jsonb_typeof(before_state) FROM ai_routing_change_log WHERE entity_id = ?", createID).Scan(&beforeType)
		if beforeType != "null" {
			t.Fatalf("nil before_state should persist as jsonb null, got %q", beforeType)
		}
	})
}
