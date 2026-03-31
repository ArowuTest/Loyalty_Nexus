-- ============================================================================
-- MIGRATION 073: CONSOLIDATED SCHEMA PATCH (DOWN)
-- ============================================================================

ALTER TABLE ai_generations
    DROP COLUMN IF EXISTS output_text,
    DROP COLUMN IF EXISTS provider,
    DROP COLUMN IF EXISTS cost_micros,
    DROP COLUMN IF EXISTS duration_ms,
    DROP COLUMN IF EXISTS refund_granted,
    DROP COLUMN IF EXISTS refund_pts,
    DROP COLUMN IF EXISTS updated_at;

ALTER TABLE studio_tools
    DROP COLUMN IF EXISTS icon,
    DROP COLUMN IF EXISTS sort_order,
    DROP COLUMN IF EXISTS is_free,
    DROP COLUMN IF EXISTS provider_tool,
    DROP COLUMN IF EXISTS refund_pct,
    DROP COLUMN IF EXISTS refund_window_mins,
    DROP COLUMN IF EXISTS ui_config;

ALTER TABLE ghost_nudge_log
    DROP COLUMN IF EXISTS message,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS created_at;

ALTER TABLE google_wallet_objects
    DROP COLUMN IF EXISTS last_sync_status;

ALTER TABLE spin_results
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS admin_notes,
    DROP COLUMN IF EXISTS bank_account_name,
    DROP COLUMN IF EXISTS bank_account_number,
    DROP COLUMN IF EXISTS bank_name,
    DROP COLUMN IF EXISTS payment_reference,
    DROP COLUMN IF EXISTS rejection_reason,
    DROP COLUMN IF EXISTS reviewed_at;

ALTER TABLE transactions
    DROP COLUMN IF EXISTS reference;

ALTER TABLE wallet_registrations
    DROP COLUMN IF EXISTS is_active;

ALTER TABLE prize_pool
    DROP COLUMN IF EXISTS prize_code,
    DROP COLUMN IF EXISTS variation_code,
    DROP COLUMN IF EXISTS icon_name,
    DROP COLUMN IF EXISTS terms_and_conditions;

ALTER TABLE wallets
    DROP COLUMN IF EXISTS pulse_counter,
    DROP COLUMN IF EXISTS draw_counter,
    DROP COLUMN IF EXISTS daily_recharge_kobo,
    DROP COLUMN IF EXISTS daily_recharge_date,
    DROP COLUMN IF EXISTS daily_spins_awarded;
