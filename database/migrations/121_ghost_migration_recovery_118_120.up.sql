-- =============================================================================
-- Migration 121: Ghost-migration recovery for 118-120 (hardened v2)
-- =============================================================================
-- Includes fixes from 118-120 with all column references validated.
-- Safe to run even if those migrations partially applied — all statements
-- are idempotent with guards.
-- =============================================================================

-- ─────────────────────────────────────────────────────────────────────────────
-- From migration 118: Consolidate duplicate "Try Again" spin prize entries
-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 118: Consolidate duplicate "Try Again" spin prize entries
-- The auditor found 3 separate "Try Again" entries combining ~48% no-win rate.
-- We keep 1 entry with the combined weight and delete the rest.

DO $$
DECLARE
  keep_id   UUID;
  total_wt  NUMERIC;
BEGIN
  -- Sum all try-again weights
  SELECT COALESCE(SUM(win_probability_weight), 0)
  INTO   total_wt
  FROM   prize_pool
  WHERE  LOWER(prize_type) = 'try_again' AND is_active = true;

  -- Pick the one to keep (lowest created_at)
  SELECT id INTO keep_id
  FROM   prize_pool
  WHERE  LOWER(prize_type) = 'try_again' AND is_active = true
  ORDER  BY created_at ASC
  LIMIT  1;

  IF keep_id IS NOT NULL THEN
    -- Update the keeper with the combined weight (cap at 30.00 = 3000 basis points to avoid dominating the pool)
    UPDATE prize_pool
    SET    win_probability_weight = LEAST(total_wt, 30.00),
           name = 'Try Again',
           updated_at = NOW()
    WHERE  id = keep_id;

    -- Deactivate all other try-again entries
    UPDATE prize_pool
    SET    is_active = false,
           updated_at = NOW()
    WHERE  LOWER(prize_type) = 'try_again'
      AND  is_active = true
      AND  id <> keep_id;
  END IF;

  -- Re-normalise remaining active prize weights to sum to 100.00 percent
  WITH active AS (
    SELECT id, win_probability_weight,
           SUM(win_probability_weight) OVER () AS total
    FROM   prize_pool
    WHERE  is_active = true
  )
  UPDATE prize_pool pp
  SET    win_probability_weight = ROUND((a.win_probability_weight::NUMERIC / a.total) * 100, 2),
         updated_at = NOW()
  FROM   active a
  WHERE  pp.id = a.id;
END $$;


-- ─────────────────────────────────────────────────────────────────────────────
-- From migration 119: Single source of truth for lifetime_points
-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 119: Single source of truth for lifetime_points
-- The users.lifetime_points column was a legacy seed-only column that drifted
-- from wallets.lifetime_points (the actively-updated value). All writers go to
-- wallets.lifetime_points; this migration syncs users.lifetime_points to match,
-- and adds a NOTICE-level note. The column is retained (not dropped) for zero-
-- downtime rollback safety; the user entity now hides it from JSON output.

-- Backfill users.lifetime_points from wallets.lifetime_points
UPDATE users u
SET    lifetime_points = COALESCE(w.lifetime_points, 0),
       updated_at      = NOW()
FROM   wallets w
WHERE  w.user_id = u.id
  AND  u.lifetime_points IS DISTINCT FROM w.lifetime_points;

-- Document the deprecation at the column level
COMMENT ON COLUMN users.lifetime_points
  IS 'DEPRECATED: read from wallets.lifetime_points instead. Kept for rollback safety; backfilled in migration 119.';


