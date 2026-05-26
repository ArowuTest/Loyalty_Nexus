# Database Migrations

Loyalty Nexus uses [golang-migrate](https://github.com/golang-migrate/migrate) to manage database schema changes. The migration binary is built into the Docker image and runs automatically on every Render deploy via `entrypoint.sh`.

## Canonical Directory

**All migration files MUST live in `database/migrations/` at the repository root.**

```
Loyalty_Nexus/
├── database/
│   └── migrations/          ← THE migration directory
│       ├── 001_*.up.sql
│       ├── 001_*.down.sql
│       ├── ...
│       └── NNN_*.up.sql
└── backend/                 ← do NOT place migrations under here
```

Do not create `backend/database/migrations/` — that path is intentionally absent. The migration runner will silently skip files placed there, leading to schema drift between dev and production.

## Adding a New Migration

1. Pick the next sequential 3-digit number (`NNN`).
2. Create both files:

```
database/migrations/NNN_short_description.up.sql      # forward migration
database/migrations/NNN_short_description.down.sql    # rollback
```

3. The `up.sql` must be idempotent — wrap destructive operations in `IF EXISTS` / `IF NOT EXISTS` guards or use `DO $$ ... $$` blocks. Render auto-redeploys can replay applied migrations on retry.
4. The `down.sql` must reverse the `up.sql` cleanly enough to recover from a bad release.

## How Migrations Apply

`entrypoint.sh` runs `/migrate fix-and-up` before the API binary starts:

- Reads files from `database/migrations/`
- Tracks applied versions in the `schema_migrations` table
- `fix-and-up` clears any "dirty" version flag from a previous failed run, then applies all pending migrations in order
- The API process only starts after migrations succeed (zero-downtime contract)

## Recovering From a Stale `schema_migrations` Row

If a deploy log shows "no change" but you know a migration's SQL never executed (e.g. table/column missing in production):

```sql
-- Diagnose: what does the runner think is applied?
SELECT * FROM schema_migrations;

-- Fix: mark the offending version as un-applied so the next deploy re-runs it
DELETE FROM schema_migrations WHERE version = NNN;
-- (or)
UPDATE schema_migrations SET dirty = false WHERE version = NNN AND dirty = true;
```

The next deploy will re-run version `NNN` from `database/migrations/`.

## What Not to Change

| Component | Status |
|-----------|--------|
| External `golang-migrate` binary | Keep — production-grade, used by Stripe, Cloudflare, etc. |
| Plain `.sql` migration files | Keep — readable by any engineer regardless of Go knowledge |
| `schema_migrations` tracking table | Keep — this is what makes deploys idempotent |
| Bootstrap of 3 critical tables in Go | Keep — safety net only; migrations remain the system of record |
| Separate `up.sql` / `down.sql` files | Keep — enables rollbacks |
| GORM `AutoMigrate` for production | Do not introduce — unsafe for live schema changes |

## Non-Migration SQL Files

Historical schema dumps and ad-hoc fix scripts that are **not** managed by
the migration runner live in `docs/database/`:

```
docs/database/consolidated_schema.sql    ← full schema snapshot (reference only)
docs/database/fix_missing_tables.sql     ← ad-hoc DDL used during early dev
docs/database/020_streak_grace_and_expiry.sql  ← unnumbered historical patch
docs/database/021_pwa_install_referrals.sql
docs/database/022_dynamic_multipliers.sql
docs/database/023_dynamic_content_updates.sql
```

These files are **not executed by `golang-migrate`** — they lack the required
`NNN_description.up.sql` / `NNN_description.down.sql` naming pattern.
Do not move them back into `database/migrations/` expecting them to run.
If a change they describe is still needed, create a properly numbered migration.

**Rule of thumb:** if it lives in `database/migrations/` and does not match
`NNN_description.{up|down}.sql` exactly, golang-migrate will silently ignore it.

