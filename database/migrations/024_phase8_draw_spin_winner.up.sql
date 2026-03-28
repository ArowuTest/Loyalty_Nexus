-- Migration 024: Phase 8 — Draw Engine + Spin Prize Pool alignment
-- Adds: full draw engine columns, prize_fulfillment_logs, winner_service tables
-- Aligns prize_pool with PrizePoolEntry entity
-- All financial rules enforced at DB level (two-pool separation, immutable tx ledger)

-- ─── Draw Engine — full schema alignment ───────────────────────────────────

-- Add missing columns to draws table
ALTER TABLE draws ADD COLUMN IF NOT EXISTS draw_type       TEXT NOT NULL DEFAULT 'MONTHLY';
ALTER TABLE draws ADD COLUMN IF NOT EXISTS recurrence      TEXT NOT NULL DEFAULT 'none';
ALTER TABLE draws ADD COLUMN IF NOT EXISTS next_draw_at    TIMESTAMPTZ;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS start_time      TIMESTAMPTZ;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS end_time        TIMESTAMPTZ;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS draw_time       TIMESTAMPTZ;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS prize_pool      NUMERIC(12,2) NOT NULL DEFAULT 0;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS winner_count    INTEGER NOT NULL DEFAULT 1;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS runner_ups_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS total_entries   INTEGER NOT NULL DEFAULT 0;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS total_winners   INTEGER NOT NULL DEFAULT 0;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS executed_at     TIMESTAMPTZ;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS completed_at    TIMESTAMPTZ;

-- draw_entries: full schema
CREATE TABLE IF NOT EXISTS draw_entries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    draw_id      UUID NOT NULL REFERENCES draws(id) ON DELETE CASCADE,
    user_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    phone_number TEXT NOT NULL,
    entry_source TEXT NOT NULL DEFAULT 'recharge',  -- recharge | subscription | bonus | csv_import
    amount       BIGINT NOT NULL DEFAULT 0,           -- kobo
    ticket_count INTEGER NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_draw_entries_draw_id ON draw_entries(draw_id);
CREATE INDEX IF NOT EXISTS idx_draw_entries_user_id ON draw_entries(user_id);

