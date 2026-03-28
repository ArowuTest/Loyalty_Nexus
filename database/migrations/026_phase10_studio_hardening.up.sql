-- ============================================================
-- Migration 026: Phase 10 — Nexus Studio schema hardening
-- Adds slug, sort_order, timestamps to studio_tools;
-- adds output_text, provider, cost_micros, duration_ms,
-- tool_slug, updated_at to ai_generations;
-- seeds all 17 canonical tools;
-- creates chat_sessions + chat_messages tables.
-- ============================================================

BEGIN;

-- ─── studio_tools: add new columns ───────────────────────────────────────────

ALTER TABLE studio_tools
    ADD COLUMN IF NOT EXISTS slug          TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sort_order    INT         NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS provider_tool TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Unique index on slug (used by FindToolBySlug)
CREATE UNIQUE INDEX IF NOT EXISTS uidx_studio_tools_slug ON studio_tools (slug);

-- Back-fill slugs for any existing rows using the name column
UPDATE studio_tools
SET slug = LOWER(REGEXP_REPLACE(TRIM(name), '[\s_]+', '-', 'g'))
WHERE slug = '';

-- ─── ai_generations: add new columns ────────────────────────────────────────

ALTER TABLE ai_generations
    ADD COLUMN IF NOT EXISTS tool_slug    TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS output_text  TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS provider     TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS cost_micros  INT         NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS duration_ms  INT         NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Index for gallery queries (user + status + expiry)
CREATE INDEX IF NOT EXISTS idx_ai_gen_user_status ON ai_generations (user_id, status, expires_at DESC);

-- Index for stale-job watchdog
CREATE INDEX IF NOT EXISTS idx_ai_gen_pending_created ON ai_generations (status, created_at)
    WHERE status IN ('pending', 'processing');

-- ─── Seed canonical 17 tools ─────────────────────────────────────────────────
-- All costs are stored in DB — never hardcoded in application layer.
-- Point costs are in PulsePoints (1 PP = ₦200 recharge equivalent).

INSERT INTO studio_tools
    (id, name, slug, description, category, point_cost, provider, provider_tool,
     icon, sort_order, is_active, created_at, updated_at)
