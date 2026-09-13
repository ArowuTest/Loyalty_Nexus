-- Migration 117: Prize pool corrections
-- BUG-017/018/019/020
-- Make prize data internally consistent without relying on admin manual cleanup.

-- BUG-018: ensure airtime prize names match stored kobo values.
UPDATE prize_pool
SET    name = '₦' || TRIM(TO_CHAR(base_value / 100.0, 'FM999999990.##')) || ' Airtime'
WHERE  LOWER(prize_type) = 'airtime'
AND    is_active = TRUE
AND    base_value > 0
AND    name <> '₦' || TRIM(TO_CHAR(base_value / 100.0, 'FM999999990.##')) || ' Airtime';

-- BUG-020: correct malformed 10MB prize units if a bad seed/update wrote bytes or other inflated values.
UPDATE prize_pool
SET    base_value = 10
WHERE  LOWER(prize_type) = 'data_bundle'
AND    LOWER(name) LIKE '%10mb%'
AND    base_value NOT BETWEEN 9 AND 11;

-- BUG-019: deactivate duplicate active prize slots by case-insensitive name, keeping the earliest UUID.
UPDATE prize_pool
SET    is_active = FALSE
WHERE  id IN (
  SELECT id
  FROM (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY LOWER(name) ORDER BY id ASC) AS rn
    FROM prize_pool
    WHERE is_active = TRUE
  ) dedup
  WHERE rn > 1
);

-- Fresh-install history could contain both the original phase-8 uncoded
-- wheel seed and the later Loyalty Nexus coded wheel seed. When the coded
-- wheel is present, retire only those known legacy seed rows before validating
-- probabilities. This is catalogue cleanup, not a permanent wheel definition:
-- Admin remains authoritative after migration.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM prize_pool
    WHERE prize_code = 'NONE' AND LOWER(prize_type) = 'try_again'
  ) THEN
    UPDATE prize_pool
    SET is_active = FALSE,
        updated_at = NOW()
    WHERE is_active = TRUE
      AND COALESCE(prize_code, '') = ''
      AND (
        (LOWER(prize_type) = 'try_again' AND base_value = 0)
        OR (LOWER(prize_type) = 'pulse_points' AND base_value IN (5, 10)
            AND name IN ('+5 Pulse Points', '+10 Pulse Points'))
        OR (LOWER(prize_type) = 'data_bundle' AND base_value IN (10, 25)
            AND name IN ('10MB Data', '25MB Data'))
        OR (LOWER(prize_type) = 'airtime' AND base_value IN (50, 100, 200))
      );
  END IF;
END $$;

-- BUG-017: normalise active prize weights to sum to exactly 100.00.
-- Existing weights are stored on win_probability_weight in basis points style numbers
-- (e.g. 4050 = 40.50%). Convert active rows to percentage values and adjust one
-- "try again" slot so the final total is exactly 100.00.
DO $$
DECLARE
  current_sum NUMERIC(18,6);
  rounded_sum NUMERIC(18,2);
  target_id   UUID;
  residual    NUMERIC(18,2);
BEGIN
  -- Convert obvious basis-point values first.
  UPDATE prize_pool
  SET    win_probability_weight = ROUND(win_probability_weight / 100.0, 2)
  WHERE  is_active = TRUE
  AND    win_probability_weight > 100;

  SELECT COALESCE(SUM(win_probability_weight), 0)
  INTO   current_sum
  FROM   prize_pool
  WHERE  is_active = TRUE;

  IF current_sum > 0 AND current_sum <> 100.00 THEN
    -- Preserve relative probabilities while bringing the active pool to 100%.
    UPDATE prize_pool
    SET    win_probability_weight =
             ROUND((win_probability_weight / current_sum) * 100.0, 2)
    WHERE  is_active = TRUE;

    SELECT COALESCE(SUM(win_probability_weight), 0)
    INTO   rounded_sum
    FROM   prize_pool
    WHERE  is_active = TRUE;

    residual := ROUND(100.00 - rounded_sum, 2);

    IF residual <> 0 THEN
      SELECT id INTO target_id
      FROM prize_pool
      WHERE is_active = TRUE
      ORDER BY win_probability_weight DESC, id ASC
      LIMIT 1;

      UPDATE prize_pool
      SET win_probability_weight = ROUND(win_probability_weight + residual, 2)
      WHERE id = target_id;
    END IF;
  ELSIF current_sum = 0 THEN
    -- Defensive recovery for a pathological all-zero active pool.
    SELECT id INTO target_id
    FROM prize_pool
    WHERE is_active = TRUE
    ORDER BY id ASC
    LIMIT 1;

    IF target_id IS NOT NULL THEN
      UPDATE prize_pool SET win_probability_weight = 100.00 WHERE id = target_id;
    END IF;
  END IF;
END $$;
