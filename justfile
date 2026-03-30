set dotenv-load
set shell := ["bash", "-cu"]

# Default values for environment variables
HTTP_PORT := env_var_or_default("HTTP_PORT", "8000")
WHATSAPP_ACCESS_TOKEN := env_var_or_default("WHATSAPP_ACCESS_TOKEN", "")
WHATSAPP_PHONE_NUMBER_ID := env_var_or_default("WHATSAPP_PHONE_NUMBER_ID", "")
WHATSAPP_APP_SECRET := env_var_or_default("WHATSAPP_APP_SECRET", "")
WHATSAPP_VERIFY_TOKEN := env_var_or_default("WHATSAPP_VERIFY_TOKEN", "")
TELEGRAM_BOT_TOKEN := env_var_or_default("TELEGRAM_BOT_TOKEN", "")

# Default goal
default: help

# Show available commands
help:
    @just --list --unsorted

# --- Aliases (Backward Compatibility) ---

# Run development environment diagnostics
doctor: be-doctor

# Run complete setup process (install deps, start docker, run migrations)
quickstart: be-quickstart

# Install developer tools
install-deps: be-install-deps

# Run full docker-compose stack
docker-up: be-docker-up

# Run minimal docker-compose stack
docker-up-minimal: be-docker-up-minimal

# Stop docker-compose services
docker-down: be-docker-down

# Generate all code (sqlc + proto + mocks)
generate-all: be-generate-all

# Run unit tests with fresh mocks
test-unit: be-test-unit

# Run linter
lint: be-lint

# Run linter with auto-fix enabled
lint-fix: be-lint-fix

# Run pre-commit checks (format + lint + test-unit)
pre-commit: be-pre-commit

# Run all tests (unit + integration)
test-all: be-test-all

# Run E2E tests for parimutuel flow
test-flow-e2e: be-test-e2e

# Run E2E test for reschedule-refund flow
test-e2e-reschedule: be-test-e2e-reschedule

# Run E2E tests for no-winners settlement (redistribute + void policies)
test-e2e-nowinners: be-test-e2e-nowinners

# Generate code from SQL
sqlc: be-sqlc

# Generate code from Protobuf
proto: be-proto

# Format code
format: be-format

# Tidy dependencies
tidy: be-tidy

# Scaffold a new module
new-module name: (be-new-module name)

# Destroy a module
destroy-module name: (be-destroy-module name)

# --- GraphQL Aliases ---
add-graphql: be-graphql-add
graphql-init: be-graphql-init
graphql-add: be-graphql-add
graphql-generate: be-graphql-generate
graphql-generate-all: be-graphql-generate-all
graphql-generate-module name: (be-graphql-generate-module name)
graphql-from-proto: be-graphql-from-proto

# --- Build Aliases ---
build-module name: (be-build-module name)
build-worker: be-build-worker
build-all: be-build-all

# --- Docker Aliases ---
docker-build: be-docker-build
docker-build-module name: (be-docker-build-module name)

# --- Dev Aliases ---
dev-worker: be-dev-worker
dev-module name: (be-dev-module name)

# Run admin panel E2E tests

# Run admin panel E2E tests in UI mode

# Run development server with automated setup
dev: be-setup
    @if ! command -v tmux > /dev/null; then \
        echo "Error: tmux is not installed. Please install it first (e.g., brew install tmux)"; \
        exit 1; \
    fi
    @if tmux has-session -t template 2>/dev/null; then \
        echo "Session 'template' already exists. Attaching..."; \
        tmux attach-session -t template; \
    else \
        echo "Starting tmux session 'template' (Backend, Terminal)..."; \
        tmux new-session -d -s template -n services; \
        tmux set-option -t template mouse on; \
        tmux set-option -t template history-limit 50000; \
        tmux split-window -v -t template:services.0 -p 50 -d; \
        tmux send-keys -t template:services.0 "just be-dev" C-m; \
        tmux select-pane -t template:services.1; \
        tmux attach-session -t template; \
    fi

