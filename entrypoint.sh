#!/bin/sh
set -e

echo "[entrypoint] Running database migrations..."
/migrate up
echo "[entrypoint] Migrations complete. Starting API..."
exec /api
