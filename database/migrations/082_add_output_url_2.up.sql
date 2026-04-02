-- Migration 082: Add output_url_2 column to ai_generations for Suno dual-track support
ALTER TABLE ai_generations
  ADD COLUMN IF NOT EXISTS output_url_2 TEXT NOT NULL DEFAULT '';
