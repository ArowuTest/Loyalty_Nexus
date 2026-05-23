-- ============================================================
-- Loyalty Nexus: Consolidated missing-table fix
-- Run once in Render Shell via: psql $DATABASE_URL -f /tmp/fix.sql
-- Or paste each block into psql interactively
-- ============================================================

-- 1. Fix admin role: ensure the seeded admin is super_admin
UPDATE admin_users SET role = 'super_admin' WHERE role = 'operations' AND email = (SELECT email FROM admin_users ORDER BY created_at LIMIT 1);
-- Also ensure ALL admin users are super_admin if there's only one
UPDATE admin_users SET role = 'super_admin' WHERE (SELECT COUNT(*) FROM admin_users) = 1;

-- 2. Create ai_provider_configs table (migration 037)
CREATE TABLE IF NOT EXISTS ai_provider_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    category        TEXT NOT NULL,
    template        TEXT NOT NULL,
    env_key         TEXT NOT NULL DEFAULT '',
    api_key_enc     TEXT NOT NULL DEFAULT '',
    model_id        TEXT NOT NULL DEFAULT '',
    extra_config    JSONB NOT NULL DEFAULT '{}',
    priority        INT NOT NULL DEFAULT 10,
    is_primary      BOOLEAN NOT NULL DEFAULT false,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    cost_micros     INT NOT NULL DEFAULT 0,
    pulse_pts       INT NOT NULL DEFAULT 0,
    notes           TEXT NOT NULL DEFAULT '',
    last_tested_at  TIMESTAMPTZ,
    last_test_ok    BOOLEAN,
    last_test_msg   TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ai_provider_configs_category ON ai_provider_configs (category);
CREATE INDEX IF NOT EXISTS idx_ai_provider_configs_cat_prio ON ai_provider_configs (category, priority) WHERE is_active = true;

