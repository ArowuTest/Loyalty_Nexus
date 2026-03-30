-- Migration 070: Rescale prize probability weights from 10,000 scale to 100.00 scale
-- Before: weights are integers summing to 10,000 (e.g. 4003 = 40.03%)
-- After:  weights are NUMERIC(5,2) summing to 100.00 (e.g. 40.03 = 40.03%)
-- This makes the admin UI intuitive: weight = percentage directly.
-- Also renames all "MoMo Cash" prizes to plain "Cash" (MoMo is the delivery mechanism, not the prize name).

-- Step 1: Change column type from INTEGER to NUMERIC(5,2)
ALTER TABLE prize_pool
    ALTER COLUMN win_probability_weight TYPE NUMERIC(5,2)
    USING ROUND(win_probability_weight::NUMERIC / 100.0, 2);

-- Step 2: Rescale all existing seeded prizes to the 100.00 scale
-- (The USING clause above handles existing rows automatically via division by 100)
-- But we need to ensure the specific seeded values are exact (no floating point drift)
UPDATE prize_pool SET win_probability_weight = 40.03 WHERE prize_code = 'NONE'    AND prize_type = 'try_again';
UPDATE prize_pool SET win_probability_weight = 25.00 WHERE prize_code = 'PTS10'   AND prize_type = 'pulse_points';
UPDATE prize_pool SET win_probability_weight = 15.00 WHERE prize_code = 'PTS25'   AND prize_type = 'pulse_points';
UPDATE prize_pool SET win_probability_weight =  8.00 WHERE prize_code = 'PTS50'   AND prize_type = 'pulse_points';
UPDATE prize_pool SET win_probability_weight =  5.00 WHERE prize_code = 'PTS100'  AND prize_type = 'pulse_points';
UPDATE prize_pool SET win_probability_weight =  3.00 WHERE prize_code = 'AIR50'   AND prize_type = 'airtime';
UPDATE prize_pool SET win_probability_weight =  1.50 WHERE prize_code = 'AIR100'  AND prize_type = 'airtime';
UPDATE prize_pool SET win_probability_weight =  0.75 WHERE prize_code = 'AIR200'  AND prize_type = 'airtime';
UPDATE prize_pool SET win_probability_weight =  0.50 WHERE prize_code = 'AIR500'  AND prize_type = 'airtime';
UPDATE prize_pool SET win_probability_weight =  0.25 WHERE prize_code = 'AIR1K'   AND prize_type = 'airtime';
UPDATE prize_pool SET win_probability_weight =  0.10 WHERE prize_code = 'AIR2K'   AND prize_type = 'airtime';
UPDATE prize_pool SET win_probability_weight =  0.30 WHERE prize_code = 'DATA500' AND prize_type = 'data_bundle';
UPDATE prize_pool SET win_probability_weight =  0.20 WHERE prize_code = 'DATA1GB' AND prize_type = 'data_bundle';
UPDATE prize_pool SET win_probability_weight =  0.10 WHERE prize_code = 'DATA2GB' AND prize_type = 'data_bundle';
UPDATE prize_pool SET win_probability_weight =  0.08 WHERE prize_code = 'CASH500' AND prize_type = 'momo_cash';
UPDATE prize_pool SET win_probability_weight =  0.10 WHERE prize_code = 'CASH1K'  AND prize_type = 'momo_cash';
UPDATE prize_pool SET win_probability_weight =  0.04 WHERE prize_code = 'CASH2K'  AND prize_type = 'momo_cash';
UPDATE prize_pool SET win_probability_weight =  0.03 WHERE prize_code = 'CASH5K'  AND prize_type = 'momo_cash';
UPDATE prize_pool SET win_probability_weight =  0.02 WHERE prize_code = 'CASH50K' AND prize_type = 'momo_cash';

-- Step 3: Rename all "MoMo Cash" prizes to plain "Cash"
-- The delivery mechanism (MoMo) is an implementation detail, not the prize name
UPDATE prize_pool SET name = '₦500 Cash'    WHERE prize_code = 'CASH500' AND prize_type = 'momo_cash';
UPDATE prize_pool SET name = '₦1,000 Cash'  WHERE prize_code = 'CASH1K'  AND prize_type = 'momo_cash';
UPDATE prize_pool SET name = '₦2,000 Cash'  WHERE prize_code = 'CASH2K'  AND prize_type = 'momo_cash';
UPDATE prize_pool SET name = '₦5,000 Cash'  WHERE prize_code = 'CASH5K'  AND prize_type = 'momo_cash';
UPDATE prize_pool SET name = '₦50,000 Cash' WHERE prize_code = 'CASH50K' AND prize_type = 'momo_cash';

-- Step 4: Add a CHECK constraint to prevent weights going below 0 or above 100
ALTER TABLE prize_pool
    ADD CONSTRAINT prize_pool_weight_range
    CHECK (win_probability_weight >= 0 AND win_probability_weight <= 100);