VALUES
-- ── Learn ─────────────────────────────────────────────────────────────────
(gen_random_uuid(), 'Translate',        'translate',        'Translate text between languages',               'Learn',  1, 'groq',        'llama-3.3-70b-versatile',            '🌍',  1,  true, NOW(), NOW()),
(gen_random_uuid(), 'Study Guide',      'study-guide',      'Create a comprehensive study guide',             'Learn',  2, 'groq',        'llama-3.3-70b-versatile',            '📖',  2,  true, NOW(), NOW()),
(gen_random_uuid(), 'Quiz Generator',   'quiz',             'Generate multiple-choice quizzes',               'Learn',  2, 'groq',        'llama-3.3-70b-versatile',            '🧠',  3,  true, NOW(), NOW()),
(gen_random_uuid(), 'Mind Map',         'mindmap',          'Turn any topic into a visual mind map',          'Learn',  2, 'groq',        'llama-3.3-70b-versatile',            '🗺️', 4,  true, NOW(), NOW()),
-- ── Build ─────────────────────────────────────────────────────────────────
(gen_random_uuid(), 'Research Brief',   'research-brief',   'Produce a concise research brief',               'Build',  3, 'groq',        'llama-3.3-70b-versatile',            '📊',  5,  true, NOW(), NOW()),
(gen_random_uuid(), 'Business Plan',    'bizplan',          'One-page Nigerian market business plan',         'Build',  5, 'groq',        'llama-3.3-70b-versatile',            '💼',  6,  true, NOW(), NOW()),
(gen_random_uuid(), 'Slide Deck',       'slide-deck',       '10-slide presentation outline in JSON',          'Build',  5, 'groq',        'llama-3.3-70b-versatile',            '📑',  7,  true, NOW(), NOW()),
-- ── Create ────────────────────────────────────────────────────────────────
(gen_random_uuid(), 'AI Photo',         'ai-photo',         'Generate a high-quality AI image',               'Create', 5, 'fal.ai',      'fal-ai/flux/dev',                    '🖼️', 8,  true, NOW(), NOW()),
(gen_random_uuid(), 'Background Remover','bg-remover',      'Remove image background instantly',              'Create', 3, 'fal.ai',      'fal-ai/birefnet',                    '✂️', 9,  true, NOW(), NOW()),
(gen_random_uuid(), 'Animate Photo',    'animate-photo',    'Bring a still photo to life',                    'Create', 10,'fal.ai',      'fal-ai/kling-video/v1.5/standard',   '🎬', 10, true, NOW(), NOW()),
(gen_random_uuid(), 'Video Premium',    'video-premium',    'AI text-to-video (Kling Pro)',                   'Create', 20,'fal.ai',      'fal-ai/kling-video/v1.5/pro',        '🎥', 11, true, NOW(), NOW()),
(gen_random_uuid(), 'Narrate',          'narrate',          'Convert text to natural-sounding speech',        'Create', 5, 'elevenlabs',  'eleven_turbo_v2',                    '🎙️',12, true, NOW(), NOW()),
(gen_random_uuid(), 'Transcribe',       'transcribe',       'Transcribe audio to text (Whisper)',             'Create', 3, 'groq',        'whisper-large-v3',                   '📝', 13, true, NOW(), NOW()),
(gen_random_uuid(), 'Jingle',          'jingle',           'Generate a short AI music jingle',               'Create', 8, 'mubert',      'RecordTrackTTM',                     '🎵', 14, true, NOW(), NOW()),
(gen_random_uuid(), 'Background Music','bg-music',         'Generate 60s background music track',            'Create', 8, 'mubert',      'RecordTrackTTM',                     '🎶', 15, true, NOW(), NOW()),
-- ── Chat ──────────────────────────────────────────────────────────────────
(gen_random_uuid(), 'Podcast',         'podcast',          'Script + narrate a 2-host audio podcast',       'Create', 10,'groq+elevenlabs','composite',                        '🎧', 16, true, NOW(), NOW()),
(gen_random_uuid(), 'Infographic',     'infographic',      'Data layout JSON + AI visual render',            'Create', 8, 'groq+fal.ai', 'composite',                          '📊', 17, true, NOW(), NOW())

ON CONFLICT (slug) DO UPDATE
    SET name          = EXCLUDED.name,
        description   = EXCLUDED.description,
        category      = EXCLUDED.category,
        point_cost    = EXCLUDED.point_cost,
        provider      = EXCLUDED.provider,
        provider_tool = EXCLUDED.provider_tool,
        icon          = EXCLUDED.icon,
        sort_order    = EXCLUDED.sort_order,
        updated_at    = NOW();

-- ─── chat_sessions ────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS chat_sessions (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title        TEXT        NOT NULL DEFAULT 'Nexus Chat',
    -- Rolling summary written after every 10 messages (spec §9.5)
    summary      TEXT        NOT NULL DEFAULT '',
    message_count INT        NOT NULL DEFAULT 0,
    last_provider TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 days'
);

CREATE INDEX IF NOT EXISTS idx_chat_sessions_user ON chat_sessions (user_id, updated_at DESC);

-- ─── chat_messages ────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS chat_messages (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID        NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT        NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content    TEXT        NOT NULL,
    provider   TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages (session_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_chat_messages_user    ON chat_messages (user_id, created_at DESC);

-- ─── chat_session_summaries (rolling compression — spec §9.5) ────────────────

CREATE TABLE IF NOT EXISTS chat_session_summaries (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id    UUID        REFERENCES chat_sessions(id) ON DELETE SET NULL,
    summary_text  TEXT        NOT NULL,
    message_range INT4RANGE,          -- which message IDs were summarised
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_summaries_user ON chat_session_summaries (user_id, created_at DESC);

-- ─── Updated-at triggers ──────────────────────────────────────────────────────

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_studio_tools_updated_at') THEN
        CREATE TRIGGER trg_studio_tools_updated_at
            BEFORE UPDATE ON studio_tools
            FOR EACH ROW EXECUTE FUNCTION update_updated_at();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_chat_sessions_updated_at') THEN
        CREATE TRIGGER trg_chat_sessions_updated_at
            BEFORE UPDATE ON chat_sessions
            FOR EACH ROW EXECUTE FUNCTION update_updated_at();
    END IF;
END;
$$;

COMMIT;
