-- ============================================================
-- Migration 111: Best-in-class indexing for 100k+ concurrent users
-- 
-- Analysis covers every WHERE / ORDER BY / JOIN / GROUP BY across:
--   prize_repo, user_repo, transaction_repo, auth_repo, wars_repo,
--   studio_repo, chat_repo, passport_repo, spin_service, lifecycle_worker
-- 
-- Rules applied:
--   1. Every FK column that is queried gets its own index (FK alone ≠ index in PG)
--   2. Composite indexes follow column order: equality first, range/sort last
--   3. Partial indexes used wherever WHERE clause is selective and constant
--   4. CONCURRENTLY not used (migration runs inside transaction-safe context)
--   5. IF NOT EXISTS everywhere — fully idempotent
-- ============================================================

-- ─── USERS ────────────────────────────────────────────────────────────────────
-- phone_number IN (variants) — OTP login, recharge lookup (high-frequency)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_number
    ON users (phone_number);

-- state: wars leaderboard GROUP BY u.state, admin user lists
CREATE INDEX IF NOT EXISTS idx_users_state_active
    ON users (state, is_active)
    WHERE state IS NOT NULL AND state != '';

-- streak_expires_at: lifecycle worker ghost-nudge sweep
CREATE INDEX IF NOT EXISTS idx_users_streak_expires_active
    ON users (streak_expires_at)
    WHERE streak_expires_at IS NOT NULL AND is_active = TRUE;

-- points_expire_at: lifecycle worker expiry sweep
CREATE INDEX IF NOT EXISTS idx_users_points_expire
    ON users (points_expire_at)
    WHERE points_expire_at IS NOT NULL;

-- referred_by: referral chain lookups
CREATE INDEX IF NOT EXISTS idx_users_referred_by
    ON users (referred_by)
    WHERE referred_by IS NOT NULL;

-- last_recharge_at: ARPU / churn analysis queries
CREATE INDEX IF NOT EXISTS idx_users_last_recharge_at
    ON users (last_recharge_at DESC)
    WHERE last_recharge_at IS NOT NULL;

-- tier + is_active: admin user listing filter
CREATE INDEX IF NOT EXISTS idx_users_tier_active
    ON users (tier, is_active);

-- ─── WALLETS ──────────────────────────────────────────────────────────────────
-- user_id: GetWallet, LockWallet — already UNIQUE but ensure index exists
CREATE UNIQUE INDEX IF NOT EXISTS idx_wallets_user_id
    ON wallets (user_id);

-- ─── TRANSACTIONS ─────────────────────────────────────────────────────────────
-- user_id + created_at DESC: user transaction history pagination (very hot)
CREATE INDEX IF NOT EXISTS idx_transactions_user_created
    ON transactions (user_id, created_at DESC);

-- phone_number + type + created_at: GetDailyRechargeByPhone, leaderboard feed
CREATE INDEX IF NOT EXISTS idx_transactions_phone_type_created
    ON transactions (phone_number, type, created_at DESC);

-- type + created_at: wars leaderboard points aggregation, prize_award daily sum
CREATE INDEX IF NOT EXISTS idx_transactions_type_created
    ON transactions (type, created_at DESC);

-- reference: FindByReference (dedup / idempotency check on every recharge)
CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_reference_unique
    ON transactions (reference)
    WHERE reference IS NOT NULL AND reference != '';

-- type + reference: recharge type dedup
CREATE INDEX IF NOT EXISTS idx_transactions_type_reference
    ON transactions (type, reference);

-- points_delta + type: wars leaderboard SUM(points_delta) WHERE type = 'points_award'
CREATE INDEX IF NOT EXISTS idx_transactions_points_delta_type
    ON transactions (type, points_delta)
    WHERE type = 'points_award';

-- ─── SPIN_RESULTS ─────────────────────────────────────────────────────────────
-- user_id + created_at DESC: ListUserWins — every spin page load
CREATE INDEX IF NOT EXISTS idx_spin_results_user_created
    ON spin_results (user_id, created_at DESC);

-- fulfillment_status: ListPendingFulfillments + ListFailedFulfillments (retry worker)
CREATE INDEX IF NOT EXISTS idx_spin_results_fulfillment_status
    ON spin_results (fulfillment_status, created_at ASC)
    WHERE fulfillment_status IN ('pending', 'processing', 'failed');

-- retry_count: retry worker skip condition (retry_count < 3)
CREATE INDEX IF NOT EXISTS idx_spin_results_failed_retryable
    ON spin_results (fulfillment_status, retry_count)
    WHERE fulfillment_status = 'failed' AND retry_count < 3;

-- claim_status: ListAdminClaims filter, admin pending claims
CREATE INDEX IF NOT EXISTS idx_spin_results_claim_status_created
    ON spin_results (claim_status, created_at DESC);

-- expires_at: claim expiry sweep
CREATE INDEX IF NOT EXISTS idx_spin_results_expires_at
    ON spin_results (expires_at)
    WHERE expires_at IS NOT NULL;

