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

echo "Scaffolding module: ${MODULE_NAME}..."

# Create directory structure
mkdir -p "${MODULE_DIR}/internal/service"
mkdir -p "${MODULE_DIR}/internal/repository"
mkdir -p "${MODULE_DIR}/internal/db/query"
mkdir -p "${MODULE_DIR}/resources/db/migration"
mkdir -p "${PROTO_DIR}"

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
process_template "templates/module/proto/module.proto.tmpl" "${PROTO_DIR}/${MODULE_NAME}.proto"
process_template "templates/module/air.toml.tmpl" ".air.${MODULE_NAME}.toml"

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
echo "Next steps:"
echo "1. Run 'make proto' to generate gRPC code."
echo "2. Run 'make sqlc' to generate DB code."
echo "3. Create cmd/${MODULE_NAME}/main.go for the module entry point."
echo "4. Update cmd/server/main.go to initialize the new module."
echo "5. Run 'make dev-module ${MODULE_NAME}' for hot-reload development."
