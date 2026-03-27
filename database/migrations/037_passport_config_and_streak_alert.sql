-- Migration 037: Passport config keys + streak_expiry_alert column
-- Satisfies REQ-4.4 zero-hardcoding requirement:
--   All ghost nudge timing, window, and threshold values live in network_configs,
--   not in application code. Admin can change them without a code deploy.

-- ─── 1. Add streak_expiry_alert to google_wallet_objects ─────────────────────
-- This flag is set by GhostNudgeWorker when a streak expiry nudge is sent.
-- It is cleared automatically when the user recharges (streak resets).
-- BuildApplePKPassBytes and BuildSaveURL read this flag to render the
-- "⚠️ STREAK EXPIRING SOON!" visual alert on the wallet pass (REQ-4.4).
ALTER TABLE google_wallet_objects
    ADD COLUMN IF NOT EXISTS streak_expiry_alert BOOLEAN NOT NULL DEFAULT FALSE;

-- ─── 2. Seed ghost nudge configuration keys into network_configs ─────────────
-- These keys are read by GhostNudgeWorker via ConfigManager.GetInt().
-- Admins edit them via the Admin Cockpit → Passport & USSD → Configuration tab.

INSERT INTO network_configs (key, value, description, updated_at)
VALUES
    -- How often the ghost nudge cron runs (REQ-4.4 mandates 60 min default).
    ('ghost_nudge_interval_minutes', '60',
     'How often the Ghost Nudge cron job runs, in minutes. REQ-4.4 default: 60.',
     NOW()),

    -- How many hours before streak expiry to trigger the nudge.
    -- REQ-4.4 mandates: "streak will expire within the next 4 hours".
    ('ghost_nudge_warning_hours', '4',
     'Hours before streak expiry to trigger a Ghost Nudge. REQ-4.4 default: 4.',
     NOW()),

    -- Minimum streak length to qualify for a nudge.
    -- REQ-4.4 mandates: "streak of 3 or more days".
    ('ghost_nudge_min_streak', '3',
     'Minimum streak count (days) for a user to qualify for a Ghost Nudge. REQ-4.4 default: 3.',
     NOW()),

    -- SMS message template for ghost nudge (supports {streak} and {hours} placeholders).
    -- The Go service uses buildNudgeMessage() which applies tier-aware messaging,
    -- but this key allows overriding the base template from the admin panel.
    ('ghost_nudge_sms_enabled', 'true',
     'Whether to send SMS nudges during Ghost Nudge runs. Set to false to disable SMS without stopping wallet pushes.',
     NOW()),

    -- Whether to push wallet pass updates during ghost nudge runs.
    ('ghost_nudge_wallet_push_enabled', 'true',
     'Whether to push wallet pass updates (Apple APNs + Google Wallet) during Ghost Nudge runs.',
     NOW()),

    -- Maximum users to nudge per cron run (prevents runaway costs).
    ('ghost_nudge_batch_limit', '500',
     'Maximum number of users to nudge in a single Ghost Nudge cron run.',
     NOW()),

    -- Cooldown period: how many hours before the same user can be nudged again.
    -- Prevents spam. Default 24h matches the ghost_nudge_log cooldown query.
    ('ghost_nudge_cooldown_hours', '24',
     'Minimum hours between Ghost Nudge messages to the same user.',
     NOW())

ON CONFLICT (key) DO NOTHING;

-- ─── 3. Seed USSD configuration keys ─────────────────────────────────────────
INSERT INTO network_configs (key, value, description, updated_at)
VALUES
    -- The USSD short code (displayed in SMS nudges and on the wallet pass).
    ('ussd_short_code', '*384#',
     'The USSD short code for Loyalty Nexus. Displayed in SMS nudges and wallet pass back fields.',
     NOW()),

    -- Session timeout in seconds (Africa''s Talking drops sessions after 20s of inactivity).
    ('ussd_session_timeout_seconds', '20',
     'USSD session inactivity timeout in seconds. Africa''s Talking default is 20s.',
     NOW()),

    -- Maximum menu depth (prevents infinite loops in state machine).
    ('ussd_max_menu_depth', '5',
     'Maximum number of USSD menu levels a user can navigate in a single session.',
     NOW())

ON CONFLICT (key) DO NOTHING;
