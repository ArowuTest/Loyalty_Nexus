-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 032 — Studio UI Config
-- Adds ui_template (VARCHAR) and ui_config (JSONB) to studio_tools so the
-- frontend renders the correct purpose-built UI for every tool without
-- hardcoding any tool logic in React.
--
-- Templates:
--   chat              → persistent conversation thread (already exists)
--   music-composer    → song-creator, instrumental, jingle, bg-music
--   image-creator     → ai-photo-pro/max/dream
--   image-editor      → photo-editor (upload-first)
--   video-creator     → video-veo, video-premium (text-to-video)
--   video-animator    → video-cinematic, animate-photo, video-jingle (image-to-video)
--   voice-studio      → narrate-pro
--   transcribe        → transcribe-african
--   vision-ask        → image-analyser, ask-my-photo
--   knowledge-doc     → study-guide, quiz, mindmap, research-brief, bizplan,
--                        slide-deck, infographic, podcast, translate
-- ─────────────────────────────────────────────────────────────────────────────

-- 1. Add columns (idempotent)
ALTER TABLE studio_tools
  ADD COLUMN IF NOT EXISTS ui_template  VARCHAR(40)  NOT NULL DEFAULT 'knowledge-doc',
  ADD COLUMN IF NOT EXISTS ui_config    JSONB        NOT NULL DEFAULT '{}';

-- ─────────────────────────────────────────────────────────────────────────────
-- 2. CHAT tools
-- ─────────────────────────────────────────────────────────────────────────────
UPDATE studio_tools SET
  ui_template = 'chat',
  ui_config   = '{
    "prompt_placeholder": "Type your message…",
    "show_history": true
  }'::jsonb
WHERE slug IN ('ai-chat', 'web-search-ai', 'code-helper');

-- ─────────────────────────────────────────────────────────────────────────────
-- 3. MUSIC COMPOSER
-- ─────────────────────────────────────────────────────────────────────────────
UPDATE studio_tools SET
  ui_template = 'music-composer',
  ui_config   = '{
    "prompt_placeholder": "e.g. Upbeat Afrobeats track with energetic female vocals and a big chorus hook…",
    "genre_tags": ["Afrobeats","Amapiano","Gospel","Highlife","R&B","Hip-Hop","Pop","Jazz","Classical","EDM","Reggae","Funk"],
    "duration_options": [15, 30, 60, 120],
    "default_duration": 30,
    "show_vocals_toggle": true,
    "default_vocals": true,
    "show_lyrics_box": true,
    "lyrics_placeholder": "Optional: paste your own lyrics or leave blank for AI-generated lyrics\n\n[Verse 1]\n…\n[Chorus]\n…"
  }'::jsonb
WHERE slug IN ('song-creator', 'jingle', 'bg-music');

-- Instrumental variant — no vocals toggle
UPDATE studio_tools SET
  ui_template = 'music-composer',
  ui_config   = '{
    "prompt_placeholder": "e.g. Calm lo-fi background music with piano and soft drums, no vocals…",
    "genre_tags": ["Lo-fi","Cinematic","Ambient","Jazz","Classical","Acoustic","Electronic","World"],
    "duration_options": [15, 30, 60, 120],
    "default_duration": 60,
    "show_vocals_toggle": false,
    "default_vocals": false,
    "show_lyrics_box": false
  }'::jsonb
WHERE slug = 'instrumental';

-- ─────────────────────────────────────────────────────────────────────────────
-- 4. IMAGE CREATOR  (text-to-image)
-- ─────────────────────────────────────────────────────────────────────────────
UPDATE studio_tools SET
  ui_template = 'image-creator',
  ui_config   = '{
    "prompt_placeholder": "Describe the image you want to create in detail…",
    "aspect_ratios": [
      {"label":"Square",    "value":"1:1",   "icon":"square"},
      {"label":"Portrait",  "value":"9:16",  "icon":"portrait"},
      {"label":"Landscape", "value":"16:9",  "icon":"landscape"},
      {"label":"Wide",      "value":"3:2",   "icon":"wide"}
    ],
    "default_aspect": "1:1",
    "style_tags": ["Photorealistic","Cinematic","Oil Painting","Watercolour","Anime","Sketch","Digital Art","Fantasy","Vintage"],
    "show_negative_prompt": true,
    "negative_prompt_placeholder": "What to avoid (e.g. blurry, watermark, extra fingers)…"
  }'::jsonb
