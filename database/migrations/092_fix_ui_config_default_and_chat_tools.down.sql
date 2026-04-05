-- Rollback 092
ALTER TABLE studio_tools ALTER COLUMN ui_config DROP DEFAULT;
DELETE FROM studio_tools WHERE slug IN ('web-search-ai', 'code-helper');
