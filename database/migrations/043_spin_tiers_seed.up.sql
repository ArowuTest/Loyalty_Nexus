-- Migration 043: Seed default spin tiers (matching RechargeMax defaults)
-- Amounts are in kobo (₦1 = 100 kobo)
-- Tier ranges must be contiguous with no gaps or overlaps

INSERT INTO spin_tiers (id, tier_name, tier_display_name, min_daily_amount, max_daily_amount, spins_per_day, tier_color, tier_icon, tier_badge, description, sort_order, is_active, created_at, updated_at)
VALUES
    (gen_random_uuid(), 'bronze',   'Bronze',   100000,   499999,  1, '#CD7F32', 'bronze-medal',   'BRONZE',   'Recharge ₦1,000–₦4,999 today for 1 spin',   1, TRUE, NOW(), NOW()),
    (gen_random_uuid(), 'silver',   'Silver',   500000,   999999,  2, '#C0C0C0', 'silver-medal',   'SILVER',   'Recharge ₦5,000–₦9,999 today for 2 spins',  2, TRUE, NOW(), NOW()),
    (gen_random_uuid(), 'gold',     'Gold',    1000000,  2999999,  3, '#FFD700', 'gold-medal',     'GOLD',     'Recharge ₦10,000–₦29,999 today for 3 spins', 3, TRUE, NOW(), NOW()),
    (gen_random_uuid(), 'platinum', 'Platinum', 3000000, 999999999, 5, '#E5E4E2', 'platinum-medal', 'PLATINUM', 'Recharge ₦30,000+ today for 5 spins',        4, TRUE, NOW(), NOW())
ON CONFLICT (tier_name) DO NOTHING;
