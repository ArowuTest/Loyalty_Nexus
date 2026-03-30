-- ============================================================
-- Migration 067: Fix spin_tiers table + re-seed studio_tools
--                with correct column names matching the entity
-- ============================================================

-- 1. Ensure spin_tiers table exists
CREATE TABLE IF NOT EXISTS spin_tiers (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tier_name         TEXT        NOT NULL,
    tier_display_name TEXT        NOT NULL,
    min_daily_amount  BIGINT      NOT NULL DEFAULT 0,
    max_daily_amount  BIGINT      NOT NULL DEFAULT 999999999999,
    spins_per_day     INTEGER     NOT NULL DEFAULT 1,
    tier_color        TEXT,
    tier_icon         TEXT,
    tier_badge        TEXT,
    description       TEXT,
    sort_order        INTEGER     NOT NULL DEFAULT 0,
    is_active         BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default spin tiers if empty
INSERT INTO spin_tiers (id, tier_name, tier_display_name, min_daily_amount, max_daily_amount, spins_per_day, tier_color, tier_icon, tier_badge, description, sort_order, is_active)
VALUES
  ('11111111-1111-1111-1111-111111111111', 'bronze',   'Bronze',   100000,      499999,        1, '#CD7F32', '🥉', 'BRONZE',   'Recharge ₦1,000–₦4,999 per day',   1, TRUE),
  ('22222222-2222-2222-2222-222222222222', 'silver',   'Silver',   500000,      999999,        2, '#C0C0C0', '🥈', 'SILVER',   'Recharge ₦5,000–₦9,999 per day',   2, TRUE),
  ('33333333-3333-3333-3333-333333333333', 'gold',     'Gold',     1000000,     1999999,       3, '#FFD700', '🥇', 'GOLD',     'Recharge ₦10,000–₦19,999 per day', 3, TRUE),
  ('44444444-4444-4444-4444-444444444444', 'platinum', 'Platinum', 2000000,     999999999999,  5, '#E5E4E2', '💎', 'PLATINUM', 'Recharge ₦20,000+ per day',         4, TRUE)
ON CONFLICT (id) DO UPDATE SET
  tier_display_name = EXCLUDED.tier_display_name,
  min_daily_amount  = EXCLUDED.min_daily_amount,
  max_daily_amount  = EXCLUDED.max_daily_amount,
  spins_per_day     = EXCLUDED.spins_per_day,
  tier_color        = EXCLUDED.tier_color,
  tier_icon         = EXCLUDED.tier_icon,
  tier_badge        = EXCLUDED.tier_badge,
  description       = EXCLUDED.description,
  sort_order        = EXCLUDED.sort_order,
  is_active         = EXCLUDED.is_active,
  updated_at        = NOW();

-- 2. Ensure notification_broadcasts table exists
CREATE TABLE IF NOT EXISTS notification_broadcasts (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    title        TEXT        NOT NULL,
    message      TEXT        NOT NULL,
    type         TEXT        NOT NULL DEFAULT 'info',
    target_count INTEGER     NOT NULL DEFAULT 0,
    status       TEXT        NOT NULL DEFAULT 'sent',
    sent_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Re-seed studio_tools using the correct column names from the entity:
--    id, name, slug, description, category, point_cost, provider,
--    provider_tool, is_active, is_free, icon, sort_order,
--    entry_point_cost, ui_template, created_at, updated_at
INSERT INTO studio_tools
    (id, name, slug, description, category, point_cost, provider, provider_tool,
     is_active, is_free, icon, sort_order, entry_point_cost, ui_template,
     created_at, updated_at)
VALUES
-- ── CHAT / TEXT TOOLS ────────────────────────────────────────────────────────
(gen_random_uuid(), 'Nexus Chat',         'nexus-chat',         'Conversational AI assistant powered by Gemini Flash.',                    'Chat',   0,   'gemini',       'gemini-2.0-flash',  true, true,  '💬', 10, 0,  'KnowledgeDoc',  NOW(), NOW()),
(gen_random_uuid(), 'Study Guide',        'study-guide',        'Generate a comprehensive study guide on any topic.',                      'Learn',  10,  'gemini',       'gemini-2.0-flash',  true, false, '📖', 20, 5,  'KnowledgeDoc',  NOW(), NOW()),
(gen_random_uuid(), 'Quiz Generator',     'quiz',               'Create multiple-choice quizzes from any subject.',                       'Learn',  10,  'gemini',       'gemini-2.0-flash',  true, false, '🧠', 21, 5,  'KnowledgeDoc',  NOW(), NOW()),
(gen_random_uuid(), 'Mind Map',           'mindmap',            'Turn any topic into a structured visual mind map.',                      'Learn',  10,  'gemini',       'gemini-2.0-flash',  true, false, '🗺️', 22, 5,  'KnowledgeDoc',  NOW(), NOW()),
(gen_random_uuid(), 'Research Brief',     'research-brief',     'Produce a concise research brief with key insights.',                    'Build',  15,  'gemini',       'gemini-2.0-flash',  true, false, '📊', 23, 5,  'KnowledgeDoc',  NOW(), NOW()),
(gen_random_uuid(), 'Slide Deck',         'slide-deck',         'Generate a professional presentation outline.',                          'Build',  20,  'gemini',       'gemini-2.0-flash',  true, false, '📑', 24, 5,  'KnowledgeDoc',  NOW(), NOW()),
(gen_random_uuid(), 'Infographic',        'infographic',        'Create an infographic content plan from any topic.',                     'Build',  20,  'gemini',       'gemini-2.0-flash',  true, false, '📈', 25, 5,  'KnowledgeDoc',  NOW(), NOW()),
(gen_random_uuid(), 'Business Plan',      'bizplan',            'Draft a structured business plan with AI.',                              'Build',  25,  'gemini',       'gemini-2.0-flash',  true, false, '💼', 26, 5,  'KnowledgeDoc',  NOW(), NOW()),
-- ── VOICE / TTS TOOLS ────────────────────────────────────────────────────────
(gen_random_uuid(), 'Narrate',            'narrate',            'Convert text to natural speech with AI voices.',                         'Create', 30,  'pollinations', 'openai-audio',      true, false, '🔊', 30, 10, 'VoiceStudio',   NOW(), NOW()),
(gen_random_uuid(), 'Narrate Pro',        'narrate-pro',        'Premium TTS with 13 voices, 7 languages, speed and format controls.',   'Create', 75,  'pollinations', 'openai-audio',      true, false, '🎤', 31, 20, 'VoiceStudio',   NOW(), NOW()),
(gen_random_uuid(), 'Translate',          'translate',          'Translate text between languages with AI.',                              'Learn',  30,  'gemini',       'gemini-2.0-flash',  true, false, '🌐', 32, 10, 'VoiceStudio',   NOW(), NOW()),
-- ── TRANSCRIPTION TOOLS ──────────────────────────────────────────────────────
(gen_random_uuid(), 'Transcribe',         'transcribe',         'Convert audio to text with Whisper AI.',                                 'Create', 40,  'pollinations', 'whisper',           true, false, '🎙️', 33, 10, 'Transcribe',    NOW(), NOW()),
(gen_random_uuid(), 'African Transcribe', 'transcribe-african', 'Transcribe audio in African languages including Yoruba, Igbo, Hausa.',  'Create', 60,  'pollinations', 'whisper',           true, false, '🌍', 34, 15, 'Transcribe',    NOW(), NOW()),
-- ── IMAGE TOOLS ──────────────────────────────────────────────────────────────
(gen_random_uuid(), 'BG Remover',         'bg-remover',         'Remove image backgrounds instantly with AI.',                            'Create', 20,  'pollinations', 'flux',              true, false, '✂️', 40, 5,  'ImageEditor',   NOW(), NOW()),
(gen_random_uuid(), 'AI Photo Creator',   'ai-photo',           'Generate stunning images from text prompts.',                            'Create', 50,  'pollinations', 'flux',              true, false, '🎨', 41, 10, 'ImageCreator',  NOW(), NOW()),
(gen_random_uuid(), 'AI Photo Pro',       'ai-photo-pro',       'Premium image generation with GPT-Image quality.',                      'Create', 150, 'pollinations', 'gpt-image-1',       true, false, '🖼️', 42, 30, 'ImageCreator',  NOW(), NOW()),
(gen_random_uuid(), 'AI Photo Max',       'ai-photo-max',       'Maximum quality AI image generation.',                                  'Create', 250, 'pollinations', 'seedream-3',        true, false, '🌟', 43, 50, 'ImageCreator',  NOW(), NOW()),
(gen_random_uuid(), 'AI Photo Dream',     'ai-photo-dream',     'Dreamlike artistic AI image generation.',                               'Create', 200, 'pollinations', 'kontext',           true, false, '✨', 44, 40, 'ImageCreator',  NOW(), NOW()),
(gen_random_uuid(), 'Photo Editor',       'photo-editor',       'Edit and transform photos with AI prompts.',                            'Create', 100, 'pollinations', 'kontext',           true, false, '🖌️', 45, 20, 'ImageEditor',   NOW(), NOW()),
-- ── VISION TOOLS ─────────────────────────────────────────────────────────────
(gen_random_uuid(), 'Image Analyser',     'image-analyser',     'Analyse and describe images with AI vision.',                           'Create', 20,  'pollinations', 'gemini-vision',     true, false, '🔍', 46, 5,  'VisionAsk',     NOW(), NOW()),
(gen_random_uuid(), 'Ask My Photo',       'ask-my-photo',       'Ask questions about any image with AI.',                                'Create', 30,  'pollinations', 'gemini-vision',     true, false, '📷', 47, 10, 'VisionAsk',     NOW(), NOW()),
-- ── MUSIC TOOLS ──────────────────────────────────────────────────────────────
(gen_random_uuid(), 'Background Music',   'bg-music',           'Generate background music for videos and content.',                     'Create', 75,  'pollinations', 'musicgen',          true, false, '🎵', 50, 20, 'MusicComposer', NOW(), NOW()),
(gen_random_uuid(), 'Jingle Maker',       'jingle',             'Create catchy jingles for your brand or product.',                      'Create', 100, 'pollinations', 'musicgen',          true, false, '🎶', 51, 25, 'MusicComposer', NOW(), NOW()),
(gen_random_uuid(), 'Song Creator',       'song-creator',       'Generate full songs with lyrics and vocals.',                           'Create', 200, 'pollinations', 'elevenmusicgen',    true, false, '🎸', 52, 50, 'MusicComposer', NOW(), NOW()),
(gen_random_uuid(), 'Instrumental',       'instrumental',       'Generate instrumental music in any genre.',                             'Create', 150, 'pollinations', 'elevenmusicgen',    true, false, '🎹', 53, 35, 'MusicComposer', NOW(), NOW()),
-- ── VIDEO TOOLS ──────────────────────────────────────────────────────────────
(gen_random_uuid(), 'Animate Photo',      'animate-photo',      'Bring still photos to life with AI animation.',                         'Create', 100, 'pollinations', 'wan-fast',          true, false, '🎬', 60, 25, 'VideoAnimator', NOW(), NOW()),
(gen_random_uuid(), 'Video Premium',      'video-premium',      'Premium quality AI video generation.',                                  'Create', 200, 'pollinations', 'seedance',          true, false, '🎥', 61, 50, 'VideoAnimator', NOW(), NOW()),
(gen_random_uuid(), 'Video Cinematic',    'video-cinematic',    'Create cinematic quality videos from text prompts.',                    'Create', 300, 'pollinations', 'wan-fast',          true, false, '🎞️', 62, 75, 'VideoCreator',  NOW(), NOW()),
(gen_random_uuid(), 'Video Veo',          'video-veo',          'Google Veo-powered ultra-realistic video generation.',                  'Create', 500, 'pollinations', 'veo2',              true, false, '🌠', 63, 100,'VideoCreator',  NOW(), NOW()),
(gen_random_uuid(), 'Video Jingle',       'video-jingle',       'Create short video clips with music for social media.',                 'Create', 150, 'pollinations', 'wan-fast',          true, false, '📱', 64, 35, 'VideoCreator',  NOW(), NOW())
ON CONFLICT (slug) DO UPDATE SET
    name             = EXCLUDED.name,
    description      = EXCLUDED.description,
    category         = EXCLUDED.category,
    point_cost       = EXCLUDED.point_cost,
    provider         = EXCLUDED.provider,
    provider_tool    = EXCLUDED.provider_tool,
    is_active        = EXCLUDED.is_active,
    is_free          = EXCLUDED.is_free,
    icon             = EXCLUDED.icon,
    sort_order       = EXCLUDED.sort_order,
    entry_point_cost = EXCLUDED.entry_point_cost,
    ui_template      = EXCLUDED.ui_template,
    updated_at       = NOW();
