-- ═══════════════════════════════════════════════════════════════════
--  021 — Digital Passport badges + Regional Wars support tables
--  Loyalty Nexus — Phase 6
-- ═══════════════════════════════════════════════════════════════════

BEGIN;

-- ── User Badges ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_badges (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_key   VARCHAR(64) NOT NULL,
    earned_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, badge_key)
);
CREATE INDEX IF NOT EXISTS idx_user_badges_user_id ON user_badges(user_id);
CREATE INDEX IF NOT EXISTS idx_user_badges_key     ON user_badges(badge_key);

-- ── Regional Wars ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS regional_wars (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    period              VARCHAR(7)  NOT NULL UNIQUE,  -- YYYY-MM
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',  -- ACTIVE|COMPLETED
    total_prize_kobo    BIGINT      NOT NULL DEFAULT 50000000,   -- ₦500,000 default
    starts_at           TIMESTAMPTZ NOT NULL,
    ends_at             TIMESTAMPTZ NOT NULL,
    resolved_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_regional_wars_status ON regional_wars(status);

-- ── Regional War Winners ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS regional_war_winners (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    war_id          UUID        NOT NULL REFERENCES regional_wars(id),
    state           VARCHAR(64) NOT NULL,
    rank            SMALLINT    NOT NULL,
    total_points    BIGINT      NOT NULL DEFAULT 0,
    prize_kobo      BIGINT      NOT NULL DEFAULT 0,
    status          VARCHAR(30) NOT NULL DEFAULT 'PENDING',  -- PENDING|PAID
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Draws ────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS draws (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(120) NOT NULL,
    status              VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
    winner_count        INT          NOT NULL DEFAULT 3,
    prize_type          VARCHAR(40)  NOT NULL DEFAULT 'MOMO_CASH',
    prize_value_kobo    BIGINT       NOT NULL DEFAULT 500000,   -- ₦5,000 default
    executed_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ── Draw Entries ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS draw_entries (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    draw_id         UUID        NOT NULL REFERENCES draws(id),
    user_id         UUID        NOT NULL REFERENCES users(id),
    phone_number    VARCHAR(20) NOT NULL,
    ticket_count    INT         NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (draw_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_draw_entries_draw_id ON draw_entries(draw_id);

-- ── Draw Winners ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS draw_winners (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    draw_id             UUID        NOT NULL REFERENCES draws(id),
    user_id             UUID        NOT NULL REFERENCES users(id),
    phone_number        VARCHAR(20) NOT NULL,
    position            SMALLINT    NOT NULL,
    prize_type          VARCHAR(40) NOT NULL,
    prize_value_kobo    BIGINT      NOT NULL DEFAULT 0,
    status              VARCHAR(30) NOT NULL DEFAULT 'PENDING_FULFILLMENT',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Extend users table with new columns (safe adds) ─────────────────
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS state              VARCHAR(64),
    ADD COLUMN IF NOT EXISTS total_spins        INT       NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS studio_use_count   INT       NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_referrals    INT       NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS lifetime_points    BIGINT    NOT NULL DEFAULT 0;

-- Update lifetime_points from wallets table if present
UPDATE users u
SET lifetime_points = COALESCE(
    (SELECT lifetime_points FROM wallets w WHERE w.user_id = u.id LIMIT 1), 0)
WHERE lifetime_points = 0;

-- Seed a default monthly draw
INSERT INTO draws (id, name, status, winner_count, prize_type, prize_value_kobo)
VALUES (gen_random_uuid(), 'Monthly Grand Draw', 'ACTIVE', 3, 'MOMO_CASH', 5000000)
ON CONFLICT DO NOTHING;

COMMIT;
