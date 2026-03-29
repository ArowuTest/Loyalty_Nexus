-- 017_tiered_earning_and_bonuses.sql
-- Purpose: Support for dynamic recharge tiers and milestone bonuses (REQ-5.2.3, REQ-5.2.8, REQ-5.2.9).

-- 1. Recharge Amount Tiers
CREATE TABLE IF NOT EXISTS recharge_tiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL, -- Standard, Silver, Gold
    min_amount_kobo BIGINT NOT NULL,
    points_per_naira NUMERIC NOT NULL, -- e.g. 1 pt per N250 -> rate = 1/250
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- 2. Streak & Milestone Bonuses
CREATE TABLE IF NOT EXISTS program_bonuses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT CHECK (event_type IN ('first_recharge', 'streak_milestone', 'referral_completion')),
    threshold INTEGER, -- days for streak, or null
    bonus_points BIGINT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Seed Initial Tiers
INSERT INTO recharge_tiers (name, min_amount_kobo, points_per_naira) VALUES
('Standard', 0, 0.004), -- 1/250
('Silver', 100000, 0.005), -- 1/200 (N1000+)
('Gold', 300000, 0.00667)
ON CONFLICT (name) DO NOTHING; -- 1/150 (N3000+)

-- Seed Initial Bonuses
INSERT INTO program_bonuses (event_type, threshold, bonus_points) VALUES
('first_recharge', NULL, 20),
('streak_milestone', 7, 10),
('streak_milestone', 14, 25),
('streak_milestone', 30, 50)
ON CONFLICT (event_type) DO NOTHING;
