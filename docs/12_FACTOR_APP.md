# 12-Factor App Compliance Guide

Este documento detalla cómo el Go Modulith Template cumple con los principios de la [metodología 12-factor app](https://12factor.net/), garantizando que la aplicación sea escalable, mantenible y lista para producción.

## Resumen de Compliance

| Factor                 | Estado | Documentación                                             |
| ---------------------- | ------ | --------------------------------------------------------- |
| I. Codebase            | ✅     | Un código base, múltiples deploys                         |
| II. Dependencies       | ✅     | Dependencias explícitas en `go.mod`                       |
| III. Config            | ✅     | Configuración en environment                              |
| IV. Backing Services   | ✅     | Servicios tratados como recursos adjuntos                 |
| V. Build, Release, Run | ✅     | Separación estricta de etapas                             |
| VI. Processes          | ✅     | Procesos stateless                                        |
| VII. Port Binding      | ✅     | Exporta servicios mediante binding a puerto               |
| VIII. Concurrency      | ✅     | Escala mediante proceso modelo                            |
| IX. Disposability      | ✅     | Procesos con inicio rápido y shutdown graceful            |
| X. Dev/Prod Parity     | ✅     | Mantener desarrollo y producción lo más similares posible |
| XI. Logs               | ✅     | Logs como streams de eventos                              |
| XII. Admin Processes   | ✅     | Tareas administrativas como procesos one-off              |

---

## I. Codebase

**Principio:** Un código base rastreado en control de revisiones, muchos deploys.

**Implementación:**

-   ✅ Código fuente en un solo repositorio Git
-   ✅ Múltiples deploys (dev, staging, prod) desde el mismo código
-   ✅ Módulos compilados como binarios separados (opcional)
-   ✅ Sin código duplicado entre deploys

**Estructura:**

```
go-modulith-template/
├── cmd/
│   ├── server/    # Monolito
│   ├── auth/      # Microservicio (opcional)
│   └── worker/    # Worker process
├── modules/       # Módulos de negocio
└── internal/      # Infraestructura compartida
```

**Ver también:** `docs/MODULITH_ARCHITECTURE.md` (Sección 2: Estructura del Proyecto)

---

## II. Dependencies

**Principio:** Declarar y aislar explícitamente las dependencias.

**Implementación:**

-   ✅ Dependencias explícitas en `go.mod`
-   ✅ Versiones fijas en `go.sum`
-   ✅ Sin dependencias del sistema operativo
-   ✅ Build reproducible

**Gestión:**

```bash
# Agregar dependencia
go get github.com/example/package@v1.2.3

# Actualizar dependencias
go get -u ./...

# Verificar dependencias
go mod verify
```

**Ver también:** `go.mod`, `go.sum`

---

## III. Config

**Principio:** Almacenar configuración en el environment.

**Implementación:**

-   ✅ Configuración en variables de entorno
-   ✅ Soporte para `.env` (desarrollo)
-   ✅ Soporte para YAML (configuraciones por entorno)
-   ✅ Prioridad: `PORT > YAML > .env > system ENV vars > defaults`

**Variables de entorno:**

```bash
# Requeridas en producción
DB_DSN=postgres://user:pass@host:5432/db
JWT_SECRET=your-secret-key-at-least-32-bytes-long

# Opcionales
ENV=prod
LOG_LEVEL=info
HTTP_PORT=8080
PORT=8080  # 12-factor standard (toma precedencia sobre HTTP_PORT)
```

**Ver también:** `docs/ENVIRONMENT.md`

---

## IV. Backing Services

**Principio:** Tratar servicios de respaldo como recursos adjuntos.

**Implementación:**

-   ✅ PostgreSQL como servicio adjunto
-   ✅ Valkey como servicio adjunto (opcional)
-   ✅ Configuración via `DB_DSN` (puede cambiar sin cambiar código)
-   ✅ Sin diferencia entre servicios locales y remotos

**Ejemplo:**

```go
// Mismo código funciona con DB local o remota
db, err := sql.Open("pgx", cfg.DBDSN)  // DB_DSN desde env
```

**Ver también:** `docker-compose.yaml`, `docs/ENVIRONMENT.md`

---

## V. Build, Release, Run

**Principio:** Separar estrictamente las etapas de build y run.

**Implementación:**

-   ✅ **Build:** Compilación y generación de artefactos
-   ✅ **Release:** Combinación de build + configuración
-   ✅ **Run:** Ejecución de la aplicación

**Etapas:**

1. **Build:**

    ```bash
    just build              # Compila binario
    just docker-build       # Crea imagen Docker
    ```

2. **Release:**

    ```bash
    # Opción 1: Migraciones en startup (recomendado para modulith)
    ./bin/server  # Ejecuta migraciones, luego inicia

    # Opción 2: Migraciones como job separado (producción)
    kubectl apply -f migration-job.yaml
    ```

3. **Run:**
    ```bash
    ./bin/server           # Ejecuta aplicación
    docker run modulith-server:latest
    helm install modulith-server ./deployment/helm/modulith
    ```

**Ver también:** `docs/DEPLOYMENT_SYNC.md` (Sección: Build, Release, Run)

---

## VI. Processes

**Principio:** Ejecutar la aplicación como uno o más procesos stateless.

**Implementación:**

-   ✅ Procesos completamente stateless
-   ✅ Sin estado en sistema de archivos
-   ✅ Estado persistente en servicios externos (DB, Valkey)
-   ✅ Cualquier instancia puede manejar cualquier request

**Verificación:**

-   ✅ No se escriben archivos temporales
-   ✅ Sesiones almacenadas en PostgreSQL
-   ✅ WebSocket state en memoria (requiere sticky sessions para escalado)

**Ver también:** `docs/MODULITH_ARCHITECTURE.md` (Sección 20: Stateless Processes)

---

## VII. Port Binding

**Principio:** Exportar servicios mediante binding a puerto.

**Implementación:**

-   ✅ Aplicación se enlaza a puerto especificado
-   ✅ Soporte para `PORT` (estándar 12-factor)
-   ✅ Soporte para `HTTP_PORT` (explícito)
-   ✅ Prioridad: `PORT > HTTP_PORT > default`

**Configuración:**

```bash
# Estándar 12-factor (Heroku, Cloud Run, Railway, etc.)
PORT=8080

# O explícito
HTTP_PORT=8080
GRPC_PORT=9000  # Default in configs/server.yaml (can be configured)
```

**Ver también:** `internal/config/config.go`, `docs/ENVIRONMENT.md`

---

## VIII. Concurrency

**Principio:** Escalar mediante el proceso modelo.

**Implementación:**

-   ✅ Escalado horizontal mediante múltiples procesos
-   ✅ Web process: Maneja HTTP/gRPC requests
-   ✅ Worker process: Procesa eventos asíncronos
-   ✅ HPA (Horizontal Pod Autoscaler) configurado

**Escalado:**

```yaml
# deployment/helm/modulith/values-server.yaml
autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
```

**Ver también:** `docs/MODULITH_ARCHITECTURE.md` (Sección 21: Concurrency)

---

## IX. Disposability

**Principio:** Maximizar robustez con inicio rápido y shutdown graceful.

**Implementación:**

-   ✅ Inicio rápido (< 10 segundos típicamente)
-   ✅ Shutdown graceful con timeout configurable
-   ✅ Manejo de señales (SIGTERM, SIGINT)
-   ✅ Cierre de conexiones WebSocket
-   ✅ Flush de telemetría antes de terminar

**Configuración:**

```yaml
# configs/server.yaml
shutdown_timeout: 30s # Tiempo máximo para shutdown graceful
```

**Ver también:** `cmd/server/setup/server.go` (función `ShutdownServers`)

---

## X. Dev/Prod Parity

**Principio:** Mantener desarrollo, staging y producción lo más similares posible.

**Implementación:**

-   ✅ Mismas versiones de dependencias (PostgreSQL 18, Valkey 7)
-   ✅ Mismas herramientas (golang-migrate, sqlc, buf)
-   ✅ Mismo código base
-   ✅ Diferencias solo en configuración

**Verificación:**

```yaml
# docker-compose.yaml (desarrollo)
db:
    image: postgres:18-alpine # ✅ Misma versión que producción

# deployment/helm/modulith/values-server.yaml (producción)
# Usar misma versión de PostgreSQL
```

**Ver también:** `docs/ENVIRONMENT.md` (Sección: Dev/Prod Parity)

---

## XI. Logs

**Principio:** Tratar logs como streams de eventos.

**Implementación:**

-   ✅ Logs a stdout/stderr
-   ✅ Formato estructurado (JSON en prod, text en dev)
-   ✅ Integración con trace_id/span_id
-   ✅ Sin escritura a archivos
-   ✅ Captura por orquestador (Kubernetes, Docker, etc.)

**Configuración:**

```yaml
# configs/server.yaml
log_level: info # debug, info, warn, error
```

**Formato:**

-   **Dev:** Text format (legible)
-   **Prod:** JSON format (parseable)

**Ver también:** `cmd/server/observability/logger.go` (función `InitLogger`)

---

## XII. Admin Processes

**Principio:** Ejecutar tareas administrativas como procesos one-off.

**Implementación:**

-   ✅ Tareas administrativas como comandos separados
-   ✅ Ejecución como procesos one-off
-   ✅ Mismo código base y configuración
-   ✅ **Admin Task Runner** (`internal/admin/runner.go`): Framework para registrar y ejecutar tareas administrativas
-   ✅ **Seed Data System** (`internal/migration/seeder.go`): Descubrimiento y ejecución automática de seed data para todos los módulos
-   ✅ Soporte de subcomandos: `migrate`, `seed`, y `admin`

**Ejemplos:**

```bash
# Ejecutar migraciones
go run cmd/server/main.go migrate
# o: just migrate

# Ejecutar seed data
go run cmd/server/main.go seed
# o: just seed

# Ejecutar tarea administrativa
./bin/server admin cleanup-sessions
./bin/server admin cleanup-magic-codes

# O con just
just admin TASK=cleanup-sessions
```

**Ver también:** `cmd/server/commands/admin.go` (función `RunAdminCommand`), `internal/admin/runner.go`, `internal/migration/seeder.go`

---

## Mejoras Implementadas

### Admin Processes Infrastructure (Factor XII)

**Estado:** ✅ Completo

**Componentes agregados:**

-   **Admin Task Runner** (`internal/admin/runner.go`): Framework para registrar y ejecutar tareas administrativas one-off
-   **Seed Data System** (`internal/migration/seeder.go`): Descubrimiento y ejecución automática de seed data para todos los módulos
-   **Subcomandos**: El binario principal soporta `migrate`, `seed`, y `admin`
-   **Interfaz de Módulo**: Método `SeedPath()` agregado para descubrimiento de seed data

**Archivos creados:**

-   `internal/admin/runner.go` - Admin task runner
-   `internal/admin/runner_test.go` - Tests para admin runner
-   `internal/migration/seeder.go` - Sistema de ejecución de seed data
-   `internal/migration/seeder_test.go` - Tests para seeder

### Module Lifecycle Hooks

**Estado:** ✅ Completo

**Implementación:**

-   `OnStartAll()` se llama después de la inicialización de módulos, antes de servir
-   `OnStopAll()` se llama durante shutdown graceful con timeout apropiado
-   Permite a los módulos inicializar recursos en startup (connection pools, background workers, etc.)
-   Permite limpieza graceful de recursos en shutdown

**Impacto:**

-   Los módulos pueden inicializar recursos correctamente en startup
-   Los módulos pueden limpiar recursos gracefulmente en shutdown
-   El shutdown respeta el timeout configurado

### Timeout Configurations

**Estado:** ✅ Completo

**Timeouts configurados:**

-   **Read Timeout**: Timeout de lectura del servidor HTTP (default: 5s)
-   **Write Timeout**: Timeout de escritura del servidor HTTP (default: 10s)
-   **Shutdown Timeout**: Timeout para shutdown graceful (default: 30s)
-   Todos los timeouts son configurables via YAML, .env, o variables de entorno

**Configuración:**

```yaml
# configs/server.yaml
read_timeout: 5s       # HTTP server read timeout
write_timeout: 10s     # HTTP server write timeout
shutdown_timeout: 30s  # Graceful shutdown timeout
```

### Health Check Aggregation

**Estado:** ✅ Completo

**Implementación:**

-   El endpoint `/readyz` ahora verifica la salud de los módulos via `HealthCheckAll()`
-   Los módulos que implementan la interfaz `ModuleHealth` son verificados automáticamente
-   Proporciona mensajes de error detallados cuando los health checks fallan
-   La verificación de conectividad a la base de datos permanece como validación secundaria

**Impacto:**

-   Los readiness probes de Kubernetes ahora validan la salud completa de la aplicación
-   Los módulos pueden implementar health checks personalizados (conectividad de cache, disponibilidad de API externa, etc.)
-   Mejor visibilidad del estado de la aplicación

### Release Workflow

**Estado:** ✅ Completo

**Workflow de Release** (`.github/workflows/release.yaml`):

-   Se activa en tags de versión (v*)
-   Compila todos los binarios (server + módulos)
-   Genera changelog desde commits de git
-   Crea releases de GitHub con binarios
-   Construye y publica imágenes Docker a GHCR
-   Soporta semantic versioning
-   Usa Docker build cache para builds más rápidos

**CI Workflow mejorado** (`.github/workflows/ci.yaml`):

-   Security scanning con gosec
-   Container scanning con Trivy
-   Coverage reporting a Codecov
-   Resultados subidos a la pestaña Security de GitHub

**Uso:**

```bash
# Crear y pushear un tag de release
git tag v1.0.0
git push origin v1.0.0

# El workflow automáticamente:
# 1. Compila binarios
# 2. Crea GitHub release
# 3. Construye y publica imágenes Docker
```

### Testcontainers Integration

**Estado:** ✅ Completo

**Componentes agregados:**

-   **Testcontainers Helper** (`internal/testutil/testcontainers.go`):
    -   Wrapper de contenedor PostgreSQL
    -   Gestión automática del ciclo de vida del contenedor
    -   Generación de connection string
    -   Helper de conexión a base de datos

-   **Ejemplo de Integration Test** (`modules/auth/internal/repository/repository_integration_test.go`):
    -   Demuestra testing con base de datos real
    -   Creación y limpieza de esquema
    -   Testing de integración de repository

**Uso:**

```bash
# Ejecutar tests de integración (requiere Docker)
just test-integration

# Ejecutar todos los tests
just test-all

# Saltar tests de integración en CI
go test -short ./...
```

---

## 🎯 Mejoras en Developer Experience

### Lo que los Desarrolladores Obtienen

1. **Zero Boilerplate para Tareas Comunes**:
   -   Las migraciones se ejecutan automáticamente
   -   Seed data con `just seed`
   -   Admin tasks via interfaz simple

2. **Estructura de Módulo Consistente**:
   -   `just new-module name` crea todo
   -   Directorio de seed data incluido
   -   Lifecycle hooks disponibles

3. **Defaults Listos para Producción**:
   -   Timeouts configurados
   -   Health checks agregados
   -   Security scanning en CI
   -   Release automation

4. **Testing Simplificado**:
   -   Testcontainers para tests con DB real
   -   Ejemplos de integration tests
   -   Unit test mocking con gomock

5. **Flexibilidad de Despliegue**:
   -   Monolito o microservicios
   -   Ejemplo de staging incluido
   -   Release workflow automatizado

---

## 📝 Guía de Migración para Módulos Existentes

Si tienes módulos existentes, actualízalos para soportar las nuevas características:

### 1. Agregar Soporte de Seed Data

```go
// En modules/yourmodule/module.go
func (m *Module) SeedPath() string {
    return "modules/yourmodule/resources/db/seed"
}
```

Crear directorio de seed y agregar archivos SQL:

```bash
mkdir -p modules/yourmodule/resources/db/seed
# Agregar 001_initial_data.sql, etc.
```

### 2. Agregar Health Checks (Opcional)

```go
// Implementar interfaz ModuleHealth
func (m *Module) HealthCheck(ctx context.Context) error {
    // Verificar salud específica del módulo
    return nil
}
```

### 3. Agregar Lifecycle Hooks (Opcional)

```go
// Implementar interfaz ModuleLifecycle
func (m *Module) OnStart(ctx context.Context) error {
    // Inicializar recursos
    return nil
}

func (m *Module) OnStop(ctx context.Context) error {
    // Limpiar recursos
    return nil
}
```

---

## Checklist de Compliance

Antes de desplegar a producción, verificar:

### Configuración

-   [ ] Variables de entorno configuradas correctamente
-   [ ] `PORT` o `HTTP_PORT` configurado
-   [ ] `DB_DSN` apunta a base de datos de producción
-   [ ] `JWT_SECRET` es fuerte (32+ bytes)

### Procesos

-   [ ] Procesos son stateless
-   [ ] No hay escritura a sistema de archivos
-   [ ] Sesiones almacenadas en DB compartida
-   [ ] Health checks configurados

### Escalado

-   [ ] HPA configurado (si aplica)
-   [ ] Connection pool ajustado según número de instancias
-   [ ] Sticky sessions configuradas para WebSocket (si aplica)

### Observabilidad

-   [ ] Logs estructurados funcionando
-   [ ] Métricas expuestas en `/metrics`
-   [ ] Tracing configurado (si aplica)
-   [ ] Health checks responden correctamente

### Dev/Prod Parity

-   [ ] Mismas versiones de DB/Valkey
-   [ ] Mismas herramientas
-   [ ] Tests ejecutados en ambiente similar

---

## Referencias

-   [12-Factor App Methodology](https://12factor.net/)
-   [Go Modulith Architecture](docs/MODULITH_ARCHITECTURE.md)
-   [Environment Variables](docs/ENVIRONMENT.md)
-   [Deployment Guide](docs/DEPLOYMENT_SYNC.md)

---

---

## 🚀 Próximos Pasos

El template está ahora listo para producción con compliance completo de 12-factor. Áreas de enfoque:

1. **Lógica de Negocio**: Los desarrolladores pueden enfocarse puramente en reglas de negocio
2. **Desarrollo de Módulos**: Usar `just new-module` para scaffold de nuevas características
3. **Testing**: Escribir integration tests usando testcontainers
4. **Despliegue**: Usar Helm charts para despliegue en Kubernetes
5. **Monitoreo**: Aprovechar telemetría existente (métricas, traces, logs)

---

**Last updated:** January 2026
**Maintained by:** Go Modulith Template Team
