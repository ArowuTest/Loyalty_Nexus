-- 020_spec_alignment_and_complete_schema.sql
-- Purpose: Align the entire schema with the Loyalty Nexus Master Specification v2.1 + SRS.
-- This migration:
--   1. Renames msisdn -> phone_number on users (spec REQ-1.1)
--   2. Renames program_configs -> network_configs (spec section 5.1)
--   3. Adds all missing columns to users table
--   4. Adds wallet_pass_id to users
--   5. Completes wallets table (two-pool ledger)
--   6. Adds full regional_wars_snapshots table
--   7. Adds fraud_events table
--   8. Adds spin_results table (separate from prize_claims for spec compliance)
--   9. Adds points_expiry table
--  10. Adds scheduled_multipliers and segment_multipliers tables
--  11. Adds asset_retention admin config
--  12. Seeds complete network_configs with ALL spec-defined parameters

-- ============================================================
-- STEP 1: Rename msisdn -> phone_number on users
-- ============================================================
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns 
             WHERE table_name='users' AND column_name='msisdn') THEN
    ALTER TABLE users RENAME COLUMN msisdn TO phone_number;
  END IF;
END $$;

-- Update index
DROP INDEX IF EXISTS idx_users_msisdn;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone ON users(phone_number);

-- Update transactions column if needed
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns 
             WHERE table_name='transactions' AND column_name='msisdn') THEN
    ALTER TABLE transactions RENAME COLUMN msisdn TO phone_number;
  END IF;
END $$;

-- Update auth_otps
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns 
             WHERE table_name='auth_otps' AND column_name='msisdn') THEN
    ALTER TABLE auth_otps RENAME COLUMN msisdn TO phone_number;
  END IF;
END $$;

-- Update network_cache
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns 
             WHERE table_name='network_cache' AND column_name='msisdn') THEN
    ALTER TABLE network_cache RENAME COLUMN msisdn TO phone_number;
  END IF;
END $$;

-- Update msisdn_blacklist
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns 
             WHERE table_name='msisdn_blacklist' AND column_name='msisdn') THEN
    ALTER TABLE msisdn_blacklist RENAME COLUMN msisdn TO phone_number;
  END IF;
END $$;

-- ============================================================
-- STEP 2: Rename program_configs -> network_configs
-- ============================================================
DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables 
             WHERE table_name='program_configs') 
  AND NOT EXISTS (SELECT 1 FROM information_schema.tables 
                  WHERE table_name='network_configs') THEN
    ALTER TABLE program_configs RENAME TO network_configs;
    ALTER TABLE network_configs RENAME COLUMN config_key TO key;
    ALTER TABLE network_configs RENAME COLUMN config_value TO value;
  END IF;
END $$;

-- ============================================================
-- STEP 3: Complete users table with all spec-required columns
-- ============================================================
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS wallet_pass_id        VARCHAR(255),         -- Apple/Google Wallet pass serial number
  ADD COLUMN IF NOT EXISTS device_type           VARCHAR(10),          -- 'smartphone' | 'feature_phone'
  ADD COLUMN IF NOT EXISTS subscription_tier     VARCHAR(20) DEFAULT 'free',
  ADD COLUMN IF NOT EXISTS last_recharge_at      TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS streak_expires_at     TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS referral_code         VARCHAR(20),
  ADD COLUMN IF NOT EXISTS referred_by           UUID REFERENCES users(id),
  ADD COLUMN IF NOT EXISTS kyc_status            VARCHAR(20) DEFAULT 'unverified',
  ADD COLUMN IF NOT EXISTS streak_grace_used     INTEGER DEFAULT 0,    -- Days used this month (REQ-5.2.13)
  ADD COLUMN IF NOT EXISTS streak_grace_month    INTEGER,              -- Month (1-12) grace was last used
  ADD COLUMN IF NOT EXISTS points_expire_at      TIMESTAMPTZ,         -- Rolling/fixed expiry (REQ-5.2.14)
  ADD COLUMN IF NOT EXISTS created_at            TIMESTAMPTZ DEFAULT NOW(),
  ADD COLUMN IF NOT EXISTS updated_at            TIMESTAMPTZ DEFAULT NOW();

