-- 005_regional_wars.sql
-- Purpose: Support for regional tournaments and multipliers.

-- 1. Region Definitions & Multipliers
-- regional_settings was partially created in migration 001.
-- We add the missing columns here using ALTER TABLE ... ADD COLUMN IF NOT EXISTS.
ALTER TABLE regional_settings ADD COLUMN IF NOT EXISTS region_name TEXT NOT NULL DEFAULT '';
ALTER TABLE regional_settings ADD COLUMN IF NOT EXISTS base_multiplier NUMERIC DEFAULT 1.0;
ALTER TABLE regional_settings ADD COLUMN IF NOT EXISTS golden_hour_multiplier NUMERIC DEFAULT 2.0;

-- 2. Regional Leaderboard (Aggregated real-time)
CREATE TABLE IF NOT EXISTS regional_stats (
    region_code TEXT PRIMARY KEY REFERENCES regional_settings(region_code),
    total_recharge_kobo BIGINT DEFAULT 0,
    active_subscribers INTEGER DEFAULT 0,
    last_recharge_at TIMESTAMPTZ,
    rank INTEGER DEFAULT 0,
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- Seed initial Nigerian regions
INSERT INTO regional_settings (region_code, region_name) VALUES
('LAG', 'Lagos'),
('ABJ', 'Abuja'),
('KAN', 'Kano'),
('PHC', 'Port Harcourt'),
('IBD', 'Ibadan'),
('ENU', 'Enugu')
ON CONFLICT (region_code) DO NOTHING;

-- 3. Region Tournament History
CREATE TABLE IF NOT EXISTS region_tournaments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    winning_region_code TEXT REFERENCES regional_settings(region_code),
    status TEXT DEFAULT 'active' -- active, completed
);
