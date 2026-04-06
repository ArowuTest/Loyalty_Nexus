-- Migration 100: Seed fraud detection threshold config keys into network_configs.
-- These replace the previously hardcoded Go constants in fraud_service.go,
-- making all fraud thresholds admin-configurable via the Config panel.

INSERT INTO network_configs (key, value, description) VALUES
  ('fraud_max_recharge_per_24h',        '20',   'Maximum number of recharge events allowed per user per 24 hours before flagging for velocity fraud'),
  ('fraud_max_spin_per_24h',            '10',   'Maximum number of spin attempts allowed per user per 24 hours before flagging for spin abuse'),
  ('fraud_min_recharge_kobo',           '10000','Minimum legitimate recharge amount in kobo (₦100 = 10000 kobo); recharges below this are logged as micro-farming'),
  ('fraud_duplicate_tx_window_seconds', '300',  'Time window in seconds within which a duplicate transaction reference is considered fraudulent (default: 300 = 5 minutes)')
ON CONFLICT (key) DO NOTHING;
