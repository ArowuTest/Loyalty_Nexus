-- ============================================================
-- Migration 109: Add reward fields to recharges table
-- ============================================================
-- These columns were added to the Go struct in vtu_recharge_service.go
-- to support the in-page success banner: points earned, draw entries,
-- and spin wheel eligibility. Without them the INSERT fails with
-- "column does not exist" (SQLSTATE 42703).

ALTER TABLE recharges
    ADD COLUMN IF NOT EXISTS points_earned  BIGINT  NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS draw_entries   INT     NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS spin_eligible  BOOLEAN NOT NULL DEFAULT FALSE;
