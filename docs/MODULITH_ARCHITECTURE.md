# Guía de Arquitectura e Implementación: Go Modulith

Esta documentación define el estándar arquitectónico y de implementación para proyectos nuevos ("greenfield"). Establece las directrices para construir un **Monolito Modular** robusto, escalable y mantenible, utilizando un stack tecnológico moderno y tipado estrictamente.

## 1. Stack Tecnológico Definido

Todas las implementaciones deben adherirse estrictamente a las siguientes tecnologías:

*   **Lenguaje:** Go 1.23+.
*   **Arquitectura:** Monolito Modular.
*   **Comunicación/Contrato:** gRPC y Protocol Buffers (Single Source of Truth).
*   **API Externa:**
    *   **gRPC:** Protocolo principal de comunicación backend-backend.
    *   **REST/HTTP:** Expuesto automáticamente vía `grpc-gateway` (Proxy inverso).
    *   **Documentación:** Swagger UI (OpenAPIv2) disponible en `/swagger-ui/` (Solo Dev).
*   **Persistencia:** SQLC (Type-safe SQL).
*   **Base de Datos:** PostgreSQL (con migraciones versionadas).
*   **Infraestructura Local:** Docker Compose.
*   **Migraciones:** `golang-migrate` (Gestión de esquema).
*   **Observabilidad:**
    *   **Logs:** Structured Logging (`log/slog`) con formato JSON.
    *   **Métricas:** OpenTelemetry (OTel) exponiendo métricas en formato Prometheus.
    *   **Tracing:** OpenTelemetry (Context propagation).

## 2. Estructura del Proyecto (Project Layout)

La organización de carpetas es crítica para mantener la modularidad. Cada módulo debe ser autocontenido.

```text
proyecto/
├── cmd/
│   ├── server/             # Entrypoint Monolito (main.go)
│   └── auth-svc/           # Entrypoint Microservicio (main.go)
├── configs/                # Configuraciones YAML por aplicación
│   ├── server.yaml         # Configuración del monolito
│   └── auth-svc.yaml       # Configuración del microservicio
├── internal/
│   └── config/             # Cargador central de configuración (YAML + Env)
├── scripts/                # Scripts de automatización (scaffolding)
├── proto/                  # Definiciones centralizadas de API
│   ├── google/             # Dependencias de Google (API, Protobuf)
│   └── [modulo]/           # Protos específicos del módulo (v1)
├── modules/                # Módulos de Negocio
│   └── [nombre_modulo]/
│       ├── internal/
│       │   ├── service/    # Implementación gRPC Server (Lógica de Negocio)
│       │   ├── repository/ # Adaptadores de acceso a datos (Interfaz)
│       │   ├── models/     # Modelos de dominio
│       │   └── db/
│       │       ├── query/  # Archivos .sql (Queries handwritten)
│       │       └── store/  # Código Go generado por SQLC
│       └── resources/
│           └── db/
│               └── migration/ # Scripts SQL DDL (Schema Versioning)
├── sqlc.yaml               # Configuración global de generación SQL
├── buf.yaml                # Configuración de Buf
└── go.mod
```

## 3. Reglas de Aislamiento de Módulos (Insulation)

El éxito de un modulith depende de la disciplina. Un módulo podrido infecta a los demás.

*   **Importaciones:** Un módulo `A` **NUNCA** puede importar nada de la carpeta `internal/` de un módulo `B`.
*   **Comunicación:** La única forma legítima de comunicación entre módulos es:
    1.  **gRPC (in-process):** Llamando a través del cliente gRPC generado (usando el gateway interno). Al ser *in-process*, **no hay saltos de red**; es una llamada a función directa a través del stack de gRPC, garantizando performance y contratos fuertes.
    2.  **Eventos:** Publicación/Suscripción (si se implementa en el futuro).
*   **Datos:** Prohibido compartir repositorios, queries de SQLC o modelos de base de datos entre módulos. Cada módulo es dueño absoluto de su esquema.
*   **DTOs:** Los mensajes de Protobuf son el lenguaje común. No se deben filtrar tipos de `store/` o `repository/` hacia afuera del propio módulo.

