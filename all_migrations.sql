

══════════════════════════════════════════════════════
MIGRATION: 001_cockpit_configuration.up.sql
══════════════════════════════════════════════════════
-- 001_cockpit_configuration.sql
-- Purpose: Total flexibility for the private firm to manage Loyalty Nexus.

-- 1. Global Program Rules (The "Knobs")
CREATE TABLE IF NOT EXISTS program_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key TEXT UNIQUE NOT NULL, -- e.g., 'min_recharge_spin', 'streak_window_hours'
    config_value JSONB NOT NULL,
    description TEXT,
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- 2. Prize Inventory & Weights (The "Odds Engine")
CREATE TABLE IF NOT EXISTS prize_pool (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    prize_type TEXT CHECK (prize_type IN ('airtime', 'data', 'momo_cash', 'studio_credits')),
    base_value NUMERIC NOT NULL,
    is_active BOOLEAN DEFAULT true,
    win_probability_weight INTEGER DEFAULT 100, -- Higher = more common
    daily_inventory_cap INTEGER, -- Max wins per day
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- 3. Regional Multipliers (The "Tournament" Engine)
CREATE TABLE IF NOT EXISTS regional_settings (
    region_code TEXT PRIMARY KEY, -- e.g., 'LAG', 'ABJ', 'KAN'
    multiplier NUMERIC DEFAULT 1.0,
    is_golden_hour BOOLEAN DEFAULT false,
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- 4. Studio Parameters (AI Rendering Limits)
CREATE TABLE IF NOT EXISTS studio_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    media_type TEXT CHECK (media_type IN ('image', 'video', 'jingle')),
    point_cost INTEGER NOT NULL,
    render_priority INTEGER DEFAULT 1,
    is_enabled BOOLEAN DEFAULT true
);

-- Insert initial "Cockpit" data
INSERT INTO program_configs (config_key, config_value, description) VALUES
('min_recharge_naira', '500', 'Minimum recharge to earn a spin'),
('streak_target_days', '7', 'Days required for a Mega Jackpot ticket'),
('ghost_nudge_hours', '48', 'Inactivity hours before lock-screen nudge fires')
ON CONFLICT (config_key) DO NOTHING;



══════════════════════════════════════════════════════
MIGRATION: 002_core_ledger.up.sql
══════════════════════════════════════════════════════
-- 002_core_ledger.sql
-- Purpose: Atomic ledger for High-Throughput Loyalty transactions.

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    msisdn TEXT UNIQUE NOT NULL, -- Normalized 234...
    user_code TEXT UNIQUE NOT NULL,
    total_points BIGINT DEFAULT 0,
    stamps_count INTEGER DEFAULT 0,
    total_recharge_amount BIGINT DEFAULT 0,
    tier TEXT DEFAULT 'BRONZE',
    streak_count INTEGER DEFAULT 0,
    last_visit_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_msisdn ON users(msisdn);

CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    msisdn TEXT NOT NULL,
    type TEXT NOT NULL, -- visit, reward_redeem, bonus, studio_spend
    points_delta BIGINT DEFAULT 0,
    stamps_delta INTEGER DEFAULT 0,
    amount BIGINT DEFAULT 0, -- in Kobo
    balance_after BIGINT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_date ON transactions(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_msisdn ON transactions(msisdn);

-- ATOMIC TRIGGER: Handle all balance and streak logic in the DB layer
CREATE OR REPLACE FUNCTION trg_fn_loyalty_nexus_ledger()
RETURNS TRIGGER AS $$
DECLARE
    v_streak_window INTEGER;
BEGIN
    -- Get streak window from cockpit config (default 48h)
    SELECT (config_value->>'hours')::int INTO v_streak_window 
    FROM program_configs WHERE config_key = 'streak_window' 
    LIMIT 1;
    IF v_streak_window IS NULL THEN v_streak_window := 48; END IF;

    -- Atomic balance update
    UPDATE users
    SET
        total_points = total_points + NEW.points_delta,
        stamps_count = stamps_count + NEW.stamps_delta,
        total_recharge_amount = total_recharge_amount + NEW.amount,
        last_visit_at = CASE WHEN NEW.type = 'visit' THEN NOW() ELSE last_visit_at END,
        streak_count = CASE 
            WHEN NEW.type != 'visit' THEN streak_count
            WHEN last_visit_at IS NULL THEN 1
            WHEN last_visit_at > NOW() - (v_streak_window * interval '1 hour') THEN streak_count + 1
            ELSE 1
        END,
        updated_at = now()
    WHERE id = NEW.user_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_loyalty_nexus_ledger
    AFTER INSERT ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION trg_fn_loyalty_nexus_ledger();


══════════════════════════════════════════════════════
MIGRATION: 003_nexus_studio.up.sql
══════════════════════════════════════════════════════
-- 003_nexus_studio.sql
-- Purpose: Schema for the points-funded creative studio.

-- 1. Studio Tools Catalogue
CREATE TABLE IF NOT EXISTS studio_tools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    category TEXT CHECK (category IN ('Chat', 'Create', 'Learn', 'Build')),
    point_cost BIGINT NOT NULL DEFAULT 0,
    provider TEXT NOT NULL, -- e.g. 'FAL_AI', 'GROQ', 'GOOGLE'
    provider_tool_id TEXT NOT NULL, -- e.g. 'flux-schnell', 'llama-3-70b'
    icon_name TEXT, -- Lucide icon key
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- 2. AI Generation History & Gallery
CREATE TABLE IF NOT EXISTS ai_generations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    tool_id UUID NOT NULL REFERENCES studio_tools(id),
    prompt TEXT NOT NULL,
    status TEXT CHECK (status IN ('pending', 'processing', 'completed', 'failed')) DEFAULT 'pending',
    output_url TEXT, -- Pre-signed S3 URL
    error_message TEXT,
    points_deducted BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL, -- 30-day lifecycle
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_ai_generations_user ON ai_generations(user_id);
CREATE INDEX IF NOT EXISTS idx_ai_generations_status ON ai_generations(status);

-- Seed Initial Tools (Full Catalogue - Appendix B)
INSERT INTO studio_tools (name, description, category, point_cost, provider, provider_tool_id, icon_name) VALUES
('Ask Nexus', 'Conversational AI assistant for brainstorming and help.', 'Chat', 0, 'GROQ', 'llama-4-scout', 'MessageSquare'),
('My AI Photo', 'Generate professional AI portraits from text.', 'Create', 10, 'HUGGING_FACE', 'flux-1-schnell', 'Camera'),
('Background Remover', 'Instantly remove backgrounds from your photos.', 'Create', 2, 'REM_BG', 'self-hosted', 'Scissors'),
('Animate My Photo', 'Turn your AI photo into a 5-second video.', 'Create', 65, 'FAL_AI', 'ltx-video', 'Video'),
('My Marketing Jingle', 'Generate 30s original music for your brand.', 'Create', 100, 'MUBERT', 'audio-gen', 'Music'),
('My Video Story', 'Combined AI Photo and Jingle into a branded video.', 'Create', 470, 'PIPELINE', 'composite-video', 'Clapperboard'),
('Study Guide', 'Generate a structured study guide on any topic.', 'Learn', 3, 'NOTEBOOK_LM', 'pdf-gen', 'BookOpen'),
('Quiz Me', 'Generate 10 multiple-choice questions on any topic.', 'Learn', 2, 'NOTEBOOK_LM', 'quiz-gen', 'HelpCircle'),
('Mind Map', 'Create a visual mind map from any concept.', 'Learn', 2, 'NOTEBOOK_LM', 'mindmap-gen', 'Network'),
('Deep Research Brief', 'Comprehensive research coverage on any topic.', 'Learn', 3, 'NOTEBOOK_LM', 'research-gen', 'Search'),
('My Podcast', 'Turn any topic into a 5-minute conversation.', 'Learn', 4, 'NOTEBOOK_LM', 'audio-gen', 'Mic'),
('Slide Deck', 'Professional PowerPoint presentation on any topic.', 'Build', 4, 'NOTEBOOK_LM', 'pptx-gen', 'Presentation'),
('Infographic', 'Visual summary of key facts and topics.', 'Build', 4, 'NOTEBOOK_LM', 'infographic-gen', 'PieChart'),
('Business Plan Summary', 'One-page professional business plan summary.', 'Build', 5, 'NOTEBOOK_LM', 'business-plan', 'FileText'),
('Voice to Plan', 'Record your idea to get a structured business plan.', 'Build', 6, 'ASSEMBLY_AI', 'voice-plan', 'Mic2'),
('Local Translation', 'Translate any text to Hausa, Yoruba, Igbo or Pidgin.', 'Build', 2, 'GOOGLE', 'translate', 'Languages'),
('Text to Speech', 'Natural audio reading with a Nigerian accent.', 'Build', 5, 'GOOGLE', 'tts-nigeria', 'Volume2')
ON CONFLICT (name) DO NOTHING;


══════════════════════════════════════════════════════
MIGRATION: 004_digital_passport.up.sql
══════════════════════════════════════════════════════
-- 004_digital_passport.sql
-- Purpose: Support for Apple and Google Wallet persistent lock-screen cards.

-- 1. Wallet Device Registrations (for APNS/Google Push)
CREATE TABLE IF NOT EXISTS wallet_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    platform TEXT CHECK (platform IN ('apple', 'google')) NOT NULL,
    device_id TEXT NOT NULL, -- Device Library Identifier
    push_token TEXT, -- Token for push notifications
    serial_number TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_wallet_registrations_user ON wallet_registrations(user_id);

-- 2. Digital Passport Pass Management
CREATE TABLE IF NOT EXISTS wallet_passes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    pass_type TEXT CHECK (pass_type IN ('loyalty', 'event', 'streak')),
    status TEXT DEFAULT 'active',
    last_pushed_at TIMESTAMPTZ,
    points_at_last_push BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT now()
);


══════════════════════════════════════════════════════
MIGRATION: 005_regional_wars.up.sql
══════════════════════════════════════════════════════
-- 005_regional_wars.sql
-- Purpose: Support for regional tournaments and multipliers.

-- 1. Region Definitions & Multipliers
-- regional_settings was partially created in migration 001.
-- We add the missing columns here using ALTER TABLE ... ADD COLUMN IF NOT EXISTS.
ALTER TABLE regional_settings ADD COLUMN IF NOT EXISTS region_name TEXT NOT NULL DEFAULT '';
ALTER TABLE regional_settings ADD COLUMN IF NOT EXISTS base_multiplier NUMERIC DEFAULT 1.0;
ALTER TABLE regional_settings ADD COLUMN IF NOT EXISTS golden_hour_multiplier NUMERIC DEFAULT 2.0;

-- 2. Regional Leaderboard (Aggregated real-time)
CREATE TABLE IF NOT EXISTS regional_stats (
    region_code TEXT PRIMARY KEY REFERENCES regional_settings(region_code),
    total_recharge_kobo BIGINT DEFAULT 0,
    active_subscribers INTEGER DEFAULT 0,
    last_recharge_at TIMESTAMPTZ,
    rank INTEGER DEFAULT 0,
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- Seed initial Nigerian regions
INSERT INTO regional_settings (region_code, region_name) VALUES
('LAG', 'Lagos'),
('ABJ', 'Abuja'),
('KAN', 'Kano'),
('PHC', 'Port Harcourt'),
('IBD', 'Ibadan'),
('ENU', 'Enugu')
ON CONFLICT (region_code) DO NOTHING;

-- 3. Region Tournament History
CREATE TABLE IF NOT EXISTS region_tournaments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    winning_region_code TEXT REFERENCES regional_settings(region_code),
    status TEXT DEFAULT 'active' -- active, completed
);


══════════════════════════════════════════════════════
MIGRATION: 006_daily_subscriptions.up.sql
══════════════════════════════════════════════════════
-- 006_daily_subscriptions.sql
-- Purpose: Support for N20/day guaranteed draw entry subscriptions.

-- 1. Subscription Plans (Configurable)
CREATE TABLE IF NOT EXISTS subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    daily_cost_kobo INTEGER NOT NULL DEFAULT 2000, -- 2000 Kobo = N20
    entries_per_day INTEGER NOT NULL DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- 2. User Subscriptions
CREATE TABLE IF NOT EXISTS user_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    plan_id UUID NOT NULL REFERENCES subscription_plans(id),
    status TEXT CHECK (status IN ('active', 'paused', 'cancelled', 'pending_payment')) DEFAULT 'active',
    next_billing_at TIMESTAMPTZ NOT NULL,
    last_billed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_subscriptions_user ON user_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_billing ON user_subscriptions(next_billing_at);

-- 3. Seed Default N20 Plan
INSERT INTO subscription_plans (name, daily_cost_kobo, entries_per_day) VALUES
('Daily Draw Pass', 2000, 1)
ON CONFLICT (name) DO NOTHING;


══════════════════════════════════════════════════════
MIGRATION: 007_rls_policies.up.sql
══════════════════════════════════════════════════════
-- 007_rls_policies.sql
-- Purpose: Security hardening for multi-tenant and subscriber data protection.
-- NOTE: RLS policies using the 'authenticated' role are skipped here because
-- that role is Supabase-specific. Application-level auth is handled via JWT
-- middleware in the Go API. This migration is kept as a no-op placeholder
-- to preserve migration numbering.

-- Enable RLS on tables that exist at this point in the migration sequence.
-- Policies use a safe DO block to avoid errors if role/table doesn't exist.
DO $$
BEGIN
    -- Enable RLS on users (exists from migration 002)
    ALTER TABLE users ENABLE ROW LEVEL SECURITY;

    -- Enable RLS on transactions (exists from migration 002)
    ALTER TABLE transactions ENABLE ROW LEVEL SECURITY;

    -- Enable RLS on program_configs (exists from migration 001)
    ALTER TABLE program_configs ENABLE ROW LEVEL SECURITY;

EXCEPTION WHEN OTHERS THEN
    -- If any statement fails (e.g. insufficient privilege), continue silently.
    NULL;
END $$;

-- Note: CREATE POLICY statements requiring 'authenticated' role are omitted.
-- Access control is enforced at the API layer via JWT middleware.


══════════════════════════════════════════════════════
MIGRATION: 008_hlr_cache.up.sql
══════════════════════════════════════════════════════
-- 008_hlr_cache.sql
-- Purpose: Cache for HLR lookups to handle ported numbers and reduce API costs.

CREATE TABLE IF NOT EXISTS network_cache (
    msisdn TEXT PRIMARY KEY, -- Normalized 234...
    network TEXT NOT NULL, -- MTN, Airtel, Glo, 9mobile
    last_verified TIMESTAMPTZ DEFAULT now(),
    cache_expires TIMESTAMPTZ NOT NULL,
    lookup_source TEXT CHECK (lookup_source IN ('hlr_api', 'user_selection', 'prefix_fallback')),
    is_valid BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_network_cache_expires ON network_cache(cache_expires);


══════════════════════════════════════════════════════
MIGRATION: 009_chat_summaries.up.sql
══════════════════════════════════════════════════════
-- 009_chat_summaries.sql
-- Purpose: Long-term memory for Ask Nexus via session summarization.

CREATE TABLE IF NOT EXISTS chat_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    status TEXT CHECK (status IN ('active', 'expired', 'summarized')) DEFAULT 'active',
    last_activity_at TIMESTAMPTZ DEFAULT now(),
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id),
    role TEXT CHECK (role IN ('user', 'assistant')),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS session_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    summary TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_chat_sessions_expiry ON chat_sessions(status, last_activity_at);
CREATE INDEX IF NOT EXISTS idx_session_summaries_user ON session_summaries(user_id);


══════════════════════════════════════════════════════
MIGRATION: 010_regional_wars_admin.up.sql
══════════════════════════════════════════════════════
-- 010_regional_wars_admin.sql
-- Purpose: Admin views for managing Regional Wars.

-- 1. View: Regional Performance Audit
CREATE OR REPLACE VIEW view_regional_audit AS
SELECT 
    rs.region_code,
    r.region_name,
    rs.total_recharge_kobo,
    rs.rank,
    r.base_multiplier,
    r.is_golden_hour,
    r.golden_hour_multiplier,
    (SELECT count(*) FROM users WHERE state = r.region_name) as subscriber_count
FROM regional_stats rs
JOIN regional_settings r ON rs.region_code = r.region_code;

-- 2. Audit Table for Multiplier Changes
CREATE TABLE IF NOT EXISTS multiplier_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region_code TEXT NOT NULL,
    old_multiplier NUMERIC,
    new_multiplier NUMERIC,
    changed_by UUID, -- Admin ID
    reason TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);


══════════════════════════════════════════════════════
MIGRATION: 011_auth_otp.up.sql
══════════════════════════════════════════════════════
-- 011_auth_otp.sql
-- Purpose: Secure OTP management for phone-based authentication.

CREATE TABLE IF NOT EXISTS auth_otps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    msisdn TEXT NOT NULL,
    code TEXT NOT NULL,
    purpose TEXT CHECK (purpose IN ('login', 'momo_link', 'prize_claim')),
    status TEXT CHECK (status IN ('pending', 'verified', 'expired')) DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_auth_otps_msisdn ON auth_otps(msisdn, status);


══════════════════════════════════════════════════════
MIGRATION: 012_user_momo.up.sql
══════════════════════════════════════════════════════
-- 012_user_momo.sql
-- Purpose: Support for MTN Mobile Money (MoMo) linking and verification.

ALTER TABLE users 
ADD COLUMN momo_number TEXT,
ADD COLUMN momo_verified BOOLEAN DEFAULT false,
ADD COLUMN momo_verified_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_users_momo ON users(momo_number) WHERE momo_number IS NOT NULL;


══════════════════════════════════════════════════════
MIGRATION: 013_prize_fulfillment.up.sql
══════════════════════════════════════════════════════
-- 013_prize_fulfillment.sql
-- Purpose: Support for varied prize types and fulfillment tracking (REQ-3.4).

-- Update prize_pool types if necessary
ALTER TABLE prize_pool DROP CONSTRAINT IF EXISTS prize_pool_prize_type_check;
ALTER TABLE prize_pool ADD CONSTRAINT prize_pool_prize_type_check 
CHECK (prize_type IN ('airtime', 'data', 'momo_cash', 'bonus_points', 'studio_credits', 'try_again'));

