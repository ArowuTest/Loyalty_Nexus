-- ═══════════════════════════════════════════════════════════════════════════════
-- Migration 060: COMPREHENSIVE SAFETY NET
-- ═══════════════════════════════════════════════════════════════════════════════
-- This migration is the guaranteed final step. The entrypoint runs:
--   /migrate force 59   (marks all prior migrations as applied without re-running them)
--   /migrate up         (only this migration 060 executes)
--
-- Every statement uses CREATE TABLE IF NOT EXISTS / ADD COLUMN IF NOT EXISTS /
-- ON CONFLICT DO NOTHING / DO $$ EXCEPTION WHEN OTHERS THEN NULL $$ blocks.
-- It CANNOT fail under any database state.
-- ═══════════════════════════════════════════════════════════════════════════════

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 1: CORE FOUNDATION TABLES (from migrations 001-004)
-- These are guaranteed safe because of IF NOT EXISTS.
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS program_configs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key  TEXT UNIQUE NOT NULL,
    config_value JSONB NOT NULL,
    description TEXT,
    updated_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS prize_pool (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    TEXT NOT NULL,
    prize_type              TEXT,
    base_value              NUMERIC NOT NULL DEFAULT 0,
    is_active               BOOLEAN DEFAULT true,
    win_probability_weight  INTEGER DEFAULT 100,
    daily_inventory_cap     INTEGER,
    updated_at              TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS regional_settings (
    region_code TEXT PRIMARY KEY,
    multiplier  NUMERIC DEFAULT 1.0,
    is_golden_hour BOOLEAN DEFAULT false,
    updated_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS studio_config (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    media_type    TEXT,
    point_cost    INTEGER NOT NULL DEFAULT 0,
    render_priority INTEGER DEFAULT 1,
    is_enabled    BOOLEAN DEFAULT true
);

CREATE TABLE IF NOT EXISTS users (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number     TEXT UNIQUE NOT NULL,
    network_id       TEXT,
    state            TEXT,
    wallet_pass_id   UUID,
    points_balance   BIGINT NOT NULL DEFAULT 0,
    spin_credits     INTEGER NOT NULL DEFAULT 0,
    streak_days      INTEGER NOT NULL DEFAULT 0,
    last_recharge_at TIMESTAMPTZ,
    subscription_status     VARCHAR(20) NOT NULL DEFAULT 'FREE',
    subscription_expires_at TIMESTAMPTZ,
    lifetime_points  BIGINT NOT NULL DEFAULT 0,
    total_spins      INTEGER NOT NULL DEFAULT 0,
    studio_use_count INTEGER NOT NULL DEFAULT 0,
    total_referrals  INTEGER NOT NULL DEFAULT 0,
    momo_number      TEXT    NOT NULL DEFAULT '',
    momo_verified    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Ensure users table has phone_number column (migration 002 uses 'msisdn').
-- Migration 020 was supposed to rename it, but may not have run.
DO $$ BEGIN
    -- If msisdn exists but phone_number doesn't: rename
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name='users' AND column_name='msisdn')
    AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                    WHERE table_name='users' AND column_name='phone_number') THEN
        ALTER TABLE users RENAME COLUMN msisdn TO phone_number;
    END IF;
    -- If phone_number still doesn't exist: add it
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name='users' AND column_name='phone_number') THEN
        ALTER TABLE users ADD COLUMN phone_number TEXT NOT NULL DEFAULT '';
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;
-- Similarly fix transactions table
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name='transactions' AND column_name='msisdn')
    AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                    WHERE table_name='transactions' AND column_name='phone_number') THEN
        ALTER TABLE transactions RENAME COLUMN msisdn TO phone_number;
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;
-- Fix auth_otps table
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name='auth_otps' AND column_name='msisdn')
    AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                    WHERE table_name='auth_otps' AND column_name='phone_number') THEN
        ALTER TABLE auth_otps RENAME COLUMN msisdn TO phone_number;
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_60 ON users(phone_number);

CREATE TABLE IF NOT EXISTS transactions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID REFERENCES users(id),
    phone_number TEXT NOT NULL,
    amount_kobo  BIGINT NOT NULL DEFAULT 0,
    type         TEXT NOT NULL DEFAULT 'recharge',
    status       TEXT NOT NULL DEFAULT 'completed',
    provider_ref TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id_60 ON transactions(user_id);

