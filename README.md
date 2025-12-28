# Go Modulith Template 🚀

Este es un template profesional para construir aplicaciones en Go siguiendo el patrón **Modulith**. Está diseñado para ser escalable, sostenible y fácil de mantener, permitiendo evolucionar de un monolito a microservicios sin fricción.

## ✨ Características Principales

- 🏗️ **Arquitectura Modular**: Código organizado por dominios con desacoplamiento mediante eventos internos.
- 🔐 **gRPC & Protobuf**: Comunicación tipada y eficiente con generación automática vía `buf`.
- 🗄️ **SQLC & Migraciones**: Acceso a datos Type-safe y gestión de esquemas con `golang-migrate`.
- 🔄 **Hot Reload**: Desarrollo fluido con **Air** (`make dev`).
- 🛡️ **Observabilidad**: Integración nativa con OpenTelemetry (Tracing & Metrics), Prometheus y Health Checks.
- ⛴️ **Cloud Ready**: Dockerfile multi-stage y Helm Charts para Kubernetes.
- 🌍 **IaC con OpenTofu**: Infraestructura reproducible gestionada con OpenTofu y Terragrunt.
- 🤖 **CI/CD**: Pipelines de GitHub Actions para validación automática.

## 🛠️ Requisitos Previos

- Go 1.23+
- Docker & Docker Compose
- Herramientas de desarrollo:
  - `sqlc`
  - `buf`
  - `migrate`
  - `air`
  - `golangci-lint`

## 🚀 Inicio Rápido

### 1. Instalar dependencias
```bash
make install-deps
```

### 2. Levantar Infraestructura (DB)
```bash
make docker-up
```

### 3. Ejecutar en Desarrollo (Hot Reload)
```bash
make dev
```

## 📖 Documentación

Para una comprensión profunda de la arquitectura y los flujos de trabajo, consulta la [Guía de Arquitectura](docs/MODULITH_ARCHITECTURE.md).

## 🛠️ Comandos Útiles (Makefile)

- `make proto`: Genera código gRPC desde archivos `.proto`.
- `make sqlc`: Genera código Type-safe para queries SQL.
- `make lint`: Ejecuta el linter estricto.
- `make test-coverage`: Ejecuta pruebas y genera reporte de cobertura.
- `make new-module MODULE_NAME=[nombre]`: Crea el boilerplate para un nuevo módulo funcional con configuración automática.

---
Creado con ❤️ para desarrolladores que buscan excelencia operativa.
