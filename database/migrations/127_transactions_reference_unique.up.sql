-- Migration 127: Ensure unique Paystack/MNO references at the DB level
--
-- NOTE: Migration 111 already creates idx_transactions_reference_unique on
-- databases migrated past it.  This migration re-asserts the index
-- idempotently (IF NOT EXISTS) so that:
--   (a) environments that predate 111 gain the constraint, and
--   (b) the constraint is documented at the point the application code
--       (recharge_service.go isDuplicateKeyError) started depending on it.
--
-- WHY A UNIQUE INDEX
-- ------------------
-- The application-layer FindByReference() idempotency check is not atomic
-- with the insert: two concurrent Paystack webhook retries for the same
-- reference can both read nil and both attempt to commit, double-awarding
-- points and spin credits.  The partial unique index makes the second
-- INSERT fail with SQLSTATE 23505, which recharge_service.go maps to
-- ErrDuplicateRecharge.
--
-- TRANSACTION SAFETY
-- ------------------
-- CONCURRENTLY is deliberately NOT used: golang-migrate executes each
-- migration file inside a transaction-safe context, and
-- CREATE INDEX CONCURRENTLY cannot run inside a transaction block
-- (same rule followed by migration 111).  A plain CREATE INDEX takes a
-- brief write lock on transactions — acceptable at current table size.
-- If this ever needs to run on a very large live table, run the
-- CONCURRENTLY variant manually via psql instead.
--
-- If duplicate non-empty references already exist, this migration fails.
-- Find them first with:
--   SELECT reference, COUNT(*) FROM transactions
--   WHERE reference IS NOT NULL AND reference != ''
--   GROUP BY reference HAVING COUNT(*) > 1;

-- Drop the old plain reference index from migration 046 if present
-- (superseded by the unique index below; avoids duplicate-index bloat)
DROP INDEX IF EXISTS idx_transactions_reference;

-- The real constraint — partial unique index on non-empty references.
-- Predicate matches migration 111 exactly so IF NOT EXISTS is a clean no-op
-- on databases where 111 already ran.
CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_reference_unique
    ON transactions (reference)
    WHERE reference IS NOT NULL AND reference != '';

-- Composite lookup index for admin queries (also created by 111; idempotent)
CREATE INDEX IF NOT EXISTS idx_transactions_type_reference
    ON transactions (type, reference);
