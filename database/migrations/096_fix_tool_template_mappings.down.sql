-- Migration 095 DOWN: Revert template mapping fixes
UPDATE studio_tools SET ui_template = 'KnowledgeDoc', updated_at = NOW() WHERE slug = 'animate-my-photo';
UPDATE studio_tools SET ui_template = 'KnowledgeDoc', updated_at = NOW() WHERE slug = 'background-remover';
UPDATE studio_tools SET ui_template = 'ImageEditor',  updated_at = NOW() WHERE slug = 'bg-remover';
UPDATE studio_tools SET ui_template = 'KnowledgeDoc', updated_at = NOW() WHERE slug = 'my-ai-photo';
UPDATE studio_tools SET ui_template = 'KnowledgeDoc', updated_at = NOW() WHERE slug = 'my-marketing-jingle';
UPDATE studio_tools SET ui_template = 'KnowledgeDoc', updated_at = NOW() WHERE slug = 'my-podcast';
UPDATE studio_tools SET ui_template = 'KnowledgeDoc', updated_at = NOW() WHERE slug = 'my-video-story';
UPDATE studio_tools SET ui_template = 'KnowledgeDoc', updated_at = NOW() WHERE slug = 'text-to-speech';
UPDATE studio_tools SET ui_template = 'video-animator', updated_at = NOW() WHERE slug = 'video-cinematic';