-- prize_type + user_id: CountUserSpinsToday scoped per user
CREATE INDEX IF NOT EXISTS idx_spin_results_prize_type_user
    ON spin_results (user_id, prize_type, created_at DESC)
    WHERE prize_type != 'try_again';

-- ─── AUTH_OTPS ────────────────────────────────────────────────────────────────
-- phone_number + status: FindActiveOTP (every login — ultra-hot)
CREATE INDEX IF NOT EXISTS idx_auth_otps_phone_status
    ON auth_otps (phone_number, status)
    WHERE status = 'active';

-- created_at: OTP cleanup sweep (lifecycle worker)
CREATE INDEX IF NOT EXISTS idx_auth_otps_created_cleanup
    ON auth_otps (created_at)
    WHERE status != 'verified';

-- ─── PRIZE_POOL ───────────────────────────────────────────────────────────────
-- is_active + sort_order: ListActivePrizesSorted — every spin + wheel config load
CREATE INDEX IF NOT EXISTS idx_prize_pool_active_sort
    ON prize_pool (is_active, sort_order)
    WHERE is_active = TRUE;

-- is_active + base_value: ListActivePrizesMaxValue (forceLowValue path)
CREATE INDEX IF NOT EXISTS idx_prize_pool_active_value
    ON prize_pool (is_active, base_value)
    WHERE is_active = TRUE;

-- ─── AI_GENERATIONS ───────────────────────────────────────────────────────────
-- user_id + status + created_at: gallery load, history, admin view
CREATE INDEX IF NOT EXISTS idx_ai_generations_user_status_created
    ON ai_generations (user_id, status, created_at DESC);

-- tool_slug + created_at: studio tool stats aggregation
CREATE INDEX IF NOT EXISTS idx_ai_generations_tool_slug_created
    ON ai_generations (tool_slug, created_at DESC)
    WHERE tool_slug IS NOT NULL AND tool_slug != '';

-- status + expires_at: stale generation cleanup (lifecycle worker)
CREATE INDEX IF NOT EXISTS idx_ai_generations_status_expires
    ON ai_generations (status, expires_at)
    WHERE status IN ('pending', 'processing') AND expires_at IS NOT NULL;

-- disputed_at: admin dispute list filter
CREATE INDEX IF NOT EXISTS idx_ai_generations_disputed
    ON ai_generations (disputed_at DESC)
    WHERE disputed_at IS NOT NULL;

-- ─── CHAT_SESSIONS ────────────────────────────────────────────────────────────
-- user_id + status: GetActiveSession, ListStaleSessions lifecycle worker
CREATE INDEX IF NOT EXISTS idx_chat_sessions_user_status
    ON chat_sessions (user_id, status, last_activity_at DESC);

-- status + last_activity_at: lifecycle worker stale session sweep
CREATE INDEX IF NOT EXISTS idx_chat_sessions_status_activity
    ON chat_sessions (status, last_activity_at)
    WHERE status = 'active';

-- ─── CHAT_MESSAGES ────────────────────────────────────────────────────────────
-- session_id + created_at: LoadHistory — every chat page load
CREATE INDEX IF NOT EXISTS idx_chat_messages_session_created
    ON chat_messages (session_id, created_at ASC);

-- user_id + created_at: retention cleanup by user
CREATE INDEX IF NOT EXISTS idx_chat_messages_user_created
    ON chat_messages (user_id, created_at DESC);

-- ─── NOTIFICATIONS ────────────────────────────────────────────────────────────
-- user_id + created_at: ListNotifications pagination
CREATE INDEX IF NOT EXISTS idx_notifications_user_created
    ON notifications (user_id, created_at DESC);

-- user_id + is_read: unread badge count (very frequent)
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread
    ON notifications (user_id, is_read)
    WHERE is_read = FALSE;

-- ─── PUSH_TOKENS ──────────────────────────────────────────────────────────────
-- user_id + is_active: FCM push — GetTokenForUser
CREATE INDEX IF NOT EXISTS idx_push_tokens_user_active
    ON push_tokens (user_id, is_active)
    WHERE is_active = TRUE;

-- ─── DRAW_ENTRIES ─────────────────────────────────────────────────────────────
-- draw_id + user_id: duplicate entry check + winner selection
CREATE INDEX IF NOT EXISTS idx_draw_entries_draw_user
    ON draw_entries (draw_id, user_id);

-- user_id + created_at: GetMyWins, leaderboard
CREATE INDEX IF NOT EXISTS idx_draw_entries_user_created
    ON draw_entries (user_id, created_at DESC);

-- ─── DRAW_WINNERS ─────────────────────────────────────────────────────────────
-- draw_id: GetWinners — always filtered by draw
CREATE INDEX IF NOT EXISTS idx_draw_winners_draw_user
    ON draw_winners (draw_id, user_id);

