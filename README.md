# Go Modulith Template 🚀

<p align="center">
  <img src="docs/assets/hero.png" width="800" alt="Go Modulith Architecture">
</p>


![Tests](https://img.shields.io/badge/tests-passing-brightgreen)
![Coverage](https://img.shields.io/badge/coverage-19.9%25-yellow)
![Go](https://img.shields.io/badge/go-1.25+-blue)
![License](https://img.shields.io/badge/license-MIT-blue)

This is a professional, production-ready template for building Go applications following the **Modular Monolith (Modulith)** pattern. Designed for high-performance, maintainability, and scalability, it allows your application to evolve from a monolith to microservices without architectural friction.

## ✨ Key Features

-   🏗️ **Modular Architecture**: Clean domain boundaries with decoupling via an internal **Event Bus**.
-   📦 **Registry Pattern**: Explicit dependency injection without magic for maximum control and testability.
-   🔐 **gRPC & Protobuf**: High-performance, type-safe RPC communication with automated generation via `buf`.
-   🗄️ **SQLC & Migrations**: Type-safe data access and multi-module schema management with `golang-migrate`.
-   ⚡ **High-Performance Caching**: Native support for **Valkey** (the open-source Redis alternative) for sessions and rate limiting.
-   🔐 **Complete Auth System**: Passwordless login, sessions, JWT, refresh tokens, and RBAC.
-   🔗 **OAuth/Social Login**: Integrated with **Goth** for Google, Facebook, GitHub, Apple, Microsoft, and Twitter/X.
-   🤖 **Messaging Bot Engine**: Built-in support for **WhatsApp** and **Telegram** provider integrations.
-   🔌 **WebSocket Real-Time**: Bidirectional communication integrated with the event bus for instant notifications.
-   📊 **Observability Stack**: Native integration with **OpenTelemetry**, Jaeger (Tracing), Prometheus (Metrics), and Grafana dashboards.
-   ⚙️ **Flexible Configuration**: Hierarchy-based system (PORT > YAML > .env > system ENV > defaults) with source logging.
-   ⚡ **Resilience & Errors**: Integrated circuit breakers, retries, and a domain-specific error system mapped to gRPC codes.
-   📧 **Notification System**: Extensible providers (SendGrid, Twilio, AWS SES/SNS) with template support.
-   📊 **Optional GraphQL**: Advanced support with gqlgen for flexible frontend APIs (subscriptions included).
-   🧪 **Test Utilities**: Comprehensive suite (`internal/testutil`) for integration tests with **Testcontainers**, gRPC servers, and mocks.
-   🛠️ **DevX Excellence**: Hot reload with **Air**, task automation with **Just**, and environment diagnostics with `doctor`.

## 🛠️ Prerequisites

-   Go 1.25+
-   Docker & Docker Compose
-   Development tools: `sqlc`, `buf`, `migrate`, `air`, `golangci-lint`, `just`.

## 🚀 Quick Start

### 1. Automated Setup & Run (Recommended)

The fastest way to get started is using the integrated setup and run command:

```bash
git clone https://github.com/LoopContext/go-modulith-template.git my-project
cd my-project
just dev
```

This single command will:
1. Start the minimal Docker infrastructure (DB + Valkey).
2. Wait for the database to be ready.
3. Run all database migrations.
4. Seed the database with test users (`admin`, `system`, `user`).
5. Install frontend dependencies (`web/solid-example`).
6. Start the **full stack (Backend + Frontend)** with **Hot Reload** in a 3-pane tmux session.

### 2. See it in Action

To run a complete representative flow (E2E) and see how the system handles Auth, Events, and Logic:

```bash
just example
```

> 💡 **Tip**: For a full "setup + example" demo in one go, use `just demo`.

### 3. Development Mode

Run the monolith with hot reload (monitors code, proto, sql, and configs):

```bash
just dev
```

Run a specific module (e.g., auth):

```bash
just dev-module auth
```

## 🏗️ Project Structure

-   `cmd/`: Main entry points (server, worker, admin tasks, migration ops).
-   `internal/`: Core shared services (registry, cache, events, authz, telemetry).
-   `modules/`: Domain-specific modules (auth, stock, etc.). Each module is independent.
-   `proto/`: Protobuf definitions for gRPC and Event schemas.
-   `scripts/`: Utility scripts for DevX (scaffolding, validation, e2e).
-   `web/`: Documentation site and optional frontend examples.

## 📖 Complete Documentation

-   **[Architecture Guide](docs/MODULITH_ARCHITECTURE.md)** - Patterns, internal communication, and error handling.
-   **[Module Communication](docs/MODULE_COMMUNICATION.md)** - In-process vs Network gRPC and the Event Bus.
-   **[OAuth/Social Integration](docs/OAUTH_INTEGRATION.md)** - Social login setup guide.
-   **[Messaging Bot Engine](docs/NOTIFICATION_SYSTEM.md)** - WhatsApp and Telegram provider setup.
-   **[Real-Time WebSocket](docs/WEBSOCKET_GUIDE.md)** - Bidirectional event broadcasting.
-   **[GraphQL Integration](docs/GRAPHQL_INTEGRATION.md)** - Optional gqlgen setup.
-   **[Deployment Guide](deployment/README.md)** - Kubernetes, Helm Charts, and IaC (OpenTofu).

## 🛠️ Main Commands (`just`)

-   `just proto`: Generate gRPC and OpenAPI code.
-   `just sqlc`: Generate type-safe SQL code.
-   `just new-module <name>`: Scaffold a new domain module.
-   `just setup`: Automated, non-interactive setup (infra + migrate + seed).
-   `just example`: Run a representative example flow (E2E) to see the system in action.
-   `just demo`: Complete end-to-end demo (setup + example).
-   `just test`: Run all unit and integration tests.
-   `just lint`: Run strict linter (MANDATORY for quality).
-   `just visualize`: Generate a visual graph of module connections.
-   `just admin TASK=<name>`: Execute maintenance tasks (e.g., `cleanup-sessions`).

---

Made with ❤️ for developers seeking operational excellence.
