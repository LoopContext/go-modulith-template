#!/bin/bash

# Script to generate GraphQL schemas from OpenAPI/Swagger files for all modules
# This automatically generates GraphQL schemas from proto definitions via OpenAPI

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OPENAPI_DIR="${PROJECT_ROOT}/gen/openapiv2/proto"
SCHEMA_DIR="${PROJECT_ROOT}/internal/graphql/schema"
TOOL_PATH="${PROJECT_ROOT}/scripts/graphql-from-proto/graphql-from-proto"

echo "🔄 Generating GraphQL schemas from OpenAPI definitions..."

# Check if GraphQL is initialized
if [ ! -d "${SCHEMA_DIR}" ]; then
    echo "❌ GraphQL not initialized. Run: just graphql-init"
    exit 1
fi

# Check if tool is built
if [ ! -f "${TOOL_PATH}" ]; then
    echo "🔨 Building graphql-from-proto tool..."
    cd "${PROJECT_ROOT}/scripts/graphql-from-proto"
    go build -o graphql-from-proto main.go
    cd "${PROJECT_ROOT}"
fi

# Find all OpenAPI files
MODULES=()
for swagger_file in "${OPENAPI_DIR}"/*/v1/*.swagger.json; do
    if [ -f "${swagger_file}" ]; then
        # Extract module name from path: gen/openapiv2/proto/{module}/v1/{module}.swagger.json
        module_path=$(dirname "$(dirname "${swagger_file}")")
        module_name=$(basename "${module_path}")
        MODULES+=("${module_name}")
    fi
done

if [ ${#MODULES[@]} -eq 0 ]; then
    echo "⚠️  No OpenAPI files found. Run 'just proto' first to generate OpenAPI definitions."
    exit 0
fi

echo "📦 Found ${#MODULES[@]} module(s) with OpenAPI definitions:"
for module in "${MODULES[@]}"; do
    echo "   - ${module}"
done

echo ""
echo "🔄 Generating GraphQL schemas..."

# Generate schema for each module
GENERATED=0
for module in "${MODULES[@]}"; do
    output_file="${SCHEMA_DIR}/${module}.graphql"

    if "${TOOL_PATH}" -module "${module}" 2>&1; then
        if [ -f "${output_file}" ]; then
            echo "✅ Generated ${output_file}"
            GENERATED=$((GENERATED + 1))
        fi
    else
        echo "⚠️  Failed to generate schema for ${module}"
    fi
done

echo ""
if [ $GENERATED -gt 0 ]; then
    echo "✅ Generated ${GENERATED} GraphQL schema(s)"
    echo ""
    echo "📝 Next steps:"
    echo "   1. Review and customize the generated schemas in ${SCHEMA_DIR}/"
    echo "   2. Run 'just graphql-generate-all' to generate resolver code"
    echo "   3. Implement resolvers in internal/graphql/resolver/"
else
    echo "⚠️  No schemas were generated"
fi

