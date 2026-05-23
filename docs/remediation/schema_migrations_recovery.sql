-- =============================================================================
-- schema_migrations Recovery Playbook
-- =============================================================================
-- Run this against the LIVE production Postgres (Render dashboard → loyalty-nexus-db
-- → connect → External database URL → psql).
--
-- Background:
--   The migration runner (golang-migrate) marks a version "applied" in
--   schema_migrations as soon as it begins execution. If the SQL fails
--   partway through, the row may remain with dirty=true OR be left as
--   applied even though the schema change never landed. Subsequent deploys
--   then skip those versions because the runner trusts schema_migrations.
--
-- This script is CONSERVATIVE: it shows you state first, then you decide which
-- rows to clear. Do NOT blindly run the DELETE block until step 1 confirms
-- which versions are stale.
--
-- Audit Round 3 finding: versions 113-117 (ElevenLabs key + video tools tables)
-- + possibly 118-119 are marked applied but the schema does not reflect them.
-- =============================================================================

-- ── STEP 1: DIAGNOSE — read state, do not modify ─────────────────────────────
-- 1a. What does the runner think is applied?
SELECT version, dirty FROM schema_migrations ORDER BY version;

-- 1b. Verify which migration's schema actually landed. Run each probe and
--     record YES/NO. If a probe returns 0 rows but schema_migrations claims
--     the version is applied, that row is STALE and must be cleared.

-- Probe for migration 113 (elevenlabs settings):
SELECT to_regclass('public.elevenlabs_settings')   AS m113_table;     -- expect: elevenlabs_settings or NULL
-- Probe for migration 114 (video generations table):
SELECT to_regclass('public.video_generations')     AS m114_table;
-- Probe for migration 115 (lipsync jobs):
SELECT to_regclass('public.lipsync_jobs')          AS m115_table;
-- Probe for migration 116 (image-to-video jobs):
SELECT to_regclass('public.image_to_video_jobs')   AS m116_table;
-- Probe for migration 117 (motion-imitation jobs):
SELECT to_regclass('public.motion_imitation_jobs') AS m117_table;
-- Probe for migration 118 (try-again consolidation):
SELECT COUNT(*) FILTER (WHERE prize_label = 'Try Again') AS m118_try_again_count
FROM prize_pool WHERE active = true;
-- Probe for migration 119 (lifetime_points backfill comment):
SELECT col_description('public.users'::regclass,
       (SELECT attnum FROM pg_attribute
        WHERE attrelid = 'public.users'::regclass AND attname = 'lifetime_points'))
       AS m119_column_comment;

-- ── STEP 2: REMEDIATE — clear ONLY the stale versions you confirmed in step 1
-- Replace the version list below with whichever versions returned NULL/0 above.
-- Example: if 113-117 are stale and 118-119 already applied correctly, use:
--          DELETE FROM schema_migrations WHERE version IN (113,114,115,116,117);

-- BEGIN;
--   DELETE FROM schema_migrations WHERE version IN (113, 114, 115, 116, 117);
--   -- If any row was dirty=true, also clear that flag:
--   UPDATE schema_migrations SET dirty = false WHERE dirty = true;
-- COMMIT;

-- ── STEP 3: REDEPLOY ─────────────────────────────────────────────────────────
-- Trigger a manual deploy on Render (Settings → Manual Deploy → Deploy latest commit).
-- entrypoint.sh runs `/migrate fix-and-up`, which will now re-run the cleared
-- versions in order. Watch the deploy log for:
--   "Migration NNN applied"   ← good
--   "no change"               ← BAD (means schema_migrations still claims applied)
--   "dirty database"          ← run UPDATE in step 2 to clear dirty flag, redeploy

-- ── STEP 4: VERIFY ───────────────────────────────────────────────────────────
-- Re-run the probes from step 1b. They should all return non-NULL / >0 now.