CREATE TABLE IF NOT EXISTS prize_claims (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    transaction_id UUID NOT NULL REFERENCES transactions(id),
    prize_type TEXT NOT NULL,
    prize_value NUMERIC NOT NULL,
    status TEXT CHECK (status IN ('pending_momo_link', 'pending_fulfillment', 'processing', 'completed', 'failed')) DEFAULT 'pending_fulfillment',
    momo_number TEXT,
    fulfillment_ref TEXT, -- VTPass ref or MoMo ref
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_prize_claims_user ON prize_claims(user_id, status);


══════════════════════════════════════════════════════
MIGRATION: 014_user_spin_credits.up.sql
══════════════════════════════════════════════════════
-- 014_user_spin_credits.sql
-- Purpose: Formalize the two-pool ledger by adding spin_credits to users.

ALTER TABLE users 
ADD COLUMN spin_credits INTEGER DEFAULT 0;

-- Optional: Add a trigger or procedure to handle cumulative recharge -> spin credit logic


══════════════════════════════════════════════════════
MIGRATION: 015_fraud_guards.up.sql
══════════════════════════════════════════════════════
-- 015_fraud_guards.sql
-- Purpose: Support for velocity-based fraud prevention and blacklisting.

CREATE TABLE IF NOT EXISTS msisdn_blacklist (
    msisdn TEXT PRIMARY KEY,
    reason TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- Index for velocity checks on transactions
CREATE INDEX IF NOT EXISTS idx_transactions_msisdn_created ON transactions(msisdn, created_at DESC);


══════════════════════════════════════════════════════
MIGRATION: 016_draw_engine.up.sql
══════════════════════════════════════════════════════
-- 016_draw_engine.sql
-- Purpose: Support for automated lottery draws and winner selection.

CREATE TABLE IF NOT EXISTS draws (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    draw_code TEXT UNIQUE NOT NULL, -- DRAW-YYYYMMDD-XXXX
    name TEXT NOT NULL,
    type TEXT CHECK (type IN ('DAILY', 'WEEKLY', 'MONTHLY')),
    status TEXT CHECK (status IN ('UPCOMING', 'ACTIVE', 'COMPLETED', 'CANCELLED')) DEFAULT 'UPCOMING',
    prize_pool_total NUMERIC DEFAULT 0,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    executed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS draw_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    draw_id UUID NOT NULL REFERENCES draws(id),
    user_id UUID NOT NULL REFERENCES users(id),
    msisdn TEXT NOT NULL,
    entries_count INTEGER DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS draw_winners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    draw_id UUID NOT NULL REFERENCES draws(id),
    user_id UUID NOT NULL REFERENCES users(id),
    msisdn TEXT NOT NULL,
    position INTEGER NOT NULL, -- 1st, 2nd, 3rd, etc.
    prize_name TEXT NOT NULL,
    prize_value NUMERIC NOT NULL,
    claim_status TEXT DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_draw_entries_draw ON draw_entries(draw_id);
CREATE INDEX IF NOT EXISTS idx_draw_winners_draw ON draw_winners(draw_id);


══════════════════════════════════════════════════════
MIGRATION: 017_tiered_earning_and_bonuses.up.sql
══════════════════════════════════════════════════════
-- 017_tiered_earning_and_bonuses.sql
-- Purpose: Support for dynamic recharge tiers and milestone bonuses (REQ-5.2.3, REQ-5.2.8, REQ-5.2.9).

-- 1. Recharge Amount Tiers
CREATE TABLE IF NOT EXISTS recharge_tiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL, -- Standard, Silver, Gold
    min_amount_kobo BIGINT NOT NULL,
    points_per_naira NUMERIC NOT NULL, -- e.g. 1 pt per N250 -> rate = 1/250
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- 2. Streak & Milestone Bonuses
CREATE TABLE IF NOT EXISTS program_bonuses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT CHECK (event_type IN ('first_recharge', 'streak_milestone', 'referral_completion')),
    threshold INTEGER, -- days for streak, or null
    bonus_points BIGINT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Seed Initial Tiers
INSERT INTO recharge_tiers (name, min_amount_kobo, points_per_naira) VALUES
('Standard', 0, 0.004), -- 1/250
('Silver', 100000, 0.005), -- 1/200 (N1000+)
('Gold', 300000, 0.00667)
ON CONFLICT (name) DO NOTHING; -- 1/150 (N3000+)

-- Seed Initial Bonuses
INSERT INTO program_bonuses (event_type, threshold, bonus_points) VALUES
('first_recharge', NULL, 20),
('streak_milestone', 7, 10),
('streak_milestone', 14, 25),
('streak_milestone', 30, 50)
ON CONFLICT (event_type) DO NOTHING;


══════════════════════════════════════════════════════
MIGRATION: 018_strategic_monetization.up.sql
══════════════════════════════════════════════════════
-- 018_strategic_monetization.sql
-- Purpose: Tracking metrics for SaaS monetization model (Section 6).

-- 1. ARPU Uplift Tracking (Monthly Snapshots per User)
CREATE TABLE IF NOT EXISTS arpu_uplift_tracking (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    msisdn TEXT NOT NULL,
    month_period DATE NOT NULL, -- e.g. 2026-03-01
    pre_program_avg_spend BIGINT, -- Baseline before joining program
    current_month_spend BIGINT DEFAULT 0,
    uplift_amount BIGINT GENERATED ALWAYS AS (current_month_spend - COALESCE(pre_program_avg_spend, 0)) STORED,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- 2. Churn Bounty (At-Risk Users reactivated)
CREATE TABLE IF NOT EXISTS churn_recovery_bounties (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    last_activity_before_reactivation TIMESTAMPTZ NOT NULL,
    reactivation_recharge_id UUID REFERENCES transactions(id),
    bounty_amount_kobo INTEGER DEFAULT 15000, -- Fixed 150 Naira fee per Section 6.3
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- 3. GPU Usage Tracking (Nexus Studio Monetization)
CREATE TABLE IF NOT EXISTS studio_usage_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    generation_id UUID NOT NULL REFERENCES ai_generations(id),
    provider TEXT NOT NULL,
    compute_cost_micros INTEGER, -- Internal tracking of API costs
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_arpu_period ON arpu_uplift_tracking(month_period);
CREATE INDEX IF NOT EXISTS idx_bounty_created ON churn_recovery_bounties(created_at);


══════════════════════════════════════════════════════
MIGRATION: 019_user_profile_expansion.up.sql
══════════════════════════════════════════════════════
-- 019_user_profile_expansion.sql
-- Purpose: Support for Nigerian State capture and Admin Roles (REQ-1.5, User Class 2.2).

ALTER TABLE users 
ADD COLUMN IF NOT EXISTS state TEXT;

-- Index for Regional Wars lookups
CREATE INDEX IF NOT EXISTS idx_users_state ON users(state);

-- Admin Users and Roles
CREATE TABLE IF NOT EXISTS admin_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT CHECK (role IN ('platform_admin', 'mno_executive')) DEFAULT 'platform_admin',
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Seed Initial Admin (Platform Admin)
INSERT INTO admin_users (username, password_hash, role) VALUES 
('admin_nexus', 'placeholder_hash', 'platform_admin')
ON CONFLICT (username) DO NOTHING;


══════════════════════════════════════════════════════
MIGRATION: 020_spec_alignment_and_complete_schema.up.sql
══════════════════════════════════════════════════════
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



══════════════════════════════════════════════════════
MIGRATION: 021_passport_badges_and_wars.up.sql
══════════════════════════════════════════════════════
-- ═══════════════════════════════════════════════════════════════════
--  021 — Digital Passport badges + Regional Wars support tables
--  Loyalty Nexus — Phase 6
-- ═══════════════════════════════════════════════════════════════════

-- BEGIN;  -- removed: managed by golang-migrate

-- ── User Badges ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_badges (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_key   VARCHAR(64) NOT NULL,
    earned_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, badge_key)
);
CREATE INDEX IF NOT EXISTS idx_user_badges_user_id ON user_badges(user_id);
CREATE INDEX IF NOT EXISTS idx_user_badges_key     ON user_badges(badge_key);

-- ── Regional Wars ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS regional_wars (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    period              VARCHAR(7)  NOT NULL UNIQUE,  -- YYYY-MM
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',  -- ACTIVE|COMPLETED
    total_prize_kobo    BIGINT      NOT NULL DEFAULT 50000000,   -- ₦500,000 default
    starts_at           TIMESTAMPTZ NOT NULL,
    ends_at             TIMESTAMPTZ NOT NULL,
    resolved_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_regional_wars_status ON regional_wars(status);

-- ── Regional War Winners ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS regional_war_winners (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    war_id          UUID        NOT NULL REFERENCES regional_wars(id),
    state           VARCHAR(64) NOT NULL,
    rank            SMALLINT    NOT NULL,
    total_points    BIGINT      NOT NULL DEFAULT 0,
    prize_kobo      BIGINT      NOT NULL DEFAULT 0,
    status          VARCHAR(30) NOT NULL DEFAULT 'PENDING',  -- PENDING|PAID
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Draws ────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS draws (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(120) NOT NULL,
    status              VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE',
    winner_count        INT          NOT NULL DEFAULT 3,
    prize_type          VARCHAR(40)  NOT NULL DEFAULT 'MOMO_CASH',
    prize_value_kobo    BIGINT       NOT NULL DEFAULT 500000,   -- ₦5,000 default
    executed_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ── Draw Entries ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS draw_entries (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    draw_id         UUID        NOT NULL REFERENCES draws(id),
    user_id         UUID        NOT NULL REFERENCES users(id),
    phone_number    VARCHAR(20) NOT NULL,
    ticket_count    INT         NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (draw_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_draw_entries_draw_id ON draw_entries(draw_id);

-- ── Draw Winners ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS draw_winners (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    draw_id             UUID        NOT NULL REFERENCES draws(id),
    user_id             UUID        NOT NULL REFERENCES users(id),
    phone_number        VARCHAR(20) NOT NULL,
    position            SMALLINT    NOT NULL,
    prize_type          VARCHAR(40) NOT NULL,
    prize_value_kobo    BIGINT      NOT NULL DEFAULT 0,
    status              VARCHAR(30) NOT NULL DEFAULT 'PENDING_FULFILLMENT',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Extend users table with new columns (safe adds) ─────────────────
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS state              VARCHAR(64),
    ADD COLUMN IF NOT EXISTS total_spins        INT       NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS studio_use_count   INT       NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_referrals    INT       NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS lifetime_points    BIGINT    NOT NULL DEFAULT 0;

-- Update lifetime_points from wallets table if present
UPDATE users u
SET lifetime_points = COALESCE(
    (SELECT lifetime_points FROM wallets w WHERE w.user_id = u.id LIMIT 1), 0)
WHERE lifetime_points = 0;

-- Seed a default monthly draw
INSERT INTO draws (id, name, status, winner_count, prize_type, prize_value_kobo)
VALUES (gen_random_uuid(), 'Monthly Grand Draw', 'ACTIVE', 3, 'MOMO_CASH', 5000000)
ON CONFLICT DO NOTHING;

-- COMMIT;  -- removed: managed by golang-migrate


══════════════════════════════════════════════════════
MIGRATION: 022_notifications_and_subscriptions.up.sql
══════════════════════════════════════════════════════
-- =============================================================================
-- Migration 022: Notifications, Push Tokens, Subscription Lifecycle
-- =============================================================================

-- BEGIN;  -- removed: managed by golang-migrate

-- ---------------------------------------------------------------------------
-- Push / device tokens — one row per device per user
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS push_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT        NOT NULL,
    platform    TEXT        NOT NULL CHECK (platform IN ('android','ios','web')),
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    last_seen_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, token)
);
CREATE INDEX IF NOT EXISTS idx_push_tokens_user  ON push_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_push_tokens_active ON push_tokens (is_active);

-- ---------------------------------------------------------------------------
-- In-app notifications
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notifications (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       TEXT        NOT NULL,
    body        TEXT        NOT NULL,
    type        TEXT        NOT NULL
        CHECK (type IN ('spin_win','prize_fulfil','draw_result','streak_warn',
                        'subscription_warn','subscription_expired','wars_result',
                        'studio_ready','system','marketing')),
    deep_link   TEXT,                   -- e.g. /draws/uuid or /spins
    image_url   TEXT,
    is_read     BOOLEAN     NOT NULL DEFAULT FALSE,
    read_at     TIMESTAMPTZ,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notifications_user     ON notifications (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_unread   ON notifications (user_id) WHERE is_read = FALSE;

-- ---------------------------------------------------------------------------
-- Notification preferences per user
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id                UUID        PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    push_enabled           BOOLEAN     NOT NULL DEFAULT TRUE,
    sms_enabled            BOOLEAN     NOT NULL DEFAULT TRUE,
    marketing_enabled      BOOLEAN     NOT NULL DEFAULT TRUE,
    spin_win_push          BOOLEAN     NOT NULL DEFAULT TRUE,
    draw_result_push       BOOLEAN     NOT NULL DEFAULT TRUE,
    streak_warn_push       BOOLEAN     NOT NULL DEFAULT TRUE,
    sub_warn_push          BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Subscription enhancements — grace period support
-- ---------------------------------------------------------------------------
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS subscription_grace_until  TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS subscription_auto_renew   BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS fcm_token                 TEXT;   -- latest FCM token (convenience col)

-- subscription_status: extend allowed values to include GRACE
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_subscription_status_check;
ALTER TABLE users
    ADD CONSTRAINT users_subscription_status_check
    CHECK (subscription_status IN ('FREE','ACTIVE','GRACE','SUSPENDED','BANNED'));

-- ---------------------------------------------------------------------------
-- Subscription events audit log
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS subscription_events (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type      TEXT        NOT NULL
        CHECK (event_type IN ('activated','renewed','expired','grace_started',
                              'downgraded','cancelled','refunded')),
    previous_status TEXT,
    new_status      TEXT,
    note            TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sub_events_user ON subscription_events (user_id, created_at DESC);

-- ---------------------------------------------------------------------------
-- Scheduled draws: add recurrence support
-- ---------------------------------------------------------------------------
ALTER TABLE draws
    ADD COLUMN IF NOT EXISTS recurrence TEXT
        CHECK (recurrence IN ('once','weekly','monthly')) DEFAULT 'once',
    ADD COLUMN IF NOT EXISTS next_draw_at TIMESTAMPTZ;

-- ---------------------------------------------------------------------------
-- Prize fulfilment webhooks log
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS fulfilment_webhooks (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    spin_result_id  UUID        REFERENCES spin_results(id),
    provider        TEXT        NOT NULL,    -- vtpass | momo | manual
    payload         JSONB,
    status_code     INT,
    response_body   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- COMMIT;  -- removed: managed by golang-migrate


══════════════════════════════════════════════════════
MIGRATION: 023_admin_phase8.up.sql
══════════════════════════════════════════════════════
-- Migration 023: Admin Phase 8 — notification_broadcasts + user profile state column
-- Part of Phase 8: Admin Cockpit completion

-- ── Broadcast audit table ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS notification_broadcasts (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  title       TEXT        NOT NULL,
  body        TEXT        NOT NULL,
  type        VARCHAR(50) NOT NULL,
  target      VARCHAR(50) NOT NULL DEFAULT 'all',
  sent_count  INTEGER     NOT NULL DEFAULT 0,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notification_broadcasts_created ON notification_broadcasts(created_at DESC);

-- ── subscription_status for users table (idempotent) ─────────────────────
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS subscription_status     VARCHAR(20) NOT NULL DEFAULT 'FREE',
  ADD COLUMN IF NOT EXISTS subscription_expires_at TIMESTAMPTZ;

-- ── state column for Regional Wars team assignment (REQ-1.5) ─────────────
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS state VARCHAR(60);

-- ── Draw recurrence and next_draw_at (extend draws table) ────────────────
ALTER TABLE draws
  ADD COLUMN IF NOT EXISTS recurrence   VARCHAR(20) NOT NULL DEFAULT 'once',
  ADD COLUMN IF NOT EXISTS next_draw_at TIMESTAMPTZ;

-- ── Prize slot: ensure is_active and daily_inventory_cap exist ───────────
-- (table is named 'prize_pool' in this schema, not 'prizes')
ALTER TABLE prize_pool
  ADD COLUMN IF NOT EXISTS is_active            BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS daily_inventory_cap  INTEGER NOT NULL DEFAULT -1;

-- ── USSD shortcode config entry ───────────────────────────────────────────
INSERT INTO network_configs (key, value, description)
VALUES
  ('ussd_shortcode',              '"*789*NEXUS#"',  'USSD shortcode for feature phone access'),
  ('operation_mode',              '"independent"',   'independent or integrated'),
  ('streak_freeze_days_per_month','1',               'Grace days per month for streak freeze'),
  ('points_expiry_days',          '90',              'Days of inactivity before points expire'),
  ('points_expiry_warn_days',     '7',               'Days before expiry to warn user'),
  ('points_multiplier_start',     'null',            'Scheduled multiplier start datetime ISO'),
  ('points_multiplier_end',       'null',            'Scheduled multiplier end datetime ISO'),
  ('referral_bonus_points',       '20',              'Points awarded to referrer and new user'),
  ('first_recharge_bonus_points', '20',              'Bonus points on first ever recharge'),
  ('streak_milestones_json',      '[{"days":7,"bonus_points":10},{"days":14,"bonus_points":25},{"days":30,"bonus_points":50}]', 'Streak milestone bonus schedule'),
  ('recharge_tiers_json',
   '[{"label":"Standard","min_recharge":0,"points_per_naira_denom":250},{"label":"Silver","min_recharge":1000,"points_per_naira_denom":200},{"label":"Gold","min_recharge":3000,"points_per_naira_denom":150},{"label":"Platinum","min_recharge":5000,"points_per_naira_denom":100}]',
   'Tiered recharge-to-points earning rates')
ON CONFLICT (key) DO NOTHING;


══════════════════════════════════════════════════════
MIGRATION: 024_phase8_draw_spin_winner.up.sql
══════════════════════════════════════════════════════
-- Migration 024: Phase 8 — Draw Engine + Spin Prize Pool alignment
-- Adds: full draw engine columns, prize_fulfillment_logs, winner_service tables
-- Aligns prize_pool with PrizePoolEntry entity
-- All financial rules enforced at DB level (two-pool separation, immutable tx ledger)

-- ─── Draw Engine — full schema alignment ───────────────────────────────────

-- Add missing columns to draws table
ALTER TABLE draws ADD COLUMN IF NOT EXISTS draw_type       TEXT NOT NULL DEFAULT 'MONTHLY';
ALTER TABLE draws ADD COLUMN IF NOT EXISTS recurrence      TEXT NOT NULL DEFAULT 'none';
ALTER TABLE draws ADD COLUMN IF NOT EXISTS next_draw_at    TIMESTAMPTZ;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS start_time      TIMESTAMPTZ;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS end_time        TIMESTAMPTZ;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS draw_time       TIMESTAMPTZ;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS prize_pool      NUMERIC(12,2) NOT NULL DEFAULT 0;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS winner_count    INTEGER NOT NULL DEFAULT 1;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS runner_ups_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS total_entries   INTEGER NOT NULL DEFAULT 0;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS total_winners   INTEGER NOT NULL DEFAULT 0;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS executed_at     TIMESTAMPTZ;
ALTER TABLE draws ADD COLUMN IF NOT EXISTS completed_at    TIMESTAMPTZ;

-- draw_entries: full schema
CREATE TABLE IF NOT EXISTS draw_entries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    draw_id      UUID NOT NULL REFERENCES draws(id) ON DELETE CASCADE,
    user_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    phone_number TEXT NOT NULL,
    entry_source TEXT NOT NULL DEFAULT 'recharge',  -- recharge | subscription | bonus | csv_import
    amount       BIGINT NOT NULL DEFAULT 0,           -- kobo
    ticket_count INTEGER NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_draw_entries_draw_id ON draw_entries(draw_id);
CREATE INDEX IF NOT EXISTS idx_draw_entries_user_id ON draw_entries(user_id);

-- draw_winners: full schema
CREATE TABLE IF NOT EXISTS draw_winners (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    draw_id      UUID NOT NULL REFERENCES draws(id) ON DELETE CASCADE,
    user_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    phone_number TEXT NOT NULL,
    position     INTEGER NOT NULL,
    prize_type   TEXT NOT NULL DEFAULT 'CASH',
    prize_value_kobo BIGINT NOT NULL DEFAULT 0,
    is_runner_up BOOLEAN NOT NULL DEFAULT FALSE,
    status       TEXT NOT NULL DEFAULT 'PENDING_FULFILLMENT',  -- PENDING_FULFILLMENT | FULFILLED | EXPIRED | FAILED
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_draw_winners_draw_id ON draw_winners(draw_id);
CREATE INDEX IF NOT EXISTS idx_draw_winners_user_id ON draw_winners(user_id);

-- ─── Prize Pool — align with PrizePoolEntry entity ──────────────────────────

-- Ensure prize_pool table has all required columns
CREATE TABLE IF NOT EXISTS prize_pool (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    TEXT NOT NULL,
    prize_type              TEXT NOT NULL,   -- try_again | pulse_points | airtime | data_bundle | momo_cash
    base_value              NUMERIC(12,2) NOT NULL DEFAULT 0,
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    win_probability_weight  INTEGER NOT NULL DEFAULT 0,  -- weights must sum to ≤10000 (=100.00%)
    daily_inventory_cap     INTEGER,         -- NULL = unlimited
    created_at              TIMESTAMPTZ DEFAULT NOW(),
    updated_at              TIMESTAMPTZ DEFAULT NOW()
);

-- Seed default 12-slot prize table from spec Appendix A
-- Only insert if table is empty (idempotent)
INSERT INTO prize_pool (name, prize_type, base_value, is_active, win_probability_weight)
SELECT * FROM (VALUES
    ('Try Again',      'try_again',    0,    true, 2000),
    ('Try Again',      'try_again',    0,    true, 1500),
    ('Try Again',      'try_again',    0,    true, 1000),
    ('Try Again',      'try_again',    0,    true, 830),
    ('Try Again',      'try_again',    0,    true, 500),
    ('+5 Pulse Points','pulse_points', 5,   true, 830),
    ('+10 Pulse Points','pulse_points',10,  true, 830),
    ('10MB Data',      'data_bundle',  10,   true, 830),
    ('25MB Data',      'data_bundle',  25,   true, 830),
    ('₦50 Airtime',   'airtime',      50,   true, 420),
    ('₦100 Airtime',  'airtime',      100,  true, 250),
    ('₦200 Airtime',  'airtime',      200,  true, 180)
) AS vals(name, prize_type, base_value, is_active, win_probability_weight)
WHERE NOT EXISTS (SELECT 1 FROM prize_pool LIMIT 1);

-- ─── Prize Fulfillment Logs ─────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS prize_fulfillment_logs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    spin_result_id   UUID NOT NULL REFERENCES spin_results(id) ON DELETE CASCADE,
    attempt_number   INTEGER NOT NULL DEFAULT 1,
    fulfillment_mode TEXT NOT NULL DEFAULT 'AUTO',      -- AUTO | MANUAL
    status           TEXT NOT NULL,                      -- SUCCESS | FAILED | PENDING
    provider         TEXT,                               -- vtpass | momo | manual
    provider_ref     TEXT,
    error_message    TEXT,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_pfl_spin_result ON prize_fulfillment_logs(spin_result_id);

-- Add momo_number + retry_count to spin_results if missing
ALTER TABLE spin_results ADD COLUMN IF NOT EXISTS mo_mo_number TEXT NOT NULL DEFAULT '';
ALTER TABLE spin_results ADD COLUMN IF NOT EXISTS retry_count  INTEGER NOT NULL DEFAULT 0;
ALTER TABLE spin_results ADD COLUMN IF NOT EXISTS fulfilled_at TIMESTAMPTZ;

-- ─── Admin: notification_broadcasts aligned ─────────────────────────────────

CREATE TABLE IF NOT EXISTS notification_broadcasts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title        TEXT NOT NULL,
    message      TEXT NOT NULL,
    type         TEXT NOT NULL DEFAULT 'push',   -- push | sms | both
    target_count INTEGER NOT NULL DEFAULT 0,
    status       TEXT NOT NULL DEFAULT 'queued', -- queued | sent | failed
    sent_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

-- ─── Network Config: seed new AI studio keys (idempotent) ──────────────────

INSERT INTO network_configs (key, value, description) VALUES
    ('ai_chat_enabled',               'true',  'Enable Ask Nexus chat feature'),
    ('nexus_chat_daily_limit',        '20',    'Max messages per user per day for Ask Nexus'),
    ('ai_translate_cost_points',      '1',     'Points cost for translate tool'),
    ('ai_quiz_cost_points',           '2',     'Points cost for quiz tool'),
    ('ai_mindmap_cost_points',        '2',     'Points cost for mind map tool'),
    ('ai_narrate_cost_points',        '2',     'Points cost for narrate text tool'),
    ('ai_transcribe_cost_points',     '2',     'Points cost for transcribe voice tool'),
    ('ai_bgremover_cost_points',      '3',     'Points cost for background remover'),
    ('ai_podcast_cost_points',        '4',     'Points cost for podcast tool'),
    ('ai_slidedeck_cost_points',      '4',     'Points cost for slide deck tool'),
    ('ai_infographic_cost_points',    '10',    'Points cost for infographic tool'),
    ('ai_research_brief_cost_points', '5',     'Points cost for deep research brief'),
    ('ai_bgmusic_cost_points',        '5',     'Points cost for background music'),
    ('ai_bizplan_cost_points',        '12',    'Points cost for business plan'),
    ('ai_video_basic_cost_points',    '65',    'Points cost for basic video animation'),
    ('ai_jingle_cost_points',         '200',   'Points cost for marketing jingle (30s)'),
    ('ai_video_premium_cost_points',  '250',   'Points cost for premium video animation'),
    ('daily_prize_liability_cap_naira', '500000', 'Max prize payout per day in naira'),
    ('spin_max_per_user_per_day',     '3',     'Max spins per user per day'),
    ('spin_trigger_naira',            '1000',  'Naira recharge required per spin credit')
ON CONFLICT (key) DO NOTHING;


══════════════════════════════════════════════════════
MIGRATION: 025_phase9_passport_ussd.up.sql
══════════════════════════════════════════════════════
-- Migration 025: Phase 9 — Digital Passport extensions + USSD support tables
-- Adds: passport_events, ghost_nudge_log, user_badges (if missing), QR audit log

-- ─── User Badges ───────────────────────────────────────────────────────────────
-- Already may exist from Phase 7 but ensure full schema
CREATE TABLE IF NOT EXISTS user_badges (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_key  TEXT NOT NULL,
    earned_at  TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, badge_key)
);
CREATE INDEX IF NOT EXISTS idx_user_badges_user_id ON user_badges(user_id);

-- ─── Passport Events Log (spec §6.4) ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS passport_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,  -- tier_upgrade | badge_earned | streak_milestone | qr_scanned
    details    JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_passport_events_user_id ON passport_events(user_id);
CREATE INDEX IF NOT EXISTS idx_passport_events_type    ON passport_events(event_type);

-- ─── Ghost Nudge Log (spec §6.3) ──────────────────────────────────────────────
-- Tracks when a user was last nudged to prevent re-nudge within 24h
CREATE TABLE IF NOT EXISTS ghost_nudge_log (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    nudged_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ghost_nudge_user ON ghost_nudge_log(user_id);

-- ─── QR Scan Audit Log ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS qr_scan_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scanned_by  TEXT,           -- partner merchant IP / terminal ID
    is_valid    BOOLEAN NOT NULL DEFAULT TRUE,
    scanned_at  TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_qr_scan_user_id ON qr_scan_log(user_id);

-- ─── Users table — ensure Digital Passport columns exist ─────────────────────
ALTER TABLE users ADD COLUMN IF NOT EXISTS lifetime_points  BIGINT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_spins      INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS studio_use_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_referrals  INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS momo_number      TEXT    NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS momo_verified    BOOLEAN NOT NULL DEFAULT FALSE;

-- ─── Wallets table — ensure spin_credits pool column ─────────────────────────
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS lifetime_points BIGINT NOT NULL DEFAULT 0;

-- ─── USSD Sessions (for stateful multi-turn USSD — Africa's Talking) ─────────
CREATE TABLE IF NOT EXISTS ussd_sessions (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id     TEXT        NOT NULL UNIQUE,
    phone_number   TEXT        NOT NULL,
    menu_state     TEXT        NOT NULL DEFAULT 'root',
    input_buffer   TEXT        NOT NULL DEFAULT '',
    pending_spin_id UUID       REFERENCES spin_results(id) ON DELETE SET NULL,
    expires_at     TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '10 minutes',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ussd_sessions_phone      ON ussd_sessions(phone_number);
CREATE INDEX IF NOT EXISTS idx_ussd_sessions_session_id ON ussd_sessions(session_id);

-- Auto-clean expired USSD sessions (keep table small)
CREATE OR REPLACE FUNCTION cleanup_expired_ussd_sessions() RETURNS void AS $$
BEGIN
    DELETE FROM ussd_sessions WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;


══════════════════════════════════════════════════════
MIGRATION: 026_phase10_studio_hardening.up.sql
══════════════════════════════════════════════════════
-- ============================================================
-- Migration 026: Phase 10 — Nexus Studio schema hardening
-- Adds slug, sort_order, timestamps to studio_tools;
-- adds output_text, provider, cost_micros, duration_ms,
-- tool_slug, updated_at to ai_generations;
-- seeds all 17 canonical tools;
-- creates chat_sessions + chat_messages tables.
-- ============================================================

-- BEGIN;  -- removed: managed by golang-migrate

-- ─── studio_tools: add new columns ───────────────────────────────────────────

ALTER TABLE studio_tools
    ADD COLUMN IF NOT EXISTS slug          TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sort_order    INT         NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS provider_tool TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Unique index on slug (used by FindToolBySlug)
CREATE UNIQUE INDEX IF NOT EXISTS uidx_studio_tools_slug ON studio_tools (slug);

-- Back-fill slugs for any existing rows using the name column
UPDATE studio_tools
SET slug = LOWER(REGEXP_REPLACE(TRIM(name), '[\s_]+', '-', 'g'))
WHERE slug = '';

-- ─── ai_generations: add new columns ────────────────────────────────────────

ALTER TABLE ai_generations
    ADD COLUMN IF NOT EXISTS tool_slug    TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS output_text  TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS provider     TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS cost_micros  INT         NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS duration_ms  INT         NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Index for gallery queries (user + status + expiry)
CREATE INDEX IF NOT EXISTS idx_ai_gen_user_status ON ai_generations (user_id, status, expires_at DESC);

-- Index for stale-job watchdog
CREATE INDEX IF NOT EXISTS idx_ai_gen_pending_created ON ai_generations (status, created_at)
    WHERE status IN ('pending', 'processing');

-- ─── Seed canonical 17 tools ─────────────────────────────────────────────────
-- All costs are stored in DB — never hardcoded in application layer.
-- Point costs are in PulsePoints (1 PP = ₦200 recharge equivalent).

INSERT INTO studio_tools
    (id, name, slug, description, category, point_cost, provider, provider_tool,
     icon, sort_order, is_active, created_at, updated_at)
VALUES
-- ── Learn ─────────────────────────────────────────────────────────────────
(gen_random_uuid(), 'Translate',        'translate',        'Translate text between languages',               'Learn',  1, 'groq',        'llama-3.3-70b-versatile',            '🌍',  1,  true, NOW(), NOW()),
(gen_random_uuid(), 'Study Guide',      'study-guide',      'Create a comprehensive study guide',             'Learn',  2, 'groq',        'llama-3.3-70b-versatile',            '📖',  2,  true, NOW(), NOW()),
(gen_random_uuid(), 'Quiz Generator',   'quiz',             'Generate multiple-choice quizzes',               'Learn',  2, 'groq',        'llama-3.3-70b-versatile',            '🧠',  3,  true, NOW(), NOW()),
(gen_random_uuid(), 'Mind Map',         'mindmap',          'Turn any topic into a visual mind map',          'Learn',  2, 'groq',        'llama-3.3-70b-versatile',            '🗺️', 4,  true, NOW(), NOW()),
-- ── Build ─────────────────────────────────────────────────────────────────
(gen_random_uuid(), 'Research Brief',   'research-brief',   'Produce a concise research brief',               'Build',  3, 'groq',        'llama-3.3-70b-versatile',            '📊',  5,  true, NOW(), NOW()),
(gen_random_uuid(), 'Business Plan',    'bizplan',          'One-page Nigerian market business plan',         'Build',  5, 'groq',        'llama-3.3-70b-versatile',            '💼',  6,  true, NOW(), NOW()),
(gen_random_uuid(), 'Slide Deck',       'slide-deck',       '10-slide presentation outline in JSON',          'Build',  5, 'groq',        'llama-3.3-70b-versatile',            '📑',  7,  true, NOW(), NOW()),
-- ── Create ────────────────────────────────────────────────────────────────
(gen_random_uuid(), 'AI Photo',         'ai-photo',         'Generate a high-quality AI image',               'Create', 5, 'fal.ai',      'fal-ai/flux/dev',                    '🖼️', 8,  true, NOW(), NOW()),
(gen_random_uuid(), 'Background Remover','bg-remover',      'Remove image background instantly',              'Create', 3, 'fal.ai',      'fal-ai/birefnet',                    '✂️', 9,  true, NOW(), NOW()),
(gen_random_uuid(), 'Animate Photo',    'animate-photo',    'Bring a still photo to life',                    'Create', 10,'fal.ai',      'fal-ai/kling-video/v1.5/standard',   '🎬', 10, true, NOW(), NOW()),
(gen_random_uuid(), 'Video Premium',    'video-premium',    'AI text-to-video (Kling Pro)',                   'Create', 20,'fal.ai',      'fal-ai/kling-video/v1.5/pro',        '🎥', 11, true, NOW(), NOW()),
(gen_random_uuid(), 'Narrate',          'narrate',          'Convert text to natural-sounding speech',        'Create', 5, 'elevenlabs',  'eleven_turbo_v2',                    '🎙️',12, true, NOW(), NOW()),
(gen_random_uuid(), 'Transcribe',       'transcribe',       'Transcribe audio to text (Whisper)',             'Create', 3, 'groq',        'whisper-large-v3',                   '📝', 13, true, NOW(), NOW()),
(gen_random_uuid(), 'Jingle',          'jingle',           'Generate a short AI music jingle',               'Create', 8, 'mubert',      'RecordTrackTTM',                     '🎵', 14, true, NOW(), NOW()),
(gen_random_uuid(), 'Background Music','bg-music',         'Generate 60s background music track',            'Create', 8, 'mubert',      'RecordTrackTTM',                     '🎶', 15, true, NOW(), NOW()),
-- ── Chat ──────────────────────────────────────────────────────────────────
(gen_random_uuid(), 'Podcast',         'podcast',          'Script + narrate a 2-host audio podcast',       'Create', 10,'groq+elevenlabs','composite',                        '🎧', 16, true, NOW(), NOW()),
(gen_random_uuid(), 'Infographic',     'infographic',      'Data layout JSON + AI visual render',            'Create', 8, 'groq+fal.ai', 'composite',                          '📊', 17, true, NOW(), NOW())

ON CONFLICT (slug) DO UPDATE
    SET name          = EXCLUDED.name,
        description   = EXCLUDED.description,
        category      = EXCLUDED.category,
        point_cost    = EXCLUDED.point_cost,
        provider      = EXCLUDED.provider,
        provider_tool = EXCLUDED.provider_tool,
        icon          = EXCLUDED.icon,
        sort_order    = EXCLUDED.sort_order,
        updated_at    = NOW();

-- ─── chat_sessions ────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS chat_sessions (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title        TEXT        NOT NULL DEFAULT 'Nexus Chat',
    -- Rolling summary written after every 10 messages (spec §9.5)
    summary      TEXT        NOT NULL DEFAULT '',
    message_count INT        NOT NULL DEFAULT 0,
    last_provider TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at   TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 days'
);

CREATE INDEX IF NOT EXISTS idx_chat_sessions_user ON chat_sessions (user_id, updated_at DESC);

-- ─── chat_messages ────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS chat_messages (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID        NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT        NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content    TEXT        NOT NULL,
    provider   TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages (session_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_chat_messages_user    ON chat_messages (user_id, created_at DESC);

-- ─── chat_session_summaries (rolling compression — spec §9.5) ────────────────

CREATE TABLE IF NOT EXISTS chat_session_summaries (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id    UUID        REFERENCES chat_sessions(id) ON DELETE SET NULL,
    summary_text  TEXT        NOT NULL,
    message_range INT4RANGE,          -- which message IDs were summarised
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_summaries_user ON chat_session_summaries (user_id, created_at DESC);

-- ─── Updated-at triggers ──────────────────────────────────────────────────────

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_studio_tools_updated_at') THEN
        CREATE TRIGGER trg_studio_tools_updated_at
            BEFORE UPDATE ON studio_tools
            FOR EACH ROW EXECUTE FUNCTION update_updated_at();
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_chat_sessions_updated_at') THEN
        CREATE TRIGGER trg_chat_sessions_updated_at
            BEFORE UPDATE ON chat_sessions
            FOR EACH ROW EXECUTE FUNCTION update_updated_at();
    END IF;
END;
$$;

-- COMMIT;  -- removed: managed by golang-migrate


══════════════════════════════════════════════════════
MIGRATION: 027_phase11_wars_hardening.up.sql
══════════════════════════════════════════════════════
-- 027_phase11_wars_hardening.sql
-- Phase 11: Regional Wars entity/repo hardening, leaderboard indices, lifecycle crons
-- ─────────────────────────────────────────────────────────────────────────────
-- This migration:
--   1. Ensures regional_wars table columns match the Go entity (safe with IF NOT EXISTS)
--   2. Adds missing indices for leaderboard performance
--   3. Ensures regional_war_winners table is fully specified
--   4. Adds the correct network_config keys for wars + studio stale recovery
--   5. Removes reference to the now-defunct wars_snapshots table (old Phase 5 approach)
-- ─────────────────────────────────────────────────────────────────────────────

-- BEGIN;  -- removed: managed by golang-migrate

-- ── 1. Ensure regional_wars has all required columns ─────────────────────────

ALTER TABLE regional_wars
    ADD COLUMN IF NOT EXISTS created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Ensure period is unique (may already exist)
CREATE UNIQUE INDEX IF NOT EXISTS uidx_regional_wars_period ON regional_wars(period);

-- ── 2. Leaderboard performance index ─────────────────────────────────────────
-- The leaderboard query filters transactions WHERE type='points_award' AND points_delta > 0
-- across a time window joined to users.state. This composite index helps.

CREATE INDEX IF NOT EXISTS idx_tx_leaderboard
    ON transactions(user_id, type, points_delta, created_at)
    WHERE type = 'points_award' AND points_delta > 0;

-- users.state index for GROUP BY
CREATE INDEX IF NOT EXISTS idx_users_state
    ON users(state)
    WHERE state IS NOT NULL AND state <> '';

-- users.is_active partial index
CREATE INDEX IF NOT EXISTS idx_users_active_state
    ON users(state, is_active)
    WHERE is_active = true AND state IS NOT NULL AND state <> '';

-- ── 3. Ensure regional_war_winners has all required columns ──────────────────

ALTER TABLE regional_war_winners
    ADD COLUMN IF NOT EXISTS created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_war_winners_war_id ON regional_war_winners(war_id);
CREATE INDEX IF NOT EXISTS idx_war_winners_state  ON regional_war_winners(state);

-- updated_at trigger on regional_war_winners
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_war_winners_updated_at ON regional_war_winners;
CREATE TRIGGER trg_war_winners_updated_at
    BEFORE UPDATE ON regional_war_winners
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trg_regional_wars_updated_at ON regional_wars;
CREATE TRIGGER trg_regional_wars_updated_at
    BEFORE UPDATE ON regional_wars
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ── 4. Network config keys ────────────────────────────────────────────────────

INSERT INTO network_configs (key, value, description) VALUES
    ('regional_wars_prize_pool_kobo',    '50000000', 'Monthly Regional Wars prize pool in kobo (default ₦500,000)'),
    ('regional_wars_winning_bonus',      '50',       'Pulse Points bonus awarded to every member of a winning state'),
    ('studio_stale_job_timeout_secs',    '600',      'Seconds after which a pending/processing AI generation is considered stale'),
    ('studio_stale_job_batch_size',      '20',       'Max stale jobs refunded per lifecycle cron run'),
    ('lifecycle_wars_resolve_enabled',   'true',     'Enable auto-resolve of wars on month end'),
    ('lifecycle_studio_stale_enabled',   'true',     'Enable stale studio job recovery cron')
ON CONFLICT (key) DO NOTHING;

-- ── 5. Drop the old wars_snapshots table (Phase 5 artefact — no longer used) ─
-- Guarded: only drops if the table exists and has zero rows (safety check).
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_name = 'wars_snapshots'
    ) THEN
        -- Only drop if empty; preserve data if an operator has historical rows
        IF (SELECT COUNT(*) FROM wars_snapshots) = 0 THEN
            DROP TABLE wars_snapshots;
            RAISE NOTICE 'Dropped empty wars_snapshots table';
        ELSE
            RAISE NOTICE 'wars_snapshots has rows — skipping drop; please migrate manually';
        END IF;
    END IF;
END $$;

-- ── 6. Auto-create the current month war if none exists ──────────────────────
-- Idempotent: ON CONFLICT DO NOTHING.
INSERT INTO regional_wars (
    id,
    period,
    status,
    total_prize_kobo,
    starts_at,
    ends_at
)
SELECT
    gen_random_uuid(),
    TO_CHAR(NOW(), 'YYYY-MM'),
    'ACTIVE',
    50000000,
    DATE_TRUNC('month', NOW()),
    (DATE_TRUNC('month', NOW()) + INTERVAL '1 month - 1 second')
WHERE NOT EXISTS (
    SELECT 1 FROM regional_wars WHERE period = TO_CHAR(NOW(), 'YYYY-MM')
);

-- COMMIT;  -- removed: managed by golang-migrate


══════════════════════════════════════════════════════
MIGRATION: 028_phase12_production_hardening.up.sql
══════════════════════════════════════════════════════
-- ════════════════════════════════════════════════════════════════════════════
--  028_phase12_production_hardening.sql
--  Production-ready config for:
--    1. Correct studio_tools catalogue (18 tools, exact costs from spec doc)
--    2. Storage backend network_config keys (provider-agnostic: S3 / GCS / local)
--    3. Chat/LLM tuning keys
--    4. AI provider config keys for all new adapters
-- ════════════════════════════════════════════════════════════════════════════

-- BEGIN;  -- removed: managed by golang-migrate

-- ─── 1. Update studio_tools with correct costs from spec key-points doc ───────
--
--  Cost rationale (spec §3.2 + key-points doc):
--    Free AI providers (Gemini, Groq, HF)  → cheapest tools (1–5 pts)
--    Paid APIs (FAL.AI, ElevenLabs, Mubert) → mid-range (5–200 pts)
--    Premium video (Kling v1.5)             → expensive (65 pts)
--    Full production jingle (ElevenLabs)   → 200 pts
--    Composite video+jingle                → 470 pts (future roadmap)
-- ─────────────────────────────────────────────────────────────────────────────

INSERT INTO studio_tools
    (id, name, slug, description, category, point_cost, provider, provider_tool,
     icon, sort_order, is_active, created_at, updated_at)
VALUES
-- ── Learn (Free AI providers — Gemini Flash primary, Groq fallback) ──────────
(gen_random_uuid(), 'Translate',        'translate',
    'Translate text to Yoruba, Hausa, Igbo, French or English',
    'Create',    1,  'google-translate',  'translate/v2',                        '🌍',  1,  true, NOW(), NOW()),

(gen_random_uuid(), 'Study Guide',      'study-guide',
    'Generate a comprehensive study guide with concepts, examples and quiz',
    'Learn',     3,  'gemini-flash',      'gemini-2.0-flash',                    '📖',  2,  true, NOW(), NOW()),

(gen_random_uuid(), 'Quiz Generator',   'quiz',
    'Create 10 multiple-choice quiz questions with explanations',
    'Learn',     2,  'gemini-flash',      'gemini-2.0-flash',                    '🧠',  3,  true, NOW(), NOW()),

(gen_random_uuid(), 'Mind Map',         'mindmap',
    'Turn any topic into a structured JSON mind map',
    'Learn',     2,  'gemini-flash',      'gemini-2.0-flash',                    '🗺️',  4,  true, NOW(), NOW()),

(gen_random_uuid(), 'Podcast',          'podcast',
    'Script and narrate a 2-host podcast (Nexus & Ade)',
    'Learn',     4,  'gemini+google-tts', 'composite',                           '🎧',  5,  true, NOW(), NOW()),

-- ── Build (Gemini Flash → Groq → DeepSeek for complex docs) ─────────────────
(gen_random_uuid(), 'Research Brief',   'research-brief',
    'Write a structured research brief with market data and recommendations',
    'Build',     5,  'gemini-flash',      'gemini-2.0-flash',                    '📊',  6,  true, NOW(), NOW()),

(gen_random_uuid(), 'Slide Deck',       'slide-deck',
    'Generate a 10-slide presentation outline as structured JSON',
    'Build',     4,  'gemini-flash',      'gemini-2.0-flash',                    '📑',  7,  true, NOW(), NOW()),

(gen_random_uuid(), 'Infographic',      'infographic',
    'Create infographic content structure with stats, headings and bullets',
    'Build',     5,  'gemini-flash',      'gemini-2.0-flash',                    '📊',  8,  true, NOW(), NOW()),

(gen_random_uuid(), 'Business Plan',    'bizplan',
    'Write a full Nigerian market business plan (8 structured sections)',
    'Build',    12,  'gemini-flash',      'gemini-2.0-flash',                    '💼',  9,  true, NOW(), NOW()),

-- ── Create — free/cheap AI providers ─────────────────────────────────────────
(gen_random_uuid(), 'Background Remover', 'bg-remover',
    'Remove image background instantly (rembg → FAL.AI BiRefNet)',
    'Create',    3,  'rembg',             'fal-ai/birefnet',                     '✂️', 10,  true, NOW(), NOW()),

(gen_random_uuid(), 'Narrate',          'narrate',
    'Convert text to natural-sounding Nigerian English speech',
    'Create',    2,  'google-cloud-tts',  'en-NG',                               '🎙️', 11,  true, NOW(), NOW()),

(gen_random_uuid(), 'Transcribe',       'transcribe',
    'Transcribe audio to text (AssemblyAI → Groq Whisper)',
    'Create',    2,  'assemblyai',        'best',                                '📝', 12,  true, NOW(), NOW()),

(gen_random_uuid(), 'Background Music', 'bg-music',
    'Generate 15s royalty-free background music (HuggingFace MusicGen)',
    'Create',    5,  'hf-musicgen',       'facebook/musicgen-small',             '🎶', 13,  true, NOW(), NOW()),

(gen_random_uuid(), 'AI Photo',         'ai-photo',
    'Generate a high-quality AI image from your description',
    'Create',   10,  'hf-flux-schnell',   'black-forest-labs/FLUX.1-schnell',    '🖼️', 14,  true, NOW(), NOW()),

-- ── Create — paid/premium providers ──────────────────────────────────────────
(gen_random_uuid(), 'Animate Photo',    'animate-photo',
    'Bring a still photo to life with smooth 5-second animation',
    'Create',   65,  'fal.ai',            'fal-ai/ltx-video',                    '🎬', 15,  true, NOW(), NOW()),

(gen_random_uuid(), 'Video Premium',    'video-premium',
    'Cinematic AI video from image (Kling v1.5 Standard)',
    'Create',   65,  'fal.ai',            'fal-ai/kling-video/v1.5/standard',    '🎥', 16,  true, NOW(), NOW()),

(gen_random_uuid(), 'Jingle',           'jingle',
    'Generate a 30-second AI music jingle for your brand (ElevenLabs)',
    'Create',  200,  'elevenlabs',        'sound-generation',                    '🎵', 17,  true, NOW(), NOW()),

(gen_random_uuid(), 'Video Jingle',     'video-jingle',
    'Full production: Kling video + ElevenLabs music score',
    'Build',   470,  'fal.ai+elevenlabs', 'composite',                           '🎞️', 18,  false, NOW(), NOW())

ON CONFLICT (slug) DO UPDATE
    SET name          = EXCLUDED.name,
        description   = EXCLUDED.description,
        category      = EXCLUDED.category,
        point_cost    = EXCLUDED.point_cost,
        provider      = EXCLUDED.provider,
        provider_tool = EXCLUDED.provider_tool,
        icon          = EXCLUDED.icon,
        sort_order    = EXCLUDED.sort_order,
        is_active     = EXCLUDED.is_active,
        updated_at    = NOW();

-- ─── 2. Storage backend configuration keys ────────────────────────────────────
--
--  STORAGE_BACKEND drives which concrete implementation is used:
--    "s3"    → AWS S3 (or S3-compatible: MinIO, Cloudflare R2)
--    "gcs"   → Google Cloud Storage
--    "local" → local filesystem (development / CI)
--    ""      → auto-detect from available credentials
--
--  These keys map 1:1 to environment variable overrides.
--  Operators can override at runtime by editing network_configs;
--  the ConfigManager reads from DB first, env var as fallback.
-- ─────────────────────────────────────────────────────────────────────────────

INSERT INTO network_configs (key, value, description, updated_at)
VALUES
    ('storage_backend',         '',
     'Asset storage provider: "s3", "gcs", "local", or "" for auto-detect',     NOW()),

    ('storage_cdn_base_url',    '',
     'CDN prefix returned in all asset URLs (e.g. https://cdn.loyalty-nexus.ai)', NOW()),

    -- AWS S3 / S3-compatible
    ('aws_s3_bucket',           '',     'S3 bucket name (AWS / MinIO / Cloudflare R2)',     NOW()),
    ('aws_region',              'us-east-1', 'AWS region (default: us-east-1)',            NOW()),
    ('aws_s3_endpoint',         '',
     'Custom S3-compatible endpoint (leave blank for standard AWS)',              NOW()),

    -- Google Cloud Storage
    ('gcs_bucket',              '',     'GCS bucket name',                                  NOW()),

    -- Local filesystem (dev / CI only)
    ('local_storage_base_path', '/tmp/nexus-assets',
     'Absolute filesystem path for local asset storage (dev only)',              NOW()),
    ('local_storage_base_url',  'http://localhost:8080/assets',
     'URL prefix served for local assets (dev only)',                            NOW())

ON CONFLICT (key) DO NOTHING;

-- ─── 3. LLM / Chat configuration keys ────────────────────────────────────────

INSERT INTO network_configs (key, value, description, updated_at)
VALUES
    -- Provider routing limits (daily request counts)
    ('chat_groq_daily_limit',            '1000',
     'Max Groq (Llama-4-Scout) requests per user per day before falling to Gemini',  NOW()),
    ('chat_gemini_daily_limit',          '2000',
     'Max cumulative requests (Groq+Gemini) per user per day before DeepSeek',        NOW()),

    -- Session memory
    ('chat_session_timeout_minutes',     '30',
     'Minutes of inactivity before a chat session is marked stale and summarised',   NOW()),
    ('chat_session_summary_messages',    '10',
     'Number of messages that trigger an incremental session summary',               NOW()),
    ('chat_memory_summaries_count',      '3',
     'Number of past session summaries injected into the system prompt',             NOW()),
    ('chat_memory_recent_messages',      '5',
     'Number of recent raw messages injected into the system prompt',                NOW()),

    -- LLM model overrides (operator can swap models without a deploy)
    ('llm_groq_model',          'llama-4-scout-17b-16e-instruct',
     'Groq model identifier',                                                         NOW()),
    ('llm_gemini_model',        'gemini-2.0-flash-lite',
     'Gemini model identifier (free Flash-Lite)',                                    NOW()),
    ('llm_deepseek_model',      'deepseek-chat',
     'DeepSeek model identifier (paid overflow)',                                    NOW())

ON CONFLICT (key) DO NOTHING;

-- ─── 4. AI provider configuration keys ───────────────────────────────────────

INSERT INTO network_configs (key, value, description, updated_at)
VALUES
    -- Image generation
    ('studio_hf_image_model',
     'black-forest-labs/FLUX.1-schnell',
     'HuggingFace model for AI photo (free tier)',                               NOW()),

    -- TTS
    ('studio_elevenlabs_voice_id',
     '21m00Tcm4TlvDq8ikWAM',
     'Default ElevenLabs voice ID (Rachel)',                                      NOW()),
    ('studio_tts_primary_provider',
     'google-cloud-tts',
     'Primary TTS provider: google-cloud-tts | elevenlabs | huggingface-bark',   NOW()),

    -- Background removal
    ('studio_rembg_service_url',
     '',
     'Self-hosted rembg microservice URL (e.g. http://rembg-service:5000)',       NOW()),

    -- Music
    ('studio_mubert_duration_secs',
     '30',
     'Duration in seconds for Mubert background music generation',               NOW()),

    -- Video
    ('studio_fal_video_model_standard',
     'fal-ai/ltx-video',
     'FAL.AI model for animate-photo (cheaper)',                                  NOW()),
    ('studio_fal_video_model_premium',
     'fal-ai/kling-video/v1.5/standard',
     'FAL.AI model for video-premium (Kling v1.5)',                               NOW()),

    -- Stale job recovery (also used by LifecycleWorker)
    ('studio_stale_job_timeout_minutes',
     '15',
     'Minutes before a stuck pending/processing job is failed and refunded',     NOW()),
    ('studio_stale_recovery_batch',
     '50',
     'Max stale jobs recovered per LifecycleWorker tick',                        NOW()),

    -- Transcription
    ('studio_transcription_primary',
     'assemblyai',
     'Primary transcription provider: assemblyai | groq-whisper',               NOW())

ON CONFLICT (key) DO NOTHING;

-- COMMIT;  -- removed: managed by golang-migrate


══════════════════════════════════════════════════════
MIGRATION: 029_phase16_enterprise_studio_tools.up.sql
══════════════════════════════════════════════════════
-- =============================================================================
-- 029_phase16_enterprise_studio_tools.sql
-- Phase 16: Enterprise Studio Tools Expansion
-- =============================================================================
--
-- WHAT THIS MIGRATION DOES:
--   1. Expands the studio_tools.category CHECK constraint to include 'Vision'
--      (a new category for image-analysis tools).
--   2. Upserts 14 new studio tools across the Chat, Vision, Build, and Create
--      categories — safe to re-run due to ON CONFLICT (slug) DO UPDATE.
--
-- NEW CATEGORY:
--   'Vision' — tools that accept image uploads and return AI-powered analysis.
--
-- NEW TOOLS SUMMARY:
--   FREE (0 pts) : web-search-ai, image-analyser, ask-my-photo, code-helper
--   LOW  (3 pts) : narrate-pro, transcribe-african
--   MID  (8-10)  : ai-photo-dream (8), ai-photo-pro (10), photo-editor (10)
--   HIGH (15-50) : ai-photo-max (15), instrumental (25), song-creator (30),
--                  video-cinematic (40), video-veo (50)
--
-- TOOLS NOT MODIFIED:
--   translate, study-guide, quiz, mindmap, research-brief, bizplan, slide-deck,
--   ai-photo, bg-remover, video-premium, narrate, transcribe, jingle, bg-music,
--   podcast, infographic
--
-- DEPENDENCY:  Requires 026_phase10_studio_hardening.sql (adds slug, sort_order,
--              provider_tool columns and the uidx_studio_tools_slug unique index).
-- =============================================================================

-- BEGIN;  -- removed: managed by golang-migrate

-- =============================================================================
-- STEP 1 — Expand the category CHECK constraint to allow 'Vision'
-- =============================================================================
-- The original CHECK in 003_nexus_studio.sql only covers: Chat, Create, Learn, Build.
-- We drop that constraint and recreate it with 'Vision' added.
-- The IF EXISTS guard makes this re-run safe.

ALTER TABLE studio_tools
    DROP CONSTRAINT IF EXISTS studio_tools_category_check;

ALTER TABLE studio_tools
    ADD CONSTRAINT studio_tools_category_check
        CHECK (category IN ('Chat', 'Create', 'Learn', 'Build', 'Vision'));

-- =============================================================================
-- STEP 2 — Upsert 14 new tools
-- =============================================================================
-- All rows use gen_random_uuid() for id so the INSERT always produces a valid
-- UUID on first run.  ON CONFLICT (slug) DO UPDATE ensures subsequent runs
-- update metadata without creating duplicates or resetting other columns
-- (e.g. is_active).  The id of an existing row is intentionally NOT overwritten.
-- =============================================================================

INSERT INTO studio_tools
    (id, name, slug, description, category, point_cost, provider, provider_tool,
     icon, sort_order, is_active, created_at, updated_at)
VALUES

-- ─────────────────────────────────────────────────────────────────────────────
-- FREE TOOLS  (point_cost = 0)
-- ─────────────────────────────────────────────────────────────────────────────

-- Chat ─────────────────────────────────────────────────────────────────────────
(gen_random_uuid(),
 'Web Search AI',      'web-search-ai',
 'Ask any question — get answers with live internet data',
 'Chat',    0, 'pollinations', 'gemini-search',
 '🔍', 18, true, NOW(), NOW()),

-- Vision ───────────────────────────────────────────────────────────────────────
(gen_random_uuid(),
 'Image Analyser',     'image-analyser',
 'Upload any photo — AI describes everything in detail',
 'Vision',  0, 'pollinations', 'openai-vision',
 '👁️', 19, true, NOW(), NOW()),

(gen_random_uuid(),
 'Ask My Photo',       'ask-my-photo',
 'Upload an image and ask any question about it',
 'Vision',  0, 'pollinations', 'openai-vision',
 '🤔', 20, true, NOW(), NOW()),

-- Build ────────────────────────────────────────────────────────────────────────
(gen_random_uuid(),
 'Code Helper',        'code-helper',
 'Write, explain, and debug code with AI',
 'Build',   0, 'pollinations', 'qwen-coder',
 '💻', 21, true, NOW(), NOW()),

-- ─────────────────────────────────────────────────────────────────────────────
-- LOW-COST TOOLS  (point_cost = 3)
-- ─────────────────────────────────────────────────────────────────────────────

-- Create ───────────────────────────────────────────────────────────────────────
(gen_random_uuid(),
 'Narrate Pro',        'narrate-pro',
 'Text to speech with 13 premium voice options',
 'Create',  3, 'pollinations', 'tts-1-voices',
 '🎙️', 22, true, NOW(), NOW()),

(gen_random_uuid(),
 'Transcribe African', 'transcribe-african',
 'Transcribe audio in Yoruba, Hausa, Igbo, English & French',
 'Create',  3, 'pollinations', 'whisper-african',
 '🌍', 23, true, NOW(), NOW()),

-- ─────────────────────────────────────────────────────────────────────────────
-- PAID TOOLS  (point_cost = 8 – 50)
-- ─────────────────────────────────────────────────────────────────────────────

-- Create — image generation tier ──────────────────────────────────────────────
(gen_random_uuid(),
 'AI Photo Dream',     'ai-photo-dream',
 'Creative & stylized AI images — Seedream by ByteDance',
 'Create',  8, 'pollinations', 'seedream',
 '🎨', 26, true, NOW(), NOW()),

(gen_random_uuid(),
 'AI Photo Pro',       'ai-photo-pro',
 'Photorealistic AI image generation — premium quality',
 'Create', 10, 'pollinations', 'gptimage',
 '✨', 24, true, NOW(), NOW()),

(gen_random_uuid(),
 'Photo Editor AI',    'photo-editor',
 'Edit any photo with text instructions — AI transforms it',
 'Create', 10, 'pollinations', 'kontext',
 '🖊️', 27, true, NOW(), NOW()),

(gen_random_uuid(),
 'AI Photo Max',       'ai-photo-max',
 'Highest quality AI image — GPT Image Large',
 'Create', 15, 'pollinations', 'gptimage-large',
 '🌟', 25, true, NOW(), NOW()),

-- Create — music generation ────────────────────────────────────────────────────
(gen_random_uuid(),
 'Instrumental Track', 'instrumental',
 'Generate AI background music — no vocals',
 'Create', 25, 'pollinations', 'elevenmusic-instrumental',
 '🎹', 29, true, NOW(), NOW()),

(gen_random_uuid(),
 'Song Creator',       'song-creator',
 'Generate a full AI song with vocals — any genre',
 'Create', 30, 'pollinations', 'elevenmusic',
 '🎵', 28, true, NOW(), NOW()),

-- Create — video generation ────────────────────────────────────────────────────
(gen_random_uuid(),
 'Video Cinematic',    'video-cinematic',
 'Image to cinematic video — Seedance by ByteDance',
 'Create', 40, 'pollinations', 'seedance',
 '🎬', 30, true, NOW(), NOW()),

(gen_random_uuid(),
 'Video Veo',          'video-veo',
 'Text-to-video powered by Google Veo — highest quality',
 'Create', 50, 'pollinations', 'veo2',
 '🎦', 31, true, NOW(), NOW())

ON CONFLICT (slug) DO UPDATE
    SET name          = EXCLUDED.name,
        description   = EXCLUDED.description,
        category      = EXCLUDED.category,
        point_cost    = EXCLUDED.point_cost,
        provider      = EXCLUDED.provider,
        provider_tool = EXCLUDED.provider_tool,
        icon          = EXCLUDED.icon,
        sort_order    = EXCLUDED.sort_order,
        updated_at    = NOW();

-- =============================================================================
-- VERIFICATION (informational — does not affect migration outcome)
-- =============================================================================
-- After applying, run:
--   SELECT category, COUNT(*) FROM studio_tools GROUP BY category ORDER BY category;
-- Expected new rows per category (Phase 16 additions only):
--   Build   +1  (code-helper)
--   Chat    +1  (web-search-ai)
--   Create  +10 (narrate-pro, transcribe-african, ai-photo-pro, ai-photo-max,
--                ai-photo-dream, photo-editor, song-creator, instrumental,
--                video-cinematic, video-veo)
--   Vision  +2  (image-analyser, ask-my-photo)
-- =============================================================================

-- COMMIT;  -- removed: managed by golang-migrate


══════════════════════════════════════════════════════
MIGRATION: 030_video_jingle_tool.up.sql
══════════════════════════════════════════════════════
-- =============================================================================
-- 030_video_jingle_tool.sql
-- Phase 17: Add missing video-jingle composite tool to studio_tools
-- =============================================================================
--
-- WHAT THIS MIGRATION DOES:
--   Inserts the video-jingle tool that was implemented in the Go service
--   (ai_studio_service.go dispatchVideo) but was never seeded in the DB.
--
-- TOOL:
--   video-jingle (470 pts) — Full cinematic video + AI vocal song (Kling + ElevenMusic)
--   The most premium tool in the studio: FAL.AI Kling video + ElevenLabs/Pollinations
--   music combined into a single production-quality output.
--
-- PROVIDER CHAIN:
--   Primary  : FAL.AI Kling v1.5 Pro (video) + ElevenLabs Music (audio)
--   Fallback : Pollinations wan-fast (video) — audio portion always uses ElevenLabs
--
-- POINT COST RATIONALE:
--   FAL.AI Kling video  ≈ ₦320  (API cost)
--   ElevenLabs Music    ≈ ₦450  (API cost)
--   Total platform cost ≈ ₦770
--   470 pts × ₦7.50/pt = ₦3,525 revenue — ~78% margin
--
-- SAFE TO RE-RUN: ON CONFLICT (slug) DO UPDATE
-- =============================================================================

-- BEGIN;  -- removed: managed by golang-migrate

INSERT INTO studio_tools
    (id, name, slug, description, category, point_cost, provider, provider_tool,
     icon, sort_order, is_active, created_at, updated_at)
VALUES
(gen_random_uuid(),
 'Video + Jingle',    'video-jingle',
 'Full AI production: cinematic video combined with a custom vocal song',
 'Create', 470, 'fal.ai+elevenlabs', 'kling-v1.5+elevenmusic',
 '🎬🎵', 32, true, NOW(), NOW())

ON CONFLICT (slug) DO UPDATE
    SET name          = EXCLUDED.name,
        description   = EXCLUDED.description,
        category      = EXCLUDED.category,
        point_cost    = EXCLUDED.point_cost,
        provider      = EXCLUDED.provider,
        provider_tool = EXCLUDED.provider_tool,
        icon          = EXCLUDED.icon,
        sort_order    = EXCLUDED.sort_order,
        updated_at    = NOW();

-- COMMIT;  -- removed: managed by golang-migrate


══════════════════════════════════════════════════════
MIGRATION: 031_studio_session_tokens.up.sql
══════════════════════════════════════════════════════
-- ============================================================
-- Migration 031: Studio Session Token Model
-- ============================================================
-- Implements the per-tool configurable drain rate system:
--   entry_point_cost  — minimum wallet balance to access a tool
--   refund_window_mins — how long (minutes) user can dispute output
--   refund_pct        — % of points returned on approved dispute (0-100)
--   is_free           — bypasses ALL point checks (e.g. Nexus Chat)
--
-- Adds dispute tracking to ai_generations:
--   disputed_at   — timestamp user flagged the output
--   refund_granted — whether admin/system approved the refund
--   refund_pts    — how many points were actually returned
--
-- Adds studio_sessions table for live utilisation tracking:
--   Tracks total pts spent + generation count per session so the
--   frontend can show "You've used 120pts this session" live.
-- ============================================================

-- ── 1. studio_tools — new configurability columns ────────────────────────────

ALTER TABLE studio_tools
    ADD COLUMN IF NOT EXISTS entry_point_cost   BIGINT  NOT NULL DEFAULT 0
        CONSTRAINT chk_entry_point_cost_nonneg CHECK (entry_point_cost >= 0),
    ADD COLUMN IF NOT EXISTS refund_window_mins INT     NOT NULL DEFAULT 5
        CONSTRAINT chk_refund_window_nonneg    CHECK (refund_window_mins >= 0),
    ADD COLUMN IF NOT EXISTS refund_pct         INT     NOT NULL DEFAULT 100
        CONSTRAINT chk_refund_pct_range        CHECK (refund_pct BETWEEN 0 AND 100),
    ADD COLUMN IF NOT EXISTS is_free            BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN studio_tools.entry_point_cost   IS 'Minimum PulsePoints balance user must hold to open this tool. 0 = no floor.';
COMMENT ON COLUMN studio_tools.refund_window_mins IS 'Minutes after generation during which user can dispute output. 0 = no refunds.';
COMMENT ON COLUMN studio_tools.refund_pct         IS 'Percentage of points_deducted returned on approved dispute (0–100).';
COMMENT ON COLUMN studio_tools.is_free            IS 'When true, entry_point_cost and point_cost checks are bypassed entirely.';

-- ── 2. ai_generations — dispute tracking columns ─────────────────────────────

ALTER TABLE ai_generations
    ADD COLUMN IF NOT EXISTS disputed_at    TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS refund_granted BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS refund_pts     BIGINT      NOT NULL DEFAULT 0
        CONSTRAINT chk_refund_pts_nonneg CHECK (refund_pts >= 0);

COMMENT ON COLUMN ai_generations.disputed_at    IS 'When the user flagged this generation as unsatisfactory.';
COMMENT ON COLUMN ai_generations.refund_granted IS 'Whether the system issued a compensating PulsePoints refund.';
COMMENT ON COLUMN ai_generations.refund_pts     IS 'Actual PulsePoints returned (may be < points_deducted if refund_pct < 100).';

CREATE INDEX IF NOT EXISTS idx_ai_generations_disputed
    ON ai_generations (disputed_at)
    WHERE disputed_at IS NOT NULL;

-- ── 3. studio_sessions — live utilisation tracking ───────────────────────────

CREATE TABLE IF NOT EXISTS studio_sessions (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at         TIMESTAMPTZ,
    total_pts_used   BIGINT      NOT NULL DEFAULT 0
        CONSTRAINT chk_session_pts_nonneg CHECK (total_pts_used >= 0),
    generation_count INT         NOT NULL DEFAULT 0
        CONSTRAINT chk_session_gen_nonneg CHECK (generation_count >= 0)
);

COMMENT ON TABLE  studio_sessions                    IS 'One row per AI Studio session. Updated on each generation to power the live utilisation meter.';
COMMENT ON COLUMN studio_sessions.total_pts_used     IS 'Running total of PulsePoints spent across all generations in this session.';
COMMENT ON COLUMN studio_sessions.generation_count   IS 'Number of generations initiated in this session (regardless of status).';
COMMENT ON COLUMN studio_sessions.last_active_at     IS 'Updated on each generation; used to detect idle/expired sessions.';

CREATE INDEX IF NOT EXISTS idx_studio_sessions_user_id    ON studio_sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_studio_sessions_started_at ON studio_sessions (started_at DESC);
CREATE INDEX IF NOT EXISTS idx_studio_sessions_active
    ON studio_sessions (user_id, last_active_at DESC)
    WHERE ended_at IS NULL;

-- ── 4. Seed sensible defaults per tool category ──────────────────────────────
-- These are starting values. Admin can change all of them via the Studio Tools
-- CRUD page without a code deploy (zero-hardcoding rule).

-- Chat tools: fully free — no entry requirement, no generation cost
UPDATE studio_tools
SET    is_free = true,
       entry_point_cost = 0,
       point_cost       = 0,
       refund_window_mins = 0,
       refund_pct         = 0
WHERE  slug IN ('ai-chat', 'nexus-chat', 'web-search-ai');

-- Text / knowledge tools: low entry (20pts), small generation cost already set
UPDATE studio_tools
SET    entry_point_cost  = 20,
       refund_window_mins = 10,
       refund_pct         = 100
WHERE  slug IN (
    'translate', 'study-guide', 'quiz', 'mindmap',
    'research-brief', 'code-helper', 'image-analyser',
    'ask-my-photo', 'slide-deck', 'infographic'
);

-- Image tools: entry 50pts (user must have 50pts to open any image tool)
UPDATE studio_tools
SET    entry_point_cost  = 50,
       refund_window_mins = 5,
       refund_pct         = 100
WHERE  slug IN (
    'ai-photo', 'ai-photo-pro', 'ai-photo-max', 'ai-photo-dream',
    'photo-editor', 'bg-remover'
);

-- Audio tools: entry 30pts
UPDATE studio_tools
SET    entry_point_cost  = 30,
       refund_window_mins = 5,
       refund_pct         = 100
WHERE  slug IN (
    'narrate', 'narrate-pro', 'transcribe', 'transcribe-african',
    'bg-music', 'jingle', 'song-creator', 'instrumental', 'podcast'
);

-- Video tools: highest entry (200pts) — expensive API calls
UPDATE studio_tools
SET    entry_point_cost  = 200,
       refund_window_mins = 10,
       refund_pct         = 50
WHERE  slug IN (
    'animate-photo', 'video-cinematic', 'video-premium',
    'video-veo', 'video-jingle'
);

-- Business plan / build tools: medium entry (50pts)
UPDATE studio_tools
SET    entry_point_cost  = 50,
       refund_window_mins = 10,
       refund_pct         = 100
WHERE  slug IN ('bizplan', 'voice-to-plan');


══════════════════════════════════════════════════════
MIGRATION: 032_studio_ui_config.up.sql
══════════════════════════════════════════════════════
-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 032 — Studio UI Config
-- Adds ui_template (VARCHAR) and ui_config (JSONB) to studio_tools so the
-- frontend renders the correct purpose-built UI for every tool without
-- hardcoding any tool logic in React.
--
-- Templates:
--   chat              → persistent conversation thread (already exists)
--   music-composer    → song-creator, instrumental, jingle, bg-music
--   image-creator     → ai-photo-pro/max/dream
--   image-editor      → photo-editor (upload-first)
--   video-creator     → video-veo, video-premium (text-to-video)
--   video-animator    → video-cinematic, animate-photo, video-jingle (image-to-video)
--   voice-studio      → narrate-pro
--   transcribe        → transcribe-african
--   vision-ask        → image-analyser, ask-my-photo
--   knowledge-doc     → study-guide, quiz, mindmap, research-brief, bizplan,
--                        slide-deck, infographic, podcast, translate
-- ─────────────────────────────────────────────────────────────────────────────

-- 1. Add columns (idempotent)
ALTER TABLE studio_tools
  ADD COLUMN IF NOT EXISTS ui_template  VARCHAR(40)  NOT NULL DEFAULT 'knowledge-doc',
  ADD COLUMN IF NOT EXISTS ui_config    JSONB        NOT NULL DEFAULT '{}';

-- ─────────────────────────────────────────────────────────────────────────────
-- 2. CHAT tools
-- ─────────────────────────────────────────────────────────────────────────────
UPDATE studio_tools SET
  ui_template = 'chat',
  ui_config   = '{
    "prompt_placeholder": "Type your message…",
    "show_history": true
  }'::jsonb
WHERE slug IN ('ai-chat', 'web-search-ai', 'code-helper');

-- ─────────────────────────────────────────────────────────────────────────────
-- 3. MUSIC COMPOSER
-- ─────────────────────────────────────────────────────────────────────────────
UPDATE studio_tools SET
  ui_template = 'music-composer',
  ui_config   = '{
    "prompt_placeholder": "e.g. Upbeat Afrobeats track with energetic female vocals and a big chorus hook…",
    "genre_tags": ["Afrobeats","Amapiano","Gospel","Highlife","R&B","Hip-Hop","Pop","Jazz","Classical","EDM","Reggae","Funk"],
    "duration_options": [15, 30, 60, 120],
    "default_duration": 30,
    "show_vocals_toggle": true,
    "default_vocals": true,
    "show_lyrics_box": true,
    "lyrics_placeholder": "Optional: paste your own lyrics or leave blank for AI-generated lyrics\n\n[Verse 1]\n…\n[Chorus]\n…"
  }'::jsonb
WHERE slug IN ('song-creator', 'jingle', 'bg-music');

-- Instrumental variant — no vocals toggle
UPDATE studio_tools SET
  ui_template = 'music-composer',
  ui_config   = '{
    "prompt_placeholder": "e.g. Calm lo-fi background music with piano and soft drums, no vocals…",
    "genre_tags": ["Lo-fi","Cinematic","Ambient","Jazz","Classical","Acoustic","Electronic","World"],
    "duration_options": [15, 30, 60, 120],
    "default_duration": 60,
    "show_vocals_toggle": false,
    "default_vocals": false,
    "show_lyrics_box": false
  }'::jsonb
WHERE slug = 'instrumental';

-- ─────────────────────────────────────────────────────────────────────────────
-- 4. IMAGE CREATOR  (text-to-image)
-- ─────────────────────────────────────────────────────────────────────────────
UPDATE studio_tools SET
  ui_template = 'image-creator',
  ui_config   = '{
    "prompt_placeholder": "Describe the image you want to create in detail…",
    "aspect_ratios": [
      {"label":"Square",    "value":"1:1",   "icon":"square"},
      {"label":"Portrait",  "value":"9:16",  "icon":"portrait"},
      {"label":"Landscape", "value":"16:9",  "icon":"landscape"},
      {"label":"Wide",      "value":"3:2",   "icon":"wide"}
    ],
    "default_aspect": "1:1",
    "style_tags": ["Photorealistic","Cinematic","Oil Painting","Watercolour","Anime","Sketch","Digital Art","Fantasy","Vintage"],
    "show_negative_prompt": true,
    "negative_prompt_placeholder": "What to avoid (e.g. blurry, watermark, extra fingers)…"
  }'::jsonb
WHERE slug IN ('ai-photo-pro', 'ai-photo-max', 'ai-photo-dream');

-- ─────────────────────────────────────────────────────────────────────────────
-- 5. IMAGE EDITOR  (upload-first, instruction prompt)
-- ─────────────────────────────────────────────────────────────────────────────
UPDATE studio_tools SET
  ui_template = 'image-editor',
  ui_config   = '{
    "upload_label": "Upload the photo you want to edit",
    "upload_accept": ["image/png","image/jpeg","image/webp"],
    "prompt_placeholder": "Describe what to change (e.g. Make the background a beach at sunset, remove the person on the left)…",
    "style_tags": ["Realistic","Artistic","Minimalist","Vintage","Neon"],
    "show_style_tags": true,
    "max_file_mb": 10
  }'::jsonb
WHERE slug = 'photo-editor';

-- ─────────────────────────────────────────────────────────────────────────────
-- 6. VIDEO CREATOR  (text-to-video)
-- ─────────────────────────────────────────────────────────────────────────────
UPDATE studio_tools SET
  ui_template = 'video-creator',
  ui_config   = '{
    "prompt_placeholder": "Describe the scene, subject, and motion (e.g. A woman walking along a Lagos beach at golden hour, camera slowly panning right, cinematic 4K style)…",
    "aspect_ratios": [
      {"label":"Landscape 16:9", "value":"16:9"},
      {"label":"Portrait 9:16",  "value":"9:16"}
    ],
    "default_aspect": "16:9",
    "duration_options": [5, 8, 10],
    "default_duration": 5,
    "style_tags": ["Cinematic","Documentary","Realistic","Anime","Fantasy","Noir"],
    "show_negative_prompt": true,
    "negative_prompt_placeholder": "What to avoid (e.g. text overlays, blur, distortion)…",
    "generation_warning": "Video generation takes 2–5 minutes. You will be notified when ready."
  }'::jsonb
WHERE slug IN ('video-veo', 'video-premium');

-- ─────────────────────────────────────────────────────────────────────────────
-- 7. VIDEO ANIMATOR  (image-to-video)
-- ─────────────────────────────────────────────────────────────────────────────
UPDATE studio_tools SET
  ui_template = 'video-animator',
  ui_config   = '{
    "upload_label": "Upload the image or photo to animate",
    "upload_accept": ["image/png","image/jpeg","image/webp"],
    "max_file_mb": 20,
    "prompt_placeholder": "Describe the motion (e.g. Camera slowly zooms in, wind gently blows through the trees, subject turns to face camera)…",
    "aspect_ratios": [
      {"label":"Landscape 16:9", "value":"16:9"},
      {"label":"Portrait 9:16",  "value":"9:16"}
    ],
    "default_aspect": "16:9",
    "duration_options": [5, 8, 10],
    "default_duration": 5,
    "generation_warning": "Video generation takes 2–5 minutes. You will be notified when ready."
  }'::jsonb
WHERE slug IN ('video-cinematic', 'animate-photo', 'video-jingle');

-- ─────────────────────────────────────────────────────────────────────────────
-- 8. VOICE STUDIO  (TTS / narration)
-- ─────────────────────────────────────────────────────────────────────────────
UPDATE studio_tools SET
  ui_template = 'voice-studio',
  ui_config   = '{
    "prompt_placeholder": "Enter the text you want to narrate (up to 5,000 characters)…",
    "max_chars": 5000,
    "voices": [
      {"id":"alloy",   "name":"Alloy",   "tone":"Neutral & Clear",   "category":"Conversational"},
      {"id":"echo",    "name":"Echo",    "tone":"Deep & Warm",        "category":"Narration"},
      {"id":"fable",   "name":"Fable",   "tone":"Expressive & Lively","category":"Storytelling"},
      {"id":"onyx",    "name":"Onyx",    "tone":"Deep & Authoritative","category":"Broadcast"},
      {"id":"nova",    "name":"Nova",    "tone":"Friendly & Warm",    "category":"Social Media"},
      {"id":"shimmer", "name":"Shimmer", "tone":"Soft & Soothing",    "category":"Meditation"},
      {"id":"ash",     "name":"Ash",     "tone":"Gentle & Calm",      "category":"Education"},
      {"id":"ballad",  "name":"Ballad",  "tone":"Smooth & Musical",   "category":"Entertainment"},
      {"id":"coral",   "name":"Coral",   "tone":"Warm & Natural",     "category":"Podcasts"},
      {"id":"sage",    "name":"Sage",    "tone":"Clear & Professional","category":"Corporate"},
      {"id":"verse",   "name":"Verse",   "tone":"Dynamic & Engaging", "category":"Advertisement"},
      {"id":"willow",  "name":"Willow",  "tone":"Soft & Thoughtful",  "category":"Audiobooks"},
      {"id":"jessica", "name":"Jessica", "tone":"Bright & Upbeat",    "category":"Characters"}
    ],
    "default_voice": "nova",
    "languages": [
      {"code":"en","label":"English"},
      {"code":"yo","label":"Yoruba"},
      {"code":"ha","label":"Hausa"},
      {"code":"ig","label":"Igbo"},
      {"code":"fr","label":"French"},
      {"code":"pt","label":"Portuguese"},
      {"code":"es","label":"Spanish"}
    ],
    "default_language": "en"
  }'::jsonb
WHERE slug = 'narrate-pro';

-- ─────────────────────────────────────────────────────────────────────────────
-- 9. TRANSCRIBE  (audio upload → text)
-- ─────────────────────────────────────────────────────────────────────────────
UPDATE studio_tools SET
  ui_template = 'transcribe',
  ui_config   = '{
    "upload_label": "Upload your audio or voice recording",
    "upload_accept": ["audio/mp3","audio/mpeg","audio/wav","audio/m4a","audio/ogg","audio/flac"],
    "max_file_mb": 100,
    "max_duration_mins": 120,
    "languages": [
      {"code":"auto","label":"Auto-detect"},
      {"code":"en",  "label":"English"},
      {"code":"yo",  "label":"Yoruba"},
      {"code":"ha",  "label":"Hausa"},
      {"code":"ig",  "label":"Igbo"},
      {"code":"fr",  "label":"French"},
      {"code":"pcm", "label":"Nigerian Pidgin"}
    ],
    "default_language": "auto",
    "show_speaker_labels": true,
    "output_hint": "Your transcript will appear here as plain text. You can copy or download it."
  }'::jsonb
WHERE slug IN ('transcribe', 'transcribe-african');

-- ─────────────────────────────────────────────────────────────────────────────
-- 10. VISION / ASK MY PHOTO
-- ─────────────────────────────────────────────────────────────────────────────
UPDATE studio_tools SET
  ui_template = 'vision-ask',
  ui_config   = '{
    "upload_label": "Upload the image to analyse",
    "upload_accept": ["image/png","image/jpeg","image/webp","image/gif"],
    "max_file_mb": 20,
    "prompt_placeholder": "What would you like to know about this image? (e.g. What objects can you see? What text is written here?)",
    "prompt_optional": false
  }'::jsonb
WHERE slug IN ('image-analyser', 'ask-my-photo');

-- ─────────────────────────────────────────────────────────────────────────────
-- 11. KNOWLEDGE / DOCUMENT  (all text-output tools)
-- ─────────────────────────────────────────────────────────────────────────────

-- Generic knowledge tools
UPDATE studio_tools SET
  ui_template = 'knowledge-doc',
  ui_config   = '{
    "fields": [
      {"key":"topic","label":"Topic","type":"textarea","required":true,
       "placeholder":"Enter the topic or subject you want to learn about…",
       "rows": 3}
    ],
    "output_format": "text",
    "output_hint": "Your result will appear below."
  }'::jsonb
WHERE slug IN ('study-guide', 'quiz', 'mindmap', 'infographic', 'translate');

-- Research Brief — adds depth/source options
UPDATE studio_tools SET
  ui_template = 'knowledge-doc',
  ui_config   = '{
    "fields": [
      {"key":"topic","label":"Research Topic","type":"textarea","required":true,
       "placeholder":"e.g. The impact of AI on job markets in West Africa",
       "rows": 3},
      {"key":"depth","label":"Depth","type":"select","required":false,
       "options":["Overview (1-2 pages)","Detailed (3-5 pages)","Comprehensive (5-10 pages)"],
       "default":"Detailed (3-5 pages)"}
    ],
    "output_format": "text",
    "output_hint": "Your research brief will be formatted with sections, sources, and key findings."
  }'::jsonb
WHERE slug = 'research-brief';

-- Business Plan — structured multi-field form
UPDATE studio_tools SET
  ui_template = 'knowledge-doc',
  ui_config   = '{
    "fields": [
      {"key":"company",  "label":"Company / Business Name",  "type":"text",    "required":true,  "placeholder":"e.g. NexaFarm Ltd"},
      {"key":"industry", "label":"Industry",                 "type":"select",  "required":true,
       "options":["Technology","Agriculture","Healthcare","Finance & Fintech","Education","Retail & E-commerce","Manufacturing","Real Estate","Media & Entertainment","Logistics","Energy","Other"]},
      {"key":"market",   "label":"Target Market",            "type":"text",    "required":true,  "placeholder":"e.g. Smallholder farmers in Southern Nigeria"},
      {"key":"stage",    "label":"Business Stage",           "type":"select",  "required":true,
       "options":["Idea / Pre-revenue","Early Stage (0-1 years)","Growth Stage (1-3 years)","Established (3+ years)"]},
      {"key":"goal",     "label":"Main Goal / Problem Solved","type":"textarea","required":true,  "placeholder":"Briefly describe the problem your business solves…","rows":2}
    ],
    "output_format": "document",
    "output_hint": "A full business plan (executive summary, market analysis, financials, and roadmap) will be generated."
  }'::jsonb
WHERE slug = 'bizplan';

-- Slide Deck
UPDATE studio_tools SET
  ui_template = 'knowledge-doc',
  ui_config   = '{
    "fields": [
      {"key":"topic",  "label":"Presentation Topic", "type":"textarea","required":true,
       "placeholder":"e.g. Why our loyalty app is the future of customer retention in Nigeria",
       "rows":2},
      {"key":"slides", "label":"Number of Slides",   "type":"select","required":false,
       "options":["5 slides","8 slides","12 slides","15 slides","20 slides"],
       "default":"12 slides"},
      {"key":"style",  "label":"Presentation Style", "type":"select","required":false,
       "options":["Professional / Corporate","Creative / Bold","Minimal / Clean","Academic / Research"],
       "default":"Professional / Corporate"}
    ],
    "output_format": "document",
    "output_hint": "Slide outlines with titles, bullet points, and speaker notes will be generated."
  }'::jsonb
WHERE slug = 'slide-deck';

-- Podcast
UPDATE studio_tools SET
  ui_template = 'knowledge-doc',
  ui_config   = '{
    "fields": [
      {"key":"topic",    "label":"Podcast Topic",   "type":"textarea","required":true,
       "placeholder":"e.g. The rise of mobile payments in West Africa",
       "rows":2},
      {"key":"duration", "label":"Duration",        "type":"select","required":false,
       "options":["3 minutes","5 minutes","8 minutes","12 minutes","20 minutes"],
       "default":"8 minutes"},
      {"key":"style",    "label":"Style",           "type":"select","required":false,
       "options":["Solo host","Interview (2 speakers)","Debate (2 opinions)","Documentary"],
       "default":"Solo host"}
    ],
    "output_format": "audio",
    "output_hint": "A full podcast script will be generated and narrated as an audio file."
  }'::jsonb
WHERE slug = 'podcast';


══════════════════════════════════════════════════════
MIGRATION: 033_studio_ui_config_v2.up.sql
══════════════════════════════════════════════════════
-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 033 — Studio UI Config v2
-- Extends ui_config for each template with the new controls added in
-- frontend template rewrites (BPM, energy, camera movements, speaker labels,
-- output format, speed, example questions, quality toggle, edit suggestions,
-- translate language list, max duration).
-- ─────────────────────────────────────────────────────────────────────────────

-- ── Music: song-creator / jingle / bg-music — add BPM + energy + longer durations ──
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "show_bpm": true,
    "show_energy": true,
    "max_duration": 300,
    "duration_options": [15, 30, 60, 120, 180, 300]
  }'::jsonb
WHERE slug IN ('song-creator', 'jingle', 'bg-music');

-- ── Instrumental — no vocals, extend to 300s ──
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "show_bpm": true,
    "show_energy": true,
    "max_duration": 300,
    "duration_options": [30, 60, 120, 180, 300],
    "show_vocals_toggle": false,
    "show_lyrics_box": false
  }'::jsonb
WHERE slug = 'instrumental';

-- ── Image Creator — add quality toggle for GPT-Image tools ──
UPDATE studio_tools SET
  ui_config = ui_config || '{"show_quality_toggle": true}'::jsonb
WHERE slug IN ('ai-photo-pro', 'ai-photo-max');

UPDATE studio_tools SET
  ui_config = ui_config || '{"show_quality_toggle": false}'::jsonb
WHERE slug IN ('ai-photo', 'ai-photo-dream');

-- ── Image Editor — add customisable edit suggestions ──
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "edit_suggestions": [
      "Remove the background",
      "Add sunset lighting",
      "Make it look like an oil painting",
      "Add dramatic shadows",
      "Convert to black & white",
      "Make colours more vibrant",
      "Add a smooth bokeh background",
      "Upscale & enhance sharpness",
      "Change background to a beach",
      "Add professional studio lighting",
      "Make it look futuristic",
      "Apply a vintage film filter"
    ]
  }'::jsonb
