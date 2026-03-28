-- ═══════════════════════════════════════════════════════════════════════════
-- Migration 059: Comprehensive Seed Data
-- ───────────────────────────────────────────────────────────────────────────
-- Fills every network_configs key that the application reads but was not
-- seeded in any prior migration. All INSERTs use ON CONFLICT DO NOTHING so
-- this migration is fully idempotent — re-running it is always safe.
--
-- Categories covered:
--   1. Core business rules (missing network_configs keys)
--   2. Streak & points expiry policies (ensure defaults exist)
--   3. Regional settings (all 37 Nigerian states + FCT)
--   4. Subscription plans (ensure all tiers present)
--   5. SMS templates (ensure all templates present)
--   6. Draw schedule (ensure at least one active draw exists)
--   7. Points expiry policy (ensure default policy exists)
-- ═══════════════════════════════════════════════════════════════════════════

BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- 1. NETWORK CONFIGS — missing keys identified by code audit
-- ─────────────────────────────────────────────────────────────────────────────

-- The network_configs table schema varies across migrations.
-- Use the most complete column set with safe defaults.

INSERT INTO network_configs (key, value, description, updated_at)
VALUES
    -- Prize pool total budget in kobo (₦500,000 = 50,000,000 kobo)
    ('prize_pool_kobo',           '50000000',
     'Total prize pool budget in kobo. Used for liability cap calculations.',
     NOW()),

    -- Daily prize liability cap (same as prize_pool_kobo alias used in some services)
    ('liability_cap_naira',       '500000',
     'Maximum daily prize liability in naira before the spin wheel is paused.',
     NOW()),

    -- Studio daily AI generation limit per user
    ('studio_daily_gen_limit',    '10',
     'Maximum number of AI generations a user can make per day across all studio tools.',
     NOW()),

    -- Winning bonus pulse points awarded on any win
    ('winning_bonus_pp',          '50',
     'Pulse points awarded to a user whenever they win any prize on the spin wheel.',
     NOW()),

    -- App base URL (used in wallet pass deep links and SMS)
    ('app_base_url',              'https://loyalty-nexus-api.onrender.com',
     'Public base URL of the API. Used in wallet pass back fields and SMS deep links.',
     NOW()),

    -- Streak window in hours (how long a streak stays alive without a recharge)
    ('streak_window_hours',       '24',
     'Hours within which a user must recharge to keep their streak alive. Default: 24.',
     NOW()),

    -- Points expiry in days
    ('points_expiry_days',        '365',
     'Number of days after which unused pulse points expire. Default: 365.',
     NOW()),

    -- Points expiry warning days before expiry
    ('points_expiry_warn_days',   '30',
     'Days before expiry to warn the user via push/SMS. Default: 30.',
     NOW()),

    -- Referral bonus for referrer
    ('referral_bonus_points',     '200',
     'Pulse points awarded to the referrer when a referred user makes their first recharge.',
     NOW()),

    -- Referral bonus for referee (the new user)
    ('referral_bonus_referee_pts','100',
     'Pulse points awarded to the new user (referee) on their first recharge via referral.',
     NOW()),

    -- First recharge bonus
    ('first_recharge_bonus_points','500',
     'Pulse points awarded on a user''s very first recharge. Default: 500.',
     NOW()),

    -- Fraud detection thresholds
    ('fraud_max_recharges_per_hour_per_user', '10',
     'Maximum recharges allowed per user per hour before fraud flag is raised.',
     NOW()),
    ('fraud_max_recharges_per_hour_per_ip',  '20',
     'Maximum recharges allowed per IP address per hour before fraud flag is raised.',
     NOW()),
    ('fraud_max_points_per_minute',          '1000',
     'Maximum pulse points that can be awarded to one user per minute.',
     NOW()),
    ('fraud_max_ai_gens_per_day',            '50',
     'Maximum AI generations per user per day before fraud flag is raised.',
     NOW()),

    -- Chat limits
    ('chat_memory_recent_messages',    '20',
     'Number of recent messages to include in chat context window.',
     NOW()),
    ('chat_memory_summaries_count',    '3',
     'Number of session summaries to include in chat context.',
     NOW()),
    ('chat_session_summary_messages',  '10',
     'Number of messages after which a session summary is generated.',
     NOW()),

    -- Storage backend
    ('storage_backend',           'local',
     'Storage backend for AI-generated assets. Values: local | s3 | gcs.',
     NOW()),
    ('local_storage_base_path',   '/tmp/nexus-assets',
     'Local filesystem path for storing AI-generated assets.',
     NOW()),
    ('local_storage_base_url',    'https://loyalty-nexus-api.onrender.com/assets',
     'Public URL prefix for locally stored assets.',
     NOW()),
    ('storage_cdn_base_url',      '',
     'CDN base URL for assets when using S3/GCS. Leave empty to use storage_backend URL.',
     NOW()),

    -- Ghost nudge SMS toggle
    ('ghost_nudge_sms_enabled',         'true',
     'Whether to send SMS nudges to inactive users. Default: true.',
     NOW()),
    ('ghost_nudge_wallet_push_enabled', 'true',
     'Whether to send wallet pass push updates as part of ghost nudge. Default: true.',
     NOW()),
    ('ghost_nudge_cooldown_hours',      '24',
     'Minimum hours between ghost nudge messages to the same user.',
     NOW()),
    ('ghost_nudge_batch_limit',         '500',
     'Maximum number of users to nudge per cron run.',
     NOW()),

    -- Regional Wars settings
    ('regional_wars_cycle_hours',       '168',
     'Duration of one Regional Wars cycle in hours. Default: 168 (7 days).',
     NOW()),
    ('wars_secondary_draw_default_prize_kobo', '500000',
     'Default prize for the Regional Wars secondary draw in kobo (₦5,000).',
     NOW()),
    ('wars_secondary_draw_default_winners',    '1',
     'Default number of winners in the Regional Wars secondary draw.',
     NOW()),

    -- Lifecycle worker toggles
    ('lifecycle_studio_stale_enabled',  'true',
     'Whether the studio stale-job cleanup worker is enabled.',
     NOW()),
    ('lifecycle_wars_resolve_enabled',  'true',
     'Whether the wars auto-resolve worker is enabled.',
     NOW()),

    -- Studio stale job settings
    ('studio_stale_job_timeout_minutes', '60',
     'Minutes after which a stuck studio job is marked as failed.',
     NOW()),
    ('studio_stale_job_batch_size',      '50',
     'Number of stale studio jobs to clean up per worker run.',
     NOW()),
    ('studio_stale_recovery_batch',      '20',
     'Number of stale jobs to attempt recovery on per worker run.',
     NOW()),
    ('studio_stale_job_timeout_secs',    '3600',
     'Seconds after which a stuck studio job is marked as failed (alias).',
     NOW()),

    -- MTN push pipeline
    ('mtn_push_hmac_secret',            '',
     'HMAC secret for validating MTN push webhook payloads. Set via admin panel.',
     NOW()),

    -- Draw entries per point
    ('draw_entries_per_point',          '1',
     'Number of draw entries awarded per pulse point earned.',
     NOW()),

    -- Spin limits
    ('spin_max_per_user_per_day',       '10',
     'Maximum number of spins a single user can take per day.',
     NOW()),
    ('spin_min_slots',                  '8',
     'Minimum number of visible slots on the spin wheel.',
     NOW()),
    ('spin_max_slots',                  '16',
     'Maximum number of visible slots on the spin wheel.',
     NOW()),

    -- Streak milestones (JSON array of day counts that trigger bonus rewards)
    ('streak_milestones_json',          '[7, 14, 30, 60, 90]',
     'Array of streak day milestones that trigger bonus pulse points.',
     NOW()),

    -- Streak freeze
    ('streak_freeze_days_per_month',    '2',
     'Number of days per month a user can freeze their streak without losing it.',
     NOW()),

    -- USSD max menu depth
    ('ussd_max_menu_depth',             '5',
     'Maximum number of nested menu levels in the USSD flow.',
     NOW()),

    -- Operation mode
    ('operation_mode',                  'independent',
     'Platform operation mode. Values: independent | telco_partner.',
     NOW()),

    -- AI chat toggle
    ('ai_chat_enabled',                 'true',
     'Whether the Nexus Chat AI assistant is enabled for users.',
     NOW()),

    -- Global points multiplier (1.0 = no multiplier)
    ('global_points_multiplier',        '1.0',
     'Global multiplier applied to all pulse point awards. 1.0 = no bonus.',
     NOW()),

    -- Pulse naira per point (how many naira = 1 pulse point)
    ('pulse_naira_per_point',           '10',
     'Naira value of 1 pulse point. Used for redemption rate display.',
     NOW()),

    -- Spin trigger and draw credit
    ('spin_trigger_naira',              '500',
     'Minimum recharge amount in naira to earn one spin credit.',
     NOW()),
    ('spin_draw_naira_per_credit',      '1000',
     'Naira recharged per draw entry credit earned.',
     NOW()),

    -- Min qualifying recharge
    ('min_qualifying_recharge_naira',   '100',
     'Minimum recharge in naira that qualifies for any loyalty reward.',
     NOW()),

    -- MTN push minimum
    ('mtn_push_min_amount_naira',       '100',
     'Minimum recharge amount in naira to qualify for MTN push notification.',
     NOW()),

    -- Streak expiry
    ('streak_expiry_hours',             '48',
     'Hours of inactivity after which a user streak is reset to zero.',
     NOW()),
    ('streak_expiry_warning_hours',     '24',
     'Hours before streak expiry to send a warning notification.',
     NOW()),
    ('streak_grace_days_per_month',     '1',
     'Grace days per month where a missed day does not break the streak.',
     NOW()),

    -- Asset retention
    ('asset_retention_days',            '7',
     'Days to retain AI-generated assets before automatic deletion.',
     NOW()),
    ('asset_expiry_warning_hours',      '24',
     'Hours before asset expiry to notify the user.',
     NOW()),

    -- Chat session settings
    ('chat_daily_message_limit',        '50',
     'Maximum chat messages a user can send per day.',
     NOW()),
    ('chat_gemini_daily_limit',         '100',
     'Maximum Gemini API calls per day across all users.',
     NOW()),
    ('chat_groq_daily_limit',           '200',
     'Maximum Groq API calls per day across all users.',
     NOW()),
    ('chat_session_timeout_minutes',    '30',
     'Minutes of inactivity before a chat session is closed.',
     NOW()),

    -- Daily prize liability cap
    ('daily_prize_liability_cap_naira', '500000',
     'Maximum total prize value in naira that can be awarded in a single day.',
     NOW()),

    -- Regional wars prize pool and winning bonus
    ('regional_wars_prize_pool_kobo',   '1000000',
     'Prize pool for Regional Wars in kobo (₦10,000).',
     NOW()),
    ('regional_wars_winning_bonus',     '500',
     'Pulse points bonus awarded to Regional Wars winners.',
     NOW())

