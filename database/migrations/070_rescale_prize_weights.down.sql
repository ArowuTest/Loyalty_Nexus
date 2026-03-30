-- Rollback migration 070: Restore prize weights to 10,000 integer scale
ALTER TABLE prize_pool DROP CONSTRAINT IF EXISTS prize_pool_weight_range;

ALTER TABLE prize_pool
    ALTER COLUMN win_probability_weight TYPE INTEGER
    USING ROUND(win_probability_weight * 100)::INTEGER;

-- Restore original seeded values
UPDATE prize_pool SET win_probability_weight = 4003 WHERE prize_code = 'NONE';
UPDATE prize_pool SET win_probability_weight = 2500 WHERE prize_code = 'PTS10';
UPDATE prize_pool SET win_probability_weight = 1500 WHERE prize_code = 'PTS25';
UPDATE prize_pool SET win_probability_weight =  800 WHERE prize_code = 'PTS50';
UPDATE prize_pool SET win_probability_weight =  500 WHERE prize_code = 'PTS100';
UPDATE prize_pool SET win_probability_weight =  300 WHERE prize_code = 'AIR50';
UPDATE prize_pool SET win_probability_weight =  150 WHERE prize_code = 'AIR100';
UPDATE prize_pool SET win_probability_weight =   75 WHERE prize_code = 'AIR200';
UPDATE prize_pool SET win_probability_weight =   50 WHERE prize_code = 'AIR500';
UPDATE prize_pool SET win_probability_weight =   25 WHERE prize_code = 'AIR1K';
UPDATE prize_pool SET win_probability_weight =   10 WHERE prize_code = 'AIR2K';
UPDATE prize_pool SET win_probability_weight =   30 WHERE prize_code = 'DATA500';
UPDATE prize_pool SET win_probability_weight =   20 WHERE prize_code = 'DATA1GB';
UPDATE prize_pool SET win_probability_weight =   10 WHERE prize_code = 'DATA2GB';
UPDATE prize_pool SET win_probability_weight =    8 WHERE prize_code = 'CASH500';
UPDATE prize_pool SET win_probability_weight =   10 WHERE prize_code = 'CASH1K';
UPDATE prize_pool SET win_probability_weight =    4 WHERE prize_code = 'CASH2K';
UPDATE prize_pool SET win_probability_weight =    3 WHERE prize_code = 'CASH5K';
UPDATE prize_pool SET win_probability_weight =    2 WHERE prize_code = 'CASH50K';

-- Restore original MoMo Cash prize names
UPDATE prize_pool SET name = '₦500 MoMo Cash'    WHERE prize_code = 'CASH500';
UPDATE prize_pool SET name = '₦1,000 MoMo Cash'  WHERE prize_code = 'CASH1K';
UPDATE prize_pool SET name = '₦2,000 MoMo Cash'  WHERE prize_code = 'CASH2K';
UPDATE prize_pool SET name = '₦5,000 MoMo Cash'  WHERE prize_code = 'CASH5K';
UPDATE prize_pool SET name = '₦50,000 MoMo Cash' WHERE prize_code = 'CASH50K';
