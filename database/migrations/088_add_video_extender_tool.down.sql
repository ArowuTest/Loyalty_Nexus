-- Rollback migration 088: Remove Video Extender tool
DELETE FROM studio_tools WHERE slug = 'video-extend';
