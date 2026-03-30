-- Migration 065: Fix admin role to super_admin and ensure fraud_events has all needed columns
-- This migration ensures the seeded admin user has super_admin role
-- and that fraud_events table has all required columns.

-- Fix admin role: set all existing admin users to super_admin
-- (safe because this is a fresh deployment with only seeded admins)
UPDATE admin_users SET role = 'super_admin' WHERE role != 'super_admin';

-- Ensure fraud_events has all required columns (it may have been created by migration 020
-- with a slightly different schema than what the handler expects)
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ;
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS resolved_by TEXT;
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS notes TEXT NOT NULL DEFAULT '';

-- Verify
SELECT 'admin_super_admin_count' as check_name, COUNT(*)::text as result FROM admin_users WHERE role = 'super_admin'
UNION ALL
SELECT 'fraud_events_columns', string_agg(column_name, ', ' ORDER BY ordinal_position) FROM information_schema.columns WHERE table_name = 'fraud_events';
