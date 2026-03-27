-- Migration 042: Add RechargeMax-aligned fields to prize_pool
-- Adds: is_no_win, no_win_message, color_scheme, sort_order, minimum_recharge,
--       icon_name, terms_and_conditions, prize_code, variation_code

ALTER TABLE prize_pool
    ADD COLUMN IF NOT EXISTS is_no_win           BOOLEAN     NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS no_win_message      TEXT        NOT NULL DEFAULT 'Better Luck Next Time',
    ADD COLUMN IF NOT EXISTS color_scheme        TEXT        NOT NULL DEFAULT '#CCCCCC',
    ADD COLUMN IF NOT EXISTS sort_order          INTEGER     NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS minimum_recharge    BIGINT      NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS icon_name           TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS terms_and_conditions TEXT       NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS prize_code          TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS variation_code      TEXT        NOT NULL DEFAULT '';

-- Mark the existing try_again prize as is_no_win
UPDATE prize_pool SET is_no_win = TRUE WHERE prize_type = 'try_again';

-- Index for admin list ordering
CREATE INDEX IF NOT EXISTS idx_prize_pool_sort_order ON prize_pool (sort_order);
