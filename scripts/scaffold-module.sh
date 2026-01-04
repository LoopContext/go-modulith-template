#!/bin/bash

# Check if module name is provided
if [ -z "$1" ]; then
    echo "Usage: $0 <module_name>"
    exit 1
fi

MODULE_NAME=$1
PROJECT_NAME="github.com/cmelgarejo/go-modulith-template"

# Capitalize first letter
MODULE_NAME_CAPITALIZED="$(tr '[:lower:]' '[:upper:]' <<< ${MODULE_NAME:0:1})${MODULE_NAME:1}"

# Ask if table name should be plural
echo ""
echo "Is the table name plural? (e.g., 'users', 'orders', 'products')"
if [[ "${MODULE_NAME}" == *s ]]; then
    echo "  [Y/n] (default: Y - module name '${MODULE_NAME}' already appears plural): "
else
    echo "  [Y/n] (default: n - module name '${MODULE_NAME}' appears singular): "
fi
read -r IS_PLURAL

# Determine if table should be plural
if [[ -z "$IS_PLURAL" ]]; then
    # Default behavior: plural if module name ends with 's'
    if [[ "${MODULE_NAME}" == *s ]]; then
        IS_PLURAL="y"
    else
        IS_PLURAL="n"
    fi
fi

# Handle pluralization for table names and SQLC struct names
if [[ "$IS_PLURAL" =~ ^[Yy] ]]; then
    if [[ "${MODULE_NAME}" == *s ]]; then
        MODULE_NAME_PLURAL="${MODULE_NAME}"
        # SQLC usually singularizes table names for structs (e.g. products -> Product)
        # Simple singularization: remove trailing 's' then capitalize
        SINGULAR_NAME="${MODULE_NAME%s}"
        MODULE_STRUCT_NAME="$(tr '[:lower:]' '[:upper:]' <<< ${SINGULAR_NAME:0:1})${SINGULAR_NAME:1}"
        # SQLC generates struct names as: SchemaName + TableName (singularized)
        # e.g., stock.stocks -> StockStock, auth.users -> AuthUser
        SQLC_STRUCT_NAME="${MODULE_NAME_CAPITALIZED}${MODULE_STRUCT_NAME}"
    else
        MODULE_NAME_PLURAL="${MODULE_NAME}s"
        MODULE_STRUCT_NAME="${MODULE_NAME_CAPITALIZED}"
        SQLC_STRUCT_NAME="${MODULE_NAME_CAPITALIZED}${MODULE_STRUCT_NAME}"
    fi
else
    MODULE_NAME_PLURAL="${MODULE_NAME}"
    MODULE_STRUCT_NAME="${MODULE_NAME_CAPITALIZED}"
    # For singular table names, SQLC struct is: SchemaName + TableName
    SQLC_STRUCT_NAME="${MODULE_NAME_CAPITALIZED}${MODULE_STRUCT_NAME}"
fi

MODULE_DIR="modules/${MODULE_NAME}"
PROTO_DIR="proto/${MODULE_NAME}/v1"
CMD_DIR="cmd/${MODULE_NAME}"
CONFIG_DIR="configs"

echo "Scaffolding module: ${MODULE_NAME}..."

# Create directory structure
mkdir -p "${MODULE_DIR}/internal/service"
mkdir -p "${MODULE_DIR}/internal/repository"
mkdir -p "${MODULE_DIR}/internal/db/query"
mkdir -p "${MODULE_DIR}/resources/db/migration"
mkdir -p "${MODULE_DIR}/resources/db/seed"
mkdir -p "${PROTO_DIR}"
mkdir -p "${CMD_DIR}"
mkdir -p "${CONFIG_DIR}"

# Helper function to process templates
process_template() {
    local src=$1
    local dest=$2
    sed -e "s/{{.PROJECT_NAME}}/${PROJECT_NAME//\//\\/}/g" \
        -e "s/{{.MODULE_NAME}}/${MODULE_NAME}/g" \
        -e "s/{{.MODULE_NAME_CAPITALIZED}}/${MODULE_NAME_CAPITALIZED}/g" \
        -e "s/{{.MODULE_STRUCT_NAME}}/${MODULE_STRUCT_NAME}/g" \
        -e "s/{{.SQLC_STRUCT_NAME}}/${SQLC_STRUCT_NAME}/g" \
        -e "s/{{.MODULE_NAME_PLURAL}}/${MODULE_NAME_PLURAL}/g" \
        "$src" > "$dest"
}

