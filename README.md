# Go Modulith Template 🚀

![Tests](https://img.shields.io/badge/tests-passing-brightgreen)
![Coverage](https://img.shields.io/badge/coverage-19.9%25-yellow)
![Go](https://img.shields.io/badge/go-1.24+-blue)
![License](https://img.shields.io/badge/license-MIT-blue)

This is a professional template for building Go applications following the **Modulith** pattern. It's designed to be scalable, maintainable, and easy to maintain, allowing evolution from a monolith to microservices without friction.

## ✨ Key Features

-   🏗️ **Modular Architecture**: Code organized by domains with decoupling through internal events.
-   📦 **Registry Pattern**: Manual, explicit dependency injection without magic for maximum control.
-   🔐 **gRPC & Protobuf**: Typed and efficient communication with automatic code generation via `buf`.
-   🗄️ **SQLC & Migrations**: Type-safe data access and schema management with `golang-migrate`.
-   ⚙️ **Flexible Configuration**: Configuration system with precedence hierarchy (YAML > .env > system ENV vars > defaults) and source logging.
-   🔄 **Hot Reload**: Smooth development with **Air** monitoring changes in code, configuration (`.env`, YAML) and resources.
-   🔌 **WebSocket Real-Time**: Bidirectional communication integrated with the event bus for real-time notifications.
-   👷 **Worker Process**: Background process for asynchronous tasks, event consumers, and scheduled jobs.
-   🔐 **Secrets Management**: Abstraction for secret management (env vars, Vault, AWS Secrets Manager).
-   📊 **Complete Observability**: Local stack with Jaeger, Prometheus, and Grafana for development and debugging.
-   📊 **Optional GraphQL**: Optional support with gqlgen for flexible and frontend-friendly APIs (subscriptions included).
-   📧 **Notification System**: Templates + extensible providers (SendGrid, Twilio, AWS SES/SNS).
-   🔑 **Complete Auth**: Passwordless login, sessions, refresh tokens, revocation, and profile management.
-   🔗 **OAuth/Social Login**: Authentication with Google, Facebook, GitHub, Apple, Microsoft, and Twitter/X.
-   🧪 **Mocking with gomock**: Automatic generation of type-safe mocks for efficient unit testing.
-   🧪 **Test Utilities**: Comprehensive testing utilities (`internal/testutil`) for integration tests, gRPC servers, event bus, and test registries.
-   🛡️ **Observability**: Native integration with OpenTelemetry (Tracing & Metrics), Prometheus, and Health Checks with context handling.
-   ⚡ **Error Handling**: Domain error system with automatic mapping to gRPC codes.
-   📡 **Telemetry Helpers**: Integrated helpers for consistent tracing across all modules.
-   🎯 **Typed Events**: Typed constants for events with autocomplete and typo prevention.
-   🔄 **Multi-Module Migrations**: Automatic discovery and execution of migrations per module.
-   🔐 **RBAC Built-in**: Authorization helpers for permissions, roles, and ownership.
-   ⛴️ **Cloud Ready**: Multi-stage Dockerfile and flexible Helm Charts for Kubernetes (supports monolith and independent modules).
-   🌍 **IaC with OpenTofu**: Reproducible base infrastructure (VPC, EKS, RDS) managed with OpenTofu and Terragrunt.
-   🤖 **CI/CD**: GitHub Actions pipelines for automatic validation.

## 🛠️ Prerequisites

-   Go 1.24+
-   Docker & Docker Compose
-   Development tools:
    -   `sqlc`
    -   `buf`
    -   `migrate`
    -   `air`
    -   `golangci-lint`

## 🚀 Quick Start

### 1. Install dependencies

```bash
make install-deps
```

### 2. Start Complete Infrastructure

The template includes a complete observability stack for local development:

```bash
make docker-up
```

This starts:

-   **PostgreSQL**: Main database
-   **Redis**: Cache and session storage
-   **Jaeger**: Distributed tracing (UI at http://localhost:16686)
-   **Prometheus**: Metrics and alerts (UI at http://localhost:9090)
-   **Grafana**: Visualization dashboards (UI at http://localhost:3000, user: `admin`, password: `admin`)

> 💡 **Tip**: To start only the database, use `docker-compose up db`.

### 3. Configure (Optional)

The project supports multiple configuration sources with clear precedence:

-   **`PORT`** (standard 12-factor variable): Highest priority, compatible with Heroku, Cloud Run, Railway, etc.
-   **YAML** (`configs/server.yaml`): High priority, ideal for environment-specific configurations
-   **`.env`**: Overrides system environment variables
-   **System environment variables**: Base values
-   **Defaults**: Hardcoded values in `config.go`

**Priority:** `PORT > YAML > .env > system ENV vars > defaults`

```bash
# Copy the example file for environment variables
cp .env.example .env

# Edit .env with your values (DB, JWT secret, OAuth, etc.)
# Or configure directly in configs/server.yaml
```

On startup, you'll see a log showing the source of each configuration variable.

> 💡 **OAuth Tip**: To enable OAuth providers (Google, GitHub, etc.), configure credentials in `configs/server.yaml` or in your `.env` file. See [complete OAuth guide](docs/OAUTH_INTEGRATION.md).

### 4. Run in Development (Hot Reload)

```bash
make dev
```

To run a specific module with hot reload:

```bash
make dev-module auth
```

To run the worker process (background tasks):

```bash
make dev-worker
# or
make build-worker && ./bin/worker
```

> 💡 **Tip**: Air automatically monitors changes in `.go`, `.yaml`, `.env`, `.proto`, `.sql` and configuration files, restarting the server instantly.

### 5. Secrets Management

The template includes an abstraction for secrets management that allows using different providers:

-   **Development**: Environment variables (`EnvProvider` implementation)
-   **Production**: HashiCorp Vault, AWS Secrets Manager, etc. (extensible)

See [environment variables documentation](docs/ENVIRONMENT.md) for more details.

### 6. Stateless Processes (12-Factor App)

The template follows the **stateless processes** principle:

-   ✅ **No local state:** No temporary files are written or state stored on disk
-   ✅ **State in external services:** Sessions in PostgreSQL, optional cache in Redis
-   ✅ **Horizontal scaling:** Any instance can handle any request
-   ⚠️ **WebSocket:** Requires sticky sessions for scaling (see documentation)

**See complete documentation:** `docs/MODULITH_ARCHITECTURE.md` (section 20: Stateless Processes)

### 7. Health Checks and Monitoring

The server exposes health check endpoints for integration with orchestrators (Kubernetes, Docker Swarm, etc.):

-   **`/livez`**: Liveness probe - always returns 200 if the process is alive
-   **`/readyz`**: Readiness probe - checks dependencies (DB, modules, event bus, WebSocket)
-   **`/healthz`**: Legacy endpoint (backward compatibility, same as `/livez`)
-   **`/healthz/ws`**: WebSocket connection status (active connections and connected users)

The `/readyz` endpoint returns detailed JSON with the status of each dependency:

```json
{
    "status": "ready",
    "checks": {
        "modules": "healthy",
        "database": "healthy",
        "event_bus": "healthy",
        "websocket": "healthy"
    }
}
```

If any dependency is unhealthy, the endpoint returns `503 Service Unavailable`.

### 8. Administrative Tasks

The template includes an administrative task system for maintenance operations:

**Available tasks:**

-   `cleanup-sessions`: Cleans expired user sessions
-   `cleanup-magic-codes`: Cleans expired magic codes

**Usage:**

```bash
# Run an administrative task
make admin TASK=cleanup-sessions

# Or directly with the binary
./bin/server admin cleanup-sessions
./bin/server admin cleanup-magic-codes

# List available tasks
./bin/server admin
```

Administrative tasks run as independent commands and are useful for:

-   Periodic cleanup of expired data
-   Database maintenance
-   Data migration operations
-   Audit tasks

## 📖 Complete Documentation

-   **[Architecture Guide](docs/MODULITH_ARCHITECTURE.md)** - ⭐ Complete architecture, patterns, error handling, telemetry, typed events, RBAC, testing and more
-   **[Module Communication](docs/MODULE_COMMUNICATION.md)** - ⭐ How communication works in Modulith vs Microservices, gRPC in-process vs network, event bus
-   **[12-Factor App Compliance](docs/12_FACTOR_APP.md)** - Complete guide to 12-factor app methodology compliance
-   **[OAuth/Social Login](docs/OAUTH_INTEGRATION.md)** - Integration with Google, Facebook, GitHub, Apple, Microsoft, Twitter
-   **[Notification System](docs/NOTIFICATION_SYSTEM.md)** - Templates, providers (SendGrid, Twilio, SES) and composite notifier
-   **[Real-Time WebSocket](docs/WEBSOCKET_GUIDE.md)** - Bidirectional communication, event bus and JWT authentication
-   **[GraphQL Integration](docs/GRAPHQL_INTEGRATION.md)** - Optional setup with gqlgen, schema per module and subscriptions
-   **[Deployment & IaC](docs/DEPLOYMENT_SYNC.md)** - OpenTofu, Helm Charts, deployment strategies and testing
-   **[Frontend Proposal](docs/FRONTEND_PROPOSAL.md)** - Go Templates + HTMX with WebSocket/GraphQL
-   **[Deployment Guide](deployment/README.md)** - Complete Kubernetes deployment guide
-   **[Helm Chart Documentation](deployment/helm/modulith/README.md)** - Detailed Helm chart documentation

## 📋 API Documentation

The project automatically generates OpenAPI/Swagger documentation:

-   **Location**: `gen/openapiv2/proto/` (generated with `make proto`)
-   **Format**: JSON compatible with Swagger UI
-   **Usage**: Import `.swagger.json` files into [Swagger Editor](https://editor.swagger.io/) or any compatible tool

Example for the auth module:

```bash
# Generate documentation
make proto

# View the API
open gen/openapiv2/proto/auth/v1/auth.swagger.json
```

## 🛠️ Useful Commands (Makefile)

### Code Generation

-   `make proto`: Generates gRPC code from `.proto` files (includes OpenAPI/Swagger in `gen/openapiv2/`).
-   `make sqlc`: Generates Type-safe code for SQL queries.

### Build

-   `make build`: Compiles the monolith binary in `bin/server`.
-   `make build-module MODULE_NAME`: Compiles the binary for a specific module (e.g.: `make build-module auth`).
-   `make build-all`: Compiles all binaries (server + all modules).
-   `make clean`: Removes all build artifacts (`bin/` directory).

### Docker

-   `make docker-build`: Builds the server Docker image (`modulith-server:latest`).
-   `make docker-build-module MODULE_NAME`: Builds the Docker image for a specific module (e.g.: `make docker-build-module auth`).

### Code Quality

-   `make lint`: Runs the strict linter (**MANDATORY** after changes to `.go` files).
-   `make test`: Runs all unit tests.
-   `make test-unit`: Runs unit tests with mocks (fast, no DB).
-   `make test-coverage`: Runs tests and generates HTML coverage report.
-   `make coverage-report`: Shows detailed coverage report in terminal.
-   `make coverage-html`: Opens coverage report in browser.
-   `make generate-mocks`: Generates interface mocks for testing.
-   `make install-mocks`: Installs gomock for mock generation.

### Development

-   `make dev`: Runs the monolith server with hot reload.
-   `make dev-module MODULE_NAME`: Runs a specific module with hot reload (e.g.: `make dev-module auth`).
-   `make new-module MODULE_NAME`: Creates boilerplate for a new functional module with automatic configuration (generates structure + `.air.{MODULE_NAME}.toml`).

### Database

-   `make docker-up`: Starts infrastructure (PostgreSQL) with Docker Compose.
-   `make docker-down`: Stops Docker containers.
-   `make migrate-up` / `make migrate`: Runs migrations for all modules (the modulith discovers them automatically).
-   `make migrate-down MODULE=auth`: Reverts the last migration for a specific module.
-   `make migrate-create MODULE=auth NAME=add_users`: Creates a new migration for a specific module.
-   `make db-down`: ⚠️ Deletes all database tables (destructive).
-   `make db-reset`: ⚠️ Deletes everything and runs all migrations (equivalent to `db-down` + `migrate-up`).

**Note:** Migrations run automatically when you start the server. The modulith discovers and applies migrations for all registered modules.

### Administrative Tasks

-   `make admin TASK=cleanup-sessions`: Runs administrative task to clean expired sessions.
-   `make admin TASK=cleanup-magic-codes`: Runs administrative task to clean expired magic codes.
-   `./bin/server admin <task_name>`: Runs an administrative task directly.

**Note:** Administrative tasks run as independent commands. You can list available tasks by running `./bin/server admin` without arguments.

### GraphQL (Optional)

-   `make graphql-init`: Adds optional GraphQL support using gqlgen and automatically generates code (one command does everything).
-   `make graphql-generate-all`: Generates GraphQL code from schemas for all modules.
-   `make graphql-generate-module MODULE_NAME=<name>`: Generates GraphQL code for a specific module (auto-generates schema from proto if missing).
-   `make graphql-from-proto`: Generates GraphQL schemas from OpenAPI/Swagger definitions for all modules.
-   `make graphql-validate`: Validates GraphQL schema.

### ⚠️ Quality Workflow

**After modifying `.go` files:**

1. Run `make lint` and fix **all** errors (0 issues).
2. Run `make test` to verify you didn't break anything.
3. **NEVER** modify `.golangci.yaml` to ignore errors - implement proper fixes.

---

Made with ❤️ for developers seeking operational excellence.
