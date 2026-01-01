.PHONY: sqlc proto install-deps install-mocks generate-mocks test-unit graphql-init graphql-generate graphql-validate add-graphql

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

sqlc:
	sqlc generate

proto:
	buf generate

generate-mocks: ## Generate all mocks from interfaces
	@echo "Generating mocks..."
	@go generate ./modules/...
	@echo "Mocks generated successfully"

docker-up: ## Run docker-compose
	docker-compose up -d

test: ## Run tests
	go test -v -race -cover ./...

test-unit: generate-mocks ## Run unit tests with fresh mocks
	go test -v -short ./...

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

docker-down:
	docker-compose down

# Load .env file
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

MIGRATIONS_DIR=modules/auth/resources/db/migration

migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" up

migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" down

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $$name

build: ## Build the monolith binary
	@mkdir -p bin
	go build -o bin/server ./cmd/server/main.go

build-module: ## Build a specific module binary (usage: make build-module MODULE_NAME)
	@if [ -z "$(MODULE_NAME)" ]; then echo "Usage: make build-module MODULE_NAME"; exit 1; fi
	@if [ ! -d "cmd/$(MODULE_NAME)" ]; then echo "Error: Module '$(MODULE_NAME)' not found in cmd/"; exit 1; fi
	@mkdir -p bin
	@echo "Building module: $(MODULE_NAME)"
	go build -o bin/$(MODULE_NAME) ./cmd/$(MODULE_NAME)/main.go

build-all: build ## Build all binaries (server + all modules)
	@mkdir -p bin
	@for dir in cmd/*/; do \
		module=$$(basename $$dir); \
		if [ "$$module" != "server" ]; then \
			echo "Building module: $$module"; \
			go build -o bin/$$module ./cmd/$$module/main.go; \
		fi \
	done

clean: ## Clean build artifacts
	rm -rf bin/

run: ## Run the monolith
	go run cmd/server/main.go

dev: ## Run the monolith with live reload (requires Air)
	@if command -v air > /dev/null; then \
		air -c .air.toml; \
	else \
		echo "Air is not installed. Please install it with: go install github.com/air-verse/air@latest"; \
	fi

dev-module: ## Run a specific module with live reload (usage: make dev-module MODULE_NAME)
	@if [ -z "$(MODULE_NAME)" ]; then echo "Usage: make dev-module MODULE_NAME"; exit 1; fi
	@if [ ! -f ".air.$(MODULE_NAME).toml" ]; then echo "Error: Air config '.air.$(MODULE_NAME).toml' not found"; exit 1; fi
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

##### Docker
docker-build: ## Build docker image for server
	docker build --build-arg TARGET=server -t modulith-server:latest .

docker-build-module: ## Build docker image for a specific module (usage: make docker-build-module MODULE_NAME)
	@if [ -z "$(MODULE_NAME)" ]; then echo "Usage: make docker-build-module MODULE_NAME"; exit 1; fi
	@if [ ! -d "cmd/$(MODULE_NAME)" ]; then echo "Error: Module '$(MODULE_NAME)' not found in cmd/"; exit 1; fi
	@echo "Building Docker image for module: $(MODULE_NAME)"
	docker build --build-arg TARGET=$(MODULE_NAME) -t modulith-$(MODULE_NAME):latest .

##### Modules
new-module:
	@if [ -z "$(MODULE_NAME)" ]; then echo "Usage: make new-module NAME"; exit 1; fi
	./scripts/scaffold-module.sh $(MODULE_NAME)

##### GraphQL (Optional)
add-graphql: ## Add optional GraphQL support using gqlgen
	./scripts/add-graphql.sh

graphql-init: ## Initialize GraphQL (alias for add-graphql)
	$(MAKE) add-graphql

graphql-generate: ## Generate GraphQL code from schema
	@if ! command -v gqlgen > /dev/null; then \
		echo "gqlgen not found. Install with: go install github.com/99designs/gqlgen@latest"; \
		exit 1; \
	fi
	@if [ ! -f "gqlgen.yml" ]; then \
		echo "GraphQL not initialized. Run: make add-graphql"; \
		exit 1; \
	fi
	gqlgen generate

graphql-validate: ## Validate GraphQL schema
	@if ! command -v gqlgen > /dev/null; then \
		echo "gqlgen not found. Install with: go install github.com/99designs/gqlgen@latest"; \
		exit 1; \
	fi
	@if [ ! -f "gqlgen.yml" ]; then \
		echo "GraphQL not initialized. Run: make add-graphql"; \
		exit 1; \
	fi
	gqlgen generate --verbose
