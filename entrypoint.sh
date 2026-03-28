#!/bin/sh
set -e
echo "[entrypoint] Running database migrations..."
/migrate up
echo "[entrypoint] Migrations complete. Starting API on port ${PORT:-8080}..."
exec /api
