-- Rollback 128: remove the Talking Avatar tool + its avatar provider rows
DELETE FROM ai_provider_configs WHERE slug IN ('fal-avatar-text', 'fal-avatar-fabric', 'heygen');
DELETE FROM studio_tools WHERE slug = 'talking-avatar';
