-- 078_chat_tool_slug.up.sql
-- Purpose: Add tool_slug to chat_sessions and session_summaries so that
--          memory is scoped per AI mode (general / web-search-ai / code-helper).
--          Also adds wallet auto-creation on user registration (ARCH-03).

-- ── chat_sessions: add tool_slug column ──────────────────────────────────────
ALTER TABLE chat_sessions
    ADD COLUMN IF NOT EXISTS tool_slug TEXT NOT NULL DEFAULT 'general';

-- ── session_summaries: add tool_slug column ──────────────────────────────────
ALTER TABLE session_summaries
    ADD COLUMN IF NOT EXISTS tool_slug TEXT NOT NULL DEFAULT 'general';

-- ── Indexes for efficient per-mode memory lookup ──────────────────────────────
CREATE INDEX IF NOT EXISTS idx_chat_sessions_user_tool
    ON chat_sessions(user_id, tool_slug, status, last_activity_at);

CREATE INDEX IF NOT EXISTS idx_session_summaries_user_tool
    ON session_summaries(user_id, tool_slug, created_at);

-- ── chat_messages: add created_at index for retention cleanup ─────────────────
CREATE INDEX IF NOT EXISTS idx_chat_messages_session_created
    ON chat_messages(session_id, created_at);

-- ── wallets: ensure every existing user has a wallet row (ARCH-03) ────────────
INSERT INTO wallets (id, user_id, pulse_points, spin_credits, lifetime_points, created_at, updated_at)
SELECT
    gen_random_uuid(),
    u.id,
    0, 0, 0,
    now(), now()
FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM wallets w WHERE w.user_id = u.id
);
