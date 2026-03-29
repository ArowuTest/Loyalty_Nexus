#!/bin/sh

echo "[entrypoint] Database migration strategy: fix-and-up"
echo "[entrypoint] If DB is dirty (previous deploy failed mid-migration), rewinding"
echo "[entrypoint] to prior clean version and re-running the failed migration."

# fix-and-up: detects dirty state, rewinds by 1, then runs all pending migrations.
# Safe for clean DBs too — just runs pending migrations normally.
if /migrate fix-and-up; then
    echo "[entrypoint] ✓ Migrations applied successfully"
else
    echo "[entrypoint] WARNING: migrations returned non-zero — checking current state"
    /migrate version || true
    echo "[entrypoint] Continuing — API will start even if some migrations failed"
fi

echo "[entrypoint] Starting API..."
exec /api
