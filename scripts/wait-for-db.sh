#!/bin/bash
set -e

# Wait for database to be ready
# Usage: ./scripts/wait-for-db.sh [max_wait_seconds]

MAX_WAIT=${1:-30}
WAIT_COUNT=0

echo "Waiting for database to be ready (timeout: ${MAX_WAIT}s)..."

while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "modulith_db"; then
        if docker exec modulith_db pg_isready -U postgres > /dev/null 2>&1; then
            echo "✓ Database is ready"
            exit 0
        fi
    fi
    sleep 1
    WAIT_COUNT=$((WAIT_COUNT + 1))
    echo -n "."
done

echo ""
echo "❌ Timeout waiting for database"
exit 1
