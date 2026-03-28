-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 038 — Fix three slugs that had no ui_template set
--
-- Root cause: slugs ai-photo, bg-remover and narrate were originally inserted
-- by migration 026 but never covered in migration 032's UPDATE blocks, so they
-- fell through to the DEFAULT 'knowledge-doc' — giving users a plain text box
-- instead of the correct purpose-built UI template.
-- ─────────────────────────────────────────────────────────────────────────────

-- ── 1. ai-photo (basic) — same template as ai-photo-pro/max/dream ─────────────
--    Dispatches to dispatchImage → FAL.AI FLUX → Pollinations
UPDATE studio_tools SET
  ui_template = 'image-creator',
  ui_config   = '{
    "prompt_placeholder": "Describe the image you want to create…",
    "aspect_ratios": [
      {"label":"Square",    "icon":"⬛", "value":"1024x1024", "default":true},
      {"label":"Portrait",  "icon":"📱", "value":"768x1344"},
      {"label":"Landscape", "icon":"🖥️", "value":"1344x768"},
      {"label":"Wide",      "icon":"🎬", "value":"1920x1080"}
    ],
    "style_tags": ["Photorealistic","Cinematic","Oil Painting","Anime","Sketch","Watercolour","Neon","Vintage"],
    "show_negative_prompt": true,
    "show_quality_toggle": false,
    "prompt_optional": false,
    "max_prompt_chars": 1000
  }'::jsonb
WHERE slug = 'ai-photo';

-- ── 2. bg-remover — image-editor (upload-first) ───────────────────────────────
--    Dispatches to dispatchBgRemover → rembg → FAL BiRefNet → remove.bg
UPDATE studio_tools SET
  ui_template = 'image-editor',
  ui_config   = '{
    "upload_label":   "Upload the photo to remove background from",
    "upload_accept":  ["image/png","image/jpeg","image/webp"],
    "upload_hint":    "Supports JPG, PNG and WebP up to 10 MB",
    "prompt_label":   null,
    "prompt_optional": true,
    "show_edit_prompt": false,
    "edit_suggestions": [],
    "output_note": "Background will be removed automatically — no prompt needed"
  }'::jsonb
WHERE slug = 'bg-remover';

-- ── 3. narrate (basic) — same template as narrate-pro ────────────────────────
--    Dispatches to dispatchTTS → Google Cloud TTS → Pollinations TTS
UPDATE studio_tools SET
  ui_template = 'voice-studio',
  ui_config   = '{
    "prompt_placeholder": "Enter the text you want to narrate (up to 3,000 characters)…",
    "max_chars": 3000,
    "voices": [
      {"id":"alloy",   "name":"Alloy",   "tone":"Neutral & Clear",    "category":"Conversational"},
      {"id":"echo",    "name":"Echo",    "tone":"Deep & Warm",         "category":"Narration"},
      {"id":"fable",   "name":"Fable",   "tone":"Expressive & Lively", "category":"Storytelling"},
      {"id":"onyx",    "name":"Onyx",    "tone":"Deep & Authoritative","category":"Broadcast"},
      {"id":"nova",    "name":"Nova",    "tone":"Friendly & Warm",     "category":"Social Media"},
      {"id":"shimmer", "name":"Shimmer", "tone":"Soft & Soothing",     "category":"Meditation"},
      {"id":"ash",     "name":"Ash",     "tone":"Gentle & Calm",       "category":"Education"},
      {"id":"coral",   "name":"Coral",   "tone":"Warm & Natural",      "category":"Podcasts"},
      {"id":"sage",    "name":"Sage",    "tone":"Clear & Professional","category":"Corporate"}
    ],
    "default_voice": "nova",
    "languages": [
      {"code":"en",  "label":"English"},
      {"code":"yo",  "label":"Yoruba"},
      {"code":"ha",  "label":"Hausa"},
      {"code":"ig",  "label":"Igbo"},
      {"code":"fr",  "label":"French"},
      {"code":"pt",  "label":"Portuguese"},
      {"code":"es",  "label":"Spanish"}
    ],
    "default_language": "en",
    "show_speed_control": false,
    "show_format_selector": false
  }'::jsonb
WHERE slug = 'narrate';