# Stop the tmux development session
stop:
    @if tmux has-session -t template 2>/dev/null; then \
        echo "Stopping tmux session 'template'..."; \
        tmux kill-session -t template; \
        echo "✅ Dev environment stopped."; \
    else \
        echo "No active 'template' tmux session found."; \
    fi

# Run setup (infra + migrate + seed) non-interactively
setup: be-setup

# Run seed data
seed-data: be-seed

# Run example E2E flow to demonstrate the system
example: be-example

# Complete demo (setup + example)
demo: setup be-example

# Run tests
test: be-test

# Comprehensive ready-to-commit check (all lint, unit tests, web e2e, and build)
check: lint-fix test-unit build

# Fast check skipping interactive E2E tests
check-short: lint-fix test-unit build

# Full CI-like check including integration tests and all backend E2E (requires Docker)
check-full: check be-test-integration be-test-e2e be-test-e2e-reschedule be-test-e2e-nowinners

# Alias for migrate-up
migrate: be-migrate

# Run seed data for all modules
seed: be-seed

# Run seed data for test events
seed-test-events: (be-db-module-seed "events")

# Build the monolith binary
build: be-build

# Run the monolith server (without hot reload)
run: be-run

# Clean build artifacts
clean: be-clean

# Visualize module connections
visualize: be-visualize

# --- Backend: Setup & Diagnostics ---

# Install developer tools
be-install-deps:
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
    go install github.com/bufbuild/buf/cmd/buf@latest
    go install github.com/air-verse/air@latest
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    go install github.com/99designs/gqlgen@latest
    go install go.uber.org/mock/mockgen@latest

# Install gomock for test mocking
be-install-mocks:
    go install go.uber.org/mock/mockgen@latest

# Validate development environment setup
be-validate-setup:
    @./scripts/validate-setup.sh

# Run complete setup process (install deps, start docker, run migrations)
be-quickstart:
    @./scripts/quickstart.sh

# Run development environment diagnostics
be-doctor:
    @./scripts/doctor.sh

# --- Backend: Code Generation ---

# Generate type-safe Go code from SQL
be-sqlc:
    sqlc generate

# Generate gRPC code from protobuf definitions
be-proto:
    @echo "Cleaning gen/ directory..."
    rm -rf gen/
    buf generate --path proto

# Generate all code (sqlc + proto + mocks)
be-generate-all: be-sqlc be-proto be-generate-mocks

# Generate all mocks from interfaces
be-generate-mocks:
    @echo "Generating mocks..."
    @go generate ./modules/...
    @echo "Mocks generated successfully"

# Create a new API version for a module
be-proto-version-create module version:
    @./scripts/proto-version-create.sh {{module}} {{version}}
    @echo "✅ New version created. Run 'just be-proto' to generate code."

# Check for breaking changes in proto files
be-proto-breaking-check module="":
    @if [ -z "{{module}}" ]; then \
        ./scripts/proto-breaking-check.sh; \
    else \
        ./scripts/proto-breaking-check.sh {{module}}; \
    fi

# Lint all proto files
be-proto-lint:
    buf lint

# --- Backend: Docker & Services ---

# Run docker-compose
be-docker-up:
    docker-compose up -d

# Run docker-compose with minimal services (db + valkey only)
be-docker-up-minimal:
    docker-compose -f docker-compose.minimal.yaml up -d

# Stop docker-compose services
be-docker-down:
    docker-compose down

# --- Backend: Setup & Automated Flow ---

# Perform automated, non-interactive setup
be-setup: be-docker-up-minimal
    @./scripts/wait-for-db.sh
    @echo "🚀 Applying migrations..."
    @just be-migrate-up
    @echo "🌱 Seeding initial data..."
    @just be-seed
    @echo "✅ Setup complete"

