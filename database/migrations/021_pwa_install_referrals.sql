-- 021_pwa_install_referrals.sql
-- Purpose: Support for PWA installation tracking and referral bonuses (REQ-1.4, REQ-5.2.10).

-- 1. Track PWA Installation status
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS pwa_installed BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS referred_by_id UUID REFERENCES users(id);

-- 2. Referral Tracking for multi-recharge validation
CREATE TABLE IF NOT EXISTS referral_tracking (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referrer_id UUID NOT NULL REFERENCES users(id),
    referred_user_id UUID NOT NULL REFERENCES users(id),
    status TEXT CHECK (status IN ('pending', 'completed')) DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT now()
);
