-- ============================================================================
-- MIGRATION 073: CONSOLIDATED SCHEMA PATCH
-- ============================================================================
-- This migration ensures that every column expected by the Go entity structs
-- exists in the database. It is fully idempotent (using IF NOT EXISTS).
-- It bridges the gap between the application's domain models and the final
-- state of the database after all previous 72 migrations.
-- ============================================================================

-- 1. ai_generations
ALTER TABLE ai_generations
    ADD COLUMN IF NOT EXISTS output_text    TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS provider       TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS cost_micros    INT         NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS duration_ms    INT         NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS refund_granted BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS refund_pts     BIGINT      NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- 2. studio_tools
ALTER TABLE studio_tools
    ADD COLUMN IF NOT EXISTS icon               TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sort_order         INT         NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS is_free            BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS provider_tool      TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS refund_pct         INT         NOT NULL DEFAULT 100,
    ADD COLUMN IF NOT EXISTS refund_window_mins INT         NOT NULL DEFAULT 5,
    ADD COLUMN IF NOT EXISTS ui_config          JSONB       NOT NULL DEFAULT '{}'::jsonb;

-- 3. ghost_nudge_log
ALTER TABLE ghost_nudge_log
    ADD COLUMN IF NOT EXISTS message    TEXT,
    ADD COLUMN IF NOT EXISTS status     TEXT,
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- 4. google_wallet_objects
-- Guard: create the table if it was never created by migration 036
-- (can happen on DBs provisioned from a dump that pre-dates migration 036)
CREATE TABLE IF NOT EXISTS google_wallet_objects (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    object_id           TEXT        NOT NULL UNIQUE,
    class_id            TEXT        NOT NULL,
    last_synced_at      TIMESTAMPTZ,
    points_at_last_sync BIGINT      NOT NULL DEFAULT 0,
    tier_at_last_sync   TEXT        NOT NULL DEFAULT 'BRONZE',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_google_wallet_objects_user ON google_wallet_objects(user_id);
CREATE INDEX        IF NOT EXISTS idx_google_wallet_objects_sync  ON google_wallet_objects(last_synced_at);
ALTER TABLE google_wallet_objects
    ADD COLUMN IF NOT EXISTS last_sync_status TEXT;

-- 5. spin_results
ALTER TABLE spin_results
    ADD COLUMN IF NOT EXISTS expires_at          TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS admin_notes         TEXT,
    ADD COLUMN IF NOT EXISTS bank_account_name   TEXT,
    ADD COLUMN IF NOT EXISTS bank_account_number TEXT,
    ADD COLUMN IF NOT EXISTS bank_name           TEXT,
    ADD COLUMN IF NOT EXISTS payment_reference   TEXT,
    ADD COLUMN IF NOT EXISTS rejection_reason    TEXT,
    ADD COLUMN IF NOT EXISTS reviewed_at         TIMESTAMPTZ;

-- 6. transactions
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS reference TEXT NOT NULL DEFAULT '';

-- 7. wallet_registrations
ALTER TABLE wallet_registrations
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

-- 8. prize_pool
ALTER TABLE prize_pool
    ADD COLUMN IF NOT EXISTS prize_code           TEXT,
    ADD COLUMN IF NOT EXISTS variation_code       TEXT,
    ADD COLUMN IF NOT EXISTS icon_name            TEXT,
    ADD COLUMN IF NOT EXISTS terms_and_conditions TEXT;

-- 9. wallets
ALTER TABLE wallets
    ADD COLUMN IF NOT EXISTS pulse_counter       BIGINT  NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS draw_counter        BIGINT  NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS daily_recharge_kobo BIGINT  NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS daily_recharge_date DATE,
    ADD COLUMN IF NOT EXISTS daily_spins_awarded INTEGER NOT NULL DEFAULT 0;
