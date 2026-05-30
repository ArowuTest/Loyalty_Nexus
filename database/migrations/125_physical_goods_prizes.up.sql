-- 125_physical_goods_prizes.up.sql
-- Adds support for physical/goods prize types.
-- These prizes are fulfilled manually by admin (shipping, logistics).
-- Users submit delivery details (name + address) when claiming.

-- Delivery details on spin_results for physical prizes
ALTER TABLE spin_results
  ADD COLUMN IF NOT EXISTS delivery_name    TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS delivery_address TEXT NOT NULL DEFAULT '';

-- new fulfillment status: admin has been notified, awaiting dispatch
-- The check constraint below mirrors the existing enum-style approach used elsewhere.
-- Note: PostgreSQL does not enforce TEXT check constraints on existing values when
-- added via ALTER TABLE; this is best-effort validation for new inserts.
ALTER TABLE spin_results
  DROP CONSTRAINT IF EXISTS spin_results_fulfillment_status_check;

ALTER TABLE spin_results
  ADD CONSTRAINT spin_results_fulfillment_status_check
    CHECK (fulfillment_status IN (
      'na',
      'pending',
      'pending_claim',
      'pending_momo_setup',
      'pending_delivery',
      'processing',
      'completed',
      'failed',
      'held'
    ));
