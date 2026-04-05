-- Migration 092: Fix studio_tools.ui_config NOT NULL default
-- Ensures the column has a proper default so inserts without ui_config succeed.
-- Also inserts web-search-ai and code-helper as Chat tools via raw SQL
-- (bypassing the Go handler's ui_config assignment issue).

-- Ensure column has a default
ALTER TABLE studio_tools
    ALTER COLUMN ui_config SET DEFAULT '{}';

-- Insert web-search-ai and code-helper as Chat tools (upsert by slug)
INSERT INTO studio_tools (
    id, name, slug, description, category,
    point_cost, provider, provider_tool,
    is_active, is_free, icon, sort_order,
    entry_point_cost, ui_template, ui_config,
    created_at, updated_at
) VALUES
(
    gen_random_uuid(),
    'Web Search AI', 'web-search-ai',
    'Ask anything — get live internet answers with sources. Current news, prices, research.',
    'Chat', 0, 'pollinations', 'gemini-search',
    true, true, '🔍', 15,
    0, 'chat', '{"chat_mode":"search"}',
    NOW(), NOW()
),
(
    gen_random_uuid(),
    'Code Helper', 'code-helper',
    'Write, explain, and debug code in any programming language with AI.',
    'Chat', 0, 'pollinations', 'qwen-coder',
    true, true, '💻', 16,
    0, 'chat', '{"chat_mode":"code"}',
    NOW(), NOW()
)
ON CONFLICT (slug) DO UPDATE SET
    category    = 'Chat',
    ui_template = 'chat',
    is_active   = true,
    updated_at  = NOW();
