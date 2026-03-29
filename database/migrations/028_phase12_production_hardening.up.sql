-- ════════════════════════════════════════════════════════════════════════════
--  028_phase12_production_hardening.sql
--  Production-ready config for:
--    1. Correct studio_tools catalogue (18 tools, exact costs from spec doc)
--    2. Storage backend network_config keys (provider-agnostic: S3 / GCS / local)
--    3. Chat/LLM tuning keys
--    4. AI provider config keys for all new adapters
-- ════════════════════════════════════════════════════════════════════════════

-- BEGIN;  -- removed: managed by golang-migrate

-- ─── 1. Update studio_tools with correct costs from spec key-points doc ───────
--
--  Cost rationale (spec §3.2 + key-points doc):
--    Free AI providers (Gemini, Groq, HF)  → cheapest tools (1–5 pts)
--    Paid APIs (FAL.AI, ElevenLabs, Mubert) → mid-range (5–200 pts)
--    Premium video (Kling v1.5)             → expensive (65 pts)
--    Full production jingle (ElevenLabs)   → 200 pts
--    Composite video+jingle                → 470 pts (future roadmap)
-- ─────────────────────────────────────────────────────────────────────────────

INSERT INTO studio_tools
    (id, name, slug, description, category, point_cost, provider, provider_tool,
     icon, sort_order, is_active, created_at, updated_at)
VALUES
-- ── Learn (Free AI providers — Gemini Flash primary, Groq fallback) ──────────
(gen_random_uuid(), 'Translate',        'translate',
    'Translate text to Yoruba, Hausa, Igbo, French or English',
    'Create',    1,  'google-translate',  'translate/v2',                        '🌍',  1,  true, NOW(), NOW()),

(gen_random_uuid(), 'Study Guide',      'study-guide',
    'Generate a comprehensive study guide with concepts, examples and quiz',
    'Learn',     3,  'gemini-flash',      'gemini-2.0-flash',                    '📖',  2,  true, NOW(), NOW()),

(gen_random_uuid(), 'Quiz Generator',   'quiz',
    'Create 10 multiple-choice quiz questions with explanations',
    'Learn',     2,  'gemini-flash',      'gemini-2.0-flash',                    '🧠',  3,  true, NOW(), NOW()),

(gen_random_uuid(), 'Mind Map',         'mindmap',
    'Turn any topic into a structured JSON mind map',
    'Learn',     2,  'gemini-flash',      'gemini-2.0-flash',                    '🗺️',  4,  true, NOW(), NOW()),

(gen_random_uuid(), 'Podcast',          'podcast',
    'Script and narrate a 2-host podcast (Nexus & Ade)',
    'Learn',     4,  'gemini+google-tts', 'composite',                           '🎧',  5,  true, NOW(), NOW()),

-- ── Build (Gemini Flash → Groq → DeepSeek for complex docs) ─────────────────
(gen_random_uuid(), 'Research Brief',   'research-brief',
    'Write a structured research brief with market data and recommendations',
    'Build',     5,  'gemini-flash',      'gemini-2.0-flash',                    '📊',  6,  true, NOW(), NOW()),

(gen_random_uuid(), 'Slide Deck',       'slide-deck',
    'Generate a 10-slide presentation outline as structured JSON',
    'Build',     4,  'gemini-flash',      'gemini-2.0-flash',                    '📑',  7,  true, NOW(), NOW()),

(gen_random_uuid(), 'Infographic',      'infographic',
    'Create infographic content structure with stats, headings and bullets',
    'Build',     5,  'gemini-flash',      'gemini-2.0-flash',                    '📊',  8,  true, NOW(), NOW()),

(gen_random_uuid(), 'Business Plan',    'bizplan',
    'Write a full Nigerian market business plan (8 structured sections)',
    'Build',    12,  'gemini-flash',      'gemini-2.0-flash',                    '💼',  9,  true, NOW(), NOW()),

-- ── Create — free/cheap AI providers ─────────────────────────────────────────
(gen_random_uuid(), 'Background Remover', 'bg-remover',
    'Remove image background instantly (rembg → FAL.AI BiRefNet)',
    'Create',    3,  'rembg',             'fal-ai/birefnet',                     '✂️', 10,  true, NOW(), NOW()),

