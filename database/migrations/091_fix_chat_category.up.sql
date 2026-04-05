-- Migration 091: Restore Chat tools + fix Chat category
--
-- Problem 1 (introduced by migration 090):
--   ask-nexus and nexus-chat were correctly set to ui_template='chat' so the
--   frontend routes them to the dedicated Chat tab. However, the frontend's
--   HIDDEN_ALIAS_SLUGS then removed them from the visible tools grid, leaving
--   category='Chat' with zero visible tools — the Chat filter pill disappeared.
--
-- Problem 2:
--   web-search-ai and code-helper are missing from the production DB.
--   They were seeded in migration 029 but that migration ran before 067 on
--   this DB instance, and 067 deleted non-canonical tools. These two tools
--   are semantically Chat tools — they open the Chat tab when clicked.
--
-- Fix:
--   1. Insert web-search-ai and code-helper into studio_tools with category='Chat'
--      and ui_template='chat' so the Chat filter pill remains visible.
--   2. Confirm ask-nexus and nexus-chat stay in category='Chat'.

-- ── 1. Expand category constraint to include any new values (idempotent) ─────
ALTER TABLE studio_tools
    DROP CONSTRAINT IF EXISTS studio_tools_category_check;

ALTER TABLE studio_tools
    ADD CONSTRAINT studio_tools_category_check
        CHECK (category IN ('Chat', 'Create', 'Learn', 'Build', 'Vision'));

-- ── 2. Insert web-search-ai and code-helper (upsert — safe if already exists) ─
INSERT INTO studio_tools (
    id, name, slug, description, category,
    point_cost, provider, provider_tool,
    is_active, is_free, icon, sort_order, entry_point_cost,
    ui_template, ui_config,
    created_at, updated_at
) VALUES
(
    gen_random_uuid(),
    'Web Search AI',   'web-search-ai',
    'Ask anything — get answers with live internet data. Current news, prices, research.',
    'Chat',  0, 'pollinations', 'gemini-search',
    true, true, '🔍', 15, 0,
    'chat', '{"chat_mode":"search","mode_label":"Web Search","mode_description":"Live internet answers with sources","mode_icon":"🌐"}',
    NOW(), NOW()
),
(
    gen_random_uuid(),
    'Code Helper',     'code-helper',
    'Write, explain, and debug code in any programming language with AI.',
    'Chat',  0, 'pollinations', 'qwen-coder',
    true, true, '💻', 16, 0,
    'chat', '{"chat_mode":"code","mode_label":"Code Helper","mode_description":"Write, debug, and explain code in any language","mode_icon":"💻"}',
    NOW(), NOW()
)
ON CONFLICT (slug) DO UPDATE SET
    category   = 'Chat',
    ui_template = 'chat',
    is_active  = true,
    updated_at = NOW();

-- ── 3. Confirm ask-nexus and nexus-chat are in Chat category (idempotent) ────
UPDATE studio_tools
SET
    category   = 'Chat',
    updated_at = NOW()
WHERE slug IN ('ask-nexus', 'nexus-chat', 'ai-chat');