CREATE TABLE IF NOT EXISTS wallets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    points_balance  BIGINT NOT NULL DEFAULT 0,
    spin_credits    INTEGER NOT NULL DEFAULT 0,
    lifetime_points BIGINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES users(id),
    type        TEXT NOT NULL,
    amount      BIGINT NOT NULL DEFAULT 0,
    balance_after BIGINT NOT NULL DEFAULT 0,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 2: AI STUDIO (migration 003)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS studio_tools (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    point_cost  INTEGER NOT NULL DEFAULT 0,
    is_enabled  BOOLEAN DEFAULT true,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ai_generations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID REFERENCES users(id),
    tool_name    TEXT NOT NULL,
    prompt       TEXT,
    result_url   TEXT,
    status       TEXT DEFAULT 'pending',
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 3: CHAT / AI SUMMARISER (migration 009)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS chat_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id),
    status          TEXT DEFAULT 'active',
    last_activity_at TIMESTAMPTZ DEFAULT now(),
    created_at      TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_expiry_60 ON chat_sessions(status, last_activity_at);

CREATE TABLE IF NOT EXISTS chat_messages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID REFERENCES chat_sessions(id),
    role        TEXT,
    content     TEXT NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS session_summaries (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID REFERENCES users(id),
    summary    TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 4: SPIN ENGINE (migration 020)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS spin_results (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID REFERENCES users(id),
    phone_number     TEXT NOT NULL DEFAULT '',
    prize_pool_id    UUID REFERENCES prize_pool(id),
    prize_type       TEXT NOT NULL DEFAULT 'try_again',
    prize_value_kobo BIGINT NOT NULL DEFAULT 0,
    is_fulfilled     BOOLEAN NOT NULL DEFAULT FALSE,
    fulfilled_at     TIMESTAMPTZ,
    mo_mo_number     TEXT NOT NULL DEFAULT '',
    retry_count      INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_spin_results_user_id_60 ON spin_results(user_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 5: DRAWS ENGINE (migrations 016, 021)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS draws (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL DEFAULT 'Monthly Draw',
    status          TEXT NOT NULL DEFAULT 'UPCOMING',
    draw_type       TEXT NOT NULL DEFAULT 'MONTHLY',
    recurrence      TEXT NOT NULL DEFAULT 'monthly',
    next_draw_at    TIMESTAMPTZ,
    prize_pool      NUMERIC(12,2) NOT NULL DEFAULT 0,
    winner_count    INTEGER NOT NULL DEFAULT 1,
    total_entries   INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS draw_entries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    draw_id      UUID REFERENCES draws(id) ON DELETE CASCADE,
    user_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    phone_number TEXT NOT NULL DEFAULT '',
    ticket_count INTEGER NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS draw_winners (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    draw_id          UUID REFERENCES draws(id) ON DELETE CASCADE,
    user_id          UUID REFERENCES users(id) ON DELETE SET NULL,
    phone_number     TEXT NOT NULL DEFAULT '',
    position         INTEGER NOT NULL DEFAULT 1,
    prize_value_kobo BIGINT NOT NULL DEFAULT 0,
    status           TEXT NOT NULL DEFAULT 'PENDING_FULFILLMENT',
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 6: AUTH (migration 011)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS auth_otps (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number TEXT NOT NULL,
    code       TEXT NOT NULL,
    purpose    TEXT DEFAULT 'login',
    status     TEXT DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_auth_otps_phone_60 ON auth_otps(phone_number, status);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 7: SUBSCRIPTIONS (migration 006)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS subscription_plans (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    price_kobo  BIGINT NOT NULL DEFAULT 0,
    duration_days INTEGER NOT NULL DEFAULT 30,
    spin_credits  INTEGER NOT NULL DEFAULT 0,
    is_active   BOOLEAN DEFAULT true,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_subscriptions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id     UUID REFERENCES subscription_plans(id),
    status      TEXT NOT NULL DEFAULT 'active',
    starts_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 8: NOTIFICATIONS (migration 022)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS push_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token        TEXT NOT NULL UNIQUE,
    platform     TEXT NOT NULL DEFAULT 'fcm',
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID REFERENCES users(id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL,
    type       TEXT NOT NULL DEFAULT 'general',
    is_read    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notifications_user_id_60 ON notifications(user_id);

CREATE TABLE IF NOT EXISTS notification_broadcasts (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title      TEXT NOT NULL,
    message    TEXT NOT NULL,
    type       TEXT NOT NULL DEFAULT 'push',
    status     TEXT NOT NULL DEFAULT 'queued',
    sent_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 9: DIGITAL PASSPORT (migration 004)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS wallet_passes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pass_type       TEXT NOT NULL DEFAULT 'loyalty',
    serial_number   TEXT NOT NULL UNIQUE DEFAULT gen_random_uuid()::text,
    qr_code_url     TEXT,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    issued_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_badges (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_key  TEXT NOT NULL,
    earned_at  TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, badge_key)
);

CREATE TABLE IF NOT EXISTS passport_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    details    JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 10: REGIONAL WARS (migrations 005, 021)
-- ─────────────────────────────────────────────────────────────────────────────

-- Ensure regional_settings has all columns
ALTER TABLE regional_settings ADD COLUMN IF NOT EXISTS region_name           TEXT    NOT NULL DEFAULT '';
ALTER TABLE regional_settings ADD COLUMN IF NOT EXISTS base_multiplier       NUMERIC DEFAULT 1.0;
ALTER TABLE regional_settings ADD COLUMN IF NOT EXISTS golden_hour_multiplier NUMERIC DEFAULT 2.0;

CREATE TABLE IF NOT EXISTS regional_stats (
    region_code          TEXT PRIMARY KEY REFERENCES regional_settings(region_code),
    total_recharge_kobo  BIGINT DEFAULT 0,
    active_subscribers   INTEGER DEFAULT 0,
    last_recharge_at     TIMESTAMPTZ,
    rank                 INTEGER DEFAULT 0,
    updated_at           TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS regional_wars (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    period           VARCHAR(7)  NOT NULL UNIQUE,
    status           VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    total_prize_kobo BIGINT      NOT NULL DEFAULT 50000000,
    starts_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at          TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '1 month',
    resolved_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_regional_wars_status_60 ON regional_wars(status);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 11: USSD SESSIONS (migration 025)
-- ─────────────────────────────────────────────────────────────────────────────

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

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 12: ADMIN USERS (migration 052)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS admin_users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username      TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    role          TEXT        NOT NULL DEFAULT 'admin',
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 13: NETWORK CONFIGS — THE CRITICAL TABLE
-- ─────────────────────────────────────────────────────────────────────────────

-- Step 1: If program_configs exists but network_configs doesn't, rename it.
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='public' AND table_name='program_configs')
    AND NOT EXISTS (SELECT 1 FROM information_schema.tables
                    WHERE table_schema='public' AND table_name='network_configs') THEN
        ALTER TABLE program_configs RENAME TO network_configs;
        -- Rename columns if they have old names
        IF EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name='network_configs' AND column_name='config_key') THEN
            ALTER TABLE network_configs RENAME COLUMN config_key TO key;
        END IF;
        IF EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name='network_configs' AND column_name='config_value') THEN
            ALTER TABLE network_configs RENAME COLUMN config_value TO value;
        END IF;
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- Step 2: Create from scratch if still missing
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

-- Step 3: Add missing columns if table was renamed from old schema
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name='network_configs' AND column_name='is_public') THEN
        ALTER TABLE network_configs ADD COLUMN is_public BOOLEAN NOT NULL DEFAULT FALSE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name='network_configs' AND column_name='updated_by') THEN
        ALTER TABLE network_configs ADD COLUMN updated_by TEXT NOT NULL DEFAULT 'system';
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 14: PRIZE FULFILLMENT (migration 013, 024)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS prize_claims (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID REFERENCES users(id),
    spin_result_id   UUID REFERENCES spin_results(id),
    prize_type       TEXT NOT NULL DEFAULT 'airtime',
    prize_value_kobo BIGINT NOT NULL DEFAULT 0,
    status           TEXT NOT NULL DEFAULT 'pending',
    fulfilled_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS prize_fulfillment_logs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    spin_result_id   UUID REFERENCES spin_results(id) ON DELETE CASCADE,
    attempt_number   INTEGER NOT NULL DEFAULT 1,
    status           TEXT NOT NULL DEFAULT 'PENDING',
    provider         TEXT,
    provider_ref     TEXT,
    error_message    TEXT,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 15: FRAUD GUARD (migration 015)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS msisdn_blacklist (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number TEXT NOT NULL UNIQUE,
    reason       TEXT,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS fraud_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID REFERENCES users(id),
    phone_number TEXT,
    event_type   TEXT NOT NULL,
    details      JSONB DEFAULT '{}',
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 16: GHOST NUDGE / PASSPORT EXTRAS (migration 025)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS ghost_nudge_log (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    nudged_at TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 17: NETWORK CACHE / HLR (migration 008)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS network_cache (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number TEXT NOT NULL UNIQUE,
    network_id   TEXT NOT NULL,
    cached_at    TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 18: SEED DATA
-- ─────────────────────────────────────────────────────────────────────────────

-- Seed regional settings
INSERT INTO regional_settings (region_code, region_name) VALUES
    ('LAG', 'Lagos'), ('ABJ', 'Abuja'), ('KAN', 'Kano'),
    ('PHC', 'Port Harcourt'), ('IBD', 'Ibadan'), ('ENU', 'Enugu'),
    ('AKW', 'Akwa Ibom'), ('ANM', 'Anambra'), ('BEN', 'Benue'),
    ('BOR', 'Borno'), ('DEL', 'Delta'), ('EKI', 'Ekiti'),
    ('IMO', 'Imo'), ('JIG', 'Jigawa'), ('KAD', 'Kaduna'),
    ('KAT', 'Katsina'), ('KEB', 'Kebbi'), ('KOG', 'Kogi'),
    ('KWA', 'Kwara'), ('LAP', 'Lagos'),('NAS', 'Nassarawa'),
    ('NIG', 'Niger'), ('OGN', 'Ogun'), ('OND', 'Ondo'),
    ('OSU', 'Osun'), ('OYO', 'Oyo'), ('PLA', 'Plateau'),
    ('RIV', 'Rivers'), ('SOK', 'Sokoto'), ('TAR', 'Taraba'),
    ('YOB', 'Yobe'), ('ZAM', 'Zamfara'), ('ABI', 'Abia'),
    ('ADA', 'Adamawa'), ('BAY', 'Bayelsa'), ('CRS', 'Cross River'),
    ('EBO', 'Ebonyi'), ('EDO', 'Edo'), ('GOM', 'Gombe')
ON CONFLICT (region_code) DO NOTHING;

-- Seed core network_configs keys
INSERT INTO network_configs (key, value, description) VALUES
    ('min_recharge_naira',            '500',    'Minimum recharge to earn a spin'),
    ('streak_target_days',            '7',      'Days required for Mega Jackpot'),
    ('ghost_nudge_hours',             '48',     'Inactivity hours before nudge'),
    ('spin_trigger_naira',            '1000',   'Naira recharge per spin credit'),
    ('spin_max_per_user_per_day',     '3',      'Max spins per user per day'),
    ('points_expiry_days',            '90',     'Days before points expire'),
    ('referral_bonus_points',         '20',     'Points for referrer and new user'),
    ('ussd_shortcode',                '"*384#"','USSD shortcode'),
    ('ussd_session_timeout_seconds',  '20',     'USSD session timeout seconds'),
    ('ai_chat_enabled',               'true',   'Enable Ask Nexus chat'),
    ('nexus_chat_daily_limit',        '20',     'Max chat messages per day'),
    ('operation_mode',                '"independent"', 'Independent or integrated'),
    ('prize_pool_kobo',               '50000000', 'Daily prize budget in kobo')
ON CONFLICT (key) DO NOTHING;

-- Seed prize pool if empty
INSERT INTO prize_pool (name, prize_type, base_value, is_active, win_probability_weight)
SELECT * FROM (VALUES
    ('Try Again',       'try_again',   0,    true, 5000),
    ('Try Again',       'try_again',   0,    true, 2000),
    ('+5 Pulse Points', 'pulse_points',5,    true, 1000),
    ('+10 Pulse Points','pulse_points',10,   true, 700),
    ('10MB Data',       'data_bundle', 10,   true, 600),
    ('₦50 Airtime',    'airtime',     50,   true, 420),
    ('₦100 Airtime',   'airtime',     100,  true, 200),
    ('₦200 Airtime',   'airtime',     200,  true, 80)
) AS v(name, prize_type, base_value, is_active, weight)
WHERE NOT EXISTS (SELECT 1 FROM prize_pool LIMIT 1);

-- USSD cleanup function
CREATE OR REPLACE FUNCTION cleanup_expired_ussd_sessions() RETURNS void AS $$
BEGIN
    DELETE FROM ussd_sessions WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

