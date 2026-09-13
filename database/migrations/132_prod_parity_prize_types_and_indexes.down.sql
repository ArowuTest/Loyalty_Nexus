-- No-op down, consistent with the repo's other corrective migrations (120/121).
--
-- This is a production-parity migration: the constraint and indexes it asserts
-- may PRE-DATE it on fresh/CI databases (they were created by the edited 112/125).
-- A destructive down would therefore drop objects this migration did not create,
-- breaking those databases. If a genuine rollback of the widened prize-type CHECK
-- is ever required, do it as a new, explicitly-scoped migration after confirming
-- no 'physical'/'goods' rows exist in prize_pool.
SELECT 1;