## 4. Dominio y Modelos

Para evitar debates infinitos, establecemos el siguiente estándar:

*   **Domain Ownership:** La lógica de negocio reside en la capa de `service/`.
*   **Modelos Simples:** No utilizamos entidades ricas (DDD complejo) a menos que sea estrictamente necesario.
*   **Flujo:** `store` (DB) -> `repository` (Adapter) -> `service` (Domain/Business) -> `proto` (DTO).
*   **Repository:** Devuelve structs simples del `store` o modelos de dominio básicos en `internal/models/`. No hay lógica de negocio en el repositorio.

## 5. Identificadores Únicos (TypeID)

Para mejorar la trazabilidad, depuración y ordenabilidad de los datos, adoptamos el estándar de **Identificadores Prefijados y Ordenables por Tiempo** (estilo Stripe).

*   **Estándar:** Utilizaremos **TypeID** (`github.com/jetpack-io/typeid-go`), que combina un prefijo legible con un **UUIDv7**.
*   **Formato:** `prefix_01h455vb4pex5vsknk084sn02q`.
    *   **Prefix:** Indica el tipo de entidad (ej. `user`, `role`, `org`). Máximo 8 caracteres.
    *   **Suffix:** Un UUIDv7 codificado en Base32 (Crockford), lo que lo hace lexicográficamente ordenable.
*   **Ventajas:**
    *   **Sortable:** La ordenabilidad por tiempo permite que las bases de datos (PostgreSQL) indexen de forma más eficiente que con UUIDs aleatorios.
    *   **Contextual:** Al ver un ID en un log (`user_...`), sabemos inmediatamente a qué entidad pertenece.
    *   **Seguridad:** Son globalmente únicos y difíciles de predecir.
*   **Ownership:** Los TypeIDs se generan **únicamente** en la capa de `service`. El repositorio y la base de datos son pasivos y nunca generan identificadores.
*   **Semántica:** Los prefijos son puramente informativos para humanos y trazabilidad; no deben usarse para lógica de autorización o acceso cross-domain.

> [!NOTE]
> En este documento, por simplicidad, los TypeIDs se representan y almacenan como `VARCHAR` completos. En implementaciones de alto rendimiento, se podría almacenar solo el sufijo binario como `UUID` y reconstruir el prefijo en la aplicación.

## 6. Manejo de Errores gRPC

No se deben retornar errores crudos de la base de datos o del sistema al cliente.

*   **Responsabilidad:** El `service` es el único responsable de mapear errores de Go a códigos de estado gRPC (`google.golang.org/grpc/status`).
*   **Helper Recomendado:** Usar `status.Error(codes.Code, message)` para respuestas inmediatas.
*   **Transparencia:** Los errores internos se loaguean con detalle pero se responden al cliente como `codes.Internal` por seguridad.

## 7. Transacciones

Las transacciones deben ser controladas por la capa de negocio (`service`) pero ejecutadas por el repositorio.

*   **Patrón WithTx:** El repositorio debe ofrecer una forma de ejecutar múltiples operaciones en una transacción atómica.
*   **Ejemplo Conceptual:**
```go
err := r.WithTx(ctx, func(txRepo Repository) error {
    // Estas operaciones ocurren dentro de la misma transacción
    if err := txRepo.CreateUser(ctx, ...); err != nil { return err }
    if err := txRepo.AssignRole(ctx, ...); err != nil { return err }
    return nil
})
```

## 8. Validación

Establecemos una frontera clara para evitar validaciones duplicadas:

*   **Estructural (Proto):** Formato, longitud, obligatorios, rangos. Se valida preferiblemente en el interceptor o al inicio del `service` usando la estructura del proto.
*   **Negocio (Service):** Existencia en BD, permisos complejos, reglas de estado, lógica temporal.

## 9. Seguridad: Autenticación y Autorización