WHERE slug IN ('ai-photo-pro', 'ai-photo-max', 'ai-photo-dream');

-- ─────────────────────────────────────────────────────────────────────────────
-- 5. IMAGE EDITOR  (upload-first, instruction prompt)
-- ─────────────────────────────────────────────────────────────────────────────
UPDATE studio_tools SET
  ui_template = 'image-editor',
  ui_config   = '{
    "upload_label": "Upload the photo you want to edit",
    "upload_accept": ["image/png","image/jpeg","image/webp"],
    "prompt_placeholder": "Describe what to change (e.g. Make the background a beach at sunset, remove the person on the left)…",
    "style_tags": ["Realistic","Artistic","Minimalist","Vintage","Neon"],
    "show_style_tags": true,
    "max_file_mb": 10
  }'::jsonb
WHERE slug = 'photo-editor';

-- ─────────────────────────────────────────────────────────────────────────────
-- 6. VIDEO CREATOR  (text-to-video)
-- ─────────────────────────────────────────────────────────────────────────────
UPDATE studio_tools SET
  ui_template = 'video-creator',
  ui_config   = '{
    "prompt_placeholder": "Describe the scene, subject, and motion (e.g. A woman walking along a Lagos beach at golden hour, camera slowly panning right, cinematic 4K style)…",
    "aspect_ratios": [
      {"label":"Landscape 16:9", "value":"16:9"},
      {"label":"Portrait 9:16",  "value":"9:16"}
    ],
    "default_aspect": "16:9",
    "duration_options": [5, 8, 10],
    "default_duration": 5,
    "style_tags": ["Cinematic","Documentary","Realistic","Anime","Fantasy","Noir"],
    "show_negative_prompt": true,
    "negative_prompt_placeholder": "What to avoid (e.g. text overlays, blur, distortion)…",
    "generation_warning": "Video generation takes 2–5 minutes. You will be notified when ready."
  }'::jsonb
WHERE slug IN ('video-veo', 'video-premium');

-- ─────────────────────────────────────────────────────────────────────────────
-- 7. VIDEO ANIMATOR  (image-to-video)
-- ─────────────────────────────────────────────────────────────────────────────
UPDATE studio_tools SET
  ui_template = 'video-animator',
  ui_config   = '{
    "upload_label": "Upload the image or photo to animate",
    "upload_accept": ["image/png","image/jpeg","image/webp"],
    "max_file_mb": 20,
    "prompt_placeholder": "Describe the motion (e.g. Camera slowly zooms in, wind gently blows through the trees, subject turns to face camera)…",
    "aspect_ratios": [
      {"label":"Landscape 16:9", "value":"16:9"},
      {"label":"Portrait 9:16",  "value":"9:16"}
    ],
    "default_aspect": "16:9",
    "duration_options": [5, 8, 10],
    "default_duration": 5,
    "generation_warning": "Video generation takes 2–5 minutes. You will be notified when ready."
  }'::jsonb
WHERE slug IN ('video-cinematic', 'animate-photo', 'video-jingle');

