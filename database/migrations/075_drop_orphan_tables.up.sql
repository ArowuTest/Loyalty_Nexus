-- Drop tables that are no longer referenced anywhere in the Go codebase
-- These were identified as truly orphaned during the schema consolidation analysis.

DROP TABLE IF EXISTS chat_session_summaries CASCADE;
DROP TABLE IF EXISTS fulfilment_webhooks CASCADE;
DROP TABLE IF EXISTS ledger_entries CASCADE;
DROP TABLE IF EXISTS multiplier_audit_logs CASCADE;
DROP TABLE IF EXISTS points_expiry_policies CASCADE;
DROP TABLE IF EXISTS prize_claims CASCADE;
DROP TABLE IF EXISTS prize_fulfillment_logs CASCADE;
DROP TABLE IF EXISTS program_bonuses CASCADE;
DROP TABLE IF EXISTS program_configs CASCADE;
DROP TABLE IF EXISTS qr_scan_log CASCADE;
DROP TABLE IF EXISTS region_tournaments CASCADE;
DROP TABLE IF EXISTS regional_wars_cycles CASCADE;
DROP TABLE IF EXISTS regional_wars_snapshots CASCADE;
DROP TABLE IF EXISTS sms_templates CASCADE;
DROP TABLE IF EXISTS studio_config CASCADE;
DROP TABLE IF EXISTS subscription_events CASCADE;
DROP TABLE IF EXISTS subscription_plans CASCADE;
DROP TABLE IF EXISTS user_subscriptions CASCADE;
DROP TABLE IF EXISTS wallet_passes CASCADE;
