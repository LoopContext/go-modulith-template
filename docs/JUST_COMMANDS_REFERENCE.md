# Just Commands Reference 🛠️

This document provides a complete reference for all `just` commands available in this project. Use these commands to automate development, testing, and deployment tasks.

> [!TIP]
> Run `just` or `just --list` in the root of the project to see a quick summary of available commands.

## 📋 Table of Contents

1. [Quick Start & Setup](#quick-start--setup)
2. [Code Generation](#code-generation)
3. [Development & Running](#development--running)
4. [Testing & Quality](#testing--quality)
5. [Docker & Infrastructure](#docker--infrastructure)
6. [Database & Migrations](#database--migrations)
7. [GraphQL](#graphql)
8. [Modules](#modules)
9. [Observability & Visualization](#observability--visualization)
10. [Bots & Messaging](#bots--messaging)

---

## ⚡ Quick Start & Setup

| Command | Description |
|---------|-------------|
| `just quickstart` | Full automated setup (deps, docker, migrations, seed). |
| `just setup` | Setup infrastructure and prepare the database. |
| `just install-deps` | Install all required Go development tools. |
| `just doctor` | Run environment diagnostics to ensure everything is ready. |
| `just demo` | Complete end-to-end demo (setup + example flow). |

## 🧬 Code Generation

| Command | Description |
|---------|-------------|
| `just generate-all` | Generate all code (SQLC, Proto, and Mocks). |
| `just sqlc` | Generate type-safe Go code from SQL queries. |
| `just proto` | Generate gRPC and OpenAPI code from Protobuf definitions. |
| `just generate-mocks` | Regenerate all test mocks. |

## 🚀 Development & Running

| Command | Description |
|---------|-------------|
| `just dev` | Start the full stack (Backend + Frontend) with hot reload in tmux. |
| `just dev-module <name>` | Start a specific module with hot reload. |
| `just build` | Build the server binary. |
| `just run` | Run the server without hot reload. |
| `just stop` | Stop the tmux development session. |

## 🧪 Testing & Quality

| Command | Description |
|---------|-------------|
| `just test` | Run all unit and integration tests. |
| `just test-unit` | Run unit tests (fast, skips integration). |
| `just test-integration` | Run tests that require Docker/Testcontainers. |
| `just lint` | Run the linter. |
| `just lint-fix` | Run the linter and automatically fix issues. |
| `just check` | Run format, lint, and unit tests (pre-commit). |
| `just coverage-report` | Show code coverage in the terminal. |
| `just coverage-html` | Open code coverage report in your browser. |

## 🐳 Docker & Infrastructure

| Command | Description |
|---------|-------------|
| `just docker-up` | Start all Docker infrastructure services. |
| `just docker-up-minimal` | Start only essential services (DB + Valkey). |
| `just docker-down` | Stop and remove all Docker containers. |
| `just docker-build` | Build the server Docker image. |

## 🗄️ Database & Migrations

| Command | Description |
|---------|-------------|
| `just migrate` | Apply all pending database migrations. |
| `just seed` | Populate the database with initial/test data. |
| `just migrate-create <mod> <name>` | Create a new migration file for a module. |
| `just db-reset` | **Destructive**: Reset the database and re-run migrations. |

## 🕸️ GraphQL

| Command | Description |
|---------|-------------|
| `just add-graphql` | Add basic GraphQL support to the project. |
| `just graphql-generate` | Generate GraphQL code from schemas. |
| `just graphql-from-proto` | Generate GraphQL schemas from Protobuf/OpenAPI. |

## 📦 Modules

| Command | Description |
|---------|-------------|
| `just new-module <name>` | Scaffold a new domain module structure. |
| `just destroy-module <name>` | Completely remove a domain module. |

## 📊 Observability & Visualization

| Command | Description |
|---------|-------------|
| `just visualize` | Generate a module connection graph (HTML). |

## 🤖 Bots & Messaging

| Command | Description |
|---------|-------------|
| `just bot-register <type>` | Register a provider (whatsapp/telegram). |
| `just bot-simulate-wa-msg` | Simulate a WhatsApp incoming message via webhook. |

---

> [!NOTE]
> All `just` commands are defined in the `justfile` located at the root of the repository. You can inspect it to see the underlying shell commands.
