-- =============================================================================
-- Migration 120: Idempotent catch-all for stale-skipped migrations 113-117
-- Definitive v4 (2026-05-24): all column references validated against live schema.
--   * provider_key removed (column absent in ai_provider_configs)
--   * draws filtered by name only (no description column in draws table)
--   * studio_tools.parameters phantom column removed; voices via ui_config JSONB
--   * program_configs guarded with DO block (table absent on live DB)
-- =============================================================================

-- ---------------------------------------------------------------------------
-- From 113: Platform health fixes
-- ---------------------------------------------------------------------------

-- Disable broken AI Studio tools
UPDATE studio_tools
SET    is_active  = FALSE,
       updated_at = NOW()
WHERE  slug IN ('video-cinematic', 'video-story', 'bg-remover')
AND    is_active = TRUE;

-- Re-enable ElevenLabs TTS provider (name+slug only, no provider_key column)
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

-- Fix airtime prize names to match base_value (stored in kobo)
UPDATE prize_pool
SET    name       = chr(8358) || (base_value / 100)::TEXT || ' Airtime',
       updated_at = NOW()
WHERE  prize_type = 'airtime'
AND    base_value > 0
AND    name NOT LIKE '%' || (base_value / 100)::TEXT || '%';

-- Convert data bundle byte counts to MB if accidentally stored as bytes
UPDATE prize_pool
SET    base_value = base_value / 1048576,
       name       = (base_value / 1048576)::TEXT || 'MB Data',
       updated_at = NOW()
WHERE  prize_type = 'data'
AND    base_value > 1000000;

-- Remove orphan spin-limit config keys
-- Guarded: program_configs table does not exist on live DB
DO $$
BEGIN
  DELETE FROM program_configs
  WHERE  key IN ('spin_max_per_day', 'spin_max_per_user_per_day');
EXCEPTION WHEN undefined_table THEN
  NULL;
END $$;

UPDATE network_configs
SET    value      = '-- UNUSED: spin limits set in spin_tiers, not here --',
       updated_at = NOW()
WHERE  key IN ('spin_max_per_day', 'spin_max_per_user_per_day');

-- ---------------------------------------------------------------------------
-- From 114: Data quality + template consistency
-- ---------------------------------------------------------------------------

-- Normalise ui_template PascalCase/kebab-case -> snake_case
UPDATE studio_tools SET ui_template = 'image_creator'     WHERE ui_template IN ('ImageCreator',    'image-creator');
UPDATE studio_tools SET ui_template = 'voice_studio'      WHERE ui_template IN ('VoiceStudio',     'voice-studio');
UPDATE studio_tools SET ui_template = 'video_creator'     WHERE ui_template IN ('VideoCreator',    'video-creator');
UPDATE studio_tools SET ui_template = 'knowledge_doc'     WHERE ui_template IN ('KnowledgeDoc',    'knowledge-doc');
UPDATE studio_tools SET ui_template = 'music_composer'    WHERE ui_template IN ('MusicComposer',   'music-composer');
UPDATE studio_tools SET ui_template = 'code_helper'       WHERE ui_template IN ('CodeHelper',      'code-helper');
UPDATE studio_tools SET ui_template = 'nexus_chat'        WHERE ui_template IN ('NexusChat',       'nexus-chat');
UPDATE studio_tools SET ui_template = 'transcribe'        WHERE ui_template IN ('Transcribe');
UPDATE studio_tools SET ui_template = 'vision_ask'        WHERE ui_template IN ('VisionAsk',       'vision-ask');
UPDATE studio_tools SET ui_template = 'image_editor'      WHERE ui_template IN ('ImageEditor',     'image-editor');
UPDATE studio_tools SET ui_template = 'voice_to_plan'     WHERE ui_template IN ('VoiceToPlan',     'voice-to-plan');
UPDATE studio_tools SET ui_template = 'build_tool'        WHERE ui_template IN ('BuildTool',       'build-tool');
UPDATE studio_tools SET ui_template = 'code_pro'          WHERE ui_template IN ('CodePro',         'code-pro');
UPDATE studio_tools SET ui_template = 'my_ai_photo'       WHERE ui_template IN ('MyAiPhoto',       'my-ai-photo');
UPDATE studio_tools SET ui_template = 'video_multi_scene' WHERE ui_template IN ('VideoMultiScene', 'video-multi-scene', 'VideoMultiscene');
UPDATE studio_tools SET ui_template = 'video_animator'    WHERE ui_template IN ('VideoAnimator',   'video-animator');

-- Deactivate test/demo draws (name-only filter; no description column in draws)
UPDATE draws
SET    status = 'CANCELLED'
WHERE  (LOWER(name) LIKE '%test%' OR LOWER(name) LIKE '%demo%')
AND    status NOT IN ('COMPLETED', 'CANCELLED');

