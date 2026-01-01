# Go Modulith Template 🚀

![Tests](https://img.shields.io/badge/tests-passing-brightgreen)
![Coverage](https://img.shields.io/badge/coverage-19.9%25-yellow)
![Go](https://img.shields.io/badge/go-1.24+-blue)
![License](https://img.shields.io/badge/license-MIT-blue)

Este es un template profesional para construir aplicaciones en Go siguiendo el patrón **Modulith**. Está diseñado para ser escalable, sostenible y fácil de mantener, permitiendo evolucionar de un monolito a microservicios sin fricción.

## ✨ Características Principales

-   🏗️ **Arquitectura Modular**: Código organizado por dominios con desacoplamiento mediante eventos internos.
-   📦 **Registry Pattern**: Inyección de dependencias manual, explícita y sin magia para máximo control.
-   🔐 **gRPC & Protobuf**: Comunicación tipada y eficiente con generación automática vía `buf`.
-   🗄️ **SQLC & Migraciones**: Acceso a datos Type-safe y gestión de esquemas con `golang-migrate`.
-   ⚙️ **Configuración Flexible**: Sistema de configuración con jerarquía de precedencia (YAML > .env > system ENV vars > defaults) y logging de fuentes.
-   🔄 **Hot Reload**: Desarrollo fluido con **Air** que monitorea cambios en código, configuración (`.env`, YAML) y recursos.
-   🔌 **WebSocket Real-Time**: Comunicación bidireccional integrada con el event bus para notificaciones en tiempo real.
-   📊 **GraphQL Opcional**: Soporte opcional con gqlgen para APIs flexibles y frontend-friendly (subscriptions incluidas).
-   📧 **Sistema de Notificaciones**: Templates + providers extensibles (SendGrid, Twilio, AWS SES/SNS).
-   🔑 **Auth Completo**: Login passwordless, sesiones, refresh tokens, revocación y gestión de perfil.
-   🔗 **OAuth/Social Login**: Autenticación con Google, Facebook, GitHub, Apple, Microsoft y Twitter/X.
-   🧪 **Mocking con gomock**: Generación automática de mocks type-safe para testing unitario eficiente.
-   🛡️ **Observabilidad**: Integración nativa con OpenTelemetry (Tracing & Metrics), Prometheus y Health Checks con manejo de contextos.
-   ⚡ **Error Handling**: Sistema de errores de dominio con mapeo automático a códigos gRPC.
-   📡 **Telemetry Helpers**: Helpers integrados para tracing consistente en todos los módulos.
-   🎯 **Eventos Tipados**: Constantes tipadas para eventos con autocomplete y prevención de typos.
-   🔄 **Migraciones Multi-Módulo**: Descubrimiento y ejecución automática de migraciones por módulo.
-   🔐 **RBAC Built-in**: Helpers de autorización para permisos, roles y ownership.
-   ⛴️ **Cloud Ready**: Dockerfile multi-stage y Helm Charts flexibles para Kubernetes (soporta monolito y módulos independientes).
-   🌍 **IaC con OpenTofu**: Infraestructura base reproducible (VPC, EKS, RDS) gestionada con OpenTofu y Terragrunt.
-   🤖 **CI/CD**: Pipelines de GitHub Actions para validación automática.

## 🛠️ Requisitos Previos

-   Go 1.24+
-   Docker & Docker Compose
-   Herramientas de desarrollo:
    -   `sqlc`
    -   `buf`
    -   `migrate`
    -   `air`
    -   `golangci-lint`

## 🚀 Inicio Rápido

### 1. Instalar dependencias

```bash
make install-deps
```

### 2. Levantar Infraestructura (DB)

```bash
make docker-up
```

### 3. Configurar (Opcional)

El proyecto soporta múltiples fuentes de configuración con precedencia clara:

-   **YAML** (`configs/server.yaml`): Mayor prioridad, ideal para configuraciones por entorno
-   **`.env`**: Sobrescribe variables del sistema
-   **Variables de entorno del sistema**: Valores base
-   **Defaults**: Valores hardcodeados en `config.go`

```bash
# Copia el archivo de ejemplo para variables de entorno
cp .env.example .env

# Edita .env con tus valores (DB, JWT secret, OAuth, etc.)
# O configura directamente en configs/server.yaml
```

Al iniciar, verás un log mostrando la fuente de cada variable de configuración.

> 💡 **Tip para OAuth**: Para habilitar proveedores OAuth (Google, GitHub, etc.), configura las credenciales en `configs/server.yaml` o en tu archivo `.env`. Ver [guía completa de OAuth](docs/OAUTH_INTEGRATION.md).

### 4. Ejecutar en Desarrollo (Hot Reload)

```bash
make dev
```

Para ejecutar un módulo específico con hot reload:

```bash
make dev-module auth
```

> 💡 **Tip**: Air monitorea automáticamente cambios en `.go`, `.yaml`, `.env`, `.proto`, `.sql` y archivos de configuración, reiniciando el servidor instantáneamente.

## 📖 Documentación Completa

-   **[Guía de Arquitectura](docs/MODULITH_ARCHITECTURE.md)** - ⭐ Arquitectura completa, patrones, manejo de errores, telemetría, eventos tipados, RBAC, testing y más
-   **[OAuth/Social Login](docs/OAUTH_INTEGRATION.md)** - Integración con Google, Facebook, GitHub, Apple, Microsoft, Twitter
-   **[Sistema de Notificaciones](docs/NOTIFICATION_SYSTEM.md)** - Templates, providers (SendGrid, Twilio, SES) y composite notifier
-   **[WebSocket en Tiempo Real](docs/WEBSOCKET_GUIDE.md)** - Comunicación bidireccional, event bus y autenticación JWT
-   **[Integración GraphQL](docs/GRAPHQL_INTEGRATION.md)** - Setup opcional con gqlgen, schema por módulo y subscriptions
-   **[Deployment & IaC](docs/DEPLOYMENT_SYNC.md)** - OpenTofu, Helm Charts, estrategias de despliegue y testing
-   **[Propuesta de Frontend](docs/FRONTEND_PROPOSAL.md)** - Go Templates + HTMX con WebSocket/GraphQL
-   **[Deployment Guide](deployment/README.md)** - Guía completa de despliegue en Kubernetes
-   **[Helm Chart Documentation](deployment/helm/modulith/README.md)** - Documentación detallada del Helm chart

## 📋 API Documentation

El proyecto genera automáticamente documentación OpenAPI/Swagger:

-   **Ubicación**: `gen/openapiv2/proto/` (generada con `make proto`)
-   **Formato**: JSON compatible con Swagger UI
-   **Uso**: Importa los archivos `.swagger.json` en [Swagger Editor](https://editor.swagger.io/) o cualquier herramienta compatible

Ejemplo para el módulo de auth:
```bash
# Genera la documentación
make proto

# Visualiza la API
open gen/openapiv2/proto/auth/v1/auth.swagger.json
```

## 🛠️ Comandos Útiles (Makefile)

### Generación de Código

-   `make proto`: Genera código gRPC desde archivos `.proto` (incluye OpenAPI/Swagger en `gen/openapiv2/`).
-   `make sqlc`: Genera código Type-safe para queries SQL.

### Build

-   `make build`: Compila el binario del monolito en `bin/server`.
-   `make build-module MODULE_NAME`: Compila el binario de un módulo específico (ej: `make build-module auth`).
-   `make build-all`: Compila todos los binarios (servidor + todos los módulos).
-   `make clean`: Elimina todos los artefactos de build (directorio `bin/`).

### Docker

-   `make docker-build`: Construye la imagen Docker del servidor (`modulith-server:latest`).
-   `make docker-build-module MODULE_NAME`: Construye la imagen Docker de un módulo específico (ej: `make docker-build-module auth`).

### Calidad de Código

-   `make lint`: Ejecuta el linter estricto (**OBLIGATORIO** después de cambios en `.go`).
-   `make test`: Ejecuta todas las pruebas unitarias.
-   `make test-unit`: Ejecuta tests unitarios con mocks (rápidos, sin DB).
-   `make test-coverage`: Ejecuta pruebas y genera reporte HTML de cobertura.
-   `make coverage-report`: Muestra reporte detallado de cobertura en terminal.
-   `make coverage-html`: Abre reporte de cobertura en el navegador.
-   `make generate-mocks`: Genera mocks de interfaces para testing.
-   `make install-mocks`: Instala gomock para generación de mocks.

### Desarrollo

-   `make dev`: Ejecuta el servidor monolito con hot reload.
-   `make dev-module MODULE_NAME`: Ejecuta un módulo específico con hot reload (ej: `make dev-module auth`).
-   `make new-module MODULE_NAME`: Crea el boilerplate para un nuevo módulo funcional con configuración automática (genera estructura + `.air.{MODULE_NAME}.toml`).

### Base de Datos

-   `make docker-up`: Levanta la infraestructura (PostgreSQL) con Docker Compose.
-   `make docker-down`: Detiene los contenedores de Docker.
-   `make migrate-up` / `make migrate`: Ejecuta las migraciones de todos los módulos (el modulith las descubre automáticamente).
-   `make migrate-down MODULE=auth`: Revierte la última migración de un módulo específico.
-   `make migrate-create MODULE=auth NAME=add_users`: Crea una nueva migración para un módulo específico.
-   `make db-down`: ⚠️ Borra todas las tablas de la base de datos (destructivo).
-   `make db-reset`: ⚠️ Borra todo y ejecuta todas las migraciones (equivalente a `db-down` + `migrate-up`).

**Nota:** Las migraciones se ejecutan automáticamente cuando inicias el servidor. El modulith descubre y aplica las migraciones de todos los módulos registrados.

### GraphQL (Opcional)

-   `make add-graphql`: Agrega soporte GraphQL opcional usando gqlgen (solo si lo necesitas).
-   `make graphql-generate`: Genera código GraphQL desde el schema.
-   `make graphql-validate`: Valida el schema GraphQL.

### ⚠️ Workflow de Calidad

**Después de modificar archivos `.go`:**

1. Ejecuta `make lint` y corrige **todos** los errores (0 issues).
2. Ejecuta `make test` para verificar que no rompiste nada.
3. **NUNCA** modifiques `.golangci.yaml` para ignorar errores - implementa fixes apropiados.

---

Creado con ❤️ para desarrolladores que buscan excelencia operativa.
