-- Migration 047: Fix trg_fn_loyalty_nexus_ledger trigger
--
-- The trigger function trg_fn_loyalty_nexus_ledger was querying a table
-- called "program_configs" which does not exist in Loyalty Nexus.
-- The correct table is "network_configs" (key TEXT, value JSONB).
--
-- This bug caused EVERY INSERT into the transactions table to fail with:
--   ERROR: relation "program_configs" does not exist (SQLSTATE 42P01)
--
-- This migration replaces the trigger function body to:
--   1. Read streak_window_hours from network_configs (default 48h)
--   2. Update users.total_recharge_amount on recharge transactions
--   3. Leave streak management to the application layer (recharge_service.go
--      and mtn_push_service.go already handle streak via UpdateStreak)
--
-- NOTE: The trigger no longer updates total_points or stamps_count because
-- Loyalty Nexus uses a separate wallet table (wallets.pulse_points) and
-- the application layer manages all balance updates atomically. The trigger
-- only needs to maintain total_recharge_amount as a denormalised counter.

CREATE OR REPLACE FUNCTION trg_fn_loyalty_nexus_ledger()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE
    v_streak_window_hours INTEGER;
BEGIN
    -- Read streak window from network_configs (default 48h).
    -- network_configs stores values as JSONB; plain integer values are stored
    -- as JSON numbers (e.g. 48), so we cast via text.
    SELECT (value::text)::int INTO v_streak_window_hours
    FROM network_configs
    WHERE key = 'streak_window_hours'
    LIMIT 1;

    IF v_streak_window_hours IS NULL THEN
        v_streak_window_hours := 48;
    END IF;

    -- Maintain the denormalised total_recharge_amount counter on users.
    -- All other balance fields (pulse_points, spin_credits, lifetime_points)
    -- are managed by the application layer via the wallets table.
    IF NEW.type = 'recharge' AND NEW.amount > 0 THEN
        UPDATE users
        SET
            total_recharge_amount = total_recharge_amount + NEW.amount,
            last_recharge_at      = NOW(),
            updated_at            = NOW()
        WHERE id = NEW.user_id;
    END IF;

    RETURN NEW;
END;
$$;

-- Verify the trigger is still attached (it should be — we only replaced the function body).
-- If for any reason it was dropped, recreate it.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger t
        JOIN pg_class c ON t.tgrelid = c.oid
        WHERE c.relname = 'transactions'
          AND t.tgname = 'trg_loyalty_nexus_ledger'
    ) THEN
        CREATE TRIGGER trg_loyalty_nexus_ledger
            AFTER INSERT ON transactions
            FOR EACH ROW EXECUTE FUNCTION trg_fn_loyalty_nexus_ledger();
    END IF;
END;
$$;

-- Seed the streak_window_hours config key if it does not already exist.
INSERT INTO network_configs (key, value, description)
VALUES ('streak_window_hours', '48', 'Hours within which consecutive recharges count as a streak')
ON CONFLICT (key) DO NOTHING;
