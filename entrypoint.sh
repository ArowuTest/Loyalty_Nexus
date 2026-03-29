#!/bin/sh
# Loyalty Nexus — API container entrypoint
#
# Attempts database migrations then starts the API unconditionally.
# The API has built-in DB retry (10 attempts, 3s apart) so it handles
# the brief window while migrations complete.
#
# Render health check: GET /health → {"status":"ok"} (no DB query)
# This passes as soon as the API binds to PORT, which happens after
# migrations complete (usually <30s on Render Postgres).
#
# If migrations fail (e.g. dirty state from a prior interrupted deploy),
# the API starts anyway and the error is logged. Fix by running:
#   /migrate force <version>  on the Render shell.

echo "[entrypoint] Running database migrations..."
if /migrate up; then
  echo "[entrypoint] Migrations complete."
else
  echo "[entrypoint] WARNING: migrations exited non-zero (check logs). Starting API anyway."
fi

echo "[entrypoint] Starting API on port ${PORT:-10000}..."
exec /api
