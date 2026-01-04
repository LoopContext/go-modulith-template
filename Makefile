.PHONY: help sqlc proto install-deps install-mocks generate-mocks generate-all test-unit graphql-init graphql-generate graphql-generate-module graphql-generate-all graphql-validate graphql-add graphql-from-proto validate-setup quickstart doctor format tidy pre-commit
.DEFAULT_GOAL := help

help: ## Show available commands
	@echo "Available commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-30s\033[0m %s\n", $$1, $$2}'

install-deps: ## Install developer tools
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install github.com/air-verse/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/99designs/gqlgen@latest
	go install go.uber.org/mock/mockgen@latest

install-mocks: ## Install gomock for test mocking
	go install go.uber.org/mock/mockgen@latest

validate-setup: ## Validate development environment setup
	@./scripts/validate-setup.sh

quickstart: ## Run complete setup process (install deps, start docker, run migrations)
	@./scripts/quickstart.sh

doctor: ## Run development environment diagnostics
	@./scripts/doctor.sh

sqlc: ## Generate type-safe Go code from SQL
	sqlc generate

proto: ## Generate gRPC code from protobuf definitions
	buf generate

proto-version-create: ## Create a new API version for a module (usage: make proto-version-create MODULE_NAME=auth VERSION=v2)
	@if [ -z "$(MODULE_NAME)" ]; then echo "Usage: make proto-version-create MODULE_NAME=<module_name> VERSION=<version>"; exit 1; fi
	@if [ -z "$(VERSION)" ]; then echo "Usage: make proto-version-create MODULE_NAME=<module_name> VERSION=<version>"; exit 1; fi
	@./scripts/proto-version-create.sh $(MODULE_NAME) $(VERSION)
	@echo "✅ New version created. Run 'make proto' to generate code."

proto-breaking-check: ## Check for breaking changes in proto files (usage: make proto-breaking-check [MODULE_NAME=module_name])
	@if [ -z "$(MODULE_NAME)" ]; then \
		./scripts/proto-breaking-check.sh; \
	else \
		./scripts/proto-breaking-check.sh $(MODULE_NAME); \
	fi

proto-lint: ## Lint all proto files
	buf lint

generate-mocks: ## Generate all mocks from interfaces
	@echo "Generating mocks..."
	@go generate ./modules/...
	@echo "Mocks generated successfully"

generate-all: sqlc proto generate-mocks ## Generate all code (sqlc + proto + mocks)

docker-up: ## Run docker-compose
	docker-compose up -d

docker-up-minimal: ## Run docker-compose with minimal services (db + redis only)
	docker-compose -f docker-compose.minimal.yaml up -d

test: ## Run tests
	go test -v -race -cover ./...

test-unit: generate-mocks ## Run unit tests with fresh mocks
	go test -v -short ./...

test-integration: ## Run integration tests (requires Docker)
	@echo "Running integration tests with testcontainers..."
	go test -v -run Integration ./...

test-all: test-unit test-integration ## Run all tests (unit + integration)

test-coverage: ## Run tests with coverage report
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

coverage-report: ## Generate detailed coverage report
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
	@echo "💡 Para ver el reporte HTML completo: make test-coverage"

coverage-html: ## Open coverage report in browser
	@go test ./... -coverprofile=coverage.out -covermode=atomic > /dev/null 2>&1
	@go tool cover -html=coverage.out

lint: ## Run linter
	golangci-lint run

format: ## Format code with gofmt (and goimports if available)
	@echo "Formatting code with gofmt..."
	@gofmt -w .
	@if command -v goimports > /dev/null; then \
		echo "Formatting imports with goimports..."; \
		goimports -w .; \
	else \
		echo "💡 Tip: Install goimports for import formatting: go install golang.org/x/tools/cmd/goimports@latest"; \
	fi
	@echo "✅ Code formatted"

tidy: ## Tidy Go module dependencies
	@echo "Tidying Go module dependencies..."
	@go mod tidy
	@echo "✅ Dependencies tidied"

pre-commit: format lint test-unit ## Run pre-commit checks (format + lint + test-unit)

docker-down: ## Stop docker-compose services
	docker-compose down

# Load .env file
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Database migrations are now handled by the modulith itself
# The server discovers and runs migrations for all registered modules

migrate-up: ## Run all module migrations (uses modulith's migration system)
	@echo "🚀 Running migrations for all modules..."
	go run cmd/server/main.go -migrate

migrate: migrate-up ## Alias for migrate-up

seed: ## Run seed data for all modules
	@echo "🌱 Running seed data for all modules..."
	go run cmd/server/main.go seed

admin: ## Run admin task (usage: make admin TASK=task_name)
	@if [ -z "$(TASK)" ]; then echo "Usage: make admin TASK=task_name"; exit 1; fi
	@echo "🔧 Running admin task: $(TASK)"
	go run cmd/server/main.go admin $(TASK)

