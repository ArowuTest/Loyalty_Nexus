-- 106_fix_draw_entries_columns.up.sql
-- Ensures draw_entries and draw_winners have all columns needed by the app.
-- Uses EXCEPTION handlers so it NEVER fails — safe for all deployment scenarios.

DO $$
BEGIN
    -- ── draw_entries: add missing columns from migration 016/045 ──────────
    BEGIN
        ALTER TABLE draw_entries ADD COLUMN msisdn TEXT NOT NULL DEFAULT '';
    EXCEPTION WHEN duplicate_column THEN NULL;
    END;
    BEGIN
        ALTER TABLE draw_entries ADD COLUMN entries_count INTEGER NOT NULL DEFAULT 1;
    EXCEPTION WHEN duplicate_column THEN NULL;
    END;
    BEGIN
        ALTER TABLE draw_entries ADD COLUMN entry_source TEXT NOT NULL DEFAULT 'recharge';
    EXCEPTION WHEN duplicate_column THEN NULL;
    END;
    BEGIN
        ALTER TABLE draw_entries ADD COLUMN amount BIGINT NOT NULL DEFAULT 0;
    EXCEPTION WHEN duplicate_column THEN NULL;
    END;
    BEGIN
        ALTER TABLE draw_entries ADD COLUMN recharge_at TIMESTAMPTZ;
    EXCEPTION WHEN duplicate_column THEN NULL;
    END;

    -- ── draw_entries: generated alias columns (migration 049) ─────────────
    -- phone_number = GENERATED ALWAYS AS (msisdn) STORED
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'draw_entries' AND column_name = 'phone_number'
    ) THEN
        BEGIN
            ALTER TABLE draw_entries
                ADD COLUMN phone_number TEXT GENERATED ALWAYS AS (msisdn) STORED;
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'Could not add generated column phone_number: %', SQLERRM;
        END;
    END IF;

    -- ticket_count = GENERATED ALWAYS AS (entries_count) STORED
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'draw_entries' AND column_name = 'ticket_count'
    ) THEN
        BEGIN
            ALTER TABLE draw_entries
                ADD COLUMN ticket_count INTEGER GENERATED ALWAYS AS (entries_count) STORED;
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'Could not add generated column ticket_count: %', SQLERRM;
        END;
    END IF;

    -- ── draw_winners: add missing columns ─────────────────────────────────
    BEGIN
        ALTER TABLE draw_winners ADD COLUMN phone_number TEXT NOT NULL DEFAULT '';
    EXCEPTION WHEN duplicate_column THEN NULL;
    END;
    BEGIN
        ALTER TABLE draw_winners ADD COLUMN prize_value_kobo BIGINT NOT NULL DEFAULT 0;
    EXCEPTION WHEN duplicate_column THEN NULL;
    END;
    BEGIN
        ALTER TABLE draw_winners ADD COLUMN status TEXT NOT NULL DEFAULT 'PENDING_FULFILLMENT';
    EXCEPTION WHEN duplicate_column THEN NULL;
    END;

    -- ── Back-fill draw_winners.phone_number from msisdn (if msisdn exists) ─
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'draw_winners' AND column_name = 'msisdn'
    ) THEN
        UPDATE draw_winners
        SET phone_number = msisdn
        WHERE phone_number = '' AND msisdn IS NOT NULL AND msisdn != '';
    END IF;

END$$;
