#!/bin/sh
# Loyalty Nexus — API container entrypoint
#
# Strategy: start /api immediately so Render's health check passes,
# then run migrations in the background.
#
# Why this works:
#   - /health returns {"status":"ok"} instantly (no DB queries)
#   - Render declares the deploy healthy as soon as /health returns 200
#   - Migrations complete in the background (~60-120s for first run)
#   - golang-migrate is idempotent — subsequent deploys skip applied files
#   - gorm.Open succeeds without migrations (it just opens a connection pool)
#   - API endpoints that hit the DB work once migrations complete
#
# If the background migration fails, the API keeps running but returns
# DB errors — the next deploy will retry the migration automatically.

echo "[entrypoint] Starting API on port ${PORT:-10000}..."
/api &
API_PID=$!

echo "[entrypoint] Running database migrations in background..."
/migrate up
MIGRATE_EXIT=$?

if [ $MIGRATE_EXIT -eq 0 ]; then
  echo "[entrypoint] Migrations complete."
else
  echo "[entrypoint] WARNING: Migrations exited with code $MIGRATE_EXIT"
fi

# Wait for the API process to keep the container alive
wait $API_PID
