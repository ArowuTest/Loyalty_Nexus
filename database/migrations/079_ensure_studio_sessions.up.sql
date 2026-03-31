-- Migration 079: Ensure studio_sessions table exists
-- This table was defined in migration 031 but may not exist in all production DB paths
-- that were bootstrapped via migration 060 (ensure_critical_tables).

CREATE TABLE IF NOT EXISTS studio_sessions (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at        TIMESTAMPTZ,
    total_pts_used  INT         NOT NULL DEFAULT 0,
    generation_count INT        NOT NULL DEFAULT 0,
    last_active_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  studio_sessions                    IS 'One row per AI Studio session. Updated on each generation to power the live utilisation meter.';
COMMENT ON COLUMN studio_sessions.total_pts_used     IS 'Running total of PulsePoints spent across all generations in this session.';
COMMENT ON COLUMN studio_sessions.generation_count   IS 'Number of generations initiated in this session (regardless of status).';
COMMENT ON COLUMN studio_sessions.last_active_at     IS 'Updated on each generation; used to detect idle/expired sessions.';

CREATE INDEX IF NOT EXISTS idx_studio_sessions_user_id    ON studio_sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_studio_sessions_started_at ON studio_sessions (started_at DESC);
CREATE INDEX IF NOT EXISTS idx_studio_sessions_active
    ON studio_sessions (user_id, last_active_at DESC)
    WHERE ended_at IS NULL;
