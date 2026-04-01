#!/bin/sh
# entrypoint.sh — Runs database migrations then starts the API binary.
# Migrations use DATABASE_URL (internal Render network — fast, no SSL needed).
# If migrations fail the container exits non-zero and Render keeps the previous
# healthy version live.

set -e

echo "[entrypoint] Running database migrations..."
/migrate fix-and-up

echo "[entrypoint] Migrations complete. Starting API..."
exec /api
