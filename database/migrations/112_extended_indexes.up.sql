-- ============================================================
-- Migration 112: Extended best-in-class indexes — remaining tables
--
-- Migration 111 covered 29 core tables.
-- This migration covers all remaining tables not yet indexed:
--
--   AI Studio:      studio_tools, studio_sessions, studio_usage_metrics, studio_config
--   Recharges:      recharges, network_operator_configs, webhook_events
--   MTN Push:       mtn_push_csv_uploads, mtn_push_csv_rows
--   Wars/Draws:     war_secondary_draw_winners, regional_stats, regional_settings
--   Admin:          admin_refresh_tokens (supplement), prize_claims
--   Wallet/Sub:     user_subscriptions, subscription_plans, wallet_passes
--   Platform:       platform_settings, asset_expiry_notifications, program_configs
--   HLR/Cache:      hlr_cache, network_cache, network_configs, msisdn_blacklist
--   Misc:           ghost_nudge_log, google_wallet_objects, session_summaries
--                   notification_broadcasts, arpu_uplift_tracking
--
-- All indexes are IF NOT EXISTS — fully idempotent.
-- Indexes that already exist in original migrations are not duplicated.
-- ============================================================

-- ─── AI STUDIO — studio_tools ─────────────────────────────────────────────────
-- category + is_active + sort_order: GetToolsByCategory (every Studio page load)
CREATE INDEX IF NOT EXISTS idx_studio_tools_category_active_sort
    ON studio_tools (category, sort_order)
    WHERE is_active = TRUE;

-- provider: route by provider (Pollinations, Gemini etc.)
CREATE INDEX IF NOT EXISTS idx_studio_tools_provider_active
    ON studio_tools (provider, is_active)
    WHERE is_active = TRUE;

-- slug: GetToolBySlug — resolving every Studio tool invocation
CREATE UNIQUE INDEX IF NOT EXISTS idx_studio_tools_slug_unique
    ON studio_tools (slug)
    WHERE slug IS NOT NULL;

-- ─── AI STUDIO — studio_sessions ──────────────────────────────────────────────
-- NOTE: idx_studio_sessions_user_id already exists in migration 031.
-- Add: last_active_at for stale session sweep (different from existing idx_studio_sessions_active)
CREATE INDEX IF NOT EXISTS idx_studio_sessions_stale_sweep
    ON studio_sessions (last_active_at)
    WHERE ended_at IS NULL;

-- ─── AI STUDIO — studio_usage_metrics ─────────────────────────────────────────
-- user_id + created_at: per-user usage stats, quota checks
CREATE INDEX IF NOT EXISTS idx_studio_usage_user_created
    ON studio_usage_metrics (user_id, created_at DESC);

-- tool_slug + created_at: aggregate tool popularity, admin dashboard
CREATE INDEX IF NOT EXISTS idx_studio_usage_tool_created
    ON studio_usage_metrics (tool_slug, created_at DESC)
    WHERE tool_slug IS NOT NULL;

-- status + created_at: billing sweep for failed/pending usage records
CREATE INDEX IF NOT EXISTS idx_studio_usage_status_created
    ON studio_usage_metrics (status, created_at)
    WHERE status IN ('pending', 'failed');

-- ─── AI STUDIO — studio_config ────────────────────────────────────────────────
-- key: GetConfig (every configuration read)
CREATE UNIQUE INDEX IF NOT EXISTS idx_studio_config_key_unique
    ON studio_config (key)
    WHERE key IS NOT NULL;

-- ─── RECHARGES ────────────────────────────────────────────────────────────────
-- NOTE: idx_recharges_msisdn, idx_recharges_payref, idx_recharges_status
--       already exist from migration 108. Adding compound indexes for hot paths:

-- user_id + status + created_at: user recharge history with status filter
CREATE INDEX IF NOT EXISTS idx_recharges_user_status_created
    ON recharges (user_id, status, created_at DESC);

-- network + status: network-specific success rate analysis, admin
CREATE INDEX IF NOT EXISTS idx_recharges_network_status
    ON recharges (network, status);

-- vtpass_request_id: fulfillment lookup by VTPass reference (idempotency)
CREATE UNIQUE INDEX IF NOT EXISTS idx_recharges_vtpass_request_id
    ON recharges (vtpass_request_id)
    WHERE vtpass_request_id IS NOT NULL AND vtpass_request_id != '';

-- completed_at: revenue reporting by completion time
CREATE INDEX IF NOT EXISTS idx_recharges_completed_at
    ON recharges (completed_at DESC)
    WHERE completed_at IS NOT NULL AND status = 'success';

-- ─── MTN PUSH — mtn_push_csv_uploads ──────────────────────────────────────────
-- created_at + status: admin upload listing, cleanup
CREATE INDEX IF NOT EXISTS idx_mtn_csv_uploads_status_created
    ON mtn_push_csv_uploads (status, created_at DESC);

