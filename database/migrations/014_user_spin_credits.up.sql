-- 014_user_spin_credits.sql
-- Purpose: Formalize the two-pool ledger by adding spin_credits to users.

ALTER TABLE users 
ADD COLUMN spin_credits INTEGER DEFAULT 0;

-- Optional: Add a trigger or procedure to handle cumulative recharge -> spin credit logic