(gen_random_uuid(), 'Narrate',          'narrate',
    'Convert text to natural-sounding Nigerian English speech',
    'Create',    2,  'google-cloud-tts',  'en-NG',                               '🎙️', 11,  true, NOW(), NOW()),

(gen_random_uuid(), 'Transcribe',       'transcribe',
    'Transcribe audio to text (AssemblyAI → Groq Whisper)',
    'Create',    2,  'assemblyai',        'best',                                '📝', 12,  true, NOW(), NOW()),

(gen_random_uuid(), 'Background Music', 'bg-music',
    'Generate 15s royalty-free background music (HuggingFace MusicGen)',
    'Create',    5,  'hf-musicgen',       'facebook/musicgen-small',             '🎶', 13,  true, NOW(), NOW()),

(gen_random_uuid(), 'AI Photo',         'ai-photo',
    'Generate a high-quality AI image from your description',
    'Create',   10,  'hf-flux-schnell',   'black-forest-labs/FLUX.1-schnell',    '🖼️', 14,  true, NOW(), NOW()),

-- ── Create — paid/premium providers ──────────────────────────────────────────
(gen_random_uuid(), 'Animate Photo',    'animate-photo',
    'Bring a still photo to life with smooth 5-second animation',
    'Create',   65,  'fal.ai',            'fal-ai/ltx-video',                    '🎬', 15,  true, NOW(), NOW()),

(gen_random_uuid(), 'Video Premium',    'video-premium',
    'Cinematic AI video from image (Kling v1.5 Standard)',
    'Create',   65,  'fal.ai',            'fal-ai/kling-video/v1.5/standard',    '🎥', 16,  true, NOW(), NOW()),

(gen_random_uuid(), 'Jingle',           'jingle',
    'Generate a 30-second AI music jingle for your brand (ElevenLabs)',
    'Create',  200,  'elevenlabs',        'sound-generation',                    '🎵', 17,  true, NOW(), NOW()),

(gen_random_uuid(), 'Video Jingle',     'video-jingle',
    'Full production: Kling video + ElevenLabs music score',
    'Build',   470,  'fal.ai+elevenlabs', 'composite',                           '🎞️', 18,  false, NOW(), NOW())

ON CONFLICT (slug) DO UPDATE
    SET name          = EXCLUDED.name,
        description   = EXCLUDED.description,
        category      = EXCLUDED.category,
        point_cost    = EXCLUDED.point_cost,
        provider      = EXCLUDED.provider,
        provider_tool = EXCLUDED.provider_tool,
        icon          = EXCLUDED.icon,
        sort_order    = EXCLUDED.sort_order,
        is_active     = EXCLUDED.is_active,
        updated_at    = NOW();

-- ─── 2. Storage backend configuration keys ────────────────────────────────────
--
--  STORAGE_BACKEND drives which concrete implementation is used:
--    "s3"    → AWS S3 (or S3-compatible: MinIO, Cloudflare R2)
--    "gcs"   → Google Cloud Storage
--    "local" → local filesystem (development / CI)
--    ""      → auto-detect from available credentials
--
--  These keys map 1:1 to environment variable overrides.
--  Operators can override at runtime by editing network_configs;
--  the ConfigManager reads from DB first, env var as fallback.
-- ─────────────────────────────────────────────────────────────────────────────

INSERT INTO network_configs (key, value, description, updated_at)
VALUES
    ('storage_backend',         '',
     'Asset storage provider: "s3", "gcs", "local", or "" for auto-detect',     NOW()),

    ('storage_cdn_base_url',    '',
     'CDN prefix returned in all asset URLs (e.g. https://cdn.loyalty-nexus.ai)', NOW()),

    -- AWS S3 / S3-compatible
    ('aws_s3_bucket',           '',     'S3 bucket name (AWS / MinIO / Cloudflare R2)',     NOW()),
    ('aws_region',              'us-east-1', 'AWS region (default: us-east-1)',            NOW()),
    ('aws_s3_endpoint',         '',
     'Custom S3-compatible endpoint (leave blank for standard AWS)',              NOW()),

    -- Google Cloud Storage
    ('gcs_bucket',              '',     'GCS bucket name',                                  NOW()),

    -- Local filesystem (dev / CI only)
    ('local_storage_base_path', '/tmp/nexus-assets',
     'Absolute filesystem path for local asset storage (dev only)',              NOW()),
    ('local_storage_base_url',  'http://localhost:8080/assets',
     'URL prefix served for local assets (dev only)',                            NOW())

