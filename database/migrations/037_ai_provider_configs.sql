-- Migration 037: AI Provider Configs
-- Dynamic provider management: admin can register, prioritise, and activate
-- any AI provider without code deployments.
--
-- Design:
--   category   = what it does (text | image | video | tts | transcribe | translate | music | bg-remove | vision)
--   template   = HOW to call it (openai-compatible | pollinations-image | pollinations-tts |
--                                pollinations-video | pollinations-music | fal-image | fal-video |
--                                fal-bg-remove | hf-image | google-tts | google-translate |
--                                elevenlabs-tts | elevenlabs-music | assemblyai | groq-whisper |
--                                mubert | remove-bg | deepseek | gemini | custom-rest)
--   env_key    = name of the env var that holds the API key (e.g. FAL_API_KEY) — never stored plaintext
--   api_key    = encrypted key (AES-GCM, key = PROVIDER_ENCRYPTION_KEY env var). NULL = use env_key only
--   priority   = lower = tried first (1 = primary, 2 = first backup, 3 = second backup, …)
--   is_primary = true if this is the preferred/main provider for this category
--   is_active  = false = skip this provider entirely (soft disable)

CREATE TABLE IF NOT EXISTS ai_provider_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,                          -- human label, e.g. "Pollinations FLUX"
    slug            TEXT NOT NULL UNIQUE,                   -- machine id, e.g. "pollinations-flux"
    category        TEXT NOT NULL,                          -- text | image | video | tts | transcribe | translate | music | bg-remove | vision
    template        TEXT NOT NULL,                          -- driver template (see above)
    env_key         TEXT NOT NULL DEFAULT '',               -- env var name holding the real key
    api_key_enc     TEXT NOT NULL DEFAULT '',               -- AES-GCM encrypted key (base64). empty = key lives in env only
    model_id        TEXT NOT NULL DEFAULT '',               -- model/endpoint override (e.g. "gemini-2.5-flash")
    extra_config    JSONB NOT NULL DEFAULT '{}',            -- template-specific params (voice_id, language, etc.)
    priority        INT NOT NULL DEFAULT 10,                -- 1=primary, higher=backup
    is_primary      BOOLEAN NOT NULL DEFAULT false,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    cost_micros     INT NOT NULL DEFAULT 0,                 -- platform cost per call in microdollars
    pulse_pts       INT NOT NULL DEFAULT 0,                 -- pulse points charged to user per call
    notes           TEXT NOT NULL DEFAULT '',               -- admin notes / display description
    last_tested_at  TIMESTAMPTZ,
    last_test_ok    BOOLEAN,
    last_test_msg   TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_provider_configs_category    ON ai_provider_configs (category);
CREATE INDEX IF NOT EXISTS idx_ai_provider_configs_cat_prio    ON ai_provider_configs (category, priority) WHERE is_active = true;

-- Trigger: auto-update updated_at
CREATE OR REPLACE FUNCTION ai_provider_configs_set_updated()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END; $$;

CREATE TRIGGER ai_provider_configs_updated_at
    BEFORE UPDATE ON ai_provider_configs
    FOR EACH ROW EXECUTE FUNCTION ai_provider_configs_set_updated();

-- ── Seed with the current hardcoded providers ────────────────────────────────
-- These mirror the exact chains in ai_studio_service.go so the admin panel
-- shows the current state on day one (and the dynamic dispatch can use them).

INSERT INTO ai_provider_configs
    (name, slug, category, template, env_key, model_id, priority, is_primary, is_active, cost_micros, pulse_pts, notes)
VALUES
-- ── TEXT ─────────────────────────────────────────────────────────────────────
('Pollinations OpenAI',   'pollinations-text',   'text', 'openai-compatible',  'POLLINATIONS_SECRET_KEY', 'openai',                          1, true,  true, 0,     0,  'Pollinations free text via OpenAI-compat endpoint'),
('Gemini 2.5 Flash',     'gemini-flash',        'text', 'gemini',             'GEMINI_API_KEY',          'gemini-2.5-flash',                2, false, true, 0,     0,  'Google Gemini 2.5 Flash — free tier'),
('Groq Llama-4 Scout',   'groq-llama4',         'text', 'openai-compatible',  'GROQ_API_KEY',            'meta-llama/llama-4-scout-17b-16e-instruct', 3, false, true, 0, 0, 'Groq inference — Llama 4 Scout'),
('DeepSeek V3',          'deepseek-v3',         'text', 'deepseek',           'DEEPSEEK_API_KEY',        'deepseek-chat',                   4, false, true, 0,     0,  'DeepSeek V3 via official API'),

-- ── IMAGE ─────────────────────────────────────────────────────────────────────
('HuggingFace FLUX Schnell', 'hf-flux-schnell', 'image', 'hf-image',          'HF_TOKEN',                'black-forest-labs/FLUX.1-schnell', 1, true,  true, 0,     0,  'HF serverless inference — free with token'),
('Pollinations FLUX',    'pollinations-flux',   'image', 'pollinations-image', 'POLLINATIONS_SECRET_KEY', 'flux',                            2, false, true, 0,     0,  'Pollinations FLUX — sk_ key required'),
('FAL FLUX Dev',         'fal-flux-dev',        'image', 'fal-image',          'FAL_API_KEY',             'fal-ai/flux/dev',                 3, false, true, 6500,  0,  'FAL.AI FLUX-dev — ~$0.025/image'),

