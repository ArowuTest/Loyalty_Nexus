-- Migration 068: Remove referral system
-- Drops referral_code, referred_by, total_referrals from users
-- Drops referral_bonus_points, referral_bonus_referee_pts from program_configs

-- Remove referral columns from users table
ALTER TABLE users
  DROP COLUMN IF EXISTS referral_code,
  DROP COLUMN IF EXISTS referred_by,
  DROP COLUMN IF EXISTS total_referrals;

-- Remove referral config keys from program_configs
-- Note: the column is config_key, not key
DELETE FROM program_configs
WHERE config_key IN ('referral_bonus_points', 'referral_bonus_referee_pts', 'REFERRAL_BONUS');

-- Drop any index on referral_code if it exists
DROP INDEX IF EXISTS idx_users_referral_code;
DROP INDEX IF EXISTS uidx_users_referral_code;
