# Documentation Index

Welcome to the Go Modulith Template documentation! This index organizes all documentation to help you find what you need quickly.

## 📚 Quick Start

**New to the project?** Start here:

1. **[MODULITH_ARCHITECTURE.md](./MODULITH_ARCHITECTURE.md)** - Core architecture guide and implementation standards
2. **[JUST_COMMANDS_REFERENCE.md](./JUST_COMMANDS_REFERENCE.md)** - Complete reference of all `just` commands
3. **[MODULE_COMMUNICATION.md](./MODULE_COMMUNICATION.md)** - How modules communicate (gRPC, events)

---

## 📖 Documentation by Category

### 🏗️ Core Architecture

Essential reading for understanding the template's architecture and design principles.

- **[MODULITH_ARCHITECTURE.md](./MODULITH_ARCHITECTURE.md)** ⭐
  - Complete architecture guide and implementation standards
  - Technology stack, project structure, module isolation rules
  - gRPC, validation, error handling, API versioning strategy
  - Authentication, authorization, configuration
  - Implementation guide from zero to production

- **[MODULE_COMMUNICATION.md](./MODULE_COMMUNICATION.md)**
  - Inter-module communication patterns
  - gRPC service calls (in-process)
  - Event-driven communication
  - Service discovery and client patterns
  - Resilience patterns (circuit breakers, retries)

- **[MODULE_VISUALIZATION.md](./MODULE_VISUALIZATION.md)**
  - Visualizing module dependencies and connections
  - Using the module graph generator
  - Understanding module relationships

### 🛠️ Development Guides

Practical guides for day-to-day development.

- **[JUST_COMMANDS_REFERENCE.md](./JUST_COMMANDS_REFERENCE.md)** ⭐
  - Complete reference of all `just` commands
  - Setup, code generation, development, testing
  - Database migrations, Docker, modules
  - API versioning, GraphQL, maintenance

- **[TESTING_GUIDE.md](./TESTING_GUIDE.md)**
  - Unit testing with mocks (gomock)
  - Integration testing with testcontainers
  - Testing gRPC services, repositories, event bus
  - Test utilities and best practices

- **[ENVIRONMENT.md](./ENVIRONMENT.md)**
  - Environment variable configuration
  - Configuration hierarchy (YAML > .env > ENV vars)
  - Development vs production setup
  - Secrets management

- **[LOGGING_STANDARDS.md](./LOGGING_STANDARDS.md)**
  - Structured logging with `log/slog`
  - Log levels, context, and formatting
  - Best practices for logging
  - Observability integration

### 🔌 Features & Integrations

Guides for specific features and third-party integrations.

- **[OAUTH_INTEGRATION.md](./OAUTH_INTEGRATION.md)**
  - OAuth/Social login setup
  - Supported providers (Google, GitHub, Facebook, etc.)
  - OAuth flow implementation
  - Account linking and management

- **[GRAPHQL_INTEGRATION.md](./GRAPHQL_INTEGRATION.md)**
  - Optional GraphQL support with gqlgen
  - Setting up GraphQL server
  - Schema definition and resolvers
  - Subscriptions via WebSocket

- **[GRAPHQL_AUTO_GENERATION.md](./GRAPHQL_AUTO_GENERATION.md)**
  - Auto-generating GraphQL from Protobuf
  - Converting OpenAPI/Swagger to GraphQL
  - Automated schema generation workflow

- **[WEBSOCKET_GUIDE.md](./WEBSOCKET_GUIDE.md)**
  - Real-time bidirectional communication
  - WebSocket hub and client management
  - Event bus integration
  - Authentication and security

- **[NOTIFICATION_SYSTEM.md](./NOTIFICATION_SYSTEM.md)**
  - Notification templates and providers
  - Email (SendGrid, AWS SES)
  - SMS (Twilio, AWS SNS)
  - Extending with custom providers

### 🎯 Architecture Patterns

Advanced patterns for distributed systems and event-driven architecture.

- **[OUTBOX_PATTERN.md](./OUTBOX_PATTERN.md)**
  - Reliable event publishing pattern
  - Transactional outbox implementation
  - Guaranteed delivery guarantees

- **[SAGA_PATTERNS.md](./SAGA_PATTERNS.md)**
  - Distributed transaction management
  - Saga orchestration patterns
  - Compensation and rollback strategies

- **[WHY_OUTBOX_AND_SAGAS.md](./WHY_OUTBOX_AND_SAGAS.md)**
  - Rationale for using Outbox and Saga patterns
  - Problem statements and solutions
  - When to use each pattern

- **[DISTRIBUTED_EVENTS.md](./DISTRIBUTED_EVENTS.md)**
  - Event-driven architecture in distributed systems
  - Event bus patterns (Redis, Kafka)
  - Event schema versioning
  - Migration strategies

### 📊 Observability & Operations

Monitoring, logging, and operational concerns.

- **[OBSERVABILITY_SETUP.md](./OBSERVABILITY_SETUP.md)**
  - OpenTelemetry integration
  - Prometheus metrics
  - Jaeger tracing
  - Grafana dashboards
  - Local observability stack

- **[LOGGING_STANDARDS.md](./LOGGING_STANDARDS.md)**
  - Structured logging practices
  - Log levels and context
  - Integration with observability tools

### 🚀 Deployment & Infrastructure

Production deployment and infrastructure as code.

- **[DEPLOYMENT_SYNC.md](./DEPLOYMENT_SYNC.md)**
  - Complete deployment infrastructure sync
  - Docker builds and images
  - Kubernetes/Helm charts
  - CI/CD integration
  - Infrastructure as Code (OpenTofu/Terragrunt)

