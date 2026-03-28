-- Migration 046: Add spin_delta and reference columns to transactions
--
-- The Transaction entity has always had SpinDelta and Reference fields
-- but they were never added to the DB schema. This migration adds them
-- with safe defaults so existing rows are not affected.

ALTER TABLE transactions
  ADD COLUMN IF NOT EXISTS spin_delta  INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS reference   TEXT    NOT NULL DEFAULT '';

-- Index on reference for idempotency lookups
CREATE INDEX IF NOT EXISTS idx_transactions_reference ON transactions (reference)
  WHERE reference <> '';

-- Index on type + reference for duplicate detection
CREATE INDEX IF NOT EXISTS idx_transactions_type_reference ON transactions (type, reference)
  WHERE reference <> '';

COMMENT ON COLUMN transactions.spin_delta IS
  'Number of spin credits added (+) or consumed (-) by this transaction. 0 for non-spin transactions.';

COMMENT ON COLUMN transactions.reference IS
  'External reference ID (e.g. MTN transaction ref, Paystack ref). Used for idempotency checks.';
