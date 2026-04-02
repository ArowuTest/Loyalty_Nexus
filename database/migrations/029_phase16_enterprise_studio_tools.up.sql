-- =============================================================================
-- 029_phase16_enterprise_studio_tools.sql
-- Phase 16: Enterprise Studio Tools Expansion
-- =============================================================================
--
-- WHAT THIS MIGRATION DOES:
--   1. Expands the studio_tools.category CHECK constraint to include 'Vision'
--      (a new category for image-analysis tools).
--   2. Upserts 14 new studio tools across the Chat, Vision, Build, and Create
--      categories — safe to re-run due to ON CONFLICT (slug) DO UPDATE.
--
-- NEW CATEGORY:
--   'Vision' — tools that accept image uploads and return AI-powered analysis.
--
-- NEW TOOLS SUMMARY:
--   FREE (0 pts) : web-search-ai, image-analyser, ask-my-photo, code-helper
--   LOW  (3 pts) : narrate-pro, transcribe-african
--   MID  (8-10)  : ai-photo-dream (8), ai-photo-pro (10), photo-editor (10)
--   HIGH (15-50) : ai-photo-max (15), instrumental (25), song-creator (30),
--                  video-cinematic (40), video-veo (50)
--
-- TOOLS NOT MODIFIED:
--   translate, study-guide, quiz, mindmap, research-brief, bizplan, slide-deck,
--   ai-photo, bg-remover, video-premium, narrate, transcribe, jingle, bg-music,
--   podcast, infographic
--
-- DEPENDENCY:  Requires 026_phase10_studio_hardening.sql (adds slug, sort_order,
--              provider_tool columns and the uidx_studio_tools_slug unique index).
-- =============================================================================

-- BEGIN;  -- removed: managed by golang-migrate

-- =============================================================================
-- STEP 1 — Expand the category CHECK constraint to allow 'Vision'
-- =============================================================================
-- The original CHECK in 003_nexus_studio.sql only covers: Chat, Create, Learn, Build.
-- We drop that constraint and recreate it with 'Vision' added.
-- The IF EXISTS guard makes this re-run safe.

ALTER TABLE studio_tools
    DROP CONSTRAINT IF EXISTS studio_tools_category_check;

ALTER TABLE studio_tools
    ADD CONSTRAINT studio_tools_category_check
        CHECK (category IN ('Chat', 'Create', 'Learn', 'Build', 'Vision'));

-- =============================================================================
-- STEP 2 — Upsert 14 new tools
-- =============================================================================
-- All rows use gen_random_uuid() for id so the INSERT always produces a valid
-- UUID on first run.  ON CONFLICT (slug) DO UPDATE ensures subsequent runs
-- update metadata without creating duplicates or resetting other columns
-- (e.g. is_active).  The id of an existing row is intentionally NOT overwritten.
-- =============================================================================

INSERT INTO studio_tools
    (id, name, slug, description, category, point_cost, provider, provider_tool,
     icon, sort_order, is_active, created_at, updated_at)
VALUES

-- ─────────────────────────────────────────────────────────────────────────────
-- FREE TOOLS  (point_cost = 0)
-- ─────────────────────────────────────────────────────────────────────────────

-- Chat ─────────────────────────────────────────────────────────────────────────
(gen_random_uuid(),
 'Web Search AI',      'web-search-ai',
 'Ask any question — get answers with live internet data',
 'Chat',    0, 'pollinations', 'gemini-search',
 '🔍', 18, true, NOW(), NOW()),

-- Vision ───────────────────────────────────────────────────────────────────────
(gen_random_uuid(),
 'Image Analyser',     'image-analyser',
 'Upload any photo — AI describes everything in detail',
 'Vision',  0, 'pollinations', 'openai-vision',
 '👁️', 19, true, NOW(), NOW()),

(gen_random_uuid(),
 'Ask My Photo',       'ask-my-photo',
 'Upload an image and ask any question about it',
 'Vision',  0, 'pollinations', 'openai-vision',
 '🤔', 20, true, NOW(), NOW()),