-- Restore Pollinations TTS voice options via ui_config JSONB (correct column, mig 032)
-- studio_tools.parameters does not exist; ui_config is the only JSONB config column
UPDATE studio_tools
SET    ui_config  = COALESCE(ui_config, '{}'::jsonb) || jsonb_build_object(
                      'voices', jsonb_build_array(
                        jsonb_build_object('id', 'Cherry', 'label', 'Cherry (Female, Friendly)'),
                        jsonb_build_object('id', 'Serena', 'label', 'Serena (Female, Professional)'),
                        jsonb_build_object('id', 'Ethan',  'label', 'Ethan  (Male, Clear)')
                      )
                    ),
       updated_at = NOW()
WHERE  slug IN ('voice-studio', 'narrate-pro', 'ai-podcast', 'my-podcast')
AND    (ui_config IS NULL
        OR ui_config->'voices' IS NULL
        OR NOT (ui_config->>'voices' LIKE '%Cherry%'));

-- ---------------------------------------------------------------------------
-- From 115: Normalise ui_template snake_case -> kebab-case
-- ---------------------------------------------------------------------------

UPDATE studio_tools SET ui_template = 'knowledge-doc'      WHERE ui_template = 'knowledge_doc';
UPDATE studio_tools SET ui_template = 'image-creator'      WHERE ui_template = 'image_creator';
UPDATE studio_tools SET ui_template = 'voice-studio'       WHERE ui_template = 'voice_studio';
UPDATE studio_tools SET ui_template = 'video-creator'      WHERE ui_template = 'video_creator';
UPDATE studio_tools SET ui_template = 'music-composer'     WHERE ui_template = 'music_composer';
UPDATE studio_tools SET ui_template = 'code-helper'        WHERE ui_template = 'code_helper';
UPDATE studio_tools SET ui_template = 'nexus-chat'         WHERE ui_template = 'nexus_chat';
UPDATE studio_tools SET ui_template = 'vision-ask'         WHERE ui_template = 'vision_ask';
UPDATE studio_tools SET ui_template = 'image-editor'       WHERE ui_template = 'image_editor';
UPDATE studio_tools SET ui_template = 'voice-to-plan'      WHERE ui_template = 'voice_to_plan';
UPDATE studio_tools SET ui_template = 'build-tool'         WHERE ui_template = 'build_tool';
UPDATE studio_tools SET ui_template = 'code-pro'           WHERE ui_template = 'code_pro';
UPDATE studio_tools SET ui_template = 'my-ai-photo'        WHERE ui_template = 'my_ai_photo';
UPDATE studio_tools SET ui_template = 'video-multi-scene'  WHERE ui_template = 'video_multi_scene';
UPDATE studio_tools SET ui_template = 'video-animator'     WHERE ui_template = 'video_animator';

-- ---------------------------------------------------------------------------
-- From 116: Remove contradictory spin limit config key
-- ---------------------------------------------------------------------------

DELETE FROM network_configs
WHERE  key = 'spin_max_per_user_per_day';

-- ---------------------------------------------------------------------------
-- From 117: Prize pool corrections
-- ---------------------------------------------------------------------------

UPDATE prize_pool
SET    name = chr(8358) || TRIM(TO_CHAR(base_value / 100.0, 'FM999999990.##')) || ' Airtime'
WHERE  LOWER(prize_type) = 'airtime'
AND    is_active = TRUE
AND    base_value > 0
AND    name <> chr(8358) || TRIM(TO_CHAR(base_value / 100.0, 'FM999999990.##')) || ' Airtime';

UPDATE prize_pool
SET    base_value = 10
WHERE  LOWER(prize_type) = 'data_bundle'
AND    LOWER(name) LIKE '%10mb%'
AND    base_value NOT BETWEEN 9 AND 11;

-- Deduplicate active prizes by name (keep oldest)
UPDATE prize_pool
SET    is_active = FALSE
WHERE  id IN (
  SELECT id FROM (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY LOWER(name) ORDER BY id ASC) AS rn
    FROM   prize_pool
    WHERE  is_active = TRUE
  ) d
  WHERE  rn > 1
);

-- Normalise weights and ensure they sum to exactly 100.00
DO $$
DECLARE
  current_sum NUMERIC(10,2);
  target_id   UUID;
  adjustment  NUMERIC(10,2);
BEGIN
  -- Rescale any weight stored as basis points (>100)
  UPDATE prize_pool
  SET    win_probability_weight = ROUND(win_probability_weight / 100.0, 2)
  WHERE  is_active = TRUE
  AND    win_probability_weight > 100;

  SELECT COALESCE(SUM(win_probability_weight), 0)
  INTO   current_sum
  FROM   prize_pool
  WHERE  is_active = TRUE;

  IF current_sum <> 100.00 THEN
    SELECT id INTO target_id
    FROM   prize_pool
    WHERE  is_active = TRUE
    ORDER BY CASE WHEN LOWER(prize_type) = 'try_again' THEN 0 ELSE 1 END,
             win_probability_weight DESC,
             id ASC
    LIMIT 1;

    adjustment := ROUND(100.00 - current_sum, 2);
    UPDATE prize_pool
    SET    win_probability_weight = ROUND(win_probability_weight + adjustment, 2)
    WHERE  id = target_id;
  END IF;
END $$;
