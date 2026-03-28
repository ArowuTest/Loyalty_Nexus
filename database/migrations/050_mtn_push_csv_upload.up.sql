-- Migration 050: MTN push CSV bulk upload
--
-- When the MTN push API is unavailable, admins can upload a CSV file
-- containing MSISDN, date, time, and recharge amount.  Each row is
-- processed through the same pipeline as a live MTN push webhook:
--   spin credits, pulse points, draw entries, ledger entries.
--
-- This table provides a full audit trail for every upload batch and
-- every individual row within it.

-- ─── Upload batch header ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mtn_push_csv_uploads (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Who uploaded and when
    uploaded_by     TEXT        NOT NULL,   -- admin user_id or email
    filename        TEXT        NOT NULL,   -- original filename for reference
    uploaded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Counts
    total_rows      INTEGER     NOT NULL DEFAULT 0,
    processed_rows  INTEGER     NOT NULL DEFAULT 0,
    skipped_rows    INTEGER     NOT NULL DEFAULT 0,  -- duplicates / below-min
    failed_rows     INTEGER     NOT NULL DEFAULT 0,

    -- Overall status
    status          TEXT        NOT NULL DEFAULT 'PENDING'
                        CHECK (status IN ('PENDING','PROCESSING','DONE','PARTIAL','FAILED')),

    -- Optional admin note
    note            TEXT,

    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_csv_uploads_status
    ON mtn_push_csv_uploads (status, uploaded_at DESC);

-- ─── Per-row result ───────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mtn_push_csv_rows (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id       UUID        NOT NULL REFERENCES mtn_push_csv_uploads(id) ON DELETE CASCADE,

    -- Original CSV values (stored verbatim for auditability)
    row_number      INTEGER     NOT NULL,   -- 1-based line number in the CSV
    raw_msisdn      TEXT        NOT NULL,
    raw_date        TEXT        NOT NULL,   -- e.g. "2025-05-14"
    raw_time        TEXT        NOT NULL,   -- e.g. "14:30:00"
    raw_amount      TEXT        NOT NULL,   -- e.g. "1000.00"
    recharge_type   TEXT        NOT NULL DEFAULT 'AIRTIME',

    -- Normalised values (set after parsing)
    msisdn          TEXT,
    recharge_at     TIMESTAMPTZ,
    amount_naira    NUMERIC(12,2),

    -- Processing outcome
    status          TEXT        NOT NULL DEFAULT 'PENDING'
                        CHECK (status IN ('PENDING','OK','SKIPPED','FAILED')),
    skip_reason     TEXT,       -- e.g. "duplicate", "below_minimum"
    error_msg       TEXT,       -- set on FAILED rows

    -- Rewards awarded (mirrors mtn_push_events columns)
    transaction_ref TEXT,       -- the synthetic ref used for idempotency
    spin_credits    INTEGER,
    pulse_points    BIGINT,
    draw_entries    INTEGER,

    processed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_csv_rows_upload
    ON mtn_push_csv_rows (upload_id, row_number);

CREATE INDEX IF NOT EXISTS idx_csv_rows_msisdn
    ON mtn_push_csv_rows (msisdn)
    WHERE msisdn IS NOT NULL;
