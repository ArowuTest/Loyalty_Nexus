-- Migration 110: Add fcm_token column to users table
-- The notification service queries fcm_token for push notifications.
-- This column was missing, causing: column "fcm_token" does not exist (SQLSTATE 42703)
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS fcm_token TEXT NOT NULL DEFAULT '';

-- Index for fast FCM token lookups (used in push notification delivery)
CREATE INDEX IF NOT EXISTS idx_users_fcm_token ON users (fcm_token) WHERE fcm_token != '';
