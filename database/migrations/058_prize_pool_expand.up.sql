-- Migration 053: Expand prize pool — add ₦1k / ₦2k airtime and ₦2k MoMo cash slots
-- Also adds ₦50 and ₦100 MoMo cash for micro-win excitement.
-- All values in kobo (base_value). Weights assume existing 044 seed is present.

INSERT INTO prize_pool (id, name, prize_code, prize_type, base_value, win_probability_weight,
                        is_active, is_no_win, no_win_message, color_scheme, sort_order,
                        minimum_recharge, icon_name)
VALUES
  -- Additional airtime tiers
  (gen_random_uuid(), '₦1,000 Airtime',  'AIR1K',  'airtime',   100000, 25, TRUE, FALSE, '', '#2196F3', 10, 500000,  'phone'),
  (gen_random_uuid(), '₦2,000 Airtime',  'AIR2K',  'airtime',   200000, 10, TRUE, FALSE, '', '#1a78c2', 11, 1000000, 'phone'),

  -- Additional MoMo cash tiers (micro + mid)
  (gen_random_uuid(), '₦500 MoMo Cash',  'CASH500', 'momo_cash',  50000, 8, TRUE, FALSE, '', '#10b981', 16, 300000,  'money-bag'),
  (gen_random_uuid(), '₦2,000 MoMo Cash','CASH2K',  'momo_cash', 200000, 4, TRUE, FALSE, '', '#059669', 17, 500000,  'money-bag')

ON CONFLICT DO NOTHING;

-- Note: After adding these rows the total weight across all active prizes will exceed
-- the original 10,000. Admin MUST open Spin Config → reduce "Better Luck Next Time"
-- weight accordingly before weights can be saved again. Current addition: 25+10+8+4 = 47
-- Reduce 'try_again' from 4050 → 4003 to stay within budget.
UPDATE prize_pool
SET    win_probability_weight = 4003
WHERE  prize_code = 'NONE' AND prize_type = 'try_again';
