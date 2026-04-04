-- Migration 091: Fix Chat category tools
--
-- Problem (introduced by migration 090):
--   ask-nexus and nexus-chat were correctly set to ui_template='chat' so the
--   frontend routes them to the dedicated Chat tab. However, this caused the
--   frontend to hide them via HIDDEN_ALIAS_SLUGS — and since those were the
--   only tools with category='Chat', the "Chat" filter pill disappeared from
--   the Tools grid entirely.
--
-- Root cause: web-search-ai and code-helper belong semantically in the Chat
--   category (they open the Chat tab when clicked), but were stored under
--   category='Create' from earlier migrations.
--
-- Fix:
--   1. Move web-search-ai and code-helper to category='Chat' so the Chat
--      filter pill remains visible in the Tools grid even after ask-nexus /
--      nexus-chat are hidden.
--   2. Ensure ask-nexus and nexus-chat keep category='Chat' (already correct,
--      just confirming idempotently).
--   3. Set ui_template='chat' on web-search-ai and code-helper so the frontend
--      knows to open the Chat tab (search mode / code mode respectively) when
--      a user clicks them from the grid.

UPDATE studio_tools
SET
    category    = 'Chat',
    ui_template = 'chat',
    updated_at  = NOW()
WHERE slug IN ('web-search-ai', 'code-helper');

-- Confirm chat tools keep their correct category (idempotent)
UPDATE studio_tools
SET
    category   = 'Chat',
    updated_at = NOW()
WHERE slug IN ('ask-nexus', 'nexus-chat', 'ai-chat');
