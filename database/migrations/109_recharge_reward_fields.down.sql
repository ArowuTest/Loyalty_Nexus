-- ============================================================
-- Migration 109 rollback: Remove reward fields from recharges
-- ============================================================

ALTER TABLE recharges
    DROP COLUMN IF EXISTS points_earned,
    DROP COLUMN IF EXISTS draw_entries,
    DROP COLUMN IF EXISTS spin_eligible;