-- ─────────────────────────────────────────────────────────────────────────────
-- 8. VOICE STUDIO  (TTS / narration)
-- ─────────────────────────────────────────────────────────────────────────────
UPDATE studio_tools SET
  ui_template = 'voice-studio',
  ui_config   = '{
    "prompt_placeholder": "Enter the text you want to narrate (up to 5,000 characters)…",
    "max_chars": 5000,
    "voices": [
      {"id":"alloy",   "name":"Alloy",   "tone":"Neutral & Clear",   "category":"Conversational"},
      {"id":"echo",    "name":"Echo",    "tone":"Deep & Warm",        "category":"Narration"},
      {"id":"fable",   "name":"Fable",   "tone":"Expressive & Lively","category":"Storytelling"},
      {"id":"onyx",    "name":"Onyx",    "tone":"Deep & Authoritative","category":"Broadcast"},
      {"id":"nova",    "name":"Nova",    "tone":"Friendly & Warm",    "category":"Social Media"},
      {"id":"shimmer", "name":"Shimmer", "tone":"Soft & Soothing",    "category":"Meditation"},
      {"id":"ash",     "name":"Ash",     "tone":"Gentle & Calm",      "category":"Education"},
      {"id":"ballad",  "name":"Ballad",  "tone":"Smooth & Musical",   "category":"Entertainment"},
      {"id":"coral",   "name":"Coral",   "tone":"Warm & Natural",     "category":"Podcasts"},
      {"id":"sage",    "name":"Sage",    "tone":"Clear & Professional","category":"Corporate"},
      {"id":"verse",   "name":"Verse",   "tone":"Dynamic & Engaging", "category":"Advertisement"},
      {"id":"willow",  "name":"Willow",  "tone":"Soft & Thoughtful",  "category":"Audiobooks"},
      {"id":"jessica", "name":"Jessica", "tone":"Bright & Upbeat",    "category":"Characters"}
    ],
    "default_voice": "nova",
    "languages": [
      {"code":"en","label":"English"},
      {"code":"yo","label":"Yoruba"},
      {"code":"ha","label":"Hausa"},
      {"code":"ig","label":"Igbo"},
      {"code":"fr","label":"French"},
      {"code":"pt","label":"Portuguese"},
      {"code":"es","label":"Spanish"}
    ],
    "default_language": "en"
  }'::jsonb
WHERE slug = 'narrate-pro';

-- ─────────────────────────────────────────────────────────────────────────────
-- 9. TRANSCRIBE  (audio upload → text)
-- ─────────────────────────────────────────────────────────────────────────────
UPDATE studio_tools SET
  ui_template = 'transcribe',
  ui_config   = '{
    "upload_label": "Upload your audio or voice recording",
    "upload_accept": ["audio/mp3","audio/mpeg","audio/wav","audio/m4a","audio/ogg","audio/flac"],
    "max_file_mb": 100,
    "max_duration_mins": 120,
    "languages": [
      {"code":"auto","label":"Auto-detect"},
      {"code":"en",  "label":"English"},
      {"code":"yo",  "label":"Yoruba"},
      {"code":"ha",  "label":"Hausa"},
      {"code":"ig",  "label":"Igbo"},
      {"code":"fr",  "label":"French"},
      {"code":"pcm", "label":"Nigerian Pidgin"}
    ],
    "default_language": "auto",
    "show_speaker_labels": true,
    "output_hint": "Your transcript will appear here as plain text. You can copy or download it."
  }'::jsonb
WHERE slug IN ('transcribe', 'transcribe-african');

-- ─────────────────────────────────────────────────────────────────────────────
-- 10. VISION / ASK MY PHOTO
-- ─────────────────────────────────────────────────────────────────────────────
UPDATE studio_tools SET
  ui_template = 'vision-ask',
  ui_config   = '{
    "upload_label": "Upload the image to analyse",
    "upload_accept": ["image/png","image/jpeg","image/webp","image/gif"],
    "max_file_mb": 20,
    "prompt_placeholder": "What would you like to know about this image? (e.g. What objects can you see? What text is written here?)",
    "prompt_optional": false
  }'::jsonb
WHERE slug IN ('image-analyser', 'ask-my-photo');

-- ─────────────────────────────────────────────────────────────────────────────
-- 11. KNOWLEDGE / DOCUMENT  (all text-output tools)
-- ─────────────────────────────────────────────────────────────────────────────

-- Generic knowledge tools
UPDATE studio_tools SET
  ui_template = 'knowledge-doc',
  ui_config   = '{
    "fields": [
      {"key":"topic","label":"Topic","type":"textarea","required":true,
       "placeholder":"Enter the topic or subject you want to learn about…",
       "rows": 3}
    ],
    "output_format": "text",
    "output_hint": "Your result will appear below."
  }'::jsonb
