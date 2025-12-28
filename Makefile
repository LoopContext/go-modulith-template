.PHONY: sqlc proto install-deps

install-deps: ## Install developer tools
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install github.com/air-verse/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

sqlc:
	sqlc generate

proto:
	buf generate

docker-up: ## Run docker-compose
	docker-compose up -d

test: ## Run tests
	go test -v -race -cover ./...

test-coverage: ## Run tests with coverage report
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

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

run: ## Run the monolith
	go run cmd/server/main.go

dev: ## Run the monolith with live reload (requires Air)
	@if command -v air > /dev/null; then \
		air -c .air.toml; \
	else \
		echo "Air is not installed. Please install it with: go install github.com/air-verse/air@latest"; \
	fi

dev-auth: ## Run the auth-svc with live reload (requires Air)
	@if command -v air > /dev/null; then \
		air -c .air.auth.toml; \
	else \
		echo "Air is not installed. Please install it with: go install github.com/air-verse/air@latest"; \
	fi

# Handle positional arguments for new-module
ifeq (new-module,$(firstword $(MAKECMDGOALS)))
  MODULE_NAME := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  $(eval $(MODULE_NAME):;@:)
endif

##### Docker
docker-build: ## Build docker image (default: TARGET=server)
	docker build --build-arg TARGET=$(if $(TARGET),$(TARGET),server) -t modulith-$(if $(TARGET),$(TARGET),server):latest .

docker-build-auth: ## Build docker image for auth service
	$(MAKE) docker-build TARGET=auth-svc

##### Modules
new-module:
	@if [ -z "$(MODULE_NAME)" ]; then echo "Usage: make new-module NAME"; exit 1; fi
	./scripts/scaffold-module.sh $(MODULE_NAME)
