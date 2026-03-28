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
