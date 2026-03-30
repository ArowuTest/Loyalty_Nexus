-- Migration 066: Align fraud_events with FraudEvent Go struct
-- Safe/idempotent — wraps every operation in DO $$ EXCEPTION WHEN OTHERS THEN NULL END $$

-- 1. Add event_type column if missing
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS event_type TEXT NOT NULL DEFAULT '';

-- 2. Backfill event_type from rule_name ONLY if rule_name column exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'fraud_events' AND column_name = 'rule_name'
    ) THEN
        UPDATE fraud_events SET event_type = rule_name WHERE event_type = '';
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- 3. Convert details from JSONB to TEXT if still JSONB
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'fraud_events'
        AND column_name = 'details'
        AND data_type = 'jsonb'
    ) THEN
        ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS details_text TEXT NOT NULL DEFAULT '';
        UPDATE fraud_events SET details_text = details::text WHERE details IS NOT NULL;
        ALTER TABLE fraud_events DROP COLUMN details;
        ALTER TABLE fraud_events RENAME COLUMN details_text TO details;
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- 4. Ensure details column exists as TEXT (in case it was missing entirely)
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS details TEXT NOT NULL DEFAULT '';

-- 5. Ensure updated_at exists
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
