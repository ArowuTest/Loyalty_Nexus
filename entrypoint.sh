#!/bin/sh

echo "[entrypoint] Running pending migrations..."

if /migrate up; then
    echo "[entrypoint] ✓ Migrations applied"
else
    echo "[entrypoint] WARNING: migrate up returned non-zero — checking version"
    /migrate version || true
fi

echo "[entrypoint] Starting API..."
exec /api
