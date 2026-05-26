-- Rollback for migration 120 (catch-all re-application)
-- Because every statement in the .up was a data correction (not schema change),
-- there is no safe automated rollback — reverting would require knowing the
-- pre-migration data state. The correct recovery is to re-run the up.sql.
-- This file intentionally left as a no-op to satisfy golang-migrate's
-- requirement for a paired down file.
SELECT 1; -- no-op
