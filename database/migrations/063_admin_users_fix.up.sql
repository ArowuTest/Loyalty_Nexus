-- Migration 063: Fix admin_users table and guarantee super_admin exists.
--
-- Two possible DB states exist depending on which migration created admin_users:
--
--   Path A (migration 019 ran first):
--     Columns: id, username TEXT NOT NULL UNIQUE, password_hash, role TEXT, created_at
--     No email, no full_name, no is_active, no last_login_at, no updated_at
--
--   Path B (migration 052 ran first, or 019 never ran):
--     Columns: id, email TEXT NOT NULL UNIQUE, password_hash, full_name, role admin_role,
--              is_active, last_login_at, created_at, updated_at
--     No username
--
-- This migration normalises both paths to the canonical schema.

-- Step 1: Add missing columns (safe to re-run — IF NOT EXISTS)
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS email         TEXT;
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS full_name     TEXT NOT NULL DEFAULT '';
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS is_active     BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Step 2: Relax username NOT NULL constraint (Path A only — no-op on Path B)
-- On Path A, username is NOT NULL UNIQUE. We must allow NULL before inserting
-- new admin rows by email only (username is a legacy column from migration 019).
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'admin_users'
          AND column_name = 'username'
          AND is_nullable = 'NO'
    ) THEN
        ALTER TABLE admin_users ALTER COLUMN username DROP NOT NULL;
        RAISE NOTICE 'migration 063: dropped NOT NULL on username (Path A DB)';
    END IF;
END $$;

-- Step 3: Add unique constraint on email (ignore if already exists)
DO $$ BEGIN
    ALTER TABLE admin_users ADD CONSTRAINT admin_users_email_key UNIQUE (email);
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- Step 4: Widen role column to accept any text value
ALTER TABLE admin_users DROP CONSTRAINT IF EXISTS admin_users_role_check;
DO $$ BEGIN
    ALTER TABLE admin_users ALTER COLUMN role TYPE TEXT USING role::TEXT;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;
DO $$ BEGIN
    ALTER TABLE admin_users ALTER COLUMN role SET DEFAULT 'super_admin';
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- Step 5: Ensure the super_admin record exists (insert by email)
INSERT INTO admin_users (id, email, password_hash, full_name, role, is_active, created_at, updated_at)
SELECT
    gen_random_uuid(),
    'admin@loyaltynexus.ng',
    '$2b$10$9U6kXVOrcNk11Dg2F38OhuvTtrmbgAIHpzUFxSnZ.L0NCyyuM/Gim',
    'Platform Admin',
    'super_admin',
    TRUE,
    NOW(),
    NOW()
WHERE NOT EXISTS (SELECT 1 FROM admin_users WHERE email = 'admin@loyaltynexus.ng');

-- Step 6: Reset the password hash and role to known-good values
UPDATE admin_users
SET password_hash = '$2b$10$9U6kXVOrcNk11Dg2F38OhuvTtrmbgAIHpzUFxSnZ.L0NCyyuM/Gim',
    role          = 'super_admin',
    is_active     = TRUE,
    updated_at    = NOW()
WHERE email = 'admin@loyaltynexus.ng';
