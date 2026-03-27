-- Migration 044: Seed default 15 wheel prizes (RechargeMax-aligned, LN prize types)
-- Probabilities are stored as integer weights summing to 10,000 (= 100.00%)
-- is_no_win = TRUE means no win record is created in spin_results for this slot

INSERT INTO prize_pool (id, name, prize_code, prize_type, base_value, win_probability_weight, is_active, is_no_win, no_win_message, color_scheme, sort_order, minimum_recharge, icon_name)
VALUES
    -- No-win slots (40.50% total = 4050/10000)
    (gen_random_uuid(), 'Better Luck Next Time', 'NONE',    'try_again',     0,      4050, TRUE, TRUE,  'Better Luck Next Time!', '#CCCCCC', 1,  0,       'sad-face'),

    -- Points prizes (53.00% total = 5300/10000)
    (gen_random_uuid(), '10 Pulse Points',       'PTS10',   'pulse_points',  10,     2500, TRUE, FALSE, '', '#4CAF50', 2,  0,       'star'),
    (gen_random_uuid(), '25 Pulse Points',       'PTS25',   'pulse_points',  25,     1500, TRUE, FALSE, '', '#4CAF50', 3,  0,       'star'),
    (gen_random_uuid(), '50 Pulse Points',       'PTS50',   'pulse_points',  50,      800, TRUE, FALSE, '', '#4CAF50', 4,  0,       'star'),
    (gen_random_uuid(), '100 Pulse Points',      'PTS100',  'pulse_points',  100,     500, TRUE, FALSE, '', '#4CAF50', 5,  0,       'star'),

    -- Airtime prizes (5.75% total = 575/10000)
    (gen_random_uuid(), '₦50 Airtime',           'AIR50',   'airtime',       5000,    300, TRUE, FALSE, '', '#2196F3', 6,  100000,  'phone'),
    (gen_random_uuid(), '₦100 Airtime',          'AIR100',  'airtime',       10000,   150, TRUE, FALSE, '', '#2196F3', 7,  100000,  'phone'),
    (gen_random_uuid(), '₦200 Airtime',          'AIR200',  'airtime',       20000,    75, TRUE, FALSE, '', '#2196F3', 8,  200000,  'phone'),
    (gen_random_uuid(), '₦500 Airtime',          'AIR500',  'airtime',       50000,    50, TRUE, FALSE, '', '#2196F3', 9,  500000,  'phone'),

    -- Data prizes (0.60% total = 60/10000)
    (gen_random_uuid(), '500MB Data',            'DATA500', 'data_bundle',   50000,    30, TRUE, FALSE, '', '#9C27B0', 10, 200000,  'wifi'),
    (gen_random_uuid(), '1GB Data',              'DATA1GB', 'data_bundle',   100000,   20, TRUE, FALSE, '', '#9C27B0', 11, 500000,  'wifi'),
    (gen_random_uuid(), '2GB Data',              'DATA2GB', 'data_bundle',   200000,   10, TRUE, FALSE, '', '#9C27B0', 12, 1000000, 'wifi'),

    -- MoMo cash prizes (0.15% total = 15/10000)
    (gen_random_uuid(), '₦1,000 MoMo Cash',      'CASH1K',  'momo_cash',     100000,   10, TRUE, FALSE, '', '#FF9800', 13, 500000,  'money-bag'),
    (gen_random_uuid(), '₦5,000 MoMo Cash',      'CASH5K',  'momo_cash',     500000,    3, TRUE, FALSE, '', '#FF9800', 14, 1000000, 'money-bag'),
    (gen_random_uuid(), '₦50,000 MoMo Cash',     'CASH50K', 'momo_cash',    5000000,    2, TRUE, FALSE, '', '#FF5722', 15, 3000000, 'trophy')
ON CONFLICT DO NOTHING;
