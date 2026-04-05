-- Revert migration 098: restore previous (incorrect) template assignments
UPDATE studio_tools SET ui_template = 'KnowledgeDoc' WHERE slug = 'animate-my-photo';
UPDATE studio_tools SET ui_template = 'KnowledgeDoc' WHERE slug = 'my-video-story';