-- uploaded_by: admin who uploaded (admin log / audit)
CREATE INDEX IF NOT EXISTS idx_mtn_csv_uploads_uploaded_by
    ON mtn_push_csv_uploads (uploaded_by)
    WHERE uploaded_by IS NOT NULL;

-- ─── MTN PUSH — mtn_push_csv_rows ─────────────────────────────────────────────
-- upload_id + status: process pending rows by batch (processor worker)
CREATE INDEX IF NOT EXISTS idx_mtn_csv_rows_upload_status
    ON mtn_push_csv_rows (upload_id, status);

-- msisdn + status: dedup check per phone number
CREATE INDEX IF NOT EXISTS idx_mtn_csv_rows_msisdn_status
    ON mtn_push_csv_rows (msisdn, status)
    WHERE status = 'pending';

-- ─── WARS — war_secondary_draw_winners ────────────────────────────────────────
-- NOTE: idx_war_sec_winners_draw_id, idx_war_sec_winners_user_id exist already.
-- Add: payment_status filter — admin payout queue
CREATE INDEX IF NOT EXISTS idx_war_sec_winners_payment_status
    ON war_secondary_draw_winners (payment_status, created_at DESC)
    WHERE payment_status IN ('pending', 'processing');

-- war_id + state: GetWinnersByWarAndState
CREATE INDEX IF NOT EXISTS idx_war_sec_winners_war_state
    ON war_secondary_draw_winners (war_id, state);

-- ─── WARS — regional_stats ────────────────────────────────────────────────────
-- war_id + state: GetStatsByWar (leaderboard computation)
CREATE INDEX IF NOT EXISTS idx_regional_stats_war_state
    ON regional_stats (war_id, state);

-- war_id + total_points DESC: sorted leaderboard
CREATE INDEX IF NOT EXISTS idx_regional_stats_war_points
    ON regional_stats (war_id, total_points DESC);

-- ─── WARS — regional_settings ─────────────────────────────────────────────────
-- war_id: GetSettingsByWar (prize config)
CREATE INDEX IF NOT EXISTS idx_regional_settings_war_id
    ON regional_settings (war_id);

-- ─── PRIZE_CLAIMS ─────────────────────────────────────────────────────────────
-- spin_result_id: GetClaimByResult (dedup / status check)
CREATE INDEX IF NOT EXISTS idx_prize_claims_spin_result_id
    ON prize_claims (spin_result_id);

-- user_id + status + created_at: GetUserClaims pagination
CREATE INDEX IF NOT EXISTS idx_prize_claims_user_status_created
    ON prize_claims (user_id, status, created_at DESC);

-- status + created_at: admin pending claims queue
CREATE INDEX IF NOT EXISTS idx_prize_claims_status_created
    ON prize_claims (status, created_at DESC)
    WHERE status IN ('pending', 'processing');

-- ─── USER_SUBSCRIPTIONS ───────────────────────────────────────────────────────
-- user_id + status: GetActiveSubscription (every premium feature gate check)
CREATE INDEX IF NOT EXISTS idx_user_subs_user_status
    ON user_subscriptions (user_id, status)
    WHERE status = 'active';

-- plan_id + status: plan usage stats
CREATE INDEX IF NOT EXISTS idx_user_subs_plan_status
    ON user_subscriptions (plan_id, status);

-- expires_at: subscription expiry sweep (lifecycle worker)
CREATE INDEX IF NOT EXISTS idx_user_subs_expires_at
    ON user_subscriptions (expires_at)
    WHERE status = 'active' AND expires_at IS NOT NULL;

-- ─── SUBSCRIPTION_PLANS ───────────────────────────────────────────────────────
-- is_active + sort_order: ListActivePlans (pricing page, subscription flow)
CREATE INDEX IF NOT EXISTS idx_sub_plans_active_sort
    ON subscription_plans (is_active, sort_order)
    WHERE is_active = TRUE;

-- ─── WALLET_PASSES ────────────────────────────────────────────────────────────
-- user_id: GetPassForUser (Apple/Google Wallet card lookup)
CREATE INDEX IF NOT EXISTS idx_wallet_passes_user_id
    ON wallet_passes (user_id);

-- pass_type_id + serial_number: wallet pass update callback (unique lookup)
CREATE UNIQUE INDEX IF NOT EXISTS idx_wallet_passes_type_serial
    ON wallet_passes (pass_type_id, serial_number)
    WHERE pass_type_id IS NOT NULL;

-- ─── PLATFORM_SETTINGS ────────────────────────────────────────────────────────
-- category: GetSettingsByCategory (admin config panel grouping)
CREATE INDEX IF NOT EXISTS idx_platform_settings_category
    ON platform_settings (category)
    WHERE category IS NOT NULL;

-- ─── ASSET_EXPIRY_NOTIFICATIONS ───────────────────────────────────────────────
-- generation_id + notif_window: HasNotificationBeenSent (deduplicate expiry nudges)
CREATE UNIQUE INDEX IF NOT EXISTS idx_asset_expiry_notif_gen_window
    ON asset_expiry_notifications (generation_id, notif_window);

