-- Migration 067: Fix spin_tiers missing table + seed studio tools
-- Fixes: spin_tiers 500 error, studio tools null

-- 1. Ensure spin_tiers table exists (migration 040 may not have run)
CREATE TABLE IF NOT EXISTS spin_tiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tier_name TEXT NOT NULL,
    tier_display_name TEXT NOT NULL,
    min_daily_amount BIGINT NOT NULL DEFAULT 0,
    max_daily_amount BIGINT NOT NULL DEFAULT 999999999999,
    spins_per_day INTEGER NOT NULL DEFAULT 1,
    tier_color TEXT,
    tier_icon TEXT,
    tier_badge TEXT,
    description TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default spin tiers
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

-- 2. Ensure studio_tools table exists
CREATE TABLE IF NOT EXISTS studio_tools (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            TEXT        NOT NULL UNIQUE,
    name            TEXT        NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    category        TEXT        NOT NULL DEFAULT 'text',
    icon            TEXT        NOT NULL DEFAULT '🤖',
    pulse_cost      INTEGER     NOT NULL DEFAULT 0,
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    is_free         BOOLEAN     NOT NULL DEFAULT FALSE,
    sort_order      INTEGER     NOT NULL DEFAULT 0,
    template_type   TEXT        NOT NULL DEFAULT 'KnowledgeDoc',
    tags            TEXT[]      NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Seed all 30+ studio tools
INSERT INTO studio_tools (slug, name, description, category, icon, pulse_cost, is_active, is_free, sort_order, template_type, tags) VALUES
-- ── FREE TOOLS (0 Pulse Points) ─────────────────────────────────────────────
('ask-nexus',        'Ask Nexus',          'Chat with a powerful AI assistant for any question.',                    'chat',    '💬', 0,   TRUE, TRUE,  1,  'KnowledgeDoc', ARRAY['free','chat','ai']),
('web-search-ai',    'Web Search AI',      'Search the web and get AI-summarised answers.',                         'chat',    '🔍', 0,   TRUE, TRUE,  2,  'KnowledgeDoc', ARRAY['free','search','web']),
('code-helper',      'Code Helper',        'Get help writing, debugging and explaining code.',                      'chat',    '💻', 0,   TRUE, TRUE,  3,  'KnowledgeDoc', ARRAY['free','code','dev']),
('image-analyser',   'Image Analyser',     'Upload an image and ask AI questions about it.',                        'vision',  '🔬', 0,   TRUE, TRUE,  4,  'VisionAsk',    ARRAY['free','vision','image']),
('ask-my-photo',     'Ask My Photo',       'Upload a photo and get detailed AI analysis.',                          'vision',  '📸', 0,   TRUE, TRUE,  5,  'VisionAsk',    ARRAY['free','vision','photo']),
('transcribe',       'Transcribe',         'Convert audio recordings to text instantly.',                           'voice',   '🎙️', 0,   TRUE, TRUE,  6,  'Transcribe',   ARRAY['free','audio','transcribe']),
('transcribe-african','African Transcribe','Transcribe audio in African languages including Yoruba, Igbo, Hausa.',  'voice',   '🌍', 0,   TRUE, TRUE,  7,  'Transcribe',   ARRAY['free','audio','african']),
-- ── LEARN TOOLS ─────────────────────────────────────────────────────────────
('study-guide',      'Study Guide',        'Turn any topic into a comprehensive study guide.',                      'learn',   '📚', 50,  TRUE, FALSE, 10, 'KnowledgeDoc', ARRAY['learn','study','education']),
('quiz',             'Quiz Maker',         'Generate quizzes and practice tests on any subject.',                   'learn',   '❓', 50,  TRUE, FALSE, 11, 'KnowledgeDoc', ARRAY['learn','quiz','test']),
('mindmap',          'Mind Map',           'Create visual mind maps to organise your ideas.',                       'learn',   '🗺️', 50,  TRUE, FALSE, 12, 'KnowledgeDoc', ARRAY['learn','mindmap','visual']),
('research-brief',   'Research Brief',     'Generate a detailed research brief on any topic.',                      'learn',   '🔬', 75,  TRUE, FALSE, 13, 'KnowledgeDoc', ARRAY['learn','research','brief']),
-- ── BUILD TOOLS ─────────────────────────────────────────────────────────────
('slide-deck',       'Slide Deck',         'Generate a professional presentation slide deck.',                      'build',   '📊', 100, TRUE, FALSE, 20, 'KnowledgeDoc', ARRAY['build','slides','presentation']),
('infographic',      'Infographic',        'Create an infographic outline from any topic or data.',                 'build',   '📈', 75,  TRUE, FALSE, 21, 'KnowledgeDoc', ARRAY['build','infographic','visual']),
('bizplan',          'Business Plan AI',   'Generate investor-ready business plans in minutes.',                    'build',   '💼', 150, TRUE, FALSE, 22, 'KnowledgeDoc', ARRAY['build','business','plan']),
('podcast',          'AI Podcast',         'Generate a podcast script from any topic.',                             'build',   '🎙️', 100, TRUE, FALSE, 23, 'KnowledgeDoc', ARRAY['build','podcast','audio']),
-- ── VOICE TOOLS ─────────────────────────────────────────────────────────────
('narrate',          'Narrate',            'Convert text to natural-sounding speech.',                              'voice',   '🔊', 30,  TRUE, FALSE, 30, 'VoiceStudio',  ARRAY['voice','tts','narrate']),
('narrate-pro',      'Narrate Pro',        'Premium TTS with 13 voices, 7 languages, speed & format controls.',    'voice',   '🎤', 75,  TRUE, FALSE, 31, 'VoiceStudio',  ARRAY['voice','tts','premium']),
('translate',        'Translate',          'Translate text between languages with AI.',                             'voice',   '🌐', 30,  TRUE, FALSE, 32, 'VoiceStudio',  ARRAY['voice','translate','language']),
-- ── IMAGE TOOLS ─────────────────────────────────────────────────────────────
('bg-remover',       'BG Remover',         'Remove image backgrounds instantly with AI.',                           'image',   '✂️', 20,  TRUE, FALSE, 40, 'ImageEditor',  ARRAY['image','background','remove']),
('ai-photo',         'AI Photo Creator',   'Generate stunning images from text prompts.',                           'image',   '🎨', 50,  TRUE, FALSE, 41, 'ImageCreator', ARRAY['image','generate','art']),
('ai-photo-pro',     'AI Photo Pro',       'Premium image generation with GPT-Image quality.',                      'image',   '🖼️', 150, TRUE, FALSE, 42, 'ImageCreator', ARRAY['image','premium','gpt']),
('ai-photo-max',     'AI Photo Max',       'Maximum quality AI image generation.',                                  'image',   '🌟', 250, TRUE, FALSE, 43, 'ImageCreator', ARRAY['image','max','quality']),
('ai-photo-dream',   'AI Photo Dream',     'Dreamlike artistic AI image generation.',                               'image',   '✨', 200, TRUE, FALSE, 44, 'ImageCreator', ARRAY['image','dream','art']),
('photo-editor',     'Photo Editor',       'Edit and transform photos with AI prompts.',                            'image',   '🖌️', 100, TRUE, FALSE, 45, 'ImageEditor',  ARRAY['image','edit','transform']),
-- ── MUSIC TOOLS ─────────────────────────────────────────────────────────────
('bg-music',         'Background Music',   'Generate background music for videos and content.',                     'music',   '🎵', 75,  TRUE, FALSE, 50, 'MusicComposer',ARRAY['music','background','ambient']),
('jingle',           'Jingle Maker',       'Create catchy jingles for your brand or product.',                      'music',   '🎶', 100, TRUE, FALSE, 51, 'MusicComposer',ARRAY['music','jingle','brand']),
('song-creator',     'Song Creator',       'Generate full songs with lyrics and vocals.',                           'music',   '🎸', 200, TRUE, FALSE, 52, 'MusicComposer',ARRAY['music','song','vocals']),
('instrumental',     'Instrumental',       'Generate instrumental music in any genre.',                             'music',   '🎹', 150, TRUE, FALSE, 53, 'MusicComposer',ARRAY['music','instrumental','genre']),
-- ── VIDEO TOOLS ─────────────────────────────────────────────────────────────
('animate-photo',    'Animate Photo',      'Bring still photos to life with AI animation.',                         'video',   '🎬', 100, TRUE, FALSE, 60, 'VideoAnimator',ARRAY['video','animate','photo']),
('video-premium',    'Video Premium',      'Premium quality AI video generation.',                                  'video',   '🎥', 200, TRUE, FALSE, 61, 'VideoAnimator',ARRAY['video','premium','generate']),
('video-cinematic',  'Video Cinematic',    'Create cinematic quality videos from text prompts.',                    'video',   '🎞️', 300, TRUE, FALSE, 62, 'VideoCreator', ARRAY['video','cinematic','text']),
('video-veo',        'Video Veo',          'Google Veo-powered ultra-realistic video generation.',                  'video',   '🌠', 500, TRUE, FALSE, 63, 'VideoCreator', ARRAY['video','veo','realistic']),
('video-jingle',     'Video Jingle',       'Create short video clips with music for social media.',                 'video',   '📱', 150, TRUE, FALSE, 64, 'VideoCreator', ARRAY['video','jingle','social'])
ON CONFLICT (slug) DO UPDATE SET
  name          = EXCLUDED.name,
  description   = EXCLUDED.description,
  category      = EXCLUDED.category,
  icon          = EXCLUDED.icon,
  pulse_cost    = EXCLUDED.pulse_cost,
  is_active     = EXCLUDED.is_active,
  is_free       = EXCLUDED.is_free,
  sort_order    = EXCLUDED.sort_order,
  template_type = EXCLUDED.template_type,
  tags          = EXCLUDED.tags,
  updated_at    = NOW();

-- 4. Fix notification_broadcasts null response — ensure table exists
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

-- Verify
SELECT 'spin_tiers' as tbl, COUNT(*) FROM spin_tiers
UNION ALL SELECT 'studio_tools', COUNT(*) FROM studio_tools
UNION ALL SELECT 'notification_broadcasts', COUNT(*) FROM notification_broadcasts;