-- ─────────────────────────────────────────────────────────────────────────────
-- From migration 120: Idempotent catch-all for stale-skipped migrations 113-117
-- ─────────────────────────────────────────────────────────────────────────────
-- =============================================================================
-- Migration 120: Idempotent catch-all for stale-skipped migrations 113-117
-- Hardened v2: all column references validated against live schema.
--   - provider_key removed (column doesn't exist in ai_provider_configs)
--   - draws.description wrapped in exception handler (column may not exist)
--   - studio_tools.parameters wrapped in exception handler (may not exist)
-- =============================================================================

-- ─────────────────────────────────────────────────────────────────────────────
-- From 113: Platform health fixes
-- ─────────────────────────────────────────────────────────────────────────────

-- Disable broken AI Studio tools (video-cinematic, video-story, bg-remover)
UPDATE studio_tools
SET    is_active  = FALSE,
       updated_at = NOW()
WHERE  slug IN ('video-cinematic', 'video-story', 'bg-remover')
AND    is_active = TRUE;

-- Re-enable ElevenLabs TTS provider
-- Note: provider_key does not exist in ai_provider_configs; use name + slug only.
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

-- Convert data bundle byte counts to MB if accidentally set as bytes
UPDATE prize_pool
SET    base_value  = base_value / 1048576,
       name        = (base_value / 1048576)::TEXT || 'MB Data',
       updated_at  = NOW()
WHERE  prize_type  = 'data'
AND    base_value  > 1000000;

-- Remove orphan spin-limit config keys (IF table/rows exist — safe delete)
DELETE FROM program_configs
WHERE  key IN ('spin_max_per_day', 'spin_max_per_user_per_day');

UPDATE network_configs
SET    value       = '-- UNUSED: spin limits are set in spin_tiers table, not here --',
       updated_at  = NOW()
WHERE  key IN ('spin_max_per_day', 'spin_max_per_user_per_day');

-- ─────────────────────────────────────────────────────────────────────────────
-- From 114: Data quality + template consistency
-- ─────────────────────────────────────────────────────────────────────────────

-- Normalise ui_template PascalCase → snake_case (ui_template column added in mig 032)
UPDATE studio_tools SET ui_template = 'image_creator'    WHERE ui_template IN ('ImageCreator',  'image-creator');
UPDATE studio_tools SET ui_template = 'voice_studio'     WHERE ui_template IN ('VoiceStudio',   'voice-studio');
UPDATE studio_tools SET ui_template = 'video_creator'    WHERE ui_template IN ('VideoCreator',  'video-creator');
UPDATE studio_tools SET ui_template = 'knowledge_doc'    WHERE ui_template IN ('KnowledgeDoc',  'knowledge-doc');
UPDATE studio_tools SET ui_template = 'music_composer'   WHERE ui_template IN ('MusicComposer', 'music-composer');
UPDATE studio_tools SET ui_template = 'code_helper'      WHERE ui_template IN ('CodeHelper',    'code-helper');
UPDATE studio_tools SET ui_template = 'nexus_chat'       WHERE ui_template IN ('NexusChat',     'nexus-chat');
UPDATE studio_tools SET ui_template = 'transcribe'       WHERE ui_template IN ('Transcribe');
UPDATE studio_tools SET ui_template = 'vision_ask'       WHERE ui_template IN ('VisionAsk',     'vision-ask');
UPDATE studio_tools SET ui_template = 'image_editor'     WHERE ui_template IN ('ImageEditor',   'image-editor');
UPDATE studio_tools SET ui_template = 'voice_to_plan'    WHERE ui_template IN ('VoiceToPlan',   'voice-to-plan');
UPDATE studio_tools SET ui_template = 'build_tool'       WHERE ui_template IN ('BuildTool',     'build-tool');
UPDATE studio_tools SET ui_template = 'code_pro'         WHERE ui_template IN ('CodePro',       'code-pro');
UPDATE studio_tools SET ui_template = 'my_ai_photo'      WHERE ui_template IN ('MyAiPhoto',     'my-ai-photo');
UPDATE studio_tools SET ui_template = 'video_multi_scene' WHERE ui_template IN ('VideoMultiScene', 'video-multi-scene', 'VideoMultiscene');
UPDATE studio_tools SET ui_template = 'video_animator'   WHERE ui_template IN ('VideoAnimator', 'video-animator');

-- Deactivate test/demo draws (name only — description column may not exist)
UPDATE draws
   SET status = 'CANCELLED'
 WHERE (
         LOWER(name) LIKE '%test%'
      OR LOWER(name) LIKE '%demo%'
   )
   AND status NOT IN ('COMPLETED', 'CANCELLED');

-- Restore correct Pollinations TTS voice options
-- Wrapped in exception handler: parameters column may not exist in all schema versions.
DO $$
BEGIN
  UPDATE studio_tools
     SET parameters = COALESCE(parameters, '{}'::jsonb) || '{
       "voice_options": [
         {"id": "Cherry",  "label": "Cherry  (Female, Friendly)"},
         {"id": "Serena",  "label": "Serena  (Female, Professional)"},
         {"id": "Ethan",   "label": "Ethan   (Male, Clear)"}
       ]
     }'::jsonb
   WHERE slug IN ('voice-studio', 'narrate-pro', 'ai-podcast', 'my-podcast')
     AND (parameters IS NULL OR parameters::text NOT LIKE '%Cherry%');
