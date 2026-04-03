-- Migration 089: Add display_name and email to users table
-- These fields allow users to personalise their profile beyond phone number.

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS display_name TEXT        NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS email        TEXT        NOT NULL DEFAULT '';

-- Partial unique index: only enforce uniqueness when email is non-empty
CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique
  ON users (email)
  WHERE email <> '';