*   **Validación de Tokens:** Se realiza centralizadamente en un **gRPC Interceptor** global.
*   **Contexto:** El interceptor extrae el `user_id` y `role` del token y los inyecta en el `context.Context` para que estén disponibles en toda la cadena de llamada.
*   **RBAC:** El chequeo de permisos (`users:read`, etc.) ocurre en la capa de `service` basándose en el rol/permisos inyectados en el contexto.

## 10. Configuración y Entorno (Environment)

La jerarquía de configuración favorece la flexibilidad tanto en desarrollo como en despliegues complejos de microservicios.

### Jerarquía de Carga
La aplicación carga la configuración en este orden (el último sobreescribe al anterior):
1.  **Fallback Dinámico:** Si una variable no está en el YAML ni en el Env, se usa el default de `config.go`.
2.  **Archivo YAML:** Ubicado en la carpeta `configs/` y especificado al iniciar la aplicación (ej. `configs/server.yaml`).
3.  **Variables de Entorno:** Cargadas desde el sistema o via archivos `.env` (usando `godotenv`).

### Agregar nueva configuración
1.  Añadir el campo al struct en `internal/config/config.go` con los tags `yaml` y `envconfig` (o similar).
2.  Actualizar los archivos YAML en `configs/` si el valor es específico del entorno.
3.  Inyectar el struct de configuración en la función `Initialize` del módulo correspondiente.

### Variables de Entorno Clave
Aunque residan en el YAML, estas variables son críticas para el entorno de ejecución:
*   `ENV`: `dev` o `prod`. Determina el nivel de logs y la activación de herramientas de depuración.
*   `DB_DSN`: Conexión a PostgreSQL.
*   `JWT_SECRET`: Clave secreta para tokens (Configurada vía Env en producción por seguridad).
*   `HTTP_PORT` / `GRPC_PORT`: Puertos de escucha.

## 11. Infraestructura Local (Docker)
Utilizamos Docker Compose para levantar dependencias (Base de Datos).
*   El puerto de PostgreSQL es configurable vía `DB_PORT` en el `.env` del host.
*   Comandos útiles en `Makefile`: `make docker-up`, `make docker-down`.

## 12. Observabilidad

La observabilidad es ciudadana de primera clase. No se debe desplegar código sin visibilidad.

### 12.1. Logs Estructurados
Usamos la librería estándar `log/slog` (Go 1.21+).
*   **Formato:** JSON en producción, Texto en desarrollo.
*   **Contexto:** Todo log debe incluir `trace_id` y `span_id` si existen en el contexto.
*   **Niveles:** INFO (flujo normal), ERROR (excepciones), DEBUG (solo dev).
*   **Privacidad (PII):** **NUNCA** loguear información sensible (emails, tokens, passwords).

```go
slog.InfoContext(ctx, "user created", "user_id", id) // Evitar loguear el email aquí
```

### 12.2. Métricas (OpenTelemetry)
Instrumentamos la aplicación usando el SDK de OpenTelemetry.
*   **Protocolo:** Prometheus (`/metrics`).
*   **Métricas Standard:**
    *   `http_request_duration_seconds` (Histograma).
    *   `grpc_server_handled_total` (Contador).
*   **Mapeo:** Middleware/Interceptores automáticos para gRPC y HTTP.

### 12.3. Health Checks
El sistema expone dos endpoints críticos para el orquestador (K8s):
*   **/healthz (Liveness):** Indica si el proceso está vivo. Retorna `200 OK`.
*   **/readyz (Readiness):** Indica si el servicio puede recibir tráfico (ej. validando conexión a DB).

### 12.4. Tracing (OpenTelemetry)
Implementamos trazabilidad distribuida usando el exportador OTLP.
*   **Propagación:** Los traces viajan automáticamente a través de los interceptores gRPC.
*   **Contexto:** Permite ver el camino de una petición desde el gateway hasta el repositorio.

## 13. Comunicación Asíncrona (Eventos)

Para evitar acoplamiento fuerte entre módulos, disponemos de un **Bus de Eventos** interno (`internal/events`).

*   **Patrón Pub/Sub:** Los módulos se suscriben a eventos (ej. `user.created`) sin conocer quién los emite.
*   **No Bloqueante:** La publicación de eventos ocurre en goroutines separadas para no penalizar el tiempo de respuesta gRPC/HTTP.
*   **Extensibilidad:** Facilita añadir efectos secundarios (auditoría, notificaciones) sin modificar el servicio original.

