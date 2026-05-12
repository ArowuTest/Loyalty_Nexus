-- ============================================================
-- Migration 108: VTU Recharge tables
-- ============================================================

-- ── network_operator_configs ─────────────────────────────────────────────────
-- Admin-toggleable per-network config. No code deploy needed to add/remove a network.
-- GAP-7 fix: IsActive toggle is the single control; no separate DB provider_mode.
CREATE TABLE IF NOT EXISTS network_operator_configs (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    network_code    TEXT        NOT NULL UNIQUE,   -- "MTN", "GLO", "AIRTEL", "9MOBILE"
    network_name    TEXT        NOT NULL,
    logo_url        TEXT        NOT NULL DEFAULT '',
    brand_color     TEXT        NOT NULL DEFAULT '#FFCC00',
    is_active       BOOLEAN     NOT NULL DEFAULT FALSE,
    airtime_enabled BOOLEAN     NOT NULL DEFAULT TRUE,
    data_enabled    BOOLEAN     NOT NULL DEFAULT TRUE,
    min_amount      BIGINT      NOT NULL DEFAULT 10000,  -- kobo (₦100)
    max_amount      BIGINT      NOT NULL DEFAULT 500000, -- kobo (₦5,000)
    sort_order      INT         NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_noc_active ON network_operator_configs (is_active, sort_order);

-- Seed all 4 networks. MTN active, others off.
INSERT INTO network_operator_configs (network_code, network_name, logo_url, brand_color, is_active, airtime_enabled, data_enabled, sort_order) VALUES
    ('MTN',     'MTN Nigeria',  '/networks/mtn.png',     '#FFCC00', TRUE,  TRUE, TRUE, 1),
    ('GLO',     'Glo Mobile',   '/networks/glo.png',     '#006600', FALSE, TRUE, TRUE, 2),
    ('AIRTEL',  'Airtel Nigeria','/networks/airtel.png', '#FF0000', FALSE, TRUE, TRUE, 3),
    ('9MOBILE', '9mobile',      '/networks/9mobile.png', '#00A859', FALSE, TRUE, TRUE, 4)
ON CONFLICT (network_code) DO NOTHING;

-- ── recharges ────────────────────────────────────────────────────────────────
-- VTU recharge transactions initiated via the Loyalty Nexus platform.
-- Separate from the existing transactions table (which records point awards).
CREATE TABLE IF NOT EXISTS recharges (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               UUID        REFERENCES users(id) ON DELETE SET NULL,
    msisdn                TEXT        NOT NULL,                   -- normalised 234XXXXXXXXXX
    network               TEXT        NOT NULL,
    recharge_type         TEXT        NOT NULL CHECK (recharge_type IN ('AIRTIME','DATA')),
    amount_kobo           BIGINT      NOT NULL,
    data_variation_code   TEXT,                                   -- GAP fix: VTPass variation_code e.g. "mtn-10mb-100"
    payment_reference     TEXT        NOT NULL UNIQUE,
    vtpass_request_id     TEXT,                                   -- our reqID sent to VTPass
    vtpass_provider_ref   TEXT,                                   -- VTPass transactionId
    paystack_event_id     TEXT,                                   -- GAP-2: dedup key
    status                TEXT        NOT NULL DEFAULT 'PENDING'
                              CHECK (status IN ('PENDING','PROCESSING','SUCCESS','FAILED','CANCELLED')),
    failure_reason        TEXT,
    email                 TEXT        NOT NULL DEFAULT '',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at          TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_recharges_msisdn    ON recharges (msisdn, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_recharges_payref    ON recharges (payment_reference);
CREATE INDEX IF NOT EXISTS idx_recharges_status    ON recharges (status) WHERE status IN ('PENDING','PROCESSING');
CREATE INDEX IF NOT EXISTS idx_recharges_pstack_ev ON recharges (paystack_event_id) WHERE paystack_event_id IS NOT NULL;

-- ── webhook_events ────────────────────────────────────────────────────────────
-- GAP-2 fix: Paystack event dedup. Store event ID before processing — if it
-- already exists, the webhook is a retry and is silently ignored.
CREATE TABLE IF NOT EXISTS webhook_events (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    source       TEXT        NOT NULL DEFAULT 'paystack', -- 'paystack' | 'vtpass'
    event_id     TEXT        NOT NULL,
    event_type   TEXT        NOT NULL DEFAULT '',
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source, event_id)
);
CREATE INDEX IF NOT EXISTS idx_webhook_events_lookup ON webhook_events (source, event_id);
