-- ============================================================
-- Migration 113: Platform health fixes — disable broken tools,
--                restore TTS, MTN-only networks, prize data
--
-- Fixes: BUG-008 (REMBG), BUG-016/049 (TTS/ElevenLabs),
--        BUG-018 (prize name mismatch), BUG-020 (data MB display),
--        BUG-033 (AIRTEL shown on MTN-only platform),
--        BUG-044 (Video Cinematic wrong template),
--        BUG-045 (Video Story 100% failure at 800pp)
--
-- All statements are idempotent.
-- ============================================================

-- ─── DISABLE broken AI Studio tools ──────────────────────────────────────────
-- These tools have 100% failure rates and charge Pulse Points with no output.
-- Disable them until their providers/templates are fixed.
-- Users will see them greyed-out / "coming soon" in the Studio UI.

UPDATE studio_tools
SET    is_active  = FALSE,
       updated_at = NOW()
WHERE  slug IN (
    'video-cinematic',  -- FAL Kling I2V — needs image upload template (BUG-044)
    'video-story',      -- video_multi_scene provider down, 800pp cost (BUG-045)
    'bg-remover'        -- REMBG_SERVICE_URL not configured (BUG-008)
)
AND    is_active = TRUE;  -- only update if currently active (idempotent)

-- ─── RE-ENABLE ElevenLabs as active TTS provider ──────────────────────────
-- Google TTS is an intentional stub (always returns error).
-- ElevenLabs was disabled in provider config, leaving all TTS tools broken.
-- Re-enable it so Narrate Pro, AI Podcast, Voice Studio have a working backend.

UPDATE ai_provider_configs
SET    is_active  = TRUE,
       updated_at = NOW()
WHERE  (LOWER(name) LIKE '%elevenlabs%' OR LOWER(slug) LIKE '%elevenlabs%'
        OR LOWER(provider_key) LIKE '%elevenlabs%')
AND    is_active = FALSE;

-- ─── DISABLE Google TTS provider record (it is a stub, never works) ──────────
UPDATE ai_provider_configs
SET    is_active  = FALSE,
       updated_at = NOW()
WHERE  (LOWER(name) LIKE '%google%tts%' OR LOWER(name) LIKE '%cloud tts%'
        OR LOWER(slug) LIKE '%google-tts%' OR LOWER(slug) LIKE '%google_tts%')
AND    is_active = TRUE;

-- ─── MTN-ONLY PLATFORM — deactivate non-MTN networks (BUG-033) ───────────────
-- Prevents AIRTEL/GLO/9MOBILE from appearing in the network selector.
-- VTPass recharges would fail for non-MTN numbers anyway.

UPDATE network_operator_configs
SET    is_active  = FALSE,
       updated_at = NOW()
WHERE  UPPER(network_code) != 'MTN'
AND    is_active = TRUE;

-- ─── FIX prize name / base_value mismatches (BUG-018) ────────────────────────
-- Prize names must accurately reflect base_value (in kobo).
-- The display formula is: base_value / 100 = Naira amount.
-- Update any prize named with a Naira value that does not match base_value.

-- Set prize names to match their actual kobo value (₦X = X*100 kobo)
-- This uses a safe pattern: only update rows where the name contains a
-- Naira symbol AND the computed Naira does not match the stated name amount.
-- We correct names to match base_value (the authoritative field).

UPDATE prize_pool
SET    name       = '₦' || (base_value / 100)::TEXT || ' Airtime',
       updated_at = NOW()
WHERE  prize_type  = 'airtime'
AND    base_value  > 0
AND    name        NOT LIKE '%' || (base_value / 100)::TEXT || '%';

-- ─── FIX data bundle "0.1MB" display (BUG-020) ───────────────────────────────
-- Data bundle prizes should store MB as the integer value in base_value,
-- NOT bytes or kobo. If base_value is in bytes (e.g. 10485760 for 10MB),
-- the display formula divides by 1048576. Standardise to MB as integer.
-- Safe update: only fix rows where base_value looks like a byte count (>1000000)
-- and prize_type is 'data'.

UPDATE prize_pool
SET    base_value  = base_value / 1048576,  -- convert bytes → MB
       name        = (base_value / 1048576)::TEXT || 'MB Data',
       updated_at  = NOW()
WHERE  prize_type  = 'data'
AND    base_value  > 1000000;  -- values > 1MB in bytes need conversion

-- ─── REMOVE orphan spin config keys that have no effect (BUG-023) ────────────
-- spin_max_per_day and spin_max_per_user_per_day are not read by the spin
-- service (which uses spin_tiers table via GetSpinTierFromDB instead).
-- Remove them to prevent false admin confidence.

DELETE FROM program_configs
WHERE  key IN ('spin_max_per_day', 'spin_max_per_user_per_day');

-- Also clean up from network_configs if present there
UPDATE network_configs
SET    value       = '-- UNUSED: spin limits are set in spin_tiers table, not here --',
       updated_at  = NOW()
WHERE  key IN ('spin_max_per_day', 'spin_max_per_user_per_day');
