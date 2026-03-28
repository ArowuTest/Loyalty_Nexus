#!/bin/sh
# Migrations are handled by Render's preDeployCommand (/migrate up)
# which runs before this container starts, using MIGRATE_DATABASE_URL.
# The entrypoint simply starts the API server immediately so Render's
# port health-check passes within the startup window.
exec /api
