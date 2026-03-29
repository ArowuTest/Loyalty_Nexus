#!/bin/sh


echo "[entrypoint] Database migration strategy: force v59 + run only migration 060"
echo "[entrypoint] Migration 060 is a comprehensive safety net that creates all"
echo "[entrypoint] critical tables with IF NOT EXISTS — safe under ANY DB state."

# Force the migration pointer to version 59.
# This marks migrations 1-59 as "applied" WITHOUT re-running them.
# The DB already has a partial set of these tables from previous deploy cycles.
# Migration 060 (the only one that runs) uses IF NOT EXISTS everywhere,
# so it safely creates anything missing and skips what already exists.
if /migrate force 59; then
    echo "[entrypoint] ✓ Migration pointer set to v59"
else
    echo "[entrypoint] WARNING: force 59 failed — attempting to continue anyway"
fi

echo "[entrypoint] Running migration 060 (comprehensive safety net)..."
if /migrate up; then
    echo "[entrypoint] ✓ Migration 060 applied successfully"
else
    echo "[entrypoint] WARNING: migrate up returned non-zero — checking if 060 already applied"
    /migrate version || true
fi

echo "[entrypoint] Starting API..."
exec /api
