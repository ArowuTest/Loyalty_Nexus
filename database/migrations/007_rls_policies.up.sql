-- 007_rls_policies.sql
-- Purpose: Security hardening for multi-tenant and subscriber data protection.
-- NOTE: RLS policies using the 'authenticated' role are skipped here because
-- that role is Supabase-specific. Application-level auth is handled via JWT
-- middleware in the Go API. This migration is kept as a no-op placeholder
-- to preserve migration numbering.

-- Enable RLS on tables that exist at this point in the migration sequence.
-- Policies use a safe DO block to avoid errors if role/table doesn't exist.
DO $$
BEGIN
    -- Enable RLS on users (exists from migration 002)
    ALTER TABLE users ENABLE ROW LEVEL SECURITY;

    -- Enable RLS on transactions (exists from migration 002)
    ALTER TABLE transactions ENABLE ROW LEVEL SECURITY;

    -- Enable RLS on program_configs (exists from migration 001)
    ALTER TABLE program_configs ENABLE ROW LEVEL SECURITY;

EXCEPTION WHEN OTHERS THEN
    -- If any statement fails (e.g. insufficient privilege), continue silently.
    NULL;
END $$;

-- Note: CREATE POLICY statements requiring 'authenticated' role are omitted.
-- Access control is enforced at the API layer via JWT middleware.
