-- 020_streak_grace_and_expiry.sql
-- Purpose: Implement streak grace days and points expiry policy (REQ-5.2.13, REQ-5.2.14).

-- 1. Track Grace Days used per month
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS streak_freeze_grace_used INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS points_expiry_date TIMESTAMPTZ;

-- 2. Audit table for point expiry notifications
CREATE TABLE IF NOT EXISTS points_expiry_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    msisdn TEXT NOT NULL,
    points_amount BIGINT NOT NULL,
    expiry_date TIMESTAMPTZ NOT NULL,
    notified_at TIMESTAMPTZ DEFAULT now()
);
