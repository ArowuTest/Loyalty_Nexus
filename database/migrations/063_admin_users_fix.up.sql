-- Migration 063: Fix admin_users table and guarantee super_admin exists.
-- 
-- History: migration 019 created admin_users with 'username' column.
-- Migration 052's CREATE TABLE IF NOT EXISTS was a no-op (table existed).
-- Result: live DB has username/password_hash/role but NO email column.
-- This migration adds all missing columns and seeds the admin.

-- Step 1: Add missing columns
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

-- Step 3: Widen role column to accept any text (it was a CHECK constraint)
ALTER TABLE admin_users DROP CONSTRAINT IF EXISTS admin_users_role_check;
ALTER TABLE admin_users ALTER COLUMN role SET DEFAULT 'super_admin';

-- Step 4: Upsert the super_admin by username first, then set email
-- (The old row from migration 019 has username='admin_nexus', no email)
-- Update it to become our proper admin
UPDATE admin_users 
SET email     = 'admin@loyaltynexus.ng',
    full_name = 'Platform Admin',
    role      = 'super_admin',
    is_active = TRUE,
    updated_at = NOW()
WHERE email IS NULL OR email = '';

-- Step 5: If no rows updated (table was empty), do a fresh insert
INSERT INTO admin_users (id, username, email, password_hash, full_name, role, is_active, created_at, updated_at)
SELECT
    gen_random_uuid(),
    'admin_nexus',
    'admin@loyaltynexus.ng',
    '$2b$10$9U6kXVOrcNk11Dg2F38OhuvTtrmbgAIHpzUFxSnZ.L0NCyyuM/Gim',
    'Platform Admin',
    'super_admin',
    TRUE,
    NOW(),
    NOW()
WHERE NOT EXISTS (SELECT 1 FROM admin_users WHERE email = 'admin@loyaltynexus.ng');

-- Step 6: Make sure the password hash is correct (reset it)
UPDATE admin_users
SET password_hash = '$2b$10$9U6kXVOrcNk11Dg2F38OhuvTtrmbgAIHpzUFxSnZ.L0NCyyuM/Gim',
    role          = 'super_admin',
    is_active     = TRUE,
    updated_at    = NOW()
WHERE email = 'admin@loyaltynexus.ng';
