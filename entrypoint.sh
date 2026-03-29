#!/bin/sh
echo "[entrypoint] Starting API on port ${PORT:-10000}..."
exec /api
