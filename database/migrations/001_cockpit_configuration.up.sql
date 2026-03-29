-- 001_cockpit_configuration.sql
-- Purpose: Total flexibility for the private firm to manage Loyalty Nexus.

-- 1. Global Program Rules (The "Knobs")
CREATE TABLE program_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key TEXT UNIQUE NOT NULL, -- e.g., 'min_recharge_spin', 'streak_window_hours'
    config_value JSONB NOT NULL,
    description TEXT,
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- 2. Prize Inventory & Weights (The "Odds Engine")
CREATE TABLE prize_pool (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    prize_type TEXT CHECK (prize_type IN ('airtime', 'data', 'momo_cash', 'studio_credits')),
    base_value NUMERIC NOT NULL,
    is_active BOOLEAN DEFAULT true,
    win_probability_weight INTEGER DEFAULT 100, -- Higher = more common
    daily_inventory_cap INTEGER, -- Max wins per day
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- 3. Regional Multipliers (The "Tournament" Engine)
CREATE TABLE regional_settings (
    region_code TEXT PRIMARY KEY, -- e.g., 'LAG', 'ABJ', 'KAN'
    multiplier NUMERIC DEFAULT 1.0,
    is_golden_hour BOOLEAN DEFAULT false,
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- 4. Studio Parameters (AI Rendering Limits)
CREATE TABLE studio_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    media_type TEXT CHECK (media_type IN ('image', 'video', 'jingle')),
    point_cost INTEGER NOT NULL,
    render_priority INTEGER DEFAULT 1,
    is_enabled BOOLEAN DEFAULT true
);

-- Insert initial "Cockpit" data
INSERT INTO program_configs (config_key, config_value, description) VALUES
('min_recharge_naira', '500', 'Minimum recharge to earn a spin'),
('streak_target_days', '7', 'Days required for a Mega Jackpot ticket'),
('ghost_nudge_hours', '48', 'Inactivity hours before lock-screen nudge fires')
ON CONFLICT (config_key) DO NOTHING;