ON CONFLICT (key) DO NOTHING;

-- ─────────────────────────────────────────────────────────────────────────────
-- 2. REGIONAL SETTINGS — all 36 Nigerian states + FCT
-- ─────────────────────────────────────────────────────────────────────────────

INSERT INTO regional_settings (region_code, region_name)
VALUES
    ('AB', 'Abia'),
    ('AD', 'Adamawa'),
    ('AK', 'Akwa Ibom'),
    ('AN', 'Anambra'),
    ('BA', 'Bauchi'),
    ('BY', 'Bayelsa'),
    ('BE', 'Benue'),
    ('BO', 'Borno'),
    ('CR', 'Cross River'),
    ('DE', 'Delta'),
    ('EB', 'Ebonyi'),
    ('ED', 'Edo'),
    ('EK', 'Ekiti'),
    ('EN', 'Enugu'),
    ('FC', 'FCT Abuja'),
    ('GO', 'Gombe'),
    ('IM', 'Imo'),
    ('JI', 'Jigawa'),
    ('KD', 'Kaduna'),
    ('KN', 'Kano'),
    ('KT', 'Katsina'),
    ('KE', 'Kebbi'),
    ('KO', 'Kogi'),
    ('KW', 'Kwara'),
    ('LA', 'Lagos'),
    ('NA', 'Nasarawa'),
    ('NI', 'Niger'),
    ('OG', 'Ogun'),
    ('ON', 'Ondo'),
    ('OS', 'Osun'),
    ('OY', 'Oyo'),
    ('PL', 'Plateau'),
    ('RI', 'Rivers'),
    ('SO', 'Sokoto'),
    ('TA', 'Taraba'),
    ('YO', 'Yobe'),
    ('ZA', 'Zamfara')
