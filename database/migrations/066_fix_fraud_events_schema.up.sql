-- Migration 066: Fix fraud_events schema to match the FraudEvent Go struct
-- The table was created with rule_name/JSONB details but the struct expects event_type/TEXT details

-- Add event_type column (maps to what the Go struct calls EventType)
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS event_type TEXT NOT NULL DEFAULT '';

-- Backfill event_type from rule_name for any existing rows
UPDATE fraud_events SET event_type = rule_name WHERE event_type = '' AND rule_name IS NOT NULL AND rule_name != '';

-- Change details from JSONB to TEXT (the Go struct uses string, not map)
-- First add a new text column, copy data, then rename
DO $$
BEGIN
    -- Only do the conversion if details is still JSONB
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
EXCEPTION WHEN OTHERS THEN
    NULL; -- ignore if already done
END $$;

-- Ensure updated_at exists (may have been added by migration 065)
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Verify
SELECT column_name, data_type FROM information_schema.columns
WHERE table_name = 'fraud_events'
ORDER BY ordinal_position;