migrate-down: ## Rollback last migration for a specific module (usage: make migrate-down MODULE_NAME=auth)
	@if [ -z "$(MODULE_NAME)" ]; then echo "Usage: make migrate-down MODULE_NAME=module_name"; exit 1; fi
	@MIGRATIONS_DIR=modules/$(MODULE_NAME)/resources/db/migration; \
	if [ ! -d "$$MIGRATIONS_DIR" ]; then \
		echo "Error: Module '$(MODULE_NAME)' not found or has no migrations directory"; \
		exit 1; \
	fi; \
	echo "⚠️  Rolling back last migration for module: $(MODULE_NAME)"; \
	migrate -path $$MIGRATIONS_DIR -database "$(DB_DSN)" down 1

migrate-create: ## Create a new migration file for a module (usage: make migrate-create MODULE_NAME=auth NAME=add_users)
	@if [ -z "$(MODULE_NAME)" ]; then echo "Usage: make migrate-create MODULE_NAME=module_name NAME=migration_name"; exit 1; fi
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create MODULE_NAME=module_name NAME=migration_name"; exit 1; fi
	@MIGRATIONS_DIR=modules/$(MODULE_NAME)/resources/db/migration; \
	if [ ! -d "$$MIGRATIONS_DIR" ]; then \
		echo "Error: Module '$(MODULE_NAME)' not found or has no migrations directory"; \
		exit 1; \
	fi; \
	migrate create -ext sql -dir $$MIGRATIONS_DIR -seq $(NAME)

migrate-force: ## Force migration version to clean dirty state (usage: make migrate-force MODULE_NAME=auth VERSION=1)
	@if [ -z "$(MODULE_NAME)" ]; then echo "Usage: make migrate-force MODULE_NAME=module_name VERSION=version_number"; exit 1; fi
	@if [ -z "$(VERSION)" ]; then echo "Usage: make migrate-force MODULE_NAME=module_name VERSION=version_number"; exit 1; fi
	@MIGRATIONS_DIR=modules/$(MODULE_NAME)/resources/db/migration; \
	if [ ! -d "$$MIGRATIONS_DIR" ]; then \
		echo "Error: Module '$(MODULE_NAME)' not found or has no migrations directory"; \
		exit 1; \
	fi; \
	echo "⚠️  Forcing migration version $(VERSION) for module $(MODULE_NAME) (clears dirty state)..."; \
	if echo "$(DB_DSN)" | grep -q "?"; then \
		MODULE_DSN="$(DB_DSN)&x-migrations-table=$(MODULE_NAME)_schema_migrations"; \
	else \
		MODULE_DSN="$(DB_DSN)?x-migrations-table=$(MODULE_NAME)_schema_migrations"; \
	fi; \
	migrate -path $$MIGRATIONS_DIR -database "$$MODULE_DSN" force $(VERSION)

db-down: ## Rollback ALL migrations for all modules (drops all tables, uses modulith's migration system)
	@echo "⚠️  WARNING: This will rollback ALL migrations for ALL modules (drops all tables)!"
	@read -p "Are you sure? Type 'yes' to confirm: " confirm && [ "$$confirm" = "yes" ] || exit 1
	@echo "🔄 Rolling back all migrations for all modules..."
	@go run cmd/server/main.go migrate-down

db-reset: ## Drop all module schemas and re-run all migrations (destructive, asks for confirmation)
	@./scripts/db-reset.sh

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X github.com/cmelgarejo/go-modulith-template/internal/version.Version=$(VERSION) \
           -X github.com/cmelgarejo/go-modulith-template/internal/version.Commit=$(COMMIT) \
           -X github.com/cmelgarejo/go-modulith-template/internal/version.BuildTime=$(BUILD_TIME)

build: ## Build the monolith binary
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/server ./cmd/server/main.go

build-module: ## Build a specific module binary (usage: make build-module MODULE_NAME)
	@if [ -z "$(MODULE_NAME)" ]; then echo "Usage: make build-module MODULE_NAME"; exit 1; fi
	@if [ ! -d "cmd/$(MODULE_NAME)" ]; then echo "Error: Module '$(MODULE_NAME)' not found in cmd/"; exit 1; fi
	@mkdir -p bin
	@echo "Building module: $(MODULE_NAME)"
	go build -ldflags "$(LDFLAGS)" -o bin/$(MODULE_NAME) ./cmd/$(MODULE_NAME)/main.go

build-worker: ## Build the worker binary
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/worker ./cmd/worker/main.go