ON CONFLICT (key) DO NOTHING;

-- ─── 3. LLM / Chat configuration keys ────────────────────────────────────────

INSERT INTO network_configs (key, value, description, updated_at)
VALUES
    -- Provider routing limits (daily request counts)
    ('chat_groq_daily_limit',            '1000',
     'Max Groq (Llama-4-Scout) requests per user per day before falling to Gemini',  NOW()),
    ('chat_gemini_daily_limit',          '2000',
     'Max cumulative requests (Groq+Gemini) per user per day before DeepSeek',        NOW()),

    -- Session memory
    ('chat_session_timeout_minutes',     '30',
     'Minutes of inactivity before a chat session is marked stale and summarised',   NOW()),
    ('chat_session_summary_messages',    '10',
     'Number of messages that trigger an incremental session summary',               NOW()),
    ('chat_memory_summaries_count',      '3',
     'Number of past session summaries injected into the system prompt',             NOW()),
    ('chat_memory_recent_messages',      '5',
     'Number of recent raw messages injected into the system prompt',                NOW()),

    -- LLM model overrides (operator can swap models without a deploy)
    ('llm_groq_model',          'llama-4-scout-17b-16e-instruct',
     'Groq model identifier',                                                         NOW()),
    ('llm_gemini_model',        'gemini-2.0-flash-lite',
     'Gemini model identifier (free Flash-Lite)',                                    NOW()),
    ('llm_deepseek_model',      'deepseek-chat',
     'DeepSeek model identifier (paid overflow)',                                    NOW())

ON CONFLICT (key) DO NOTHING;

-- ─── 4. AI provider configuration keys ───────────────────────────────────────

INSERT INTO network_configs (key, value, description, updated_at)
VALUES
    -- Image generation
    ('studio_hf_image_model',
     'black-forest-labs/FLUX.1-schnell',
     'HuggingFace model for AI photo (free tier)',                               NOW()),

    -- TTS
    ('studio_elevenlabs_voice_id',
     '21m00Tcm4TlvDq8ikWAM',
     'Default ElevenLabs voice ID (Rachel)',                                      NOW()),
    ('studio_tts_primary_provider',
     'google-cloud-tts',
     'Primary TTS provider: google-cloud-tts | elevenlabs | huggingface-bark',   NOW()),

    -- Background removal
    ('studio_rembg_service_url',
     '',
     'Self-hosted rembg microservice URL (e.g. http://rembg-service:5000)',       NOW()),

    -- Music
    ('studio_mubert_duration_secs',
     '30',
     'Duration in seconds for Mubert background music generation',               NOW()),

    -- Video
    ('studio_fal_video_model_standard',
     'fal-ai/ltx-video',
     'FAL.AI model for animate-photo (cheaper)',                                  NOW()),
    ('studio_fal_video_model_premium',
     'fal-ai/kling-video/v1.5/standard',
     'FAL.AI model for video-premium (Kling v1.5)',                               NOW()),

    -- Stale job recovery (also used by LifecycleWorker)
    ('studio_stale_job_timeout_minutes',
     '15',
     'Minutes before a stuck pending/processing job is failed and refunded',     NOW()),
    ('studio_stale_recovery_batch',
     '50',
     'Max stale jobs recovered per LifecycleWorker tick',                        NOW()),

    -- Transcription
    ('studio_transcription_primary',
     'assemblyai',
     'Primary transcription provider: assemblyai | groq-whisper',               NOW())

ON CONFLICT (key) DO NOTHING;

-- COMMIT;  -- removed: managed by golang-migrate
