-- Migration 063: Fix admin_users table and guarantee super_admin exists.
-- 
-- The live DB was created by migration 052 which uses email as the primary
-- identifier (no username column). This migration adds any missing columns
-- and ensures the super_admin record exists.

-- Step 1: Add missing columns (safe to re-run — IF NOT EXISTS)
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS email         TEXT;
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS full_name     TEXT NOT NULL DEFAULT '';
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS is_active     BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Step 2: Add unique constraint on email (ignore if already exists)
DO $$ BEGIN
    ALTER TABLE admin_users ADD CONSTRAINT admin_users_email_key UNIQUE (email);
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- Step 3: Widen role column to accept any text (drop CHECK constraint if present)
ALTER TABLE admin_users DROP CONSTRAINT IF EXISTS admin_users_role_check;
DO $$ BEGIN
    ALTER TABLE admin_users ALTER COLUMN role SET DEFAULT 'super_admin';
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- Step 4: Ensure the super_admin record exists (upsert by email)
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

-- Step 5: Reset the password hash and role to known-good values
UPDATE admin_users
SET password_hash = '$2b$10$9U6kXVOrcNk11Dg2F38OhuvTtrmbgAIHpzUFxSnZ.L0NCyyuM/Gim',
    role          = 'super_admin',
    is_active     = TRUE,
    updated_at    = NOW()
WHERE email = 'admin@loyaltynexus.ng';
