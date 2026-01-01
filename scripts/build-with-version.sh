#!/bin/bash
# Build script with version information for Air
# Usage: ./scripts/build-with-version.sh <target> <output_bin>
# Example: ./scripts/build-with-version.sh server main
# Example: ./scripts/build-with-version.sh auth auth

TARGET=${1:-server}
OUTPUT_BIN=${2:-main}

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags "-X github.com/cmelgarejo/go-modulith-template/internal/version.Version=$VERSION -X github.com/cmelgarejo/go-modulith-template/internal/version.Commit=$COMMIT -X github.com/cmelgarejo/go-modulith-template/internal/version.BuildTime=$BUILD_TIME" -o ./bin/$OUTPUT_BIN ./cmd/$TARGET

