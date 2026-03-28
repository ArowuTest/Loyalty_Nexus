-- 006_daily_subscriptions.sql
-- Purpose: Support for N20/day guaranteed draw entry subscriptions.

-- 1. Subscription Plans (Configurable)
CREATE TABLE subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    daily_cost_kobo INTEGER NOT NULL DEFAULT 2000, -- 2000 Kobo = N20
    entries_per_day INTEGER NOT NULL DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- 2. User Subscriptions
CREATE TABLE user_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    plan_id UUID NOT NULL REFERENCES subscription_plans(id),
    status TEXT CHECK (status IN ('active', 'paused', 'cancelled', 'pending_payment')) DEFAULT 'active',
    next_billing_at TIMESTAMPTZ NOT NULL,
    last_billed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_user_subscriptions_user ON user_subscriptions(user_id);
CREATE INDEX idx_user_subscriptions_billing ON user_subscriptions(next_billing_at);

-- 3. Seed Default N20 Plan
INSERT INTO subscription_plans (name, daily_cost_kobo, entries_per_day) VALUES
('Daily Draw Pass', 2000, 1);
