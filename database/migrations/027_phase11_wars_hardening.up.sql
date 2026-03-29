-- 027_phase11_wars_hardening.sql
-- Phase 11: Regional Wars entity/repo hardening, leaderboard indices, lifecycle crons
-- ─────────────────────────────────────────────────────────────────────────────
-- This migration:
--   1. Ensures regional_wars table columns match the Go entity (safe with IF NOT EXISTS)
--   2. Adds missing indices for leaderboard performance
--   3. Ensures regional_war_winners table is fully specified
--   4. Adds the correct network_config keys for wars + studio stale recovery
--   5. Removes reference to the now-defunct wars_snapshots table (old Phase 5 approach)
-- ─────────────────────────────────────────────────────────────────────────────

-- BEGIN;  -- removed: managed by golang-migrate

-- ── 1. Ensure regional_wars has all required columns ─────────────────────────

ALTER TABLE regional_wars
    ADD COLUMN IF NOT EXISTS created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Ensure period is unique (may already exist)
CREATE UNIQUE INDEX IF NOT EXISTS uidx_regional_wars_period ON regional_wars(period);

-- ── 2. Leaderboard performance index ─────────────────────────────────────────
-- The leaderboard query filters transactions WHERE type='points_award' AND points_delta > 0
-- across a time window joined to users.state. This composite index helps.

CREATE INDEX IF NOT EXISTS idx_tx_leaderboard
    ON transactions(user_id, type, points_delta, created_at)
    WHERE type = 'points_award' AND points_delta > 0;

-- users.state index for GROUP BY
CREATE INDEX IF NOT EXISTS idx_users_state
    ON users(state)
    WHERE state IS NOT NULL AND state <> '';

-- users.is_active partial index
CREATE INDEX IF NOT EXISTS idx_users_active_state
    ON users(state, is_active)
    WHERE is_active = true AND state IS NOT NULL AND state <> '';

-- ── 3. Ensure regional_war_winners has all required columns ──────────────────

ALTER TABLE regional_war_winners
    ADD COLUMN IF NOT EXISTS created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_war_winners_war_id ON regional_war_winners(war_id);
CREATE INDEX IF NOT EXISTS idx_war_winners_state  ON regional_war_winners(state);

-- updated_at trigger on regional_war_winners
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_war_winners_updated_at ON regional_war_winners;
CREATE TRIGGER trg_war_winners_updated_at
    BEFORE UPDATE ON regional_war_winners
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trg_regional_wars_updated_at ON regional_wars;
CREATE TRIGGER trg_regional_wars_updated_at
    BEFORE UPDATE ON regional_wars
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ── 4. Network config keys ────────────────────────────────────────────────────

INSERT INTO network_configs (key, value, description) VALUES
    ('regional_wars_prize_pool_kobo',    '50000000', 'Monthly Regional Wars prize pool in kobo (default ₦500,000)'),
    ('regional_wars_winning_bonus',      '50',       'Pulse Points bonus awarded to every member of a winning state'),
    ('studio_stale_job_timeout_secs',    '600',      'Seconds after which a pending/processing AI generation is considered stale'),
    ('studio_stale_job_batch_size',      '20',       'Max stale jobs refunded per lifecycle cron run'),
    ('lifecycle_wars_resolve_enabled',   'true',     'Enable auto-resolve of wars on month end'),
    ('lifecycle_studio_stale_enabled',   'true',     'Enable stale studio job recovery cron')
ON CONFLICT (key) DO NOTHING;

-- ── 5. Drop the old wars_snapshots table (Phase 5 artefact — no longer used) ─
-- Guarded: only drops if the table exists and has zero rows (safety check).
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_name = 'wars_snapshots'
    ) THEN
        -- Only drop if empty; preserve data if an operator has historical rows
        IF (SELECT COUNT(*) FROM wars_snapshots) = 0 THEN
            DROP TABLE wars_snapshots;
            RAISE NOTICE 'Dropped empty wars_snapshots table';
        ELSE
            RAISE NOTICE 'wars_snapshots has rows — skipping drop; please migrate manually';
        END IF;
    END IF;
END $$;

-- ── 6. Auto-create the current month war if none exists ──────────────────────
-- Idempotent: ON CONFLICT DO NOTHING.
INSERT INTO regional_wars (
    id,
    period,
    status,
    total_prize_kobo,
    starts_at,
    ends_at
)
SELECT
    gen_random_uuid(),
    TO_CHAR(NOW(), 'YYYY-MM'),
    'ACTIVE',
    50000000,
    DATE_TRUNC('month', NOW()),
    (DATE_TRUNC('month', NOW()) + INTERVAL '1 month - 1 second')
WHERE NOT EXISTS (
    SELECT 1 FROM regional_wars WHERE period = TO_CHAR(NOW(), 'YYYY-MM')
);

-- COMMIT;  -- removed: managed by golang-migrate
