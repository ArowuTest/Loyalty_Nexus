-- Migration 094: Platform fixes bundle (v2 — safe phone normalisation)

-- ── FIX 1: Clean up duplicate / inactive legacy spin prizes ─────────────────
DELETE FROM prize_pool
WHERE is_active = false
  AND (prize_type, base_value) IN (
      SELECT prize_type, base_value FROM prize_pool WHERE is_active = true
  );
DELETE FROM prize_pool WHERE prize_type = 'bonus_points';
DELETE FROM prize_pool WHERE prize_type = 'studio_credits';

-- ── FIX 2: Zero stale users.spin_credits ────────────────────────────────────
UPDATE users SET spin_credits = 0 WHERE spin_credits != 0;

-- ── FIX 3: Safe phone normalisation ─────────────────────────────────────────
-- Some +234xxx rows are true duplicates of existing 234xxx rows.
-- Step A: delete the +prefix duplicate (the 234xxx canonical row already exists)
DELETE FROM users a
USING users b
WHERE a.phone_number = '+' || b.phone_number
  AND b.phone_number NOT LIKE '+%';

-- Step B: rename any remaining +prefix rows that have no canonical counterpart
UPDATE users
SET phone_number = SUBSTRING(phone_number FROM 2)
WHERE phone_number LIKE '+%';

-- ── FIX 4: Add draw_entries_today to wallets ─────────────────────────────────
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS draw_entries_today INTEGER NOT NULL DEFAULT 0;
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS draw_entries_date  DATE;

-- ── FIX 5: Resolve March 2026 Regional War ───────────────────────────────────
UPDATE regional_wars
SET status = 'COMPLETED', updated_at = NOW()
WHERE period = '2026-03' AND status = 'ACTIVE';

-- Assign states to test users (use only the canonical 234xxx format)
UPDATE users SET state = 'Lagos'  WHERE phone_number = '2348027000000';
UPDATE users SET state = 'Abuja'  WHERE phone_number = '2348020000000';
UPDATE users SET state = 'Kano'   WHERE phone_number = '2348023000000';
UPDATE users SET state = 'Rivers' WHERE phone_number = '2348025000000';
UPDATE users SET state = 'Ogun'   WHERE phone_number = '2348029000000';