-- Build ────────────────────────────────────────────────────────────────────────
(gen_random_uuid(),
 'Code Helper',        'code-helper',
 'Write, explain, and debug code with AI',
 'Build',   0, 'pollinations', 'qwen-coder',
 '💻', 21, true, NOW(), NOW()),

-- ─────────────────────────────────────────────────────────────────────────────
-- LOW-COST TOOLS  (point_cost = 3)
-- ─────────────────────────────────────────────────────────────────────────────

-- Create ───────────────────────────────────────────────────────────────────────
(gen_random_uuid(),
 'Narrate Pro',        'narrate-pro',
 'Text to speech with 13 premium voice options',
 'Create',  3, 'pollinations', 'tts-1-voices',
 '🎙️', 22, true, NOW(), NOW()),

(gen_random_uuid(),
 'Transcribe African', 'transcribe-african',
 'Transcribe audio in Yoruba, Hausa, Igbo, English & French',
 'Create',  3, 'pollinations', 'whisper-african',
 '🌍', 23, true, NOW(), NOW()),

-- ─────────────────────────────────────────────────────────────────────────────
-- PAID TOOLS  (point_cost = 8 – 50)
-- ─────────────────────────────────────────────────────────────────────────────

-- Create — image generation tier ──────────────────────────────────────────────
(gen_random_uuid(),
 'AI Photo Dream',     'ai-photo-dream',
 'Creative & stylized AI images — Seedream by ByteDance',
 'Create',  8, 'pollinations', 'seedream',
 '🎨', 26, true, NOW(), NOW()),

(gen_random_uuid(),
 'AI Photo Pro',       'ai-photo-pro',
 'Photorealistic AI image generation — premium quality',
 'Create', 10, 'pollinations', 'gptimage',
 '✨', 24, true, NOW(), NOW()),

(gen_random_uuid(),
 'Photo Editor AI',    'photo-editor',
 'Edit any photo with text instructions — AI transforms it',
 'Create', 10, 'pollinations', 'kontext',
 '🖊️', 27, true, NOW(), NOW()),

(gen_random_uuid(),
 'AI Photo Max',       'ai-photo-max',
 'Highest quality AI image — GPT Image Large',
 'Create', 15, 'pollinations', 'gptimage-large',
 '🌟', 25, true, NOW(), NOW()),

-- Create — music generation ────────────────────────────────────────────────────
(gen_random_uuid(),
 'Instrumental Track', 'instrumental',
 'Generate AI background music — no vocals',
 'Create', 25, 'pollinations', 'elevenmusic-instrumental',
 '🎹', 29, true, NOW(), NOW()),

(gen_random_uuid(),
 'Song/Music Composer',  'song-creator',
 'Generate a full AI song with vocals — any genre',
 'Create', 30, 'pollinations', 'elevenmusic',
 '🎵', 28, true, NOW(), NOW()),

-- Create — video generation ────────────────────────────────────────────────────
(gen_random_uuid(),
 'Video Cinematic',    'video-cinematic',
 'Image to cinematic video — Seedance by ByteDance',
 'Create', 40, 'pollinations', 'seedance',
 '🎬', 30, true, NOW(), NOW()),

(gen_random_uuid(),
 'Video Veo',          'video-veo',
 'Text-to-video powered by Google Veo — highest quality',
 'Create', 50, 'pollinations', 'veo2',
 '🎦', 31, true, NOW(), NOW())

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

-- =============================================================================
-- VERIFICATION (informational — does not affect migration outcome)
-- =============================================================================
-- After applying, run:
--   SELECT category, COUNT(*) FROM studio_tools GROUP BY category ORDER BY category;
-- Expected new rows per category (Phase 16 additions only):
--   Build   +1  (code-helper)
--   Chat    +1  (web-search-ai)
--   Create  +10 (narrate-pro, transcribe-african, ai-photo-pro, ai-photo-max,
--                ai-photo-dream, photo-editor, song-creator, instrumental,
--                video-cinematic, video-veo)
--   Vision  +2  (image-analyser, ask-my-photo)
-- =============================================================================

-- COMMIT;  -- removed: managed by golang-migrate
