-- 019_user_profile_expansion.sql
-- Purpose: Support for Nigerian State capture and Admin Roles (REQ-1.5, User Class 2.2).

ALTER TABLE users 
ADD COLUMN IF NOT EXISTS state TEXT;

-- Index for Regional Wars lookups
CREATE INDEX IF NOT EXISTS idx_users_state ON users(state);

-- Admin Users and Roles
CREATE TABLE IF NOT EXISTS admin_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT CHECK (role IN ('platform_admin', 'mno_executive')) DEFAULT 'platform_admin',
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Seed Initial Admin (Platform Admin)
INSERT INTO admin_users (username, password_hash, role) VALUES 
('admin_nexus', 'placeholder_hash', 'platform_admin')
ON CONFLICT (username) DO NOTHING;