-- draw_winners: full schema
CREATE TABLE IF NOT EXISTS draw_winners (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    draw_id      UUID NOT NULL REFERENCES draws(id) ON DELETE CASCADE,
    user_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    phone_number TEXT NOT NULL,
    position     INTEGER NOT NULL,
    prize_type   TEXT NOT NULL DEFAULT 'CASH',
    prize_value_kobo BIGINT NOT NULL DEFAULT 0,
    is_runner_up BOOLEAN NOT NULL DEFAULT FALSE,
    status       TEXT NOT NULL DEFAULT 'PENDING_FULFILLMENT',  -- PENDING_FULFILLMENT | FULFILLED | EXPIRED | FAILED
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_draw_winners_draw_id ON draw_winners(draw_id);
CREATE INDEX IF NOT EXISTS idx_draw_winners_user_id ON draw_winners(user_id);

-- ─── Prize Pool — align with PrizePoolEntry entity ──────────────────────────

-- Ensure prize_pool table has all required columns
CREATE TABLE IF NOT EXISTS prize_pool (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    TEXT NOT NULL,
    prize_type              TEXT NOT NULL,   -- try_again | pulse_points | airtime | data_bundle | momo_cash
    base_value              NUMERIC(12,2) NOT NULL DEFAULT 0,
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    win_probability_weight  INTEGER NOT NULL DEFAULT 0,  -- weights must sum to ≤10000 (=100.00%)
    daily_inventory_cap     INTEGER,         -- NULL = unlimited
    created_at              TIMESTAMPTZ DEFAULT NOW(),
    updated_at              TIMESTAMPTZ DEFAULT NOW()
);

-- Seed default 12-slot prize table from spec Appendix A
-- Only insert if table is empty (idempotent)
INSERT INTO prize_pool (name, prize_type, base_value, is_active, win_probability_weight)
SELECT * FROM (VALUES
    ('Try Again',      'try_again',    0,    true, 2000),
    ('Try Again',      'try_again',    0,    true, 1500),
    ('Try Again',      'try_again',    0,    true, 1000),
    ('Try Again',      'try_again',    0,    true, 830),
    ('Try Again',      'try_again',    0,    true, 500),
    ('+5 Pulse Points','pulse_points', 5,   true, 830),
    ('+10 Pulse Points','pulse_points',10,  true, 830),
    ('10MB Data',      'data_bundle',  10,   true, 830),
    ('25MB Data',      'data_bundle',  25,   true, 830),
    ('₦50 Airtime',   'airtime',      50,   true, 420),
    ('₦100 Airtime',  'airtime',      100,  true, 250),
    ('₦200 Airtime',  'airtime',      200,  true, 180)
) AS vals(name, prize_type, base_value, is_active, win_probability_weight)
WHERE NOT EXISTS (SELECT 1 FROM prize_pool LIMIT 1);

-- ─── Prize Fulfillment Logs ─────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS prize_fulfillment_logs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    spin_result_id   UUID NOT NULL REFERENCES spin_results(id) ON DELETE CASCADE,
    attempt_number   INTEGER NOT NULL DEFAULT 1,
    fulfillment_mode TEXT NOT NULL DEFAULT 'AUTO',      -- AUTO | MANUAL
    status           TEXT NOT NULL,                      -- SUCCESS | FAILED | PENDING
    provider         TEXT,                               -- vtpass | momo | manual
    provider_ref     TEXT,
    error_message    TEXT,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_pfl_spin_result ON prize_fulfillment_logs(spin_result_id);

-- Add momo_number + retry_count to spin_results if missing
ALTER TABLE spin_results ADD COLUMN IF NOT EXISTS mo_mo_number TEXT NOT NULL DEFAULT '';
ALTER TABLE spin_results ADD COLUMN IF NOT EXISTS retry_count  INTEGER NOT NULL DEFAULT 0;
ALTER TABLE spin_results ADD COLUMN IF NOT EXISTS fulfilled_at TIMESTAMPTZ;

-- ─── Admin: notification_broadcasts aligned ─────────────────────────────────

CREATE TABLE IF NOT EXISTS notification_broadcasts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title        TEXT NOT NULL,
    message      TEXT NOT NULL,
    type         TEXT NOT NULL DEFAULT 'push',   -- push | sms | both
    target_count INTEGER NOT NULL DEFAULT 0,
    status       TEXT NOT NULL DEFAULT 'queued', -- queued | sent | failed
    sent_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

-- ─── Network Config: seed new AI studio keys (idempotent) ──────────────────

INSERT INTO network_configs (key, value, description) VALUES
    ('ai_chat_enabled',               'true',  'Enable Ask Nexus chat feature'),
    ('nexus_chat_daily_limit',        '20',    'Max messages per user per day for Ask Nexus'),
    ('ai_translate_cost_points',      '1',     'Points cost for translate tool'),
    ('ai_quiz_cost_points',           '2',     'Points cost for quiz tool'),
    ('ai_mindmap_cost_points',        '2',     'Points cost for mind map tool'),
    ('ai_narrate_cost_points',        '2',     'Points cost for narrate text tool'),
    ('ai_transcribe_cost_points',     '2',     'Points cost for transcribe voice tool'),
    ('ai_bgremover_cost_points',      '3',     'Points cost for background remover'),
    ('ai_podcast_cost_points',        '4',     'Points cost for podcast tool'),
    ('ai_slidedeck_cost_points',      '4',     'Points cost for slide deck tool'),
    ('ai_infographic_cost_points',    '10',    'Points cost for infographic tool'),
    ('ai_research_brief_cost_points', '5',     'Points cost for deep research brief'),
    ('ai_bgmusic_cost_points',        '5',     'Points cost for background music'),
    ('ai_bizplan_cost_points',        '12',    'Points cost for business plan'),
    ('ai_video_basic_cost_points',    '65',    'Points cost for basic video animation'),
    ('ai_jingle_cost_points',         '200',   'Points cost for marketing jingle (30s)'),
    ('ai_video_premium_cost_points',  '250',   'Points cost for premium video animation'),
    ('daily_prize_liability_cap_naira', '500000', 'Max prize payout per day in naira'),
    ('spin_max_per_user_per_day',     '3',     'Max spins per user per day'),
    ('spin_trigger_naira',            '1000',  'Naira recharge required per spin credit')
ON CONFLICT (key) DO NOTHING;
