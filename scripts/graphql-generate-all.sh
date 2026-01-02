#!/bin/bash

# Script to generate GraphQL code for all modules that have schemas
# Auto-discovers modules with GraphQL schemas and generates code for each

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCHEMA_DIR="${PROJECT_ROOT}/internal/graphql/schema"

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

# Check if schema directory exists
if [ ! -d "${SCHEMA_DIR}" ]; then
    echo "❌ GraphQL schema directory not found: ${SCHEMA_DIR}"
    echo "   Run 'make graphql-init' to initialize GraphQL"
    exit 1
fi

echo "🔍 Discovering modules with GraphQL schemas..."

# Check if root schema exists
ROOT_SCHEMA="${SCHEMA_DIR}/schema.graphql"
if [ ! -f "${ROOT_SCHEMA}" ]; then
    echo "❌ Root schema not found: ${ROOT_SCHEMA}"
    echo "   Run 'make graphql-init' to initialize GraphQL"
    exit 1
fi

# Find all module schema files (excluding schema.graphql which is the root)
MODULES=()
for schema_file in "${SCHEMA_DIR}"/*.graphql; do
    if [ -f "${schema_file}" ]; then
        filename=$(basename "${schema_file}" .graphql)
        if [ "${filename}" != "schema" ]; then
            MODULES+=("${filename}")
        fi
    fi
done

if [ ${#MODULES[@]} -eq 0 ]; then
    echo "📦 Generating GraphQL code for root schema only"
    echo "   (No module schemas found - add module schemas as needed)"
else
    echo "📦 Found ${#MODULES[@]} module(s) with GraphQL schemas:"
    for module in "${MODULES[@]}"; do
        echo "   - ${module}"
    done
fi

echo ""
echo "🔄 Generating GraphQL code..."

# Generate for all schemas at once (gqlgen handles root + module schemas)
cd "${PROJECT_ROOT}"
if gqlgen generate 2>&1; then
    echo ""
    echo "✅ GraphQL code generated successfully for all modules!"
    echo ""
    echo "📝 Generated files:"
    echo "   - internal/graphql/generated/generated.go"
    echo "   - internal/graphql/generated/models_gen.go"
    echo "   - Resolver stubs in internal/graphql/resolver/"
else
    echo ""
    echo "⚠️  Generation completed with warnings"
    echo "   Check the output above for any schema errors"
    exit 1
fi