# Process each template
process_template "templates/module/module.go.tmpl" "${MODULE_DIR}/module.go"
process_template "templates/module/internal/service/service.go.tmpl" "${MODULE_DIR}/internal/service/service.go"
process_template "templates/module/internal/repository/repository.go.tmpl" "${MODULE_DIR}/internal/repository/repository.go"
process_template "templates/module/internal/db/query/queries.sql.tmpl" "${MODULE_DIR}/internal/db/query/${MODULE_NAME}.sql"
process_template "templates/module/resources/db/migration/000001_initial.up.sql.tmpl" "${MODULE_DIR}/resources/db/migration/000001_initial_schema.up.sql"
process_template "templates/module/resources/db/migration/000001_initial.down.sql.tmpl" "${MODULE_DIR}/resources/db/migration/000001_initial_schema.down.sql"
process_template "templates/module/resources/db/seed/001_example_data.sql.tmpl" "${MODULE_DIR}/resources/db/seed/001_example_data.sql"
process_template "templates/module/proto/module.proto.tmpl" "${PROTO_DIR}/${MODULE_NAME}.proto"
process_template "templates/module/air.toml.tmpl" ".air.${MODULE_NAME}.toml"
process_template "templates/module/cmd/main.go.tmpl" "${CMD_DIR}/main.go"
process_template "templates/module/configs/config.yaml.tmpl" "${CONFIG_DIR}/${MODULE_NAME}.yaml"

# Check if GraphQL is initialized and create GraphQL files
GRAPHQL_SCHEMA_DIR="internal/graphql/schema"
GRAPHQL_RESOLVER_DIR="internal/graphql/resolver"
GRAPHQL_SCHEMA_FILE="${GRAPHQL_SCHEMA_DIR}/${MODULE_NAME}.graphql"
GRAPHQL_RESOLVER_FILE="${GRAPHQL_RESOLVER_DIR}/${MODULE_NAME}.go"

if [ -d "${GRAPHQL_SCHEMA_DIR}" ]; then
    echo "📊 GraphQL detected - creating GraphQL schema and resolver files..."

    # Create GraphQL schema file
    if [ ! -f "${GRAPHQL_SCHEMA_FILE}" ]; then
        process_template "templates/module/graphql/schema.graphql.tmpl" "${GRAPHQL_SCHEMA_FILE}"
        echo "  ✅ Created ${GRAPHQL_SCHEMA_FILE}"
        echo ""
        echo "  💡 Tip: After defining your proto file and running 'make proto',"
        echo "     run 'make graphql-generate-module MODULE_NAME=${MODULE_NAME}'"
        echo "     which will auto-generate the schema from proto if missing"
    else
        echo "  ℹ️  ${GRAPHQL_SCHEMA_FILE} already exists, skipping..."
    fi

    # Create GraphQL resolver file
    if [ ! -f "${GRAPHQL_RESOLVER_FILE}" ]; then
        process_template "templates/module/graphql/resolver.go.tmpl" "${GRAPHQL_RESOLVER_FILE}"
        echo "  ✅ Created ${GRAPHQL_RESOLVER_FILE}"
    else
        echo "  ℹ️  ${GRAPHQL_RESOLVER_FILE} already exists, skipping..."
    fi
else
    echo "  ℹ️  GraphQL not initialized - skipping GraphQL files"
    echo "     Run 'make graphql-init' to enable GraphQL support"
fi

# Update sqlc.yaml
if ! grep -q "modules/${MODULE_NAME}/internal/db/store" sqlc.yaml; then
    echo "Updating sqlc.yaml..."
    cat >> sqlc.yaml <<EOF
  - engine: "postgresql"
    queries: "modules/${MODULE_NAME}/internal/db/query/"
    schema: "modules/${MODULE_NAME}/resources/db/migration/000001_initial_schema.up.sql"
    gen:
      go:
        package: "store"
        out: "modules/${MODULE_NAME}/internal/db/store"
        sql_package: "database/sql"
        emit_interface: true
        emit_json_tags: true
EOF
fi

