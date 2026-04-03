-- Rollback migration 084
-- Note: Only drops columns that were ADDED by this migration.
-- Does NOT drop fulfillment_status if it was already present from migration 020.

DROP INDEX IF EXISTS idx_spin_results_fulfillment_status;
DROP INDEX IF EXISTS idx_fraud_events_unresolved;

-- We do not drop fulfillment_status or resolved as they may have been
-- created by migration 020 and are required by the application.
-- To fully roll back, restore from a backup taken before this migration.