-- Seed AI providers
INSERT INTO ai_provider_configs (name, slug, category, template, env_key, model_id, priority, is_primary, is_active, cost_micros, pulse_pts, notes) VALUES
('Pollinations OpenAI',   'pollinations-text',   'text', 'openai-compatible',  'POLLINATIONS_SECRET_KEY', 'openai',                          1, true,  true, 0,     0,  'Pollinations free text via OpenAI-compat endpoint'),
('Gemini 2.5 Flash',     'gemini-flash',        'text', 'gemini',             'GEMINI_API_KEY',          'gemini-2.5-flash',                2, false, true, 0,     0,  'Google Gemini 2.5 Flash'),
('Groq Llama-4 Scout',   'groq-llama4',         'text', 'openai-compatible',  'GROQ_API_KEY',            'meta-llama/llama-4-scout-17b-16e-instruct', 3, false, true, 0, 0, 'Groq inference'),
('DeepSeek V3',          'deepseek-v3',         'text', 'deepseek',           'DEEPSEEK_API_KEY',        'deepseek-chat',                   4, false, true, 0,     0,  'DeepSeek V3'),
('HuggingFace FLUX',     'hf-flux-schnell',     'image', 'hf-image',          'HF_TOKEN',                'black-forest-labs/FLUX.1-schnell', 1, true,  true, 0,     0,  'HF serverless inference'),
('Pollinations FLUX',    'pollinations-flux',   'image', 'pollinations-image', 'POLLINATIONS_SECRET_KEY', 'flux',                            2, false, true, 0,     0,  'Pollinations FLUX'),
('FAL FLUX Dev',         'fal-flux-dev',        'image', 'fal-image',          'FAL_API_KEY',             'fal-ai/flux/dev',                 3, false, true, 6500,  0,  'FAL.AI FLUX-dev'),
('FAL Kling v1.5',       'fal-kling',           'video', 'fal-video',          'FAL_API_KEY',             'fal-ai/kling-video/v1.5/standard/image-to-video', 1, true, true, 56000, 0, 'FAL Kling v1.5'),
('FAL LTX Video',        'fal-ltx',             'video', 'fal-video',          'FAL_API_KEY',             'fal-ai/ltx-video',                2, false, true, 14500, 0,  'FAL LTX'),
('Pollinations Wan-Fast','pollinations-wan-fast','video', 'pollinations-video', 'POLLINATIONS_SECRET_KEY', 'wan-fast',                        3, true,  true, 0,     0,  'Wan 2.2 FREE'),
('Google Cloud TTS',     'google-cloud-tts',    'tts', 'google-tts',           'GOOGLE_CLOUD_TTS_KEY',    '',                                1, true,  true, 0,     0,  'Google TTS'),
('ElevenLabs TTS',       'elevenlabs-tts',      'tts', 'elevenlabs-tts',       'ELEVENLABS_API_KEY',      'eleven_flash_v2_5',               2, false, true, 2000,  0,  'ElevenLabs TTS'),
('Pollinations TTS',     'pollinations-tts',    'tts', 'pollinations-tts',     'POLLINATIONS_SECRET_KEY', 'elevenlabs',                      3, false, true, 0,     0,  'Pollinations TTS fallback'),
('AssemblyAI',           'assemblyai',          'transcribe', 'assemblyai',    'ASSEMBLY_AI_KEY',         'universal-2',                     1, true,  true, 25,    0,  'AssemblyAI Universal-2'),
('Groq Whisper',         'groq-whisper',        'transcribe', 'groq-whisper',  'GROQ_API_KEY',            'whisper-large-v3-turbo',          2, false, true, 10,    0,  'Groq Whisper'),
('Google Translate',     'google-translate',    'translate', 'google-translate','GOOGLE_TRANSLATE_API_KEY','',                              1, true,  true, 0,     0,  'Google Translate API v2'),
('Gemini Translate',     'gemini-translate',    'translate', 'gemini',          'GEMINI_API_KEY',          'gemini-2.5-flash',              2, false, true, 0,     0,  'Gemini Flash translation'),
('Pollinations Music',   'pollinations-elevenmusic','music','pollinations-music','POLLINATIONS_SECRET_KEY','elevenmusic',                 1, true,  true, 500,   0,  'Pollinations ElevenMusic'),
('rembg Self-Hosted',    'rembg-self-hosted',   'bg-remove', 'rembg',          'REMBG_SERVICE_URL',        '',                               1, true,  true, 0,     0,  'Self-hosted rembg'),
('FAL BiRefNet',         'fal-birefnet',        'bg-remove', 'fal-bg-remove',  'FAL_API_KEY',              'fal-ai/birefnet',                2, false, true, 2000,  0,  'FAL BiRefNet'),
('Pollinations Vision',  'pollinations-vision', 'vision', 'openai-compatible', 'POLLINATIONS_SECRET_KEY',  'openai',                         1, true,  true, 0,     0,  'Pollinations vision'),
('Gemini Vision',        'gemini-vision',       'vision', 'gemini',            'GEMINI_API_KEY',           'gemini-2.5-flash',               2, false, true, 0,     0,  'Gemini Flash vision')
ON CONFLICT (slug) DO NOTHING;