WHERE slug IN ('study-guide', 'quiz', 'mindmap', 'infographic', 'translate');

-- Research Brief — adds depth/source options
UPDATE studio_tools SET
  ui_template = 'knowledge-doc',
  ui_config   = '{
    "fields": [
      {"key":"topic","label":"Research Topic","type":"textarea","required":true,
       "placeholder":"e.g. The impact of AI on job markets in West Africa",
       "rows": 3},
      {"key":"depth","label":"Depth","type":"select","required":false,
       "options":["Overview (1-2 pages)","Detailed (3-5 pages)","Comprehensive (5-10 pages)"],
       "default":"Detailed (3-5 pages)"}
    ],
    "output_format": "text",
    "output_hint": "Your research brief will be formatted with sections, sources, and key findings."
  }'::jsonb
WHERE slug = 'research-brief';

-- Business Plan — structured multi-field form
UPDATE studio_tools SET
  ui_template = 'knowledge-doc',
  ui_config   = '{
    "fields": [
      {"key":"company",  "label":"Company / Business Name",  "type":"text",    "required":true,  "placeholder":"e.g. NexaFarm Ltd"},
      {"key":"industry", "label":"Industry",                 "type":"select",  "required":true,
       "options":["Technology","Agriculture","Healthcare","Finance & Fintech","Education","Retail & E-commerce","Manufacturing","Real Estate","Media & Entertainment","Logistics","Energy","Other"]},
      {"key":"market",   "label":"Target Market",            "type":"text",    "required":true,  "placeholder":"e.g. Smallholder farmers in Southern Nigeria"},
      {"key":"stage",    "label":"Business Stage",           "type":"select",  "required":true,
       "options":["Idea / Pre-revenue","Early Stage (0-1 years)","Growth Stage (1-3 years)","Established (3+ years)"]},
      {"key":"goal",     "label":"Main Goal / Problem Solved","type":"textarea","required":true,  "placeholder":"Briefly describe the problem your business solves…","rows":2}
    ],
    "output_format": "document",
    "output_hint": "A full business plan (executive summary, market analysis, financials, and roadmap) will be generated."
  }'::jsonb
WHERE slug = 'bizplan';

-- Slide Deck
UPDATE studio_tools SET
  ui_template = 'knowledge-doc',
  ui_config   = '{
    "fields": [
      {"key":"topic",  "label":"Presentation Topic", "type":"textarea","required":true,
       "placeholder":"e.g. Why our loyalty app is the future of customer retention in Nigeria",
       "rows":2},
      {"key":"slides", "label":"Number of Slides",   "type":"select","required":false,
       "options":["5 slides","8 slides","12 slides","15 slides","20 slides"],
       "default":"12 slides"},
      {"key":"style",  "label":"Presentation Style", "type":"select","required":false,
       "options":["Professional / Corporate","Creative / Bold","Minimal / Clean","Academic / Research"],
       "default":"Professional / Corporate"}
    ],
    "output_format": "document",
    "output_hint": "Slide outlines with titles, bullet points, and speaker notes will be generated."
  }'::jsonb
WHERE slug = 'slide-deck';

-- Podcast
UPDATE studio_tools SET
  ui_template = 'knowledge-doc',
  ui_config   = '{
    "fields": [
      {"key":"topic",    "label":"Podcast Topic",   "type":"textarea","required":true,
       "placeholder":"e.g. The rise of mobile payments in West Africa",
       "rows":2},
      {"key":"duration", "label":"Duration",        "type":"select","required":false,
       "options":["3 minutes","5 minutes","8 minutes","12 minutes","20 minutes"],
       "default":"8 minutes"},
      {"key":"style",    "label":"Style",           "type":"select","required":false,
       "options":["Solo host","Interview (2 speakers)","Debate (2 opinions)","Documentary"],
       "default":"Solo host"}
    ],
    "output_format": "audio",
    "output_hint": "A full podcast script will be generated and narrated as an audio file."
  }'::jsonb
WHERE slug = 'podcast';
