-- ════════════════════════════════════════════════════════════
-- Migration 035: Fix tool categories + ensure all tools have
--               correct ui_template assignments
-- ════════════════════════════════════════════════════════════

-- BEGIN;  -- removed: managed by golang-migrate

-- ── 1. Move chat-native tools to "Chat" category ────────────
UPDATE studio_tools SET category = 'Chat' WHERE slug IN ('web-search-ai', 'code-helper', 'ai-chat');

-- ── 2. Ensure web-search-ai is free (it is a chat feature) ──
UPDATE studio_tools SET is_free = true, point_cost = 0 WHERE slug = 'web-search-ai';

-- ── 3. Fix ui_template assignments ───────────────────────────

-- Image tools
UPDATE studio_tools SET ui_template = 'image_creator'
  WHERE slug IN ('ai-photo','ai-photo-pro','ai-photo-max','ai-photo-dream','infographic') AND ui_template IS NULL;

UPDATE studio_tools SET ui_template = 'image_editor'
  WHERE slug = 'photo-editor' AND ui_template IS NULL;

-- Video tools
UPDATE studio_tools SET ui_template = 'video_creator'
  WHERE slug IN ('video-premium','video-veo') AND ui_template IS NULL;

UPDATE studio_tools SET ui_template = 'video_animator'
  WHERE slug IN ('animate-photo','video-cinematic') AND ui_template IS NULL;

-- Audio tools
UPDATE studio_tools SET ui_template = 'voice_studio'
  WHERE slug IN ('narrate','narrate-pro','jingle','podcast') AND ui_template IS NULL;

UPDATE studio_tools SET ui_template = 'music_composer'
  WHERE slug IN ('bg-music','song-creator','instrumental') AND ui_template IS NULL;

UPDATE studio_tools SET ui_template = 'transcribe'
  WHERE slug IN ('transcribe','transcribe-african') AND ui_template IS NULL;

-- Vision tools
UPDATE studio_tools SET ui_template = 'vision_ask'
  WHERE slug IN ('image-analyser','ask-my-photo') AND ui_template IS NULL;

-- Knowledge tools
UPDATE studio_tools SET ui_template = 'knowledge_doc'
  WHERE slug IN (
    'translate','summarise','quiz','mindmap','slide-deck',
    'essay','email-writer','cv-writer'
  ) AND ui_template IS NULL;

-- ── 4. Ensure all rows have is_active set ─────────────────────
UPDATE studio_tools SET is_active = true WHERE is_active IS NULL;

-- ── 5. Refresh updated_at ─────────────────────────────────────
UPDATE studio_tools SET updated_at = NOW()
  WHERE slug IN (
    'web-search-ai','code-helper','ai-chat',
    'ai-photo','ai-photo-pro','ai-photo-max','ai-photo-dream','infographic',
    'photo-editor','video-premium','video-veo','animate-photo','video-cinematic',
    'narrate','narrate-pro','jingle','podcast','bg-music','song-creator','instrumental',
    'transcribe','transcribe-african','image-analyser','ask-my-photo',
    'translate','summarise','quiz','mindmap','slide-deck','essay','email-writer','cv-writer'
  );

-- COMMIT;  -- removed: managed by golang-migrate
