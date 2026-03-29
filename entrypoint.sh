#!/bin/sh

echo "[entrypoint] Attempting database migrations..."

# First attempt
/migrate up > /tmp/migrate_out.txt 2>&1
MIGRATE_EXIT=$?

if [ $MIGRATE_EXIT -eq 0 ]; then
    echo "[entrypoint] All migrations completed successfully"
elif grep -q "Dirty database" /tmp/migrate_out.txt; then
    # Extract dirty version number
    DIRTY_VERSION=$(grep -oE "Dirty database version [0-9]+" /tmp/migrate_out.txt | grep -oE "[0-9]+$")
    DIRTY_VERSION=${DIRTY_VERSION:-5}
    PREV_VERSION=$((DIRTY_VERSION - 1))
    echo "[entrypoint] Dirty migration at v${DIRTY_VERSION}. Rewinding to v${PREV_VERSION} to re-run it..."
    /migrate force "${PREV_VERSION}"
    echo "[entrypoint] Retrying all pending migrations..."
    if /migrate up; then
        echo "[entrypoint] Migrations completed after dirty fix"
    else
        echo "[entrypoint] WARNING: Migrations still have issues (API will start anyway)"
        cat /tmp/migrate_out.txt
    fi
else
    echo "[entrypoint] WARNING: Migration error (not dirty - continuing):"
    cat /tmp/migrate_out.txt
fi

echo "[entrypoint] Starting API on port ${PORT:-10000}..."
exec /api
