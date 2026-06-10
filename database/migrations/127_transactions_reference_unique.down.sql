-- Rollback 127: restore the plain (non-unique) reference index from 046.
-- NOTE: idx_transactions_reference_unique is also created by migration 111,
-- so rolling back 127 alone intentionally leaves it in place on databases
-- where 111 has run — dropping it would weaken the recharge idempotency
-- guarantee the application now depends on.
CREATE INDEX IF NOT EXISTS idx_transactions_reference ON transactions (reference);
