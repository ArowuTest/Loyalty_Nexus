-- 076_sync_wallet_spin_credits.down.sql
-- No rollback needed: this migration only updates data to a more correct state.
-- Rolling back would require knowing the previous spin_credits values, which
-- are not stored. The data change is safe to keep on rollback.
SELECT 1;
