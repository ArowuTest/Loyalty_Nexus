-- Migration 077: Add updated_at column to spin_results
-- The trg_set_updated_at trigger (added by migration 020) references NEW.updated_at
-- but the spin_results table was never given this column, causing SQLSTATE 42703
-- on any UPDATE (e.g., prize claims). This migration adds the missing column.

ALTER TABLE spin_results
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Back-fill existing rows
UPDATE spin_results SET updated_at = created_at WHERE updated_at = NOW() AND created_at < NOW();