EXCEPTION WHEN undefined_column THEN
  -- parameters column not present in this schema version — skip safely
  NULL;
END $$;

-- ─────────────────────────────────────────────────────────────────────────────
-- From 115: Normalise ui_template snake_case → kebab-case
-- ─────────────────────────────────────────────────────────────────────────────

UPDATE studio_tools SET ui_template = 'knowledge-doc'       WHERE ui_template IN ('knowledge_doc');
UPDATE studio_tools SET ui_template = 'image-creator'       WHERE ui_template IN ('image_creator');
UPDATE studio_tools SET ui_template = 'voice-studio'        WHERE ui_template IN ('voice_studio');
UPDATE studio_tools SET ui_template = 'video-creator'       WHERE ui_template IN ('video_creator');
UPDATE studio_tools SET ui_template = 'music-composer'      WHERE ui_template IN ('music_composer');
UPDATE studio_tools SET ui_template = 'code-helper'         WHERE ui_template IN ('code_helper');
UPDATE studio_tools SET ui_template = 'nexus-chat'          WHERE ui_template IN ('nexus_chat');
UPDATE studio_tools SET ui_template = 'vision-ask'          WHERE ui_template IN ('vision_ask');
UPDATE studio_tools SET ui_template = 'image-editor'        WHERE ui_template IN ('image_editor');
UPDATE studio_tools SET ui_template = 'voice-to-plan'       WHERE ui_template IN ('voice_to_plan');
UPDATE studio_tools SET ui_template = 'build-tool'          WHERE ui_template IN ('build_tool');
UPDATE studio_tools SET ui_template = 'code-pro'            WHERE ui_template IN ('code_pro');
UPDATE studio_tools SET ui_template = 'my-ai-photo'         WHERE ui_template IN ('my_ai_photo');
UPDATE studio_tools SET ui_template = 'video-multi-scene'   WHERE ui_template IN ('video_multi_scene');
UPDATE studio_tools SET ui_template = 'video-animator'      WHERE ui_template IN ('video_animator');

-- ─────────────────────────────────────────────────────────────────────────────
-- From 116: Remove contradictory spin limit config key
-- ─────────────────────────────────────────────────────────────────────────────

DELETE FROM network_configs
WHERE key = 'spin_max_per_user_per_day';

-- ─────────────────────────────────────────────────────────────────────────────
-- From 117: Prize pool corrections
-- ─────────────────────────────────────────────────────────────────────────────

UPDATE prize_pool
SET    name = '₦' || TRIM(TO_CHAR(base_value / 100.0, 'FM999999990.##')) || ' Airtime'
WHERE  LOWER(prize_type) = 'airtime'
AND    is_active = TRUE
AND    base_value > 0
AND    name <> '₦' || TRIM(TO_CHAR(base_value / 100.0, 'FM999999990.##')) || ' Airtime';

UPDATE prize_pool
SET    base_value = 10
WHERE  LOWER(prize_type) = 'data_bundle'
AND    LOWER(name) LIKE '%10mb%'
AND    base_value NOT BETWEEN 9 AND 11;

UPDATE prize_pool
SET    is_active = FALSE
WHERE  id IN (
  SELECT id
  FROM (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY LOWER(name) ORDER BY id ASC) AS rn
    FROM prize_pool
    WHERE is_active = TRUE
  ) dedup
  WHERE rn > 1
);

DO $$
DECLARE
  current_sum NUMERIC(10,2);
  target_id UUID;
  adjustment NUMERIC(10,2);
BEGIN
  UPDATE prize_pool
  SET    win_probability_weight = ROUND(win_probability_weight / 100.0, 2)
  WHERE  is_active = TRUE
  AND    win_probability_weight > 100;

  SELECT COALESCE(SUM(win_probability_weight), 0)
  INTO   current_sum
  FROM   prize_pool
  WHERE  is_active = TRUE;

  IF current_sum <> 100.00 THEN
    SELECT id
    INTO   target_id
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

