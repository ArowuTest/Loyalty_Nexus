-- 125_physical_goods_prizes.down.sql
ALTER TABLE spin_results DROP COLUMN IF EXISTS delivery_name;
ALTER TABLE spin_results DROP COLUMN IF EXISTS delivery_address;

ALTER TABLE spin_results DROP CONSTRAINT IF EXISTS spin_results_fulfillment_status_check;
