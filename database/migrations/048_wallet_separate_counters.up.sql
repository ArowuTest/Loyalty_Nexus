-- Migration 048: Separate recharge accumulators for spin/draw vs pulse points
--
-- The original recharge_counter was used for both spin credits and pulse points,
-- which is incorrect. Loyalty Nexus has two independent reward currencies:
--
--   1. Spin Credits + Draw Entries  — awarded every ₦200 recharge
--   2. Pulse Points                 — awarded every ₦250 recharge (AI Studio currency)
--
-- We add two dedicated counters so each accumulator tracks its own remainder
-- independently. The old recharge_counter column is kept for backwards
-- compatibility but will no longer be written by the MTN push pipeline.
--
-- Admin-configurable thresholds (network_configs):
--   spin_draw_naira_per_credit   — naira per spin credit + draw entry (default 200)
--   pulse_naira_per_point        — naira per pulse point (default 250)
--   mtn_push_min_amount_naira    — minimum qualifying recharge (default 50)

ALTER TABLE wallets
    ADD COLUMN IF NOT EXISTS spin_draw_counter BIGINT NOT NULL DEFAULT 0
        CHECK (spin_draw_counter >= 0),
    ADD COLUMN IF NOT EXISTS pulse_counter     BIGINT NOT NULL DEFAULT 0
        CHECK (pulse_counter >= 0);

COMMENT ON COLUMN wallets.spin_draw_counter IS
    'Kobo remainder accumulator for spin credits and draw entries (resets modulo spin_draw_naira_per_credit×100)';
COMMENT ON COLUMN wallets.pulse_counter IS
    'Kobo remainder accumulator for Pulse Points (resets modulo pulse_naira_per_point×100)';

-- Seed the three configurable thresholds.
-- ON CONFLICT DO NOTHING so re-running the migration is safe.
INSERT INTO network_configs (key, value, description) VALUES
    ('spin_draw_naira_per_credit', '200',  'Naira per spin credit and draw entry awarded on recharge'),
    ('pulse_naira_per_point',      '250',  'Naira per Pulse Point awarded on recharge (AI Studio currency)'),
    ('mtn_push_min_amount_naira',  '50',   'Minimum recharge amount in naira to qualify for rewards')
ON CONFLICT (key) DO NOTHING;