-- ─── DRAWS ────────────────────────────────────────────────────────────────────
-- status + draw_time: upcoming draws fetch, scheduler worker
CREATE INDEX IF NOT EXISTS idx_draws_status_draw_time
    ON draws (status, draw_time)
    WHERE status IN ('UPCOMING', 'ACTIVE');

-- next_draw_at: scheduled draw worker trigger
CREATE INDEX IF NOT EXISTS idx_draws_next_draw_at
    ON draws (next_draw_at)
    WHERE next_draw_at IS NOT NULL AND status = 'UPCOMING';

-- ─── REGIONAL_WARS ────────────────────────────────────────────────────────────
-- status: GetCurrentWar, admin listing
CREATE INDEX IF NOT EXISTS idx_regional_wars_status_period
    ON regional_wars (status, period);

-- ─── REGIONAL_WAR_WINNERS ─────────────────────────────────────────────────────
-- war_id + state: GetWinnersByWarID
CREATE INDEX IF NOT EXISTS idx_war_winners_war_state
    ON regional_war_winners (war_id, state);

-- ─── USER_BADGES ──────────────────────────────────────────────────────────────
-- user_id + badge_key: HasBadge check, GetUserBadges
CREATE INDEX IF NOT EXISTS idx_user_badges_user_key
    ON user_badges (user_id, badge_key);

-- ─── PASSPORT_EVENTS ──────────────────────────────────────────────────────────
-- user_id + event_type + created_at: passport timeline
CREATE INDEX IF NOT EXISTS idx_passport_events_user_type_created
    ON passport_events (user_id, event_type, created_at DESC);

-- ─── USSD_SESSIONS ────────────────────────────────────────────────────────────
-- phone_number + session_id (compound): GetBySessionID + GetByPhone both served
CREATE INDEX IF NOT EXISTS idx_ussd_sessions_phone_session
    ON ussd_sessions (phone_number, session_id);

-- ─── FRAUD_EVENTS ─────────────────────────────────────────────────────────────
-- user_id + resolved: GetFraudByUser, ListUnresolved
CREATE INDEX IF NOT EXISTS idx_fraud_events_user_resolved
    ON fraud_events (user_id, resolved);

-- resolved + created_at: admin fraud listing
CREATE INDEX IF NOT EXISTS idx_fraud_events_resolved_created
    ON fraud_events (resolved, created_at DESC)
    WHERE resolved = FALSE;

-- ─── MTN_PUSH_EVENTS ──────────────────────────────────────────────────────────
-- msisdn + status: dedup check on every MTN push
CREATE INDEX IF NOT EXISTS idx_mtn_push_events_msisdn_status
    ON mtn_push_events (msisdn, status);

-- ─── LEDGER_ENTRIES ───────────────────────────────────────────────────────────
-- user_id + created_at: wallet ledger history
CREATE INDEX IF NOT EXISTS idx_ledger_entries_user_created
    ON ledger_entries (user_id, created_at DESC);

-- ─── SUBSCRIPTION_EVENTS ──────────────────────────────────────────────────────
-- user_id + created_at: GetSubscriptionHistory
CREATE INDEX IF NOT EXISTS idx_subscription_events_user_created
    ON subscription_events (user_id, created_at DESC);

-- ─── ADMIN_USERS ──────────────────────────────────────────────────────────────
-- email (login): already has index but ensure uniqueness enforced
CREATE UNIQUE INDEX IF NOT EXISTS idx_admin_users_email_unique
    ON admin_users (email)
    WHERE email IS NOT NULL;

-- ─── PRIZE_FULFILLMENT_LOGS ───────────────────────────────────────────────────
-- spin_result_id + created_at: retry audit trail
CREATE INDEX IF NOT EXISTS idx_pfl_spin_result_created
    ON prize_fulfillment_logs (spin_result_id, created_at DESC);

-- ─── WAR_SECONDARY_DRAWS ──────────────────────────────────────────────────────
-- war_id + state: GetSecondaryDraws
CREATE INDEX IF NOT EXISTS idx_war_sec_draws_war_state
    ON war_secondary_draws (war_id, state);

-- ─── AI_PROVIDER_CONFIGS ──────────────────────────────────────────────────────
-- category + is_active + priority: provider routing (on every AI call)
CREATE INDEX IF NOT EXISTS idx_ai_provider_cat_active_prio
    ON ai_provider_configs (category, priority)
    WHERE is_active = TRUE;

-- ─── PULSE_POINT_AWARDS ───────────────────────────────────────────────────────
-- user_id + awarded_at: GetBonusPulseAwards pagination
CREATE INDEX IF NOT EXISTS idx_ppa_user_awarded
    ON pulse_point_awards (user_id, awarded_at DESC);

-- ─── DRAW_SCHEDULES ───────────────────────────────────────────────────────────
-- is_active + draw_day_of_week: scheduler worker tick
CREATE INDEX IF NOT EXISTS idx_draw_schedules_active_day
    ON draw_schedules (is_active, draw_day_of_week)
    WHERE is_active = TRUE;
