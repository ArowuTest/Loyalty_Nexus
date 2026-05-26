#!/bin/sh
# entrypoint.sh — Runs database migrations then starts the API binary.
# Migrations use DATABASE_URL (internal Render network — fast, no SSL needed).
# Migration failures are logged but do NOT prevent the API from starting —
# the app uses fallback SQL patterns to handle schema variance gracefully.

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
    echo "[entrypoint] WARNING: migrations reported errors (exit $?) — starting API anyway."
    echo "[entrypoint] The API uses runtime schema detection for schema variance."
fi

echo "[entrypoint] Starting API..."
exec /api
