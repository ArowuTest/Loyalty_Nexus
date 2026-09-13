-- 010_regional_wars_admin.sql
-- Purpose: Admin views for managing Regional Wars.

-- The audit view joins users by state, but users.state was historically added later
-- in migration 019. Create it defensively here so clean installs can reach the view.
ALTER TABLE users ADD COLUMN IF NOT EXISTS state TEXT;

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
