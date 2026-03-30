-- Migration 068 down: Restore referral columns (rollback)
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS referral_code    TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS referred_by      UUID,
  ADD COLUMN IF NOT EXISTS total_referrals  INT  NOT NULL DEFAULT 0;
