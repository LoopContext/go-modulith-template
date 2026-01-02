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

# Handle pluralization for table names and SQLC struct names
if [[ "${MODULE_NAME}" == *s ]]; then
    MODULE_NAME_PLURAL="${MODULE_NAME}"
    # SQLC usually singularizes table names for structs (e.g. products -> Product)
    # Simple singularization: remove trailing 's' then capitalize
    SINGULAR_NAME="${MODULE_NAME%s}"
    MODULE_STRUCT_NAME="$(tr '[:lower:]' '[:upper:]' <<< ${SINGULAR_NAME:0:1})${SINGULAR_NAME:1}"
else
    MODULE_NAME_PLURAL="${MODULE_NAME}s"
    MODULE_STRUCT_NAME="${MODULE_NAME_CAPITALIZED}"
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
        echo "     you can auto-generate the GraphQL schema with:"
        echo "     make graphql-from-proto-module MODULE_NAME=${MODULE_NAME}"
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
echo "Next steps:"
echo "1. Run 'make proto' to generate gRPC code."
echo "2. Run 'make sqlc' to generate DB code."

if [ -d "${GRAPHQL_SCHEMA_DIR}" ]; then
    echo "3. Edit ${GRAPHQL_SCHEMA_FILE} to define your GraphQL schema."
    echo "4. Run 'make graphql-generate-module MODULE_NAME=${MODULE_NAME}' to generate GraphQL code for this module."
    echo "   Or run 'make graphql-generate-all' to generate for all modules."
    echo "5. Implement resolvers in ${GRAPHQL_RESOLVER_FILE}."
    echo "6. Update cmd/server/main.go to register the new module:"
    echo "   Add: reg.Register(${MODULE_NAME}.NewModule())"
    echo "7. Run 'make dev-module ${MODULE_NAME}' for hot-reload development."
    echo "8. Or run 'make build-module ${MODULE_NAME}' to build standalone binary."
else
    echo "3. Update cmd/server/main.go to register the new module:"
    echo "   Add: reg.Register(${MODULE_NAME}.NewModule())"
    echo "4. Run 'make dev-module ${MODULE_NAME}' for hot-reload development."
    echo "5. Or run 'make build-module ${MODULE_NAME}' to build standalone binary."
    echo ""
    echo "💡 Tip: Run 'make graphql-init' to enable GraphQL support for future modules."
fi
