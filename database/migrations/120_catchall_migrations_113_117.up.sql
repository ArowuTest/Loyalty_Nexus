-- =============================================================================
-- Migration 120: Idempotent catch-all for stale-skipped migrations 113-117
-- =============================================================================
-- Definitive v3 — all column references confirmed against live schema:
--   ai_provider_configs: id, name, slug, category, template, env_key,
--     api_key_enc, model_id, extra_config, priority, is_primary, is_active,
--     cost_micros, pulse_pts, notes, last_tested_at, last_test_ok,
--     last_test_msg, created_at, updated_at.
--     NO provider_key column.
--   studio_tools: slug, ui_template, ui_config (JSONB).
--     NO parameters column.
--   draws: id, name, status, draw_type, start_time, end_time, etc.
--     NO description column.
-- =============================================================================

-- ─────────────────────────────────────────────────────────────────────────────
-- From 113: Platform health fixes
-- ─────────────────────────────────────────────────────────────────────────────

-- Disable broken AI Studio tools
UPDATE studio_tools
SET    is_active  = FALSE,
       updated_at = NOW()
WHERE  slug IN ('video-cinematic', 'video-story', 'bg-remover')
AND    is_active = TRUE;

-- Re-enable ElevenLabs TTS provider (name + slug only — no provider_key column)
UPDATE ai_provider_configs
SET    is_active  = TRUE,
       updated_at = NOW()
WHERE  (LOWER(name) LIKE '%elevenlabs%' OR LOWER(slug) LIKE '%elevenlabs%')
AND    is_active = FALSE;

-- Disable Google TTS stub
UPDATE ai_provider_configs
SET    is_active  = FALSE,
       updated_at = NOW()
WHERE  (LOWER(name) LIKE '%google%tts%' OR LOWER(name) LIKE '%cloud tts%'
        OR LOWER(slug) LIKE '%google-tts%' OR LOWER(slug) LIKE '%google_tts%')
AND    is_active = TRUE;

-- MTN-only platform: deactivate non-MTN networks
UPDATE network_operator_configs
SET    is_active  = FALSE,
       updated_at = NOW()
WHERE  UPPER(network_code) != 'MTN'
AND    is_active = TRUE;

-- Fix airtime prize names to match base_value (kobo)
UPDATE prize_pool
SET    name       = '₦' || (base_value / 100)::TEXT || ' Airtime',
       updated_at = NOW()
WHERE  prize_type  = 'airtime'
AND    base_value  > 0
AND    name        NOT LIKE '%' || (base_value / 100)::TEXT || '%';

-- Convert data bundle byte counts to MB if set as bytes
UPDATE prize_pool
SET    base_value  = base_value / 1048576,
       name        = (base_value / 1048576)::TEXT || 'MB Data',
       updated_at  = NOW()
WHERE  prize_type  = 'data'
AND    base_value  > 1000000;

-- Remove orphan spin-limit config keys
DELETE FROM program_configs
WHERE  key IN ('spin_max_per_day', 'spin_max_per_user_per_day');

UPDATE network_configs
SET    value      = '-- UNUSED: spin limits are set in spin_tiers table --',
       updated_at = NOW()
WHERE  key IN ('spin_max_per_day', 'spin_max_per_user_per_day');

-- ─────────────────────────────────────────────────────────────────────────────
-- From 114: Data quality + template consistency
-- ─────────────────────────────────────────────────────────────────────────────

-- Normalise ui_template PascalCase → snake_case (ui_template added in mig 032)
UPDATE studio_tools SET ui_template = 'image_creator'     WHERE ui_template IN ('ImageCreator',   'image-creator');
UPDATE studio_tools SET ui_template = 'voice_studio'      WHERE ui_template IN ('VoiceStudio',    'voice-studio');
UPDATE studio_tools SET ui_template = 'video_creator'     WHERE ui_template IN ('VideoCreator',   'video-creator');
UPDATE studio_tools SET ui_template = 'knowledge_doc'     WHERE ui_template IN ('KnowledgeDoc',   'knowledge-doc');
UPDATE studio_tools SET ui_template = 'music_composer'    WHERE ui_template IN ('MusicComposer',  'music-composer');
UPDATE studio_tools SET ui_template = 'code_helper'       WHERE ui_template IN ('CodeHelper',     'code-helper');
UPDATE studio_tools SET ui_template = 'nexus_chat'        WHERE ui_template IN ('NexusChat',      'nexus-chat');
UPDATE studio_tools SET ui_template = 'transcribe'        WHERE ui_template IN ('Transcribe');
UPDATE studio_tools SET ui_template = 'vision_ask'        WHERE ui_template IN ('VisionAsk',      'vision-ask');
UPDATE studio_tools SET ui_template = 'image_editor'      WHERE ui_template IN ('ImageEditor',    'image-editor');
UPDATE studio_tools SET ui_template = 'voice_to_plan'     WHERE ui_template IN ('VoiceToPlan',    'voice-to-plan');
UPDATE studio_tools SET ui_template = 'build_tool'        WHERE ui_template IN ('BuildTool',      'build-tool');
UPDATE studio_tools SET ui_template = 'code_pro'          WHERE ui_template IN ('CodePro',        'code-pro');
UPDATE studio_tools SET ui_template = 'my_ai_photo'       WHERE ui_template IN ('MyAiPhoto',      'my-ai-photo');
UPDATE studio_tools SET ui_template = 'video_multi_scene' WHERE ui_template IN ('VideoMultiScene','video-multi-scene','VideoMultiscene');
UPDATE studio_tools SET ui_template = 'video_animator'    WHERE ui_template IN ('VideoAnimator',  'video-animator');

