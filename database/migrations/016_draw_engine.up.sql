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
