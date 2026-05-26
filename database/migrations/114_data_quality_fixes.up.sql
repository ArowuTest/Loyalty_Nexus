-- Migration 114: Platform data quality & template consistency fixes
-- Covers: BUG-010, BUG-022, BUG-046
--
-- BUG-010: Standardise ui_template values to snake_case throughout.
--   Frontend template registry uses snake_case keys (image_creator, not ImageCreator).
--   Any PascalCase or kebab-case variants will silently fall back to knowledge_doc.
UPDATE studio_tools SET ui_template = 'image_creator'   WHERE ui_template IN ('ImageCreator',  'image-creator');
UPDATE studio_tools SET ui_template = 'voice_studio'    WHERE ui_template IN ('VoiceStudio',   'voice-studio');
UPDATE studio_tools SET ui_template = 'video_creator'   WHERE ui_template IN ('VideoCreator',  'video-creator');
UPDATE studio_tools SET ui_template = 'knowledge_doc'   WHERE ui_template IN ('KnowledgeDoc',  'knowledge-doc');
UPDATE studio_tools SET ui_template = 'music_composer'  WHERE ui_template IN ('MusicComposer', 'music-composer');
UPDATE studio_tools SET ui_template = 'code_helper'     WHERE ui_template IN ('CodeHelper',    'code-helper');
UPDATE studio_tools SET ui_template = 'nexus_chat'      WHERE ui_template IN ('NexusChat',     'nexus-chat');
UPDATE studio_tools SET ui_template = 'transcribe'      WHERE ui_template IN ('Transcribe');
UPDATE studio_tools SET ui_template = 'vision_ask'      WHERE ui_template IN ('VisionAsk',     'vision-ask');
UPDATE studio_tools SET ui_template = 'image_editor'    WHERE ui_template IN ('ImageEditor',   'image-editor');
UPDATE studio_tools SET ui_template = 'voice_to_plan'   WHERE ui_template IN ('VoiceToPlan',   'voice-to-plan');
UPDATE studio_tools SET ui_template = 'build_tool'      WHERE ui_template IN ('BuildTool',     'build-tool');
UPDATE studio_tools SET ui_template = 'code_pro'        WHERE ui_template IN ('CodePro',       'code-pro');
UPDATE studio_tools SET ui_template = 'my_ai_photo'     WHERE ui_template IN ('MyAiPhoto',     'my-ai-photo');
UPDATE studio_tools SET ui_template = 'video_multi_scene' WHERE ui_template IN ('VideoMultiScene', 'video-multi-scene', 'VideoMultiscene');
UPDATE studio_tools SET ui_template = 'video_animator'  WHERE ui_template IN ('VideoAnimator', 'video-animator');

-- BUG-022: Deactivate test/demo draws so they don't appear in admin UI
--   or leak into user-facing draw history queries.
UPDATE draws
   SET status = 'CANCELLED'
 WHERE (
         LOWER(name)        LIKE '%test%'
      OR LOWER(name)        LIKE '%demo%'
      OR LOWER(description) LIKE '%test%'
      OR LOWER(description) LIKE '%demo%'
   )
   AND status NOT IN ('COMPLETED', 'CANCELLED');

-- BUG-046: Voice Studio currently exposes OpenAI voice IDs (alloy, echo, fable, onyx,
--   nova, shimmer) but the backend TTS chain uses Pollinations TTS (which supports
--   Cherry / Serena / Ethan) and ElevenLabs (now re-enabled via Migration 113).
--   Replace voice_ids JSON so the UI offers the Pollinations TTS voices that actually work.
--   This is a parameters JSONB update — safe to run multiple times (idempotent).
UPDATE studio_tools
   SET parameters = COALESCE(parameters, '{}'::jsonb) || '{
     "voice_options": [
       {"id": "Cherry",  "label": "Cherry  (Female, Friendly)"},
       {"id": "Serena",  "label": "Serena  (Female, Professional)"},
       {"id": "Ethan",   "label": "Ethan   (Male, Clear)"}
     ]
   }'::jsonb
 WHERE slug IN ('voice-studio', 'narrate-pro', 'ai-podcast', 'my-podcast')
   AND parameters IS NOT NULL
   AND parameters::text LIKE '%alloy%';   -- only update rows still using OpenAI IDs

-- For rows where parameters is null or doesn't have OpenAI IDs yet, insert the field:
UPDATE studio_tools
   SET parameters = COALESCE(parameters, '{}'::jsonb) || '{
     "voice_options": [
       {"id": "Cherry",  "label": "Cherry  (Female, Friendly)"},
       {"id": "Serena",  "label": "Serena  (Female, Professional)"},
       {"id": "Ethan",   "label": "Ethan   (Male, Clear)"}
     ]
   }'::jsonb
 WHERE slug IN ('voice-studio', 'narrate-pro', 'ai-podcast', 'my-podcast')
   AND (parameters IS NULL OR parameters::text NOT LIKE '%voice_options%');