### Ejemplo de Uso (Service -> Audit)
```go
// En modules/users/internal/service/service.go
bus.Publish(ctx, events.Event{
    Name: "user.created",
    Payload: map[string]string{"id": id, "email": email},
})

// En modules/audit/module.go
eventBus.Subscribe("user.created", func(ctx context.Context, e events.Event) error {
    slog.InfoContext(ctx, "audit: logging user creation", "payload", e.Payload)
    return nil
})
```

## 14. Escalabilidad y Alta Disponibilidad

El diseño modular y el empaquetado permiten escalar el sistema de forma eficiente:

### Horizontal Pod Autoscaler (HPA)
El sistema soporta escalado automático basado en CPU/Memoria definido en el Helm Chart. Se recomienda un umbral del 80% para disparar nuevas réplicas.

### Graceful Shutdown
La aplicación maneja señales de terminación para cerrar conexiones a base de datos y terminar peticiones gRPC en curso antes de morir.

### Pod Disruption Budget (PDB)
Garantizamos un mínimo de disponibilidad durante mantenimientos del cluster Kubernetes, asegurando que siempre haya al menos una réplica operativa.

## 15. Guía de Implementación: De Cero a Producción

### Fase 1: Definición del Contrato (Protocol Buffers)

El desarrollo comienza definiendo la API. Esto garantiza que frontend y backend acuerden la estructura de datos antes de escribir código.

`proto/users/v1/users.proto`:
```protobuf
syntax = "proto3";

package users.v1;

import "google/api/annotations.proto";

service UserService {
  // Crea un nuevo usuario
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse) {
    option (google.api.http) = {
      post: "/v1/users"
      body: "*"
    };
  }
}

message CreateUserRequest {
  string username = 1;
  string email = 2;
}

message CreateUserResponse {
  string id = 1;
  string username = 2;
}
```

### Fase 2: Persistencia (Schema & SQLC)

Diseñamos la base de datos y las operaciones necesarias. SQLC se encargará de generar el código de acceso a datos.

**1. Migración (DDL):**
Creamos las migraciones utilizando `golang-migrate`.
`modules/users/resources/db/migration/000001_initial_schema.up.sql`:
```sql
CREATE TABLE users (
  id VARCHAR(64) PRIMARY KEY,
  username VARCHAR(255) NOT NULL,
  email VARCHAR(255) NOT NULL UNIQUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP -- Debe actualizarse desde la aplicación o vía Trigger
);
```

**2. Queries (SQL):**
`modules/users/internal/db/query/users.sql`:
```sql
-- name: CreateUser :exec
INSERT INTO users (id, username, email) VALUES ($1, $2, $3);

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;
```

**3. Configuración SQLC:**
`sqlc.yaml`:
```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "modules/users/internal/db/query/"
    schema: "modules/users/resources/db/migration/"
    gen:
      go:
        package: "store"
        out: "modules/users/internal/db/store"
        sql_package: "database/sql"
        emit_interface: true
        emit_json_tags: true
```

**4. Generación:**
Ejecutar `sqlc generate`. Esto crea `modules/users/internal/db/store/` con tipos seguros.

### Fase 3: Capa de Repositorio (Adapter)

Creamos una capa intermedia que abstrae `sqlc` del resto de la aplicación. El repositorio es un **esclavo** del servicio: no genera IDs ni contiene lógica.

`modules/users/internal/repository/repository.go`:
```go
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"proyecto/modules/users/internal/db/store"
)

// Repository define las operaciones de negocio sobre los datos
type Repository interface {
    CreateUser(ctx context.Context, id, username, email string) error
}

type SQLRepository struct {
    q  *store.Queries
    db *sql.DB
}

func NewSQLRepository(db *sql.DB) *SQLRepository {
    return &SQLRepository{
        q:  store.New(db),
        db: db,
    }
}

func (r *SQLRepository) CreateUser(ctx context.Context, id, username, email string) error {
    // Ejecución segura y tipada. El repositorio NO genera el ID.
    err := r.q.CreateUser(ctx, store.CreateUserParams{
        ID:       id,
        Username: username,
        Email:    email,
    })
    if err != nil {
        return fmt.Errorf("error persistiendo usuario: %w", err)
    }
    return nil
}
```