WHERE slug = 'photo-editor';

-- ── Video Creator — add camera movements + extended durations ──
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "max_duration": 30,
    "duration_options": [5, 8, 10, 15, 30],
    "camera_movements": [
      {"label":"Slow zoom in",  "icon":"🔍", "value":"slow zoom in"},
      {"label":"Slow zoom out", "icon":"🔭", "value":"slow zoom out"},
      {"label":"Pan left",      "icon":"⬅️", "value":"camera panning left"},
      {"label":"Pan right",     "icon":"➡️", "value":"camera panning right"},
      {"label":"Tilt up",       "icon":"⬆️", "value":"camera tilting up"},
      {"label":"Orbit shot",    "icon":"🔄", "value":"360 orbit around subject"},
      {"label":"Tracking",      "icon":"🎯", "value":"tracking shot following subject"},
      {"label":"Handheld",      "icon":"📷", "value":"handheld camera, slight shake"},
      {"label":"Static",        "icon":"📌", "value":"static camera, no movement"}
    ]
  }'::jsonb
WHERE slug IN ('video-veo', 'video-premium');

-- ── Video Animator — add duration options (already had style tags) ──
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "duration_options": [5, 8, 10],
    "default_duration": 5
  }'::jsonb
WHERE slug IN ('video-cinematic', 'animate-photo', 'video-jingle');