-- Generate referral codes for existing users
UPDATE users SET referral_code = UPPER(SUBSTR(MD5(id::text), 1, 8)) WHERE referral_code IS NULL;
ALTER TABLE users ALTER COLUMN referral_code SET DEFAULT UPPER(SUBSTR(MD5(gen_random_uuid()::text), 1, 8));
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_referral_code ON users(referral_code);

-- ============================================================
-- STEP 4: Dedicated wallets table (two-pool ledger)
-- ============================================================
CREATE TABLE IF NOT EXISTS wallets (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  pulse_points    BIGINT DEFAULT 0 CHECK (pulse_points >= 0),   -- AI Studio currency
  spin_credits    INTEGER DEFAULT 0 CHECK (spin_credits >= 0),  -- Spin Wheel currency
  lifetime_points BIGINT DEFAULT 0,                              -- Never decremented, for tier calc
  recharge_counter BIGINT DEFAULT 0,                             -- Cumulative towards next spin credit
  updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Migrate existing spin_credits from users -> wallets
INSERT INTO wallets (user_id, pulse_points, spin_credits, lifetime_points)
SELECT id, 
       COALESCE(total_points, 0),
       COALESCE(spin_credits, 0),
       COALESCE(total_points, 0)
FROM users
ON CONFLICT (user_id) DO NOTHING;

-- ============================================================
-- STEP 5: Complete spin_results table (spec section 5, REQ-3.4)
-- ============================================================
CREATE TABLE IF NOT EXISTS spin_results (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id             UUID NOT NULL REFERENCES users(id),
  prize_type          VARCHAR(30) NOT NULL 
                        CHECK (prize_type IN ('try_again','pulse_points','airtime','data_bundle','momo_cash')),
  prize_value         DECIMAL(12,2) NOT NULL DEFAULT 0,
  slot_index          INTEGER,                                    -- Which wheel slot was selected
  fulfillment_status  VARCHAR(30) DEFAULT 'pending'
                        CHECK (fulfillment_status IN (
                          'na','pending','pending_momo_setup','pending_claim',
                          'processing','completed','failed','held'
                        )),
  fulfillment_ref     VARCHAR(255),                               -- VTPass ref or MoMo X-Reference-Id
  momo_number         VARCHAR(15),                                -- MoMo target for cash prizes
  error_message       TEXT,
  retry_count         INTEGER DEFAULT 0,
  claimed_at          TIMESTAMPTZ,
  fulfilled_at        TIMESTAMPTZ,
  created_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_spin_results_user ON spin_results(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_spin_results_status ON spin_results(fulfillment_status) 
  WHERE fulfillment_status IN ('pending','processing','held');

-- ============================================================
-- STEP 6: fraud_events table (spec section 11)
-- ============================================================
CREATE TABLE IF NOT EXISTS fraud_events (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID REFERENCES users(id),
  phone_number VARCHAR(15),
  rule_name   VARCHAR(100) NOT NULL,
  severity    VARCHAR(20) DEFAULT 'medium' 
                CHECK (severity IN ('low','medium','high','critical')),
  details     JSONB DEFAULT '{}',
  resolved    BOOLEAN DEFAULT FALSE,
  resolved_by UUID REFERENCES admin_users(id),
  resolved_at TIMESTAMPTZ,
  created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fraud_events_unresolved ON fraud_events(resolved, created_at DESC) 
  WHERE resolved = FALSE;
CREATE INDEX IF NOT EXISTS idx_fraud_events_user ON fraud_events(user_id);

-- ============================================================
-- STEP 7: Complete regional_wars_snapshots (spec section 3.5, SRS REQ-5.1 to REQ-5.5)
-- ============================================================
CREATE TABLE IF NOT EXISTS regional_wars_cycles (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  cycle_number        SERIAL,
  start_at            TIMESTAMPTZ NOT NULL,
  end_at              TIMESTAMPTZ NOT NULL,
  status              VARCHAR(20) DEFAULT 'active' 
                        CHECK (status IN ('active','completed','paused','reset')),
  winning_state_code  VARCHAR(10),
  winning_bonus_pts   INTEGER,                                    -- Points awarded to winners
  bonus_awarded_at    TIMESTAMPTZ,
  created_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS regional_wars_snapshots (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  cycle_id          UUID NOT NULL REFERENCES regional_wars_cycles(id),
  state_code        VARCHAR(10) NOT NULL,
  state_name        VARCHAR(100) NOT NULL,
  total_recharge    BIGINT DEFAULT 0,                             -- In kobo
  subscriber_count  INTEGER DEFAULT 0,
  recharge_count    INTEGER DEFAULT 0,
  rank              INTEGER,
  cycle_start       TIMESTAMPTZ NOT NULL,
  cycle_end         TIMESTAMPTZ NOT NULL,
  snapshotted_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rw_snapshots_cycle ON regional_wars_snapshots(cycle_id, rank);
CREATE INDEX IF NOT EXISTS idx_rw_snapshots_state ON regional_wars_snapshots(state_code);

-- ============================================================
-- STEP 8: Scheduled and segment-specific multipliers (REQ-5.2.6, REQ-5.2.7)
-- ============================================================
CREATE TABLE IF NOT EXISTS scheduled_multipliers (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name        VARCHAR(100) NOT NULL,              -- e.g. "Double Points Weekend"
  multiplier  NUMERIC(4,2) NOT NULL DEFAULT 1.0,
  start_at    TIMESTAMPTZ NOT NULL,
  end_at      TIMESTAMPTZ NOT NULL,
  is_active   BOOLEAN DEFAULT TRUE,
  created_by  UUID REFERENCES admin_users(id),
  created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS segment_multipliers (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name            VARCHAR(100) NOT NULL,           -- e.g. "Re-engagement: Inactive 7 days"
  multiplier      NUMERIC(4,2) NOT NULL DEFAULT 1.0,
  segment_type    VARCHAR(50) NOT NULL
                    CHECK (segment_type IN ('inactive_days','state','tier','first_recharge')),
  segment_value   TEXT NOT NULL,                   -- e.g. "7" for days, "LAG" for state
  start_at        TIMESTAMPTZ,
  end_at          TIMESTAMPTZ,
  is_active       BOOLEAN DEFAULT TRUE,
  created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================
-- STEP 9: Points expiry policy config table
-- ============================================================
CREATE TABLE IF NOT EXISTS points_expiry_policies (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  expiry_enabled  BOOLEAN DEFAULT FALSE,
  expiry_days     INTEGER DEFAULT 90,              -- Days of inactivity before expiry
  expiry_type     VARCHAR(10) DEFAULT 'rolling'    -- 'rolling' (reset on recharge) | 'fixed'
                    CHECK (expiry_type IN ('rolling','fixed')),
  warn_days_before INTEGER DEFAULT 7,              -- SMS warning lead time (REQ-5.2.15)
  updated_at      TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO points_expiry_policies (expiry_enabled, expiry_days, expiry_type, warn_days_before)
VALUES (false, 90, 'rolling', 7)
ON CONFLICT DO NOTHING;

-- ============================================================
-- STEP 10: SMS templates (REQ-5.7.1)
-- ============================================================
CREATE TABLE IF NOT EXISTS sms_templates (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key         VARCHAR(100) UNIQUE NOT NULL,
  template    TEXT NOT NULL,
  description TEXT,
  updated_at  TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO sms_templates (key, template, description) VALUES
('otp_login',         'Your Loyalty Nexus verification code is {{code}}. Valid for 5 minutes.',  'OTP login delivery'),
('prize_airtime',     'Congrats! You won {{amount}} airtime on Loyalty Nexus. It has been credited to {{phone}}.',  'Airtime prize win'),
('prize_momo',        'Congrats! You won ₦{{amount}} MoMo Cash. Confirm your MoMo number in the app to receive it.', 'MoMo cash prize win'),
('streak_expiry',     'Your Loyalty Nexus streak (Day {{streak}}) expires in {{hours}} hours! Recharge now to keep it alive.', 'Streak expiry warning'),
('asset_ready',       'Your {{tool_name}} is ready on Nexus Studio! Open the app to download it before it expires.',  'AI generation ready'),
('asset_expiry',      'Your {{tool_name}} on Nexus Studio expires in 48 hours. Download it now before it is gone.',  'Asset expiry warning'),
('points_expiry',     'Your {{points}} Pulse Points expire in {{days}} days. Recharge now to keep them active.',      'Points expiry warning'),
('welcome',           'Welcome to Loyalty Nexus! You have earned {{bonus}} bonus Pulse Points. Start exploring Nexus Studio now.', 'First recharge welcome'),
('regional_wars_win', 'Congratulations! {{state}} won this round of Regional Wars! You have been awarded {{bonus}} bonus Pulse Points.', 'Regional Wars winner')
ON CONFLICT (key) DO NOTHING;

-- ============================================================
-- STEP 11: Complete network_configs seed with ALL spec parameters
-- ============================================================
-- Safe upsert helper
DO $$ BEGIN

  -- Points Engine (REQ-5.2.1 to REQ-5.2.4)
  INSERT INTO network_configs (key, value, description) VALUES
    ('points_per_250_naira',          '1',       'Base earning rate: 1 Pulse Point per ₦250 recharged'),
    ('spin_trigger_naira',            '1000',    'Cumulative naira to earn 1 Spin Credit'),
    ('min_qualifying_recharge_naira', '50',      'Minimum recharge to qualify for any point award (REQ-5.2.11)'),
    ('global_points_multiplier',      '1.0',     'Global Pulse Point multiplier — set to 2.0 for Double Points Weekend'),

    -- Streak (REQ-5.2.12, REQ-5.2.13)
    ('streak_expiry_hours',           '36',      'Hours after last recharge before streak resets to zero'),
    ('streak_grace_days_per_month',   '1',       'Grace days per month a streak is protected without recharge'),

    -- Spin Wheel (REQ-5.3.1 to REQ-5.3.6)
    ('spin_max_slots',                '16',      'Maximum slots on the spin wheel'),
    ('spin_min_slots',                '8',       'Minimum slots on the spin wheel'),
    ('spin_max_per_user_per_day',     '3',       'Max spins per user per day (fraud control)'),
    ('daily_prize_liability_cap_naira','500000', 'Daily prize liability cap in naira — forces try_again when hit'),

    -- AI Studio (REQ-5.4.1 to REQ-5.4.5)
    ('chat_daily_message_limit',      '20',      'Max Ask Nexus messages per user per day'),
    ('chat_session_timeout_minutes',  '30',      'Minutes of inactivity before session summarisation fires'),
    ('asset_retention_days',          '30',      'Days AI-generated assets are retained before deletion'),
    ('asset_expiry_warning_hours',    '48',      'Hours before asset expiry to send SMS warning'),

    -- Regional Wars (REQ-5.5.1, REQ-5.5.2)
    ('regional_wars_cycle_hours',     '24',      'Duration in hours of each Regional Wars cycle'),
    ('regional_wars_winning_bonus',   '50',      'Pulse Points awarded to all winners at end of cycle'),

    -- Streak notifications (REQ-5.7.2)
    ('streak_expiry_warning_hours',   '4',       'Hours before streak expiry to fire Ghost Nudge / SMS'),

    -- Fraud (REQ-5.6.1)
    ('fraud_max_recharges_per_hour_per_user', '5',  'Velocity threshold: recharges/hour per user'),
    ('fraud_max_recharges_per_hour_per_ip',  '10',  'Velocity threshold: recharges/hour per IP'),
    ('fraud_max_points_per_minute',          '500', 'Velocity threshold: point delta/minute triggers freeze'),
    ('fraud_max_ai_gens_per_day',            '50',  'Throttle AI generation at this daily count'),

    -- Bonus Rules (REQ-5.2.8 to REQ-5.2.10)
    ('first_recharge_bonus_points',   '20',      'Flat Pulse Points awarded on user''s very first recharge'),
    ('referral_bonus_referrer_pts',   '15',      'Points awarded to referrer on referred user''s first recharge'),
    ('referral_bonus_referee_pts',    '10',      'Points awarded to new user on their first recharge via referral'),

    -- Operation Mode (SRS REQ dual-mode)
    ('operation_mode',                'independent', 'Platform mode: independent | integrated'),
    ('ussd_shortcode',                '*789*NEXUS#',  'USSD shortcode for feature phone access'),

    -- AI Studio tool point costs (admin-overridable, REQ-5.4.2)
    ('tool_cost_ask_nexus',           '0',       'Ask Nexus chat — always free'),
    ('tool_cost_ai_photo',            '10',      'My AI Photo'),
    ('tool_cost_background_remover',  '2',       'Background Remover'),
    ('tool_cost_animate_photo',       '65',      'Animate My Photo (basic)'),
    ('tool_cost_marketing_jingle',    '100',     'My Marketing Jingle (Mubert)'),
    ('tool_cost_video_story',         '470',     'My Video Story (combined)'),
    ('tool_cost_study_guide',         '3',       'Study Guide (NotebookLM)'),
    ('tool_cost_quiz_me',             '2',       'Quiz Me (NotebookLM)'),
    ('tool_cost_mind_map',            '2',       'Mind Map (NotebookLM)'),
    ('tool_cost_deep_research',       '3',       'Deep Research Brief (NotebookLM)'),
    ('tool_cost_podcast',             '4',       'My Podcast (NotebookLM)'),
    ('tool_cost_slide_deck',          '4',       'Slide Deck (NotebookLM)'),
    ('tool_cost_infographic',         '4',       'Infographic (NotebookLM)'),
    ('tool_cost_business_plan',       '5',       'Business Plan Summary (NotebookLM + Gemini)'),
    ('tool_cost_voice_to_plan',       '6',       'Voice to Business Plan (AssemblyAI + Gemini)'),
    ('tool_cost_translate',           '2',       'Local Language Translation (Google Translate)'),
    ('tool_cost_tts',                 '5',       'Text to Speech Nigerian Voice (Google TTS)'),

    -- LLM Routing thresholds
    ('chat_groq_daily_limit',         '1000',    'Groq free tier daily request ceiling before fallback'),
    ('chat_gemini_daily_limit',       '2000',    'Groq+Gemini combined daily ceiling before DeepSeek overflow')
  ON CONFLICT (key) DO UPDATE SET
    description = EXCLUDED.description;
    -- Note: We never overwrite value on conflict (admin may have changed it)

END $$;

-- ============================================================
-- STEP 12: Indexes for performance (spec NFR 7.1 — 200ms p95)
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_transactions_user_type ON transactions(user_id, type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_phone_created ON transactions(phone_number, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_state ON users(state);
CREATE INDEX IF NOT EXISTS idx_users_streak_expires ON users(streak_expires_at) WHERE streak_expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_ai_generations_expires ON ai_generations(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ai_generations_user_status ON ai_generations(user_id, status, created_at DESC);

-- ============================================================
-- STEP 13: Updated_at auto-trigger function
-- ============================================================
CREATE OR REPLACE FUNCTION fn_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

DO $$ DECLARE
  t TEXT;
BEGIN
  FOREACH t IN ARRAY ARRAY['users','wallets','spin_results','sms_templates','network_configs',
    'wallet_passes','wallet_registrations','studio_tools','ai_generations','prize_claims',
    'scheduled_multipliers','segment_multipliers','points_expiry_policies','regional_wars_cycles'] LOOP
    EXECUTE format('
      DROP TRIGGER IF EXISTS trg_set_updated_at ON %I;
      CREATE TRIGGER trg_set_updated_at BEFORE UPDATE ON %I
        FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();', t, t);
  END LOOP;
END $$;

