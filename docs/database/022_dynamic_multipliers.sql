-- 022_dynamic_multipliers.sql
-- Purpose: Support for scheduled and segment-specific multipliers (REQ-5.2.6, REQ-5.2.7).

CREATE TABLE IF NOT EXISTS scheduled_multipliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    multiplier NUMERIC NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    segment_type TEXT CHECK (segment_type IN ('global', 'state', 'inactive_users')) DEFAULT 'global',
    segment_value TEXT, -- e.g. state_code or null
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_multiplier_time ON scheduled_multipliers(start_time, end_time) WHERE is_active = true;
