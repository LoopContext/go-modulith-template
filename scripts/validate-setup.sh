#!/bin/bash

# Setup validation script for Go Modulith Template
# Checks prerequisites and environment setup

set -e

ERRORS=0
WARNINGS=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔍 Validating setup for Go Modulith Template..."
echo ""

# Check Go version
check_go() {
    echo -n "Checking Go version... "
    if ! command -v go > /dev/null; then
        echo -e "${RED}✗${NC}"
        echo "  Error: Go is not installed"
        echo "  Install from: https://go.dev/dl/"
        ((ERRORS++))
        return 1
    fi

    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    REQUIRED_VERSION="1.24"

    if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
        echo -e "${YELLOW}⚠${NC} (found $GO_VERSION, requires $REQUIRED_VERSION+)"
        echo "  Warning: Go version $GO_VERSION found, but $REQUIRED_VERSION+ is required"
        ((WARNINGS++))
        return 1
    else
        echo -e "${GREEN}✓${NC} ($GO_VERSION)"
        return 0
    fi
}

# Check Docker
check_docker() {
    echo -n "Checking Docker... "
    if ! command -v docker > /dev/null; then
        echo -e "${RED}✗${NC}"
        echo "  Error: Docker is not installed"
        echo "  Install from: https://docs.docker.com/get-docker/"
        ((ERRORS++))
        return 1
    fi

    if ! docker info > /dev/null 2>&1; then
        echo -e "${RED}✗${NC}"
        echo "  Error: Docker daemon is not running"
        echo "  Start Docker Desktop or docker service"
        ((ERRORS++))
        return 1
    fi

    echo -e "${GREEN}✓${NC}"
    return 0
}

# Check Docker Compose
check_docker_compose() {
    echo -n "Checking Docker Compose... "
    if ! command -v docker-compose > /dev/null 2>&1 && ! docker compose version > /dev/null 2>&1; then
        echo -e "${RED}✗${NC}"
        echo "  Error: Docker Compose is not installed"
        echo "  Install from: https://docs.docker.com/compose/install/"
        ((ERRORS++))
        return 1
    fi

    echo -e "${GREEN}✓${NC}"
    return 0
}

# Check required tools
check_tool() {
    local tool=$1
    local install_cmd=$2

    echo -n "Checking $tool... "
    if ! command -v "$tool" > /dev/null; then
        echo -e "${YELLOW}⚠${NC} (not found)"
        echo "  Warning: $tool is not installed"
        echo "  Install with: $install_cmd"
        ((WARNINGS++))
        return 1
    else
        echo -e "${GREEN}✓${NC}"
        return 0
    fi
}

# Check port availability
check_port() {
    local port=$1
    local service=$2

    echo -n "Checking port $port ($service)... "
    if command -v lsof > /dev/null 2>&1; then
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
            echo -e "${YELLOW}⚠${NC} (in use)"
            echo "  Warning: Port $port is already in use"
            ((WARNINGS++))
            return 1
        fi
    elif command -v netstat > /dev/null 2>&1; then
        if netstat -an 2>/dev/null | grep -q ":$port.*LISTEN"; then
            echo -e "${YELLOW}⚠${NC} (in use)"
            echo "  Warning: Port $port is already in use"
            ((WARNINGS++))
            return 1
        fi
    else
        echo -e "${YELLOW}?${NC} (cannot check)"
        echo "  Warning: Cannot check port availability (lsof/netstat not available)"
        return 0
    fi

    echo -e "${GREEN}✓${NC}"
    return 0
}

# Check database connectivity (if docker containers are running)
check_database() {
    echo -n "Checking database connectivity... "

    # Check if docker containers are running
    if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -q "modulith_db"; then
        echo -e "${YELLOW}⚠${NC} (containers not running)"
        echo "  Info: Database container is not running. Run 'make docker-up' to start it."
        return 0  # Not an error, just informational
    fi

    # Try to connect (basic check)
    if command -v psql > /dev/null 2>&1; then
        if PGPASSWORD=postgres psql -h localhost -U postgres -d postgres -c "SELECT 1" > /dev/null 2>&1; then
            echo -e "${GREEN}✓${NC}"
            return 0
        fi
    fi

    echo -e "${YELLOW}?${NC} (cannot verify)"
    echo "  Info: Database container is running, but connection cannot be verified"
    echo "  (This is OK if you haven't run migrations yet)"
    return 0
}

# Run all checks
check_go
check_docker
check_docker_compose

echo ""
echo "Checking development tools..."
check_tool "sqlc" "make install-deps"
check_tool "buf" "make install-deps"
check_tool "migrate" "make install-deps"
check_tool "air" "make install-deps"
check_tool "golangci-lint" "make install-deps"

echo ""
echo "Checking port availability..."
check_port 8000 "HTTP"
check_port 9000 "gRPC"
check_port 5432 "PostgreSQL"
check_port 6379 "Redis"

echo ""
check_database

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if [ $ERRORS -gt 0 ]; then
    echo -e "${RED}✗ Validation failed with $ERRORS error(s)${NC}"
    if [ $WARNINGS -gt 0 ]; then
        echo -e "${YELLOW}  and $WARNINGS warning(s)${NC}"
    fi
    echo ""
    echo "Please fix the errors above before proceeding."
    exit 1
elif [ $WARNINGS -gt 0 ]; then
    echo -e "${YELLOW}⚠ Validation passed with $WARNINGS warning(s)${NC}"
    echo ""
    echo "You can proceed, but consider addressing the warnings above."
    exit 0
else
    echo -e "${GREEN}✓ All checks passed!${NC}"
    exit 0
fi

