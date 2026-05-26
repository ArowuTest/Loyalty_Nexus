-- Migration 118: Consolidate duplicate "Try Again" spin prize entries
-- The auditor found 3 separate "Try Again" entries combining ~48% no-win rate.
-- We keep 1 entry with the combined weight and delete the rest.

DO $$
DECLARE
  keep_id   UUID;
  total_wt  NUMERIC;
BEGIN
  -- Sum all try-again weights
  SELECT COALESCE(SUM(win_probability_weight), 0)
  INTO   total_wt
  FROM   prize_pool
  WHERE  LOWER(prize_type) = 'try_again' AND is_active = true;

  -- Pick the one to keep (lowest created_at)
  SELECT id INTO keep_id
  FROM   prize_pool
  WHERE  LOWER(prize_type) = 'try_again' AND is_active = true
  ORDER  BY created_at ASC
  LIMIT  1;

  IF keep_id IS NOT NULL THEN
    -- Update the keeper with the combined weight (cap at 30.00 = 3000 basis points to avoid dominating the pool)
    UPDATE prize_pool
    SET    win_probability_weight = LEAST(total_wt, 30.00),
           name = 'Try Again',
           updated_at = NOW()
    WHERE  id = keep_id;

    -- Deactivate all other try-again entries
    UPDATE prize_pool
    SET    is_active = false,
           updated_at = NOW()
    WHERE  LOWER(prize_type) = 'try_again'
      AND  is_active = true
      AND  id <> keep_id;
  END IF;

  -- Re-normalise remaining active prize weights to sum to 100.00 percent
  WITH active AS (
    SELECT id, win_probability_weight,
           SUM(win_probability_weight) OVER () AS total
    FROM   prize_pool
    WHERE  is_active = true
  )
  UPDATE prize_pool pp
  SET    win_probability_weight = ROUND((a.win_probability_weight::NUMERIC / a.total) * 100, 2),
         updated_at = NOW()
  FROM   active a
  WHERE  pp.id = a.id;
END $$;
