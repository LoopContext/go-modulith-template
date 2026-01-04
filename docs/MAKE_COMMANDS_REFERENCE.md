# Make Commands Reference

Complete reference of all available `make` commands for the Go Modulith Template.

## 📋 Quick Navigation

- [Setup & Installation](#setup--installation)
- [Code Generation](#code-generation)
- [API Versioning](#api-versioning)
- [Development](#development)
- [Testing](#testing)
- [Database & Migrations](#database--migrations)
- [Build & Docker](#build--docker)
- [Modules](#modules)
- [GraphQL (Optional)](#graphql-optional)
- [Maintenance](#maintenance)

---

## Setup & Installation

### `make help`
Show all available commands with descriptions.

### `make install-deps`
Install all developer tools:
- `migrate` (golang-migrate)
- `sqlc`
- `buf`
- `air`
- `golangci-lint`
- `gqlgen`
- `mockgen`

### `make install-mocks`
Install gomock for test mocking (alias for installing mockgen).

### `make validate-setup`
Validate development environment setup (checks Go version, Docker, tools, ports).

### `make quickstart`
**Recommended for first-time setup.** Runs complete setup process:
1. Validates environment
2. Installs missing development tools
3. Starts Docker infrastructure
4. Runs database migrations
5. Optionally runs seed data

### `make doctor`
Run development environment diagnostics (troubleshooting tool).

---

## Code Generation

### `make sqlc`
Generate type-safe Go code from SQL queries (uses `sqlc generate`).

### `make proto`
Generate gRPC code from Protobuf definitions (uses `buf generate`).

### `make generate-mocks`
Generate all mocks from interfaces in `./modules/...` (runs `go generate`).

### `make generate-all`
Generate all code at once (sqlc + proto + mocks).
- Runs `sqlc`, `proto`, and `generate-mocks` in sequence
- Useful before committing or when starting fresh

---

## API Versioning

### `make proto-version-create MODULE_NAME=<name> VERSION=<version>`
Create a new API version for a module (e.g., v2, v3).
- Automatically copies the latest version as a starting point
- Updates package names, REST paths, and Go package options
- Example: `make proto-version-create MODULE_NAME=auth VERSION=v2`
- After creation, run `make proto` to generate code

### `make proto-breaking-check [MODULE_NAME=<name>]`
Check for breaking changes and linting issues in proto files.
- Without MODULE_NAME: checks all modules
- With MODULE_NAME: checks specific module only
- Example: `make proto-breaking-check MODULE_NAME=auth`
- Uses `buf lint` to detect issues

### `make proto-lint`
Lint all proto files using buf.
- Validates proto syntax and best practices
- Run before committing proto changes

---

## Development

### `make run`
Run the monolith server **without** hot reload (uses `go run`).

### `make dev`
Run the monolith server **with hot reload** (requires Air).
- Automatically monitors `.go`, `.yaml`, `.env`, `.proto`, `.sql` files
- Runs pre-flight checks before starting

### `make dev-worker`
Run the worker process with hot reload (requires Air).
- For background tasks, event consumers, scheduled jobs

### `make dev-module MODULE_NAME=<name>`
Run a specific module with hot reload.
Example:
```bash
make dev-module auth
```

---

## Testing

### `make test`
Run all tests with verbose output, race detection, and coverage.

### `make test-unit`
Run unit tests only (short tests, generates mocks first).

### `make test-integration`
Run integration tests only (requires Docker, uses testcontainers).

### `make test-all`
Run all tests (unit + integration).

### `make test-coverage`
Run tests and generate HTML coverage report (opens in browser).

### `make coverage-report`
Generate detailed coverage report (terminal output).

### `make coverage-html`
Generate and open coverage report in browser.

---

## Database & Migrations

### `make migrate-up` or `make migrate`
Run all module migrations (uses modulith's migration system).
- Automatically discovers migrations from all registered modules
- Executes in registration order

### `make migrate-down MODULE_NAME=<module_name>`
Rollback last migration for a specific module.
Example:
```bash
make migrate-down MODULE_NAME=auth
```

### `make migrate-create MODULE_NAME=<module_name> NAME=<migration_name>`
Create a new migration file for a module.
Example:
```bash
make migrate-create MODULE_NAME=auth NAME=add_users_table
```

### `make db-down`
Drop all database tables (destructive, asks for confirmation).

### `make db-reset`
Drop database and re-run all migrations (`db-down + migrate-up`).

### `make seed`
Run seed data for all modules.

### `make admin TASK=<task_name>`
Run admin task.
Example:
```bash
make admin TASK=cleanup_old_sessions
```

---

## Build & Docker

### `make build`
Build the monolith binary (output: `bin/server`).

### `make build-worker`
Build the worker binary (output: `bin/worker`).

### `make build-module MODULE_NAME=<name>`
Build a specific module binary.
Example:
```bash
make build-module auth
```

### `make build-all`
Build all binaries (server + worker + all modules).

### `make clean`
Clean build artifacts (removes `bin/` directory).

### `make docker-up`
Start all Docker Compose services:
- PostgreSQL (database)
- Redis (cache/sessions)
- Jaeger (tracing UI: http://localhost:16686)
- Prometheus (metrics: http://localhost:9090)
- Grafana (dashboards: http://localhost:3000)

### `make docker-up-minimal`
Start minimal Docker services (PostgreSQL + Redis only).

### `make docker-down`
Stop all Docker Compose services.

### `make docker-build`
Build Docker image for server (`modulith-server:latest`).

### `make docker-build-module MODULE_NAME=<name>`
Build Docker image for a specific module.
Example:
```bash
make docker-build-module auth
```

---

## Modules

### `make new-module MODULE_NAME=<name>`
Scaffold a new module with all boilerplate.
Example:
```bash
make new-module orders
```

---

## GraphQL (Optional)

### `make graphql-add` or `make graphql-init`
Add optional GraphQL support using gqlgen (automatically generates code).

### `make graphql-generate` or `make graphql-generate-all`
Generate GraphQL code for all modules (auto-discovers modules with schemas).

### `make graphql-generate-module MODULE_NAME=<name>`
Generate GraphQL code for a specific module.
Example:
```bash
make graphql-generate-module auth
```

### `make graphql-from-proto`
Generate GraphQL schemas from OpenAPI/Swagger files for all modules.

### `make graphql-validate`
Validate GraphQL schema.

---

## Maintenance

### `make lint`
Run golangci-lint to check code quality.

### `make format`
Format code with `gofmt` (and `goimports` if available).
- Uses built-in `gofmt` for code formatting
- Uses `goimports` if installed (also formats imports)
- Provides helpful tip if `goimports` is not installed

### `make tidy`
Tidy Go module dependencies (runs `go mod tidy`).
- Removes unused dependencies
- Adds missing dependencies
- Updates `go.sum` file

### `make pre-commit`
Run pre-commit checks (format + lint + test-unit).
- Formats code
- Runs linter
- Runs unit tests
- Perfect for running before committing changes

---

## 🎯 Common Workflows

### First-Time Setup
```bash
make quickstart
```

### Daily Development
```bash
# Start infrastructure
make docker-up-minimal

# Run migrations (if needed)
make migrate-up

# Start dev server with hot reload
make dev
```

### Before Committing
```bash
# Generate all code
make generate-all

# Run pre-commit checks (format + lint + test-unit)
make pre-commit

# Or run individual checks
make format
make lint
make test-unit
```

### Adding a New Module
```bash
# Scaffold module
make new-module orders

# Create migration
make migrate-create MODULE_NAME=orders NAME=create_orders_table

# Generate code (after adding SQL/proto)
make sqlc proto

# Run migrations
make migrate-up
```

### Building for Production
```bash
# Build all binaries
make build-all

# Or build Docker images
make docker-build
make docker-build-module MODULE_NAME=auth
```

---

## 📝 Notes

- Most commands support the `help` target: `make help`
- Docker commands require Docker and Docker Compose
- Development commands (`dev`, `dev-worker`, `dev-module`) require Air to be installed
- Migration commands require database connection (configured via `.env` or `configs/server.yaml`)
- GraphQL commands are optional (only needed if GraphQL support is enabled)

---

## 🔍 Potential Future Improvements

The following commands could be added for even better developer experience:

1. **`make update-deps`**: Update Go dependencies (with confirmation)
2. **`make vendor`**: Vendor dependencies (`go mod vendor`)