-- ── VIDEO ─────────────────────────────────────────────────────────────────────
('FAL Kling v1.5',       'fal-kling',           'video', 'fal-video',          'FAL_API_KEY',             'fal-ai/kling-video/v1.5/standard/image-to-video', 1, true, true, 56000, 0, 'FAL Kling v1.5 — premium quality'),
('FAL LTX Video',        'fal-ltx',             'video', 'fal-video',          'FAL_API_KEY',             'fal-ai/ltx-video',                2, false, true, 14500, 0,  'FAL LTX — faster/cheaper option'),
('Pollinations Seedance','pollinations-seedance','video', 'pollinations-video', 'POLLINATIONS_SECRET_KEY', 'seedance',                        3, false, true, 50000, 0,  'Pollinations Seedance — ~28s generation'),
('Pollinations Wan-Fast','pollinations-wan-fast','video', 'pollinations-video', 'POLLINATIONS_SECRET_KEY', 'wan-fast',                        4, false, true, 30000, 0,  'Wan 2.2 — slower ~50s backup'),

-- ── TTS ───────────────────────────────────────────────────────────────────────
('Google Cloud TTS',     'google-cloud-tts',    'tts', 'google-tts',           'GOOGLE_CLOUD_TTS_KEY',    '',                                1, true,  true, 0,     0,  'Google TTS — 1M chars/month free'),
('ElevenLabs TTS',       'elevenlabs-tts',      'tts', 'elevenlabs-tts',       'ELEVENLABS_API_KEY',      'eleven_flash_v2_5',               2, false, true, 2000,  0,  'ElevenLabs — Sarah voice (premade)'),
('Pollinations TTS',     'pollinations-tts',    'tts', 'pollinations-tts',     'POLLINATIONS_SECRET_KEY', 'elevenlabs',                      3, false, true, 0,     0,  'Pollinations TTS fallback'),

-- ── TRANSCRIBE ────────────────────────────────────────────────────────────────
('AssemblyAI',           'assemblyai',          'transcribe', 'assemblyai',    'ASSEMBLY_AI_KEY',         'universal-2',                     1, true,  true, 25,    0,  'AssemblyAI Universal-2 model'),
('Groq Whisper',         'groq-whisper',        'transcribe', 'groq-whisper',  'GROQ_API_KEY',            'whisper-large-v3-turbo',          2, false, true, 10,    0,  'Groq Whisper large-v3-turbo'),

-- ── TRANSLATE ─────────────────────────────────────────────────────────────────
('Google Translate',     'google-translate',    'translate', 'google-translate','GOOGLE_TRANSLATE_API_KEY','',                              1, true,  true, 0,     0,  'Google Translate API v2'),
('Gemini Translate',     'gemini-translate',    'translate', 'gemini',          'GEMINI_API_KEY',          'gemini-2.5-flash',              2, false, true, 0,     0,  'Gemini Flash as translation fallback'),

-- ── MUSIC ─────────────────────────────────────────────────────────────────────
('Pollinations ElevenMusic','pollinations-elevenmusic','music','pollinations-music','POLLINATIONS_SECRET_KEY','elevenmusic',                 1, true,  true, 500,   0,  'Pollinations ElevenMusic — instrumental'),
('Mubert',               'mubert',              'music', 'mubert',             'MUBERT_API_KEY',           '',                               2, false, false, 0,    0,  'Mubert royalty-free music — key pending'),
('ElevenLabs Music',     'elevenlabs-music',    'music', 'elevenlabs-music',   'ELEVENLABS_API_KEY',       '',                               3, false, true, 500,   0,  'ElevenLabs music/sound generation'),

-- ── BG REMOVE ─────────────────────────────────────────────────────────────────
('rembg Self-Hosted',    'rembg-self-hosted',   'bg-remove', 'rembg',          'REMBG_SERVICE_URL',        '',                               1, true,  true, 0,     0,  'Self-hosted rembg microservice — free'),
('FAL BiRefNet',         'fal-birefnet',        'bg-remove', 'fal-bg-remove',  'FAL_API_KEY',              'fal-ai/birefnet',                2, false, true, 2000,  0,  'FAL BiRefNet — ~$0.003/megapixel'),
('remove.bg',            'remove-bg',           'bg-remove', 'remove-bg',      'REMOVEBG_API_KEY',         '',                               3, false, true, 1000,  0,  'remove.bg — $0.20/image last resort'),

-- ── VISION ────────────────────────────────────────────────────────────────────
('Pollinations Vision',  'pollinations-vision', 'vision', 'openai-compatible', 'POLLINATIONS_SECRET_KEY',  'openai',                         1, true,  true, 0,     0,  'Pollinations multimodal vision'),
('Gemini Vision',        'gemini-vision',       'vision', 'gemini',            'GEMINI_API_KEY',           'gemini-2.5-flash',               2, false, true, 0,     0,  'Gemini Flash vision fallback')

ON CONFLICT (slug) DO NOTHING;
