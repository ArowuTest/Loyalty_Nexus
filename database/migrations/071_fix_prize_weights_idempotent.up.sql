-- Migration 071: Idempotent corrective rescale for prize_pool weights
--
-- Context:
--   Migration 070 changed win_probability_weight from INTEGER (0–10000 basis-points)
--   to NUMERIC(5,2) (0–100.00 direct percentage). It used a USING clause to divide
--   existing rows by 100 at the time of the ALTER TABLE.
--
--   However, on databases where migration 070 ran BEFORE the full prize seed data
--   was present (e.g. a fresh deploy where 044/058 inserted rows after 070 ran),
--   or where the column was already NUMERIC before 070 (so the USING clause was
--   a no-op), some rows may still carry the old basis-point values (> 100).
--
--   This migration is fully idempotent: it only touches rows where
--   win_probability_weight > 100 (which is impossible on the 0–100 scale),
--   dividing them by 100 to bring them into the correct range.
--
--   Safe to re-run on any database state — rows already on the 0–100 scale
--   are untouched because their weight is ≤ 100.

UPDATE prize_pool
SET    win_probability_weight = ROUND(win_probability_weight / 100.0, 2)
WHERE  win_probability_weight > 100;

-- Verify: after this migration, no active prize should have weight > 100
-- (enforced by the CHECK constraint added in migration 070)
DO $$
DECLARE
    bad_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO bad_count
    FROM prize_pool
    WHERE win_probability_weight > 100;

    IF bad_count > 0 THEN
        RAISE EXCEPTION 'Migration 071 failed: % prize(s) still have weight > 100 after rescale', bad_count;
    END IF;
END $$;
