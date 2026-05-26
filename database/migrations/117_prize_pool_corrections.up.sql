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

-- BUG-017: normalise active prize weights to sum to exactly 100.00.
-- Existing weights are stored on win_probability_weight in basis points style numbers
-- (e.g. 4050 = 40.50%). Convert active rows to percentage values and adjust one
-- "try again" slot so the final total is exactly 100.00.
DO $$
DECLARE
  current_sum NUMERIC(10,2);
  target_id UUID;
  adjustment NUMERIC(10,2);
BEGIN
  UPDATE prize_pool
  SET    win_probability_weight = ROUND(win_probability_weight / 100.0, 2)
  WHERE  is_active = TRUE
  AND    win_probability_weight > 100;

  SELECT COALESCE(SUM(win_probability_weight), 0)
  INTO   current_sum
  FROM   prize_pool
  WHERE  is_active = TRUE;

  IF current_sum <> 100.00 THEN
    SELECT id
    INTO   target_id
    FROM   prize_pool
    WHERE  is_active = TRUE
    ORDER BY CASE WHEN LOWER(prize_type) = 'try_again' THEN 0 ELSE 1 END,
             win_probability_weight DESC,
             id ASC
    LIMIT 1;

    adjustment := ROUND(100.00 - current_sum, 2);

    UPDATE prize_pool
    SET    win_probability_weight = ROUND(win_probability_weight + adjustment, 2)
    WHERE  id = target_id;
  END IF;
END $$;
