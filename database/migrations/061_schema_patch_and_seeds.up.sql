-- Migration 061: Schema patch + seeds for existing DBs
-- Adds all columns the Go entities require that may be missing from the live users
-- table (created by migration 002 before later ALTER TABLE migrations ran).
-- Also seeds the super_admin and 5 test users idempotently.
-- Every statement uses IF NOT EXISTS / ON CONFLICT DO NOTHING/UPDATE — fully safe to re-run.

-- ─── 1. USERS TABLE — ensure all entity columns exist ────────────────────────
ALTER TABLE users ADD COLUMN IF NOT EXISTS user_code              TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS state                  TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS tier                   TEXT         NOT NULL DEFAULT 'BRONZE';
ALTER TABLE users ADD COLUMN IF NOT EXISTS streak_count           INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS streak_expires_at      TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS streak_grace_used      INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS streak_grace_month     INTEGER;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_recharge_amount  BIGINT       NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_recharge_at       TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS momo_number            TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS momo_verified          BOOLEAN      NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS momo_verified_at       TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS wallet_pass_id         TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS device_type            TEXT         NOT NULL DEFAULT 'smartphone';
ALTER TABLE users ADD COLUMN IF NOT EXISTS subscription_tier      TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS subscription_status    TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS subscription_expires_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS referral_code          TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS referred_by            UUID;
ALTER TABLE users ADD COLUMN IF NOT EXISTS kyc_status             TEXT         NOT NULL DEFAULT 'unverified';
ALTER TABLE users ADD COLUMN IF NOT EXISTS points_expire_at       TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_points           BIGINT       NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS stamps_count           INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS lifetime_points        BIGINT       NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_spins            INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS studio_use_count       INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_referrals        INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS google_wallet_object_id TEXT        NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS apple_pass_serial      TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS spin_credits           INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active              BOOLEAN      NOT NULL DEFAULT TRUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS created_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW();
ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW();

-- ─── 2. ADMIN_USERS TABLE — ensure email/full_name/role columns exist ─────────
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS email         TEXT;
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS full_name     TEXT NOT NULL DEFAULT '';
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;
-- Make role TEXT in case it was created as an ENUM and the ENUM type is missing
DO $$ BEGIN
    ALTER TABLE admin_users ALTER COLUMN role TYPE TEXT;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;
-- Add unique constraint on email if not present
DO $$ BEGIN
    ALTER TABLE admin_users ADD CONSTRAINT admin_users_email_key UNIQUE (email);
EXCEPTION WHEN duplicate_table THEN NULL;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- ─── 3. SEED 5 TEST USERS ─────────────────────────────────────────────────────
-- Admin is seeded at startup via ADMIN_SEED_EMAIL / ADMIN_SEED_PASSWORD env vars.
INSERT INTO users (
    id, phone_number, user_code, state, tier,
    streak_count, streak_expires_at, streak_grace_used,
    total_recharge_amount, last_recharge_at,
    momo_number, momo_verified,
    referral_code, kyc_status,
    total_points, stamps_count, lifetime_points,
    total_spins, studio_use_count, spin_credits,
    is_active, created_at, updated_at
) VALUES
    -- Platinum power user
    (gen_random_uuid(), '+2348027000000', 'NXS-DEMO-04', 'Rivers', 'PLATINUM',
     7,  NOW() + INTERVAL '5 days', 0,
     100000000, NOW() - INTERVAL '12 hours',
     '2348027000000', TRUE,
     'DEMO04REF', 'verified',
     3200, 7, 8500,
     45, 15, 5,
     TRUE, NOW() - INTERVAL '90 days', NOW()),
    -- Gold user
    (gen_random_uuid(), '+2348020000000', 'NXS-DEMO-01', 'Lagos', 'GOLD',
     5,  NOW() + INTERVAL '3 days', 0,
     25000000, NOW() - INTERVAL '1 day',
     '', FALSE,
     'DEMO01REF', 'verified',
     800, 3, 2500,
     12, 4, 2,
     TRUE, NOW() - INTERVAL '30 days', NOW()),
    -- Silver user
    (gen_random_uuid(), '+2348023000000', 'NXS-DEMO-02', 'Abuja', 'SILVER',
     3,  NOW() + INTERVAL '1 day', 0,
     12000000, NOW() - INTERVAL '3 days',
     '', FALSE,
     'DEMO02REF', 'verified',
     350, 2, 900,
     6, 1, 1,
     TRUE, NOW() - INTERVAL '20 days', NOW()),
    -- Bronze new user
    (gen_random_uuid(), '+2348025000000', 'NXS-DEMO-03', 'Kano', 'BRONZE',
     1,  NOW() + INTERVAL '2 days', 0,
     5000000, NOW() - INTERVAL '5 days',
     '', FALSE,
     'DEMO03REF', 'unverified',
     50, 0, 150,
     2, 0, 0,
     TRUE, NOW() - INTERVAL '7 days', NOW()),
    -- Bronze streak-lapsed user
    (gen_random_uuid(), '+2348029000000', 'NXS-DEMO-05', 'Enugu', 'BRONZE',
     0,  NULL, 0,
     2000000, NOW() - INTERVAL '14 days',
     '', FALSE,
     'DEMO05REF', 'unverified',
     0, 0, 60,
     1, 0, 0,
     TRUE, NOW() - INTERVAL '45 days', NOW())
ON CONFLICT (phone_number) DO UPDATE SET
    tier                  = EXCLUDED.tier,
    total_points          = EXCLUDED.total_points,
    lifetime_points       = EXCLUDED.lifetime_points,
    total_recharge_amount = EXCLUDED.total_recharge_amount,
    last_recharge_at      = EXCLUDED.last_recharge_at,
    spin_credits          = EXCLUDED.spin_credits,
    stamps_count          = EXCLUDED.stamps_count,
    state                 = EXCLUDED.state,
    kyc_status            = EXCLUDED.kyc_status,
    is_active             = TRUE,
    updated_at            = NOW();
