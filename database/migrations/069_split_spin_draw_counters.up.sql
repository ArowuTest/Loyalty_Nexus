-- Migration 069: Split spin_draw_counter into separate spin_counter and draw_counter
--               + Add daily recharge tracking for tier-based spin credit logic
--
-- BACKGROUND
-- ----------
-- Migration 048 added a single `spin_draw_counter` that was shared between
-- spin credits AND draw entries (both used ₦200 threshold).
--
-- The correct business logic is:
--   • Spin Credits  — Tier-based on CUMULATIVE DAILY recharge:
--                     ₦1,000–₦4,999/day  → 1 spin  (Bronze)
--                     ₦5,000–₦9,999/day  → 2 spins (Silver)
--                     ₦10,000–₦19,999/day → 3 spins (Gold)
--                     ₦20,000+/day        → 5 spins (Platinum)
--                     The tier's spins_per_day is the DAILY CAP, not additive.
--                     Each time the cumulative daily total crosses a tier boundary,
--                     the user is awarded the DIFFERENCE (new_cap - already_awarded).
--   • Draw Entries  — ₦200 per entry, simple accumulator per transaction
--   • Pulse Points  — ₦250 per point (unchanged)
--
-- These are completely independent currencies with different thresholds and
-- different accumulation rules, so they need separate counters.
--
-- CHANGES
-- -------
--   1. Add spin_counter   BIGINT — kobo remainder for spin credit accumulation (NOT USED for tier logic)
--   2. Add draw_counter   BIGINT — kobo remainder for draw entry accumulation
--   3. Add daily_recharge_kobo BIGINT — cumulative recharge today (resets at midnight WAT)
--   4. Add daily_recharge_date DATE   — the date daily_recharge_kobo was last reset
--   5. Add daily_spins_awarded INT    — spins already awarded today (prevents double-awarding on tier upgrade)
--   6. Migrate existing spin_draw_counter → draw_counter (only if spin_draw_counter exists)
--   7. Zero out spin_draw_counter if it exists (deprecated)
--   8. Update network_configs with correct separated thresholds
--
-- NOTE: Steps 6 and 7 are wrapped in a DO $$ block that checks whether
-- spin_draw_counter exists before referencing it. This makes the migration
-- safe on databases that were provisioned without migration 048 (e.g. a fresh
-- Render deploy where the wallets table was created by migration 060 which
-- does not include spin_draw_counter).

-- Step 1: Add the new counter and daily tracking columns
ALTER TABLE wallets
    ADD COLUMN IF NOT EXISTS spin_counter         BIGINT  NOT NULL DEFAULT 0 CHECK (spin_counter >= 0),
    ADD COLUMN IF NOT EXISTS draw_counter         BIGINT  NOT NULL DEFAULT 0 CHECK (draw_counter >= 0),
    ADD COLUMN IF NOT EXISTS daily_recharge_kobo  BIGINT  NOT NULL DEFAULT 0 CHECK (daily_recharge_kobo >= 0),
    ADD COLUMN IF NOT EXISTS daily_recharge_date  DATE    NULL,
    ADD COLUMN IF NOT EXISTS daily_spins_awarded  INTEGER NOT NULL DEFAULT 0 CHECK (daily_spins_awarded >= 0);

COMMENT ON COLUMN wallets.spin_counter IS
    'Reserved for future use. Tier-based spins use daily_recharge_kobo + spin_tiers table instead.';
COMMENT ON COLUMN wallets.draw_counter IS
    'Kobo remainder accumulator for Draw Entries (resets modulo draw_naira_per_entry×100). Threshold: ₦200.';
COMMENT ON COLUMN wallets.daily_recharge_kobo IS
    'Cumulative recharge amount in kobo for the current calendar day (WAT). Resets to 0 at midnight WAT.';
COMMENT ON COLUMN wallets.daily_recharge_date IS
    'The calendar date (WAT) for which daily_recharge_kobo and daily_spins_awarded are current. NULL = never recharged.';
COMMENT ON COLUMN wallets.daily_spins_awarded IS
    'Number of spin credits already awarded today. Used to calculate incremental spin awards when tier upgrades.';

-- Steps 2 & 3: Migrate spin_draw_counter → draw_counter, then zero it out.
-- Wrapped in a DO block so this is a no-op on DBs that never had spin_draw_counter
-- (e.g. fresh deploys where wallets was created by migration 060 without that column).
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'wallets' AND column_name = 'spin_draw_counter'
    ) THEN
        -- Step 2: Carry over any existing remainder to the new draw_counter
        UPDATE wallets
        SET draw_counter = spin_draw_counter
        WHERE spin_draw_counter > 0;

        -- Step 3: Zero out the deprecated column
        UPDATE wallets SET spin_draw_counter = 0;
    END IF;
END $$;

-- Step 4: Update network_configs — replace the old shared key with two separate keys
-- Remove the old shared key (safe even if it does not exist)
DELETE FROM network_configs WHERE key = 'spin_draw_naira_per_credit';

-- Insert the two new separate threshold keys
INSERT INTO network_configs (key, value, description) VALUES
    ('spin_naira_per_credit',  '1000', 'Minimum daily recharge in naira to qualify for spin credits (Bronze tier threshold)'),
    ('draw_naira_per_entry',   '200',  'Naira per Draw Entry awarded on recharge (simple accumulator per transaction)'),
    ('spin_max_per_day',       '5',    'Maximum spin credits a user can earn per calendar day (Platinum tier cap)')
ON CONFLICT (key) DO UPDATE SET
    value       = EXCLUDED.value,
    description = EXCLUDED.description;

-- Step 5: Fix incorrect pulse_naira_per_point seed value
-- Migration 059 seeded pulse_naira_per_point as '10' which is wrong.
-- The correct production value is ₦250 per Pulse Point.
-- This corrects that seed so the Points Engine charges the right amount.
INSERT INTO network_configs (key, value, description) VALUES
    ('pulse_naira_per_point', '250', 'Naira per Pulse Point awarded on recharge (flat accumulator, no tier multiplier)')
ON CONFLICT (key) DO UPDATE SET
    value       = '250',
    description = EXCLUDED.description;
