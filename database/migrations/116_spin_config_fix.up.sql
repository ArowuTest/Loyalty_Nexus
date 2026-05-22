-- Migration 116: Remove duplicate contradictory spin limit config key
-- Spin limits are enforced from spin_tiers via GetSpinTierFromDB / SpinsPerDay.
-- Keep spin_max_per_day if it exists for backward-compatible admin visibility;
-- remove the contradictory per-user variant.

DELETE FROM network_configs
WHERE key = 'spin_max_per_user_per_day';