-- Deactivate test/demo draws (name only — draws.description does not exist)
UPDATE draws
   SET status = 'CANCELLED'
 WHERE (LOWER(name) LIKE '%test%' OR LOWER(name) LIKE '%demo%')
   AND status NOT IN ('COMPLETED', 'CANCELLED');

-- Replace OpenAI voice IDs in ui_config with Pollinations TTS voices.
-- Investigation findings:
--   - "parameters" column does not exist in studio_tools. The real JSONB
--     column is "ui_config" (added in migration 032).
--   - The voice list lives at ui_config->"voices" (not "voice_options").
--   - Current ui_config.voices has OpenAI IDs (alloy, echo, fable...).
--   - Backend TTS chain uses Pollinations (Cherry/Serena/Ethan) and
--     ElevenLabs — not OpenAI. Replacing so UI and backend agree.
UPDATE studio_tools
   SET ui_config   = ui_config || '{
     "voices": [
       {"id": "Cherry",  "name": "Cherry",  "tone": "Female, Friendly",      "category": "Conversational"},
       {"id": "Serena",  "name": "Serena",  "tone": "Female, Professional",  "category": "Narration"},
       {"id": "Ethan",   "name": "Ethan",   "tone": "Male, Clear",           "category": "Broadcast"}
     ],
     "default_voice": "Cherry"
   }'::jsonb,
       updated_at  = NOW()
 WHERE slug IN ('voice-studio', 'narrate-pro', 'ai-podcast', 'my-podcast')
   AND ui_config::text LIKE '%alloy%';

-- ─────────────────────────────────────────────────────────────────────────────
-- From 115: Normalise ui_template snake_case → kebab-case
-- ─────────────────────────────────────────────────────────────────────────────

UPDATE studio_tools SET ui_template = 'knowledge-doc'      WHERE ui_template IN ('knowledge_doc');
UPDATE studio_tools SET ui_template = 'image-creator'      WHERE ui_template IN ('image_creator');
UPDATE studio_tools SET ui_template = 'voice-studio'       WHERE ui_template IN ('voice_studio');
UPDATE studio_tools SET ui_template = 'video-creator'      WHERE ui_template IN ('video_creator');
UPDATE studio_tools SET ui_template = 'music-composer'     WHERE ui_template IN ('music_composer');
UPDATE studio_tools SET ui_template = 'code-helper'        WHERE ui_template IN ('code_helper');
UPDATE studio_tools SET ui_template = 'nexus-chat'         WHERE ui_template IN ('nexus_chat');
UPDATE studio_tools SET ui_template = 'vision-ask'         WHERE ui_template IN ('vision_ask');
UPDATE studio_tools SET ui_template = 'image-editor'       WHERE ui_template IN ('image_editor');
UPDATE studio_tools SET ui_template = 'voice-to-plan'      WHERE ui_template IN ('voice_to_plan');
UPDATE studio_tools SET ui_template = 'build-tool'         WHERE ui_template IN ('build_tool');
UPDATE studio_tools SET ui_template = 'code-pro'           WHERE ui_template IN ('code_pro');
UPDATE studio_tools SET ui_template = 'my-ai-photo'        WHERE ui_template IN ('my_ai_photo');
UPDATE studio_tools SET ui_template = 'video-multi-scene'  WHERE ui_template IN ('video_multi_scene');
UPDATE studio_tools SET ui_template = 'video-animator'     WHERE ui_template IN ('video_animator');

-- ─────────────────────────────────────────────────────────────────────────────
-- From 116: Remove contradictory spin limit config key
-- ─────────────────────────────────────────────────────────────────────────────

DELETE FROM network_configs WHERE key = 'spin_max_per_user_per_day';

-- ─────────────────────────────────────────────────────────────────────────────
-- From 117: Prize pool corrections
-- ─────────────────────────────────────────────────────────────────────────────

UPDATE prize_pool
SET    name = '₦' || TRIM(TO_CHAR(base_value / 100.0, 'FM999999990.##')) || ' Airtime'
WHERE  LOWER(prize_type) = 'airtime'
AND    is_active = TRUE AND base_value > 0
AND    name <> '₦' || TRIM(TO_CHAR(base_value / 100.0, 'FM999999990.##')) || ' Airtime';

UPDATE prize_pool SET base_value = 10
WHERE  LOWER(prize_type) = 'data_bundle'
AND    LOWER(name) LIKE '%10mb%'
AND    base_value NOT BETWEEN 9 AND 11;

UPDATE prize_pool SET is_active = FALSE
WHERE  id IN (
  SELECT id FROM (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY LOWER(name) ORDER BY id ASC) AS rn
    FROM prize_pool WHERE is_active = TRUE
  ) dedup WHERE rn > 1
);

DO $$
DECLARE
  current_sum NUMERIC(10,2);
  target_id   UUID;
  adjustment  NUMERIC(10,2);
BEGIN
  UPDATE prize_pool
  SET    win_probability_weight = ROUND(win_probability_weight / 100.0, 2)
  WHERE  is_active = TRUE AND win_probability_weight > 100;

  SELECT COALESCE(SUM(win_probability_weight), 0) INTO current_sum
  FROM   prize_pool WHERE is_active = TRUE;

  IF current_sum <> 100.00 THEN
    SELECT id INTO target_id FROM prize_pool WHERE is_active = TRUE
    ORDER BY CASE WHEN LOWER(prize_type) = 'try_again' THEN 0 ELSE 1 END,
             win_probability_weight DESC, id ASC LIMIT 1;
    adjustment := ROUND(100.00 - current_sum, 2);
    UPDATE prize_pool
    SET    win_probability_weight = ROUND(win_probability_weight + adjustment, 2)
    WHERE  id = target_id;
  END IF;
END $$;
