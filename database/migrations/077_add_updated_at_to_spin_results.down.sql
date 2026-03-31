-- Revert migration 077: Remove updated_at column from spin_results
ALTER TABLE spin_results DROP COLUMN IF EXISTS updated_at;