### Fase 4: Capa de Servicio (Lógica de Negocio)

Implementamos la interfaz gRPC generada por `protoc`. Aquí reside la lógica de orquestación y el **ownership** del dominio.

`modules/users/internal/service/service.go`:
```go
package service

import (
	"context"
	"log/slog"

	"go.jetify.com/typeid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	usersv1 "proyecto/gen/go/users/v1" // Código generado por Buf/Protoc
	"proyecto/modules/users/internal/repository"
)

type UserService struct {
    usersv1.UnimplementedUserServiceServer
    repo repository.Repository
}

func NewUserService(repo repository.Repository) *UserService {
    return &UserService{repo: repo}
}

func (s *UserService) CreateUser(ctx context.Context, req *usersv1.CreateUserRequest) (*usersv1.CreateUserResponse, error) {
    // 1. Lógica de Dominio: Generación de Identidades (TypeID)
    tid, _ := typeid.WithPrefix("user") // Generación centralizada en el Service
    idStr := tid.String()

    // 2. Llamada a persistencia
    err := s.repo.CreateUser(ctx, idStr, req.Username, req.Email)
    if err != nil {
        slog.ErrorContext(ctx, "failed to create user", "error", err)
        return nil, status.Error(codes.Internal, "failed to create user")
    }

    // 3. Mapeo a respuesta Proto
    return &usersv1.CreateUserResponse{
        Id:       idStr,
        Username: req.Username,
    }, nil
}
```

---
## 16. Workflows de Desarrollo (Development Workflows)

### Agregar un nuevo campo a una tabla

1.  Crear nuevo script de migración: `modules/[mod]/resources/db/migration/00X_add_field.up.sql`.
2.  Actualizar Queries en `.sql` si es necesario incluir el campo en SELECTs o INSERTs.
3.  Ejecutar `sqlc generate`. El struct Go se actualizará automáticamente.
4.  Corregir errores de compilación (el compilador de Go te avisará dónde falta el campo).

### Testing

Establecemos una disciplina de testing que garantice la calidad sin burocracia:

### Hot Reload (Desarrollo Rápido)

Para una experiencia de desarrollo fluida, utilizamos **Air** para recompilar automáticamente el código al guardar:

1.  **Monolito:** `make dev`
2.  **Microservicio Auth:** `make dev-auth`

> [!TIP]
> Air vigila cambios en archivos `.go`, `.yaml`, `.proto` y `.sql`, reiniciando el binario instantáneamente.

*   **Convención:** Archivos `*_test.go` al lado del código que prueban.
*   **Unit Tests:**
    *   **Enfoque:** Probar lógica de negocio pura y transformaciones.
    *   **Mocks:** **Mockear** la interfaz `repository.Repository` en los tests del Servicio. Prohibido usar DB real en unit tests.
*   **Integration Tests:**
    *   **Ubicación:** Pueden vivir dentro de cada módulo o en una carpeta `tests/integration` aparte.
    *   **Infra:** Usar `docker-compose` o **Testcontainers** para levantar una base de datos real.
    *   **Flujo:** Probar el endpoint gRPC -> Repository -> DB real y verificar efectos secundarios.

## 17. Generación Automática de Módulos (Scaffolding)

Para acelerar el inicio de nuevos módulos y asegurar que sigan los estándares definidos, disponemos de una herramienta de scaffolding.

*   **Comando:** `make new-module [nombre]`
*   **Archivos Generados:**
    *   `modules/[name]/module.go`: Inicialización y registro del Gateway.
    *   `modules/[name]/internal/service/service.go`: Boilerplate del servicio con TypeID y mapeo de errores.
    *   `modules/[name]/internal/repository/repository.go`: Interfaz y adaptador SQL con soporte de transacciones (`WithTx`).
    *   `modules/[name]/resources/db/migration/`: Script inicial de base de datos.
    *   `proto/[name]/v1/`: Contrato inicial del servicio.

