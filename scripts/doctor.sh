#!/bin/bash

# Development environment diagnostic script for Go Modulith Template
# Comprehensive health check and diagnostics

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

ERRORS=0
WARNINGS=0

echo -e "${BLUE}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Development Environment Diagnostics                  ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if command exists and report status
check_cmd() {
    local cmd=$1
    if command -v "$cmd" > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} $cmd"
        return 0
    else
        echo -e "${RED}✗${NC} $cmd (not found)"
        ((ERRORS++))
        return 1
    fi
}

# Check port with detail
check_port_detailed() {
    local port=$1
    local service=$2
    echo -n "  Port $port ($service): "

    if command -v lsof > /dev/null 2>&1; then
        local pid=$(lsof -ti :$port 2>/dev/null || true)
        if [ -n "$pid" ]; then
            local proc=$(ps -p $pid -o comm= 2>/dev/null || echo "unknown")
            echo -e "${YELLOW}⚠${NC} in use by $proc (PID: $pid)"
            ((WARNINGS++))
            return 1
        fi
    elif command -v netstat > /dev/null 2>&1; then
        if netstat -an 2>/dev/null | grep -q ":$port.*LISTEN"; then
            echo -e "${YELLOW}⚠${NC} in use"
            ((WARNINGS++))
            return 1
        fi
    fi

    echo -e "${GREEN}✓${NC} available"
    return 0
}

# Section: Prerequisites
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Prerequisites${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

check_cmd "go"
if command -v go > /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}')
    echo "  Version: $GO_VERSION"
fi

check_cmd "docker"
check_cmd "docker-compose"

echo ""
echo "Development Tools:"
check_cmd "sqlc"
check_cmd "buf"
check_cmd "migrate"
check_cmd "air"
check_cmd "golangci-lint"

echo ""

# Section: Docker Containers
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Docker Containers${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

if ! command -v docker > /dev/null 2>&1; then
    echo -e "${RED}✗${NC} Docker not available"
    ((ERRORS++))
else
    if ! docker info > /dev/null 2>&1; then
        echo -e "${RED}✗${NC} Docker daemon not running"
        echo "  Start Docker Desktop or docker service"
        ((ERRORS++))
    else
        CONTAINERS=("modulith_db" "modulith_redis" "modulith_jaeger" "modulith_prometheus" "modulith_grafana")
        for container in "${CONTAINERS[@]}"; do
            if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "^${container}$"; then
                STATUS=$(docker inspect --format='{{.State.Status}}' "$container" 2>/dev/null || echo "unknown")
                HEALTH=$(docker inspect --format='{{.State.Health.Status}}' "$container" 2>/dev/null || echo "no-healthcheck")
                if [ "$STATUS" = "running" ]; then
                    if [ "$HEALTH" = "healthy" ] || [ "$HEALTH" = "no-healthcheck" ]; then
                        echo -e "${GREEN}✓${NC} $container (running)"
                    else
                        echo -e "${YELLOW}⚠${NC} $container (running, but unhealthy: $HEALTH)"
                        ((WARNINGS++))
                    fi
                else
                    echo -e "${YELLOW}⚠${NC} $container ($STATUS)"
                    ((WARNINGS++))
                fi
            else
                echo -e "${YELLOW}⚠${NC} $container (not running)"
                ((WARNINGS++))
            fi
        done
    fi
fi

echo ""

# Section: Port Availability
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Port Availability${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

check_port_detailed 8000 "HTTP"
check_port_detailed 9000 "gRPC"
check_port_detailed 5432 "PostgreSQL"
check_port_detailed 6379 "Redis"
check_port_detailed 16686 "Jaeger UI"
check_port_detailed 9090 "Prometheus"
check_port_detailed 3000 "Grafana"

echo ""

# Section: Database Connectivity
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Database Connectivity${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "modulith_db"; then
    echo -n "  PostgreSQL connection: "
    if docker exec modulith_db pg_isready -U postgres > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} connected"

        # Try to query
        echo -n "  Database query test: "
        if docker exec modulith_db psql -U postgres -d postgres -c "SELECT 1" > /dev/null 2>&1; then
            echo -e "${GREEN}✓${NC} working"
        else
            echo -e "${YELLOW}⚠${NC} connection issues"
            ((WARNINGS++))
        fi
    else
        echo -e "${RED}✗${NC} not ready"
        ((ERRORS++))
    fi
else
    echo -e "${YELLOW}⚠${NC} Database container not running"
    echo "  Start with: make docker-up"
    ((WARNINGS++))
fi

echo ""

# Section: Redis Connectivity
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Redis Connectivity${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "modulith_redis"; then
    echo -n "  Redis connection: "
    if docker exec modulith_redis redis-cli ping > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} connected"
    else
        echo -e "${RED}✗${NC} not ready"
        ((ERRORS++))
    fi
else
    echo -e "${YELLOW}⚠${NC} Redis container not running"
    ((WARNINGS++))
fi

echo ""

# Section: Configuration Files
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Configuration Files${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

CONFIG_FILES=("configs/server.yaml" "go.mod" "go.sum" "docker-compose.yaml")
for file in "${CONFIG_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC} $file"
    else
        echo -e "${RED}✗${NC} $file (missing)"
        ((ERRORS++))
    fi
done

echo ""

# Section: Module Registration
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Module Registration${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

REGISTRY_FILE="cmd/server/setup/registry.go"
if [ -f "$REGISTRY_FILE" ]; then
    echo -e "${GREEN}✓${NC} $REGISTRY_FILE exists"

    # Check if RegisterModules function exists
    if grep -q "func RegisterModules" "$REGISTRY_FILE"; then
        echo -e "${GREEN}✓${NC} RegisterModules function found"

        # Count registered modules (lines with reg.Register)
        MODULE_COUNT=$(grep -c "reg.Register" "$REGISTRY_FILE" || echo "0")
        echo "  Registered modules: $MODULE_COUNT"

        # Check for common issues
        if grep -q "reg.Register.*NewModule()" "$REGISTRY_FILE"; then
            echo -e "${GREEN}✓${NC} Module registrations look correct"
        fi
    else
        echo -e "${RED}✗${NC} RegisterModules function not found"
        ((ERRORS++))
    fi
else
    echo -e "${RED}✗${NC} $REGISTRY_FILE not found"
    ((ERRORS++))
fi

echo ""

# Section: Build Check
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Build Check${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

echo -n "  Go modules: "
if go mod tidy -e > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} valid"
else
    echo -e "${YELLOW}⚠${NC} issues detected (run 'go mod tidy')"
    ((WARNINGS++))
fi

echo ""

# Summary
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
if [ $ERRORS -gt 0 ]; then
    echo -e "${RED}✗ Diagnostics completed with $ERRORS error(s)${NC}"
    if [ $WARNINGS -gt 0 ]; then
        echo -e "${YELLOW}  and $WARNINGS warning(s)${NC}"
    fi
    echo ""
    echo "Please fix the errors above."
    exit 1
elif [ $WARNINGS -gt 0 ]; then
    echo -e "${YELLOW}⚠ Diagnostics completed with $WARNINGS warning(s)${NC}"
    echo ""
    echo "System is functional but has some warnings."
    exit 0
else
    echo -e "${GREEN}✓ All diagnostics passed!${NC}"
    echo ""
    echo "Your development environment is healthy."
    exit 0
fi

