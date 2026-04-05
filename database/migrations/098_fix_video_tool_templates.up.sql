-- Migration 098: Fix remaining video tool template assignments
-- 
-- animate-my-photo: was incorrectly set to 'KnowledgeDoc' in the live DB
--   (migration 096 was supposed to fix this but the live DB still shows KnowledgeDoc
--    because it was added after migration 096 ran — this migration corrects it)
--
-- my-video-story: was remapped to 'animate-photo' backend slug (single image animation)
--   but should use the new 'video-script' template which provides the full
--   script-driven multi-scene animation workflow (Pika/Kling story mode equivalent)
--   Backend dispatch: my-video-story → video-story path (multi-image + scene captions)

UPDATE studio_tools
SET ui_template = 'video-animator'
WHERE slug = 'animate-my-photo'
  AND ui_template != 'video-animator';

UPDATE studio_tools
SET ui_template = 'video-script'
WHERE slug = 'my-video-story'
  AND ui_template != 'video-script';

-- Also update the backend dispatch alias so my-video-story uses the video-story
-- multi-image path instead of the animate-photo single-image path.
-- This is handled in the backend code (ai_studio_service.go) — no DB change needed here.
-- The video-script template sends extra_params.image_urls which routes to the
-- video-story multi-image dispatch path in the backend.
