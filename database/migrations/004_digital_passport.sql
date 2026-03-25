-- 004_digital_passport.sql
-- Purpose: Support for Apple and Google Wallet persistent lock-screen cards.

-- 1. Wallet Device Registrations (for APNS/Google Push)
CREATE TABLE wallet_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    platform TEXT CHECK (platform IN ('apple', 'google')) NOT NULL,
    device_id TEXT NOT NULL, -- Device Library Identifier
    push_token TEXT, -- Token for push notifications
    serial_number TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_wallet_registrations_user ON wallet_registrations(user_id);

-- 2. Digital Passport Pass Management
CREATE TABLE wallet_passes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    pass_type TEXT CHECK (pass_type IN ('loyalty', 'event', 'streak')),
    status TEXT DEFAULT 'active',
    last_pushed_at TIMESTAMPTZ,
    points_at_last_push BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT now()
);
