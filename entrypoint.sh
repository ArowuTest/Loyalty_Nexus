#!/bin/sh
# entrypoint.sh — Starts the API binary.
# Database migrations are handled by Render's preDeployCommand (/migrate up)
# which runs before this container goes live. This script only starts the app.

echo "[entrypoint] Starting API..."
exec /api
