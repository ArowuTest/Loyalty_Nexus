#!/bin/sh

echo "[entrypoint] Running database migrations..."
if /migrate up; then
    echo "[entrypoint] Migrations completed successfully"
else
    echo "[entrypoint] WARNING: Migrations returned non-zero (continuing anyway)"
fi

echo "[entrypoint] Starting API on port ${PORT:-10000}..."
exec /api
