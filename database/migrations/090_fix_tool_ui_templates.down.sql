-- 090_fix_tool_ui_templates.down.sql
-- Reverts the ui_template corrections from 090_fix_tool_ui_templates.up.sql

UPDATE studio_tools SET ui_template = 'KnowledgeDoc' WHERE slug IN ('nexus-chat', 'ask-nexus', 'ai-chat');
UPDATE studio_tools SET ui_template = 'VoiceStudio'  WHERE slug = 'translate';
UPDATE studio_tools SET ui_template = 'VideoAnimator' WHERE slug = 'video-premium';
UPDATE studio_tools SET ui_template = 'VideoCreator'  WHERE slug = 'video-jingle';
