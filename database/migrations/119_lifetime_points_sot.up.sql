-- Migration 119: Single source of truth for lifetime_points
-- The users.lifetime_points column was a legacy seed-only column that drifted
-- from wallets.lifetime_points (the actively-updated value). All writers go to
-- wallets.lifetime_points; this migration syncs users.lifetime_points to match,
-- and adds a NOTICE-level note. The column is retained (not dropped) for zero-
-- downtime rollback safety; the user entity now hides it from JSON output.

-- Backfill users.lifetime_points from wallets.lifetime_points
UPDATE users u
SET    lifetime_points = COALESCE(w.lifetime_points, 0),
       updated_at      = NOW()
FROM   wallets w
WHERE  w.user_id = u.id
  AND  u.lifetime_points IS DISTINCT FROM w.lifetime_points;

-- Document the deprecation at the column level
COMMENT ON COLUMN users.lifetime_points
  IS 'DEPRECATED: read from wallets.lifetime_points instead. Kept for rollback safety; backfilled in migration 119.';
