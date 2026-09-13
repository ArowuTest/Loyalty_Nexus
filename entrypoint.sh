#!/bin/sh
# entrypoint.sh — Runs database migrations then starts the API binary.
# Migrations use DATABASE_URL (internal Render network — fast, no SSL needed).
# Migration failure ABORTS startup (exit non-zero) — a partial or unknown schema
# must never serve traffic behind a green /health check. On a failed deploy Render
# keeps the previous release live, which is the safe outcome.

set -e

echo "[entrypoint] Running database migrations..."
# ─────────────────────────────────────────────────────────────────────────────
# Database migrations are managed by golang-migrate (external binary, built
# into this Docker image at /migrate).
#
# Migration files live in /database/migrations/ (relative to repo root, NOT
# under /backend/). Applied versions are tracked in the schema_migrations
# table — do NOT delete that table.
#
# To add a new migration: create
#   database/migrations/<NNN>_<description>.up.sql
#   database/migrations/<NNN>_<description>.down.sql
#
# Never place migration files in backend/database/migrations/ — that
# directory does not exist and the runner will not see those files.
#
# See MIGRATIONS.md at the repo root for the full playbook.
# ─────────────────────────────────────────────────────────────────────────────
if /migrate fix-and-up; then
    echo "[entrypoint] Migrations complete."
else
    rc=$?
    echo "[entrypoint] FATAL: migrations failed (exit $rc) — refusing to start the API."
    echo "[entrypoint] A partial/unknown schema must not serve traffic; aborting the deploy."
    echo "[entrypoint] Render will keep the previous release live. Inspect the migration log above."
    exit "$rc"
fi

echo "[entrypoint] Starting API..."
exec /api