- **[12_FACTOR_APP.md](./12_FACTOR_APP.md)**
  - 12-Factor App principles compliance
  - Configuration, dependencies, logs
  - Process management, port binding
  - Dev/prod parity

### 📋 Planning & Research

Future improvements, research, and proposals.

- **[FRONTEND_PROPOSAL.md](./FRONTEND_PROPOSAL.md)**
  - Frontend integration proposals
  - API gateway patterns
  - GraphQL considerations

- **[IMPROVEMENTS_ROADMAP.md](./IMPROVEMENTS_ROADMAP.md)**
  - Planned improvements and enhancements
  - Feature roadmap
  - Technical debt items

- **[LIBRARY_RESEARCH.md](./LIBRARY_RESEARCH.md)**
  - Library and tool research
  - Technology evaluations
  - Comparison studies

---

## 🗺️ Documentation Map

### By Use Case

**I want to...**

- **Understand the architecture** → [MODULITH_ARCHITECTURE.md](./MODULITH_ARCHITECTURE.md)
- **Set up my development environment** → [JUST_COMMANDS_REFERENCE.md](./JUST_COMMANDS_REFERENCE.md) → Setup section
- **Create a new module** → [MODULITH_ARCHITECTURE.md](./MODULITH_ARCHITECTURE.md) → Module scaffolding
- **Understand module communication** → [MODULE_COMMUNICATION.md](./MODULE_COMMUNICATION.md)
- **Write tests** → [TESTING_GUIDE.md](./TESTING_GUIDE.md)
- **Add OAuth login** → [OAUTH_INTEGRATION.md](./OAUTH_INTEGRATION.md)
- **Set up GraphQL** → [GRAPHQL_INTEGRATION.md](./GRAPHQL_INTEGRATION.md)
- **Deploy to production** → [DEPLOYMENT_SYNC.md](./DEPLOYMENT_SYNC.md)
- **Set up observability** → [OBSERVABILITY_SETUP.md](./OBSERVABILITY_SETUP.md)
- **Understand event patterns** → [OUTBOX_PATTERN.md](./OUTBOX_PATTERN.md), [SAGA_PATTERNS.md](./SAGA_PATTERNS.md)
- **Use WebSockets** → [WEBSOCKET_GUIDE.md](./WEBSOCKET_GUIDE.md)
- **Configure environment** → [ENVIRONMENT.md](./ENVIRONMENT.md)

### By Experience Level

**Beginner:**
1. [MODULITH_ARCHITECTURE.md](./MODULITH_ARCHITECTURE.md) - Start here
2. [JUST_COMMANDS_REFERENCE.md](./JUST_COMMANDS_REFERENCE.md) - Learn the tools
3. [MODULE_COMMUNICATION.md](./MODULE_COMMUNICATION.md) - Understand communication
4. [TESTING_GUIDE.md](./TESTING_GUIDE.md) - Learn testing

**Intermediate:**
- [OAUTH_INTEGRATION.md](./OAUTH_INTEGRATION.md)
- [GRAPHQL_INTEGRATION.md](./GRAPHQL_INTEGRATION.md)
- [WEBSOCKET_GUIDE.md](./WEBSOCKET_GUIDE.md)
- [OBSERVABILITY_SETUP.md](./OBSERVABILITY_SETUP.md)

**Advanced:**
- [OUTBOX_PATTERN.md](./OUTBOX_PATTERN.md)
- [SAGA_PATTERNS.md](./SAGA_PATTERNS.md)
- [DISTRIBUTED_EVENTS.md](./DISTRIBUTED_EVENTS.md)
- [DEPLOYMENT_SYNC.md](./DEPLOYMENT_SYNC.md)

---

## 📝 Documentation Standards

All documentation follows these principles:

- **Practical Examples**: Code examples and real-world scenarios
- **Clear Structure**: Organized sections with clear headings
- **Cross-References**: Links to related documentation
- **Best Practices**: Recommendations and patterns
- **Troubleshooting**: Common issues and solutions

---

## 🔄 Keeping Documentation Updated

This documentation is actively maintained. If you find:

- Outdated information
- Missing features
- Unclear explanations
- Broken links

Please open an issue or submit a pull request to improve the documentation.

---

## 📚 Additional Resources

- **Main README**: See the root [README.md](../README.md) for project overview
- **Contributing**: See [CONTRIBUTING.md](../CONTRIBUTING.md) for contribution guidelines
- **Changelog**: See [CHANGELOG.md](../CHANGELOG.md) for version history

---

## 🎯 Quick Reference

| Document | Purpose | When to Read |
|----------|---------|--------------|
| [MODULITH_ARCHITECTURE.md](./MODULITH_ARCHITECTURE.md) | Core architecture | First read |
| [JUST_COMMANDS_REFERENCE.md](./JUST_COMMANDS_REFERENCE.md) | Command reference | Daily use |
| [MODULE_COMMUNICATION.md](./MODULE_COMMUNICATION.md) | Module patterns | Building modules |
| [TESTING_GUIDE.md](./TESTING_GUIDE.md) | Testing practices | Writing tests |
| [DEPLOYMENT_SYNC.md](./DEPLOYMENT_SYNC.md) | Deployment guide | Going to production |
| [OBSERVABILITY_SETUP.md](./OBSERVABILITY_SETUP.md) | Monitoring setup | Setting up observability |

---

**Last Updated**: January 2025