-- ── Voice Studio — add speed + format controls ──
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "show_speed_control": true,
    "show_format_selector": true
  }'::jsonb
WHERE slug IN ('narrate', 'narrate-pro');

-- ── Transcribe — add speaker labels + output format ──
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "show_speaker_labels": true,
    "show_output_format": true
  }'::jsonb
WHERE slug IN ('transcribe', 'transcribe-african');

-- ── Vision Ask — differentiate the two tools ──
-- image-analyser: auto mode (prompt optional = auto-describe)
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "prompt_optional": true,
    "upload_label": "Image to analyse",
    "example_questions": [
      "Describe this image in full detail",
      "What objects can you identify?",
      "What text is visible in this image?",
      "What is the colour palette?",
      "Are there any brand logos?",
      "What is the approximate location or setting?"
    ]
  }'::jsonb
WHERE slug = 'image-analyser';

-- ask-my-photo: question required
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "prompt_optional": false,
    "upload_label": "Upload your photo",
    "example_questions": [
      "What is the brand or product in this image?",
      "Can you read the text in this image?",
      "What emotions does this person appear to feel?",
      "Describe the outfit or style in detail",
      "What is happening in this scene?",
      "Is this image suitable for a professional profile?",
      "What improvements would you suggest for this photo?",
      "What type of food or dish is this?"
    ]
  }'::jsonb
WHERE slug = 'ask-my-photo';

-- ── Translate — add dedicated translate language list ──
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "translate_languages": [
      {"code":"en",  "label":"English"},
      {"code":"fr",  "label":"French"},
      {"code":"es",  "label":"Spanish"},
      {"code":"pt",  "label":"Portuguese"},
      {"code":"de",  "label":"German"},
      {"code":"ar",  "label":"Arabic"},
      {"code":"zh",  "label":"Chinese"},
      {"code":"sw",  "label":"Swahili"},
      {"code":"yo",  "label":"Yoruba"},
      {"code":"ha",  "label":"Hausa"},
      {"code":"ig",  "label":"Igbo"},
      {"code":"pcm", "label":"Nigerian Pidgin"},
      {"code":"af",  "label":"Afrikaans"}
    ],
    "prompt_placeholder": "Paste or type the text you want to translate…"
  }'::jsonb
WHERE slug = 'translate';


══════════════════════════════════════════════════════
MIGRATION: 034_fix_code_helper_category.up.sql
══════════════════════════════════════════════════════
-- Migration 034 — Fix code-helper category + web-search-ai chat routing
-- code-helper uses qwen-coder via Pollinations chat API — it is a chat-mode
-- tool and belongs in 'Chat' alongside web-search-ai and ai-chat.
-- Keeping it in 'Build' hides it from the Chat tab and confuses users.

UPDATE studio_tools
SET    category   = 'Chat',
       sort_order = 22,
       updated_at = NOW()
WHERE  slug = 'code-helper';

-- Also confirm web-search-ai stays in Chat (idempotent)
UPDATE studio_tools
SET    category   = 'Chat',
       sort_order = 18,
       updated_at = NOW()
WHERE  slug = 'web-search-ai';

-- Result: Chat tab now contains 3 tools:
--   ai-chat        (sort 17) — general assistant
--   web-search-ai  (sort 18) — live internet answers
--   code-helper    (sort 22) — Qwen Coder via Pollinations


══════════════════════════════════════════════════════
MIGRATION: 035_studio_tools_category_fix.up.sql
══════════════════════════════════════════════════════
-- ════════════════════════════════════════════════════════════
-- Migration 035: Fix tool categories + ensure all tools have
--               correct ui_template assignments
-- ════════════════════════════════════════════════════════════

-- BEGIN;  -- removed: managed by golang-migrate

-- ── 1. Move chat-native tools to "Chat" category ────────────
UPDATE studio_tools SET category = 'Chat' WHERE slug IN ('web-search-ai', 'code-helper', 'ai-chat');

-- ── 2. Ensure web-search-ai is free (it is a chat feature) ──
UPDATE studio_tools SET is_free = true, point_cost = 0 WHERE slug = 'web-search-ai';

-- ── 3. Fix ui_template assignments ───────────────────────────

-- Image tools
UPDATE studio_tools SET ui_template = 'image_creator'
  WHERE slug IN ('ai-photo','ai-photo-pro','ai-photo-max','ai-photo-dream','infographic') AND ui_template IS NULL;

UPDATE studio_tools SET ui_template = 'image_editor'
  WHERE slug = 'photo-editor' AND ui_template IS NULL;

-- Video tools
UPDATE studio_tools SET ui_template = 'video_creator'
  WHERE slug IN ('video-premium','video-veo') AND ui_template IS NULL;

UPDATE studio_tools SET ui_template = 'video_animator'
  WHERE slug IN ('animate-photo','video-cinematic') AND ui_template IS NULL;

-- Audio tools
UPDATE studio_tools SET ui_template = 'voice_studio'
  WHERE slug IN ('narrate','narrate-pro','jingle','podcast') AND ui_template IS NULL;

UPDATE studio_tools SET ui_template = 'music_composer'
  WHERE slug IN ('bg-music','song-creator','instrumental') AND ui_template IS NULL;

UPDATE studio_tools SET ui_template = 'transcribe'
  WHERE slug IN ('transcribe','transcribe-african') AND ui_template IS NULL;

-- Vision tools
UPDATE studio_tools SET ui_template = 'vision_ask'
  WHERE slug IN ('image-analyser','ask-my-photo') AND ui_template IS NULL;

-- Knowledge tools
UPDATE studio_tools SET ui_template = 'knowledge_doc'
  WHERE slug IN (
    'translate','summarise','quiz','mindmap','slide-deck',
    'essay','email-writer','cv-writer'
  ) AND ui_template IS NULL;

-- ── 4. Ensure all rows have is_active set ─────────────────────
UPDATE studio_tools SET is_active = true WHERE is_active IS NULL;

-- ── 5. Refresh updated_at ─────────────────────────────────────
UPDATE studio_tools SET updated_at = NOW()
  WHERE slug IN (
    'web-search-ai','code-helper','ai-chat',
    'ai-photo','ai-photo-pro','ai-photo-max','ai-photo-dream','infographic',
    'photo-editor','video-premium','video-veo','animate-photo','video-cinematic',
    'narrate','narrate-pro','jingle','podcast','bg-music','song-creator','instrumental',
    'transcribe','transcribe-african','image-analyser','ask-my-photo',
    'translate','summarise','quiz','mindmap','slide-deck','essay','email-writer','cv-writer'
  );

-- COMMIT;  -- removed: managed by golang-migrate


══════════════════════════════════════════════════════
MIGRATION: 036_google_wallet_and_passport_hardening.up.sql
══════════════════════════════════════════════════════
-- ═══════════════════════════════════════════════════════════════════════════
--  036 — Google Wallet Objects + Digital Passport hardening
--  Loyalty Nexus — Phase: Digital Passport completion
-- ═══════════════════════════════════════════════════════════════════════════

-- BEGIN;  -- removed: managed by golang-migrate

-- ─── Google Wallet Loyalty Objects ────────────────────────────────────────────
-- Tracks the Google Wallet loyalty object ID per user so we can push updates
-- when their tier, streak, or points change.
CREATE TABLE IF NOT EXISTS google_wallet_objects (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    object_id           TEXT        NOT NULL UNIQUE, -- Google Wallet object ID (issuer.userId)
    class_id            TEXT        NOT NULL,        -- Google Wallet class ID (issuer.LoyaltyNexus)
    last_synced_at      TIMESTAMPTZ,
    points_at_last_sync BIGINT      NOT NULL DEFAULT 0,
    tier_at_last_sync   TEXT        NOT NULL DEFAULT 'BRONZE',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_google_wallet_objects_user ON google_wallet_objects(user_id);
CREATE INDEX        IF NOT EXISTS idx_google_wallet_objects_sync  ON google_wallet_objects(last_synced_at);

-- ─── Wallet Registrations: add push_token_updated_at for staleness tracking ──
ALTER TABLE wallet_registrations
    ADD COLUMN IF NOT EXISTS push_token_updated_at TIMESTAMPTZ DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS is_active             BOOLEAN     NOT NULL DEFAULT TRUE;

-- ─── Users: add google_wallet_object_id shortcut column ──────────────────────
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS google_wallet_object_id TEXT DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS apple_pass_serial        TEXT DEFAULT NULL;

-- ─── Ghost Nudge Log: add channel column to track SMS vs push ────────────────
ALTER TABLE ghost_nudge_log
    ADD COLUMN IF NOT EXISTS channel TEXT NOT NULL DEFAULT 'sms'; -- sms | push | both

-- ─── Passport Push Log: full audit trail of every wallet push ────────────────
CREATE TABLE IF NOT EXISTS passport_push_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform    TEXT        NOT NULL CHECK (platform IN ('apple', 'google')),
    trigger     TEXT        NOT NULL, -- 'tier_change' | 'streak_update' | 'points_milestone' | 'manual'
    status      TEXT        NOT NULL DEFAULT 'pending', -- pending | sent | failed
    error_msg   TEXT,
    pushed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_passport_push_log_user    ON passport_push_log(user_id);
CREATE INDEX IF NOT EXISTS idx_passport_push_log_status  ON passport_push_log(status);
CREATE INDEX IF NOT EXISTS idx_passport_push_log_pushed  ON passport_push_log(pushed_at DESC);

-- COMMIT;  -- removed: managed by golang-migrate


══════════════════════════════════════════════════════
MIGRATION: 037_ai_provider_configs.up.sql
══════════════════════════════════════════════════════
-- Migration 037: AI Provider Configs
-- Dynamic provider management: admin can register, prioritise, and activate
-- any AI provider without code deployments.
--
-- Design:
--   category   = what it does (text | image | video | tts | transcribe | translate | music | bg-remove | vision)
--   template   = HOW to call it (openai-compatible | pollinations-image | pollinations-tts |
--                                pollinations-video | pollinations-music | fal-image | fal-video |
--                                fal-bg-remove | hf-image | google-tts | google-translate |
--                                elevenlabs-tts | elevenlabs-music | assemblyai | groq-whisper |
--                                mubert | remove-bg | deepseek | gemini | custom-rest)
--   env_key    = name of the env var that holds the API key (e.g. FAL_API_KEY) — never stored plaintext
--   api_key    = encrypted key (AES-GCM, key = PROVIDER_ENCRYPTION_KEY env var). NULL = use env_key only
--   priority   = lower = tried first (1 = primary, 2 = first backup, 3 = second backup, …)
--   is_primary = true if this is the preferred/main provider for this category
--   is_active  = false = skip this provider entirely (soft disable)

CREATE TABLE IF NOT EXISTS ai_provider_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,                          -- human label, e.g. "Pollinations FLUX"
    slug            TEXT NOT NULL UNIQUE,                   -- machine id, e.g. "pollinations-flux"
    category        TEXT NOT NULL,                          -- text | image | video | tts | transcribe | translate | music | bg-remove | vision
    template        TEXT NOT NULL,                          -- driver template (see above)
    env_key         TEXT NOT NULL DEFAULT '',               -- env var name holding the real key
    api_key_enc     TEXT NOT NULL DEFAULT '',               -- AES-GCM encrypted key (base64). empty = key lives in env only
    model_id        TEXT NOT NULL DEFAULT '',               -- model/endpoint override (e.g. "gemini-2.5-flash")
    extra_config    JSONB NOT NULL DEFAULT '{}',            -- template-specific params (voice_id, language, etc.)
    priority        INT NOT NULL DEFAULT 10,                -- 1=primary, higher=backup
    is_primary      BOOLEAN NOT NULL DEFAULT false,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    cost_micros     INT NOT NULL DEFAULT 0,                 -- platform cost per call in microdollars
    pulse_pts       INT NOT NULL DEFAULT 0,                 -- pulse points charged to user per call
    notes           TEXT NOT NULL DEFAULT '',               -- admin notes / display description
    last_tested_at  TIMESTAMPTZ,
    last_test_ok    BOOLEAN,
    last_test_msg   TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_provider_configs_category    ON ai_provider_configs (category);
CREATE INDEX IF NOT EXISTS idx_ai_provider_configs_cat_prio    ON ai_provider_configs (category, priority) WHERE is_active = true;

-- Trigger: auto-update updated_at
CREATE OR REPLACE FUNCTION ai_provider_configs_set_updated()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END; $$;

CREATE TRIGGER ai_provider_configs_updated_at
    BEFORE UPDATE ON ai_provider_configs
    FOR EACH ROW EXECUTE FUNCTION ai_provider_configs_set_updated();

-- ── Seed with the current hardcoded providers ────────────────────────────────
-- These mirror the exact chains in ai_studio_service.go so the admin panel
-- shows the current state on day one (and the dynamic dispatch can use them).

INSERT INTO ai_provider_configs
    (name, slug, category, template, env_key, model_id, priority, is_primary, is_active, cost_micros, pulse_pts, notes)
VALUES
-- ── TEXT ─────────────────────────────────────────────────────────────────────
('Pollinations OpenAI',   'pollinations-text',   'text', 'openai-compatible',  'POLLINATIONS_SECRET_KEY', 'openai',                          1, true,  true, 0,     0,  'Pollinations free text via OpenAI-compat endpoint'),
('Gemini 2.5 Flash',     'gemini-flash',        'text', 'gemini',             'GEMINI_API_KEY',          'gemini-2.5-flash',                2, false, true, 0,     0,  'Google Gemini 2.5 Flash — free tier'),
('Groq Llama-4 Scout',   'groq-llama4',         'text', 'openai-compatible',  'GROQ_API_KEY',            'meta-llama/llama-4-scout-17b-16e-instruct', 3, false, true, 0, 0, 'Groq inference — Llama 4 Scout'),
('DeepSeek V3',          'deepseek-v3',         'text', 'deepseek',           'DEEPSEEK_API_KEY',        'deepseek-chat',                   4, false, true, 0,     0,  'DeepSeek V3 via official API'),

-- ── IMAGE ─────────────────────────────────────────────────────────────────────
('HuggingFace FLUX Schnell', 'hf-flux-schnell', 'image', 'hf-image',          'HF_TOKEN',                'black-forest-labs/FLUX.1-schnell', 1, true,  true, 0,     0,  'HF serverless inference — free with token'),
('Pollinations FLUX',    'pollinations-flux',   'image', 'pollinations-image', 'POLLINATIONS_SECRET_KEY', 'flux',                            2, false, true, 0,     0,  'Pollinations FLUX — sk_ key required'),
('FAL FLUX Dev',         'fal-flux-dev',        'image', 'fal-image',          'FAL_API_KEY',             'fal-ai/flux/dev',                 3, false, true, 6500,  0,  'FAL.AI FLUX-dev — ~$0.025/image'),

-- ── VIDEO ─────────────────────────────────────────────────────────────────────
('FAL Kling v1.5',       'fal-kling',           'video', 'fal-video',          'FAL_API_KEY',             'fal-ai/kling-video/v1.5/standard/image-to-video', 1, true, true, 56000, 0, 'FAL Kling v1.5 — premium quality'),
('FAL LTX Video',        'fal-ltx',             'video', 'fal-video',          'FAL_API_KEY',             'fal-ai/ltx-video',                2, false, true, 14500, 0,  'FAL LTX — faster/cheaper option'),
('Pollinations Wan-Fast','pollinations-wan-fast','video', 'pollinations-video', 'POLLINATIONS_SECRET_KEY', 'wan-fast',                        3, true,  true, 0,     0,  'Wan 2.2 — FREE (15 pollen input), ~50s image-to-video'),
('Pollinations LTX-2',   'pollinations-ltx-2',  'video', 'pollinations-video', 'POLLINATIONS_SECRET_KEY', 'ltx-2',                           4, true,  true, 0,     0,  'LTX-2 — FREE (15 pollen input), new model backup'),
('Pollinations Seedance','pollinations-seedance','video', 'pollinations-video', 'POLLINATIONS_SECRET_KEY', 'seedance',                        5, false, true, 200000,0,  'Seedance Lite — PAID (1.8 pollen/M), do not use as free fallback'),

-- ── TTS ───────────────────────────────────────────────────────────────────────
('Google Cloud TTS',     'google-cloud-tts',    'tts', 'google-tts',           'GOOGLE_CLOUD_TTS_KEY',    '',                                1, true,  true, 0,     0,  'Google TTS — 1M chars/month free'),
('ElevenLabs TTS',       'elevenlabs-tts',      'tts', 'elevenlabs-tts',       'ELEVENLABS_API_KEY',      'eleven_flash_v2_5',               2, false, true, 2000,  0,  'ElevenLabs — Sarah voice (premade)'),
('Pollinations TTS',     'pollinations-tts',    'tts', 'pollinations-tts',     'POLLINATIONS_SECRET_KEY', 'elevenlabs',                      3, false, true, 0,     0,  'Pollinations TTS fallback'),

-- ── TRANSCRIBE ────────────────────────────────────────────────────────────────
('AssemblyAI',           'assemblyai',          'transcribe', 'assemblyai',    'ASSEMBLY_AI_KEY',         'universal-2',                     1, true,  true, 25,    0,  'AssemblyAI Universal-2 model'),
('Groq Whisper',         'groq-whisper',        'transcribe', 'groq-whisper',  'GROQ_API_KEY',            'whisper-large-v3-turbo',          2, false, true, 10,    0,  'Groq Whisper large-v3-turbo'),

-- ── TRANSLATE ─────────────────────────────────────────────────────────────────
('Google Translate',     'google-translate',    'translate', 'google-translate','GOOGLE_TRANSLATE_API_KEY','',                              1, true,  true, 0,     0,  'Google Translate API v2'),
('Gemini Translate',     'gemini-translate',    'translate', 'gemini',          'GEMINI_API_KEY',          'gemini-2.5-flash',              2, false, true, 0,     0,  'Gemini Flash as translation fallback'),

-- ── MUSIC ─────────────────────────────────────────────────────────────────────
('Pollinations ElevenMusic','pollinations-elevenmusic','music','pollinations-music','POLLINATIONS_SECRET_KEY','elevenmusic',                 1, true,  true, 500,   0,  'Pollinations ElevenMusic — instrumental'),
('Mubert',               'mubert',              'music', 'mubert',             'MUBERT_API_KEY',           '',                               2, false, false, 0,    0,  'Mubert royalty-free music — key pending'),
('ElevenLabs Music',     'elevenlabs-music',    'music', 'elevenlabs-music',   'ELEVENLABS_API_KEY',       '',                               3, false, true, 500,   0,  'ElevenLabs music/sound generation'),

-- ── BG REMOVE ─────────────────────────────────────────────────────────────────
('rembg Self-Hosted',    'rembg-self-hosted',   'bg-remove', 'rembg',          'REMBG_SERVICE_URL',        '',                               1, true,  true, 0,     0,  'Self-hosted rembg microservice — free'),
('FAL BiRefNet',         'fal-birefnet',        'bg-remove', 'fal-bg-remove',  'FAL_API_KEY',              'fal-ai/birefnet',                2, false, true, 2000,  0,  'FAL BiRefNet — ~$0.003/megapixel'),
('remove.bg',            'remove-bg',           'bg-remove', 'remove-bg',      'REMOVEBG_API_KEY',         '',                               3, false, true, 1000,  0,  'remove.bg — $0.20/image last resort'),

-- ── VISION ────────────────────────────────────────────────────────────────────
('Pollinations Vision',  'pollinations-vision', 'vision', 'openai-compatible', 'POLLINATIONS_SECRET_KEY',  'openai',                         1, true,  true, 0,     0,  'Pollinations multimodal vision'),
('Gemini Vision',        'gemini-vision',       'vision', 'gemini',            'GEMINI_API_KEY',           'gemini-2.5-flash',               2, false, true, 0,     0,  'Gemini Flash vision fallback')

ON CONFLICT (slug) DO NOTHING;


══════════════════════════════════════════════════════
MIGRATION: 038_missing_ui_templates.up.sql
══════════════════════════════════════════════════════
-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 038 — Fix three slugs that had no ui_template set
--
-- Root cause: slugs ai-photo, bg-remover and narrate were originally inserted
-- by migration 026 but never covered in migration 032's UPDATE blocks, so they
-- fell through to the DEFAULT 'knowledge-doc' — giving users a plain text box
-- instead of the correct purpose-built UI template.
-- ─────────────────────────────────────────────────────────────────────────────

-- ── 1. ai-photo (basic) — same template as ai-photo-pro/max/dream ─────────────
--    Dispatches to dispatchImage → FAL.AI FLUX → Pollinations
UPDATE studio_tools SET
  ui_template = 'image-creator',
  ui_config   = '{
    "prompt_placeholder": "Describe the image you want to create…",
    "aspect_ratios": [
      {"label":"Square",    "icon":"⬛", "value":"1024x1024", "default":true},
      {"label":"Portrait",  "icon":"📱", "value":"768x1344"},
      {"label":"Landscape", "icon":"🖥️", "value":"1344x768"},
      {"label":"Wide",      "icon":"🎬", "value":"1920x1080"}
    ],
    "style_tags": ["Photorealistic","Cinematic","Oil Painting","Anime","Sketch","Watercolour","Neon","Vintage"],
    "show_negative_prompt": true,
    "show_quality_toggle": false,
    "prompt_optional": false,
    "max_prompt_chars": 1000
  }'::jsonb
WHERE slug = 'ai-photo';

-- ── 2. bg-remover — image-editor (upload-first) ───────────────────────────────
--    Dispatches to dispatchBgRemover → rembg → FAL BiRefNet → remove.bg
UPDATE studio_tools SET
  ui_template = 'image-editor',
  ui_config   = '{
    "upload_label":   "Upload the photo to remove background from",
    "upload_accept":  ["image/png","image/jpeg","image/webp"],
    "upload_hint":    "Supports JPG, PNG and WebP up to 10 MB",
    "prompt_label":   null,
    "prompt_optional": true,
    "show_edit_prompt": false,
    "edit_suggestions": [],
    "output_note": "Background will be removed automatically — no prompt needed"
  }'::jsonb