ON CONFLICT (region_code) DO NOTHING;

-- ─────────────────────────────────────────────────────────────────────────────
-- 3. SUBSCRIPTION PLANS — ensure all tiers are present
-- ─────────────────────────────────────────────────────────────────────────────

INSERT INTO subscription_plans (name, daily_cost_kobo, entries_per_day)
VALUES
    ('Basic',    5000,  1),
    ('Standard', 10000, 3),
    ('Premium',  20000, 7)
ON CONFLICT (name) DO NOTHING;

-- ─────────────────────────────────────────────────────────────────────────────
-- 4. SMS TEMPLATES — ensure all required templates exist
-- ─────────────────────────────────────────────────────────────────────────────

INSERT INTO sms_templates (key, template, description)
VALUES
    ('otp_login',
     'Your Loyalty Nexus verification code is {{code}}. Valid for 5 minutes. Do not share this code.',
     'OTP login delivery'),

    ('prize_airtime',
     'Congrats! You won {{amount}} airtime on Loyalty Nexus. It has been credited to {{phone}}. Keep recharging to win more!',
     'Airtime prize win notification'),

    ('prize_momo',
     'Congrats! You won ₦{{amount}} MoMo Cash on Loyalty Nexus. Confirm your MoMo number in the app to receive your prize.',
     'MoMo cash prize win notification'),

    ('prize_points',
     'You won {{points}} Pulse Points on Loyalty Nexus! Your new balance is {{balance}} points. Redeem in the app.',
     'Pulse points prize win notification'),

    ('streak_expiry',
     'Your Loyalty Nexus streak (Day {{streak}}) expires in {{hours}} hours! Recharge now to keep it alive and earn bonus spins.',
     'Streak expiry warning'),

    ('streak_milestone',
     'Amazing! You hit a {{days}}-day streak on Loyalty Nexus and earned {{bonus}} bonus Pulse Points! Keep it up!',
     'Streak milestone bonus notification'),

    ('asset_ready',
     'Your {{tool_name}} is ready on Nexus Studio! Open the app to download it before it expires in {{hours}} hours.',
     'AI generation ready notification'),

    ('ghost_nudge',
     'Hi! Your Loyalty Nexus streak is at risk. Recharge now to keep your Day {{streak}} streak alive and earn a spin!',
     'Ghost nudge for inactive users'),

    ('ghost_nudge_bronze',
     'Your Loyalty Nexus streak (Day {{streak}}) expires in {{hours}} hours. Recharge ₦{{min_amount}} now to keep it!',
     'Ghost nudge for Bronze tier users'),

    ('ghost_nudge_silver',
     'Silver member alert! Your Day {{streak}} streak expires in {{hours}} hours. Recharge to protect your Silver status!',
     'Ghost nudge for Silver tier users'),

    ('ghost_nudge_gold',
     'Gold member! Your Day {{streak}} streak expires in {{hours}} hours. Don''t lose your Gold rewards — recharge now!',
     'Ghost nudge for Gold tier users'),

    ('ghost_nudge_platinum',
     'Platinum VIP! Your Day {{streak}} streak expires in {{hours}} hours. Recharge to protect your exclusive Platinum benefits!',
     'Ghost nudge for Platinum tier users'),

    ('welcome',
     'Welcome to Loyalty Nexus! Recharge airtime or data to earn Pulse Points, spin the wheel, and win amazing prizes. Dial {{ussd_code}} to get started!',
     'Welcome message for new users'),

    ('referral_success',
     'Great news! Your friend {{name}} joined Loyalty Nexus using your referral. You''ve earned {{points}} bonus Pulse Points!',
     'Referral success notification for referrer'),

    ('points_expiry_warning',
     'Your {{points}} Pulse Points on Loyalty Nexus expire in {{days}} days. Redeem them in the app before they''re gone!',
     'Points expiry warning'),

    ('draw_winner',
     'Congratulations! You won ₦{{amount}} in the Loyalty Nexus {{draw_name}}! Check the app for details on how to claim your prize.',
     'Draw winner notification'),

    ('wars_winner',
     'Your state {{state}} won the Regional Wars this week! You earned {{points}} bonus Pulse Points. Keep recharging to defend your title!',
     'Regional Wars winner notification'),

    ('subscription_activated',
     'Your Loyalty Nexus {{plan}} subscription is now active. You''ll earn {{entries}} draw entries every day. Recharge to maximise your rewards!',
     'Subscription activation confirmation'),

    ('subscription_expiry',
     'Your Loyalty Nexus {{plan}} subscription expires tomorrow. Renew in the app to keep earning daily draw entries!',
     'Subscription expiry reminder')

