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
ALTER TABLE prizes
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
