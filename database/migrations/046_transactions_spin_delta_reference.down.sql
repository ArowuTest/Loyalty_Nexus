-- Rollback migration 046: Remove spin_delta and reference columns from transactions
DROP INDEX IF EXISTS idx_transactions_reference;
DROP INDEX IF EXISTS idx_transactions_type_reference;
ALTER TABLE transactions
  DROP COLUMN IF EXISTS spin_delta,
  DROP COLUMN IF EXISTS reference;
