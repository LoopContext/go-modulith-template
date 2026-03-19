#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
MAP_FILE="$ROOT_DIR/docs/docs-site/i18n-map.csv"

if [[ ! -f "$MAP_FILE" ]]; then
  echo "i18n map not found: $MAP_FILE"
  exit 1
fi

missing=0
while IFS=, read -r es_path en_path; do
  [[ "$es_path" == "es_path" ]] && continue

  if [[ ! -f "$ROOT_DIR/docs/$es_path" ]]; then
    echo "Missing ES file: docs/$es_path"
    missing=1
  fi

  if [[ ! -f "$ROOT_DIR/docs/$en_path" ]]; then
    echo "Missing EN file: docs/$en_path"
    missing=1
  fi
done < "$MAP_FILE"

if [[ $missing -ne 0 ]]; then
  exit 1
fi

echo "OK: ES/EN mapped files exist"