# Register module in registry.go
register_module_in_registry() {
    local registry_file="cmd/server/setup/registry.go"
    local import_path="github.com/cmelgarejo/go-modulith-template/modules/${MODULE_NAME}"

    if [ ! -f "$registry_file" ]; then
        echo "⚠️  Warning: ${registry_file} not found, skipping auto-registration"
        return
    fi

    # Check if module is already registered
    if grep -q "reg.Register(${MODULE_NAME}.NewModule())" "$registry_file"; then
        echo "ℹ️  Module ${MODULE_NAME} is already registered in ${registry_file}"
        return
    fi

    # Determine sed command based on OS
    if [[ "$OSTYPE" == "darwin"* ]]; then
        SED_IN_PLACE="sed -i ''"
    else
        SED_IN_PLACE="sed -i"
    fi

    # Use awk to modify the file - more reliable than sed for complex operations
    local register_line_text="\treg.Register(${MODULE_NAME}.NewModule())"
    awk -v module_name="${MODULE_NAME}" \
        -v import_path="${import_path}" \
        -v register_line="${register_line_text}" '
    BEGIN {
        import_added = 0
        register_added = 0
        in_import_block = 0
        in_register_func = 0
        last_import_line = ""
    }

    # Track import block
    /^import \(/ {
        in_import_block = 1
        print
        next
    }
    in_import_block && /^\)/ {
        # Before closing import block, add import if not already present
        if (!import_added) {
            print "\t\"" import_path "\""
            import_added = 1
        }
        in_import_block = 0
        print
        next
    }
    in_import_block {
        # Check if our import already exists
        if (index($0, "modules/" module_name) > 0) {
            import_added = 1
        }
        # Track the last import line to insert after it
        last_import_line = $0
        print
        next
    }

    # Track RegisterModules function
    /^func RegisterModules/ {
        in_register_func = 1
        print
        next
    }

    in_register_func && /^\t\/\/ Add more modules as needed:/ {
        # Add registration before the comment
        if (!register_added) {
            print register_line
            register_added = 1
        }
        print
        next
    }

    in_register_func && /^}$/ && !register_added {
        # Add registration before closing brace if comment not found
        print register_line
        register_added = 1
        print
        in_register_func = 0
        next
    }

    in_register_func && /^}$/ {
        in_register_func = 0
        print
        next
    }

    # Default: print the line
    { print }
    ' "$registry_file" > "${registry_file}.tmp" && mv "${registry_file}.tmp" "$registry_file"

    if grep -q "\"${import_path}\"" "$registry_file"; then
        echo "  ✅ Added import for ${MODULE_NAME} module"
    fi
    if grep -q "reg.Register(${MODULE_NAME}.NewModule())" "$registry_file"; then
        echo "  ✅ Registered ${MODULE_NAME} module in ${registry_file}"
    fi
}

echo ""
echo "🔧 Registering module in registry..."
register_module_in_registry

echo ""
echo "⚙️  Running code generation (make generate-all)..."
if make generate-all; then
    echo ""
    echo "  ✅ Code generation completed successfully"
else
    echo ""
    echo "  ❌ Error: Code generation failed!"
    echo "     Please run 'make generate-all' manually to see the errors"
    exit 1
fi

echo "Module ${MODULE_NAME} scaffolded successfully!"
echo ""
echo "Generated files:"
echo "  - ${MODULE_DIR}/module.go"
echo "  - ${MODULE_DIR}/internal/service/service.go"
echo "  - ${MODULE_DIR}/internal/repository/repository.go"
echo "  - ${CMD_DIR}/main.go (standalone service entry point)"
echo "  - ${CONFIG_DIR}/${MODULE_NAME}.yaml (module configuration)"
echo "  - .air.${MODULE_NAME}.toml (hot reload config)"

if [ -d "${GRAPHQL_SCHEMA_DIR}" ]; then
    echo "  - ${GRAPHQL_SCHEMA_FILE} (GraphQL schema)"
    echo "  - ${GRAPHQL_RESOLVER_FILE} (GraphQL resolver)"
fi

echo ""
echo "✅ Module setup complete! Next steps:"

if [ -d "${GRAPHQL_SCHEMA_DIR}" ]; then
    echo ""
    echo "1. Edit ${GRAPHQL_SCHEMA_FILE} to define your GraphQL schema."
    echo "2. Run 'make graphql-generate-module MODULE_NAME=${MODULE_NAME}' to generate GraphQL code for this module."
    echo "   Or run 'make graphql-generate-all' to generate for all modules."
    echo "3. Implement resolvers in ${GRAPHQL_RESOLVER_FILE}."
    echo "4. Run 'make dev-module ${MODULE_NAME}' for hot-reload development."
    echo "   Or run 'make build-module ${MODULE_NAME}' to build standalone binary."
else
    echo ""
    echo "1. Run 'make dev-module ${MODULE_NAME}' for hot-reload development."
    echo "   Or run 'make build-module ${MODULE_NAME}' to build standalone binary."
    echo ""
    echo "💡 Tip: Run 'make graphql-init' to enable GraphQL support for future modules."
fi
