# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

-   WebSocket security implementation with JWT authentication
-   Origin checking for WebSocket connections with configurable allowed origins
-   Support for multiple authentication methods in WebSocket handler (query param, cookie, Authorization header)
-   Integration test examples demonstrating end-to-end module testing
-   Example integration tests showing testcontainers usage, migration testing, and event bus integration
-   CHANGELOG.md following Keep a Changelog format

### Changed

-   WebSocket handler now requires HandlerConfig for initialization
-   WebSocket authentication now uses JWT verifier from configuration
-   WebSocket origin checking now respects CORS configuration

### Security

-   WebSocket connections now require authentication in production mode
-   Origin checking prevents unauthorized cross-origin WebSocket connections
-   Fail-secure default: production mode denies connections when no origins configured

## [0.1.0] - 2025-01-XX

### Added

-   Initial release of Go Modulith Template
-   Modular architecture with registry pattern
-   gRPC and Protocol Buffer support with automatic code generation
-   SQLC for type-safe database access
-   Database migrations with automatic discovery per module
-   Hot reload development with Air
-   WebSocket real-time communication with event bus integration
-   Complete authentication module with passwordless login
-   OAuth/Social login support (Google, Facebook, GitHub, Apple, Microsoft, Twitter)
-   Worker process for background tasks
-   Event bus for inter-module communication
-   Complete observability stack (OpenTelemetry, Prometheus, Grafana, Jaeger)
-   Optional GraphQL support with gqlgen
-   Notification system with multiple providers (SendGrid, Twilio, AWS SES/SNS)
-   Internationalization (i18n) support
-   Feature flags system
-   Circuit breaker and retry mechanisms
-   Secrets management abstraction
-   Health check endpoints
-   Administrative tasks system
-   Module scaffolding script
-   Comprehensive documentation
-   CI/CD with GitHub Actions
-   Docker and Kubernetes deployment support
-   OpenTofu infrastructure as code
-   Helm charts for Kubernetes deployment

### Documentation

-   Architecture guide (MODULITH_ARCHITECTURE.md)
-   Module communication guide (MODULE_COMMUNICATION.md)
-   12-Factor App compliance guide (12_FACTOR_APP.md)
-   OAuth integration guide (OAUTH_INTEGRATION.md)
-   Notification system guide (NOTIFICATION_SYSTEM.md)
-   WebSocket guide (WEBSOCKET_GUIDE.md)
-   GraphQL integration guide (GRAPHQL_INTEGRATION.md)
-   Deployment guide (DEPLOYMENT_SYNC.md)
-   Frontend prloopcontextal (FRONTEND_PRLoopContextAL.md)
-   Getting started guide (GETTING_STARTED.md)
-   Contributing guidelines (CONTRIBUTING.md)

---

## Types of Changes

-   **Added** for new features
-   **Changed** for changes in existing functionality
-   **Deprecated** for soon-to-be removed features
-   **Removed** for now removed features
-   **Fixed** for any bug fixes
-   **Security** for vulnerability fixes
