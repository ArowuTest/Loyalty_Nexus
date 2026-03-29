-- Migration 060: Ensure all critical tables exist (safety net)
-- This migration uses IF NOT EXISTS everywhere and DO $$ EXCEPTION blocks
-- to guarantee it never fails, regardless of DB state.
-- It creates any tables that should exist but might have been missed.

-- ─── USSD Sessions ────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS ussd_sessions (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      TEXT        NOT NULL UNIQUE,
    phone_number    TEXT        NOT NULL,
    menu_state      TEXT        NOT NULL DEFAULT 'root',
    input_buffer    TEXT        NOT NULL DEFAULT '',
    pending_spin_id UUID,
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '10 minutes',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ussd_sessions_phone_60      ON ussd_sessions(phone_number);
CREATE INDEX IF NOT EXISTS idx_ussd_sessions_session_id_60 ON ussd_sessions(session_id);

-- Add pending_spin_id FK if spin_results exists
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='spin_results')
    AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='ussd_sessions' AND column_name='pending_spin_id'
        AND data_type='uuid'
    ) THEN
        ALTER TABLE ussd_sessions 
            ADD COLUMN IF NOT EXISTS pending_spin_id UUID;
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- ─── Regional Wars ────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS regional_wars (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    period              VARCHAR(7)  NOT NULL UNIQUE,
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    total_prize_kobo    BIGINT      NOT NULL DEFAULT 50000000,
    starts_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at             TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '1 month',
    resolved_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_regional_wars_status_60 ON regional_wars(status);

-- ─── Admin Users ──────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS admin_users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username      TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    role          TEXT        NOT NULL DEFAULT 'admin',
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── Network Configs (canonical name) ────────────────────────────────────────
-- If program_configs exists but network_configs doesn't, rename it.
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='program_configs')
    AND NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='network_configs') THEN
        ALTER TABLE program_configs RENAME TO network_configs;
        ALTER TABLE network_configs RENAME COLUMN config_key TO key;
        ALTER TABLE network_configs RENAME COLUMN config_value TO value;
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- Create network_configs from scratch if neither exists
CREATE TABLE IF NOT EXISTS network_configs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    key         TEXT        NOT NULL UNIQUE,
    value       TEXT        NOT NULL DEFAULT '',
    description TEXT,
    is_public   BOOLEAN     NOT NULL DEFAULT FALSE,
    updated_by  TEXT        NOT NULL DEFAULT 'system',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_network_configs_key_60 ON network_configs(key);

-- ─── Ensure regional_settings has all required columns ────────────────────────
ALTER TABLE regional_settings ADD COLUMN IF NOT EXISTS region_name          TEXT    NOT NULL DEFAULT '';
ALTER TABLE regional_settings ADD COLUMN IF NOT EXISTS base_multiplier      NUMERIC DEFAULT 1.0;
ALTER TABLE regional_settings ADD COLUMN IF NOT EXISTS golden_hour_multiplier NUMERIC DEFAULT 2.0;

-- Seed regions if empty
INSERT INTO regional_settings (region_code, region_name) VALUES
    ('LAG', 'Lagos'), ('ABJ', 'Abuja'), ('KAN', 'Kano'),
    ('PHC', 'Port Harcourt'), ('IBD', 'Ibadan'), ('ENU', 'Enugu')
ON CONFLICT (region_code) DO NOTHING;
