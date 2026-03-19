#!/bin/bash

# Pre-flight check for development commands
# Checks basic requirements before starting dev server

set -e

ERRORS=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check Docker
if ! command -v docker > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    exit 1
fi

if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker daemon is not running${NC}"
    echo "  Start Docker Desktop or docker service"
    exit 1
fi

# Check if database container is running
if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -q "modulith_db"; then
    echo -e "${RED}Error: Database container is not running${NC}"
    echo "  Start it with: just docker-up"
    exit 1
fi

# Check if database is accessible (if container is running)
if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "modulith_db"; then
    if ! docker exec modulith_db pg_isready -U postgres > /dev/null 2>&1; then
        echo -e "${YELLOW}Warning: Database is not ready yet${NC}"
        echo "  Wait a moment and try again, or check: docker-compose ps"
        exit 1
    fi
fi

# Check ports (non-blocking, just warn)
check_port() {
    local port=$1
    local service=$2
    if command -v lsof > /dev/null 2>&1; then
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
            echo -e "${YELLOW}Warning: Port $port ($service) is already in use${NC}"
            return 1
        fi
    fi
    return 0
}

check_port 8000 "HTTP"
check_port 9000 "gRPC"

exit 0

