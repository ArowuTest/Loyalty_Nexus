-- Rollback 091: restore web-search-ai and code-helper to original category
UPDATE studio_tools
SET
    category    = 'Create',
    ui_template = 'knowledge-doc',
    updated_at  = NOW()
WHERE slug IN ('web-search-ai', 'code-helper');