-- ─── PROGRAM_CONFIGS ──────────────────────────────────────────────────────────
-- key: GetConfig (every program configuration read — very hot, KV lookup)
CREATE UNIQUE INDEX IF NOT EXISTS idx_program_configs_key_unique
    ON program_configs (key)
    WHERE key IS NOT NULL;

-- ─── HLR_CACHE ────────────────────────────────────────────────────────────────
-- msisdn: HLR lookup (every MTN push / recharge initiation)
CREATE UNIQUE INDEX IF NOT EXISTS idx_hlr_cache_msisdn_unique
    ON hlr_cache (msisdn);

-- expires_at: cache invalidation sweep
CREATE INDEX IF NOT EXISTS idx_hlr_cache_expires_at
    ON hlr_cache (expires_at)
    WHERE expires_at IS NOT NULL;

-- ─── NETWORK_CACHE ────────────────────────────────────────────────────────────
-- msisdn: GetCachedNetwork (every recharge before VTPass call)
CREATE UNIQUE INDEX IF NOT EXISTS idx_network_cache_msisdn_unique
    ON network_cache (msisdn);

-- ─── NETWORK_CONFIGS ──────────────────────────────────────────────────────────
-- network_code: GetNetworkConfig (every recharge routing decision)
CREATE UNIQUE INDEX IF NOT EXISTS idx_network_configs_code_unique
    ON network_configs (network_code)
    WHERE network_code IS NOT NULL;

-- ─── MSISDN_BLACKLIST ─────────────────────────────────────────────────────────
-- msisdn: IsBlacklisted (fraud check on every recharge / spin)
CREATE UNIQUE INDEX IF NOT EXISTS idx_msisdn_blacklist_unique
    ON msisdn_blacklist (msisdn);

-- ─── GHOST_NUDGE_LOG ──────────────────────────────────────────────────────────
-- user_id + sent_at: HasNudgeBeenSent (prevent duplicate nudges)
CREATE INDEX IF NOT EXISTS idx_ghost_nudge_user_sent
    ON ghost_nudge_log (user_id, sent_at DESC);

-- nudge_type + sent_at: batch nudge sweep (lifecycle worker)
CREATE INDEX IF NOT EXISTS idx_ghost_nudge_type_sent
    ON ghost_nudge_log (nudge_type, sent_at DESC);

-- ─── GOOGLE_WALLET_OBJECTS ────────────────────────────────────────────────────
-- user_id: GetGoogleWalletObject (wallet pass display)
CREATE INDEX IF NOT EXISTS idx_google_wallet_objects_user_id
    ON google_wallet_objects (user_id);

-- object_id: GetByObjectId (Google Wallet callback)
CREATE UNIQUE INDEX IF NOT EXISTS idx_google_wallet_objects_object_id
    ON google_wallet_objects (object_id)
    WHERE object_id IS NOT NULL;

-- ─── SESSION_SUMMARIES ────────────────────────────────────────────────────────
-- session_id: GetSummaryBySession (chat context recall)
CREATE UNIQUE INDEX IF NOT EXISTS idx_session_summaries_session_id
    ON session_summaries (session_id);

-- user_id + created_at: GetRecentSummaries (multi-session context)
CREATE INDEX IF NOT EXISTS idx_session_summaries_user_created
    ON session_summaries (user_id, created_at DESC);

-- ─── NOTIFICATION_BROADCASTS ──────────────────────────────────────────────────
-- status + scheduled_at: broadcast worker queue
CREATE INDEX IF NOT EXISTS idx_notif_broadcasts_status_scheduled
    ON notification_broadcasts (status, scheduled_at)
    WHERE status IN ('pending', 'processing');

-- created_by + created_at: admin broadcast history
CREATE INDEX IF NOT EXISTS idx_notif_broadcasts_created_by_at
    ON notification_broadcasts (created_by, created_at DESC)
    WHERE created_by IS NOT NULL;

-- ─── ARPU_UPLIFT_TRACKING ─────────────────────────────────────────────────────
-- user_id + period: GetArpuByUserPeriod (analytics)
CREATE INDEX IF NOT EXISTS idx_arpu_uplift_user_period
    ON arpu_uplift_tracking (user_id, period);

-- period + segment: aggregate ARPU by segment/period (BI reports)
CREATE INDEX IF NOT EXISTS idx_arpu_uplift_period_segment
    ON arpu_uplift_tracking (period, segment)
    WHERE segment IS NOT NULL;

-- ─── ADMIN_REFRESH_TOKENS (supplement) ────────────────────────────────────────
-- NOTE: idx_admin_refresh_tokens_admin_id, _token_hash, _expires_at exist already.
-- Add partial index for active (non-revoked) tokens only — the hot path
CREATE INDEX IF NOT EXISTS idx_admin_refresh_active
    ON admin_refresh_tokens (admin_id, expires_at)
    WHERE revoked_at IS NULL;
