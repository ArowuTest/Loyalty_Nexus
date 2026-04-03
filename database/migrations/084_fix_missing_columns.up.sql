-- Migration 084: Fix missing columns causing SQLSTATE 42703 errors in Render logs
--
-- Root cause: Migration 060 (ensure_critical_tables) recreated spin_results and
-- fraud_events with stripped-down schemas that omitted columns present in migration 020.
-- On fresh DB instances where 060 ran before 020 (or 020 was skipped), these columns
-- are absent, causing runtime errors in:
--   - admin_handler.go:103  (fulfillment_status on spin_results)
--   - admin_handler.go:1133 (fulfillment_status on spin_results)
--   - admin_handler.go:1139 (resolved on fraud_events)
--   - spin_service.go:860   (fulfillment_status on spin_results)
--
-- All statements use ADD COLUMN IF NOT EXISTS — safe to run on any DB state.

-- ── 1. spin_results: add fulfillment_status ──────────────────────────────────
ALTER TABLE spin_results
    ADD COLUMN IF NOT EXISTS fulfillment_status VARCHAR(30) NOT NULL DEFAULT 'pending'
        CHECK (fulfillment_status IN (
            'na','pending','pending_momo_setup','pending_claim',
            'processing','completed','failed','held'
        ));

-- Backfill: rows created by migration 060 used is_fulfilled BOOLEAN instead.
-- Map TRUE → 'completed', FALSE → 'pending' for any rows that have is_fulfilled set.
UPDATE spin_results
SET fulfillment_status = CASE
    WHEN is_fulfilled IS TRUE THEN 'completed'
    ELSE 'pending'
END
WHERE fulfillment_status = 'pending'
  AND is_fulfilled IS NOT NULL;

-- Add index to match the one from migration 020
CREATE INDEX IF NOT EXISTS idx_spin_results_fulfillment_status
    ON spin_results(fulfillment_status);

-- ── 2. spin_results: add other columns that 060 omitted ──────────────────────
ALTER TABLE spin_results
    ADD COLUMN IF NOT EXISTS prize_value      DECIMAL(12,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS slot_index       INTEGER,
    ADD COLUMN IF NOT EXISTS fulfillment_ref  VARCHAR(255),
    ADD COLUMN IF NOT EXISTS momo_number      VARCHAR(15),
    ADD COLUMN IF NOT EXISTS error_message    TEXT,
    ADD COLUMN IF NOT EXISTS claimed_at       TIMESTAMPTZ;

-- ── 3. fraud_events: add resolved ────────────────────────────────────────────
ALTER TABLE fraud_events
    ADD COLUMN IF NOT EXISTS resolved     BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS resolved_by  UUID,
    ADD COLUMN IF NOT EXISTS resolved_at  TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS rule_name    VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS severity     VARCHAR(20)  NOT NULL DEFAULT 'medium'
        CHECK (severity IN ('low','medium','high','critical'));

-- Add index to match the one from migration 020
CREATE INDEX IF NOT EXISTS idx_fraud_events_unresolved
    ON fraud_events(resolved, created_at DESC);
