-- Rollback migration 087: Remove Video Editor tool
DELETE FROM studio_tools WHERE slug = 'video-edit';
