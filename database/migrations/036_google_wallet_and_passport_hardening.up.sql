-- ═══════════════════════════════════════════════════════════════════════════
--  036 — Google Wallet Objects + Digital Passport hardening
--  Loyalty Nexus — Phase: Digital Passport completion
-- ═══════════════════════════════════════════════════════════════════════════

BEGIN;

-- ─── Google Wallet Loyalty Objects ────────────────────────────────────────────
-- Tracks the Google Wallet loyalty object ID per user so we can push updates
-- when their tier, streak, or points change.
CREATE TABLE IF NOT EXISTS google_wallet_objects (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    object_id           TEXT        NOT NULL UNIQUE, -- Google Wallet object ID (issuer.userId)
    class_id            TEXT        NOT NULL,        -- Google Wallet class ID (issuer.LoyaltyNexus)
    last_synced_at      TIMESTAMPTZ,
    points_at_last_sync BIGINT      NOT NULL DEFAULT 0,
    tier_at_last_sync   TEXT        NOT NULL DEFAULT 'BRONZE',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_google_wallet_objects_user ON google_wallet_objects(user_id);
CREATE INDEX        IF NOT EXISTS idx_google_wallet_objects_sync  ON google_wallet_objects(last_synced_at);

-- ─── Wallet Registrations: add push_token_updated_at for staleness tracking ──
ALTER TABLE wallet_registrations
    ADD COLUMN IF NOT EXISTS push_token_updated_at TIMESTAMPTZ DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS is_active             BOOLEAN     NOT NULL DEFAULT TRUE;

-- ─── Users: add google_wallet_object_id shortcut column ──────────────────────
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS google_wallet_object_id TEXT DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS apple_pass_serial        TEXT DEFAULT NULL;

-- ─── Ghost Nudge Log: add channel column to track SMS vs push ────────────────
ALTER TABLE ghost_nudge_log
    ADD COLUMN IF NOT EXISTS channel TEXT NOT NULL DEFAULT 'sms'; -- sms | push | both

-- ─── Passport Push Log: full audit trail of every wallet push ────────────────
CREATE TABLE IF NOT EXISTS passport_push_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform    TEXT        NOT NULL CHECK (platform IN ('apple', 'google')),
    trigger     TEXT        NOT NULL, -- 'tier_change' | 'streak_update' | 'points_milestone' | 'manual'
    status      TEXT        NOT NULL DEFAULT 'pending', -- pending | sent | failed
    error_msg   TEXT,
    pushed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_passport_push_log_user    ON passport_push_log(user_id);
CREATE INDEX IF NOT EXISTS idx_passport_push_log_status  ON passport_push_log(status);
CREATE INDEX IF NOT EXISTS idx_passport_push_log_pushed  ON passport_push_log(pushed_at DESC);

COMMIT;