# Run a representative example flow (E2E)
be-example:
    @echo "🚀 Running example E2E flow..."
    @just be-test-e2e
    @echo "✅ Example flow completed"

# --- Backend: Testing ---

# Run tests
be-test:
    go test -v -race -cover ./...

# Run unit tests with fresh mocks
be-test-unit: be-generate-mocks
    go test -v -race -short ./...

# Run integration tests (requires Docker)
be-test-integration:
    @echo "Running integration tests with testcontainers..."
    go test -v -run Integration ./...

# Run all tests (unit + integration)
be-test-all: be-test-unit be-test-integration

# Run all backend and frontend tests
test-full: be-test-all
    @echo "✅ All tests completed successfully!"

# Run E2E tests for parimutuel flow
be-test-e2e:
    @echo "🚀 Running E2E: Setup..."
    go run scripts/e2e/setup/main.go
    @echo "🚀 Running E2E: Placing positions..."
    go run scripts/e2e/positions/main.go
    @echo "🚀 Running E2E: Resolving and settling event..."
    go run scripts/e2e/resolve/main.go
    @echo "✅ E2E flow completed successfully!"

# Run E2E test for reschedule-refund flow (creator reschedules -> all positions refunded)
be-test-e2e-reschedule:
    @echo "🚀 Running E2E: Reschedule-Refund flow..."
    go run scripts/e2e/reschedule/main.go
    @echo "✅ E2E reschedule-refund flow completed successfully!"

# Run E2E test for no-winners settlement (redistribute + void policies)
be-test-e2e-nowinners:
    @echo "🚀 Running E2E: No-Winners settlement flow..."
    go run scripts/e2e/nowinners/main.go
    @echo "✅ E2E no-winners flow completed successfully!"

# Run tests with coverage report
be-test-coverage:
    go test -v -race -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out

# Generate detailed coverage report
be-coverage-report:
    @echo "=== 📊 Coverage Total del Proyecto ==="
    @echo ""
    @go test ./... -coverprofile=coverage.out -covermode=atomic 2>&1 | grep "coverage:" | grep -v "0.0%" | grep -v "no test"
    @echo ""
    @echo "=== 📈 Resumen por Componente ==="
    @go tool cover -func=coverage.out | grep -v "\.pb\.go" | grep -v "\.pb\.gw\.go" | grep -v "generated" | tail -20
    @echo ""
    @echo "=== 🎯 Coverage Total (sin código generado) ==="
    @go tool cover -func=coverage.out | grep -v "\.pb\.go" | grep -v "\.pb\.gw\.go" | grep -v "generated" | grep -v "cmd/" | tail -1
    @echo ""
    @echo "💡 Para ver el reporte HTML completo: just be-test-coverage"

# Open coverage report in browser
be-coverage-html:
    @go test ./... -coverprofile=coverage.out -covermode=atomic > /dev/null 2>&1
    @go tool cover -html=coverage.out

# --- Backend: Linting & Formatting ---

# Run linter
be-lint:
    golangci-lint run

# Run linter with auto-fix enabled
be-lint-fix:
    golangci-lint run --fix

# Format code with gofmt (and goimports if available)
be-format:
    @echo "Formatting code with gofmt..."
    @gofmt -w .
    @if command -v goimports > /dev/null; then \
        echo "Formatting imports with goimports..."; \
        goimports -w .; \
    else \
        echo "💡 Tip: Install goimports for import formatting: go install golang.org/x/tools/cmd/goimports@latest"; \
    fi
    @echo "✅ Code formatted"

# Tidy Go module dependencies
be-tidy:
    @echo "Tidying Go module dependencies..."
    @rm go.sum
    @go mod tidy
    @echo "✅ Dependencies tidied"

# Run pre-commit checks (format + lint + test-unit)
be-pre-commit: be-format be-lint be-test-unit

# --- Backend: Database & Migrations ---

# Run all module migrations
be-migrate-up:
    @echo "🚀 Running migrations for all modules..."
    go run ./cmd/server migrate

