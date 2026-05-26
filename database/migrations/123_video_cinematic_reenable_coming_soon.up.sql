-- Migration 123: Re-enable video-cinematic + add coming_soon column + mark bg-remover
--
-- video-cinematic was disabled in migration 113 because the FAL Kling template
-- required an image_url field that was missing. The tool has since been
-- rearchitected: Grok xAI (Tier 0) → Pollinations wan-fast (Tier 1, FREE,
-- 91.4% success) → Pollinations p-video (Tier 2, FREE, 100% success).
-- FAL is no longer in this provider chain — the image_url requirement is gone.
--
-- bg-remover has no REMBG endpoint configured. Rather than hiding it entirely,
-- mark it coming_soon so users can see it is planned.
--
-- All statements are idempotent.

-- Add coming_soon column (safe on re-run)
ALTER TABLE studio_tools
  ADD COLUMN IF NOT EXISTS coming_soon BOOLEAN NOT NULL DEFAULT FALSE;

-- Re-enable video-cinematic
UPDATE studio_tools
SET    is_active   = TRUE,
       coming_soon = FALSE,
       updated_at  = NOW()
WHERE  slug = 'video-cinematic';

-- Mark bg-remover as coming soon (keeps is_active=false so generation is blocked)
UPDATE studio_tools
SET    coming_soon = TRUE,
       is_active   = FALSE,
       updated_at  = NOW()
WHERE  slug = 'bg-remover';
