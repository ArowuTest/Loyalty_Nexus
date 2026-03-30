-- Migration 071 DOWN: No-op rollback
--
-- This migration corrected data that was in an inconsistent state.
-- Rolling it back would re-introduce incorrect weights, which is unsafe.
-- To roll back prize weight changes, use migration 070's down migration
-- which reverts the full column type and all seeded values.
SELECT 1;
