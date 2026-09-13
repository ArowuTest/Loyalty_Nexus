-- Reverses the append-only trigger only.
--
-- The generation_id foreign key is intentionally LEFT as ON DELETE SET NULL:
-- restoring CASCADE would re-enable the audit-erasing retention bug this
-- migration exists to fix, and every other FK on the table is already SET NULL.
DROP TRIGGER IF EXISTS trg_ai_generation_attempts_append_only ON ai_generation_attempts;
DROP FUNCTION IF EXISTS ai_generation_attempts_append_only();
