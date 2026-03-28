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
