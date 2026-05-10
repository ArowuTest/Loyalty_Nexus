-- 106_platform_settings.up.sql
-- Admin-configurable platform settings stored as key/value pairs.
-- Settings are cached in Redis (5-min TTL) so reads are near-zero cost.

CREATE TABLE IF NOT EXISTS platform_settings (
    key         VARCHAR(120) PRIMARY KEY,
    value       TEXT         NOT NULL,
    label       VARCHAR(255) NOT NULL DEFAULT '',   -- human-readable name for admin UI
    description TEXT         NOT NULL DEFAULT '',   -- tooltip/help text
    category    VARCHAR(80)  NOT NULL DEFAULT 'general',
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by  VARCHAR(120) NOT NULL DEFAULT 'system'
);

-- ── Storage TTL defaults (hours) ─────────────────────────────────────────────
INSERT INTO platform_settings (key, value, label, description, category) VALUES
  ('storage_ttl_bronze_hours',   '48',  'Bronze Storage TTL (hours)',   'How long generated assets are kept for Bronze members',   'storage'),
  ('storage_ttl_silver_hours',   '48',  'Silver Storage TTL (hours)',   'How long generated assets are kept for Silver members',    'storage'),
  ('storage_ttl_gold_hours',     '72',  'Gold Storage TTL (hours)',     'How long generated assets are kept for Gold members',      'storage'),
  ('storage_ttl_platinum_hours', '168', 'Platinum Storage TTL (hours)', 'How long generated assets are kept for Platinum members (168 = 7 days)', 'storage'),
  ('storage_ttl_free_hours',     '24',  'Free Storage TTL (hours)',     'How long generated assets are kept for unauthenticated/free users', 'storage')
ON CONFLICT (key) DO NOTHING;

-- ── Pre-expiry notification windows (hours before expiry) ────────────────────
INSERT INTO platform_settings (key, value, label, description, category) VALUES
  ('notify_expiry_first_hours',  '24', 'First Expiry Warning (hours before)', 'Send first push/email this many hours before asset expires',  'storage'),
  ('notify_expiry_second_hours', '6',  'Second Expiry Warning (hours before)', 'Send second push/email this many hours before asset expires', 'storage')
ON CONFLICT (key) DO NOTHING;

-- ── Asset expiry notification tracking ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS asset_expiry_notifications (
    generation_id UUID        NOT NULL REFERENCES ai_generations(id) ON DELETE CASCADE,
    window        VARCHAR(20) NOT NULL,  -- 'first' | 'second'
    sent_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (generation_id, window)
);

-- ── Expiry cleanup tracking ───────────────────────────────────────────────────
-- Add column to ai_generations if it doesn't exist
DO $$ BEGIN
  ALTER TABLE ai_generations ADD COLUMN IF NOT EXISTS expired_cleaned_at TIMESTAMPTZ;
EXCEPTION WHEN OTHERS THEN NULL;
END $$;

CREATE INDEX IF NOT EXISTS idx_ai_gen_expires_at ON ai_generations(expires_at)
  WHERE output_url IS NOT NULL AND expired_cleaned_at IS NULL;
