-- Migration 104: Add vanity slug to ai_generations for personalised website URLs
-- Allows /s/my-business-name instead of /s/{uuid}

ALTER TABLE ai_generations ADD COLUMN IF NOT EXISTS slug VARCHAR(100);

-- Unique index only on rows that have a slug (NULLs are not unique-constrained)
CREATE UNIQUE INDEX IF NOT EXISTS idx_ai_generations_slug
  ON ai_generations (slug)
  WHERE slug IS NOT NULL AND slug <> '';
