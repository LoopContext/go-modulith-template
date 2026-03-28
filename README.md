# Go Modulith Template 🚀
 
<p align="center">
  <img src="docs/assets/hero.png" width="800" alt="Go Modulith Architecture">
</p>


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
-   ⚡ **High-Performance Caching**: Native support for **Valkey** (the truly open-source Redis alternative) for caching and sessions.
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

### Quick Setup (Recommended)

The fastest way to get started is using the automated quickstart script:

```bash
just quickstart
```

This will:

1. Validate your environment setup
2. Install missing development tools
3. Start Docker infrastructure
4. Run database migrations
5. Optionally run seed data

> 💡 **Tip**: For a minimal setup (database + Valkey only), use `just docker-up-minimal`.

### Manual Setup

#### 1. Validate Setup (Optional but Recommended)

Check that all prerequisites are installed:

```bash
just validate-setup
```

#### 2. Install dependencies

```bash
just install-deps
```

#### 3. Start Complete Infrastructure

The template includes a complete observability stack for local development:

```bash
just docker-up
```

This starts:

-   **PostgreSQL**: Main database
-   **Valkey**: Cache and session storage
-   **Jaeger**: Distributed tracing (UI at http://localhost:16686)
-   **Prometheus**: Metrics and alerts (UI at http://localhost:9090)
-   **Grafana**: Visualization dashboards (UI at http://localhost:3000, user: `admin`, password: `admin`)

> 💡 **Tip**: To start only the database and Valkey, use `just docker-up-minimal`.

#### 4. Configure (Optional)

The project supports multiple configuration sources with clear precedence:

-   **`PORT`** (standard 12-factor variable): Highest priority, compatible with Heroku, Cloud Run, Railway, etc.
-   **YAML** (`configs/server.yaml`): High priority, ideal for environment-specific configurations
-   **`.env`**: Overrides system environment variables
-   **System environment variables**: Base values
-   **Defaults**: Hardcoded values in `config.go`

**Priority:** `PORT > YAML > .env > system ENV vars > defaults`

Edit `configs/server.yaml` with your configuration values (DB connection, JWT secret, OAuth, etc.).

On startup, you'll see a log showing the source of each configuration variable.

> 💡 **OAuth Tip**: To enable OAuth providers (Google, GitHub, etc.), configure credentials in `configs/server.yaml` or in your `.env` file. See [complete OAuth guide](docs/OAUTH_INTEGRATION.md).

#### 5. Run in Development (Hot Reload)

```bash
just dev
```

To run a specific module with hot reload:

```bash
just dev-module auth
```

To run the worker process (background tasks):

```bash
just dev-worker
# or
just build-worker && ./bin/worker
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
-   ✅ **State in external services:** Sessions in PostgreSQL, optional cache in Valkey
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
just admin TASK=cleanup-sessions

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

-   **Location**: `gen/openapiv2/proto/` (generated with `just proto`)
-   **Format**: JSON compatible with Swagger UI
-   **Usage**: Import `.swagger.json` files into [Swagger Editor](https://editor.swagger.io/) or any compatible tool

Example for the auth module:

```bash
# Generate documentation
just proto

# View the API
open gen/openapiv2/proto/auth/v1/auth.swagger.json
```

## 🛠️ Useful Commands (Makefile)

### Code Generation

-   `just proto`: Generates gRPC code from `.proto` files (includes OpenAPI/Swagger in `gen/openapiv2/`).
-   `just sqlc`: Generates Type-safe code for SQL queries.

### Build

-   `just build`: Compiles the monolith binary in `bin/server`.
-   `just build-module MODULE_NAME`: Compiles the binary for a specific module (e.g.: `just build-module auth`).
-   `just build-all`: Compiles all binaries (server + all modules).
-   `just clean`: Removes all build artifacts (`bin/` directory).

### Setup & Validation

-   `just quickstart`: Automated setup process (installs deps, starts docker, runs migrations).
-   `just validate-setup`: Validates development environment setup (prerequisites, tools, ports).
-   `just doctor`: Comprehensive development environment diagnostics (containers, connectivity, configuration).

### Docker

-   `just docker-up`: Starts all infrastructure services (PostgreSQL, Valkey, Jaeger, Prometheus, Grafana).
-   `just docker-up-minimal`: Starts minimal services (PostgreSQL + Valkey only) for faster startup.
-   `just docker-down`: Stops Docker containers.
-   `just docker-build`: Builds the server Docker image (`modulith-server:latest`).
-   `just docker-build-module MODULE_NAME`: Builds the Docker image for a specific module (e.g.: `just docker-build-module auth`).

### Code Quality

-   `just lint`: Runs the strict linter (**MANDATORY** after changes to `.go` files).
-   `just test`: Runs all unit tests.
-   `just test-unit`: Runs unit tests with mocks (fast, no DB).
-   `just test-coverage`: Runs tests and generates HTML coverage report.
-   `just coverage-report`: Shows detailed coverage report in terminal.
-   `just coverage-html`: Opens coverage report in browser.
-   `just generate-mocks`: Generates interface mocks for testing.
-   `just install-mocks`: Installs gomock for mock generation.

### Development

-   `just dev`: Runs the monolith server with hot reload.
-   `just dev-module MODULE_NAME`: Runs a specific module with hot reload (e.g.: `just dev-module auth`).
-   `just new-module MODULE_NAME`: Creates boilerplate for a new functional module with automatic configuration (generates structure + `.air.{MODULE_NAME}.toml`).

### Database

-   `just migrate-up` / `just migrate`: Runs migrations for all modules (the modulith discovers them automatically).
-   `just migrate-down MODULE=auth`: Reverts the last migration for a specific module.
-   `just migrate-create MODULE=auth NAME=add_users`: Creates a new migration for a specific module.
-   `just db-down`: ⚠️ Deletes all database tables (destructive).
-   `just db-reset`: ⚠️ Deletes everything and runs all migrations (equivalent to `db-down` + `migrate-up`).

**Note:** Migrations run automatically when you start the server. The modulith discovers and applies migrations for all registered modules.

### Administrative Tasks

-   `just admin TASK=cleanup-sessions`: Runs administrative task to clean expired sessions.
-   `just admin TASK=cleanup-magic-codes`: Runs administrative task to clean expired magic codes.
-   `./bin/server admin <task_name>`: Runs an administrative task directly.

**Note:** Administrative tasks run as independent commands. You can list available tasks by running `./bin/server admin` without arguments.

### GraphQL (Optional)

-   `just graphql-init`: Adds optional GraphQL support using gqlgen and automatically generates code (one command does everything).
-   `just graphql-generate-all`: Generates GraphQL code from schemas for all modules.
-   `just graphql-generate-module MODULE_NAME=<name>`: Generates GraphQL code for a specific module (auto-generates schema from proto if missing).
-   `just graphql-from-proto`: Generates GraphQL schemas from OpenAPI/Swagger definitions for all modules.
-   `just graphql-validate`: Validates GraphQL schema.

### ⚠️ Quality Workflow

**After modifying `.go` files:**

1. Run `just lint` and fix **all** errors (0 issues).
2. Run `just test` to verify you didn't break anything.
3. **NEVER** modify `.golangci.yaml` to ignore errors - implement proper fixes.

### Troubleshooting

If you encounter issues with your development environment:

1. **Run diagnostics**: `just doctor` - Comprehensive health check of your environment
2. **Validate setup**: `just validate-setup` - Check prerequisites and configuration
3. **Check containers**: `docker-compose ps` - Verify Docker containers are running
4. **View logs**: `docker-compose logs [service]` - Check service logs
5. **Reset database**: `just db-reset` - Drop and recreate database (destructive)

Common issues:

-   **Port conflicts**: Use `just doctor` to identify which ports are in use
-   **Docker not running**: Start Docker Desktop or docker service
-   **Database connection errors**: Ensure containers are running with `just docker-up`
-   **Missing tools**: Run `just install-deps` to install all development tools

---

Made with ❤️ for developers seeking operational excellence.
