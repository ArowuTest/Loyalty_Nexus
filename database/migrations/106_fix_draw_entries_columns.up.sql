-- 106_fix_draw_entries_columns.up.sql
-- Ensures draw_entries has all required columns regardless of which
-- CREATE TABLE variant ran first (016 vs 060 schemas differ).
-- Safe to run multiple times — all ADD COLUMN IF NOT EXISTS.

-- Core columns from migration 016 (original schema)
ALTER TABLE draw_entries
    ADD COLUMN IF NOT EXISTS msisdn        TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS entries_count INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS entry_source  TEXT NOT NULL DEFAULT 'recharge',
    ADD COLUMN IF NOT EXISTS amount        BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS recharge_at   TIMESTAMPTZ;

-- Generated aliases added in migration 049
-- (Only add if not already generated columns — safe with IF NOT EXISTS)
-- phone_number: alias for msisdn
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'draw_entries' AND column_name = 'phone_number'
    ) THEN
        ALTER TABLE draw_entries
            ADD COLUMN phone_number TEXT GENERATED ALWAYS AS (msisdn) STORED;
    END IF;
END$$;

-- ticket_count: alias for entries_count
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'draw_entries' AND column_name = 'ticket_count'
    ) THEN
        ALTER TABLE draw_entries
            ADD COLUMN ticket_count INTEGER GENERATED ALWAYS AS (entries_count) STORED;
    END IF;
END$$;

-- Ensure draw_winners has all required columns (migration 060 vs 016/024 differ)
ALTER TABLE draw_winners
    ADD COLUMN IF NOT EXISTS phone_number     TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS prize_value_kobo BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS status           TEXT NOT NULL DEFAULT 'PENDING_FULFILLMENT';

-- Back-fill phone_number for winners that have msisdn but not phone_number
UPDATE draw_winners
SET phone_number = msisdn
WHERE phone_number = '' AND msisdn IS NOT NULL AND msisdn != '';
