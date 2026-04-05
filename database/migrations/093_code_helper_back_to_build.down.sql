-- Rollback 093
UPDATE studio_tools SET category = 'Chat', ui_template = 'chat', updated_at = NOW()
WHERE slug = 'code-helper';
