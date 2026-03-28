-- 015_fraud_guards.sql
-- Purpose: Support for velocity-based fraud prevention and blacklisting.

CREATE TABLE msisdn_blacklist (
    msisdn TEXT PRIMARY KEY,
    reason TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- Index for velocity checks on transactions
CREATE INDEX IF NOT EXISTS idx_transactions_msisdn_created ON transactions(msisdn, created_at DESC);
