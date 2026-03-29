-- 013_prize_fulfillment.sql
-- Purpose: Support for varied prize types and fulfillment tracking (REQ-3.4).

-- Update prize_pool types if necessary
ALTER TABLE prize_pool DROP CONSTRAINT IF EXISTS prize_pool_prize_type_check;
ALTER TABLE prize_pool ADD CONSTRAINT prize_pool_prize_type_check 
CHECK (prize_type IN ('airtime', 'data', 'momo_cash', 'bonus_points', 'studio_credits', 'try_again'));

CREATE TABLE IF NOT EXISTS prize_claims (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    transaction_id UUID NOT NULL REFERENCES transactions(id),
    prize_type TEXT NOT NULL,
    prize_value NUMERIC NOT NULL,
    status TEXT CHECK (status IN ('pending_momo_link', 'pending_fulfillment', 'processing', 'completed', 'failed')) DEFAULT 'pending_fulfillment',
    momo_number TEXT,
    fulfillment_ref TEXT, -- VTPass ref or MoMo ref
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_prize_claims_user ON prize_claims(user_id, status);
