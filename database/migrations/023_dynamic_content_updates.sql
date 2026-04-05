-- 023_dynamic_content_updates.sql
-- Purpose: Support for real-time dynamic content and admin-managed SMS templates (REQ-5.7.1).

-- 1. Dynamic SMS Templates
CREATE TABLE IF NOT EXISTS notification_templates (
    slug TEXT PRIMARY KEY, -- e.g. 'otp_delivery', 'prize_win', 'streak_expiry'
    content_template TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- Seed Initial Templates
INSERT INTO notification_templates (slug, content_template) VALUES
('otp_delivery', 'Your Loyalty Nexus login code is {{code}}. Valid for 10 minutes.'),
('prize_win', 'Congratulations! You won {{prize_name}} on the Loyalty Nexus Spin Wheel. Check your proﬁle to claim.'),
('streak_expiry', 'Your Loyalty Nexus recharge streak expires in 4 hours! Recharge now to save your {{streak}}-day progress.'),
('asset_completion', 'Your Loyalty Nexus {{tool_name}} is ready! Open the app gallery to view and download it.');

-- 2. Regional Wars Snapshots (REQ-5.3)
-- (Schema already handles real-time via sorted sets, this is for historical archival)
CREATE TABLE IF NOT EXISTS regional_wars_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cycle_start TIMESTAMPTZ NOT NULL,
    cycle_end TIMESTAMPTZ NOT NULL,
    winning_region_code TEXT NOT NULL,
    total_volume_kobo BIGINT,
    created_at TIMESTAMPTZ DEFAULT now()
);
