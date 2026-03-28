-- Migration 041: Add claim lifecycle fields to spin_results
-- Aligns with RechargeMax claim/fulfillment flow.
-- claim_status is separate from fulfillment_status:
--   fulfillment_status = internal VTPass/MoMo dispatch state
--   claim_status       = user-facing claim lifecycle (PENDING → CLAIMED / PENDING_ADMIN_REVIEW → APPROVED / REJECTED)

BEGIN;

-- ── Claim lifecycle ────────────────────────────────────────────────────────
ALTER TABLE spin_results
    ADD COLUMN IF NOT EXISTS claim_status         TEXT        NOT NULL DEFAULT 'PENDING',
    ADD COLUMN IF NOT EXISTS expires_at           TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '30 days');

-- ── MoMo / bank payout details (supplied by user at claim time) ────────────
ALTER TABLE spin_results
    ADD COLUMN IF NOT EXISTS momo_claim_number    TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS bank_account_number  TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS bank_account_name    TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS bank_name            TEXT        NOT NULL DEFAULT '';

-- ── Admin review metadata ──────────────────────────────────────────────────
ALTER TABLE spin_results
    ADD COLUMN IF NOT EXISTS reviewed_by          UUID        REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS reviewed_at          TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS rejection_reason     TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS admin_notes          TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS payment_reference    TEXT        NOT NULL DEFAULT '';

-- ── Indexes for admin claim list queries ──────────────────────────────────
CREATE INDEX IF NOT EXISTS idx_spin_results_claim_status  ON spin_results (claim_status);
CREATE INDEX IF NOT EXISTS idx_spin_results_expires_at    ON spin_results (expires_at);
CREATE INDEX IF NOT EXISTS idx_spin_results_reviewed_by   ON spin_results (reviewed_by);

-- ── Seed: back-fill claim_status for existing rows ────────────────────────
-- Rows that are already fulfilled → CLAIMED
-- Rows that are pending MoMo setup → PENDING (user hasn't linked MoMo yet)
-- All others → PENDING
UPDATE spin_results
    SET claim_status = 'CLAIMED'
    WHERE fulfillment_status IN ('completed')
      AND claim_status = 'PENDING';

COMMIT;
