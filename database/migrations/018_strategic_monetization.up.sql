-- 018_strategic_monetization.sql
-- Purpose: Tracking metrics for SaaS monetization model (Section 6).

-- 1. ARPU Uplift Tracking (Monthly Snapshots per User)
CREATE TABLE IF NOT EXISTS arpu_uplift_tracking (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    msisdn TEXT NOT NULL,
    month_period DATE NOT NULL, -- e.g. 2026-03-01
    pre_program_avg_spend BIGINT, -- Baseline before joining program
    current_month_spend BIGINT DEFAULT 0,
    uplift_amount BIGINT GENERATED ALWAYS AS (current_month_spend - COALESCE(pre_program_avg_spend, 0)) STORED,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- 2. Churn Bounty (At-Risk Users reactivated)
CREATE TABLE IF NOT EXISTS churn_recovery_bounties (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    last_activity_before_reactivation TIMESTAMPTZ NOT NULL,
    reactivation_recharge_id UUID REFERENCES transactions(id),
    bounty_amount_kobo INTEGER DEFAULT 15000, -- Fixed 150 Naira fee per Section 6.3
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- 3. GPU Usage Tracking (Nexus Studio Monetization)
CREATE TABLE IF NOT EXISTS studio_usage_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    generation_id UUID NOT NULL REFERENCES ai_generations(id),
    provider TEXT NOT NULL,
    compute_cost_micros INTEGER, -- Internal tracking of API costs
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_arpu_period ON arpu_uplift_tracking(month_period);
CREATE INDEX IF NOT EXISTS idx_bounty_created ON churn_recovery_bounties(created_at);
