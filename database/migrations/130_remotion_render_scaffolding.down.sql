-- Rollback 130: remove the Remotion scaffolding tool + provider row
DELETE FROM ai_provider_configs WHERE slug = 'remotion-selfhosted';
DELETE FROM studio_tools WHERE slug = 'video-slideshow';
