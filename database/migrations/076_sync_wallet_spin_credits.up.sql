-- 076_sync_wallet_spin_credits.up.sql
--
-- Backfill wallets.spin_credits from users.spin_credits for any wallet rows
-- that were created with spin_credits = 0 while the user already had credits
-- in the users table (e.g., wallets created by the FirstOrCreate fix in the
-- bonus_pulse_service before the seed-on-create fix was applied).
--
-- This migration is fully idempotent: it only updates rows where the wallet
-- has fewer credits than the user, so running it multiple times is safe.

UPDATE wallets w
SET    spin_credits = u.spin_credits
FROM   users u
WHERE  w.user_id      = u.id
  AND  u.spin_credits > w.spin_credits;
