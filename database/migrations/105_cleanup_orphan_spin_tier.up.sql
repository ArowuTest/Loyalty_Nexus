-- Migration 105: Remove orphan spin tier with empty name / zero max_daily_amount
DELETE FROM spin_tiers WHERE tier_name = '' OR (max_daily_amount = 0 AND tier_name NOT LIKE '%plat%');
