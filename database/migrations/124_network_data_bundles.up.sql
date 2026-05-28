-- Migration 124: DB-first data bundle catalog
--
-- Creates network_data_bundles table so bundles are served from the DB
-- (populated by DataBundleSyncJob 5×/day) rather than calling VTPass on
-- every user page load.
--
-- Design mirrors RechargeMax migration 073.

CREATE TABLE IF NOT EXISTS network_data_bundles (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    network_code     TEXT NOT NULL,
    variation_code   TEXT NOT NULL,
    name             TEXT NOT NULL,
    price            NUMERIC(12, 2) NOT NULL DEFAULT 0,
    data_size        TEXT NOT NULL DEFAULT '',
    is_active        BOOLEAN NOT NULL DEFAULT TRUE,
    last_synced_at   TIMESTAMP WITH TIME ZONE,
    created_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT network_data_bundles_network_code_variation_code_key
        UNIQUE (network_code, variation_code)
);

CREATE INDEX IF NOT EXISTS idx_network_data_bundles_network_active
    ON network_data_bundles (network_code, is_active);