ON CONFLICT (key) DO NOTHING;

-- ─────────────────────────────────────────────────────────────────────────────
-- 5. DRAWS — ensure at least one active monthly draw exists
-- ─────────────────────────────────────────────────────────────────────────────

INSERT INTO draws (id, name, status, winner_count, prize_type, prize_value_kobo)
VALUES
    (gen_random_uuid(), 'Monthly Grand Draw — April 2026', 'ACTIVE', 3, 'MOMO_CASH', 5000000),
    (gen_random_uuid(), 'Weekly Mega Spin Draw',           'ACTIVE', 1, 'MOMO_CASH', 1000000)
ON CONFLICT DO NOTHING;

-- ─────────────────────────────────────────────────────────────────────────────
-- 6. POINTS EXPIRY POLICIES — ensure default rolling policy exists
-- ─────────────────────────────────────────────────────────────────────────────

INSERT INTO points_expiry_policies (policy_type, expiry_days, warn_days_before, is_active)
VALUES
    ('rolling', 365, 30, true)
ON CONFLICT DO NOTHING;

-- ─────────────────────────────────────────────────────────────────────────────
-- 7. RECHARGE TIERS — ensure all tiers are present (supplement migration 040)
-- ─────────────────────────────────────────────────────────────────────────────

INSERT INTO recharge_tiers (tier_name, min_kobo, max_kobo, points_per_recharge, spin_credits, tier_label)
VALUES
    ('bronze',   10000,   49999,  10, 1, 'Bronze'),
    ('silver',   50000,   99999,  25, 2, 'Silver'),
    ('gold',    100000,  299999,  60, 3, 'Gold'),
    ('platinum',300000, 99999999,150, 5, 'Platinum')
ON CONFLICT (tier_name) DO NOTHING;

COMMIT;
