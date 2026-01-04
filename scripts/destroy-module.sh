#!/bin/bash

# Check if module name is provided
if [ -z "$1" ]; then
    echo "Usage: $0 <module_name>"
    exit 1
fi

MODULE_NAME=$1
MODULE_DIR="modules/${MODULE_NAME}"
PROTO_DIR="proto/${MODULE_NAME}"
PROTO_GEN_DIR="gen/go/proto/${MODULE_NAME}"
OPENAPI_GEN_DIR="gen/openapiv2/proto/${MODULE_NAME}"
CMD_DIR="cmd/${MODULE_NAME}"
CONFIG_FILE="configs/${MODULE_NAME}.yaml"
AIR_CONFIG=".air.${MODULE_NAME}.toml"
GRAPHQL_SCHEMA="internal/graphql/schema/${MODULE_NAME}.graphql"
GRAPHQL_RESOLVER="internal/graphql/resolver/${MODULE_NAME}.go"
REGISTRY_FILE="cmd/server/setup/registry.go"

echo "⚠️  WARNING: This will PERMANENTLY DELETE the module '${MODULE_NAME}' and all its data!"
echo "This includes:"
echo "  - All module files and directories"
echo "  - Database schema and all data"
echo "  - All migrations"
echo "  - Configuration files"
echo "  - GraphQL files (if any)"
echo ""
read -p "Type the module name to confirm deletion: " confirm

if [ "$confirm" != "$MODULE_NAME" ]; then
    echo "❌ Confirmation failed. Module name mismatch. Aborting."
    exit 1
fi

# Check if module directory exists
if [ ! -d "$MODULE_DIR" ]; then
    echo "❌ Error: Module directory '$MODULE_DIR' not found"
    exit 1
fi

# Load DB_DSN from .env file if it exists
if [ -f .env ]; then
    export $(grep -v '^#' .env | grep DB_DSN | xargs)
fi

# Step 1: Rollback all migrations and drop schema
echo ""
echo "🗄️  Step 1: Rolling back database migrations..."

MIGRATIONS_DIR="${MODULE_DIR}/resources/db/migration"
if [ -d "$MIGRATIONS_DIR" ] && [ -n "$DB_DSN" ]; then
    # Build module-specific DSN
    if echo "$DB_DSN" | grep -q "?"; then
        MODULE_DSN="${DB_DSN}&x-migrations-table=${MODULE_NAME}_schema_migrations"
    else
        MODULE_DSN="${DB_DSN}?x-migrations-table=${MODULE_NAME}_schema_migrations"
    fi

    # Get current version
    CURRENT_VERSION=$(migrate -path "$MIGRATIONS_DIR" -database "$MODULE_DSN" version 2>&1 | grep -E '^[0-9]+$' || echo "0")

    if [ "$CURRENT_VERSION" != "0" ] && [ "$CURRENT_VERSION" != "error: no migration" ]; then
        echo "  Current migration version: $CURRENT_VERSION"
        echo "  Rolling back all migrations..."

        # Rollback all migrations (use a large number to rollback all)
        migrate -path "$MIGRATIONS_DIR" -database "$MODULE_DSN" down 999999 2>&1 || true

        # Drop the schema (this also drops all tables)
        echo "  Dropping database schema '${MODULE_NAME}'..."
        psql "$DB_DSN" -c "DROP SCHEMA IF EXISTS ${MODULE_NAME} CASCADE;" 2>&1 | grep -v "NOTICE" || true

        # Drop the migrations tracking table
        echo "  Dropping migrations tracking table..."
        psql "$DB_DSN" -c "DROP TABLE IF EXISTS ${MODULE_NAME}_schema_migrations CASCADE;" 2>&1 | grep -v "NOTICE" || true

        echo "  ✅ Database cleanup completed"
    else
        echo "  No migrations found or database not initialized, skipping rollback"
        # Still try to drop schema and table in case they exist
        psql "$DB_DSN" -c "DROP SCHEMA IF EXISTS ${MODULE_NAME} CASCADE;" 2>&1 | grep -v "NOTICE" || true
        psql "$DB_DSN" -c "DROP TABLE IF EXISTS ${MODULE_NAME}_schema_migrations CASCADE;" 2>&1 | grep -v "NOTICE" || true
    fi
else
    echo "  ⚠️  Migrations directory not found or DB_DSN not set, skipping database cleanup"
fi

