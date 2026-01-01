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
-   ✅ Redis como servicio adjunto (opcional)
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
    make build              # Compila binario
    make docker-build       # Crea imagen Docker
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
-   ✅ Estado persistente en servicios externos (DB, Redis)
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
GRPC_PORT=9050
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

**Ver también:** `cmd/server/main.go` (función `shutdownServers`)

---

## X. Dev/Prod Parity

**Principio:** Mantener desarrollo, staging y producción lo más similares posible.

**Implementación:**

-   ✅ Mismas versiones de dependencias (PostgreSQL 18, Redis 7)
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

**Ver también:** `cmd/server/main.go` (función `initLogger`)

---

## XII. Admin Processes

**Principio:** Ejecutar tareas administrativas como procesos one-off.

**Implementación:**

-   ✅ Tareas administrativas como comandos separados
-   ✅ Ejecución como procesos one-off
-   ✅ Mismo código base y configuración

**Ejemplos:**

```bash
# Ejecutar tarea administrativa
./bin/server admin cleanup-sessions
./bin/server admin cleanup-magic-codes

# O con make
make admin TASK=cleanup-sessions
```

**Ver también:** `internal/admin/`, `cmd/server/main.go` (función `runAdminCommand`)

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

-   [ ] Mismas versiones de DB/Redis
-   [ ] Mismas herramientas
-   [ ] Tests ejecutados en ambiente similar

---

## Referencias

-   [12-Factor App Methodology](https://12factor.net/)
-   [Go Modulith Architecture](docs/MODULITH_ARCHITECTURE.md)
-   [Environment Variables](docs/ENVIRONMENT.md)
-   [Deployment Guide](docs/DEPLOYMENT_SYNC.md)

---

**Última actualización:** Enero 2026
**Mantenido por:** Go Modulith Template Team
