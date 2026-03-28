#!/bin/sh
# Loyalty Nexus — API container entrypoint
#
# Runs database migrations then starts the API.
# The API retries DB connection up to 10 times (30s total) so a brief
# delay while migrations apply is fully handled gracefully.
# golang-migrate is idempotent — already-applied files are skipped.
# Render allows up to 5 minutes for the port to become available.
set -e
echo "[entrypoint] Running database migrations..."
/migrate up
echo "[entrypoint] Migrations complete. Starting API on port ${PORT:-10000}..."
exec /api
