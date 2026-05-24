-- Rollback for migration 121 (ghost-migration recovery)
-- This migration re-applied data corrections — there is no safe automated rollback.
-- Recovery: re-run the up.sql if the data state is uncertain.
SELECT 1; -- intentional no-op