# Alias for migrate-up
be-migrate: be-migrate-up

# Run seed data for all modules
be-seed:
    @echo "🌱 Running seed data for all modules..."
    go run ./cmd/server seed

# Run seed data for DEV environment
be-seed-dev:
    @echo "🌱 Running seed data for DEV..."
    ENV=dev go run ./cmd/server seed

# Run seed data for PROD environment
be-seed-prod:
    @echo "🌱 Running seed data for PROD..."
    ENV=prod go run ./cmd/server seed

# Run admin task (Server Admin)
be-admin-task task:
    @echo "🔧 Running admin task: {{task}}"
    go run ./cmd/server admin {{task}}

# Rollback last migration for a specific module
be-migrate-down module:
    @MIGRATIONS_DIR=modules/{{module}}/resources/db/migration; \
    if [ ! -d "$$MIGRATIONS_DIR" ]; then \
        echo "Error: Module '{{module}}' not found or has no migrations directory"; \
        exit 1; \
    fi; \
    echo "⚠️  Rolling back last migration for module: {{module}}"; \
    go run ./cmd/server migrate-down

# Create a new migration file for a module
be-migrate-create module name:
    @MIGRATIONS_DIR=modules/{{module}}/resources/db/migration; \
    if [ ! -d "$$MIGRATIONS_DIR" ]; then \
        echo "Error: Module '{{module}}' not found or has no migrations directory"; \
        exit 1; \
    fi; \
    migrate create -ext sql -dir $$MIGRATIONS_DIR -seq {{name}}

# Force migration version to clean dirty state
be-migrate-force module version:
    @MIGRATIONS_DIR=modules/{{module}}/resources/db/migration; \
    if [ ! -d "$$MIGRATIONS_DIR" ]; then \
        echo "Error: Module '{{module}}' not found or has no migrations directory"; \
        exit 1; \
    fi; \
    echo "⚠️  Forcing migration version {{version}} for module {{module}} (clears dirty state)..."; \
    if echo "$$DB_DSN" | grep -q "?"; then \
        MODULE_DSN="$$DB_DSN&x-migrations-table={{module}}_schema_migrations"; \
    else \
        MODULE_DSN="$$DB_DSN?x-migrations-table={{module}}_schema_migrations"; \
    fi; \
    migrate -path $$MIGRATIONS_DIR -database "$$MODULE_DSN" force {{version}}

# Rollback ALL migrations for all modules (drops all tables)
be-db-down:
    #!/usr/bin/env bash
    echo "⚠️  WARNING: This will rollback ALL migrations for ALL modules (drops all tables)!"
    read -p "Are you sure? Type 'yes' to confirm: " confirm
    if [ "$confirm" == "yes" ]; then
        echo "🔄 Rolling back all migrations for all modules..."
        go run ./cmd/server migrate-down
    else
        echo "❌ Confirmation failed. Aborting."
        exit 1
    fi

# FORCIBLY drop all module schemas (guaranteed clean state)
be-db-nuke:
    #!/usr/bin/env bash
    echo "⚠️  WARNING: This will FORCIBLY DROP ALL SCHEMAS and ALL DATA for ALL modules!"
    read -p "Are you sure? Type 'yes' to confirm: " confirm
    if [ "$confirm" == "yes" ]; then
        echo "🔥 Nuking all module schemas..."
        # Note: migrate-nuke logic is currently not in server binary.
        # This target should likely be updated once implemented.
        echo "Error: migrate-nuke not yet implemented in server binary."
        exit 1
    else
        echo "❌ Confirmation failed. Aborting."
        exit 1
    fi

