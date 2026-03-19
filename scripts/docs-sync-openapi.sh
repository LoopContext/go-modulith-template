#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SRC_DIR="$ROOT_DIR/gen/openapiv2/proto"
DST_DIR="$ROOT_DIR/docs/api/openapi"

if [[ ! -d "$SRC_DIR" ]]; then
  echo "OpenAPI source directory not found: $SRC_DIR"
  echo "Run 'just be-proto' first to generate swagger files."
  exit 1
fi

mkdir -p "$DST_DIR"

# Keep destination in sync with generated specs.
rm -rf "$DST_DIR"/*
cp -R "$SRC_DIR"/* "$DST_DIR"/

echo "OK: OpenAPI specs synced to docs/api/openapi"