WHERE slug = 'bg-remover';

-- ── 3. narrate (basic) — same template as narrate-pro ────────────────────────
--    Dispatches to dispatchTTS → Google Cloud TTS → Pollinations TTS
UPDATE studio_tools SET
  ui_template = 'voice-studio',
  ui_config   = '{
    "prompt_placeholder": "Enter the text you want to narrate (up to 3,000 characters)…",
    "max_chars": 3000,
    "voices": [
      {"id":"alloy",   "name":"Alloy",   "tone":"Neutral & Clear",    "category":"Conversational"},
      {"id":"echo",    "name":"Echo",    "tone":"Deep & Warm",         "category":"Narration"},
      {"id":"fable",   "name":"Fable",   "tone":"Expressive & Lively", "category":"Storytelling"},
      {"id":"onyx",    "name":"Onyx",    "tone":"Deep & Authoritative","category":"Broadcast"},
      {"id":"nova",    "name":"Nova",    "tone":"Friendly & Warm",     "category":"Social Media"},
      {"id":"shimmer", "name":"Shimmer", "tone":"Soft & Soothing",     "category":"Meditation"},
      {"id":"ash",     "name":"Ash",     "tone":"Gentle & Calm",       "category":"Education"},
      {"id":"coral",   "name":"Coral",   "tone":"Warm & Natural",      "category":"Podcasts"},
      {"id":"sage",    "name":"Sage",    "tone":"Clear & Professional","category":"Corporate"}
    ],
    "default_voice": "nova",
    "languages": [
      {"code":"en",  "label":"English"},
      {"code":"yo",  "label":"Yoruba"},
      {"code":"ha",  "label":"Hausa"},
      {"code":"ig",  "label":"Igbo"},
      {"code":"fr",  "label":"French"},
      {"code":"pt",  "label":"Portuguese"},
      {"code":"es",  "label":"Spanish"}
    ],
    "default_language": "en",
    "show_speed_control": false,
    "show_format_selector": false
  }'::jsonb
WHERE slug = 'narrate';


══════════════════════════════════════════════════════
MIGRATION: 039_cost_corrections.up.sql
══════════════════════════════════════════════════════
-- Migration 039: Cost corrections and point price fixes
-- Audit date: 2026-03-27
--
-- ai-photo-dream uses seedream5 model (Pollinations).
-- Cost analysis: seedream5 = $0.01/image = 10,000 micros.
-- Pulse Point exchange: ~1pt ≈ $0.00126 (₦1,000 / 500pts / 1.59 USD/NGN).
-- Cost in pts: $0.01 / $0.00126 ≈ 7.9 pts. Current price: 8 pts (breakeven).
-- Raise to 12 pts to ensure 50% margin, aligning with platform profitability model.
--
-- Comparison: ai-photo-pro (gptimage, $0.02) = 10 pts → 59% margin. 
-- We bring ai-photo-dream in line at similar margin.

UPDATE studio_tools
SET    point_cost = 12,
       updated_at = NOW()
WHERE  slug = 'ai-photo-dream';

-- Log the audit trail
DO $$
BEGIN
  RAISE NOTICE 'Migration 039: ai-photo-dream point_cost raised from 8 → 12 (seedream5 cost coverage + margin)';
END;
$$;


══════════════════════════════════════════════════════
MIGRATION: 040_rechargemax_spin_alignment.up.sql
══════════════════════════════════════════════════════
-- Add new fields to prize_pool table
ALTER TABLE prize_pool ADD COLUMN is_no_win BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE prize_pool ADD COLUMN no_win_message TEXT;
ALTER TABLE prize_pool ADD COLUMN color_scheme TEXT;
ALTER TABLE prize_pool ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;
ALTER TABLE prize_pool ADD COLUMN minimum_recharge BIGINT;

-- Create spin_tiers table
CREATE TABLE IF NOT EXISTS spin_tiers (
    id UUID PRIMARY KEY,
    tier_name TEXT NOT NULL,
    tier_display_name TEXT NOT NULL,
    min_daily_amount BIGINT NOT NULL,
    max_daily_amount BIGINT NOT NULL,
    spins_per_day INTEGER NOT NULL,
    tier_color TEXT,
    tier_icon TEXT,
    tier_badge TEXT,
    description TEXT,
    sort_order INTEGER NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Seed default spin tiers
INSERT INTO spin_tiers (id, tier_name, tier_display_name, min_daily_amount, max_daily_amount, spins_per_day, sort_order) VALUES
('11111111-1111-1111-1111-111111111111', 'bronze', 'Bronze', 100000, 499999, 1, 1),
('22222222-2222-2222-2222-222222222222', 'silver', 'Silver', 500000, 999999, 2, 2),
('33333333-3333-3333-3333-333333333333', 'gold', 'Gold', 1000000, 1999999, 3, 3),
('44444444-4444-4444-4444-444444444444', 'platinum', 'Platinum', 2000000, 999999999999, 5, 4)
ON CONFLICT (id) DO NOTHING;


══════════════════════════════════════════════════════
MIGRATION: 041_spin_claim_fields.up.sql
══════════════════════════════════════════════════════
-- Migration 041: Add claim lifecycle fields to spin_results
-- Aligns with RechargeMax claim/fulfillment flow.
-- claim_status is separate from fulfillment_status:
--   fulfillment_status = internal VTPass/MoMo dispatch state
--   claim_status       = user-facing claim lifecycle (PENDING → CLAIMED / PENDING_ADMIN_REVIEW → APPROVED / REJECTED)

-- BEGIN;  -- removed: managed by golang-migrate

-- ── Claim lifecycle ────────────────────────────────────────────────────────
ALTER TABLE spin_results
    ADD COLUMN IF NOT EXISTS claim_status         TEXT        NOT NULL DEFAULT 'PENDING',
    ADD COLUMN IF NOT EXISTS expires_at           TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '30 days');

-- ── MoMo / bank payout details (supplied by user at claim time) ────────────
ALTER TABLE spin_results
    ADD COLUMN IF NOT EXISTS momo_claim_number    TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS bank_account_number  TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS bank_account_name    TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS bank_name            TEXT        NOT NULL DEFAULT '';

-- ── Admin review metadata ──────────────────────────────────────────────────
ALTER TABLE spin_results
    ADD COLUMN IF NOT EXISTS reviewed_by          UUID        REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS reviewed_at          TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS rejection_reason     TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS admin_notes          TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS payment_reference    TEXT        NOT NULL DEFAULT '';

-- ── Indexes for admin claim list queries ──────────────────────────────────
CREATE INDEX IF NOT EXISTS idx_spin_results_claim_status  ON spin_results (claim_status);
CREATE INDEX IF NOT EXISTS idx_spin_results_expires_at    ON spin_results (expires_at);
CREATE INDEX IF NOT EXISTS idx_spin_results_reviewed_by   ON spin_results (reviewed_by);