# FORCIBLY drop a specific module schema
be-db-module-nuke module:
    #!/usr/bin/env bash
    echo "⚠️  WARNING: This will FORCIBLY DROP SCHEMA and DATA for module '{{module}}'!"
    read -p "Are you sure? Type 'yes' to confirm: " confirm
    if [ "$confirm" == "yes" ]; then
        echo "🔥 Nuking module '{{module}}' schema..."
        # Note: migrate-nuke-module logic is currently not in server binary.
        echo "Error: migrate-nuke-module not yet implemented in server binary."
        exit 1
    else
        echo "❌ Confirmation failed. Aborting."
        exit 1
    fi

# Run migrations for a specific module
be-db-module-migrate module:
    @echo "🚀 Running migrations for module: {{module}}..."
    go run ./cmd/server migrate

# Run seed data for a specific module
be-db-module-seed module:
    @echo "🌱 Running seed data for module: {{module}}..."
    go run ./cmd/server seed-module {{module}}

# Drop all module schemas and re-run all migrations (destructive, asks for confirmation)
be-db-reset:
    @./scripts/db-reset.sh

# Alias for migrate-up
be-db-migrate: be-migrate-up

# Alias for seed
be-db-seed:
    @echo "🌱 Running seed data for all modules..."
    just be-seed

# Complete reset and re-initialization of the whole db
be-db-reinit:
    #!/usr/bin/env bash
    set -e
    echo "⚠️  WARNING: This will reset and re-initialize the whole db!"
    read -p "Are you sure? Type 'yes' to confirm: " confirm
    if [ "$confirm" == "yes" ]; then
        echo "🔄 Nuking and re-initializing the whole db..."
        # Pipe 'yes' to db-nuke which expects 'yes'
        yes yes | just be-db-nuke
        just be-db-migrate
        just be-db-seed
    else
        echo "❌ Confirmation failed. Aborting."
        exit 1
    fi

# --- Backend: Build ---

VERSION := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
COMMIT := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
BUILD_TIME := `date -u +"%Y-%m-%dT%H:%M:%SZ"`
LDFLAGS := "-X github.com/LoopContext/go-modulith-template/internal/appversion.Version=" + VERSION + " " + \
           "-X github.com/LoopContext/go-modulith-template/internal/appversion.Commit=" + COMMIT + " " + \
           "-X github.com/LoopContext/go-modulith-template/internal/appversion.BuildTime=" + BUILD_TIME

# Build the monolith binary
be-build:
    @mkdir -p bin
    go build -ldflags "{{LDFLAGS}}" -o bin/server ./cmd/server

# Build a specific module binary
be-build-module module:
    @if [ ! -d "cmd/{{module}}" ]; then echo "Error: Module '{{module}}' not found in cmd/"; exit 1; fi
    @mkdir -p bin
    @echo "Building module: {{module}}"
    go build -ldflags "{{LDFLAGS}}" -o bin/{{module}} ./cmd/{{module}}/main.go

# Build the worker binary
be-build-worker:
    @mkdir -p bin
    go build -ldflags "{{LDFLAGS}}" -o bin/worker ./cmd/worker/main.go

