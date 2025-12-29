# Go Modulith Template 🚀

Este es un template profesional para construir aplicaciones en Go siguiendo el patrón **Modulith**. Está diseñado para ser escalable, sostenible y fácil de mantener, permitiendo evolucionar de un monolito a microservicios sin fricción.

## ✨ Características Principales

-   🏗️ **Arquitectura Modular**: Código organizado por dominios con desacoplamiento mediante eventos internos.
-   🔐 **gRPC & Protobuf**: Comunicación tipada y eficiente con generación automática vía `buf`.
-   🗄️ **SQLC & Migraciones**: Acceso a datos Type-safe y gestión de esquemas con `golang-migrate`.
-   ⚙️ **Configuración Flexible**: Sistema de configuración con jerarquía de precedencia (YAML > .env > system ENV vars > defaults) y logging de fuentes.
-   🔄 **Hot Reload**: Desarrollo fluido con **Air** que monitorea cambios en código, configuración (`.env`, YAML) y recursos.
-   🛡️ **Observabilidad**: Integración nativa con OpenTelemetry (Tracing & Metrics), Prometheus y Health Checks con manejo de contextos.
-   ⛴️ **Cloud Ready**: Dockerfile multi-stage y Helm Charts flexibles para Kubernetes (soporta monolito y módulos independientes).
-   🌍 **IaC con OpenTofu**: Infraestructura base reproducible (VPC, EKS, RDS) gestionada con OpenTofu y Terragrunt.
-   🤖 **CI/CD**: Pipelines de GitHub Actions para validación automática.

## 🛠️ Requisitos Previos

-   Go 1.23+
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

Al iniciar, verás un log mostrando la fuente de cada variable de configuración.

### 4. Ejecutar en Desarrollo (Hot Reload)

```bash
make dev
```

Para ejecutar un módulo específico con hot reload:

```bash
make dev-module auth
```

> 💡 **Tip**: Air monitorea automáticamente cambios en `.go`, `.yaml`, `.env`, `.proto`, `.sql` y archivos de configuración, reiniciando el servidor instantáneamente.

## 📖 Documentación

-   **[Guía de Arquitectura](docs/MODULITH_ARCHITECTURE.md)** - Comprensión profunda de la arquitectura y flujos de trabajo
-   **[Deployment Guide](deployment/README.md)** - Guía completa de despliegue en Kubernetes con IaC
-   **[Helm Chart Documentation](deployment/helm/modulith/README.md)** - Documentación detallada del Helm chart

## 🛠️ Comandos Útiles (Makefile)

### Generación de Código
-   `make proto`: Genera código gRPC desde archivos `.proto`.
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
-   `make test-coverage`: Ejecuta pruebas y genera reporte de cobertura.

### Desarrollo
-   `make dev`: Ejecuta el servidor monolito con hot reload.
-   `make dev-module MODULE_NAME`: Ejecuta un módulo específico con hot reload (ej: `make dev-module auth`).
-   `make new-module MODULE_NAME`: Crea el boilerplate para un nuevo módulo funcional con configuración automática (genera estructura + `.air.{MODULE_NAME}.toml`).

### ⚠️ Workflow de Calidad

**Después de modificar archivos `.go`:**

1. Ejecuta `make lint` y corrige **todos** los errores (0 issues).
2. Ejecuta `make test` para verificar que no rompiste nada.
3. **NUNCA** modifiques `.golangci.yaml` para ignorar errores - implementa fixes apropiados.

---

Creado con ❤️ para desarrolladores que buscan excelencia operativa.
