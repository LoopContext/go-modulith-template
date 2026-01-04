# Getting Started Guide: Clone, Setup, and Add a New Module

This guide will walk you through the complete process of cloning this repository as a template, setting it up, and adding a new module from scratch.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Step 1: Clone the Repository](#step-1-clone-the-repository)
3. [Step 2: Setup Project](#step-2-setup-project)
4. [Step 3: Configure the Project](#step-3-configure-the-project)
5. [Step 4: Start Infrastructure](#step-4-start-infrastructure)
6. [Step 5: Add a New Module](#step-5-add-a-new-module)
7. [Step 6: Register the Module](#step-6-register-the-module)
8. [Step 7: Generate Code](#step-7-generate-code)
9. [Step 8: Run Migrations](#step-8-run-migrations)
10. [Step 9: Test Your Module](#step-9-test-your-module)
11. [Step 10: Run the Server](#step-10-run-the-server)

---

## Prerequisites

Before you begin, ensure you have the following installed:

-   **Go 1.24+** - [Download Go](https://go.dev/dl/)
-   **Docker & Docker Compose** - [Install Docker](https://docs.docker.com/get-docker/)
-   **Git** - For cloning the repository

---

## Step 1: Clone the Repository

Clone this repository to use it as a template for your project:

```bash
# Clone the repository
git clone https://github.com/cmelgarejo/go-modulith-template.git my-project
cd my-project

# Remove the existing git history (optional, if you want a fresh start)
rm -rf .git
git init
git add .
git commit -m "Initial commit from go-modulith-template"
```

> **Note:** If you're using this as a GitHub template, you can use the "Use this template" button on GitHub, which will create a new repository with a clean history.

---

## Step 2: Setup Project

### Option A: Quick Setup (Recommended)

For the fastest setup, use the automated quickstart script:

```bash
make quickstart
```

This will automatically:

-   Validate your environment
-   Install missing development tools
-   Start Docker infrastructure
-   Run database migrations
-   Optionally run seed data

### Option B: Manual Setup

#### 2.1 Install Dependencies

Install all required development tools:

```bash
make install-deps
```

This will install:

-   `migrate` - Database migration tool
-   `sqlc` - Type-safe SQL code generator
-   `buf` - Protocol buffer compiler
-   `air` - Hot reload tool for development
-   `golangci-lint` - Go linter
-   `gqlgen` - GraphQL code generator (optional)
-   `mockgen` - Mock generator for testing

**Verify installations:**

```bash
# Check that tools are installed
which migrate sqlc buf air golangci-lint

# Or run validation
make validate-setup
```

---

## Step 3: Configure the Project

### 3.1 Update Module Name (Optional)

If you want to change the Go module name, update `go.mod`:

```bash
# Edit go.mod and change the module path
# From: module github.com/cmelgarejo/go-modulith-template
# To:   module github.com/your-org/your-project
```

Then update all imports in the codebase. You can use a find-and-replace tool or script.

### 3.2 Configure Database and Environment

The project uses a flexible configuration system with the following priority:

1. Environment variables (highest priority)
2. `.env` file
3. `configs/server.yaml` (default configuration)

#### Option A: Use YAML configuration (recommended for development)

Edit `configs/server.yaml`:

```yaml
env: dev
log_level: debug # debug, info, warn, error
http_port: 8000
grpc_port: 9000
service_name: modulith-server
db_dsn: postgres://postgres:postgres@localhost:5432/modulith_demo?sslmode=disable

# Database connection pool settings
db_max_open_conns: 25
db_max_idle_conns: 25
db_conn_max_lifetime: 5m
db_connect_timeout: 10s

# Timeouts
read_timeout: 5s # HTTP server read timeout
write_timeout: 10s # HTTP server write timeout
shutdown_timeout: 30s # Graceful shutdown timeout

auth:
    jwt_secret: your-secret-key-at-least-32-bytes-long-change-this
```

#### Option B: Use environment variables

Create a `.env` file (optional):

```bash
# Copy example if available, or create new
cat > .env <<EOF
ENV=dev
LOG_LEVEL=debug
HTTP_PORT=8000
GRPC_PORT=9000
DB_DSN=postgres://postgres:postgres@localhost:5432/modulith_demo?sslmode=disable
JWT_SECRET=your-secret-key-at-least-32-bytes-long-change-this
EOF
```

> **Important:** Change the `JWT_SECRET` to a secure random string (at least 32 bytes) before deploying to production.

---

## Step 4: Start Infrastructure

> **Note:** If you used `make quickstart`, this step is already complete. Skip to Step 5.

Start the required infrastructure services (PostgreSQL, Redis, etc.):

```bash
make docker-up
```

This starts:

-   **PostgreSQL** - Main database (port 5432)
-   **Redis** - Cache and session storage (port 6379)
-   **Jaeger** - Distributed tracing UI [http://localhost:16686](http://localhost:16686)
-   **Prometheus** - Metrics collection [http://localhost:9090](http://localhost:9090)
-   **Grafana** - Visualization dashboards ([http://localhost:3000](http://localhost:3000), user: `admin`, password: `admin`)

**Verify services are running:**

```bash
docker-compose ps

# Or run comprehensive diagnostics
make doctor
```

You should see all services in "Up" status.

> **Tip:** To start only the database and Redis (faster startup), use `make docker-up-minimal`.

---

## Step 5: Add a New Module

Now let's add a new module. For this example, we'll create an "order" module:

```bash
make new-module order
```

This command will:

-   Create the module directory structure
-   Generate boilerplate code from templates
-   Create migration files
-   Create proto definitions
-   Generate Air configuration for hot reload
-   Update `sqlc.yaml` with the new module

**Generated structure:**

```bash
modules/order/
├── module.go                    # Module implementation
├── internal/
│   ├── service/
│   │   └── service.go          # Business logic
│   ├── repository/
│   │   └── repository.go     # Data access layer
│   └── db/
│       └── query/
│           └── order.sql      # SQL queries
└── resources/
    └── db/
        ├── migration/          # Database migrations
        └── seed/               # Seed data

proto/order/v1/
└── order.proto                 # gRPC service definition

cmd/order/
└── main.go                     # Standalone service entry point

configs/
└── order.yaml                  # Module configuration

.air.order.toml                  # Hot reload config for this module
```

**What was generated:**

1. **Module structure** - Complete module with service, repository, and database layers
2. **gRPC service** - Protocol buffer definition for the module
3. **Database migrations** - Initial schema migration files
4. **SQL queries** - Template SQL queries for sqlc
5. **Configuration** - Module-specific YAML configuration
6. **Standalone binary** - Entry point to run the module independently

---

## Step 6: Register the Module

After scaffolding, you need to register the module in the main server. Edit `cmd/server/setup/registry.go`:

**Find the `RegisterModules` function:**

```go
// RegisterModules registers all modules with the registry.
func RegisterModules(reg *registry.Registry) {
    // Register all modules here
    reg.Register(auth.NewModule())
    // Add more modules as needed:
    // reg.Register(order.NewModule())
    // reg.Register(payment.NewModule())
}
```

**Add your new module:**

```go
// RegisterModules registers all modules with the registry.
func RegisterModules(reg *registry.Registry) {
    // Register all modules here
    reg.Register(auth.NewModule())
    reg.Register(order.NewModule())  // Add this line
    // Add more modules as needed:
    // reg.Register(payment.NewModule())
}
```

**Add the import at the top of the file:**

```go
import (
    // ... existing imports ...
    "github.com/cmelgarejo/go-modulith-template/modules/auth"
    "github.com/cmelgarejo/go-modulith-template/modules/order"  // Add this line
    // ... rest of imports ...
)
```

---

## Step 7: Generate Code

Now generate the code from your proto definitions and SQL queries:

### 7.1 Generate gRPC Code

Generate Go code from your Protocol Buffer definitions:

```bash
make proto
```

This will:

-   Generate gRPC service code from `proto/order/v1/order.proto`
-   Create client and server stubs
-   Generate OpenAPI/Swagger documentation in `gen/openapiv2/`

### 7.2 Generate SQL Code

Generate type-safe Go code from your SQL queries:

```bash
make sqlc
```

This will:

-   Generate Go code from SQL queries in `modules/order/internal/db/query/`
-   Create type-safe database access code in `modules/order/internal/db/store/`
-   Generate repository interfaces

**Verify generated code:**

```bash
# Check that files were generated
ls -la gen/go/proto/order/v1/
ls -la modules/order/internal/db/store/
```

---

## Step 8: Run Migrations

Run database migrations to create the schema for your new module:

```bash
make migrate
```

Or run migrations manually using the subcommand:

```bash
go run cmd/server/main.go migrate
```

Or using the flag:

```bash
go run cmd/server/main.go -migrate
```

This will:

-   Discover all modules with migrations
-   Run migrations in order
-   Create database tables for your new module

**Verify migrations:**

```bash
# Connect to the database
psql postgres://postgres:postgres@localhost:5432/modulith_demo

# List tables
\dt

# Check migration versions (each module has its own migration tracking)
SELECT * FROM schema_migrations;
```

You should see tables for your new module (e.g., `orders` table if you created an order module).

> **Note:** Migrations run automatically when you start the server. The modulith discovers and applies migrations for all registered modules.

---

## Step 9: Test Your Module

### 9.1 Run Unit Tests

Run tests to verify everything works:

```bash
make test
```

Or run tests for a specific module:

```bash
go test ./modules/order/...
```

### 9.2 Generate Mocks (if needed)

If you're writing tests that require mocks:

```bash
make generate-mocks
```

This generates mocks for all interfaces in your modules.

### 9.3 Verify Code Quality

Run the linter to ensure code quality:

```bash
make lint
```

Fix any issues reported by the linter.

---

## Step 10: Run the Server

Now you're ready to run the server with your new module!

### Option A: Run with Hot Reload (Recommended for Development)

Run the monolith server with hot reload:

```bash
make dev
```

This will:

-   Start the gRPC server (port 9000 by default, configurable)
-   Start the HTTP gateway (port 8000 by default, configurable)
-   Automatically reload on code changes
-   Monitor changes in `.go`, `.yaml`, `.env`, `.proto`, and `.sql` files
-   Run migrations automatically on startup

### Option B: Run a Specific Module Standalone

Run your module as a standalone service:

```bash
make dev-module order
```

This runs only the order module with hot reload.

### Option C: Build and Run

Build and run without hot reload:

```bash
# Build the server
make build

# Run it
./bin/server
```

---

## Verify Everything Works

### 10.1 Check Health Endpoints

```bash
# Liveness probe
curl http://localhost:8000/livez

# Readiness probe (checks all dependencies)
curl http://localhost:8000/readyz

# WebSocket health
curl http://localhost:8000/healthz/ws
```

### 10.2 Check gRPC Service

If you have `grpcurl` installed:

```bash
# List services
grpcurl -plaintext localhost:9000 list

# List methods for your service
grpcurl -plaintext localhost:9000 list order.v1.OrderService
```

### 10.3 Check Swagger UI (Development Only)

In development mode, Swagger UI is available at:

```bash
http://localhost:8000/swagger-ui/
```

You can explore and test your API endpoints here.

### 10.4 Check Logs

The server logs will show:

-   Module initialization
-   Migration status
-   Server startup
-   Request logs

Look for messages like:

```bash
Starting application version=...
Module 'order' initialized successfully
✅ Migrations completed successfully
Starting gRPC server port=9000
Starting HTTP Gateway port=8000
```

---

## Testing Your Module

### Unit Tests

Write unit tests for your service and repository layers:

```bash
# Run unit tests for your module
go test ./modules/order/... -v

# Run with coverage
go test ./modules/order/... -cover
```

### Integration Tests

For integration tests that require a real database, use testcontainers:

```bash
# Run integration tests (requires Docker)
make test-integration

# Or run specific integration tests
go test -v -run Integration ./examples/...
```

**Example Integration Test:**

See `examples/integration_test_example.go` for a complete example showing:

-   Setting up PostgreSQL with testcontainers
-   Running migrations in tests
-   Testing service methods end-to-end
-   Verifying event bus integration
-   Testing repository layer with real database

**Key Points:**

-   Integration tests should be marked with `-run Integration` flag
-   Use `testing.Short()` to skip integration tests in short mode
-   Always clean up test data and containers in `defer` blocks
-   Use the `testutil` package for testcontainer setup

### Test Coverage

```bash
# Generate coverage report
make coverage-report

# View HTML coverage report
make coverage-html
```

## Next Steps

Now that you have a working module, you can:

1. **Customize the Module**

    - Edit `modules/order/internal/service/service.go` to add business logic
    - Update `modules/order/internal/repository/repository.go` for data access
    - Add more SQL queries in `modules/order/internal/db/query/`

2. **Add More Methods**

    - Update `proto/order/v1/order.proto` to add new RPC methods
    - Run `make proto` to regenerate code
    - Implement the methods in your service

3. **Add Database Migrations**

    ```bash
    make migrate-create MODULE=order NAME=add_indexes
    ```

    This creates new migration files in `modules/order/resources/db/migration/`

4. **Add Seed Data**

    - Edit `modules/order/resources/db/seed/001_example_data.sql`
    - Run `make seed` or `go run cmd/server/main.go seed`
    - Note: The module must implement `SeedPath()` method in `module.go` (automatically included when using `make new-module`)

5. **Add Tests**

    - Write unit tests in `modules/order/internal/service/service_test.go`
    - Write integration tests using testcontainers

6. **Configure Module Settings**
    - Edit `configs/order.yaml` for module-specific configuration
    - Access configuration in your module via `registry.Config()`

---

## Troubleshooting

### Diagnostic Tools

Before troubleshooting, run diagnostic tools:

```bash
# Comprehensive environment diagnostics
make doctor

# Validate setup and prerequisites
make validate-setup
```

### Issue: Module not found after registration

**Solution:** Make sure you:

1. Added the import for your module in `cmd/server/setup/registry.go`
2. Called `reg.Register(order.NewModule())` in `RegisterModules` function
3. Ran `go mod tidy` to update dependencies
4. The module path in the import matches your actual module path

### Issue: Migrations fail

**Solution:**

-   Check database connection string in `configs/server.yaml`
-   Ensure PostgreSQL is running: `docker-compose ps` or `make doctor`
-   Check migration files are valid SQL
-   Verify database container is healthy: `docker ps`

### Issue: Proto generation fails

**Solution:**

-   Verify `buf` is installed: `which buf` or `make validate-setup`
-   Check `buf.yaml` and `buf.gen.yaml` are correct
-   Ensure proto files are valid: `buf lint`

### Issue: SQLC generation fails

**Solution:**

-   Check `sqlc.yaml` has correct paths
-   Verify SQL queries are valid
-   Ensure migration files exist for schema

### Issue: Server won't start

**Solution:**

-   Run `make doctor` to check environment health
-   Check logs for specific errors
-   Verify all required environment variables are set
-   Ensure database is accessible: `make doctor` will check this
-   Check ports 8000 and 9000 are not in use: `make validate-setup` shows port status
-   Verify Docker containers are running: `docker-compose ps`

### Issue: Port conflicts

**Solution:**

-   Run `make doctor` to identify which ports are in use
-   Stop conflicting services or change ports in `configs/server.yaml`
-   Check `docker-compose ps` for running containers

---

## Summary

You've successfully:

1. ✅ Cloned the repository
2. ✅ Installed all dependencies
3. ✅ Configured the project
4. ✅ Started infrastructure
5. ✅ Created a new module
6. ✅ Registered the module
7. ✅ Generated code (proto + sqlc)
8. ✅ Run migrations
9. ✅ Tested the module
10. ✅ Started the server

Your modulith application is now running with your new module! 🎉

For more information, see:

-   [Architecture Guide](docs/MODULITH_ARCHITECTURE.md)
-   [Module Communication](docs/MODULE_COMMUNICATION.md)
-   [API Documentation](README.md#api-documentation)
