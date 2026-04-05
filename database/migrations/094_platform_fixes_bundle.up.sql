-- Migration 094: Platform fixes bundle (v3 — drop unsafe phone normalisation)
--
-- Phone normalisation removed: the Go code already handles both +234xxx and
-- 234xxx formats via phoneVariants(). Deleting users with transactions is not
-- safe due to FK constraints. This step was cosmetic and is not needed.

-- ── FIX 1: Clean up duplicate / inactive legacy spin prizes ─────────────────
DELETE FROM prize_pool
WHERE is_active = false
  AND (prize_type, base_value) IN (
      SELECT prize_type, base_value FROM prize_pool WHERE is_active = true
  );
DELETE FROM prize_pool WHERE prize_type = 'bonus_points';
DELETE FROM prize_pool WHERE prize_type = 'studio_credits';

-- ── FIX 2: Zero stale users.spin_credits (wallets is single source of truth) ─
UPDATE users SET spin_credits = 0 WHERE spin_credits != 0;

-- ── FIX 3: Add draw_entries_today to wallets ─────────────────────────────────
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS draw_entries_today INTEGER NOT NULL DEFAULT 0;
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS draw_entries_date  DATE;

-- ── FIX 4: Resolve March 2026 Regional War ───────────────────────────────────
UPDATE regional_wars
SET status = 'COMPLETED', updated_at = NOW()
WHERE period = '2026-03' AND status = 'ACTIVE';

-- ── FIX 5: Assign states to test users ───────────────────────────────────────
UPDATE users SET state = 'Lagos'  WHERE phone_number IN ('2348027000000', '+2348027000000');
UPDATE users SET state = 'Abuja'  WHERE phone_number IN ('2348020000000', '+2348020000000');
UPDATE users SET state = 'Kano'   WHERE phone_number IN ('2348023000000', '+2348023000000');
UPDATE users SET state = 'Rivers' WHERE phone_number IN ('2348025000000', '+2348025000000');
UPDATE users SET state = 'Ogun'   WHERE phone_number IN ('2348029000000', '+2348029000000');
