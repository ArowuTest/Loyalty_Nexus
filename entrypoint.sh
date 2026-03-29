#!/bin/sh

echo "[entrypoint] Forcing re-run of migration 060 to apply column patches and test user seeds..."

# Force pointer back to v59 so migration 060 re-runs on every deploy.
# Migration 060 uses ADD COLUMN IF NOT EXISTS everywhere — 100% idempotent.
# This is safe to run repeatedly: columns that already exist are skipped.
if /migrate force 59; then
    echo "[entrypoint] ✓ Pointer set to v59"
else
    echo "[entrypoint] WARNING: force 59 failed — attempting up anyway"
fi

if /migrate up; then
    echo "[entrypoint] ✓ Migration 060 applied (columns + seeds ensured)"
else
    echo "[entrypoint] WARNING: migrate up returned non-zero"
    /migrate version || true
fi

echo "[entrypoint] Starting API..."
exec /api
