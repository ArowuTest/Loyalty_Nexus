-- Migration 097: Fix prize pool weights to sum to exactly 100%
-- Current active total is 93.5%. The small "Try Again" slot (1.5%) is bumped to 8%.
-- Also seed a proper daily draw schedule so ResolveQualifyingDraws works.

-- Fix weights: adjust the lowest-weight Try Again to fill the gap to 100%
UPDATE prize_pool
SET win_probability_weight = 8.0
WHERE name = 'Try Again'
  AND win_probability_weight = 1.5
  AND is_active = true;

-- Add colour schemes to slots that are missing them (for a visually consistent wheel)
UPDATE prize_pool SET color_scheme = '#1a1a2e' WHERE is_active = true AND (color_scheme IS NULL OR color_scheme = '') AND prize_type = 'try_again';
UPDATE prize_pool SET color_scheme = '#5f72f9' WHERE is_active = true AND (color_scheme IS NULL OR color_scheme = '') AND prize_type = 'pulse_points';
UPDATE prize_pool SET color_scheme = '#f97316' WHERE is_active = true AND (color_scheme IS NULL OR color_scheme = '') AND prize_type = 'airtime';
UPDATE prize_pool SET color_scheme = '#10b981' WHERE is_active = true AND (color_scheme IS NULL OR color_scheme = '') AND prize_type = 'data_bundle';
UPDATE prize_pool SET color_scheme = '#FFD700' WHERE is_active = true AND (color_scheme IS NULL OR color_scheme = '') AND prize_type = 'momo_cash';

-- Seed the daily draw schedule (DAILY draw, Mon–Sun, WAT 21:00)
-- This ensures ResolveQualifyingDraws returns the active draw on every recharge.
INSERT INTO draw_schedules (
  id, draw_name, draw_type,
  draw_day_of_week, draw_time_wat,
  window_open_dow, window_open_time,
  window_close_dow, window_close_time,
  cutoff_hour_utc, is_active, sort_order
) VALUES (
  gen_random_uuid(), 'Daily Draw', 'DAILY',
  -1, '21:00:00',
  -1, '00:00:00',
  -1, '20:59:59',
  20, true, 1
) ON CONFLICT DO NOTHING;

-- Create the first active daily draw record for April 2026 (if not already present)
INSERT INTO draws (
  id, name, draw_code, draw_type, status, prize_pool,
  winner_count, runner_ups_count, recurrence,
  start_time, end_time, draw_time, next_draw_at, created_at
) VALUES (
  gen_random_uuid(),
  'Daily Draw — April 2026', 'DRAW-DAILY-APR2026', 'DAILY', 'ACTIVE', 50000,
  1, 3, 'daily',
  NOW() - INTERVAL '1 hour',
  NOW() + INTERVAL '23 hours',
  NOW() + INTERVAL '9 hours',
  NOW() + INTERVAL '33 hours',
  NOW()
) ON CONFLICT (draw_code) DO NOTHING;
