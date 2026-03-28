-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 033 — Studio UI Config v2
-- Extends ui_config for each template with the new controls added in
-- frontend template rewrites (BPM, energy, camera movements, speaker labels,
-- output format, speed, example questions, quality toggle, edit suggestions,
-- translate language list, max duration).
-- ─────────────────────────────────────────────────────────────────────────────

-- ── Music: song-creator / jingle / bg-music — add BPM + energy + longer durations ──
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "show_bpm": true,
    "show_energy": true,
    "max_duration": 300,
    "duration_options": [15, 30, 60, 120, 180, 300]
  }'::jsonb
WHERE slug IN ('song-creator', 'jingle', 'bg-music');

-- ── Instrumental — no vocals, extend to 300s ──
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "show_bpm": true,
    "show_energy": true,
    "max_duration": 300,
    "duration_options": [30, 60, 120, 180, 300],
    "show_vocals_toggle": false,
    "show_lyrics_box": false
  }'::jsonb
WHERE slug = 'instrumental';

-- ── Image Creator — add quality toggle for GPT-Image tools ──
UPDATE studio_tools SET
  ui_config = ui_config || '{"show_quality_toggle": true}'::jsonb
WHERE slug IN ('ai-photo-pro', 'ai-photo-max');

UPDATE studio_tools SET
  ui_config = ui_config || '{"show_quality_toggle": false}'::jsonb
WHERE slug IN ('ai-photo', 'ai-photo-dream');

-- ── Image Editor — add customisable edit suggestions ──
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "edit_suggestions": [
      "Remove the background",
      "Add sunset lighting",
      "Make it look like an oil painting",
      "Add dramatic shadows",
      "Convert to black & white",
      "Make colours more vibrant",
      "Add a smooth bokeh background",
      "Upscale & enhance sharpness",
      "Change background to a beach",
      "Add professional studio lighting",
      "Make it look futuristic",
      "Apply a vintage film filter"
    ]
  }'::jsonb
WHERE slug = 'photo-editor';

-- ── Video Creator — add camera movements + extended durations ──
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "max_duration": 30,
    "duration_options": [5, 8, 10, 15, 30],
    "camera_movements": [
      {"label":"Slow zoom in",  "icon":"🔍", "value":"slow zoom in"},
      {"label":"Slow zoom out", "icon":"🔭", "value":"slow zoom out"},
      {"label":"Pan left",      "icon":"⬅️", "value":"camera panning left"},
      {"label":"Pan right",     "icon":"➡️", "value":"camera panning right"},
      {"label":"Tilt up",       "icon":"⬆️", "value":"camera tilting up"},
      {"label":"Orbit shot",    "icon":"🔄", "value":"360 orbit around subject"},
      {"label":"Tracking",      "icon":"🎯", "value":"tracking shot following subject"},
      {"label":"Handheld",      "icon":"📷", "value":"handheld camera, slight shake"},
      {"label":"Static",        "icon":"📌", "value":"static camera, no movement"}
    ]
  }'::jsonb
WHERE slug IN ('video-veo', 'video-premium');

-- ── Video Animator — add duration options (already had style tags) ──
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "duration_options": [5, 8, 10],
    "default_duration": 5
  }'::jsonb
WHERE slug IN ('video-cinematic', 'animate-photo', 'video-jingle');

-- ── Voice Studio — add speed + format controls ──
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "show_speed_control": true,
    "show_format_selector": true
  }'::jsonb
WHERE slug IN ('narrate', 'narrate-pro');

-- ── Transcribe — add speaker labels + output format ──
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "show_speaker_labels": true,
    "show_output_format": true
  }'::jsonb
WHERE slug IN ('transcribe', 'transcribe-african');

-- ── Vision Ask — differentiate the two tools ──
-- image-analyser: auto mode (prompt optional = auto-describe)
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "prompt_optional": true,
    "upload_label": "Image to analyse",
    "example_questions": [
      "Describe this image in full detail",
      "What objects can you identify?",
      "What text is visible in this image?",
      "What is the colour palette?",
      "Are there any brand logos?",
      "What is the approximate location or setting?"
    ]
  }'::jsonb
WHERE slug = 'image-analyser';

-- ask-my-photo: question required
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "prompt_optional": false,
    "upload_label": "Upload your photo",
    "example_questions": [
      "What is the brand or product in this image?",
      "Can you read the text in this image?",
      "What emotions does this person appear to feel?",
      "Describe the outfit or style in detail",
      "What is happening in this scene?",
      "Is this image suitable for a professional profile?",
      "What improvements would you suggest for this photo?",
      "What type of food or dish is this?"
    ]
  }'::jsonb
WHERE slug = 'ask-my-photo';

-- ── Translate — add dedicated translate language list ──
UPDATE studio_tools SET
  ui_config = ui_config || '{
    "translate_languages": [
      {"code":"en",  "label":"English"},
      {"code":"fr",  "label":"French"},
      {"code":"es",  "label":"Spanish"},
      {"code":"pt",  "label":"Portuguese"},
      {"code":"de",  "label":"German"},
      {"code":"ar",  "label":"Arabic"},
      {"code":"zh",  "label":"Chinese"},
      {"code":"sw",  "label":"Swahili"},
      {"code":"yo",  "label":"Yoruba"},
      {"code":"ha",  "label":"Hausa"},
      {"code":"ig",  "label":"Igbo"},
      {"code":"pcm", "label":"Nigerian Pidgin"},
      {"code":"af",  "label":"Afrikaans"}
    ],
    "prompt_placeholder": "Paste or type the text you want to translate…"
  }'::jsonb
WHERE slug = 'translate';
