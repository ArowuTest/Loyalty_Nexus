#!/bin/sh
# Loyalty Nexus — API entrypoint
#
# Migrations are handled by Render's preDeployCommand (/migrate up)
# which runs BEFORE this container is started, using the external
# database hostname (MIGRATE_DATABASE_URL) with SSL.
#
# This entrypoint simply starts the API immediately so Render's port
# health-check passes within the startup window.
exec /api