## 18. Despliegue Granular y Configuración (Microservices Path)

Un Modulito bien diseñado permite transicionar de un único binario (Monolito) a múltiples binarios (Microservicios) sin cambiar la lógica de los módulos.

### Configuración por Módulo
Cada módulo debe definir su propio struct de configuración para evitar depender de variables globales.

```go
// modules/auth/module.go
type Config struct {
    JWTSecret string `yaml:"jwt_secret"`
}

func Initialize(db *sql.DB, grpcServer *grpc.Server, cfg Config) error { ... }
```

### Uso de YAML y Variables de Entorno
El proyecto utiliza un cargador centralizado en `internal/config` (basado en `yaml.v3`) con la siguiente jerarquía:
1.  **Archivos por Aplicación**: Se recomienda una carpeta `configs/` con archivos YAML específicos para cada entrypoint (ej. `configs/server.yaml`, `configs/auth-svc.yaml`).
2.  **Schema Unificado**: Aunque los archivos sean distintos, todos mapean al struct `AppConfig` central para mantener consistencia. Un microservicio simplemente ignorará las secciones YAML que no le correspondan.
3.  **Override por Environment Variable**: Las variables de entorno (ej. `DB_DSN`, `JWT_SECRET`) siempre tienen prioridad, siguiendo los principios de **12-Factor App**.

### De Monolito a Microservicios
La separación se logra creando diferentes puntos de entrada (`cmd/`) que apuntan a sus respectivos archivos de configuración:

1.  **Modo Monolito (`cmd/server/main.go`):** Inicia todos los módulos, una única conexión a DB y un único servidor gRPC.
2.  **Modo Microservicio (`cmd/auth-svc/main.go`):** Solo importa e inicializa el módulo de `auth`.

### Comunicación Inter-Módulo en Microservicios
Cuando los módulos viven en binarios distintos, las llamadas gRPC que antes eran in-process (directas) ahora deben viajar por la red. Para que esto sea transparente:
*   Se utiliza un **Service Discovery** o un **Load Balancer** interno.
*   El cliente gRPC inyectado en un módulo debe apuntar a la dirección del microservicio externo en lugar de `127.0.0.1` (o usar la misma interfaz de cliente).

---
## 19. Contenerización y Despliegue en la Nube

El proyecto está preparado para ejecutarse en entornos de contenedores (Docker) y orquestadores (Kubernetes) de forma nativa.

### Dockerfile: Multi-Stage Build
Utilizamos un `Dockerfile` optimizado con dos etapas:
1.  **Builder:** Compila el binario en una imagen de Go (Alpine). Permite elegir el target vía `--build-arg TARGET=[server|auth-svc]`.
2.  **Runner:** Una imagen ligera (`alpine`) que solo contiene el binario y los archivos de configuración necesarios.

```bash
# Ejemplo: Construir el microservicio de Auth
make docker-build-auth
```

### Helm Charts: Kubernetes
En `deployment/helm/modulith` se encuentra el chart estándar para desplegar en K8s.
- **Valores por Entorno:** Se recomienda usar diferentes archivos `values.yaml` (ej. `values-prod.yaml`) para manejar secretos y recursos por cluster.
- **Flexibilidad:** El mismo chart puede desplegar tanto el Monolito como instancias individuales de microservicios ajustando la imagen y los comandos de inicio.

## 20. Infraestructura como Código (IaC)

Manejamos la infraestructura utilizando un enfoque modular con **OpenTofu** (Fork Open Source de Terraform) y **Terragrunt** para garantizar entornos consistentes y reproducibles.

### Estructura de Directorios
*   `deployment/opentofu/modules/`: Definición de componentes base (VPC, RDS, EKS).
*   `deployment/terragrunt/envs/`: Configuraciones específicas por entorno (`dev`, `prod`).

