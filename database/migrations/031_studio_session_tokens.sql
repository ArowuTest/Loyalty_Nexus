-- ============================================================
-- Migration 031: Studio Session Token Model
-- ============================================================
-- Implements the per-tool configurable drain rate system:
--   entry_point_cost  — minimum wallet balance to access a tool
--   refund_window_mins — how long (minutes) user can dispute output
--   refund_pct        — % of points returned on approved dispute (0-100)
--   is_free           — bypasses ALL point checks (e.g. Nexus Chat)
--
-- Adds dispute tracking to ai_generations:
--   disputed_at   — timestamp user flagged the output
--   refund_granted — whether admin/system approved the refund
--   refund_pts    — how many points were actually returned
--
-- Adds studio_sessions table for live utilisation tracking:
--   Tracks total pts spent + generation count per session so the
--   frontend can show "You've used 120pts this session" live.
-- ============================================================

-- ── 1. studio_tools — new configurability columns ────────────────────────────

ALTER TABLE studio_tools
    ADD COLUMN IF NOT EXISTS entry_point_cost   BIGINT  NOT NULL DEFAULT 0
        CONSTRAINT chk_entry_point_cost_nonneg CHECK (entry_point_cost >= 0),
    ADD COLUMN IF NOT EXISTS refund_window_mins INT     NOT NULL DEFAULT 5
        CONSTRAINT chk_refund_window_nonneg    CHECK (refund_window_mins >= 0),
    ADD COLUMN IF NOT EXISTS refund_pct         INT     NOT NULL DEFAULT 100
        CONSTRAINT chk_refund_pct_range        CHECK (refund_pct BETWEEN 0 AND 100),
    ADD COLUMN IF NOT EXISTS is_free            BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN studio_tools.entry_point_cost   IS 'Minimum PulsePoints balance user must hold to open this tool. 0 = no floor.';
COMMENT ON COLUMN studio_tools.refund_window_mins IS 'Minutes after generation during which user can dispute output. 0 = no refunds.';
COMMENT ON COLUMN studio_tools.refund_pct         IS 'Percentage of points_deducted returned on approved dispute (0–100).';
COMMENT ON COLUMN studio_tools.is_free            IS 'When true, entry_point_cost and point_cost checks are bypassed entirely.';

-- ── 2. ai_generations — dispute tracking columns ─────────────────────────────

ALTER TABLE ai_generations
    ADD COLUMN IF NOT EXISTS disputed_at    TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS refund_granted BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS refund_pts     BIGINT      NOT NULL DEFAULT 0
        CONSTRAINT chk_refund_pts_nonneg CHECK (refund_pts >= 0);

COMMENT ON COLUMN ai_generations.disputed_at    IS 'When the user flagged this generation as unsatisfactory.';
COMMENT ON COLUMN ai_generations.refund_granted IS 'Whether the system issued a compensating PulsePoints refund.';
COMMENT ON COLUMN ai_generations.refund_pts     IS 'Actual PulsePoints returned (may be < points_deducted if refund_pct < 100).';

CREATE INDEX IF NOT EXISTS idx_ai_generations_disputed
    ON ai_generations (disputed_at)
    WHERE disputed_at IS NOT NULL;

-- ── 3. studio_sessions — live utilisation tracking ───────────────────────────

CREATE TABLE IF NOT EXISTS studio_sessions (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at         TIMESTAMPTZ,
    total_pts_used   BIGINT      NOT NULL DEFAULT 0
        CONSTRAINT chk_session_pts_nonneg CHECK (total_pts_used >= 0),
    generation_count INT         NOT NULL DEFAULT 0
        CONSTRAINT chk_session_gen_nonneg CHECK (generation_count >= 0)
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

-- ── 4. Seed sensible defaults per tool category ──────────────────────────────
-- These are starting values. Admin can change all of them via the Studio Tools
-- CRUD page without a code deploy (zero-hardcoding rule).

-- Chat tools: fully free — no entry requirement, no generation cost
UPDATE studio_tools
SET    is_free = true,
       entry_point_cost = 0,
       point_cost       = 0,
       refund_window_mins = 0,
       refund_pct         = 0
WHERE  slug IN ('ai-chat', 'nexus-chat', 'web-search-ai');

-- Text / knowledge tools: low entry (20pts), small generation cost already set
UPDATE studio_tools
SET    entry_point_cost  = 20,
       refund_window_mins = 10,
       refund_pct         = 100
WHERE  slug IN (
    'translate', 'study-guide', 'quiz', 'mindmap',
    'research-brief', 'code-helper', 'image-analyser',
    'ask-my-photo', 'slide-deck', 'infographic'
);

-- Image tools: entry 50pts (user must have 50pts to open any image tool)
UPDATE studio_tools
SET    entry_point_cost  = 50,
       refund_window_mins = 5,
       refund_pct         = 100
WHERE  slug IN (
    'ai-photo', 'ai-photo-pro', 'ai-photo-max', 'ai-photo-dream',
    'photo-editor', 'bg-remover'
);

-- Audio tools: entry 30pts
UPDATE studio_tools
SET    entry_point_cost  = 30,
       refund_window_mins = 5,
       refund_pct         = 100
WHERE  slug IN (
    'narrate', 'narrate-pro', 'transcribe', 'transcribe-african',
    'bg-music', 'jingle', 'song-creator', 'instrumental', 'podcast'
);

-- Video tools: highest entry (200pts) — expensive API calls
UPDATE studio_tools
SET    entry_point_cost  = 200,
       refund_window_mins = 10,
       refund_pct         = 50
WHERE  slug IN (
    'animate-photo', 'video-cinematic', 'video-premium',
    'video-veo', 'video-jingle'
);

-- Business plan / build tools: medium entry (50pts)
UPDATE studio_tools
SET    entry_point_cost  = 50,
       refund_window_mins = 10,
       refund_pct         = 100
WHERE  slug IN ('bizplan', 'voice-to-plan');
