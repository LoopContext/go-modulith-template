#!/bin/bash

# Quickstart script for Go Modulith Template
# Automates the complete setup process

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo -e "${BLUE}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Go Modulith Template - Quickstart Setup            ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════╝${NC}"
echo ""

# Step 1: Run validation
echo -e "${BLUE}Step 1/5:${NC} Validating setup..."
if ! ./scripts/validate-setup.sh; then
    echo ""
    echo -e "${RED}❌ Setup validation failed${NC}"
    echo "Please fix the errors above and run this script again."
    exit 1
fi
echo ""

# Step 2: Install missing dependencies
echo -e "${BLUE}Step 2/5:${NC} Checking development tools..."

MISSING_TOOLS=()
for tool in sqlc buf migrate air golangci-lint; do
    if ! command -v "$tool" > /dev/null 2>&1; then
        MISSING_TOOLS+=("$tool")
    fi
done

if [ ${#MISSING_TOOLS[@]} -gt 0 ]; then
    echo -e "${YELLOW}⚠ Missing tools: ${MISSING_TOOLS[*]}${NC}"
    echo -n "Install missing tools now? [y/N] "
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        echo "Installing development tools..."
        make install-deps
        echo -e "${GREEN}✓ Tools installed${NC}"
    else
        echo -e "${YELLOW}⚠ Skipping tool installation${NC}"
        echo "You can install them later with: make install-deps"
    fi
else
    echo -e "${GREEN}✓ All development tools are installed${NC}"
fi
echo ""

# Step 3: Start Docker infrastructure
echo -e "${BLUE}Step 3/5:${NC} Starting Docker infrastructure..."

# Check if containers are already running
if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "modulith_db"; then
    echo -e "${YELLOW}⚠ Docker containers are already running${NC}"
    echo -n "Restart containers? [y/N] "
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        echo "Stopping existing containers..."
        docker-compose down 2>/dev/null || true
        echo "Starting containers..."
        docker-compose up -d
    else
        echo -e "${GREEN}✓ Using existing containers${NC}"
    fi
else
    echo "Starting Docker containers..."
    docker-compose up -d
fi

# Wait for services to be healthy
echo ""
echo "Waiting for services to be ready..."
MAX_WAIT=60
WAIT_COUNT=0

while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "modulith_db"; then
        if docker exec modulith_db pg_isready -U postgres > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Database is ready${NC}"
            break
        fi
    fi
    sleep 1
    WAIT_COUNT=$((WAIT_COUNT + 1))
    echo -n "."
done
echo ""

if [ $WAIT_COUNT -eq $MAX_WAIT ]; then
    echo -e "${RED}❌ Timeout waiting for database${NC}"
    echo "Please check Docker containers: docker-compose ps"
    exit 1
fi

# Wait a bit more for Redis
sleep 2
if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "modulith_redis"; then
    if docker exec modulith_redis redis-cli ping > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Redis is ready${NC}"
    fi
fi
echo ""

# Step 4: Run migrations
echo -e "${BLUE}Step 4/5:${NC} Running database migrations..."
if make migrate 2>&1 | grep -q "no change"; then
    echo -e "${GREEN}✓ Database is up to date${NC}"
else
    echo -e "${GREEN}✓ Migrations completed${NC}"
fi
echo ""

# Step 5: Optionally run seed data
echo -e "${BLUE}Step 5/5:${NC} Seed data..."
echo -n "Run seed data? [y/N] "
read -r response
if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
    echo "Running seed data..."
    make seed
    echo -e "${GREEN}✓ Seed data completed${NC}"
else
    echo -e "${YELLOW}⚠ Skipping seed data${NC}"
    echo "You can run it later with: make seed"
fi
echo ""

# Summary
echo -e "${BLUE}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  ✓ Setup Complete!                                   ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════╝${NC}"
echo ""
echo "Next steps:"
echo ""
echo "  1. Start the development server:"
echo -e "     ${GREEN}make dev${NC}"
echo ""
echo "  2. Or run a specific module:"
echo -e "     ${GREEN}make dev-module auth${NC}"
echo ""
echo "  3. Check health endpoints:"
echo "     curl http://localhost:8000/readyz"
echo ""
echo "  4. View observability:"
echo "     - Jaeger: http://localhost:16686"
echo "     - Prometheus: http://localhost:9090"
echo "     - Grafana: http://localhost:3000 (admin/admin)"
echo ""
echo "  5. Run diagnostics:"
echo -e "     ${GREEN}make doctor${NC}"
echo ""