# Build all binaries (server + worker + all modules)
be-build-all: be-build be-build-worker
    @mkdir -p bin
    @for dir in cmd/*/; do \
        module=$$(basename $$dir); \
        if [ "$$module" != "server" ] && [ "$$module" != "worker" ]; then \
            echo "Building module: $$module"; \
            go build -ldflags "{{LDFLAGS}}" -o bin/$$module ./cmd/$$module/main.go; \
        fi \
    done

# Clean build artifacts
be-clean:
    rm -rf bin/

# --- Backend: Development & Execution ---

# Run the monolith server (without hot reload)
be-run:
    go run -ldflags "{{LDFLAGS}}" ./cmd/server/ || true

# Run the monolith with live reload (requires Air)
be-dev:
    @./scripts/preflight-check.sh || exit 1
    @if command -v air > /dev/null; then \
        air -c .air.toml; \
    else \
        echo "Air is not installed. Please install it with: go install github.com/air-verse/air@latest"; \
    fi

# Run the worker with live reload (requires Air)
be-dev-worker:
    @./scripts/preflight-check.sh || exit 1
    @if command -v air > /dev/null; then \
        air -c .air.worker.toml; \
    else \
        echo "Air is not installed. Please install it with: go install github.com/air-verse/air@latest"; \
    fi

# Run a specific module with live reload
be-dev-module module:
    @if [ ! -f ".air.{{module}}.toml" ]; then echo "Error: Air config '.air.{{module}}.toml' not found"; exit 1; fi
    @./scripts/preflight-check.sh || exit 1
    @if command -v air > /dev/null; then \
        echo "Starting module: {{module}} with hot reload..."; \
        air -c .air.{{module}}.toml; \
    else \
        echo "Air is not installed. Please install it with: go install github.com/air-verse/air@latest"; \
    fi

# --- Backend: Docker Build ---

# Build docker image for server
be-docker-build:
    docker build \
        --build-arg TARGET=server \
        --build-arg VERSION={{VERSION}} \
        --build-arg COMMIT={{COMMIT}} \
        --build-arg BUILD_TIME={{BUILD_TIME}} \
        -t template-server:latest .

# Build docker image for a specific module
be-docker-build-module module:
    @if [ ! -d "cmd/{{module}}" ]; then echo "Error: Module '{{module}}' not found in cmd/"; exit 1; fi
    @echo "Building Docker image for module: {{module}}"
    docker build \
        --build-arg TARGET={{module}} \
        --build-arg VERSION={{VERSION}} \
        --build-arg COMMIT={{COMMIT}} \
        --build-arg BUILD_TIME={{BUILD_TIME}} \
        -t template-{{module}}:latest .

# --- Backend: Modules ---

# Scaffold a new module
be-new-module module:
    ./scripts/scaffold-module.sh {{module}}

# Destroy a module completely
be-destroy-module module:
    ./scripts/destroy-module.sh {{module}}

# --- Backend: GraphQL ---

# Add optional GraphQL support using gqlgen (automatically generates code)
be-graphql-add:
    ./scripts/graphql-add-to-project.sh

# Initialize GraphQL (alias for graphql-add)
be-graphql-init: be-graphql-add

# Generate GraphQL code for all modules
be-graphql-generate: be-graphql-generate-all

# Generate GraphQL code for a specific module
be-graphql-generate-module module:
    ./scripts/graphql-generate-module.sh {{module}}

# Generate GraphQL code for all modules (auto-discovers modules with schemas)
be-graphql-generate-all:
    ./scripts/graphql-generate-all.sh

# Generate GraphQL schemas from OpenAPI/Swagger files for all modules
be-graphql-from-proto:
    ./scripts/graphql-from-proto-all.sh

# Validate GraphQL schema (dummy, following Makefile)
be-graphql-validate:
    @echo "GraphQL validation not implemented"

# --- Backend: Visualization ---

# Visualize module connections
be-visualize format="html" serve="false":
    @echo "🔍 Analyzing Modulith modulith architecture..."
    @if [ "{{serve}}" = "true" ]; then \
        go run ./cmd/visualize/main.go -format={{format}} -serve; \
    else \
        go run ./cmd/visualize/main.go -format={{format}}; \
    fi

# --- Backend: Messaging & Bot ---

# Register a messaging provider (whatsapp or telegram)
be-bot-register integration:
    #!/usr/bin/env bash
    if [ "{{integration}}" == "whatsapp" ]; then
        echo "Registering WhatsApp provider..."
        curl -X POST http://localhost:{{HTTP_PORT}}/v1/messaging/providers \
            -H "Content-Type: application/json" \
            -d '{ \
                "name": "Local WhatsApp", \
                "type": "PROVIDER_TYPE_WHATSAPP", \
                "config": "{\"access_token\":\"{{WHATSAPP_ACCESS_TOKEN}}\", \"phone_number_id\":\"{{WHATSAPP_PHONE_NUMBER_ID}}\", \"app_secret\":\"{{WHATSAPP_APP_SECRET}}\"}", \
                "webhook_verify_token": "{{WHATSAPP_VERIFY_TOKEN}}" \
            }'
    elif [ "{{integration}}" == "telegram" ]; then
        echo "Registering Telegram provider..."
        curl -X POST http://localhost:{{HTTP_PORT}}/v1/messaging/providers \
            -H "Content-Type: application/json" \
            -d '{ \
                "name": "Local Telegram", \
                "type": "PROVIDER_TYPE_TELEGRAM", \
                "config": "{\"token\":\"{{TELEGRAM_BOT_TOKEN}}\"}" \
            }'
    else
        echo "Unsupported integration: {{integration}}"
        exit 1
    fi

# Simulate a WhatsApp message webhook
be-bot-simulate-wa-msg provider_id contact_id text="hello":
    @echo "Simulating WhatsApp message: {{text}}"
    @curl -X POST http://localhost:{{HTTP_PORT}}/v1/messaging/webhook/{{provider_id}} \
        -H "Content-Type: application/json" \
        -H "X-Hub-Signature-256: sha256=MOCK_SIGNATURE" \
        -d '{ \
            "object": "whatsapp_business_account", \
            "entry": [{ \
                "changes": [{ \
                    "field": "messages", \
                    "value": { \
                        "messages": [{ \
                            "from": "{{contact_id}}", \
                            "id": "wamid.MOCK_ID", \
                            "text": { "body": "{{text}}" }, \
                            "type": "text" \
                        }] \
                    } \
                }] \
            }] \
        }'

# Simulate a Telegram message webhook
be-bot-simulate-tg-msg provider_id contact_id text="hello":
    @echo "Simulating Telegram message: {{text}}"
    @curl -X POST http://localhost:{{HTTP_PORT}}/v1/messaging/webhook/{{provider_id}} \
        -H "Content-Type: application/json" \
        -d '{ \
            "update_id": 12345, \
            "message": { \
                "message_id": 1, \
                "from": { "id": {{contact_id}}, "first_name": "TestUser" }, \
                "chat": { "id": {{contact_id}}, "type": "private" }, \
                "text": "{{text}}" \
            } \
        }'

# Link a messaging contact to a user
be-bot-link-user contact_id user_id:
    @echo "Linking contact {{contact_id}} to user {{user_id}}..."
    @curl -X POST http://localhost:{{HTTP_PORT}}/v1/messaging/contacts/{{contact_id}}/link \
        -H "Content-Type: application/json" \
        -d '{ \
            "user_id": "{{user_id}}" \
        }'

# --- Documentation Site (MkDocs) ---

# Install docs site dependencies in local venv
docs-install:
    @PYTHON_BIN="$(command -v python || command -v python3)"; \
    if [ -z "$PYTHON_BIN" ]; then \
        echo "❌ Python not found. Install python3 and retry."; \
        exit 1; \
    fi; \
    "$PYTHON_BIN" -m venv .venv-docs
    . .venv-docs/bin/activate && pip install -r docs/docs-site/requirements.txt

# Serve docs site locally (default: http://127.0.0.1:8001)
docs-serve port="8001":
    @just docs-build
    . .venv-docs/bin/activate && mkdocs serve -f docs/docs-site/mkdocs.yml -a 127.0.0.1:{{port}}

# Build docs site static output into temp/docs-site
docs-build:
    @if [ ! -d ".venv-docs" ]; then just docs-install; fi
    @./scripts/docs-i18n-check.sh
    @./scripts/docs-sync-openapi.sh
    . .venv-docs/bin/activate && mkdocs build -f docs/docs-site/mkdocs.yml

# Validate ES/EN documentation parity map
docs-i18n-check:
    @./scripts/docs-i18n-check.sh

# Sync generated backend OpenAPI specs into docs/api/openapi
docs-openapi-sync:
    @./scripts/docs-sync-openapi.sh
