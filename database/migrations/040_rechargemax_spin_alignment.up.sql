-- Add new fields to prize_pool table
ALTER TABLE prize_pool ADD COLUMN is_no_win BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE prize_pool ADD COLUMN no_win_message TEXT;
ALTER TABLE prize_pool ADD COLUMN color_scheme TEXT;
ALTER TABLE prize_pool ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;
ALTER TABLE prize_pool ADD COLUMN minimum_recharge BIGINT;

-- Create spin_tiers table
CREATE TABLE IF NOT EXISTS spin_tiers (
    id UUID PRIMARY KEY,
    tier_name TEXT NOT NULL,
    tier_display_name TEXT NOT NULL,
    min_daily_amount BIGINT NOT NULL,
    max_daily_amount BIGINT NOT NULL,
    spins_per_day INTEGER NOT NULL,
    tier_color TEXT,
    tier_icon TEXT,
    tier_badge TEXT,
    description TEXT,
    sort_order INTEGER NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Seed default spin tiers
INSERT INTO spin_tiers (id, tier_name, tier_display_name, min_daily_amount, max_daily_amount, spins_per_day, sort_order) VALUES
('11111111-1111-1111-1111-111111111111', 'bronze', 'Bronze', 100000, 499999, 1, 1),
('22222222-2222-2222-2222-222222222222', 'silver', 'Silver', 500000, 999999, 2, 2),
('33333333-3333-3333-3333-333333333333', 'gold', 'Gold', 1000000, 1999999, 3, 3),
('44444444-4444-4444-4444-444444444444', 'platinum', 'Platinum', 2000000, 999999999999, 5, 4)
ON CONFLICT (id) DO NOTHING;
