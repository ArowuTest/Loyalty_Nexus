-- 011_auth_otp.sql
-- Purpose: Secure OTP management for phone-based authentication.

CREATE TABLE IF NOT EXISTS auth_otps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    msisdn TEXT NOT NULL,
    code TEXT NOT NULL,
    purpose TEXT CHECK (purpose IN ('login', 'momo_link', 'prize_claim')),
    status TEXT CHECK (status IN ('pending', 'verified', 'expired')) DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_auth_otps_msisdn ON auth_otps(msisdn, status);
