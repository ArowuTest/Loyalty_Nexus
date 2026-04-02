-- Rollback migration 082
ALTER TABLE ai_generations DROP COLUMN IF EXISTS output_url_2;
