-- Migration 080: Add index to support OTP rate limiting
-- 
-- The SendOTP function now enforces a rate limit of 3 OTPs per phone
-- per 10 minutes. This index ensures the COUNT query is efficient.
CREATE INDEX IF NOT EXISTS idx_auth_otps_phone_created
    ON auth_otps (phone_number, created_at DESC);
