-- Migration 115: Normalise ui_template values to kebab-case (BUG-009/010 completion)
--
-- Migration 114 converted PascalCase → snake_case.
-- The admin studio-tools dropdown and the user-facing tool page template registry
-- both use kebab-case keys (knowledge-doc, image-creator, voice-studio, etc.).
-- This migration converts any remaining snake_case values to kebab-case so that:
--   1. Admin tool editor shows the correct selected template in the dropdown.
--   2. User-facing /studio/tool/[slug] page renders the correct template component.

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
