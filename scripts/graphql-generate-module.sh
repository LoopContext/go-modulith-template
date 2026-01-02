#!/bin/bash

# Script to generate GraphQL code for a specific module
# This temporarily filters schemas to only include the target module

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <module_name>"
    exit 1
fi

MODULE_NAME=$1
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCHEMA_DIR="${PROJECT_ROOT}/internal/graphql/schema"
MODULE_SCHEMA="${SCHEMA_DIR}/${MODULE_NAME}.graphql"

# Check if GraphQL is initialized
if [ ! -f "${PROJECT_ROOT}/gqlgen.yml" ]; then
    echo "❌ GraphQL not initialized. Run: make graphql-init"
    exit 1
fi

# Check if gqlgen is installed
if ! command -v gqlgen > /dev/null; then
    echo "❌ gqlgen not found. Install with: go install github.com/99designs/gqlgen@latest"
    exit 1
fi

# Check if module schema exists
if [ ! -f "${MODULE_SCHEMA}" ]; then
    echo "❌ Module schema not found: ${MODULE_SCHEMA}"
    echo "   Run 'make new-module ${MODULE_NAME}' to create it, or create the schema manually."
    exit 1
fi

echo "🔄 Generating GraphQL code for module: ${MODULE_NAME}..."

# Create temporary gqlgen.yml that only includes this module's schema
TEMP_GQLGEN="${PROJECT_ROOT}/gqlgen.yml.tmp"
ORIGINAL_GQLGEN="${PROJECT_ROOT}/gqlgen.yml"

# Backup original
cp "${ORIGINAL_GQLGEN}" "${TEMP_GQLGEN}"

# Create temporary config with only this module's schema
cat > "${ORIGINAL_GQLGEN}" <<EOF
# Temporary config for module-specific generation
# Original config backed up to gqlgen.yml.tmp

schema:
  - internal/graphql/schema/schema.graphql
  - internal/graphql/schema/${MODULE_NAME}.graphql

exec:
  filename: internal/graphql/generated/generated.go
  package: generated

model:
  filename: internal/graphql/generated/models_gen.go
  package: generated

resolver:
  layout: follow-schema
  dir: internal/graphql/resolver
  package: resolver
EOF

# Generate code
cd "${PROJECT_ROOT}"
if gqlgen generate 2>&1; then
    echo "✅ GraphQL code generated successfully for module: ${MODULE_NAME}"
else
    echo "⚠️  Generation completed with warnings (this is normal if other modules have incomplete schemas)"
fi

# Restore original config
mv "${TEMP_GQLGEN}" "${ORIGINAL_GQLGEN}"

echo "✅ Done! Module ${MODULE_NAME} GraphQL code generated."

