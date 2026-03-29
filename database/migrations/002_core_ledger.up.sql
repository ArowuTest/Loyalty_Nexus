-- 002_core_ledger.sql
-- Purpose: Atomic ledger for High-Throughput Loyalty transactions.

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    msisdn TEXT UNIQUE NOT NULL, -- Normalized 234...
    user_code TEXT UNIQUE NOT NULL,
    total_points BIGINT DEFAULT 0,
    stamps_count INTEGER DEFAULT 0,
    total_recharge_amount BIGINT DEFAULT 0,
    tier TEXT DEFAULT 'BRONZE',
    streak_count INTEGER DEFAULT 0,
    last_visit_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_msisdn ON users(msisdn);

CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    msisdn TEXT NOT NULL,
    type TEXT NOT NULL, -- visit, reward_redeem, bonus, studio_spend
    points_delta BIGINT DEFAULT 0,
    stamps_delta INTEGER DEFAULT 0,
    amount BIGINT DEFAULT 0, -- in Kobo
    balance_after BIGINT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_date ON transactions(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_msisdn ON transactions(msisdn);

-- ATOMIC TRIGGER: Handle all balance and streak logic in the DB layer
CREATE OR REPLACE FUNCTION trg_fn_loyalty_nexus_ledger()
RETURNS TRIGGER AS $$
DECLARE
    v_streak_window INTEGER;
BEGIN
    -- Get streak window from cockpit config (default 48h)
    SELECT (config_value->>'hours')::int INTO v_streak_window 
    FROM program_configs WHERE config_key = 'streak_window' 
    LIMIT 1;
    IF v_streak_window IS NULL THEN v_streak_window := 48; END IF;

    -- Atomic balance update
    UPDATE users
    SET
        total_points = total_points + NEW.points_delta,
        stamps_count = stamps_count + NEW.stamps_delta,
        total_recharge_amount = total_recharge_amount + NEW.amount,
        last_visit_at = CASE WHEN NEW.type = 'visit' THEN NOW() ELSE last_visit_at END,
        streak_count = CASE 
            WHEN NEW.type != 'visit' THEN streak_count
            WHEN last_visit_at IS NULL THEN 1
            WHEN last_visit_at > NOW() - (v_streak_window * interval '1 hour') THEN streak_count + 1
            ELSE 1
        END,
        updated_at = now()
    WHERE id = NEW.user_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_loyalty_nexus_ledger
    AFTER INSERT ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION trg_fn_loyalty_nexus_ledger();
