-- 106_platform_settings.down.sql
DROP TABLE IF EXISTS platform_settings;
DROP TABLE IF EXISTS asset_expiry_notifications;
ALTER TABLE ai_generations DROP COLUMN IF EXISTS expired_cleaned_at;
DROP INDEX IF EXISTS idx_ai_gen_expires_at;