build-all: build build-worker ## Build all binaries (server + worker + all modules)
	@mkdir -p bin
	@for dir in cmd/*/; do \
		module=$$(basename $$dir); \
		if [ "$$module" != "server" ] && [ "$$module" != "worker" ]; then \
			echo "Building module: $$module"; \
			go build -ldflags "$(LDFLAGS)" -o bin/$$module ./cmd/$$module/main.go; \
		fi \
	done

clean: ## Clean build artifacts
	rm -rf bin/

run: ## Run the monolith server (without hot reload)
	go run -ldflags "$(LDFLAGS)" cmd/server/main.go || true

dev: ## Run the monolith with live reload (requires Air)
	@./scripts/preflight-check.sh || exit 1
	@if command -v air > /dev/null; then \
		air -c .air.toml; \
	else \
		echo "Air is not installed. Please install it with: go install github.com/air-verse/air@latest"; \
	fi

dev-worker: ## Run the worker with live reload (requires Air)
	@./scripts/preflight-check.sh || exit 1
	@if command -v air > /dev/null; then \
		air -c .air.worker.toml; \
	else \
		echo "Air is not installed. Please install it with: go install github.com/air-verse/air@latest"; \
	fi

dev-module: ## Run a specific module with live reload (usage: make dev-module MODULE_NAME)
	@if [ -z "$(MODULE_NAME)" ]; then echo "Usage: make dev-module MODULE_NAME"; exit 1; fi
	@if [ ! -f ".air.$(MODULE_NAME).toml" ]; then echo "Error: Air config '.air.$(MODULE_NAME).toml' not found"; exit 1; fi
	@./scripts/preflight-check.sh || exit 1
	@if command -v air > /dev/null; then \
		echo "Starting module: $(MODULE_NAME) with hot reload..."; \
		air -c .air.$(MODULE_NAME).toml; \
	else \
		echo "Air is not installed. Please install it with: go install github.com/air-verse/air@latest"; \
	fi

# Handle positional arguments for new-module
ifeq (new-module,$(firstword $(MAKECMDGOALS)))
  MODULE_NAME := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  $(eval $(MODULE_NAME):;@:)
endif

# Handle positional arguments for destroy-module
ifeq (destroy-module,$(firstword $(MAKECMDGOALS)))
  MODULE_NAME := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  $(eval $(MODULE_NAME):;@:)
endif

# Handle positional arguments for build-module
ifeq (build-module,$(firstword $(MAKECMDGOALS)))
  MODULE_NAME := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  $(eval $(MODULE_NAME):;@:)
endif

# Handle positional arguments for docker-build-module
ifeq (docker-build-module,$(firstword $(MAKECMDGOALS)))
  MODULE_NAME := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  $(eval $(MODULE_NAME):;@:)
endif

# Handle positional arguments for dev-module
ifeq (dev-module,$(firstword $(MAKECMDGOALS)))
  MODULE_NAME := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  $(eval $(MODULE_NAME):;@:)
endif

# Handle positional arguments for graphql-generate-module
ifeq (graphql-generate-module,$(firstword $(MAKECMDGOALS)))
  MODULE_NAME := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  $(eval $(MODULE_NAME):;@:)
endif


##### Docker
docker-build: ## Build docker image for server
	docker build \
		--build-arg TARGET=server \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t modulith-server:latest .

docker-build-module: ## Build docker image for a specific module (usage: make docker-build-module MODULE_NAME)
	@if [ -z "$(MODULE_NAME)" ]; then echo "Usage: make docker-build-module MODULE_NAME"; exit 1; fi
	@if [ ! -d "cmd/$(MODULE_NAME)" ]; then echo "Error: Module '$(MODULE_NAME)' not found in cmd/"; exit 1; fi
	@echo "Building Docker image for module: $(MODULE_NAME)"
	docker build \
		--build-arg TARGET=$(MODULE_NAME) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t modulith-$(MODULE_NAME):latest .

##### Modules
new-module: ## Scaffold a new module (usage: make new-module MODULE_NAME)
	@if [ -z "$(MODULE_NAME)" ]; then echo "Usage: make new-module NAME"; exit 1; fi
	./scripts/scaffold-module.sh $(MODULE_NAME)

destroy-module: ## Destroy a module completely (usage: make destroy-module MODULE_NAME)
	@if [ -z "$(MODULE_NAME)" ]; then echo "Usage: make destroy-module MODULE_NAME"; exit 1; fi
	./scripts/destroy-module.sh $(MODULE_NAME)

##### GraphQL (Optional)
graphql-add: ## Add optional GraphQL support using gqlgen (automatically generates code)
	./scripts/graphql-add-to-project.sh

graphql-init: ## Initialize GraphQL (alias for graphql-add)
	$(MAKE) graphql-add

graphql-generate: graphql-generate-all ## Generate GraphQL code for all modules (alias for graphql-generate-all)

graphql-generate-module: ## Generate GraphQL code for a specific module (usage: make graphql-generate-module auth)
	@if [ -z "$(MODULE_NAME)" ]; then echo "Usage: make graphql-generate-module <module_name>"; exit 1; fi
	./scripts/graphql-generate-module.sh $(MODULE_NAME)

graphql-generate-all: ## Generate GraphQL code for all modules (auto-discovers modules with schemas)
	./scripts/graphql-generate-all.sh

graphql-from-proto: ## Generate GraphQL schemas from OpenAPI/Swagger files for all modules
	./scripts/graphql-from-proto-all.sh

graphql-validate: ## Validate GraphQL schema

visualize: ## Visualize module connections (usage: make visualize [FORMAT=html|json|dot] [SERVE=true])
	@echo "🔍 Analyzing modulith architecture..."
	@FORMAT=$${FORMAT:-html}; \
	SERVE=$${SERVE:-false}; \
	if [ "$$SERVE" = "true" ]; then \
		go run ./cmd/visualize/main.go -format=$$FORMAT -serve; \
	else \
		go run ./cmd/visualize/main.go -format=$$FORMAT; \
	fi
