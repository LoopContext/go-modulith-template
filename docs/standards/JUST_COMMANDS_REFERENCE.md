# Just Commands Reference

Complete reference of all available `just` commands for the Go Modulith Template.

## 📋 Quick Navigation

- [Setup & Installation](#setup--installation)
- [Code Generation](#code-generation)
- [API Versioning](#api-versioning)
- [Development](#development)
- [Testing](#testing)
- [Database & Migrations](#database--migrations)
- [Build & Docker](#build--docker)
- [Modules](#modules)
- [Frontend](#frontend)
- [GraphQL (Optional)](#graphql-optional)
- [Maintenance](#maintenance)

---

## Setup & Installation

### `just help`
Show all available commands with descriptions.

### `just install-deps`
Install all developer tools:
- `migrate` (golang-migrate)
- `sqlc`
- `buf`
- `air`
- `golangci-lint`
- `gqlgen`
- `mockgen`

### `just install-mocks`
Install gomock for test mocking (alias for installing mockgen).

### `just validate-setup`
Validate development environment setup (checks Go version, Docker, tools, ports).

### `just quickstart`
**Recommended for first-time setup.** Runs complete setup process:
1. Validates environment
2. Installs missing development tools
3. Starts Docker infrastructure
4. Runs database migrations
5. Optionally runs seed data

### `just doctor`
Run development environment diagnostics (troubleshooting tool).

---

## Code Generation

### `just sqlc`
Generate type-safe Go code from SQL queries (uses `sqlc generate`).

### `just proto`
Generate gRPC code from Protobuf definitions (uses `buf generate`).

### `just generate-mocks`
Generate all mocks from interfaces in `./modules/...` (runs `go generate`).

### `just generate-all`
Generate all code at once (sqlc + proto + mocks).
- Runs `sqlc`, `proto`, and `generate-mocks` in sequence
- Useful before committing or when starting fresh

---

## API Versioning

### `just proto-version-create <module> <version>`
Create a new API version for a module (e.g., v2, v3).
- Automatically copies the latest version as a starting point
- Updates package names, REST paths, and Go package options
- Example: `just proto-version-create auth v2`
- After creation, run `just proto` to generate code

### `just proto-breaking-check [module]`
Check for breaking changes and linting issues in proto files.
- Without argument: checks all modules
- With argument: checks specific module only
- Example: `just proto-breaking-check auth`
- Uses `buf lint` to detect issues

### `just proto-lint`
Lint all proto files using buf.
- Validates proto syntax and best practices
- Run before committing proto changes

---

## Development

### `just run`
Run the monolith server **without** hot reload (uses `go run`).

### `just dev`
Run the monolith server **with hot reload** (requires Air).
- Automatically monitors `.go`, `.yaml`, `.env`, `.proto`, `.sql` files
- Runs pre-flight checks before starting

### `just dev-worker`
Run the worker process with hot reload (requires Air).
- For background tasks, event consumers, scheduled jobs

### `just dev-module <module>`
Run a specific module with hot reload.
Example:
```bash
just dev-module auth
```

---

## Testing

### `just test`
Run all tests with verbose output, race detection, and coverage.

### `just test-unit`
Run unit tests only (short tests, generates mocks first).

### `just test-integration`
Run integration tests only (requires Docker, uses testcontainers).

### `just test-all`
Run all tests (unit + integration).

### `just test-flow-e2e`
Run the full E2E parimutuel flow (setup → positions → resolve/settle).

### `just test-e2e-reschedule`
Run E2E test for reschedule-refund flow (creator reschedules → all positions refunded).

### `just test-e2e-nowinners`
Run E2E test for no-winners settlement policies (redistribute + void).
Tests both scenarios sequentially: redistribute (partial refund minus 5% fee) and void (full refund).

### `just test-coverage`
Run tests and generate HTML coverage report (opens in browser).

### `just coverage-report`
Generate detailed coverage report (terminal output).

### `just coverage-html`
Generate and open coverage report in browser.

---

## Database & Migrations

### `just migrate-up` or `just migrate`
Run all module migrations (uses modulith's migration system).
- Automatically discovers migrations from all registered modules
- Executes in registration order

### `just migrate-down <module>`
Rollback last migration for a specific module.
Example:
```bash
just migrate-down auth
```

### `just migrate-create <module> <name>`
Create a new migration file for a module.
Example:
```bash
just migrate-create auth add_users_table
```

### `just db-down`
Rollback all migrations for all modules (drops all tables).
- Uses the modulith's migration system to rollback in reverse order of registration
- Includes a resilient fallback that drops schemas if migrations are inconsistent or dirty

### `just db-nuke`
**Guaranteed clean state.** Forcibly drops all module schemas and migration tracking tables.
- Use this if migrations are severely corrupted or when a total reset is needed
- Asks for confirmation before proceeding

### `just db-reset`
Drop database and re-run all migrations (`db-down + migrate-up`).
- Note: If `db-reset` fails due to extreme inconsistency, use `db-nuke` instead

### `just seed`
Run seed data for all modules.

### `just admin <task>`
Run admin task.
Example:
```bash
just admin cleanup_old_sessions
```

---

## Build & Docker

### `just build`
Build the monolith binary (output: `bin/server`).

### `just build-worker`
Build the worker binary (output: `bin/worker`).

### `just build-module <module>`
Build a specific module binary.
Example:
```bash
just build-module auth
```

### `just build-all`
Build all binaries (server + worker + all modules).

### `just clean`
Clean build artifacts (removes `bin/` directory).

### `just docker-up`
Start all Docker Compose services:
- PostgreSQL (database)
- Valkey (cache/sessions)
- Jaeger (tracing)
- Prometheus/Grafana (metrics)

**`just docker-up-minimal`**

Start minimal Docker services (PostgreSQL + Valkey only).

### `just docker-down`
Stop all Docker Compose services.

### `just docker-build`
Build Docker image for server (`modulith-server:latest`).

### `just docker-build-module <module>`
Build Docker image for a specific module.
Example:
```bash
just docker-build-module auth
```

---

## Modules

### `just new-module <name>`
Scaffold a new module with all boilerplate.
Example:
```bash
just new-module orders
```

---

## Frontend

### `just fe-install`
Install dependencies for all web projects (`web/app` and `web/admin`).

### `just fe-dev`
Run development servers for all web projects in parallel.

### `just fe-build`
Build all web projects for production.

### `just fe-lint`
Lint all web projects, including the import alias guard.

### `just fe-lint-fix`
Fix linting issues in all web projects, including the import alias guard.

### `just fe-imports-check`
Verify that frontend imports use aliases (`@/*`) instead of deep relative paths (`../../../`).
- Used in PR/CI workflows to prevent alias regressions.
- If it fails, run `just fe-imports-fix`, review changes, then rerun `just fe-imports-check`.

### `just fe-imports-fix`
Automatically convert deep relative imports to `@/*` aliases.
- Safe to run locally before lint/test when touching frontend files.

### `just fe-test`
Run all frontend tests, including the alias guard unit tests.

### `just fe-guard-test`
Run the unit tests for the frontend import alias guard script.

---

## GraphQL (Optional)

### `just graphql-add` or `just graphql-init`
Add optional GraphQL support using gqlgen (automatically generates code).

### `just graphql-generate` or `just graphql-generate-all`
Generate GraphQL code for all modules (auto-discovers modules with schemas).

### `just graphql-generate-module <module>`
Generate GraphQL code for a specific module.
Example:
```bash
just graphql-generate-module auth
```

### `just graphql-from-proto`
Generate GraphQL schemas from OpenAPI/Swagger files for all modules.

### `just graphql-validate`
Validate GraphQL schema.

---

## Maintenance

### `just lint`
Run golangci-lint to check code quality.

### `just format`
Format code with `gofmt` (and `goimports` if available).
- Uses built-in `gofmt` for code formatting
- Uses `goimports` if installed (also formats imports)
- Provides helpful tip if `goimports` is not installed

### `just tidy`
Tidy Go module dependencies (runs `go mod tidy`).
- Removes unused dependencies
- Adds missing dependencies
- Updates `go.sum` file

### `just pre-commit`
Run pre-commit checks (format + lint + test-unit).
- Formats code
- Runs linter
- Runs unit tests
- Perfect for running before committing changes

---

## 🎯 Common Workflows

### First-Time Setup
```bash
just quickstart
```

### Daily Development
```bash
# Start infrastructure
just docker-up-minimal

# Run migrations (if needed)
just migrate-up

# Start dev server with hot reload
just dev
```

### Before Committing
```bash
# Generate all code
just generate-all

# Run pre-commit checks (format + lint + test-unit)
just pre-commit

# Frontend alias guard (when touching web/app or web/admin)
just fe-imports-check
# Auto-fix if needed
just fe-imports-fix

# Or run individual checks
just format
just lint
just test-unit
```

### Adding a New Module
```bash
# Scaffold module
just new-module orders

# Create migration
just migrate-create MODULE_NAME=orders NAME=create_orders_table

# Generate code (after adding SQL/proto)
just sqlc proto

# Run migrations
just migrate-up
```

### Building for Production
```bash
# Build all binaries
just build-all

# Or build Docker images
just docker-build
just docker-build-module MODULE_NAME=auth
```

---

## 📝 Notes

- Most commands support the `help` target: `just help`
- Docker commands require Docker and Docker Compose
- Development commands (`dev`, `dev-worker`, `dev-module`) require Air to be installed
- Migration commands require database connection (configured via `.env` or `configs/server.yaml`)
- GraphQL commands are optional (only needed if GraphQL support is enabled)

---

## 🔍 Potential Future Improvements

The following commands could be added for even better developer experience:

1. **`just update-deps`**: Update Go dependencies (with confirmation)
2. **`just vendor`**: Vendor dependencies (`go mod vendor`)