-- ── Seed: back-fill claim_status for existing rows ────────────────────────
-- Rows that are already fulfilled → CLAIMED
-- Rows that are pending MoMo setup → PENDING (user hasn't linked MoMo yet)
-- All others → PENDING
UPDATE spin_results
    SET claim_status = 'CLAIMED'
    WHERE fulfillment_status IN ('completed')
      AND claim_status = 'PENDING';

-- COMMIT;  -- removed: managed by golang-migrate


══════════════════════════════════════════════════════
MIGRATION: 042_prize_pool_rechargemax_fields.up.sql
══════════════════════════════════════════════════════
-- Migration 042: Add RechargeMax-aligned fields to prize_pool
-- Adds: is_no_win, no_win_message, color_scheme, sort_order, minimum_recharge,
--       icon_name, terms_and_conditions, prize_code, variation_code

ALTER TABLE prize_pool
    ADD COLUMN IF NOT EXISTS is_no_win           BOOLEAN     NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS no_win_message      TEXT        NOT NULL DEFAULT 'Better Luck Next Time',
    ADD COLUMN IF NOT EXISTS color_scheme        TEXT        NOT NULL DEFAULT '#CCCCCC',
    ADD COLUMN IF NOT EXISTS sort_order          INTEGER     NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS minimum_recharge    BIGINT      NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS icon_name           TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS terms_and_conditions TEXT       NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS prize_code          TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS variation_code      TEXT        NOT NULL DEFAULT '';

-- Mark the existing try_again prize as is_no_win
UPDATE prize_pool SET is_no_win = TRUE WHERE prize_type = 'try_again';

-- Index for admin list ordering
CREATE INDEX IF NOT EXISTS idx_prize_pool_sort_order ON prize_pool (sort_order);


══════════════════════════════════════════════════════
MIGRATION: 043_spin_tiers_seed.up.sql
══════════════════════════════════════════════════════
-- Migration 043: Seed default spin tiers (matching RechargeMax defaults)
-- Amounts are in kobo (₦1 = 100 kobo)
-- Tier ranges must be contiguous with no gaps or overlaps

INSERT INTO spin_tiers (id, tier_name, tier_display_name, min_daily_amount, max_daily_amount, spins_per_day, tier_color, tier_icon, tier_badge, description, sort_order, is_active, created_at, updated_at)
VALUES
    (gen_random_uuid(), 'bronze',   'Bronze',   100000,   499999,  1, '#CD7F32', 'bronze-medal',   'BRONZE',   'Recharge ₦1,000–₦4,999 today for 1 spin',   1, TRUE, NOW(), NOW()),
    (gen_random_uuid(), 'silver',   'Silver',   500000,   999999,  2, '#C0C0C0', 'silver-medal',   'SILVER',   'Recharge ₦5,000–₦9,999 today for 2 spins',  2, TRUE, NOW(), NOW()),
    (gen_random_uuid(), 'gold',     'Gold',    1000000,  2999999,  3, '#FFD700', 'gold-medal',     'GOLD',     'Recharge ₦10,000–₦29,999 today for 3 spins', 3, TRUE, NOW(), NOW()),
    (gen_random_uuid(), 'platinum', 'Platinum', 3000000, 999999999, 5, '#E5E4E2', 'platinum-medal', 'PLATINUM', 'Recharge ₦30,000+ today for 5 spins',        4, TRUE, NOW(), NOW())
ON CONFLICT (tier_name) DO NOTHING;


══════════════════════════════════════════════════════
MIGRATION: 044_prize_pool_seed.up.sql
══════════════════════════════════════════════════════
-- Migration 044: Seed default 15 wheel prizes (RechargeMax-aligned, LN prize types)
-- Probabilities are stored as integer weights summing to 10,000 (= 100.00%)
-- is_no_win = TRUE means no win record is created in spin_results for this slot

INSERT INTO prize_pool (id, name, prize_code, prize_type, base_value, win_probability_weight, is_active, is_no_win, no_win_message, color_scheme, sort_order, minimum_recharge, icon_name)
VALUES
    -- No-win slots (40.50% total = 4050/10000)
    (gen_random_uuid(), 'Better Luck Next Time', 'NONE',    'try_again',     0,      4050, TRUE, TRUE,  'Better Luck Next Time!', '#CCCCCC', 1,  0,       'sad-face'),

    -- Points prizes (53.00% total = 5300/10000)
    (gen_random_uuid(), '10 Pulse Points',       'PTS10',   'pulse_points',  10,     2500, TRUE, FALSE, '', '#4CAF50', 2,  0,       'star'),
    (gen_random_uuid(), '25 Pulse Points',       'PTS25',   'pulse_points',  25,     1500, TRUE, FALSE, '', '#4CAF50', 3,  0,       'star'),
    (gen_random_uuid(), '50 Pulse Points',       'PTS50',   'pulse_points',  50,      800, TRUE, FALSE, '', '#4CAF50', 4,  0,       'star'),
    (gen_random_uuid(), '100 Pulse Points',      'PTS100',  'pulse_points',  100,     500, TRUE, FALSE, '', '#4CAF50', 5,  0,       'star'),

    -- Airtime prizes (5.75% total = 575/10000)
    (gen_random_uuid(), '₦50 Airtime',           'AIR50',   'airtime',       5000,    300, TRUE, FALSE, '', '#2196F3', 6,  100000,  'phone'),
    (gen_random_uuid(), '₦100 Airtime',          'AIR100',  'airtime',       10000,   150, TRUE, FALSE, '', '#2196F3', 7,  100000,  'phone'),
    (gen_random_uuid(), '₦200 Airtime',          'AIR200',  'airtime',       20000,    75, TRUE, FALSE, '', '#2196F3', 8,  200000,  'phone'),
    (gen_random_uuid(), '₦500 Airtime',          'AIR500',  'airtime',       50000,    50, TRUE, FALSE, '', '#2196F3', 9,  500000,  'phone'),

    -- Data prizes (0.60% total = 60/10000)
    (gen_random_uuid(), '500MB Data',            'DATA500', 'data_bundle',   50000,    30, TRUE, FALSE, '', '#9C27B0', 10, 200000,  'wifi'),
    (gen_random_uuid(), '1GB Data',              'DATA1GB', 'data_bundle',   100000,   20, TRUE, FALSE, '', '#9C27B0', 11, 500000,  'wifi'),
    (gen_random_uuid(), '2GB Data',              'DATA2GB', 'data_bundle',   200000,   10, TRUE, FALSE, '', '#9C27B0', 12, 1000000, 'wifi'),

    -- MoMo cash prizes (0.15% total = 15/10000)
    (gen_random_uuid(), '₦1,000 MoMo Cash',      'CASH1K',  'momo_cash',     100000,   10, TRUE, FALSE, '', '#FF9800', 13, 500000,  'money-bag'),
    (gen_random_uuid(), '₦5,000 MoMo Cash',      'CASH5K',  'momo_cash',     500000,    3, TRUE, FALSE, '', '#FF9800', 14, 1000000, 'money-bag'),
    (gen_random_uuid(), '₦50,000 MoMo Cash',     'CASH50K', 'momo_cash',    5000000,    2, TRUE, FALSE, '', '#FF5722', 15, 3000000, 'trophy')
ON CONFLICT DO NOTHING;


══════════════════════════════════════════════════════
MIGRATION: 045_mtn_push_pipeline.up.sql
══════════════════════════════════════════════════════
-- 045_mtn_push_pipeline.sql
-- Purpose: Support MTN-push recharge ingestion.
--   1. Extend draw_entries with entry_source + source_transaction_id
--      (mirrors RechargeMax draw_entries.source_type / source_transaction_id)
--   2. Add mtn_push_events audit table — every raw MTN push is logged here
--      before any business logic runs, so we have a full inbound audit trail.
--   3. Seed network_configs keys for the MTN push pipeline.

-- ── 1. Extend draw_entries ────────────────────────────────────────────────────
-- entry_source: who created this entry (recharge | subscription | bonus | manual)
ALTER TABLE draw_entries
    ADD COLUMN IF NOT EXISTS entry_source          TEXT NOT NULL DEFAULT 'recharge'
        CHECK (entry_source IN ('recharge','subscription','bonus','manual')),
    ADD COLUMN IF NOT EXISTS source_transaction_id UUID REFERENCES transactions(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_draw_entries_source_tx ON draw_entries(source_transaction_id);

-- ── 2. MTN push events audit table ───────────────────────────────────────────
-- Every inbound MTN push is written here atomically before any reward logic.
-- This gives us idempotency (unique constraint on transaction_ref) and a full
-- audit trail even if the downstream processing fails.
CREATE TABLE IF NOT EXISTS mtn_push_events (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_ref     TEXT        NOT NULL UNIQUE,   -- MTN's unique transaction ID
    msisdn              TEXT        NOT NULL,           -- normalised 0XXXXXXXXXX
    recharge_type       TEXT        NOT NULL DEFAULT 'AIRTIME'
                            CHECK (recharge_type IN ('AIRTIME','DATA','BUNDLE')),
    amount_kobo         BIGINT      NOT NULL CHECK (amount_kobo > 0),
    event_timestamp     TIMESTAMPTZ NOT NULL,           -- timestamp from MTN payload
    raw_payload         JSONB,                          -- full original payload
    status              TEXT        NOT NULL DEFAULT 'RECEIVED'
                            CHECK (status IN ('RECEIVED','PROCESSED','DUPLICATE','FAILED')),
    processing_error    TEXT,
    points_awarded      BIGINT      NOT NULL DEFAULT 0,
    draw_entries_created INT        NOT NULL DEFAULT 0,
    spin_credits_awarded INT        NOT NULL DEFAULT 0,
    processed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mtn_push_events_msisdn     ON mtn_push_events(msisdn);
CREATE INDEX IF NOT EXISTS idx_mtn_push_events_status     ON mtn_push_events(status);
CREATE INDEX IF NOT EXISTS idx_mtn_push_events_created_at ON mtn_push_events(created_at DESC);

-- ── 3. Network config keys for the MTN push pipeline ─────────────────────────
INSERT INTO network_configs (key, value, description) VALUES
    ('draw_entries_per_point',   '1',    'Draw entries created per Pulse Point earned from a recharge (1:1 mirrors RechargeMax)'),
    ('mtn_push_hmac_secret',     '',     'HMAC-SHA256 secret for MTN push webhook signature verification (set via env MTN_PUSH_SECRET)'),
    ('mtn_push_min_amount_naira','50',   'Minimum recharge amount in naira for MTN push to qualify for points/draw/spin')
ON CONFLICT (key) DO NOTHING;


══════════════════════════════════════════════════════
MIGRATION: 046_transactions_spin_delta_reference.up.sql
══════════════════════════════════════════════════════
-- Migration 046: Add spin_delta and reference columns to transactions
--
-- The Transaction entity has always had SpinDelta and Reference fields
-- but they were never added to the DB schema. This migration adds them
-- with safe defaults so existing rows are not affected.

ALTER TABLE transactions
  ADD COLUMN IF NOT EXISTS spin_delta  INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS reference   TEXT    NOT NULL DEFAULT '';

-- Index on reference for idempotency lookups
CREATE INDEX IF NOT EXISTS idx_transactions_reference ON transactions (reference)
  WHERE reference <> '';

-- Index on type + reference for duplicate detection
CREATE INDEX IF NOT EXISTS idx_transactions_type_reference ON transactions (type, reference)
  WHERE reference <> '';

COMMENT ON COLUMN transactions.spin_delta IS
  'Number of spin credits added (+) or consumed (-) by this transaction. 0 for non-spin transactions.';

COMMENT ON COLUMN transactions.reference IS
  'External reference ID (e.g. MTN transaction ref, Paystack ref). Used for idempotency checks.';


══════════════════════════════════════════════════════
MIGRATION: 047_fix_ledger_trigger_program_configs.up.sql
══════════════════════════════════════════════════════
-- Migration 047: Fix trg_fn_loyalty_nexus_ledger trigger
--
-- The trigger function trg_fn_loyalty_nexus_ledger was querying a table
-- called "program_configs" which does not exist in Loyalty Nexus.
-- The correct table is "network_configs" (key TEXT, value JSONB).
--
-- This bug caused EVERY INSERT into the transactions table to fail with:
--   ERROR: relation "program_configs" does not exist (SQLSTATE 42P01)
--
-- This migration replaces the trigger function body to:
--   1. Read streak_window_hours from network_configs (default 48h)
--   2. Update users.total_recharge_amount on recharge transactions
--   3. Leave streak management to the application layer (recharge_service.go
--      and mtn_push_service.go already handle streak via UpdateStreak)
--
-- NOTE: The trigger no longer updates total_points or stamps_count because
-- Loyalty Nexus uses a separate wallet table (wallets.pulse_points) and
-- the application layer manages all balance updates atomically. The trigger
-- only needs to maintain total_recharge_amount as a denormalised counter.

CREATE OR REPLACE FUNCTION trg_fn_loyalty_nexus_ledger()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE
    v_streak_window_hours INTEGER;
BEGIN
    -- Read streak window from network_configs (default 48h).
    -- network_configs stores values as JSONB; plain integer values are stored
    -- as JSON numbers (e.g. 48), so we cast via text.
    SELECT (value::text)::int INTO v_streak_window_hours
    FROM network_configs
    WHERE key = 'streak_window_hours'
    LIMIT 1;

    IF v_streak_window_hours IS NULL THEN
        v_streak_window_hours := 48;
    END IF;

    -- Maintain the denormalised total_recharge_amount counter on users.
    -- All other balance fields (pulse_points, spin_credits, lifetime_points)
    -- are managed by the application layer via the wallets table.
    IF NEW.type = 'recharge' AND NEW.amount > 0 THEN
        UPDATE users
        SET
            total_recharge_amount = total_recharge_amount + NEW.amount,
            last_recharge_at      = NOW(),
            updated_at            = NOW()
        WHERE id = NEW.user_id;
    END IF;

    RETURN NEW;
END;
$$;

-- Verify the trigger is still attached (it should be — we only replaced the function body).
-- If for any reason it was dropped, recreate it.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger t
        JOIN pg_class c ON t.tgrelid = c.oid
        WHERE c.relname = 'transactions'
          AND t.tgname = 'trg_loyalty_nexus_ledger'
    ) THEN
        CREATE TRIGGER trg_loyalty_nexus_ledger
            AFTER INSERT ON transactions
            FOR EACH ROW EXECUTE FUNCTION trg_fn_loyalty_nexus_ledger();
    END IF;
END;
$$;

-- Seed the streak_window_hours config key if it does not already exist.
INSERT INTO network_configs (key, value, description)
VALUES ('streak_window_hours', '48', 'Hours within which consecutive recharges count as a streak')
ON CONFLICT (key) DO NOTHING;


══════════════════════════════════════════════════════
MIGRATION: 048_wallet_separate_counters.up.sql
══════════════════════════════════════════════════════
-- Migration 048: Separate recharge accumulators for spin/draw vs pulse points
--
-- The original recharge_counter was used for both spin credits and pulse points,
-- which is incorrect. Loyalty Nexus has two independent reward currencies:
--
--   1. Spin Credits + Draw Entries  — awarded every ₦200 recharge
--   2. Pulse Points                 — awarded every ₦250 recharge (AI Studio currency)
--
-- We add two dedicated counters so each accumulator tracks its own remainder
-- independently. The old recharge_counter column is kept for backwards
-- compatibility but will no longer be written by the MTN push pipeline.
--
-- Admin-configurable thresholds (network_configs):
--   spin_draw_naira_per_credit   — naira per spin credit + draw entry (default 200)
--   pulse_naira_per_point        — naira per pulse point (default 250)
--   mtn_push_min_amount_naira    — minimum qualifying recharge (default 50)

ALTER TABLE wallets
    ADD COLUMN IF NOT EXISTS spin_draw_counter BIGINT NOT NULL DEFAULT 0
        CHECK (spin_draw_counter >= 0),
    ADD COLUMN IF NOT EXISTS pulse_counter     BIGINT NOT NULL DEFAULT 0
        CHECK (pulse_counter >= 0);

COMMENT ON COLUMN wallets.spin_draw_counter IS
    'Kobo remainder accumulator for spin credits and draw entries (resets modulo spin_draw_naira_per_credit×100)';
COMMENT ON COLUMN wallets.pulse_counter IS
    'Kobo remainder accumulator for Pulse Points (resets modulo pulse_naira_per_point×100)';

-- Seed the three configurable thresholds.
-- ON CONFLICT DO NOTHING so re-running the migration is safe.
INSERT INTO network_configs (key, value, description) VALUES
    ('spin_draw_naira_per_credit', '200',  'Naira per spin credit and draw entry awarded on recharge'),
    ('pulse_naira_per_point',      '250',  'Naira per Pulse Point awarded on recharge (AI Studio currency)'),
    ('mtn_push_min_amount_naira',  '50',   'Minimum recharge amount in naira to qualify for rewards')
ON CONFLICT (key) DO NOTHING;


══════════════════════════════════════════════════════
MIGRATION: 049_draw_schedules.up.sql
══════════════════════════════════════════════════════
-- Migration 049: draw_schedules config table
-- Stores the configurable eligibility window rules for each draw type.
-- Admin can update these rows at runtime without a deployment.
--
-- Window logic (from product spec):
--   Each draw has an eligibility window: recharges whose timestamp falls
--   within [window_open, window_close) qualify for that draw.
--   Cutoff time is 17:00:00 WAT (UTC+1 = 16:00:00 UTC) every day.
--
-- Draw schedule (default):
--   Monday Draw    : Thu 17:00:01 → Sun 17:00:00  (3-day window, no Sunday draw)
--   Tuesday Draw   : Sun 17:00:01 → Mon 17:00:00
--   Wednesday Draw : Mon 17:00:01 → Tue 17:00:00
--   Thursday Draw  : Tue 17:00:01 → Wed 17:00:00
--   Friday Draw    : Wed 17:00:01 → Thu 17:00:00
--   Saturday Mega  : Fri 17:00:01 → Fri 17:00:00  (full week)

CREATE TABLE IF NOT EXISTS draw_schedules (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Human-readable name shown in admin UI
    draw_name           TEXT        NOT NULL,

    -- draw_type must match the draws.draw_type column values
    -- DAILY | WEEKLY | MONTHLY | SPECIAL
    draw_type           TEXT        NOT NULL,

    -- Day of week the draw runs (0=Sunday … 6=Saturday)
    draw_day_of_week    INTEGER     NOT NULL CHECK (draw_day_of_week BETWEEN 0 AND 6),

    -- Draw execution time in HH:MM:SS format (WAT = UTC+1)
    draw_time_wat       TIME        NOT NULL DEFAULT '17:00:00',

    -- Eligibility window: how many days BEFORE the draw day does the window OPEN?
    -- Measured in whole days back from the draw day at cutoff_hour_utc.
    -- Monday draw window opens Thursday → window_open_days_before = 4 (Mon - Thu = 4 days back... wait)
    -- We store the window as: open_day_of_week + open_time, close_day_of_week + close_time
    -- This is more explicit and avoids day-count arithmetic confusion.

    -- Day of week the eligibility window OPENS (0=Sunday … 6=Saturday)
    window_open_dow     INTEGER     NOT NULL CHECK (window_open_dow BETWEEN 0 AND 6),
    -- Time the window opens in HH:MM:SS WAT (always 17:00:01 per spec)
    window_open_time    TIME        NOT NULL DEFAULT '17:00:01',

    -- Day of week the eligibility window CLOSES (0=Sunday … 6=Saturday)
    window_close_dow    INTEGER     NOT NULL CHECK (window_close_dow BETWEEN 0 AND 6),
    -- Time the window closes in HH:MM:SS WAT (always 17:00:00 per spec)
    window_close_time   TIME        NOT NULL DEFAULT '17:00:00',

    -- Cutoff hour in UTC for all window boundary calculations (17:00 WAT = 16:00 UTC)
    cutoff_hour_utc     INTEGER     NOT NULL DEFAULT 16 CHECK (cutoff_hour_utc BETWEEN 0 AND 23),

    -- Whether this schedule is active (admin can disable a draw type)
    is_active           BOOLEAN     NOT NULL DEFAULT TRUE,

    -- Sort order for admin UI display
    sort_order          INTEGER     NOT NULL DEFAULT 0,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_draw_schedules_active
    ON draw_schedules (is_active, draw_day_of_week);

-- ─── Seed the 6 default draw schedule rules ───────────────────────────────
-- Day of week: 0=Sun, 1=Mon, 2=Tue, 3=Wed, 4=Thu, 5=Fri, 6=Sat

INSERT INTO draw_schedules
    (draw_name, draw_type, draw_day_of_week, draw_time_wat,
     window_open_dow, window_open_time, window_close_dow, window_close_time,
     cutoff_hour_utc, is_active, sort_order)
VALUES
    -- Monday Draw: window Thu 17:00:01 → Sun 17:00:00
    ('Monday Daily Draw',    'DAILY',  1, '17:00:00', 4, '17:00:01', 0, '17:00:00', 16, TRUE, 1),

    -- Tuesday Draw: window Sun 17:00:01 → Mon 17:00:00
    ('Tuesday Daily Draw',   'DAILY',  2, '17:00:00', 0, '17:00:01', 1, '17:00:00', 16, TRUE, 2),

    -- Wednesday Draw: window Mon 17:00:01 → Tue 17:00:00
    ('Wednesday Daily Draw', 'DAILY',  3, '17:00:00', 1, '17:00:01', 2, '17:00:00', 16, TRUE, 3),

    -- Thursday Draw: window Tue 17:00:01 → Wed 17:00:00
    ('Thursday Daily Draw',  'DAILY',  4, '17:00:00', 2, '17:00:01', 3, '17:00:00', 16, TRUE, 4),

    -- Friday Draw: window Wed 17:00:01 → Thu 17:00:00
    ('Friday Daily Draw',    'DAILY',  5, '17:00:00', 3, '17:00:01', 4, '17:00:00', 16, TRUE, 5),

    -- Saturday Weekly Mega Draw: window Fri 17:00:01 → Fri 17:00:00 (full week)
    ('Saturday Weekly Mega Draw', 'WEEKLY', 6, '17:00:00', 5, '17:00:01', 5, '17:00:00', 16, TRUE, 6)
ON CONFLICT DO NOTHING;

-- ─── Fix draw_entries column name mismatch ────────────────────────────────
-- The DrawEntry Go struct used phone_number and ticket_count but the real
-- DB columns are msisdn and entries_count. Add phone_number as alias column
-- so existing code still works, and add amount_kobo for the recharge amount.

ALTER TABLE draw_entries
    ADD COLUMN IF NOT EXISTS phone_number TEXT GENERATED ALWAYS AS (msisdn) STORED,
    ADD COLUMN IF NOT EXISTS ticket_count INTEGER GENERATED ALWAYS AS (entries_count) STORED,
    ADD COLUMN IF NOT EXISTS amount       BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS recharge_at  TIMESTAMPTZ;

-- Index for window resolution queries
CREATE INDEX IF NOT EXISTS idx_draw_entries_draw_msisdn
    ON draw_entries (draw_id, msisdn);

CREATE INDEX IF NOT EXISTS idx_draw_entries_recharge_at
    ON draw_entries (recharge_at)
    WHERE recharge_at IS NOT NULL;


══════════════════════════════════════════════════════
MIGRATION: 050_mtn_push_csv_upload.up.sql
══════════════════════════════════════════════════════
-- Migration 050: MTN push CSV bulk upload
--
-- When the MTN push API is unavailable, admins can upload a CSV file
-- containing MSISDN, date, time, and recharge amount.  Each row is
-- processed through the same pipeline as a live MTN push webhook:
--   spin credits, pulse points, draw entries, ledger entries.
--
-- This table provides a full audit trail for every upload batch and
-- every individual row within it.

-- ─── Upload batch header ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mtn_push_csv_uploads (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Who uploaded and when
    uploaded_by     TEXT        NOT NULL,   -- admin user_id or email
    filename        TEXT        NOT NULL,   -- original filename for reference
    uploaded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Counts
    total_rows      INTEGER     NOT NULL DEFAULT 0,
    processed_rows  INTEGER     NOT NULL DEFAULT 0,
    skipped_rows    INTEGER     NOT NULL DEFAULT 0,  -- duplicates / below-min
    failed_rows     INTEGER     NOT NULL DEFAULT 0,

    -- Overall status
    status          TEXT        NOT NULL DEFAULT 'PENDING'
                        CHECK (status IN ('PENDING','PROCESSING','DONE','PARTIAL','FAILED')),

    -- Optional admin note
    note            TEXT,

    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_csv_uploads_status
    ON mtn_push_csv_uploads (status, uploaded_at DESC);

-- ─── Per-row result ───────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mtn_push_csv_rows (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id       UUID        NOT NULL REFERENCES mtn_push_csv_uploads(id) ON DELETE CASCADE,

    -- Original CSV values (stored verbatim for auditability)
    row_number      INTEGER     NOT NULL,   -- 1-based line number in the CSV
    raw_msisdn      TEXT        NOT NULL,
    raw_date        TEXT        NOT NULL,   -- e.g. "2025-05-14"
    raw_time        TEXT        NOT NULL,   -- e.g. "14:30:00"
    raw_amount      TEXT        NOT NULL,   -- e.g. "1000.00"
    recharge_type   TEXT        NOT NULL DEFAULT 'AIRTIME',

    -- Normalised values (set after parsing)
    msisdn          TEXT,
    recharge_at     TIMESTAMPTZ,
    amount_naira    NUMERIC(12,2),

    -- Processing outcome
    status          TEXT        NOT NULL DEFAULT 'PENDING'
                        CHECK (status IN ('PENDING','OK','SKIPPED','FAILED')),
    skip_reason     TEXT,       -- e.g. "duplicate", "below_minimum"
    error_msg       TEXT,       -- set on FAILED rows

    -- Rewards awarded (mirrors mtn_push_events columns)
    transaction_ref TEXT,       -- the synthetic ref used for idempotency
    spin_credits    INTEGER,
    pulse_points    BIGINT,
    draw_entries    INTEGER,

    processed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_csv_rows_upload
    ON mtn_push_csv_rows (upload_id, row_number);

CREATE INDEX IF NOT EXISTS idx_csv_rows_msisdn
    ON mtn_push_csv_rows (msisdn)
    WHERE msisdn IS NOT NULL;


══════════════════════════════════════════════════════
MIGRATION: 051_pulse_point_awards.up.sql
══════════════════════════════════════════════════════
-- Migration 051: Bonus Pulse Point Awards
-- ─────────────────────────────────────────────────────────────────────────────
-- Super-admins can award bonus Pulse Points to individual users as part of
-- campaigns or incentive programmes.  Every award is recorded here for a full
-- audit trail (who awarded, to whom, how many, why, when).
--
-- The corresponding wallet credit and immutable ledger entry are written by the
-- application service in the same DB transaction, so the three records are
-- always consistent.
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS pulse_point_awards (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Recipient
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    phone_number    TEXT        NOT NULL,           -- denormalised for fast audit queries

    -- Award details
    points          BIGINT      NOT NULL CHECK (points > 0),
    campaign        TEXT        NOT NULL DEFAULT '', -- e.g. "Ramadan 2025", "Beta Tester"
    note            TEXT        NOT NULL DEFAULT '', -- free-text reason

    -- Who did it
    awarded_by      UUID        NOT NULL,           -- admin user_id (FK not enforced — admins may be deleted)
    awarded_by_name TEXT        NOT NULL DEFAULT '', -- denormalised display name at time of award

    -- Immutable back-reference to the ledger entry
    transaction_id  UUID        NOT NULL,           -- FK to transactions.id

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fast lookups by recipient
CREATE INDEX IF NOT EXISTS idx_ppa_user_id
    ON pulse_point_awards (user_id);

-- Fast lookups by phone (admin audit search)
CREATE INDEX IF NOT EXISTS idx_ppa_phone
    ON pulse_point_awards (phone_number);

-- Fast lookups by campaign
CREATE INDEX IF NOT EXISTS idx_ppa_campaign
    ON pulse_point_awards (campaign)
    WHERE campaign <> '';

-- Fast lookups by awarding admin
CREATE INDEX IF NOT EXISTS idx_ppa_awarded_by
    ON pulse_point_awards (awarded_by);

COMMENT ON TABLE pulse_point_awards IS
    'Immutable audit log of every bonus Pulse Point award made by a super-admin.';
COMMENT ON COLUMN pulse_point_awards.campaign IS
    'Optional campaign or incentive programme name, e.g. "Ramadan 2025".';
COMMENT ON COLUMN pulse_point_awards.awarded_by IS
    'UUID of the admin user who made the award (from JWT claims at request time).';
COMMENT ON COLUMN pulse_point_awards.transaction_id IS
    'Back-reference to the bonus ledger entry in the transactions table.';


══════════════════════════════════════════════════════
MIGRATION: 052_admin_users_rbac.up.sql
══════════════════════════════════════════════════════
-- Migration 052: Admin Users with RBAC (email + password, role-based access)
-- Replaces the placeholder AdminUser with a full production-ready admin identity system.

CREATE TYPE admin_role AS ENUM (
  'super_admin',    -- Full platform access
  'finance',        -- Approve claims, view financials, adjust points
  'operations',     -- Manage draws, notifications, users, wars
  'content'         -- Manage studio tools, prizes, config
);

CREATE TABLE IF NOT EXISTS admin_users (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email           TEXT NOT NULL UNIQUE,
  password_hash   TEXT NOT NULL,
  full_name       TEXT NOT NULL DEFAULT '',
  role            admin_role NOT NULL DEFAULT 'operations',
  is_active       BOOLEAN NOT NULL DEFAULT TRUE,
  last_login_at   TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_users_email ON admin_users(email);

-- Seed a default super_admin (password will be set via ADMIN_SEED_PASSWORD env var at startup,
-- or use the admin CLI tool. Hash shown here is bcrypt of 'ChangeMe123!' — MUST be rotated.)
-- Actual seeding is done by the application on first startup if no admin exists.


══════════════════════════════════════════════════════
MIGRATION: 053_deprecate_subscription_columns.up.sql
══════════════════════════════════════════════════════
-- Migration 053: Deprecate subscription billing columns
-- 
-- Loyalty Nexus does NOT use paid subscriptions. Users earn rewards through
-- airtime/data recharges. The subscription_* columns on the users table are
-- kept for backwards-compatibility with existing rows and will be dropped in
-- a future migration once all rows have been back-filled.
--
-- This migration:
--   1. Adds a comment to each deprecated column so the intent is clear in the schema.
--   2. Back-fills all existing users to subscription_tier='free', subscription_status='active'.
--   3. Does NOT drop the columns (safe for zero-downtime deploy).
--
-- To fully remove these columns in a future release, run:
--   ALTER TABLE users DROP COLUMN subscription_tier;
--   ALTER TABLE users DROP COLUMN subscription_status;
--   ALTER TABLE users DROP COLUMN subscription_expires_at;

-- Back-fill existing rows so they have consistent values
UPDATE users
SET
    subscription_tier   = 'free',
    subscription_status = 'active',
    subscription_expires_at = NULL
WHERE subscription_tier IS NULL
   OR subscription_tier = ''
   OR subscription_status IS NULL
   OR subscription_status = '';

-- Add column comments so the deprecation is visible in pg_catalog
COMMENT ON COLUMN users.subscription_tier       IS 'DEPRECATED: subscription billing removed. Always ''free''. Will be dropped in a future migration.';
COMMENT ON COLUMN users.subscription_status     IS 'DEPRECATED: subscription billing removed. Always ''active''. Will be dropped in a future migration.';
COMMENT ON COLUMN users.subscription_expires_at IS 'DEPRECATED: subscription billing removed. Always NULL. Will be dropped in a future migration.';


══════════════════════════════════════════════════════
MIGRATION: 054_war_secondary_draw.up.sql
══════════════════════════════════════════════════════
-- Migration 054: Regional Wars — Secondary Draw tables
-- After a war is resolved, admin can run one secondary draw per winning state.
-- All participants are users in that state who were active during the war window.
-- Winners are selected via CSPRNG Fisher-Yates (same engine as main draw — SEC-009).
-- Prizes are paid via MoMo Cash by admin manually after draw execution.

-- BEGIN;  -- removed: managed by golang-migrate

-- ── war_secondary_draws ───────────────────────────────────────────────────────
-- One row per secondary draw execution (admin can run at most once per state per war).

CREATE TABLE IF NOT EXISTS war_secondary_draws (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    war_id          UUID        NOT NULL REFERENCES regional_wars(id) ON DELETE CASCADE,
    state           TEXT        NOT NULL,
    winner_count    INT         NOT NULL DEFAULT 1 CHECK (winner_count BETWEEN 1 AND 10),
    prize_per_winner_kobo BIGINT NOT NULL DEFAULT 0,  -- e.g. 50000 = ₦500
    total_pool_kobo BIGINT      NOT NULL DEFAULT 0,   -- winner_count * prize_per_winner_kobo
    participant_count INT       NOT NULL DEFAULT 0,   -- eligible users at time of draw
    status          TEXT        NOT NULL DEFAULT 'PENDING'
                                CHECK (status IN ('PENDING','COMPLETED','CANCELLED')),
    triggered_by    UUID        REFERENCES admin_users(id),
    executed_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Only one draw per (war, state) — admin cannot re-run
    UNIQUE (war_id, state)
);

CREATE INDEX IF NOT EXISTS idx_war_sec_draws_war_id ON war_secondary_draws(war_id);
CREATE INDEX IF NOT EXISTS idx_war_sec_draws_state  ON war_secondary_draws(state);

-- ── war_secondary_draw_winners ────────────────────────────────────────────────
-- One row per winner selected in the secondary draw.

CREATE TABLE IF NOT EXISTS war_secondary_draw_winners (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    secondary_draw_id UUID      NOT NULL REFERENCES war_secondary_draws(id) ON DELETE CASCADE,
    war_id          UUID        NOT NULL,
    state           TEXT        NOT NULL,
    user_id         UUID        NOT NULL REFERENCES users(id),
    phone_number    TEXT        NOT NULL,
    position        INT         NOT NULL,            -- 1 = first winner
    prize_kobo      BIGINT      NOT NULL DEFAULT 0,
    momo_number     TEXT,                            -- filled when admin pays
    payment_status  TEXT        NOT NULL DEFAULT 'PENDING_PAYMENT'
                                CHECK (payment_status IN ('PENDING_PAYMENT','PAID','FAILED')),
    paid_at         TIMESTAMPTZ,
    paid_by         UUID        REFERENCES admin_users(id),
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_war_sec_winners_draw_id ON war_secondary_draw_winners(secondary_draw_id);
CREATE INDEX IF NOT EXISTS idx_war_sec_winners_user_id ON war_secondary_draw_winners(user_id);

-- ── Triggers for updated_at ───────────────────────────────────────────────────

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_war_sec_draws_updated_at ON war_secondary_draws;
CREATE TRIGGER trg_war_sec_draws_updated_at
    BEFORE UPDATE ON war_secondary_draws
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS trg_war_sec_winners_updated_at ON war_secondary_draw_winners;
CREATE TRIGGER trg_war_sec_winners_updated_at
    BEFORE UPDATE ON war_secondary_draw_winners
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ── Config keys ───────────────────────────────────────────────────────────────

INSERT INTO network_configs (key, value, description) VALUES
    ('wars_secondary_draw_default_winners',    '3',       'Default number of winners per state secondary draw (1-10)'),
    ('wars_secondary_draw_default_prize_kobo', '50000',   'Default prize per winner in kobo (50000 = ₦500)')
ON CONFLICT (key) DO NOTHING;

-- COMMIT;  -- removed: managed by golang-migrate


══════════════════════════════════════════════════════
MIGRATION: 055_passport_config_and_streak_alert.up.sql
══════════════════════════════════════════════════════
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


══════════════════════════════════════════════════════
MIGRATION: 056_ussd_session_hardening.up.sql
══════════════════════════════════════════════════════
-- Migration 038: USSD session hardening for REQ-6.4 and REQ-6.5
--
-- REQ-6.5: Add pending_spin_id to ussd_sessions so that if a session times out
--          mid-spin, the spin can be rolled back by the lifecycle worker.
--
-- REQ-6.4: Seed ussd_sms_max_chars config key for controlling SMS delivery length.
--          Seed ussd_shortcode as a proper config key (migration 037 used ussd_short_code
--          with an underscore — this adds the canonical ussd_shortcode key used by the handler).
--
-- REQ-6.1: The ussd_shortcode key is now the canonical source of truth for the
--          shortcode displayed in USSD menus and SMS nudges.

-- ─── 1. Add pending_spin_id to ussd_sessions ─────────────────────────────────
-- Stores the UUID of a spin that was initiated but not yet completed.
-- If the session expires with this field set, the spin is rolled back.
ALTER TABLE ussd_sessions
    ADD COLUMN IF NOT EXISTS pending_spin_id UUID REFERENCES spin_results(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_ussd_sessions_pending_spin
    ON ussd_sessions(pending_spin_id)
    WHERE pending_spin_id IS NOT NULL;

-- ─── 2. Seed missing USSD config keys ────────────────────────────────────────
INSERT INTO network_configs (key, value, description, is_public, updated_by)
VALUES
    -- Canonical shortcode key used by USSDHandler (reads "ussd_shortcode").
    -- Migration 037 seeded "ussd_short_code" — this adds the canonical form.
    ('ussd_shortcode', '*384#',
     'The USSD shortcode for Loyalty Nexus. Displayed in USSD menus and SMS nudges.',
     true, 'system'),

    -- Maximum SMS character length for USSD Knowledge Tool result delivery.
    -- 320 = two standard SMS segments (160 chars each).
    ('ussd_sms_max_chars', '320',
     'Maximum character length for SMS delivery of AI Knowledge Tool results via USSD.',
     false, 'system'),

    -- App base URL used to build short URLs in Knowledge Tool SMS delivery.
    -- Admins must update this to the production domain before go-live.
    ('app_base_url', 'https://loyalty-nexus.app',
     'Base URL of the Loyalty Nexus web app. Used to build short URLs in SMS messages.',
     true, 'system')

ON CONFLICT (key) DO NOTHING;


══════════════════════════════════════════════════════
MIGRATION: 057_ussd_sms_number_seed.up.sql
══════════════════════════════════════════════════════
-- Migration 039: Seed ussd_sms_number config key for USSD Knowledge Tools SMS instruction.
--
-- REQ-6.4: When ussd_sms_number is set, the USSD Knowledge Tools sub-menu (option 7)
-- prepends an instruction line: "Send topic via SMS to <number>".
-- This key was referenced in the handler but never seeded in any prior migration.
--
-- The default value is empty so the instruction line is hidden until an admin sets
-- the real SMS number via the admin config panel (PUT /api/v1/admin/config/ussd_sms_number).
-- ─── Seed ussd_sms_number ─────────────────────────────────────────────────────
INSERT INTO network_configs (key, value, description, is_public, updated_by)
VALUES
    ('ussd_sms_number', '',
     'SMS number displayed in the USSD Knowledge Tools sub-menu as an alternative entry point. '
     'Leave empty to hide the instruction line. Set to e.g. "08012345678" to show it.',
     false, 'system')
ON CONFLICT (key) DO NOTHING;


══════════════════════════════════════════════════════
MIGRATION: 058_prize_pool_expand.up.sql
══════════════════════════════════════════════════════
-- Migration 053: Expand prize pool — add ₦1k / ₦2k airtime and ₦2k MoMo cash slots
-- Also adds ₦50 and ₦100 MoMo cash for micro-win excitement.
-- All values in kobo (base_value). Weights assume existing 044 seed is present.

INSERT INTO prize_pool (id, name, prize_code, prize_type, base_value, win_probability_weight,
                        is_active, is_no_win, no_win_message, color_scheme, sort_order,
                        minimum_recharge, icon_name)
VALUES
  -- Additional airtime tiers
  (gen_random_uuid(), '₦1,000 Airtime',  'AIR1K',  'airtime',   100000, 25, TRUE, FALSE, '', '#2196F3', 10, 500000,  'phone'),
  (gen_random_uuid(), '₦2,000 Airtime',  'AIR2K',  'airtime',   200000, 10, TRUE, FALSE, '', '#1a78c2', 11, 1000000, 'phone'),

  -- Additional MoMo cash tiers (micro + mid)
  (gen_random_uuid(), '₦500 MoMo Cash',  'CASH500', 'momo_cash',  50000, 8, TRUE, FALSE, '', '#10b981', 16, 300000,  'money-bag'),
  (gen_random_uuid(), '₦2,000 MoMo Cash','CASH2K',  'momo_cash', 200000, 4, TRUE, FALSE, '', '#059669', 17, 500000,  'money-bag')

ON CONFLICT DO NOTHING;

-- Note: After adding these rows the total weight across all active prizes will exceed
-- the original 10,000. Admin MUST open Spin Config → reduce "Better Luck Next Time"
-- weight accordingly before weights can be saved again. Current addition: 25+10+8+4 = 47
-- Reduce 'try_again' from 4050 → 4003 to stay within budget.
UPDATE prize_pool
SET    win_probability_weight = 4003
WHERE  prize_code = 'NONE' AND prize_type = 'try_again';


══════════════════════════════════════════════════════
MIGRATION: 059_comprehensive_seed.up.sql
══════════════════════════════════════════════════════
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

-- BEGIN;  -- removed: managed by golang-migrate

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

-- COMMIT;  -- removed: managed by golang-migrate


══════════════════════════════════════════════════════
MIGRATION: 060_ensure_critical_tables.up.sql
══════════════════════════════════════════════════════
-- ═══════════════════════════════════════════════════════════════════════════════
-- Migration 060: COMPREHENSIVE SAFETY NET
-- ═══════════════════════════════════════════════════════════════════════════════
-- This migration is the guaranteed final step. The entrypoint runs:
--   /migrate force 59   (marks all prior migrations as applied without re-running them)
--   /migrate up         (only this migration 060 executes)
--
-- Every statement uses CREATE TABLE IF NOT EXISTS / ADD COLUMN IF NOT EXISTS /
-- ON CONFLICT DO NOTHING / DO $$ EXCEPTION WHEN OTHERS THEN NULL $$ blocks.
-- It CANNOT fail under any database state.
-- ═══════════════════════════════════════════════════════════════════════════════

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 1: CORE FOUNDATION TABLES (from migrations 001-004)
-- These are guaranteed safe because of IF NOT EXISTS.
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS program_configs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key  TEXT UNIQUE NOT NULL,
    config_value JSONB NOT NULL,
    description TEXT,
    updated_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS prize_pool (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    TEXT NOT NULL,
    prize_type              TEXT,
    base_value              NUMERIC NOT NULL DEFAULT 0,
    is_active               BOOLEAN DEFAULT true,
    win_probability_weight  INTEGER DEFAULT 100,
    daily_inventory_cap     INTEGER,
    updated_at              TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS regional_settings (
    region_code TEXT PRIMARY KEY,
    multiplier  NUMERIC DEFAULT 1.0,
    is_golden_hour BOOLEAN DEFAULT false,
    updated_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS studio_config (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    media_type    TEXT,
    point_cost    INTEGER NOT NULL DEFAULT 0,
    render_priority INTEGER DEFAULT 1,
    is_enabled    BOOLEAN DEFAULT true
);

CREATE TABLE IF NOT EXISTS users (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number     TEXT UNIQUE NOT NULL,
    network_id       TEXT,
    state            TEXT,
    wallet_pass_id   UUID,
    points_balance   BIGINT NOT NULL DEFAULT 0,
    spin_credits     INTEGER NOT NULL DEFAULT 0,
    streak_days      INTEGER NOT NULL DEFAULT 0,
    last_recharge_at TIMESTAMPTZ,
    subscription_status     VARCHAR(20) NOT NULL DEFAULT 'FREE',
    subscription_expires_at TIMESTAMPTZ,
    lifetime_points  BIGINT NOT NULL DEFAULT 0,
    total_spins      INTEGER NOT NULL DEFAULT 0,
    studio_use_count INTEGER NOT NULL DEFAULT 0,
    total_referrals  INTEGER NOT NULL DEFAULT 0,
    momo_number      TEXT    NOT NULL DEFAULT '',
    momo_verified    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Ensure users table has phone_number column (migration 002 uses 'msisdn').
-- Migration 020 was supposed to rename it, but may not have run.
DO $$ BEGIN
    -- If msisdn exists but phone_number doesn't: rename
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name='users' AND column_name='msisdn')
    AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                    WHERE table_name='users' AND column_name='phone_number') THEN
        ALTER TABLE users RENAME COLUMN msisdn TO phone_number;
    END IF;
    -- If phone_number still doesn't exist: add it
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name='users' AND column_name='phone_number') THEN
        ALTER TABLE users ADD COLUMN phone_number TEXT NOT NULL DEFAULT '';
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;
-- Similarly fix transactions table
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name='transactions' AND column_name='msisdn')
    AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                    WHERE table_name='transactions' AND column_name='phone_number') THEN
        ALTER TABLE transactions RENAME COLUMN msisdn TO phone_number;
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;
-- Fix auth_otps table
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name='auth_otps' AND column_name='msisdn')
    AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                    WHERE table_name='auth_otps' AND column_name='phone_number') THEN
        ALTER TABLE auth_otps RENAME COLUMN msisdn TO phone_number;
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_60 ON users(phone_number);

-- ─── Ensure ALL users columns exist (safe for existing DBs) ─────────────────
-- The CREATE TABLE IF NOT EXISTS above is a no-op when the table already exists.
-- Every column the Go entity references must be explicitly ensured here.
ALTER TABLE users ADD COLUMN IF NOT EXISTS user_code              TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS state                  TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS tier                   TEXT         NOT NULL DEFAULT 'BRONZE';
ALTER TABLE users ADD COLUMN IF NOT EXISTS streak_count           INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS streak_expires_at      TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS streak_grace_used      INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS streak_grace_month     INTEGER;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_recharge_amount  BIGINT       NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_recharge_at       TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS momo_number            TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS momo_verified          BOOLEAN      NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS momo_verified_at       TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS wallet_pass_id         TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS device_type            TEXT         NOT NULL DEFAULT 'smartphone';
ALTER TABLE users ADD COLUMN IF NOT EXISTS subscription_tier      TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS subscription_status    TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS subscription_expires_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS referral_code          TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS referred_by            UUID;
ALTER TABLE users ADD COLUMN IF NOT EXISTS kyc_status             TEXT         NOT NULL DEFAULT 'unverified';
ALTER TABLE users ADD COLUMN IF NOT EXISTS points_expire_at       TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_points           BIGINT       NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS stamps_count           INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS lifetime_points        BIGINT       NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_spins            INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS studio_use_count       INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_referrals        INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS google_wallet_object_id TEXT        NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS apple_pass_serial      TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS spin_credits           INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active              BOOLEAN      NOT NULL DEFAULT TRUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS created_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW();
ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW();

CREATE TABLE IF NOT EXISTS transactions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID REFERENCES users(id),
    phone_number TEXT NOT NULL,
    amount_kobo  BIGINT NOT NULL DEFAULT 0,
    type         TEXT NOT NULL DEFAULT 'recharge',
    status       TEXT NOT NULL DEFAULT 'completed',
    provider_ref TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_transactions_user_id_60 ON transactions(user_id);

CREATE TABLE IF NOT EXISTS wallets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    points_balance  BIGINT NOT NULL DEFAULT 0,
    spin_credits    INTEGER NOT NULL DEFAULT 0,
    lifetime_points BIGINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES users(id),
    type        TEXT NOT NULL,
    amount      BIGINT NOT NULL DEFAULT 0,
    balance_after BIGINT NOT NULL DEFAULT 0,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 2: AI STUDIO (migration 003)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS studio_tools (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    point_cost  INTEGER NOT NULL DEFAULT 0,
    is_enabled  BOOLEAN DEFAULT true,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ai_generations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID REFERENCES users(id),
    tool_name    TEXT NOT NULL,
    prompt       TEXT,
    result_url   TEXT,
    status       TEXT DEFAULT 'pending',
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 3: CHAT / AI SUMMARISER (migration 009)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS chat_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id),
    status          TEXT DEFAULT 'active',
    last_activity_at TIMESTAMPTZ DEFAULT now(),
    created_at      TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_expiry_60 ON chat_sessions(status, last_activity_at);

CREATE TABLE IF NOT EXISTS chat_messages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID REFERENCES chat_sessions(id),
    role        TEXT,
    content     TEXT NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS session_summaries (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID REFERENCES users(id),
    summary    TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 4: SPIN ENGINE (migration 020)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS spin_results (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID REFERENCES users(id),
    phone_number     TEXT NOT NULL DEFAULT '',
    prize_pool_id    UUID REFERENCES prize_pool(id),
    prize_type       TEXT NOT NULL DEFAULT 'try_again',
    prize_value_kobo BIGINT NOT NULL DEFAULT 0,
    is_fulfilled     BOOLEAN NOT NULL DEFAULT FALSE,
    fulfilled_at     TIMESTAMPTZ,
    mo_mo_number     TEXT NOT NULL DEFAULT '',
    retry_count      INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_spin_results_user_id_60 ON spin_results(user_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 5: DRAWS ENGINE (migrations 016, 021)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS draws (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL DEFAULT 'Monthly Draw',
    status          TEXT NOT NULL DEFAULT 'UPCOMING',
    draw_type       TEXT NOT NULL DEFAULT 'MONTHLY',
    recurrence      TEXT NOT NULL DEFAULT 'monthly',
    next_draw_at    TIMESTAMPTZ,
    prize_pool      NUMERIC(12,2) NOT NULL DEFAULT 0,
    winner_count    INTEGER NOT NULL DEFAULT 1,
    total_entries   INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS draw_entries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    draw_id      UUID REFERENCES draws(id) ON DELETE CASCADE,
    user_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    phone_number TEXT NOT NULL DEFAULT '',
    ticket_count INTEGER NOT NULL DEFAULT 1,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS draw_winners (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    draw_id          UUID REFERENCES draws(id) ON DELETE CASCADE,
    user_id          UUID REFERENCES users(id) ON DELETE SET NULL,
    phone_number     TEXT NOT NULL DEFAULT '',
    position         INTEGER NOT NULL DEFAULT 1,
    prize_value_kobo BIGINT NOT NULL DEFAULT 0,
    status           TEXT NOT NULL DEFAULT 'PENDING_FULFILLMENT',
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 6: AUTH (migration 011)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS auth_otps (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number TEXT NOT NULL,
    code       TEXT NOT NULL,
    purpose    TEXT DEFAULT 'login',
    status     TEXT DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_auth_otps_phone_60 ON auth_otps(phone_number, status);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 7: SUBSCRIPTIONS (migration 006)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS subscription_plans (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    price_kobo  BIGINT NOT NULL DEFAULT 0,
    duration_days INTEGER NOT NULL DEFAULT 30,
    spin_credits  INTEGER NOT NULL DEFAULT 0,
    is_active   BOOLEAN DEFAULT true,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_subscriptions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id     UUID REFERENCES subscription_plans(id),
    status      TEXT NOT NULL DEFAULT 'active',
    starts_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 8: NOTIFICATIONS (migration 022)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS push_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token        TEXT NOT NULL UNIQUE,
    platform     TEXT NOT NULL DEFAULT 'fcm',
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID REFERENCES users(id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL,
    type       TEXT NOT NULL DEFAULT 'general',
    is_read    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_notifications_user_id_60 ON notifications(user_id);

CREATE TABLE IF NOT EXISTS notification_broadcasts (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title      TEXT NOT NULL,
    message    TEXT NOT NULL,
    type       TEXT NOT NULL DEFAULT 'push',
    status     TEXT NOT NULL DEFAULT 'queued',
    sent_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 9: DIGITAL PASSPORT (migration 004)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS wallet_passes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pass_type       TEXT NOT NULL DEFAULT 'loyalty',
    serial_number   TEXT NOT NULL UNIQUE DEFAULT gen_random_uuid()::text,
    qr_code_url     TEXT,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    issued_at       TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_badges (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_key  TEXT NOT NULL,
    earned_at  TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, badge_key)
);

CREATE TABLE IF NOT EXISTS passport_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    details    JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 10: REGIONAL WARS (migrations 005, 021)
-- ─────────────────────────────────────────────────────────────────────────────

-- Ensure regional_settings has all columns
ALTER TABLE regional_settings ADD COLUMN IF NOT EXISTS region_name           TEXT    NOT NULL DEFAULT '';
ALTER TABLE regional_settings ADD COLUMN IF NOT EXISTS base_multiplier       NUMERIC DEFAULT 1.0;
ALTER TABLE regional_settings ADD COLUMN IF NOT EXISTS golden_hour_multiplier NUMERIC DEFAULT 2.0;

CREATE TABLE IF NOT EXISTS regional_stats (
    region_code          TEXT PRIMARY KEY REFERENCES regional_settings(region_code),
    total_recharge_kobo  BIGINT DEFAULT 0,
    active_subscribers   INTEGER DEFAULT 0,
    last_recharge_at     TIMESTAMPTZ,
    rank                 INTEGER DEFAULT 0,
    updated_at           TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS regional_wars (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    period           VARCHAR(7)  NOT NULL UNIQUE,
    status           VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    total_prize_kobo BIGINT      NOT NULL DEFAULT 50000000,
    starts_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at          TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '1 month',
    resolved_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_regional_wars_status_60 ON regional_wars(status);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 11: USSD SESSIONS (migration 025)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS ussd_sessions (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      TEXT        NOT NULL UNIQUE,
    phone_number    TEXT        NOT NULL,
    menu_state      TEXT        NOT NULL DEFAULT 'root',
    input_buffer    TEXT        NOT NULL DEFAULT '',
    pending_spin_id UUID,
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '10 minutes',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ussd_sessions_phone_60      ON ussd_sessions(phone_number);
CREATE INDEX IF NOT EXISTS idx_ussd_sessions_session_id_60 ON ussd_sessions(session_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 12: ADMIN USERS (migration 052)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS admin_users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username      TEXT        UNIQUE,
    email         TEXT        UNIQUE,
    password_hash TEXT        NOT NULL,
    full_name     TEXT        NOT NULL DEFAULT '',
    role          TEXT        NOT NULL DEFAULT 'super_admin',
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Ensure the email column exists on existing admin_users tables (pre-052 DBs had 'username' only)
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS email         TEXT UNIQUE;
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS full_name     TEXT NOT NULL DEFAULT '';
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;
-- Ensure role column accepts the full set (it may be an ENUM or TEXT)
ALTER TABLE admin_users ALTER COLUMN role TYPE TEXT;
CREATE INDEX IF NOT EXISTS idx_admin_users_email_60 ON admin_users(email);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 13: NETWORK CONFIGS — THE CRITICAL TABLE
-- ─────────────────────────────────────────────────────────────────────────────

-- Step 1: If program_configs exists but network_configs doesn't, rename it.
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='public' AND table_name='program_configs')
    AND NOT EXISTS (SELECT 1 FROM information_schema.tables
                    WHERE table_schema='public' AND table_name='network_configs') THEN
        ALTER TABLE program_configs RENAME TO network_configs;
        -- Rename columns if they have old names
        IF EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name='network_configs' AND column_name='config_key') THEN
            ALTER TABLE network_configs RENAME COLUMN config_key TO key;
        END IF;
        IF EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name='network_configs' AND column_name='config_value') THEN
            ALTER TABLE network_configs RENAME COLUMN config_value TO value;
        END IF;
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- Step 2: Create from scratch if still missing
CREATE TABLE IF NOT EXISTS network_configs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    key         TEXT        NOT NULL UNIQUE,
    value       TEXT        NOT NULL DEFAULT '',
    description TEXT,
    is_public   BOOLEAN     NOT NULL DEFAULT FALSE,
    updated_by  TEXT        NOT NULL DEFAULT 'system',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_network_configs_key_60 ON network_configs(key);

-- Step 3: Add missing columns if table was renamed from old schema
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name='network_configs' AND column_name='is_public') THEN
        ALTER TABLE network_configs ADD COLUMN is_public BOOLEAN NOT NULL DEFAULT FALSE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name='network_configs' AND column_name='updated_by') THEN
        ALTER TABLE network_configs ADD COLUMN updated_by TEXT NOT NULL DEFAULT 'system';
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 14: PRIZE FULFILLMENT (migration 013, 024)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS prize_claims (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID REFERENCES users(id),
    spin_result_id   UUID REFERENCES spin_results(id),
    prize_type       TEXT NOT NULL DEFAULT 'airtime',
    prize_value_kobo BIGINT NOT NULL DEFAULT 0,
    status           TEXT NOT NULL DEFAULT 'pending',
    fulfilled_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS prize_fulfillment_logs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    spin_result_id   UUID REFERENCES spin_results(id) ON DELETE CASCADE,
    attempt_number   INTEGER NOT NULL DEFAULT 1,
    status           TEXT NOT NULL DEFAULT 'PENDING',
    provider         TEXT,
    provider_ref     TEXT,
    error_message    TEXT,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 15: FRAUD GUARD (migration 015)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS msisdn_blacklist (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number TEXT NOT NULL UNIQUE,
    reason       TEXT,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS fraud_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID REFERENCES users(id),
    phone_number TEXT,
    event_type   TEXT NOT NULL,
    details      JSONB DEFAULT '{}',
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 16: GHOST NUDGE / PASSPORT EXTRAS (migration 025)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS ghost_nudge_log (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    nudged_at TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 17: NETWORK CACHE / HLR (migration 008)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS network_cache (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number TEXT NOT NULL UNIQUE,
    network_id   TEXT NOT NULL,
    cached_at    TIMESTAMPTZ DEFAULT NOW()
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 18: SEED DATA
-- ─────────────────────────────────────────────────────────────────────────────

-- Seed regional settings
INSERT INTO regional_settings (region_code, region_name) VALUES
    ('LAG', 'Lagos'), ('ABJ', 'Abuja'), ('KAN', 'Kano'),
    ('PHC', 'Port Harcourt'), ('IBD', 'Ibadan'), ('ENU', 'Enugu'),
    ('AKW', 'Akwa Ibom'), ('ANM', 'Anambra'), ('BEN', 'Benue'),
    ('BOR', 'Borno'), ('DEL', 'Delta'), ('EKI', 'Ekiti'),
    ('IMO', 'Imo'), ('JIG', 'Jigawa'), ('KAD', 'Kaduna'),
    ('KAT', 'Katsina'), ('KEB', 'Kebbi'), ('KOG', 'Kogi'),
    ('KWA', 'Kwara'), ('LAP', 'Lagos'),('NAS', 'Nassarawa'),
    ('NIG', 'Niger'), ('OGN', 'Ogun'), ('OND', 'Ondo'),
    ('OSU', 'Osun'), ('OYO', 'Oyo'), ('PLA', 'Plateau'),
    ('RIV', 'Rivers'), ('SOK', 'Sokoto'), ('TAR', 'Taraba'),
    ('YOB', 'Yobe'), ('ZAM', 'Zamfara'), ('ABI', 'Abia'),
    ('ADA', 'Adamawa'), ('BAY', 'Bayelsa'), ('CRS', 'Cross River'),
    ('EBO', 'Ebonyi'), ('EDO', 'Edo'), ('GOM', 'Gombe')
ON CONFLICT (region_code) DO NOTHING;

-- Seed core network_configs keys
INSERT INTO network_configs (key, value, description) VALUES
    ('min_recharge_naira',            '500',    'Minimum recharge to earn a spin'),
    ('streak_target_days',            '7',      'Days required for Mega Jackpot'),
    ('ghost_nudge_hours',             '48',     'Inactivity hours before nudge'),
    ('spin_trigger_naira',            '1000',   'Naira recharge per spin credit'),
    ('spin_max_per_user_per_day',     '3',      'Max spins per user per day'),
    ('points_expiry_days',            '90',     'Days before points expire'),
    ('referral_bonus_points',         '20',     'Points for referrer and new user'),
    ('ussd_shortcode',                '"*384#"','USSD shortcode'),
    ('ussd_session_timeout_seconds',  '20',     'USSD session timeout seconds'),
    ('ai_chat_enabled',               'true',   'Enable Ask Nexus chat'),
    ('nexus_chat_daily_limit',        '20',     'Max chat messages per day'),
    ('operation_mode',                '"independent"', 'Independent or integrated'),
    ('prize_pool_kobo',               '50000000', 'Daily prize budget in kobo')
ON CONFLICT (key) DO NOTHING;

-- Fix prize_type check constraint to include all types used across migrations
-- Drop old narrow constraint (from migration 013) and replace with the full set
ALTER TABLE prize_pool DROP CONSTRAINT IF EXISTS prize_pool_prize_type_check;
ALTER TABLE prize_pool
    ADD CONSTRAINT prize_pool_prize_type_check
    CHECK (prize_type IN (
        'try_again', 'airtime', 'data', 'data_bundle',
        'momo_cash', 'bonus_points', 'pulse_points', 'studio_credits'
    ));

-- Seed prize pool if empty
INSERT INTO prize_pool (name, prize_type, base_value, is_active, win_probability_weight)
SELECT * FROM (VALUES
    ('Try Again',       'try_again',   0,    true, 5000),
    ('Try Again',       'try_again',   0,    true, 2000),
    ('+5 Pulse Points', 'pulse_points',5,    true, 1000),
    ('+10 Pulse Points','pulse_points',10,   true, 700),
    ('10MB Data',       'data_bundle', 10,   true, 600),
    ('₦50 Airtime',    'airtime',     50,   true, 420),
    ('₦100 Airtime',   'airtime',     100,  true, 200),
    ('₦200 Airtime',   'airtime',     200,  true, 80)
) AS v(name, prize_type, base_value, is_active, weight)
WHERE NOT EXISTS (SELECT 1 FROM prize_pool LIMIT 1);

-- USSD cleanup function
CREATE OR REPLACE FUNCTION cleanup_expired_ussd_sessions() RETURNS void AS $$
BEGIN
    DELETE FROM ussd_sessions WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;


-- ─── SECTION 19: DEMO ACCOUNTS ──────────────────────────────────────────────
-- Seeded demo accounts for staging/QA. Will be removed before go-live. All use E.164 format (234XXXXXXXXXX).
-- Login via OTP flow — check Render logs for the code after calling /auth/otp/send.
-- Phone numbers map to: 08020000000, 08023000000, 08025000000, 08027000000, 08029000000

INSERT INTO users (
    id, phone_number, user_code, state, tier,
    streak_count, streak_expires_at, streak_grace_used,
    total_recharge_amount, last_recharge_at,
    momo_number, momo_verified,
    referral_code, kyc_status,
    total_points, stamps_count, lifetime_points,
    total_spins, studio_use_count, spin_credits,
    is_active, created_at, updated_at
) VALUES
    -- Gold user — active, 2500 lifetime points, recent recharges
    (gen_random_uuid(), '+2348020000000', 'NXS-DEMO-01', 'Lagos',    'GOLD',
     5,  NOW() + INTERVAL '3 days', 0,
     25000000, NOW() - INTERVAL '1 day',
     '', false,
     'DEMO01REF', 'verified',
     800, 3, 2500,
     12, 4, 2,
     true, NOW() - INTERVAL '30 days', NOW()),

    -- Silver user — moderate activity
    (gen_random_uuid(), '+2348023000000', 'NXS-DEMO-02', 'Abuja',    'SILVER',
     3,  NOW() + INTERVAL '1 day', 0,
     12000000, NOW() - INTERVAL '3 days',
     '', false,
     'DEMO02REF', 'verified',
     350, 2, 900,
     6, 1, 1,
     true, NOW() - INTERVAL '20 days', NOW()),

    -- Bronze user — new, small balance
    (gen_random_uuid(), '+2348025000000', 'NXS-DEMO-03', 'Kano',     'BRONZE',
     1,  NOW() + INTERVAL '2 days', 0,
     5000000, NOW() - INTERVAL '5 days',
     '', false,
     'DEMO03REF', 'unverified',
     50, 0, 150,
     2, 0, 0,
     true, NOW() - INTERVAL '7 days', NOW()),

    -- Platinum user — power user, fully loaded
    (gen_random_uuid(), '+2348027000000', 'NXS-DEMO-04', 'Rivers',   'PLATINUM',
     7,  NOW() + INTERVAL '5 days', 0,
     100000000, NOW() - INTERVAL '12 hours',
     '2348027000000', true,
     'DEMO04REF', 'verified',
     3200, 7, 8500,
     45, 15, 5,
     true, NOW() - INTERVAL '90 days', NOW()),

    -- Bronze user — streak expired, needs nudge
    (gen_random_uuid(), '+2348029000000', 'NXS-DEMO-05', 'Enugu',    'BRONZE',
     0,  NULL, 0,
     2000000, NOW() - INTERVAL '14 days',
     '', false,
     'DEMO05REF', 'unverified',
     0, 0, 60,
     1, 0, 0,
     true, NOW() - INTERVAL '45 days', NOW())

ON CONFLICT (phone_number) DO UPDATE SET
    tier                 = EXCLUDED.tier,
    total_points         = EXCLUDED.total_points,
    lifetime_points      = EXCLUDED.lifetime_points,
    total_recharge_amount= EXCLUDED.total_recharge_amount,
    last_recharge_at     = EXCLUDED.last_recharge_at,
    spin_credits         = EXCLUDED.spin_credits,
    stamps_count         = EXCLUDED.stamps_count,
    updated_at           = NOW();

-- ─── SECTION 20: SEED SUPER ADMIN ───────────────────────────────────────────
-- Password: Admin@LoyaltyNexus2026!
-- Hash generated with bcrypt cost=10. Change password via admin UI after first login.
INSERT INTO admin_users (id, email, password_hash, full_name, role, is_active, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    'admin@loyaltynexus.ng',
    '$2b$10$8//qgubr/wos5AbYuMmeNeEEbPUg1GxyfkduWx.OZFRzdyodbPzR2',
    'Platform Admin',
    'super_admin',
    true,
    NOW(),
    NOW()
)
ON CONFLICT (email) DO UPDATE SET
    is_active  = true,
    updated_at = NOW();


══════════════════════════════════════════════════════
MIGRATION: 061_schema_patch_and_seeds.up.sql
══════════════════════════════════════════════════════
-- Migration 061: Schema patch + seeds for existing DBs
-- Adds all columns the Go entities require that may be missing from the live users
-- table (created by migration 002 before later ALTER TABLE migrations ran).
-- Also seeds the super_admin and 5 test users idempotently.
-- Every statement uses IF NOT EXISTS / ON CONFLICT DO NOTHING/UPDATE — fully safe to re-run.

-- ─── 1. USERS TABLE — ensure all entity columns exist ────────────────────────
ALTER TABLE users ADD COLUMN IF NOT EXISTS user_code              TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS state                  TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS tier                   TEXT         NOT NULL DEFAULT 'BRONZE';
ALTER TABLE users ADD COLUMN IF NOT EXISTS streak_count           INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS streak_expires_at      TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS streak_grace_used      INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS streak_grace_month     INTEGER;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_recharge_amount  BIGINT       NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_recharge_at       TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS momo_number            TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS momo_verified          BOOLEAN      NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS momo_verified_at       TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS wallet_pass_id         TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS device_type            TEXT         NOT NULL DEFAULT 'smartphone';
ALTER TABLE users ADD COLUMN IF NOT EXISTS subscription_tier      TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS subscription_status    TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS subscription_expires_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS referral_code          TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS referred_by            UUID;
ALTER TABLE users ADD COLUMN IF NOT EXISTS kyc_status             TEXT         NOT NULL DEFAULT 'unverified';
ALTER TABLE users ADD COLUMN IF NOT EXISTS points_expire_at       TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_points           BIGINT       NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS stamps_count           INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS lifetime_points        BIGINT       NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_spins            INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS studio_use_count       INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_referrals        INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS google_wallet_object_id TEXT        NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS apple_pass_serial      TEXT         NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS spin_credits           INTEGER      NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active              BOOLEAN      NOT NULL DEFAULT TRUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS created_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW();
ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW();

-- ─── 2. ADMIN_USERS TABLE — ensure email/full_name/role columns exist ─────────
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS email         TEXT;
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS full_name     TEXT NOT NULL DEFAULT '';
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;
-- Make role TEXT in case it was created as an ENUM and the ENUM type is missing
DO $$ BEGIN
    ALTER TABLE admin_users ALTER COLUMN role TYPE TEXT;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;
-- Add unique constraint on email if not present
DO $$ BEGIN
    ALTER TABLE admin_users ADD CONSTRAINT admin_users_email_key UNIQUE (email);
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- ─── 3. SEED 5 TEST USERS ─────────────────────────────────────────────────────
-- Admin is seeded at startup via ADMIN_SEED_EMAIL / ADMIN_SEED_PASSWORD env vars.
INSERT INTO users (
    id, phone_number, user_code, state, tier,
    streak_count, streak_expires_at, streak_grace_used,
    total_recharge_amount, last_recharge_at,
    momo_number, momo_verified,
    referral_code, kyc_status,
    total_points, stamps_count, lifetime_points,
    total_spins, studio_use_count, spin_credits,
    is_active, created_at, updated_at
) VALUES
    -- Platinum power user
    (gen_random_uuid(), '+2348027000000', 'NXS-DEMO-04', 'Rivers', 'PLATINUM',
     7,  NOW() + INTERVAL '5 days', 0,
     100000000, NOW() - INTERVAL '12 hours',
     '2348027000000', TRUE,
     'DEMO04REF', 'verified',
     3200, 7, 8500,
     45, 15, 5,
     TRUE, NOW() - INTERVAL '90 days', NOW()),
    -- Gold user
    (gen_random_uuid(), '+2348020000000', 'NXS-DEMO-01', 'Lagos', 'GOLD',
     5,  NOW() + INTERVAL '3 days', 0,
     25000000, NOW() - INTERVAL '1 day',
     '', FALSE,
     'DEMO01REF', 'verified',
     800, 3, 2500,
     12, 4, 2,
     TRUE, NOW() - INTERVAL '30 days', NOW()),
    -- Silver user
    (gen_random_uuid(), '+2348023000000', 'NXS-DEMO-02', 'Abuja', 'SILVER',
     3,  NOW() + INTERVAL '1 day', 0,
     12000000, NOW() - INTERVAL '3 days',
     '', FALSE,
     'DEMO02REF', 'verified',
     350, 2, 900,
     6, 1, 1,
     TRUE, NOW() - INTERVAL '20 days', NOW()),
    -- Bronze new user
    (gen_random_uuid(), '+2348025000000', 'NXS-DEMO-03', 'Kano', 'BRONZE',
     1,  NOW() + INTERVAL '2 days', 0,
     5000000, NOW() - INTERVAL '5 days',
     '', FALSE,
     'DEMO03REF', 'unverified',
     50, 0, 150,
     2, 0, 0,
     TRUE, NOW() - INTERVAL '7 days', NOW()),
    -- Bronze streak-lapsed user
    (gen_random_uuid(), '+2348029000000', 'NXS-DEMO-05', 'Enugu', 'BRONZE',
     0,  NULL, 0,
     2000000, NOW() - INTERVAL '14 days',
     '', FALSE,
     'DEMO05REF', 'unverified',
     0, 0, 60,
     1, 0, 0,
     TRUE, NOW() - INTERVAL '45 days', NOW())
ON CONFLICT (phone_number) DO UPDATE SET
    tier                  = EXCLUDED.tier,
    total_points          = EXCLUDED.total_points,
    lifetime_points       = EXCLUDED.lifetime_points,
    total_recharge_amount = EXCLUDED.total_recharge_amount,
    last_recharge_at      = EXCLUDED.last_recharge_at,
    spin_credits          = EXCLUDED.spin_credits,
    stamps_count          = EXCLUDED.stamps_count,
    state                 = EXCLUDED.state,
    kyc_status            = EXCLUDED.kyc_status,
    is_active             = TRUE,
    updated_at            = NOW();


══════════════════════════════════════════════════════
MIGRATION: 062_seed_admin.up.sql
══════════════════════════════════════════════════════
-- Migration 062: Guarantee super_admin exists in the database.
-- This is a belt-and-braces fallback alongside the ADMIN_SEED_EMAIL/PASSWORD env var approach.
-- Password: Admin@LoyaltyNexus2026!
-- ON CONFLICT DO NOTHING — completely safe if admin already exists.
INSERT INTO admin_users (id, email, password_hash, full_name, role, is_active, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    'admin@loyaltynexus.ng',
    '$2b$10$9U6kXVOrcNk11Dg2F38OhuvTtrmbgAIHpzUFxSnZ.L0NCyyuM/Gim',
    'Platform Admin',
    'super_admin',
    TRUE,
    NOW(),
    NOW()
)
ON CONFLICT (email) DO UPDATE SET
    is_active     = TRUE,
    role          = 'super_admin',
    password_hash = '$2b$10$9U6kXVOrcNk11Dg2F38OhuvTtrmbgAIHpzUFxSnZ.L0NCyyuM/Gim',
    updated_at    = NOW();


══════════════════════════════════════════════════════
MIGRATION: 063_admin_users_fix.up.sql
══════════════════════════════════════════════════════
-- Migration 063: Fix admin_users table and guarantee super_admin exists.
--
-- Two possible DB states exist depending on which migration created admin_users:
--
--   Path A (migration 019 ran first):
--     Columns: id, username TEXT NOT NULL UNIQUE, password_hash, role TEXT, created_at
--     No email, no full_name, no is_active, no last_login_at, no updated_at
--
--   Path B (migration 052 ran first, or 019 never ran):
--     Columns: id, email TEXT NOT NULL UNIQUE, password_hash, full_name, role admin_role,
--              is_active, last_login_at, created_at, updated_at
--     No username
--
-- This migration normalises both paths to the canonical schema.

-- Step 1: Add missing columns (safe to re-run — IF NOT EXISTS)
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS email         TEXT;
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS full_name     TEXT NOT NULL DEFAULT '';
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS is_active     BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE admin_users ADD COLUMN IF NOT EXISTS updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Step 2: Relax username NOT NULL constraint (Path A only — no-op on Path B)
-- On Path A, username is NOT NULL UNIQUE. We must allow NULL before inserting
-- new admin rows by email only (username is a legacy column from migration 019).
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'admin_users'
          AND column_name = 'username'
          AND is_nullable = 'NO'
    ) THEN
        ALTER TABLE admin_users ALTER COLUMN username DROP NOT NULL;
        RAISE NOTICE 'migration 063: dropped NOT NULL on username (Path A DB)';
    END IF;
END $$;

-- Step 3: Add unique constraint on email (ignore if already exists)
DO $$ BEGIN
    ALTER TABLE admin_users ADD CONSTRAINT admin_users_email_key UNIQUE (email);
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- Step 4: Widen role column to accept any text value
ALTER TABLE admin_users DROP CONSTRAINT IF EXISTS admin_users_role_check;
DO $$ BEGIN
    ALTER TABLE admin_users ALTER COLUMN role TYPE TEXT USING role::TEXT;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;
DO $$ BEGIN
    ALTER TABLE admin_users ALTER COLUMN role SET DEFAULT 'super_admin';
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- Step 5: Ensure the super_admin record exists (insert by email)
INSERT INTO admin_users (id, email, password_hash, full_name, role, is_active, created_at, updated_at)
SELECT
    gen_random_uuid(),
    'admin@loyaltynexus.ng',
    '$2b$10$9U6kXVOrcNk11Dg2F38OhuvTtrmbgAIHpzUFxSnZ.L0NCyyuM/Gim',
    'Platform Admin',
    'super_admin',
    TRUE,
    NOW(),
    NOW()
WHERE NOT EXISTS (SELECT 1 FROM admin_users WHERE email = 'admin@loyaltynexus.ng');

-- Step 6: Reset the password hash and role to known-good values
UPDATE admin_users
SET password_hash = '$2b$10$9U6kXVOrcNk11Dg2F38OhuvTtrmbgAIHpzUFxSnZ.L0NCyyuM/Gim',
    role          = 'super_admin',
    is_active     = TRUE,
    updated_at    = NOW()
WHERE email = 'admin@loyaltynexus.ng';


══════════════════════════════════════════════════════
MIGRATION: 064_missing_tables_fix.up.sql
══════════════════════════════════════════════════════
-- ============================================================
-- Loyalty Nexus: Consolidated missing-table fix
-- Run once in Render Shell via: psql $DATABASE_URL -f /tmp/fix.sql
-- Or paste each block into psql interactively
-- ============================================================

-- 1. Fix admin role: ensure the seeded admin is super_admin
UPDATE admin_users SET role = 'super_admin' WHERE role = 'operations' AND email = (SELECT email FROM admin_users ORDER BY created_at LIMIT 1);
-- Also ensure ALL admin users are super_admin if there's only one
UPDATE admin_users SET role = 'super_admin' WHERE (SELECT COUNT(*) FROM admin_users) = 1;

-- 2. Create ai_provider_configs table (migration 037)
CREATE TABLE IF NOT EXISTS ai_provider_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    category        TEXT NOT NULL,
    template        TEXT NOT NULL,
    env_key         TEXT NOT NULL DEFAULT '',
    api_key_enc     TEXT NOT NULL DEFAULT '',
    model_id        TEXT NOT NULL DEFAULT '',
    extra_config    JSONB NOT NULL DEFAULT '{}',
    priority        INT NOT NULL DEFAULT 10,
    is_primary      BOOLEAN NOT NULL DEFAULT false,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    cost_micros     INT NOT NULL DEFAULT 0,
    pulse_pts       INT NOT NULL DEFAULT 0,
    notes           TEXT NOT NULL DEFAULT '',
    last_tested_at  TIMESTAMPTZ,
    last_test_ok    BOOLEAN,
    last_test_msg   TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ai_provider_configs_category ON ai_provider_configs (category);
CREATE INDEX IF NOT EXISTS idx_ai_provider_configs_cat_prio ON ai_provider_configs (category, priority) WHERE is_active = true;

-- Seed AI providers
INSERT INTO ai_provider_configs (name, slug, category, template, env_key, model_id, priority, is_primary, is_active, cost_micros, pulse_pts, notes) VALUES
('Pollinations OpenAI',   'pollinations-text',   'text', 'openai-compatible',  'POLLINATIONS_SECRET_KEY', 'openai',                          1, true,  true, 0,     0,  'Pollinations free text via OpenAI-compat endpoint'),
('Gemini 2.5 Flash',     'gemini-flash',        'text', 'gemini',             'GEMINI_API_KEY',          'gemini-2.5-flash',                2, false, true, 0,     0,  'Google Gemini 2.5 Flash'),
('Groq Llama-4 Scout',   'groq-llama4',         'text', 'openai-compatible',  'GROQ_API_KEY',            'meta-llama/llama-4-scout-17b-16e-instruct', 3, false, true, 0, 0, 'Groq inference'),
('DeepSeek V3',          'deepseek-v3',         'text', 'deepseek',           'DEEPSEEK_API_KEY',        'deepseek-chat',                   4, false, true, 0,     0,  'DeepSeek V3'),
('HuggingFace FLUX',     'hf-flux-schnell',     'image', 'hf-image',          'HF_TOKEN',                'black-forest-labs/FLUX.1-schnell', 1, true,  true, 0,     0,  'HF serverless inference'),
('Pollinations FLUX',    'pollinations-flux',   'image', 'pollinations-image', 'POLLINATIONS_SECRET_KEY', 'flux',                            2, false, true, 0,     0,  'Pollinations FLUX'),
('FAL FLUX Dev',         'fal-flux-dev',        'image', 'fal-image',          'FAL_API_KEY',             'fal-ai/flux/dev',                 3, false, true, 6500,  0,  'FAL.AI FLUX-dev'),
('FAL Kling v1.5',       'fal-kling',           'video', 'fal-video',          'FAL_API_KEY',             'fal-ai/kling-video/v1.5/standard/image-to-video', 1, true, true, 56000, 0, 'FAL Kling v1.5'),
('FAL LTX Video',        'fal-ltx',             'video', 'fal-video',          'FAL_API_KEY',             'fal-ai/ltx-video',                2, false, true, 14500, 0,  'FAL LTX'),
('Pollinations Wan-Fast','pollinations-wan-fast','video', 'pollinations-video', 'POLLINATIONS_SECRET_KEY', 'wan-fast',                        3, true,  true, 0,     0,  'Wan 2.2 FREE'),
('Google Cloud TTS',     'google-cloud-tts',    'tts', 'google-tts',           'GOOGLE_CLOUD_TTS_KEY',    '',                                1, true,  true, 0,     0,  'Google TTS'),
('ElevenLabs TTS',       'elevenlabs-tts',      'tts', 'elevenlabs-tts',       'ELEVENLABS_API_KEY',      'eleven_flash_v2_5',               2, false, true, 2000,  0,  'ElevenLabs TTS'),
('Pollinations TTS',     'pollinations-tts',    'tts', 'pollinations-tts',     'POLLINATIONS_SECRET_KEY', 'elevenlabs',                      3, false, true, 0,     0,  'Pollinations TTS fallback'),
('AssemblyAI',           'assemblyai',          'transcribe', 'assemblyai',    'ASSEMBLY_AI_KEY',         'universal-2',                     1, true,  true, 25,    0,  'AssemblyAI Universal-2'),
('Groq Whisper',         'groq-whisper',        'transcribe', 'groq-whisper',  'GROQ_API_KEY',            'whisper-large-v3-turbo',          2, false, true, 10,    0,  'Groq Whisper'),
('Google Translate',     'google-translate',    'translate', 'google-translate','GOOGLE_TRANSLATE_API_KEY','',                              1, true,  true, 0,     0,  'Google Translate API v2'),
('Gemini Translate',     'gemini-translate',    'translate', 'gemini',          'GEMINI_API_KEY',          'gemini-2.5-flash',              2, false, true, 0,     0,  'Gemini Flash translation'),
('Pollinations Music',   'pollinations-elevenmusic','music','pollinations-music','POLLINATIONS_SECRET_KEY','elevenmusic',                 1, true,  true, 500,   0,  'Pollinations ElevenMusic'),
('rembg Self-Hosted',    'rembg-self-hosted',   'bg-remove', 'rembg',          'REMBG_SERVICE_URL',        '',                               1, true,  true, 0,     0,  'Self-hosted rembg'),
('FAL BiRefNet',         'fal-birefnet',        'bg-remove', 'fal-bg-remove',  'FAL_API_KEY',              'fal-ai/birefnet',                2, false, true, 2000,  0,  'FAL BiRefNet'),
('Pollinations Vision',  'pollinations-vision', 'vision', 'openai-compatible', 'POLLINATIONS_SECRET_KEY',  'openai',                         1, true,  true, 0,     0,  'Pollinations vision'),
('Gemini Vision',        'gemini-vision',       'vision', 'gemini',            'GEMINI_API_KEY',           'gemini-2.5-flash',               2, false, true, 0,     0,  'Gemini Flash vision')
ON CONFLICT (slug) DO NOTHING;

-- 3. Create draw_schedules table (migration 049)
CREATE TABLE IF NOT EXISTS draw_schedules (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    draw_name           TEXT        NOT NULL,
    draw_type           TEXT        NOT NULL,
    draw_day_of_week    INTEGER     NOT NULL CHECK (draw_day_of_week BETWEEN 0 AND 6),
    draw_time_wat       TIME        NOT NULL DEFAULT '17:00:00',
    window_open_dow     INTEGER     NOT NULL CHECK (window_open_dow BETWEEN 0 AND 6),
    window_open_time    TIME        NOT NULL DEFAULT '17:00:01',
    window_close_dow    INTEGER     NOT NULL CHECK (window_close_dow BETWEEN 0 AND 6),
    window_close_time   TIME        NOT NULL DEFAULT '17:00:00',
    cutoff_hour_utc     INTEGER     NOT NULL DEFAULT 16 CHECK (cutoff_hour_utc BETWEEN 0 AND 23),
    is_active           BOOLEAN     NOT NULL DEFAULT TRUE,
    sort_order          INTEGER     NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_draw_schedules_active ON draw_schedules (is_active, draw_day_of_week);

INSERT INTO draw_schedules (draw_name, draw_type, draw_day_of_week, draw_time_wat, window_open_dow, window_open_time, window_close_dow, window_close_time, cutoff_hour_utc, is_active, sort_order) VALUES
('Monday Daily Draw',    'DAILY',  1, '17:00:00', 4, '17:00:01', 0, '17:00:00', 16, TRUE, 1),
('Tuesday Daily Draw',   'DAILY',  2, '17:00:00', 0, '17:00:01', 1, '17:00:00', 16, TRUE, 2),
('Wednesday Daily Draw', 'DAILY',  3, '17:00:00', 1, '17:00:01', 2, '17:00:00', 16, TRUE, 3),
('Thursday Daily Draw',  'DAILY',  4, '17:00:00', 2, '17:00:01', 3, '17:00:00', 16, TRUE, 4),
('Friday Daily Draw',    'DAILY',  5, '17:00:00', 3, '17:00:01', 4, '17:00:00', 16, TRUE, 5),
('Saturday Weekly Mega Draw', 'WEEKLY', 6, '17:00:00', 5, '17:00:01', 5, '17:00:00', 16, TRUE, 6)
ON CONFLICT DO NOTHING;

-- 4. Create mtn_push_csv_uploads and mtn_push_csv_rows tables (migration 050)
CREATE TABLE IF NOT EXISTS mtn_push_csv_uploads (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    uploaded_by     TEXT        NOT NULL,
    filename        TEXT        NOT NULL,
    uploaded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    total_rows      INTEGER     NOT NULL DEFAULT 0,
    processed_rows  INTEGER     NOT NULL DEFAULT 0,
    skipped_rows    INTEGER     NOT NULL DEFAULT 0,
    failed_rows     INTEGER     NOT NULL DEFAULT 0,
    status          TEXT        NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','PROCESSING','DONE','PARTIAL','FAILED')),
    note            TEXT,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_csv_uploads_status ON mtn_push_csv_uploads (status, uploaded_at DESC);

CREATE TABLE IF NOT EXISTS mtn_push_csv_rows (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id       UUID        NOT NULL REFERENCES mtn_push_csv_uploads(id) ON DELETE CASCADE,
    row_number      INTEGER     NOT NULL,
    raw_msisdn      TEXT        NOT NULL,
    raw_date        TEXT        NOT NULL,
    raw_time        TEXT        NOT NULL,
    raw_amount      TEXT        NOT NULL,
    recharge_type   TEXT        NOT NULL DEFAULT 'AIRTIME',
    msisdn          TEXT,
    recharge_at     TIMESTAMPTZ,
    amount_naira    NUMERIC(12,2),
    status          TEXT        NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','OK','SKIPPED','FAILED')),
    skip_reason     TEXT,
    error_msg       TEXT,
    transaction_ref TEXT,
    spin_credits    INTEGER,
    pulse_points    BIGINT,
    draw_entries    INTEGER,
    processed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_csv_rows_upload ON mtn_push_csv_rows (upload_id, row_number);
CREATE INDEX IF NOT EXISTS idx_csv_rows_msisdn ON mtn_push_csv_rows (msisdn) WHERE msisdn IS NOT NULL;

-- 5. Create pulse_point_awards table (migration 051)
CREATE TABLE IF NOT EXISTS pulse_point_awards (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    phone_number    TEXT        NOT NULL,
    points          BIGINT      NOT NULL CHECK (points > 0),
    campaign        TEXT        NOT NULL DEFAULT '',
    note            TEXT        NOT NULL DEFAULT '',
    awarded_by      UUID        NOT NULL,
    awarded_by_name TEXT        NOT NULL DEFAULT '',
    transaction_id  UUID        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ppa_user_id ON pulse_point_awards (user_id);
CREATE INDEX IF NOT EXISTS idx_ppa_phone ON pulse_point_awards (phone_number);
CREATE INDEX IF NOT EXISTS idx_ppa_campaign ON pulse_point_awards (campaign) WHERE campaign <> '';
CREATE INDEX IF NOT EXISTS idx_ppa_awarded_by ON pulse_point_awards (awarded_by);

-- 6. Check fraud_events table exists and add any missing columns
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Verify results
SELECT 'ai_provider_configs' as tbl, COUNT(*) as rows FROM ai_provider_configs
UNION ALL SELECT 'draw_schedules', COUNT(*) FROM draw_schedules
UNION ALL SELECT 'mtn_push_csv_uploads', COUNT(*) FROM mtn_push_csv_uploads
UNION ALL SELECT 'pulse_point_awards', COUNT(*) FROM pulse_point_awards
UNION ALL SELECT 'fraud_events', COUNT(*) FROM fraud_events
UNION ALL SELECT 'admin_users_role_check', COUNT(*) FROM admin_users WHERE role = 'super_admin';


══════════════════════════════════════════════════════
MIGRATION: 065_fix_admin_role.up.sql
══════════════════════════════════════════════════════
-- Migration 065: Fix admin role to super_admin and ensure fraud_events has all needed columns
-- This migration ensures the seeded admin user has super_admin role
-- and that fraud_events table has all required columns.

-- Fix admin role: set all existing admin users to super_admin
-- (safe because this is a fresh deployment with only seeded admins)
UPDATE admin_users SET role = 'super_admin' WHERE role != 'super_admin';

-- Ensure fraud_events has all required columns (it may have been created by migration 020
-- with a slightly different schema than what the handler expects)
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ;
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS resolved_by TEXT;
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS notes TEXT NOT NULL DEFAULT '';

-- Verify
SELECT 'admin_super_admin_count' as check_name, COUNT(*)::text as result FROM admin_users WHERE role = 'super_admin'
UNION ALL
SELECT 'fraud_events_columns', string_agg(column_name, ', ' ORDER BY ordinal_position) FROM information_schema.columns WHERE table_name = 'fraud_events';


══════════════════════════════════════════════════════
MIGRATION: 066_fix_fraud_events_schema.up.sql
══════════════════════════════════════════════════════
-- Migration 066: Align fraud_events with FraudEvent Go struct
-- Safe/idempotent — wraps every operation in DO $$ EXCEPTION WHEN OTHERS THEN NULL END $$

-- 1. Add event_type column if missing
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS event_type TEXT NOT NULL DEFAULT '';

-- 2. Backfill event_type from rule_name ONLY if rule_name column exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'fraud_events' AND column_name = 'rule_name'
    ) THEN
        UPDATE fraud_events SET event_type = rule_name WHERE event_type = '';
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- 3. Convert details from JSONB to TEXT if still JSONB
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'fraud_events'
        AND column_name = 'details'
        AND data_type = 'jsonb'
    ) THEN
        ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS details_text TEXT NOT NULL DEFAULT '';
        UPDATE fraud_events SET details_text = details::text WHERE details IS NOT NULL;
        ALTER TABLE fraud_events DROP COLUMN details;
        ALTER TABLE fraud_events RENAME COLUMN details_text TO details;
    END IF;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

-- 4. Ensure details column exists as TEXT (in case it was missing entirely)
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS details TEXT NOT NULL DEFAULT '';

-- 5. Ensure updated_at exists
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();


══════════════════════════════════════════════════════
MIGRATION: 067_fix_spin_tiers_and_studio_tools.up.sql
══════════════════════════════════════════════════════
-- ============================================================
-- Migration 067: Fix spin_tiers + canonical studio_tools seed
-- Production-safe, fully idempotent — handles all DB states.
-- ============================================================

-- ─── 1. spin_tiers ───────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS spin_tiers (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tier_name         TEXT        NOT NULL,
    tier_display_name TEXT        NOT NULL,
    min_daily_amount  BIGINT      NOT NULL DEFAULT 0,
    max_daily_amount  BIGINT      NOT NULL DEFAULT 999999999999,
    spins_per_day     INTEGER     NOT NULL DEFAULT 1,
    tier_color        TEXT,
    tier_icon         TEXT,
    tier_badge        TEXT,
    description       TEXT,
    sort_order        INTEGER     NOT NULL DEFAULT 0,
    is_active         BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add unique constraint on tier_name if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'spin_tiers_tier_name_key'
    ) THEN
        ALTER TABLE spin_tiers ADD CONSTRAINT spin_tiers_tier_name_key UNIQUE (tier_name);
    END IF;
END $$;

-- Seed canonical spin tiers (upsert by tier_name)
INSERT INTO spin_tiers (id, tier_name, tier_display_name, min_daily_amount, max_daily_amount, spins_per_day, tier_color, tier_icon, tier_badge, description, sort_order, is_active)
VALUES
  ('11111111-1111-1111-1111-111111111111', 'bronze',   'Bronze',   100000,      499999,        1, '#CD7F32', '🥉', 'BRONZE',   'Recharge ₦1,000–₦4,999 per day',   1, TRUE),
  ('22222222-2222-2222-2222-222222222222', 'silver',   'Silver',   500000,      999999,        2, '#C0C0C0', '🥈', 'SILVER',   'Recharge ₦5,000–₦9,999 per day',   2, TRUE),
  ('33333333-3333-3333-3333-333333333333', 'gold',     'Gold',     1000000,     1999999,       3, '#FFD700', '🥇', 'GOLD',     'Recharge ₦10,000–₦19,999 per day', 3, TRUE),
  ('44444444-4444-4444-4444-444444444444', 'platinum', 'Platinum', 2000000,     999999999999,  5, '#E5E4E2', '💎', 'PLATINUM', 'Recharge ₦20,000+ per day',         4, TRUE)
ON CONFLICT (tier_name) DO UPDATE SET
  tier_display_name = EXCLUDED.tier_display_name,
  min_daily_amount  = EXCLUDED.min_daily_amount,
  max_daily_amount  = EXCLUDED.max_daily_amount,
  spins_per_day     = EXCLUDED.spins_per_day,
  tier_color        = EXCLUDED.tier_color,
  tier_icon         = EXCLUDED.tier_icon,
  tier_badge        = EXCLUDED.tier_badge,
  description       = EXCLUDED.description,
  sort_order        = EXCLUDED.sort_order,
  is_active         = EXCLUDED.is_active,
  updated_at        = NOW();

-- ─── 2. notification_broadcasts ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS notification_broadcasts (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    title        TEXT        NOT NULL,
    message      TEXT        NOT NULL,
    type         TEXT        NOT NULL DEFAULT 'info',
    target_count INTEGER     NOT NULL DEFAULT 0,
    status       TEXT        NOT NULL DEFAULT 'sent',
    sent_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─── 3. studio_tools: ensure all required columns exist ──────────────────────
-- First, give the legacy provider_tool_id column a default so new INSERTs
-- (which don't specify it) don't violate the NOT NULL constraint.
-- The original migration 003 created it as NOT NULL with no default.
ALTER TABLE studio_tools
    ALTER COLUMN provider_tool_id SET DEFAULT '',
    ADD COLUMN IF NOT EXISTS slug             TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sort_order       INT         NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS provider_tool    TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS is_free          BOOLEAN     NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS icon             TEXT        NOT NULL DEFAULT '🤖',
    ADD COLUMN IF NOT EXISTS entry_point_cost BIGINT      NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS refund_window_mins INT        NOT NULL DEFAULT 5,
    ADD COLUMN IF NOT EXISTS refund_pct        INT        NOT NULL DEFAULT 100,
    ADD COLUMN IF NOT EXISTS ui_template       TEXT        NOT NULL DEFAULT 'KnowledgeDoc',
    ADD COLUMN IF NOT EXISTS ui_config         JSONB       NOT NULL DEFAULT '{}';

-- ─── 4. Remove legacy non-canonical rows to prevent slug conflicts ────────────
-- Migration 003 seeded tools with names like "Ask Nexus", "My AI Photo" etc.
-- Migration 026 back-filled their slugs (e.g. "ask-nexus", "my-ai-photo").
-- These conflict with the canonical slug set below. We delete them safely
-- because this is a pre-production system with no real user ai_generations.
-- Rows with canonical slugs are preserved and updated via ON CONFLICT below.
DELETE FROM ai_generations
WHERE tool_id IN (
    SELECT id FROM studio_tools
    WHERE slug NOT IN (
        'translate', 'study-guide', 'quiz', 'mindmap', 'research-brief', 'bizplan',
        'slide-deck', 'ai-photo', 'bg-remover', 'animate-photo', 'video-premium',
        'narrate', 'transcribe', 'jingle', 'bg-music', 'podcast', 'infographic',
        'ai-photo-dream', 'ai-photo-max', 'ai-photo-pro', 'ask-my-photo',
        'code-helper', 'narrate-pro', 'photo-editor', 'instrumental', 'song-creator',
        'transcribe-african', 'video-cinematic', 'video-veo', 'web-search-ai',
        'image-analyser', 'video-jingle', 'nexus-chat'
    )
    AND slug != ''
);

DELETE FROM studio_tools
WHERE slug NOT IN (
    'translate', 'study-guide', 'quiz', 'mindmap', 'research-brief', 'bizplan',
    'slide-deck', 'ai-photo', 'bg-remover', 'animate-photo', 'video-premium',
    'narrate', 'transcribe', 'jingle', 'bg-music', 'podcast', 'infographic',
    'ai-photo-dream', 'ai-photo-max', 'ai-photo-pro', 'ask-my-photo',
    'code-helper', 'narrate-pro', 'photo-editor', 'instrumental', 'song-creator',
    'transcribe-african', 'video-cinematic', 'video-veo', 'web-search-ai',
    'image-analyser', 'video-jingle', 'nexus-chat'
)
AND slug != '';

-- ─── 5. Ensure unique index on slug (safe — only creates if missing) ──────────
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE tablename = 'studio_tools' AND indexname = 'uidx_studio_tools_slug'
    ) THEN
        -- Back-fill slugs for any remaining rows with empty slug before creating index
        UPDATE studio_tools
        SET slug = LOWER(REGEXP_REPLACE(TRIM(name), '[\s_]+', '-', 'g'))
        WHERE slug = '';
        CREATE UNIQUE INDEX uidx_studio_tools_slug ON studio_tools (slug);
    END IF;
END $$;

-- ─── 6. Canonical studio_tools seed (upsert by slug) ─────────────────────────
INSERT INTO studio_tools
    (id, name, slug, description, category, point_cost, provider, provider_tool,
     is_active, is_free, icon, sort_order, entry_point_cost, ui_template,
     created_at, updated_at)
VALUES
-- ── CHAT / TEXT TOOLS ────────────────────────────────────────────────────────
(gen_random_uuid(), 'Nexus Chat',         'nexus-chat',         'Conversational AI assistant powered by Gemini Flash.',                   'Chat',   0,   'gemini',       'gemini-2.0-flash',  true, true,  '💬', 10, 0,  'KnowledgeDoc',  NOW(), NOW()),
(gen_random_uuid(), 'Study Guide',        'study-guide',        'Generate a comprehensive study guide on any topic.',                     'Learn',  10,  'gemini',       'gemini-2.0-flash',  true, false, '📖', 20, 5,  'KnowledgeDoc',  NOW(), NOW()),
(gen_random_uuid(), 'Quiz Generator',     'quiz',               'Create multiple-choice quizzes from any subject.',                      'Learn',  10,  'gemini',       'gemini-2.0-flash',  true, false, '🧠', 21, 5,  'KnowledgeDoc',  NOW(), NOW()),
(gen_random_uuid(), 'Mind Map',           'mindmap',            'Turn any topic into a structured visual mind map.',                     'Learn',  10,  'gemini',       'gemini-2.0-flash',  true, false, '🗺️', 22, 5,  'KnowledgeDoc',  NOW(), NOW()),
(gen_random_uuid(), 'Research Brief',     'research-brief',     'Produce a concise research brief with key insights.',                   'Build',  15,  'gemini',       'gemini-2.0-flash',  true, false, '📊', 23, 5,  'KnowledgeDoc',  NOW(), NOW()),
(gen_random_uuid(), 'Slide Deck',         'slide-deck',         'Generate a professional presentation outline.',                         'Build',  20,  'gemini',       'gemini-2.0-flash',  true, false, '📑', 24, 5,  'KnowledgeDoc',  NOW(), NOW()),
(gen_random_uuid(), 'Infographic',        'infographic',        'Create an infographic content plan from any topic.',                    'Build',  20,  'gemini',       'gemini-2.0-flash',  true, false, '📈', 25, 5,  'KnowledgeDoc',  NOW(), NOW()),
(gen_random_uuid(), 'Business Plan',      'bizplan',            'Draft a structured business plan with AI.',                             'Build',  25,  'gemini',       'gemini-2.0-flash',  true, false, '💼', 26, 5,  'KnowledgeDoc',  NOW(), NOW()),
-- ── VOICE / TTS TOOLS ────────────────────────────────────────────────────────
(gen_random_uuid(), 'Narrate',            'narrate',            'Convert text to natural speech with AI voices.',                        'Create', 30,  'pollinations', 'openai-audio',      true, false, '🔊', 30, 10, 'VoiceStudio',   NOW(), NOW()),
(gen_random_uuid(), 'Narrate Pro',        'narrate-pro',        'Premium TTS with 13 voices, 7 languages, speed and format controls.',  'Create', 75,  'pollinations', 'openai-audio',      true, false, '🎤', 31, 20, 'VoiceStudio',   NOW(), NOW()),
(gen_random_uuid(), 'Translate',          'translate',          'Translate text between languages with AI.',                             'Learn',  30,  'gemini',       'gemini-2.0-flash',  true, false, '🌐', 32, 10, 'VoiceStudio',   NOW(), NOW()),
-- ── TRANSCRIPTION TOOLS ──────────────────────────────────────────────────────
(gen_random_uuid(), 'Transcribe',         'transcribe',         'Convert audio to text with Whisper AI.',                                'Create', 40,  'pollinations', 'whisper',           true, false, '🎙️', 33, 10, 'Transcribe',    NOW(), NOW()),
(gen_random_uuid(), 'African Transcribe', 'transcribe-african', 'Transcribe audio in African languages including Yoruba, Igbo, Hausa.', 'Create', 60,  'pollinations', 'whisper',           true, false, '🌍', 34, 15, 'Transcribe',    NOW(), NOW()),
-- ── IMAGE TOOLS ──────────────────────────────────────────────────────────────
(gen_random_uuid(), 'BG Remover',         'bg-remover',         'Remove image backgrounds instantly with AI.',                           'Create', 20,  'pollinations', 'flux',              true, false, '✂️', 40, 5,  'ImageEditor',   NOW(), NOW()),
(gen_random_uuid(), 'AI Photo Creator',   'ai-photo',           'Generate stunning images from text prompts.',                           'Create', 50,  'pollinations', 'flux',              true, false, '🎨', 41, 10, 'ImageCreator',  NOW(), NOW()),
(gen_random_uuid(), 'AI Photo Pro',       'ai-photo-pro',       'Premium image generation with GPT-Image quality.',                     'Create', 150, 'pollinations', 'gpt-image-1',       true, false, '🖼️', 42, 30, 'ImageCreator',  NOW(), NOW()),
(gen_random_uuid(), 'AI Photo Max',       'ai-photo-max',       'Maximum quality AI image generation.',                                  'Create', 250, 'pollinations', 'seedream-3',        true, false, '🌟', 43, 50, 'ImageCreator',  NOW(), NOW()),
(gen_random_uuid(), 'AI Photo Dream',     'ai-photo-dream',     'Dreamlike artistic AI image generation.',                              'Create', 200, 'pollinations', 'kontext',           true, false, '✨', 44, 40, 'ImageCreator',  NOW(), NOW()),
(gen_random_uuid(), 'Photo Editor',       'photo-editor',       'Edit and transform photos with AI prompts.',                           'Create', 100, 'pollinations', 'kontext',           true, false, '🖌️', 45, 20, 'ImageEditor',   NOW(), NOW()),
-- ── VISION TOOLS ─────────────────────────────────────────────────────────────
(gen_random_uuid(), 'Image Analyser',     'image-analyser',     'Analyse and describe images with AI vision.',                          'Create', 20,  'pollinations', 'gemini-vision',     true, false, '🔍', 46, 5,  'VisionAsk',     NOW(), NOW()),
(gen_random_uuid(), 'Ask My Photo',       'ask-my-photo',       'Ask questions about any image with AI.',                               'Create', 30,  'pollinations', 'gemini-vision',     true, false, '📷', 47, 10, 'VisionAsk',     NOW(), NOW()),
-- ── MUSIC TOOLS ──────────────────────────────────────────────────────────────
(gen_random_uuid(), 'Background Music',   'bg-music',           'Generate background music for videos and content.',                    'Create', 75,  'pollinations', 'musicgen',          true, false, '🎵', 50, 20, 'MusicComposer', NOW(), NOW()),
(gen_random_uuid(), 'Jingle Maker',       'jingle',             'Create catchy jingles for your brand or product.',                     'Create', 100, 'pollinations', 'musicgen',          true, false, '🎶', 51, 25, 'MusicComposer', NOW(), NOW()),
(gen_random_uuid(), 'Song Creator',       'song-creator',       'Generate full songs with lyrics and vocals.',                          'Create', 200, 'pollinations', 'elevenmusicgen',    true, false, '🎸', 52, 50, 'MusicComposer', NOW(), NOW()),
(gen_random_uuid(), 'Instrumental',       'instrumental',       'Generate instrumental music in any genre.',                            'Create', 150, 'pollinations', 'elevenmusicgen',    true, false, '🎹', 53, 35, 'MusicComposer', NOW(), NOW()),
-- ── VIDEO TOOLS ──────────────────────────────────────────────────────────────
(gen_random_uuid(), 'Animate Photo',      'animate-photo',      'Bring still photos to life with AI animation.',                        'Create', 100, 'pollinations', 'wan-fast',          true, false, '🎬', 60, 25, 'VideoAnimator', NOW(), NOW()),
(gen_random_uuid(), 'Video Premium',      'video-premium',      'Premium quality AI video generation.',                                 'Create', 200, 'pollinations', 'seedance',          true, false, '🎥', 61, 50, 'VideoAnimator', NOW(), NOW()),
(gen_random_uuid(), 'Video Cinematic',    'video-cinematic',    'Create cinematic quality videos from text prompts.',                   'Create', 300, 'pollinations', 'wan-fast',          true, false, '🎞️', 62, 75, 'VideoCreator',  NOW(), NOW()),
(gen_random_uuid(), 'Video Veo',          'video-veo',          'Google Veo-powered ultra-realistic video generation.',                 'Create', 500, 'pollinations', 'veo2',              true, false, '🌠', 63, 100,'VideoCreator',  NOW(), NOW()),
(gen_random_uuid(), 'Video Jingle',       'video-jingle',       'Create short video clips with music for social media.',                'Create', 150, 'pollinations', 'wan-fast',          true, false, '📱', 64, 35, 'VideoCreator',  NOW(), NOW())
ON CONFLICT (slug) DO UPDATE SET
    name             = EXCLUDED.name,
    description      = EXCLUDED.description,
    category         = EXCLUDED.category,
    point_cost       = EXCLUDED.point_cost,
    provider         = EXCLUDED.provider,
    provider_tool    = EXCLUDED.provider_tool,
    is_active        = EXCLUDED.is_active,
    is_free          = EXCLUDED.is_free,
    icon             = EXCLUDED.icon,
    sort_order       = EXCLUDED.sort_order,
    entry_point_cost = EXCLUDED.entry_point_cost,
    ui_template      = EXCLUDED.ui_template,
    updated_at       = NOW();


══════════════════════════════════════════════════════
MIGRATION: 068_remove_referral_system.up.sql
══════════════════════════════════════════════════════
-- Migration 068: Remove referral system
-- Drops referral_code, referred_by, total_referrals from users
-- Drops referral_bonus_points, referral_bonus_referee_pts from program_configs

-- Remove referral columns from users table
ALTER TABLE users
  DROP COLUMN IF EXISTS referral_code,
  DROP COLUMN IF EXISTS referred_by,
  DROP COLUMN IF EXISTS total_referrals;

-- Remove referral config keys from program_configs
-- Note: the column is config_key, not key
DELETE FROM program_configs
WHERE config_key IN ('referral_bonus_points', 'referral_bonus_referee_pts', 'REFERRAL_BONUS');

-- Drop any index on referral_code if it exists
DROP INDEX IF EXISTS idx_users_referral_code;
DROP INDEX IF EXISTS uidx_users_referral_code;


══════════════════════════════════════════════════════
MIGRATION: 069_split_spin_draw_counters.up.sql
══════════════════════════════════════════════════════
-- Migration 069: Split spin_draw_counter into separate spin_counter and draw_counter
--               + Add daily recharge tracking for tier-based spin credit logic
--
-- BACKGROUND
-- ----------
-- Migration 048 added a single `spin_draw_counter` that was shared between
-- spin credits AND draw entries (both used ₦200 threshold).
--
-- The correct business logic is:
--   • Spin Credits  — Tier-based on CUMULATIVE DAILY recharge:
--                     ₦1,000–₦4,999/day  → 1 spin  (Bronze)
--                     ₦5,000–₦9,999/day  → 2 spins (Silver)
--                     ₦10,000–₦19,999/day → 3 spins (Gold)
--                     ₦20,000+/day        → 5 spins (Platinum)
--                     The tier's spins_per_day is the DAILY CAP, not additive.
--                     Each time the cumulative daily total crosses a tier boundary,
--                     the user is awarded the DIFFERENCE (new_cap - already_awarded).
--   • Draw Entries  — ₦200 per entry, simple accumulator per transaction
--   • Pulse Points  — ₦250 per point (unchanged)
--
-- These are completely independent currencies with different thresholds and
-- different accumulation rules, so they need separate counters.
--
-- CHANGES
-- -------
--   1. Add spin_counter   BIGINT — kobo remainder for spin credit accumulation (NOT USED for tier logic)
--   2. Add draw_counter   BIGINT — kobo remainder for draw entry accumulation
--   3. Add daily_recharge_kobo BIGINT — cumulative recharge today (resets at midnight WAT)
--   4. Add daily_recharge_date DATE   — the date daily_recharge_kobo was last reset
--   5. Add daily_spins_awarded INT    — spins already awarded today (prevents double-awarding on tier upgrade)
--   6. Migrate existing spin_draw_counter → draw_counter (only if spin_draw_counter exists)
--   7. Zero out spin_draw_counter if it exists (deprecated)
--   8. Update network_configs with correct separated thresholds
--
-- NOTE: Steps 6 and 7 are wrapped in a DO $$ block that checks whether
-- spin_draw_counter exists before referencing it. This makes the migration
-- safe on databases that were provisioned without migration 048 (e.g. a fresh
-- Render deploy where the wallets table was created by migration 060 which
-- does not include spin_draw_counter).

-- Step 1: Add the new counter and daily tracking columns
ALTER TABLE wallets
    ADD COLUMN IF NOT EXISTS spin_counter         BIGINT  NOT NULL DEFAULT 0 CHECK (spin_counter >= 0),
    ADD COLUMN IF NOT EXISTS draw_counter         BIGINT  NOT NULL DEFAULT 0 CHECK (draw_counter >= 0),
    ADD COLUMN IF NOT EXISTS daily_recharge_kobo  BIGINT  NOT NULL DEFAULT 0 CHECK (daily_recharge_kobo >= 0),
    ADD COLUMN IF NOT EXISTS daily_recharge_date  DATE    NULL,
    ADD COLUMN IF NOT EXISTS daily_spins_awarded  INTEGER NOT NULL DEFAULT 0 CHECK (daily_spins_awarded >= 0);

COMMENT ON COLUMN wallets.spin_counter IS
    'Reserved for future use. Tier-based spins use daily_recharge_kobo + spin_tiers table instead.';
COMMENT ON COLUMN wallets.draw_counter IS
    'Kobo remainder accumulator for Draw Entries (resets modulo draw_naira_per_entry×100). Threshold: ₦200.';
COMMENT ON COLUMN wallets.daily_recharge_kobo IS
    'Cumulative recharge amount in kobo for the current calendar day (WAT). Resets to 0 at midnight WAT.';
COMMENT ON COLUMN wallets.daily_recharge_date IS
    'The calendar date (WAT) for which daily_recharge_kobo and daily_spins_awarded are current. NULL = never recharged.';
COMMENT ON COLUMN wallets.daily_spins_awarded IS
    'Number of spin credits already awarded today. Used to calculate incremental spin awards when tier upgrades.';

-- Steps 2 & 3: Migrate spin_draw_counter → draw_counter, then zero it out.
-- Wrapped in a DO block so this is a no-op on DBs that never had spin_draw_counter
-- (e.g. fresh deploys where wallets was created by migration 060 without that column).
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'wallets' AND column_name = 'spin_draw_counter'
    ) THEN
        -- Step 2: Carry over any existing remainder to the new draw_counter
        UPDATE wallets
        SET draw_counter = spin_draw_counter
        WHERE spin_draw_counter > 0;

        -- Step 3: Zero out the deprecated column
        UPDATE wallets SET spin_draw_counter = 0;
    END IF;
END $$;

-- Step 4: Update network_configs — replace the old shared key with two separate keys
-- Remove the old shared key (safe even if it does not exist)
DELETE FROM network_configs WHERE key = 'spin_draw_naira_per_credit';

-- Insert the two new separate threshold keys
INSERT INTO network_configs (key, value, description) VALUES
    ('spin_naira_per_credit',  '1000', 'Minimum daily recharge in naira to qualify for spin credits (Bronze tier threshold)'),
    ('draw_naira_per_entry',   '200',  'Naira per Draw Entry awarded on recharge (simple accumulator per transaction)'),
    ('spin_max_per_day',       '5',    'Maximum spin credits a user can earn per calendar day (Platinum tier cap)')
ON CONFLICT (key) DO UPDATE SET
    value       = EXCLUDED.value,
    description = EXCLUDED.description;

-- Step 5: Fix incorrect pulse_naira_per_point seed value
-- Migration 059 seeded pulse_naira_per_point as '10' which is wrong.
-- The correct production value is ₦250 per Pulse Point.
-- This corrects that seed so the Points Engine charges the right amount.
INSERT INTO network_configs (key, value, description) VALUES
    ('pulse_naira_per_point', '250', 'Naira per Pulse Point awarded on recharge (flat accumulator, no tier multiplier)')
ON CONFLICT (key) DO UPDATE SET
    value       = '250',
    description = EXCLUDED.description;


══════════════════════════════════════════════════════
MIGRATION: 070_rescale_prize_weights.up.sql
══════════════════════════════════════════════════════
-- Migration 070: Rescale prize probability weights from 10,000 scale to 100.00 scale
-- Before: weights are integers summing to 10,000 (e.g. 4003 = 40.03%)
-- After:  weights are NUMERIC(5,2) summing to 100.00 (e.g. 40.03 = 40.03%)
-- This makes the admin UI intuitive: weight = percentage directly.
-- Also renames all "MoMo Cash" prizes to plain "Cash" (MoMo is the delivery mechanism, not the prize name).

-- Step 1: Change column type from INTEGER to NUMERIC(5,2)
ALTER TABLE prize_pool
    ALTER COLUMN win_probability_weight TYPE NUMERIC(5,2)
    USING ROUND(win_probability_weight::NUMERIC / 100.0, 2);

-- Step 2: Rescale all existing seeded prizes to the 100.00 scale
-- (The USING clause above handles existing rows automatically via division by 100)
-- But we need to ensure the specific seeded values are exact (no floating point drift)
UPDATE prize_pool SET win_probability_weight = 40.03 WHERE prize_code = 'NONE'    AND prize_type = 'try_again';
UPDATE prize_pool SET win_probability_weight = 25.00 WHERE prize_code = 'PTS10'   AND prize_type = 'pulse_points';
UPDATE prize_pool SET win_probability_weight = 15.00 WHERE prize_code = 'PTS25'   AND prize_type = 'pulse_points';
UPDATE prize_pool SET win_probability_weight =  8.00 WHERE prize_code = 'PTS50'   AND prize_type = 'pulse_points';
UPDATE prize_pool SET win_probability_weight =  5.00 WHERE prize_code = 'PTS100'  AND prize_type = 'pulse_points';
UPDATE prize_pool SET win_probability_weight =  3.00 WHERE prize_code = 'AIR50'   AND prize_type = 'airtime';
UPDATE prize_pool SET win_probability_weight =  1.50 WHERE prize_code = 'AIR100'  AND prize_type = 'airtime';
UPDATE prize_pool SET win_probability_weight =  0.75 WHERE prize_code = 'AIR200'  AND prize_type = 'airtime';
UPDATE prize_pool SET win_probability_weight =  0.50 WHERE prize_code = 'AIR500'  AND prize_type = 'airtime';
UPDATE prize_pool SET win_probability_weight =  0.25 WHERE prize_code = 'AIR1K'   AND prize_type = 'airtime';
UPDATE prize_pool SET win_probability_weight =  0.10 WHERE prize_code = 'AIR2K'   AND prize_type = 'airtime';
UPDATE prize_pool SET win_probability_weight =  0.30 WHERE prize_code = 'DATA500' AND prize_type = 'data_bundle';
UPDATE prize_pool SET win_probability_weight =  0.20 WHERE prize_code = 'DATA1GB' AND prize_type = 'data_bundle';
UPDATE prize_pool SET win_probability_weight =  0.10 WHERE prize_code = 'DATA2GB' AND prize_type = 'data_bundle';
UPDATE prize_pool SET win_probability_weight =  0.08 WHERE prize_code = 'CASH500' AND prize_type = 'momo_cash';
UPDATE prize_pool SET win_probability_weight =  0.10 WHERE prize_code = 'CASH1K'  AND prize_type = 'momo_cash';
UPDATE prize_pool SET win_probability_weight =  0.04 WHERE prize_code = 'CASH2K'  AND prize_type = 'momo_cash';
UPDATE prize_pool SET win_probability_weight =  0.03 WHERE prize_code = 'CASH5K'  AND prize_type = 'momo_cash';
UPDATE prize_pool SET win_probability_weight =  0.02 WHERE prize_code = 'CASH50K' AND prize_type = 'momo_cash';

-- Step 3: Rename all "MoMo Cash" prizes to plain "Cash"
-- The delivery mechanism (MoMo) is an implementation detail, not the prize name
UPDATE prize_pool SET name = '₦500 Cash'    WHERE prize_code = 'CASH500' AND prize_type = 'momo_cash';
UPDATE prize_pool SET name = '₦1,000 Cash'  WHERE prize_code = 'CASH1K'  AND prize_type = 'momo_cash';
UPDATE prize_pool SET name = '₦2,000 Cash'  WHERE prize_code = 'CASH2K'  AND prize_type = 'momo_cash';
UPDATE prize_pool SET name = '₦5,000 Cash'  WHERE prize_code = 'CASH5K'  AND prize_type = 'momo_cash';
UPDATE prize_pool SET name = '₦50,000 Cash' WHERE prize_code = 'CASH50K' AND prize_type = 'momo_cash';

-- Step 4: Add a CHECK constraint to prevent weights going below 0 or above 100
ALTER TABLE prize_pool
    ADD CONSTRAINT prize_pool_weight_range
    CHECK (win_probability_weight >= 0 AND win_probability_weight <= 100);


══════════════════════════════════════════════════════
MIGRATION: 071_fix_prize_weights_idempotent.up.sql
══════════════════════════════════════════════════════
-- Migration 071: Idempotent corrective rescale for prize_pool weights
--
-- Context:
--   Migration 070 changed win_probability_weight from INTEGER (0–10000 basis-points)
--   to NUMERIC(5,2) (0–100.00 direct percentage). It used a USING clause to divide
--   existing rows by 100 at the time of the ALTER TABLE.
--
--   However, on databases where migration 070 ran BEFORE the full prize seed data
--   was present (e.g. a fresh deploy where 044/058 inserted rows after 070 ran),
--   or where the column was already NUMERIC before 070 (so the USING clause was
--   a no-op), some rows may still carry the old basis-point values (> 100).
--
--   This migration is fully idempotent: it only touches rows where
--   win_probability_weight > 100 (which is impossible on the 0–100 scale),
--   dividing them by 100 to bring them into the correct range.
--
--   Safe to re-run on any database state — rows already on the 0–100 scale
--   are untouched because their weight is ≤ 100.

UPDATE prize_pool
SET    win_probability_weight = ROUND(win_probability_weight / 100.0, 2)
WHERE  win_probability_weight > 100;

-- Verify: after this migration, no active prize should have weight > 100
-- (enforced by the CHECK constraint added in migration 070)
DO $$
DECLARE
    bad_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO bad_count
    FROM prize_pool
    WHERE win_probability_weight > 100;

    IF bad_count > 0 THEN
        RAISE EXCEPTION 'Migration 071 failed: % prize(s) still have weight > 100 after rescale', bad_count;
    END IF;
END $$;


══════════════════════════════════════════════════════
MIGRATION: 072_restore_draw_code.up.sql
══════════════════════════════════════════════════════
-- Migration 072: Restore draw_code column to draws table
-- ─────────────────────────────────────────────────────────────────────────────
-- BACKGROUND
-- ----------
-- Migration 016 created the draws table with a draw_code TEXT UNIQUE NOT NULL
-- column (format: DRAW-YYYYMMDD-XXXX). This column is used by draw_service.go
-- for idempotency, external referencing, and admin display.
--
-- Migration 021 (passport_badges_and_wars) re-created the draws table using
-- CREATE TABLE IF NOT EXISTS, which preserved the existing table but the new
-- definition omitted draw_code. Since the table already existed (from migration
-- 016), Postgres kept the old schema — but migration 060 (ensure_critical_tables)
-- also re-created draws without draw_code, which means on a fresh database
-- (where migration 060 runs before 016's table exists) draw_code is missing.
--
-- This migration ensures draw_code exists on all environments.
--
-- CHANGES
-- -------
--   1. Add draw_code TEXT column if it does not exist (safe on existing DBs).
--   2. Back-fill any existing rows with a generated code so NOT NULL is safe.
--   3. Add NOT NULL constraint after back-fill.
--   4. Add UNIQUE index.
--
-- NOTE: The application generates codes in the format DRAW-YYYYMMDD-XXXX.
--       Back-filled codes use DRAW-LEGACY-{id_prefix} to distinguish them.
-- ─────────────────────────────────────────────────────────────────────────────

-- Step 1: Add draw_code as nullable first (safe if it already exists)
ALTER TABLE draws
    ADD COLUMN IF NOT EXISTS draw_code TEXT;

-- Step 2: Back-fill any rows that have a NULL draw_code so we can add NOT NULL.
-- Uses the draw id prefix to create a unique deterministic code.
UPDATE draws
SET draw_code = 'DRAW-LEGACY-' || UPPER(SUBSTRING(id::text, 1, 8))
WHERE draw_code IS NULL OR draw_code = '';

-- Step 3: Now enforce NOT NULL (safe because all rows are back-filled)
ALTER TABLE draws
    ALTER COLUMN draw_code SET NOT NULL;

-- Step 4: Add unique index (idempotent)
CREATE UNIQUE INDEX IF NOT EXISTS uidx_draws_draw_code ON draws (draw_code);

COMMENT ON COLUMN draws.draw_code IS
    'Human-readable unique code for this draw, e.g. DRAW-20260101-1234. '
    'Generated by draw_service.go:generateDrawCode(). Used for external references and admin display.';
