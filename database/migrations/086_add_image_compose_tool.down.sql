-- Rollback migration 086: Remove image-compose tool
DELETE FROM studio_tools WHERE slug = 'image-compose';
