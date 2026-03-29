# 🚀 Go-Modulith-Template

![Go Version](https://img.shields.io/badge/Go-1.25-blue.svg)
![Architecture](https://img.shields.io/badge/Architecture-Modular_Monolith-green.svg)
![License](https://img.shields.io/badge/License-MIT-gray.svg)
![AI-Native](https://img.shields.io/badge/AI-Native_Development-blueviolet.svg)

A professional, high-performance **Modular Monolith ("Modulith")** boilerplate for Go (1.24/1.25+). This project establishes an architectural standard for building robust, scalable, and maintainable applications with a modern, strictly typed technology stack.

> [!NOTE]
> This is a **pure Go** project designed for operational excellence and high-speed development with AI-native tools like Cursor.

---

## ✨ Core Pillars

### 🏛️ Modern Architecture
- **In-Process gRPC**: Module communication via the gRPC stack with zero network overhead.
- **Strict Isolation**: Explicit module boundaries ensuring that "a rotten module never infects the others".
- **API First**: Protocol Buffers as the single source of truth for gRPC, REST (Gateway), and GraphQL.
- **TypeID (UUIDv7)**: Lexicographically sortable, prefixed, and contextual identifiers (Stripe-style).

### 🛠️ Developer Experience (DX)
- **AI-Native Rules**: +10 `.cursor/rules/` files to guide code generation, ensuring architectural consistency and code quality.
- **Comprehensive Tooling**: `just` command-line toolkit for diagnostics (`doctor`), setup (`quickstart`), and daily development.
- **Fast Scaffolding**: One command to create new functional modules with migrations, protos, and config.
- **Hot Reload**: Advanced live-reloading server with `air`.

### 🛡️ Enterprise-Grade Features
- **Security & RBAC**: Internal authorization system with role and permission-based access control.
- **Saga & Outbox Patterns**: Robust handling of distributed transactions and event consistency.
- **Resilience**: Integrated metrics, circuit breakers, and retry mechanisms.
- **Observability**: First-class OpenTelemetry (OTel) support for Metrics and Distributed Tracing.

---

## 🏗️ Technology Stack

| Layer | Technology |
| :--- | :--- |
| **Language** | Go 1.25+ |
| **API** | gRPC, Protocol Buffers, gRPC-Gateway (REST), GraphQL |
| **Persistence** | PostgreSQL, SQLC (Type-safe SQL), golang-migrate |
| **Infrastructure** | Docker Compose, Just (Task runner) |
| **Observability** | OpenTelemetry, Prometheus, Jaeger, slog (Structured Logging) |
| **Security** | JWT, OAuth (goth), Protovalidate |
| **I18n** | go-i18n (Multi-language support) |

---

## 📂 Project Structure

```text
├── cmd/               # Entrypoints (Monolith server, Worker, Visualizer)
├── proto/             # API Definitions (Single Source of Truth)
├── modules/           # Functional Business Modules (Auth, Stock, etc.)
├── internal/          # Shared components (Saga, Outbox, Events, Telemetry)
├── .cursor/rules/     # Architectural guidelines for AI assistants
└── configs/           # Environment-specific YAML configurations
```

---

## ⚡ Quick Start

```bash
# 1. Clone your new project
git clone https://github.com/LoopContext/go-modulith-template.git my-project
cd my-project

# 2. Automated setup (installs deps, starts docker, runs migrations)
just quickstart

# 3. Development with hot reload
just dev
```

---

## 📖 Key Modules & Features

### 👤 Auth Module
A complete authentication provider supporting:
- **Magic Codes**: Passwordless authentication via email.
- **OAuth Integration**: Link accounts from Google, GitHub, Facebook, and more.
- **Session Management**: Secure, persistent user sessions.

### 🔄 Distributed Patterns
- **Events Bus**: Internal Pub/Sub for decoupled module communication.
- **Outbox Pattern**: Ensures event consistency with database transactions.
- **Saga Pattern**: Orchestrates complex flows (like order creation) across modules.

### 🌐 Multiple API Flavors
- **gRPC**: Native high-performance communication.
- **REST**: Automatically exposed via `grpc-gateway`.
- **GraphQL**: Subscriptions and flexible queries (via gqlgen).
- **WebSocket**: Bidirectional real-time events.

---

## 🧪 Testing & Quality

We maintain a strict quality barrier:
- **Testcontainers**: Real database integration tests.
- **Strict Linting**: Comprehensive `golangci-lint` rules (MANDATORY).
- **Mocking**: Automated mock generation for all module interfaces.

```bash
just test-unit       # Fast unit tests
just test-coverage   # Full coverage report
just lint            # Quality validation
```

---

## 🧠 AI-Assisted Development

This project is optimized for **Cursor** and **Windsurf**. The guidelines in `.cursor/rules/` ensure that your AI assistant understands the architecture, from TypeID generation to gRPC error handling.

> [!TIP]
> Just use "Chat" or "Composer" in Cursor and look at the project-specific rules in the status bar to see them in action.

---

Made with ❤️ by the **LoopContext** team for developers seeking operational excellence.
