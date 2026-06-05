-- Rollback 127: restore plain index, drop unique constraint
DROP INDEX IF EXISTS idx_transactions_reference_unique;
DROP INDEX IF EXISTS idx_transactions_type_reference;

CREATE INDEX IF NOT EXISTS idx_transactions_reference ON transactions (reference)
    WHERE reference <> '';

CREATE INDEX IF NOT EXISTS idx_transactions_type_reference ON transactions (type, reference)
    WHERE reference <> '';
