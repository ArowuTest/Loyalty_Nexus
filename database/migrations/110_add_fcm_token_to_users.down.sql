-- Migration 110 rollback
ALTER TABLE users DROP COLUMN IF EXISTS fcm_token;
DROP INDEX IF EXISTS idx_users_fcm_token;
