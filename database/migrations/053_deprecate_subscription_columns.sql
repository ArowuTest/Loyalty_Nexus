-- Migration 053: Deprecate subscription billing columns
-- 
-- Loyalty Nexus does NOT use paid subscriptions. Users earn rewards through
-- airtime/data recharges. The subscription_* columns on the users table are
-- kept for backwards-compatibility with existing rows and will be dropped in
-- a future migration once all rows have been back-filled.
--
-- This migration:
--   1. Adds a comment to each deprecated column so the intent is clear in the schema.
--   2. Back-fills all existing users to subscription_tier='free', subscription_status='active'.
--   3. Does NOT drop the columns (safe for zero-downtime deploy).
--
-- To fully remove these columns in a future release, run:
--   ALTER TABLE users DROP COLUMN subscription_tier;
--   ALTER TABLE users DROP COLUMN subscription_status;
--   ALTER TABLE users DROP COLUMN subscription_expires_at;

-- Back-fill existing rows so they have consistent values
UPDATE users
SET
    subscription_tier   = 'free',
    subscription_status = 'active',
    subscription_expires_at = NULL
WHERE subscription_tier IS NULL
   OR subscription_tier = ''
   OR subscription_status IS NULL
   OR subscription_status = '';

-- Add column comments so the deprecation is visible in pg_catalog
COMMENT ON COLUMN users.subscription_tier       IS 'DEPRECATED: subscription billing removed. Always ''free''. Will be dropped in a future migration.';
COMMENT ON COLUMN users.subscription_status     IS 'DEPRECATED: subscription billing removed. Always ''active''. Will be dropped in a future migration.';
COMMENT ON COLUMN users.subscription_expires_at IS 'DEPRECATED: subscription billing removed. Always NULL. Will be dropped in a future migration.';
