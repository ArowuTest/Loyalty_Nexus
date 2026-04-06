-- Rollback migration 100: Remove fraud threshold config keys
DELETE FROM network_configs WHERE key IN (
  'fraud_max_recharge_per_24h',
  'fraud_max_spin_per_24h',
  'fraud_min_recharge_kobo',
  'fraud_duplicate_tx_window_seconds'
);
