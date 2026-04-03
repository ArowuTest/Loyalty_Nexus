-- Revert migration 083
UPDATE studio_tools SET ui_template = 'video-creator' WHERE slug = 'video-cinematic';
UPDATE studio_tools SET ui_template = 'video-animator' WHERE slug = 'video-jingle';
