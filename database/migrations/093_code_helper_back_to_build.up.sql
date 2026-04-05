-- Migration 093: Move code-helper back to Build category
-- code-helper uses the ToolDrawer (KnowledgeDoc template) and belongs
-- in Build alongside other productivity tools.
-- Only ask-nexus, nexus-chat, and web-search-ai are Chat category.

UPDATE studio_tools
SET
    category   = 'Build',
    ui_template = 'knowledge-doc',
    updated_at = NOW()
WHERE slug = 'code-helper';
