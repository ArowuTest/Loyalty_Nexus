#!/bin/sh
# entrypoint.sh — Runs database migrations then starts the API binary.
# Migrations use DATABASE_URL (internal Render network — fast, no SSL needed).
# Migration failures are logged but do NOT prevent the API from starting —
# the app uses fallback SQL patterns to handle schema variance gracefully.

set -e

echo "[entrypoint] Running database migrations..."
if /migrate fix-and-up; then
    echo "[entrypoint] Migrations complete."
else
    echo "[entrypoint] WARNING: migrations reported errors (exit $?) — starting API anyway."
    echo "[entrypoint] The API uses runtime schema detection for schema variance."
fi

echo "[entrypoint] Starting API..."
exec /api