-- 3. Create draw_schedules table (migration 049)
CREATE TABLE IF NOT EXISTS draw_schedules (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    draw_name           TEXT        NOT NULL,
    draw_type           TEXT        NOT NULL,
    draw_day_of_week    INTEGER     NOT NULL CHECK (draw_day_of_week BETWEEN 0 AND 6),
    draw_time_wat       TIME        NOT NULL DEFAULT '17:00:00',
    window_open_dow     INTEGER     NOT NULL CHECK (window_open_dow BETWEEN 0 AND 6),
    window_open_time    TIME        NOT NULL DEFAULT '17:00:01',
    window_close_dow    INTEGER     NOT NULL CHECK (window_close_dow BETWEEN 0 AND 6),
    window_close_time   TIME        NOT NULL DEFAULT '17:00:00',
    cutoff_hour_utc     INTEGER     NOT NULL DEFAULT 16 CHECK (cutoff_hour_utc BETWEEN 0 AND 23),
    is_active           BOOLEAN     NOT NULL DEFAULT TRUE,
    sort_order          INTEGER     NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_draw_schedules_active ON draw_schedules (is_active, draw_day_of_week);

INSERT INTO draw_schedules (draw_name, draw_type, draw_day_of_week, draw_time_wat, window_open_dow, window_open_time, window_close_dow, window_close_time, cutoff_hour_utc, is_active, sort_order) VALUES
('Monday Daily Draw',    'DAILY',  1, '17:00:00', 4, '17:00:01', 0, '17:00:00', 16, TRUE, 1),
('Tuesday Daily Draw',   'DAILY',  2, '17:00:00', 0, '17:00:01', 1, '17:00:00', 16, TRUE, 2),
('Wednesday Daily Draw', 'DAILY',  3, '17:00:00', 1, '17:00:01', 2, '17:00:00', 16, TRUE, 3),
('Thursday Daily Draw',  'DAILY',  4, '17:00:00', 2, '17:00:01', 3, '17:00:00', 16, TRUE, 4),
('Friday Daily Draw',    'DAILY',  5, '17:00:00', 3, '17:00:01', 4, '17:00:00', 16, TRUE, 5),
('Saturday Weekly Mega Draw', 'WEEKLY', 6, '17:00:00', 5, '17:00:01', 5, '17:00:00', 16, TRUE, 6)
ON CONFLICT DO NOTHING;

-- 4. Create mtn_push_csv_uploads and mtn_push_csv_rows tables (migration 050)
CREATE TABLE IF NOT EXISTS mtn_push_csv_uploads (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    uploaded_by     TEXT        NOT NULL,
    filename        TEXT        NOT NULL,
    uploaded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    total_rows      INTEGER     NOT NULL DEFAULT 0,
    processed_rows  INTEGER     NOT NULL DEFAULT 0,
    skipped_rows    INTEGER     NOT NULL DEFAULT 0,
    failed_rows     INTEGER     NOT NULL DEFAULT 0,
    status          TEXT        NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','PROCESSING','DONE','PARTIAL','FAILED')),
    note            TEXT,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_csv_uploads_status ON mtn_push_csv_uploads (status, uploaded_at DESC);

CREATE TABLE IF NOT EXISTS mtn_push_csv_rows (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id       UUID        NOT NULL REFERENCES mtn_push_csv_uploads(id) ON DELETE CASCADE,
    row_number      INTEGER     NOT NULL,
    raw_msisdn      TEXT        NOT NULL,
    raw_date        TEXT        NOT NULL,
    raw_time        TEXT        NOT NULL,
    raw_amount      TEXT        NOT NULL,
    recharge_type   TEXT        NOT NULL DEFAULT 'AIRTIME',
    msisdn          TEXT,
    recharge_at     TIMESTAMPTZ,
    amount_naira    NUMERIC(12,2),
    status          TEXT        NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','OK','SKIPPED','FAILED')),
    skip_reason     TEXT,
    error_msg       TEXT,
    transaction_ref TEXT,
    spin_credits    INTEGER,
    pulse_points    BIGINT,
    draw_entries    INTEGER,
    processed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_csv_rows_upload ON mtn_push_csv_rows (upload_id, row_number);
CREATE INDEX IF NOT EXISTS idx_csv_rows_msisdn ON mtn_push_csv_rows (msisdn) WHERE msisdn IS NOT NULL;

-- 5. Create pulse_point_awards table (migration 051)
CREATE TABLE IF NOT EXISTS pulse_point_awards (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    phone_number    TEXT        NOT NULL,
    points          BIGINT      NOT NULL CHECK (points > 0),
    campaign        TEXT        NOT NULL DEFAULT '',
    note            TEXT        NOT NULL DEFAULT '',
    awarded_by      UUID        NOT NULL,
    awarded_by_name TEXT        NOT NULL DEFAULT '',
    transaction_id  UUID        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ppa_user_id ON pulse_point_awards (user_id);
CREATE INDEX IF NOT EXISTS idx_ppa_phone ON pulse_point_awards (phone_number);
CREATE INDEX IF NOT EXISTS idx_ppa_campaign ON pulse_point_awards (campaign) WHERE campaign <> '';
CREATE INDEX IF NOT EXISTS idx_ppa_awarded_by ON pulse_point_awards (awarded_by);

-- 6. Check fraud_events table exists and add any missing columns
ALTER TABLE fraud_events ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Verify results
SELECT 'ai_provider_configs' as tbl, COUNT(*) as rows FROM ai_provider_configs
UNION ALL SELECT 'draw_schedules', COUNT(*) FROM draw_schedules
UNION ALL SELECT 'mtn_push_csv_uploads', COUNT(*) FROM mtn_push_csv_uploads
UNION ALL SELECT 'pulse_point_awards', COUNT(*) FROM pulse_point_awards
UNION ALL SELECT 'fraud_events', COUNT(*) FROM fraud_events
UNION ALL SELECT 'admin_users_role_check', COUNT(*) FROM admin_users WHERE role = 'super_admin';
