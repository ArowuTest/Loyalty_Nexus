#!/bin/sh
# Loyalty Nexus — API container entrypoint
#
# CRITICAL: Start /api FIRST so Render health check passes within 5 seconds.
# Then run migrations in background. The API's built-in DB retry loop (30
# attempts, 3s apart = 90s window) bridges any gap.

set -e

echo "[entrypoint] Starting API on port ${PORT:-10000}..."
/api &
API_PID=$!

echo "[entrypoint] Running database migrations in background (PID $$)..."
(
  if /migrate up 2>&1; then
    echo "[entrypoint] Migrations complete."
  else
    echo "[entrypoint] WARNING: migrations exited non-zero. API continues."
  fi
) &

echo "[entrypoint] Waiting for API process (PID $API_PID)..."
wait $API_PID
