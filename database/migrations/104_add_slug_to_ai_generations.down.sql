-- Migration 104 down: remove vanity slug from ai_generations
DROP INDEX IF EXISTS idx_ai_generations_slug;
ALTER TABLE ai_generations DROP COLUMN IF EXISTS slug;
