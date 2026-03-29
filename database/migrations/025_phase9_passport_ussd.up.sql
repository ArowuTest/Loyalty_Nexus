-- Migration 025: Phase 9 — Digital Passport extensions + USSD support tables
-- Adds: passport_events, ghost_nudge_log, user_badges (if missing), QR audit log

-- ─── User Badges ───────────────────────────────────────────────────────────────
-- Already may exist from Phase 7 but ensure full schema
CREATE TABLE IF NOT EXISTS user_badges (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_key  TEXT NOT NULL,
    earned_at  TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, badge_key)
);
CREATE INDEX IF NOT EXISTS idx_user_badges_user_id ON user_badges(user_id);

-- ─── Passport Events Log (spec §6.4) ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS passport_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,  -- tier_upgrade | badge_earned | streak_milestone | qr_scanned
    details    JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_passport_events_user_id ON passport_events(user_id);
CREATE INDEX IF NOT EXISTS idx_passport_events_type    ON passport_events(event_type);

-- ─── Ghost Nudge Log (spec §6.3) ──────────────────────────────────────────────
-- Tracks when a user was last nudged to prevent re-nudge within 24h
CREATE TABLE IF NOT EXISTS ghost_nudge_log (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    nudged_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ghost_nudge_user ON ghost_nudge_log(user_id);

-- ─── QR Scan Audit Log ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS qr_scan_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scanned_by  TEXT,           -- partner merchant IP / terminal ID
    is_valid    BOOLEAN NOT NULL DEFAULT TRUE,
    scanned_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_qr_scan_user_id ON qr_scan_log(user_id);

-- ─── Users table — ensure Digital Passport columns exist ─────────────────────
ALTER TABLE users ADD COLUMN IF NOT EXISTS lifetime_points  BIGINT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_spins      INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS studio_use_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_referrals  INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS momo_number      TEXT    NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS momo_verified    BOOLEAN NOT NULL DEFAULT FALSE;

-- ─── Wallets table — ensure spin_credits pool column ─────────────────────────
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS lifetime_points BIGINT NOT NULL DEFAULT 0;

-- ─── USSD Sessions (for stateful multi-turn USSD — Africa's Talking) ─────────
CREATE TABLE IF NOT EXISTS ussd_sessions (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id     TEXT        NOT NULL UNIQUE,
    phone_number   TEXT        NOT NULL,
    menu_state     TEXT        NOT NULL DEFAULT 'root',
    input_buffer   TEXT        NOT NULL DEFAULT '',
    pending_spin_id UUID       REFERENCES spin_results(id) ON DELETE SET NULL,
    expires_at     TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '10 minutes',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ussd_sessions_phone      ON ussd_sessions(phone_number);
CREATE INDEX IF NOT EXISTS idx_ussd_sessions_session_id ON ussd_sessions(session_id);

-- Auto-clean expired USSD sessions (keep table small)
CREATE OR REPLACE FUNCTION cleanup_expired_ussd_sessions() RETURNS void AS $$
BEGIN
    DELETE FROM ussd_sessions WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;
