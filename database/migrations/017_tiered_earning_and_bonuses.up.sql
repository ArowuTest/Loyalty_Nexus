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
INSERT INTO recharge_tiers (name, min_amount_kobo, points_per_naira)
SELECT v.name, v.min_amount_kobo, v.points_per_naira
FROM (VALUES
('Standard', 0::BIGINT, 0.004::NUMERIC), -- 1/250
('Silver', 100000::BIGINT, 0.005::NUMERIC), -- 1/200 (N1000+)
('Gold', 300000::BIGINT, 0.00667::NUMERIC)
) AS v(name, min_amount_kobo, points_per_naira)
WHERE NOT EXISTS (SELECT 1 FROM recharge_tiers r WHERE r.name = v.name); -- 1/150 (N3000+)

-- Seed Initial Bonuses
INSERT INTO program_bonuses (event_type, threshold, bonus_points)
SELECT v.event_type, v.threshold, v.bonus_points
FROM (VALUES
('first_recharge', NULL::INTEGER, 20::BIGINT),
('streak_milestone', 7::INTEGER, 10::BIGINT),
('streak_milestone', 14::INTEGER, 25::BIGINT),
('streak_milestone', 30::INTEGER, 50::BIGINT)
) AS v(event_type, threshold, bonus_points)
WHERE NOT EXISTS (
  SELECT 1 FROM program_bonuses b
  WHERE b.event_type = v.event_type AND b.threshold IS NOT DISTINCT FROM v.threshold
);
