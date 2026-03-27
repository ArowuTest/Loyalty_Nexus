-- 045_mtn_push_pipeline.sql
-- Purpose: Support MTN-push recharge ingestion.
--   1. Extend draw_entries with entry_source + source_transaction_id
--      (mirrors RechargeMax draw_entries.source_type / source_transaction_id)
--   2. Add mtn_push_events audit table — every raw MTN push is logged here
--      before any business logic runs, so we have a full inbound audit trail.
--   3. Seed network_configs keys for the MTN push pipeline.

-- ── 1. Extend draw_entries ────────────────────────────────────────────────────
-- entry_source: who created this entry (recharge | subscription | bonus | manual)
ALTER TABLE draw_entries
    ADD COLUMN IF NOT EXISTS entry_source          TEXT NOT NULL DEFAULT 'recharge'
        CHECK (entry_source IN ('recharge','subscription','bonus','manual')),
    ADD COLUMN IF NOT EXISTS source_transaction_id UUID REFERENCES transactions(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_draw_entries_source_tx ON draw_entries(source_transaction_id);

-- ── 2. MTN push events audit table ───────────────────────────────────────────
-- Every inbound MTN push is written here atomically before any reward logic.
-- This gives us idempotency (unique constraint on transaction_ref) and a full
-- audit trail even if the downstream processing fails.
CREATE TABLE IF NOT EXISTS mtn_push_events (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_ref     TEXT        NOT NULL UNIQUE,   -- MTN's unique transaction ID
    msisdn              TEXT        NOT NULL,           -- normalised 0XXXXXXXXXX
    recharge_type       TEXT        NOT NULL DEFAULT 'AIRTIME'
                            CHECK (recharge_type IN ('AIRTIME','DATA','BUNDLE')),
    amount_kobo         BIGINT      NOT NULL CHECK (amount_kobo > 0),
    event_timestamp     TIMESTAMPTZ NOT NULL,           -- timestamp from MTN payload
    raw_payload         JSONB,                          -- full original payload
    status              TEXT        NOT NULL DEFAULT 'RECEIVED'
                            CHECK (status IN ('RECEIVED','PROCESSED','DUPLICATE','FAILED')),
    processing_error    TEXT,
    points_awarded      BIGINT      NOT NULL DEFAULT 0,
    draw_entries_created INT        NOT NULL DEFAULT 0,
    spin_credits_awarded INT        NOT NULL DEFAULT 0,
    processed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mtn_push_events_msisdn     ON mtn_push_events(msisdn);
CREATE INDEX IF NOT EXISTS idx_mtn_push_events_status     ON mtn_push_events(status);
CREATE INDEX IF NOT EXISTS idx_mtn_push_events_created_at ON mtn_push_events(created_at DESC);

-- ── 3. Network config keys for the MTN push pipeline ─────────────────────────
INSERT INTO network_configs (key, value, description) VALUES
    ('draw_entries_per_point',   '1',    'Draw entries created per Pulse Point earned from a recharge (1:1 mirrors RechargeMax)'),
    ('mtn_push_hmac_secret',     '',     'HMAC-SHA256 secret for MTN push webhook signature verification (set via env MTN_PUSH_SECRET)'),
    ('mtn_push_min_amount_naira','50',   'Minimum recharge amount in naira for MTN push to qualify for points/draw/spin')
ON CONFLICT (key) DO NOTHING;
