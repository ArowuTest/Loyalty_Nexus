-- Rollback migration 123
UPDATE studio_tools
SET    is_active   = FALSE,
       coming_soon = FALSE,
       updated_at  = NOW()
WHERE  slug = 'video-cinematic';

UPDATE studio_tools
SET    coming_soon = FALSE,
       updated_at  = NOW()
WHERE  slug = 'bg-remover';

ALTER TABLE studio_tools DROP COLUMN IF EXISTS coming_soon;
