#!/bin/bash

# Script to completely reset the database by dropping all module schemas
# This is more thorough than rolling back migrations - it drops everything

# Load DB_DSN from .env file if it exists
if [ -f .env ]; then
    export $(grep -v '^#' .env | grep DB_DSN | xargs)
fi

if [ -z "$DB_DSN" ]; then
    echo "❌ Error: DB_DSN not set. Please set it in .env file or environment variable"
    exit 1
fi

echo "⚠️  WARNING: This will DROP ALL MODULE SCHEMAS and ALL DATA!"
echo "This includes:"
echo "  - All module schemas (auth, stock, order, etc.)"
echo "  - All tables in those schemas"
echo "  - All migration tracking tables"
echo "  - ALL DATA will be permanently deleted"
echo ""
read -p "Are you sure? Type 'yes' to confirm: " confirm

if [ "$confirm" != "yes" ]; then
    echo "❌ Confirmation failed. Aborting."
    exit 1
fi

echo ""
echo "🗑️  Dropping all module schemas..."

# Get list of all modules by looking at modules/ directory
MODULE_SCHEMAS=""
if [ -d "modules" ]; then
    for module_dir in modules/*/; do
        if [ -d "$module_dir" ]; then
            module_name=$(basename "$module_dir")
            # Check if this module has migrations (indicating it has a schema)
            if [ -d "${module_dir}resources/db/migration" ]; then
                MODULE_SCHEMAS="${MODULE_SCHEMAS} ${module_name}"
            fi
        fi
    done
fi

# Drop each module schema (CASCADE will drop all tables)
if [ -z "$MODULE_SCHEMAS" ]; then
    echo "  ℹ️  No modules with migrations found"
else
    for schema in $MODULE_SCHEMAS; do
        echo "  Dropping schema: $schema"
        psql "$DB_DSN" -c "DROP SCHEMA IF EXISTS ${schema} CASCADE;" 2>&1 | grep -v "NOTICE" || true

        # Also drop the migrations tracking table for this module (in public schema)
        echo "  Dropping migrations tracking table: ${schema}_schema_migrations"
        psql "$DB_DSN" -c "DROP TABLE IF EXISTS public.${schema}_schema_migrations CASCADE;" 2>&1 | grep -v "NOTICE" || true
    done
fi

echo ""
echo "✅ All module schemas dropped"
echo ""
echo "🔄 Running migrations to recreate schemas..."
if make migrate-up >/dev/null 2>&1; then
    echo "✅ Migrations completed successfully"
else
    echo "⚠️  Migration had errors (run 'make migrate-up' manually to see details)"
    exit 1
fi

echo ""
echo "🎉 Database reset complete!"

