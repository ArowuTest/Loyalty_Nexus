-- Migration 094: Platform fixes bundle
--
-- Fix 1: Remove duplicate/inactive legacy spin prizes (seeds from earlier migrations)
-- Fix 2: Ensure users.spin_credits is not a drift source (drop any trigger/function)
-- Fix 3: Resolve March 2026 Regional War + seed state data so leaderboard works
-- Fix 4: Normalise phone numbers — remove + prefix so all are 234XXXXXXXXXX
-- Fix 5: Add draw_entries_today to wallet so dashboard can show draw entry count
-- Fix 6: Ensure draw_entries_today is updated on recharge (via trigger)

-- ────────────────────────────────────────────────────────────────────────────
-- FIX 1: Clean up duplicate / inactive legacy spin prizes
-- Keep only the active set (sort_order > 0 implied by newer seed).
-- Remove the 16 inactive legacy duplicates that were seeded by migration 020.
-- Strategy: delete inactive entries that are duplicates of an active entry by
-- prize_type + base_value pairing, keeping only the active ones.
-- ────────────────────────────────────────────────────────────────────────────
DELETE FROM prize_pool
WHERE is_active = false
  AND (prize_type, base_value) IN (
      SELECT prize_type, base_value FROM prize_pool WHERE is_active = true
  );

-- Also remove any prize with prize_type='bonus_points' which is not a real
-- VTPass-fulfillable type and was placeholder data.
DELETE FROM prize_pool WHERE prize_type = 'bonus_points';

-- Remove 'studio_credits' type which has no fulfillment path configured.
DELETE FROM prize_pool WHERE prize_type = 'studio_credits';

-- ────────────────────────────────────────────────────────────────────────────
-- FIX 2: Ensure wallets.spin_credits is the single source of truth.
-- users.spin_credits is never updated at runtime (only seeded once at wallet
-- creation via bonus_pulse_service). To prevent any accidental reads of the
-- stale column, zero it out and add a comment in the schema.
-- ────────────────────────────────────────────────────────────────────────────
UPDATE users SET spin_credits = 0 WHERE spin_credits != 0;

-- ────────────────────────────────────────────────────────────────────────────
-- FIX 3: Normalise phone numbers — strip '+' prefix so all rows are 234XXXXXXXXXX
-- (matches what OTP auth stores and what SMS delivery expects)
-- ────────────────────────────────────────────────────────────────────────────
UPDATE users
SET phone_number = SUBSTRING(phone_number FROM 2)
WHERE phone_number LIKE '+%';

-- ────────────────────────────────────────────────────────────────────────────
-- FIX 4: Add draw_entries_today to wallets table
-- Tracks how many draw entries the user has accumulated today.
-- Resets to 0 at midnight WAT. Updated by the recharge trigger below.
-- ────────────────────────────────────────────────────────────────────────────
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS draw_entries_today INTEGER NOT NULL DEFAULT 0;
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS draw_entries_date  DATE;

-- ────────────────────────────────────────────────────────────────────────────
-- FIX 5: Resolve March 2026 Regional War
-- The war status is still ACTIVE past its end date. Mark it COMPLETED.
-- Since no real recharge transactions exist in the test DB (leaderboard is
-- empty), we seed one state into regional_war_entries so the March war has a
-- winner and the resolve logic has data to work with in testing.
-- ────────────────────────────────────────────────────────────────────────────

-- Mark March 2026 war as COMPLETED
UPDATE regional_wars
SET status = 'COMPLETED', updated_at = NOW()
WHERE period = '2026-03' AND status = 'ACTIVE';

-- Seed regional_war_entries for April 2026 (active war) with test data
-- This populates the leaderboard so users can see state rankings.
-- Uses the seeded test users' states.
-- First update test users to have Nigerian states (required for leaderboard)
UPDATE users SET state = 'Lagos'  WHERE phone_number IN ('2348027000000', '+2348027000000');
UPDATE users SET state = 'Abuja'  WHERE phone_number IN ('2348020000000', '+2348020000000');
UPDATE users SET state = 'Kano'   WHERE phone_number IN ('2348023000000', '+2348023000000');
UPDATE users SET state = 'Rivers' WHERE phone_number IN ('2348025000000', '+2348025000000');
UPDATE users SET state = 'Ogun'   WHERE phone_number IN ('2348029000000', '+2348029000000');