### Módulos Principales
1.  **VPC (Red):** Configura subredes públicas (ELBs) y privadas (Nodos/DB) con NAT Gateway.
2.  **RDS (Base de Datos):** Instancia de PostgreSQL 16 aislada en subredes privadas.
3.  **EKS (Compute):** Cluster de Kubernetes gestionado con Node Groups escalables.

### Despliegue con Terragrunt
Terragrunt nos permite mantener el código DRY (Don't Repeat Yourself) y es 100% compatible con OpenTofu. Para desplegar el entorno de desarrollo:

```bash
cd deployment/terragrunt/envs/dev
terragrunt run-all plan  # Previsualizar cambios (usa tofu internamente)
terragrunt run-all apply # Aplicar infraestructura
```

---
## 21. CI/CD y Calidad de Código

El proyecto integra un pipeline de automatización para garantizar la estabilidad:

### GitHub Actions
Se ejecutan automáticamente en cada Push/PR:
1.  **Checksum/Verify:** Valida que las dependencias no hayan sido alteradas.
### Calidad de Código Estricta

El proyecto impone un estándar de calidad de "Clase Mundial" a través de un linter altamente configurado:

1.  **Linter Estricto:** `golangci-lint` está configurado para detectar no solo errores, sino también:
    - **Complejidad Ciclomática y Cognitiva:** Evita funciones inmanejables.
    - **Nivel de Anidación:** Máximo 5 niveles (linters `nestif`).
    - **Documentación:** Todo elemento público **DEBE** tener comentarios de Godoc.
    - **Seguridad:** Análisis estático con `gosec` en cada commit.
2.  **Validación de Configuración:** El cargador de configuración valida semánticamente las variables críticas antes de que la aplicación inicie (Fail-Fast).
3.  **Tests con Race Detection:** No se permite código con condiciones de carrera (`-race`).

## 22. Checklist de Replicabilidad para LLMs

Si estás utilizando un LLM para generar o extender este proyecto, asegúrate de seguir este orden lógico para mantener la integridad:

1.  **Skeleton Primero:** Crea la estructura de carpetas y los archivos `go.mod`, `buf.yaml`, `sqlc.yaml`.
2.  **Contrato (Proto):** Define los archivos `.proto` y genera el código con `buf generate`.
3.  **Persistencia (SQL):** Crea las migraciones `.sql` y genera el store con `sqlc generate`.
4.  **Repositorio:** Implementa la interfaz `Repository` envolviendo el código de `sqlc`.
5.  **Servicio:** Crea la lógica de negocio, genera los **TypeIDs** y realiza el mapeo de errores gRPC.
6.  **Cableado (Module):** Exporta la función `Initialize` del módulo y regístrala en `cmd/server/main.go`.
7.  **Inyección:** Asegúrate de que el `db *sql.DB` y el `bus *events.Bus` se pasen correctamente entre capas.

## 23. Abstracciones de Notificación (Event-Driven Notifiers)

Para evitar el acoplamiento con proveedores externos (Twilio, SendGrid, etc.), el sistema utiliza el **Patrón Adapter** combinado con un enfoque **Event-Driven**.

*   **Interfaces:** Definidas en `internal/notifier/notifier.go` (`EmailProvider`, `SMSProvider`).
*   **Implementación Reactiva:** Un `notifier.Subscriber` escucha eventos globales (ej. `auth.magic_code_requested`) y despacha la notificación de forma **asíncrona** y **no bloqueante**.
*   **LogNotifier para Dev:** Imprime las notificaciones en los logs estructurados, permitiendo probar flujos como el "Magic Code" sin configurar APIs externas.
*   **Inyección y Registro:**
    - El módulo (ej. `auth`) emite el evento al `Bus`.
    - El `Subscriber` se registra al `Bus` en el `main.go`, garantizando que la lógica de entrega esté totalmente fuera del dominio del módulo.

---
## 24. Futuras Mejoras y Nota Final

Esta arquitectura favorece la seguridad en tiempo de compilación y la disciplina operativa. Go 1.23+ se elige por el soporte nativo de `slog`, mejoras en el `toolchain` y optimizaciones de performance que permiten un código más limpio y eficiente.
