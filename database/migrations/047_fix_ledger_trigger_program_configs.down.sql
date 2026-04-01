-- Rollback migration 047: Restore original trigger function (no-op rollback)
-- The original trigger referenced a non-existent table and caused errors.
-- Rolling back would re-introduce the broken trigger, so this is a no-op.
-- The trigger function remains as fixed by migration 047.
SELECT 1;
