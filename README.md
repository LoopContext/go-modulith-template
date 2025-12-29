# Go Modulith Template 🚀

Este es un template profesional para construir aplicaciones en Go siguiendo el patrón **Modulith**. Está diseñado para ser escalable, sostenible y fácil de mantener, permitiendo evolucionar de un monolito a microservicios sin fricción.

## ✨ Características Principales

- 🏗️ **Arquitectura Modular**: Código organizado por dominios con desacoplamiento mediante eventos internos.
- 🔐 **gRPC & Protobuf**: Comunicación tipada y eficiente con generación automática vía `buf`.
- 🗄️ **SQLC & Migraciones**: Acceso a datos Type-safe y gestión de esquemas con `golang-migrate`.
- ⚙️ **Configuración Flexible**: Sistema de configuración con jerarquía de precedencia (YAML > .env > system ENV vars > defaults) y logging de fuentes.
- 🔄 **Hot Reload**: Desarrollo fluido con **Air** que monitorea cambios en código, configuración (`.env`, YAML) y recursos.
- 🛡️ **Observabilidad**: Integración nativa con OpenTelemetry (Tracing & Metrics), Prometheus y Health Checks con manejo de contextos.
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

### 3. Configurar (Opcional)
El proyecto soporta múltiples fuentes de configuración con precedencia clara:
- **YAML** (`configs/server.yaml`): Mayor prioridad, ideal para configuraciones por entorno
- **`.env`**: Sobrescribe variables del sistema
- **Variables de entorno del sistema**: Valores base
- **Defaults**: Valores hardcodeados en `config.go`

Al iniciar, verás un log mostrando la fuente de cada variable de configuración.

### 4. Ejecutar en Desarrollo (Hot Reload)
```bash
make dev
```

> 💡 **Tip**: Air monitorea automáticamente cambios en `.go`, `.yaml`, `.env`, `.proto`, `.sql` y archivos de configuración, reiniciando el servidor instantáneamente.

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
