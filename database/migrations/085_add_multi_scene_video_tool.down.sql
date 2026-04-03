-- Rollback migration 085: Remove Video Story Builder tool
DELETE FROM studio_tools WHERE slug = 'video-story';
