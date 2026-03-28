-- =============================================================================
-- 030_video_jingle_tool.sql
-- Phase 17: Add missing video-jingle composite tool to studio_tools
-- =============================================================================
--
-- WHAT THIS MIGRATION DOES:
--   Inserts the video-jingle tool that was implemented in the Go service
--   (ai_studio_service.go dispatchVideo) but was never seeded in the DB.
--
-- TOOL:
--   video-jingle (470 pts) — Full cinematic video + AI vocal song (Kling + ElevenMusic)
--   The most premium tool in the studio: FAL.AI Kling video + ElevenLabs/Pollinations
--   music combined into a single production-quality output.
--
-- PROVIDER CHAIN:
--   Primary  : FAL.AI Kling v1.5 Pro (video) + ElevenLabs Music (audio)
--   Fallback : Pollinations wan-fast (video) — audio portion always uses ElevenLabs
--
-- POINT COST RATIONALE:
--   FAL.AI Kling video  ≈ ₦320  (API cost)
--   ElevenLabs Music    ≈ ₦450  (API cost)
--   Total platform cost ≈ ₦770
--   470 pts × ₦7.50/pt = ₦3,525 revenue — ~78% margin
--
-- SAFE TO RE-RUN: ON CONFLICT (slug) DO UPDATE
-- =============================================================================

BEGIN;

INSERT INTO studio_tools
    (id, name, slug, description, category, point_cost, provider, provider_tool,
     icon, sort_order, is_active, created_at, updated_at)
VALUES
(gen_random_uuid(),
 'Video + Jingle',    'video-jingle',
 'Full AI production: cinematic video combined with a custom vocal song',
 'Create', 470, 'fal.ai+elevenlabs', 'kling-v1.5+elevenmusic',
 '🎬🎵', 32, true, NOW(), NOW())

ON CONFLICT (slug) DO UPDATE
    SET name          = EXCLUDED.name,
        description   = EXCLUDED.description,
        category      = EXCLUDED.category,
        point_cost    = EXCLUDED.point_cost,
        provider      = EXCLUDED.provider,
        provider_tool = EXCLUDED.provider_tool,
        icon          = EXCLUDED.icon,
        sort_order    = EXCLUDED.sort_order,
        updated_at    = NOW();

COMMIT;
