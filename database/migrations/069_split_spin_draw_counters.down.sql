-- Migration 069 DOWN: Restore shared spin_draw_counter from separate counters

-- Restore the old shared key
INSERT INTO network_configs (key, value, description) VALUES
    ('spin_draw_naira_per_credit', '200', 'Naira per spin credit and draw entry awarded on recharge')
ON CONFLICT (key) DO UPDATE SET
    value       = EXCLUDED.value,
    description = EXCLUDED.description;

-- Remove the new separate keys
DELETE FROM network_configs WHERE key IN ('spin_naira_per_credit', 'draw_naira_per_entry', 'spin_max_per_day');

-- Restore spin_draw_counter from draw_counter (best approximation)
UPDATE wallets SET spin_draw_counter = draw_counter;

-- Drop the new columns
ALTER TABLE wallets
    DROP COLUMN IF EXISTS spin_counter,
    DROP COLUMN IF EXISTS draw_counter,
    DROP COLUMN IF EXISTS daily_recharge_kobo,
    DROP COLUMN IF EXISTS daily_recharge_date,
    DROP COLUMN IF EXISTS daily_spins_awarded;
