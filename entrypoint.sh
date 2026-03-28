#!/bin/sh
# Loyalty Nexus — API container entrypoint
#
# Strategy: run migrations then start the API.
# golang-migrate is idempotent — already-applied migrations are skipped.
# 59 SQL files typically complete in under 10 seconds on Render's Postgres.
# Render allows up to 5 minutes for the port to become available after
# container start, so migrations completing first is safe.
#
# If DATABASE_URL is unreachable, /migrate exits non-zero and Render
# keeps the previous healthy version live (zero-downtime guarantee).
set -e

echo "[entrypoint] Running database migrations..."
/migrate up
echo "[entrypoint] Migrations complete. Starting API on port ${PORT:-10000}..."
exec /api
