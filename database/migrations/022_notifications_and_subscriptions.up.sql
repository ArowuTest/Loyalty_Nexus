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
