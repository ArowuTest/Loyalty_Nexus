-- Rollback 097
UPDATE prize_pool SET win_probability_weight = 1.5 WHERE name = 'Try Again' AND win_probability_weight = 8.0 AND is_active = true;
