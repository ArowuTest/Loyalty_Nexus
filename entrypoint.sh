#!/bin/sh

echo "[entrypoint] Running database migrations..."

# All migrations now use CREATE TABLE IF NOT EXISTS and ON CONFLICT DO NOTHING,
# making them fully idempotent. We force the migration pointer back to v4 to
# ensure migrations 5+ are re-run and any missing tables are created safely.
# Tables that already exist are skipped by IF NOT EXISTS.
/migrate force 4
echo "[entrypoint] Migration pointer set to v4. Applying all migrations from v5..."

if /migrate up; then
    echo "[entrypoint] All migrations completed successfully"
else
    echo "[entrypoint] WARNING: Some migrations returned non-zero — API starting"
fi

echo "[entrypoint] Starting API on port ${PORT:-10000}..."
exec /api
