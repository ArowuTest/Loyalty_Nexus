-- 012_user_momo.sql
-- Purpose: Support for MTN Mobile Money (MoMo) linking and verification.

ALTER TABLE users 
ADD COLUMN momo_number TEXT,
ADD COLUMN momo_verified BOOLEAN DEFAULT false,
ADD COLUMN momo_verified_at TIMESTAMPTZ;

CREATE INDEX idx_users_momo ON users(momo_number) WHERE momo_number IS NOT NULL;
