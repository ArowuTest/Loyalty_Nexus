-- Migration 052: Admin Users with RBAC (email + password, role-based access)
-- Replaces the placeholder AdminUser with a full production-ready admin identity system.

CREATE TYPE admin_role AS ENUM (
  'super_admin',    -- Full platform access
  'finance',        -- Approve claims, view financials, adjust points
  'operations',     -- Manage draws, notifications, users, wars
  'content'         -- Manage studio tools, prizes, config
);

CREATE TABLE admin_users (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email           TEXT NOT NULL UNIQUE,
  password_hash   TEXT NOT NULL,
  full_name       TEXT NOT NULL DEFAULT '',
  role            admin_role NOT NULL DEFAULT 'operations',
  is_active       BOOLEAN NOT NULL DEFAULT TRUE,
  last_login_at   TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_users_email ON admin_users(email);

-- Seed a default super_admin (password will be set via ADMIN_SEED_PASSWORD env var at startup,
-- or use the admin CLI tool. Hash shown here is bcrypt of 'ChangeMe123!' — MUST be rotated.)
-- Actual seeding is done by the application on first startup if no admin exists.
