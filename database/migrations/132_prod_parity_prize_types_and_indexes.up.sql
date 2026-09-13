-- Migration 132: production parity for objects that live ONLY in EDITED,
-- already-applied migrations and therefore never reached production.
--
-- WHY THIS EXISTS
-- The runner is golang-migrate with a single-row schema_migrations table and no
-- checksums, so editing an already-applied file (003..125) changes ONLY fresh
-- installs — production ran the COMMITTED version and its version cursor is already
-- past it, so the edit is inert there. This migration re-issues, under a NEW
-- version, the specific objects the runtime now depends on that are missing on prod.
--
-- Every statement is idempotent, so this is a NO-OP on a fresh/CI database (where
-- the edits already applied) and a FIX on production. Same pattern as 120/121.
--
-- SCOPE WAS VERIFIED, NOT ASSUMED. Each candidate object was checked against the
-- COMMITTED migration history (the only thing prod actually ran). Two indexes from
-- the original review list were DELIBERATELY OMITTED because their invariant is
-- already enforced on prod under a different name — adding a second object would be
-- pure write-amplification on a hot path:
--   • users.phone_number uniqueness  → already enforced by constraint
--     users_msisdn_key (committed migration 002: `msisdn TEXT UNIQUE`; column later
--     renamed msisdn→phone_number in committed 020/060, and an auto-named constraint
--     survives a column rename). A duplicate insert is already rejected on prod.
--     So idx_users_phone_number is NOT created here.
--   • spin_results.fulfillment_status  → already served on prod by the PARTIAL index
--     idx_spin_results_status (committed migration 020:
--     `WHERE fulfillment_status IN ('pending','processing','held')`), which is a
--     better fit for the fulfilment worker than a plain full index. So
--     idx_spin_results_fulfillment_status is NOT created here.
-- The three objects below have NO committed equivalent on production.

-- ── 1. Prize types: allow 'physical'/'goods' (edited into 125; never ran on prod)
-- Needed by the spin/prize (V2) work, which introduces physical/goods prizes; prod
-- currently has the narrower migration-060 set. The CHECK below is a SUPERSET of
-- both the current-prod set AND the physical/goods set, so ADD can never fail
-- validating existing rows in either state.
ALTER TABLE prize_pool DROP CONSTRAINT IF EXISTS prize_pool_prize_type_check;
ALTER TABLE prize_pool
    ADD CONSTRAINT prize_pool_prize_type_check
    CHECK (prize_type IN (
        'try_again', 'airtime', 'data', 'data_bundle', 'momo_cash',
        'pulse_points', 'bonus_points', 'studio_credits', 'physical', 'goods'
    ));

-- ── 2. Recharge-history hot path (edited into 112; never ran on prod) ─────────
-- The /user/recharges endpoint filters by user_id + status. Prod's committed
-- recharge indexes (108) cover msisdn and payment_reference but NOT user_id, so
-- this is a genuine gap. Non-unique → always safe to create.
CREATE INDEX IF NOT EXISTS idx_recharges_user_status_created
    ON recharges (user_id, status, created_at DESC);

-- ── 3. Fulfilment idempotency guard (edited into 112; never ran on prod) ──────
-- The UNIQUE partial index on vtpass_request_id is the ONLY thing preventing a
-- recharge from being fulfilled twice under the same VTPass reference; committed
-- migration 108 declares vtpass_request_id as a plain nullable TEXT with no
-- uniqueness, so prod has no such guard. This is the highest-value object here.
--
-- It is created only when the data permits, so this migration can never fail a
-- deploy on unexpected production data — a duplicate is surfaced as a WARNING for
-- manual investigation (a duplicate here means a real double-fulfilment already
-- happened and must be looked at) rather than aborting the deploy.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM recharges
        WHERE vtpass_request_id IS NOT NULL AND vtpass_request_id <> ''
        GROUP BY vtpass_request_id HAVING count(*) > 1
    ) THEN
        RAISE WARNING 'idx_recharges_vtpass_request_id NOT created: duplicate vtpass_request_id values exist — investigate double-fulfilment, then create the UNIQUE index manually';
    ELSE
        CREATE UNIQUE INDEX IF NOT EXISTS idx_recharges_vtpass_request_id
            ON recharges (vtpass_request_id)
            WHERE vtpass_request_id IS NOT NULL AND vtpass_request_id <> '';
    END IF;
END $$;
