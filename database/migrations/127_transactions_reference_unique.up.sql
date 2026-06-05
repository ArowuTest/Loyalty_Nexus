-- Migration 127: Enforce unique Paystack/MNO references at the DB level
--
-- PROBLEM
-- -------
-- Migration 046 added a plain index on transactions.reference for lookup speed,
-- but NOT a UNIQUE constraint.  The application layer checks FindByReference()
-- before inserting, but that check is outside any transaction — two concurrent
-- Paystack webhook retries for the same reference can both find nil and both
-- commit, causing double point/spin-credit awards.
--
-- FIX
-- ---
-- A PARTIAL unique index enforces uniqueness only on non-empty references.
-- Empty-string references (non-Paystack ledger entries) are excluded so the
-- index doesn't interfere with existing rows that have no external reference.
--
-- The plain index from 046 is dropped first to avoid index bloat; the unique
-- index replaces it for both uniqueness enforcement and lookup speed.
--
-- CONCURRENCY BEHAVIOUR
-- ---------------------
-- On a duplicate insert:
--   - PostgreSQL raises ERROR 23505 (unique_violation)
--   - GORM returns an error that wraps the PG error
--   - recharge_service.go returns ErrDuplicateRecharge (already handled)
-- No application-layer change is needed — the existing error path is correct.
--
-- SAFE TO RUN ON LIVE DB
-- ----------------------
-- CREATE UNIQUE INDEX CONCURRENTLY does not take an AccessExclusiveLock;
-- it builds the index without blocking reads or writes.
-- DROP INDEX CONCURRENTLY likewise avoids locking.
-- If duplicate reference values already exist in the table the command will
-- fail — run the dedup query below first in that case:
--
--   SELECT reference, COUNT(*) FROM transactions
--   WHERE reference <> '' GROUP BY reference HAVING COUNT(*) > 1;

-- Remove the old plain index (replaced by the unique index below)
DROP INDEX IF EXISTS idx_transactions_reference;
DROP INDEX IF EXISTS idx_transactions_type_reference;

-- Enforce uniqueness on non-empty references — the real fix
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_transactions_reference_unique
    ON transactions (reference)
    WHERE reference <> '';

-- Restore the type+reference composite lookup index (non-unique, for admin queries)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_transactions_type_reference
    ON transactions (type, reference)
    WHERE reference <> '';
