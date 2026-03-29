-- 008_hlr_cache.sql
-- Purpose: Cache for HLR lookups to handle ported numbers and reduce API costs.

CREATE TABLE IF NOT EXISTS network_cache (
    msisdn TEXT PRIMARY KEY, -- Normalized 234...
    network TEXT NOT NULL, -- MTN, Airtel, Glo, 9mobile
    last_verified TIMESTAMPTZ DEFAULT now(),
    cache_expires TIMESTAMPTZ NOT NULL,
    lookup_source TEXT CHECK (lookup_source IN ('hlr_api', 'user_selection', 'prefix_fallback')),
    is_valid BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_network_cache_expires ON network_cache(cache_expires);
