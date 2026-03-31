-- Migration 072 DOWN: Remove draw_code column from draws table
DROP INDEX IF EXISTS uidx_draws_draw_code;
ALTER TABLE draws DROP COLUMN IF EXISTS draw_code;