# Step 2: Remove module from sqlc.yaml
echo ""
echo "📝 Step 2: Removing module from sqlc.yaml..."
if [ -f "sqlc.yaml" ]; then
    # Use awk to remove the entire block that contains the module path
    # Each module block starts with "  - engine:" and ends before the next "  - engine:" or end of sql array
    awk -v module_name="${MODULE_NAME}" '
    BEGIN { skip_block = 0 }
    /^  - engine:/ {
        skip_block = 0
        line = $0
        getline
        # Check the queries/schema/out line for the module path
        if ($0 ~ "modules\/" module_name "\/") {
            skip_block = 1
            next
        } else {
            print line
            print
            next
        }
    }
    skip_block && /^  - engine:/ {
        skip_block = 0
        print
        next
    }
    skip_block {
        next
    }
    { print }
    ' sqlc.yaml > sqlc.yaml.tmp && mv sqlc.yaml.tmp sqlc.yaml

    echo "  ✅ Removed module entry from sqlc.yaml"
else
    echo "  ℹ️  sqlc.yaml not found, skipping"
fi

# Step 3: Remove module from registry.go
echo ""
echo "📝 Step 3: Removing module from registry.go..."
if [ -f "$REGISTRY_FILE" ]; then
    # Remove import line (match the full import path with quotes)
    # Remove registration line (match reg.Register calls)
    # Use a temporary file approach that works on both macOS and Linux
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS sed requires -i with an extension (empty string for no backup)
        # Match the full import line: "github.com/.../modules/patricio" (with optional leading whitespace)
        sed -i '' "/.*\"github.com\/cmelgarejo\/go-modulith-template\/modules\/${MODULE_NAME}\"/d" "$REGISTRY_FILE"
        # Match registration line: reg.Register(patricio.NewModule()) (with optional leading whitespace)
        sed -i '' "/.*reg\.Register(${MODULE_NAME}\.NewModule())/d" "$REGISTRY_FILE"
    else
        # Linux sed
        sed -i "/.*\"github.com\/cmelgarejo\/go-modulith-template\/modules\/${MODULE_NAME}\"/d" "$REGISTRY_FILE"
        sed -i "/.*reg\.Register(${MODULE_NAME}\.NewModule())/d" "$REGISTRY_FILE"
    fi

    echo "  ✅ Removed module registration from registry.go"
else
    echo "  ℹ️  registry.go not found, skipping"
fi

# Step 4: Delete all module files and directories
echo ""
echo "🗑️  Step 4: Deleting module files..."

# Delete module directory
if [ -d "$MODULE_DIR" ]; then
    rm -rf "$MODULE_DIR"
    echo "  ✅ Deleted $MODULE_DIR"
fi

# Delete proto directory
if [ -d "$PROTO_DIR" ]; then
    rm -rf "$PROTO_DIR"
    echo "  ✅ Deleted $PROTO_DIR"
fi

# Delete generated proto files
if [ -d "$PROTO_GEN_DIR" ]; then
    rm -rf "$PROTO_GEN_DIR"
    echo "  ✅ Deleted $PROTO_GEN_DIR"
fi

# Delete generated OpenAPI/Swagger files
if [ -d "$OPENAPI_GEN_DIR" ]; then
    rm -rf "$OPENAPI_GEN_DIR"
    echo "  ✅ Deleted $OPENAPI_GEN_DIR"
fi

# Delete cmd directory
if [ -d "$CMD_DIR" ]; then
    rm -rf "$CMD_DIR"
    echo "  ✅ Deleted $CMD_DIR"
fi

# Delete config file
if [ -f "$CONFIG_FILE" ]; then
    rm -f "$CONFIG_FILE"
    echo "  ✅ Deleted $CONFIG_FILE"
fi

# Delete air config
if [ -f "$AIR_CONFIG" ]; then
    rm -f "$AIR_CONFIG"
    echo "  ✅ Deleted $AIR_CONFIG"
fi

# Delete GraphQL files if they exist
if [ -f "$GRAPHQL_SCHEMA" ]; then
    rm -f "$GRAPHQL_SCHEMA"
    echo "  ✅ Deleted $GRAPHQL_SCHEMA"
fi

if [ -f "$GRAPHQL_RESOLVER" ]; then
    rm -f "$GRAPHQL_RESOLVER"
    echo "  ✅ Deleted $GRAPHQL_RESOLVER"
fi

echo ""
echo "✅ Module '${MODULE_NAME}' has been completely destroyed!"
echo ""

# Step 5: Regenerate code and tidy dependencies
echo "🔄 Step 5: Regenerating code and tidying dependencies..."
echo ""
echo "  Running 'make generate-all'..."
if make generate-all >/dev/null 2>&1; then
    echo "  ✅ Code generation completed"
else
    echo "  ⚠️  Code generation had errors (run 'make generate-all' manually to see details)"
fi

echo ""
echo "  Running 'make tidy'..."
if make tidy >/dev/null 2>&1; then
    echo "  ✅ Dependencies tidied"
else
    echo "  ⚠️  Tidy had errors (run 'make tidy' manually to see details)"
fi

echo ""
echo "🎉 Module destruction complete!"

