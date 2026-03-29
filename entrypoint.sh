#!/bin/sh

echo "[entrypoint] Running database migrations (with auto dirty-state recovery)..."
if /migrate fix-and-up; then
    echo "[entrypoint] Migrations completed successfully"
else
    echo "[entrypoint] WARNING: Migrations returned non-zero — API starting in degraded mode"
fi

echo "[entrypoint] Starting API on port ${PORT:-10000}..."
exec /api
