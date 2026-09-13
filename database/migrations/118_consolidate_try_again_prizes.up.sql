-- Migration 118: Clean historical duplicate Spin & Win seed rows.
--
-- Loyalty Nexus wheel configuration is Admin-authored. This migration only
-- reconciles known historical seed overlap; it does NOT impose a permanent
-- catalogue, a fixed no-win percentage, or a RechargeMax-specific policy.
--
-- Final bootstrap invariant:
--   * known old seed duplicates are inactive
--   * duplicate active try_again rows are consolidated
--   * all remaining active probabilities sum to exactly 100.00%
--   * Admin may subsequently publish any active prize mix whose probabilities
--     sum to exactly 100.00%

DO $$
DECLARE
  keep_id       UUID;
  total_try     NUMERIC(18,6);
  active_total  NUMERIC(18,6);
  rounded_total NUMERIC(18,2);
  residual      NUMERIC(18,2);
  residual_id   UUID;
BEGIN
  -- Retire only exact, known historical seed rows from the early wheel.
  -- Do not deactivate arbitrary Admin-created uncoded prizes.
  UPDATE prize_pool
  SET is_active = FALSE,
      updated_at = NOW()
  WHERE is_active = TRUE
    AND COALESCE(prize_code, '') = ''
    AND (
      (LOWER(prize_type) = 'pulse_points' AND base_value IN (5, 10)
       AND name IN ('+5 Pulse Points', '+10 Pulse Points'))
      OR (LOWER(prize_type) = 'data_bundle' AND base_value IN (10, 25)
          AND name IN ('10MB Data', '25MB Data'))
      OR (LOWER(prize_type) = 'airtime' AND base_value IN (50, 100, 200)
          AND name IN ('₦0.50 Airtime', '₦1 Airtime', '₦2 Airtime'))
    );

  -- Consolidate duplicate no-win seed rows while preserving their combined
  -- pre-normalization relative weight.
  SELECT id INTO keep_id
  FROM prize_pool
  WHERE LOWER(prize_type) = 'try_again'
    AND is_active = TRUE
  ORDER BY CASE WHEN prize_code = 'NONE' THEN 0 ELSE 1 END,
           sort_order ASC,
           id ASC
  LIMIT 1;

  IF keep_id IS NOT NULL THEN
    SELECT COALESCE(SUM(win_probability_weight), 0)
    INTO total_try
    FROM prize_pool
    WHERE LOWER(prize_type) = 'try_again'
      AND is_active = TRUE;

    UPDATE prize_pool
    SET win_probability_weight = total_try,
        is_no_win = TRUE,
        updated_at = NOW()
    WHERE id = keep_id;

    UPDATE prize_pool
    SET is_active = FALSE,
        updated_at = NOW()
    WHERE LOWER(prize_type) = 'try_again'
      AND is_active = TRUE
      AND id <> keep_id;
  END IF;

  SELECT COALESCE(SUM(win_probability_weight), 0)
  INTO active_total
  FROM prize_pool
  WHERE is_active = TRUE;

  IF active_total <= 0 THEN
    RAISE EXCEPTION 'Migration 118 failed: no positive active prize probability remains';
  END IF;

  -- Preserve the relative weights of the legitimate remaining Admin/default
  -- rows while restoring the published-wheel 100% invariant.
  UPDATE prize_pool
  SET win_probability_weight = ROUND((win_probability_weight / active_total) * 100.00, 2),
      updated_at = NOW()
  WHERE is_active = TRUE;

  SELECT ROUND(COALESCE(SUM(win_probability_weight), 0), 2)
  INTO rounded_total
  FROM prize_pool
  WHERE is_active = TRUE;

  residual := ROUND(100.00 - rounded_total, 2);
  IF residual <> 0 THEN
    SELECT id INTO residual_id
    FROM prize_pool
    WHERE is_active = TRUE
    ORDER BY win_probability_weight DESC, id ASC
    LIMIT 1;

    UPDATE prize_pool
    SET win_probability_weight = ROUND(win_probability_weight + residual, 2),
        updated_at = NOW()
    WHERE id = residual_id;
  END IF;

  IF EXISTS (
    SELECT 1
    FROM (
      SELECT ROUND(COALESCE(SUM(win_probability_weight), 0), 2) AS total,
             COUNT(*) FILTER (WHERE win_probability_weight <= 0) AS non_positive
      FROM prize_pool
      WHERE is_active = TRUE
    ) s
    WHERE s.total <> 100.00 OR s.non_positive > 0
  ) THEN
    RAISE EXCEPTION 'Migration 118 failed: active prize probabilities are not a valid 100%% wheel';
  END IF;
END $$;
