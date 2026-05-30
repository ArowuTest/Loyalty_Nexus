-- 126_prize_fulfillment_config.up.sql
-- Admin-configurable fulfillment mode per prize type.
-- Mirrors the RechargeMax prize_fulfillment_config pattern:
--   - fulfillment_mode = 'MANUAL' (default) → user must click Claim from dashboard
--   - fulfillment_mode = 'AUTO'   → background goroutine fires VTPass at spin time
-- All prize types seeded as MANUAL so existing behaviour is unchanged on deploy.

CREATE TABLE prize_fulfillment_config (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    prize_type          TEXT        NOT NULL UNIQUE,
    fulfillment_mode    TEXT        NOT NULL DEFAULT 'MANUAL' CHECK (fulfillment_mode IN ('MANUAL', 'AUTO')),
    max_retry_attempts  INT         NOT NULL DEFAULT 3 CHECK (max_retry_attempts >= 1 AND max_retry_attempts <= 10),
    retry_delay_seconds INT         NOT NULL DEFAULT 30 CHECK (retry_delay_seconds >= 5),
    fallback_to_manual  BOOLEAN     NOT NULL DEFAULT TRUE,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed all known prize types with MANUAL mode (safe default).
INSERT INTO prize_fulfillment_config (prize_type, fulfillment_mode) VALUES
    ('try_again',    'MANUAL'),
    ('pulse_points', 'MANUAL'),
    ('airtime',      'MANUAL'),
    ('data_bundle',  'MANUAL'),
    ('momo_cash',    'MANUAL'),
    ('physical',     'MANUAL'),
    ('goods',        'MANUAL')
ON CONFLICT (prize_type) DO NOTHING;
