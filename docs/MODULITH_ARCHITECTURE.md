# Guía de Arquitectura e Implementación: Go Modulith

Esta documentación define el estándar arquitectónico y de implementación para proyectos nuevos ("greenfield"). Establece las directrices para construir un **Monolito Modular** robusto, escalable y mantenible, utilizando un stack tecnológico moderno y tipado estrictamente.

## 1. Stack Tecnológico Definido

Todas las implementaciones deben adherirse estrictamente a las siguientes tecnologías:

-   **Lenguaje:** Go 1.23+.
-   **Arquitectura:** Monolito Modular.
-   **Comunicación/Contrato:** gRPC y Protocol Buffers (Single Source of Truth).
-   **API Externa:**
    -   **gRPC:** Protocolo principal de comunicación backend-backend.
    -   **REST/HTTP:** Expuesto automáticamente vía `grpc-gateway` (Proxy inverso).
    -   **WebSocket:** Comunicación bidireccional en tiempo real (`/ws`).
    -   **GraphQL (Opcional):** API flexible con subscripciones vía WebSocket (`/graphql`).
    -   **Documentación:** Swagger UI (OpenAPIv2) disponible en `/swagger-ui/` (Solo Dev).
-   **Persistencia:** SQLC (Type-safe SQL).
-   **Base de Datos:** PostgreSQL (con migraciones versionadas).
-   **Infraestructura Local:** Docker Compose.
-   **Migraciones:** `golang-migrate` (Gestión de esquema).
-   **Observabilidad:**
    -   **Logs:** Structured Logging (`log/slog`) con formato JSON.
    -   **Métricas:** OpenTelemetry (OTel) exponiendo métricas en formato Prometheus.
    -   **Tracing:** OpenTelemetry (Context propagation).

## 2. Estructura del Proyecto (Project Layout)

La organización de carpetas es crítica para mantener la modularidad. Cada módulo debe ser autocontenido.

```text
proyecto/
├── cmd/
│   ├── server/             # Entrypoint Monolito (main.go)
│   └── auth/           # Entrypoint Microservicio (main.go)
├── configs/                # Configuraciones YAML por aplicación
│   ├── server.yaml         # Configuración del monolito
│   └── auth.yaml       # Configuración del microservicio
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

-   **Importaciones:** Un módulo `A` **NUNCA** puede importar nada de la carpeta `internal/` de un módulo `B`.
-   **Comunicación:** La única forma legítima de comunicación entre módulos es:
    1.  **gRPC (in-process):** Llamando a través del cliente gRPC generado (usando el gateway interno). Al ser _in-process_, **no hay saltos de red**; es una llamada a función directa a través del stack de gRPC, garantizando performance y contratos fuertes.
    2.  **Eventos:** Publicación/Suscripción (si se implementa en el futuro).
-   **Datos:** Prohibido compartir repositorios, queries de SQLC o modelos de base de datos entre módulos. Cada módulo es dueño absoluto de su esquema.
-   **DTOs:** Los mensajes de Protobuf son el lenguaje común. No se deben filtrar tipos de `store/` o `repository/` hacia afuera del propio módulo.

## 4. Dominio y Modelos

Para evitar debates infinitos, establecemos el siguiente estándar:

-   **Domain Ownership:** La lógica de negocio reside en la capa de `service/`.
-   **Modelos Simples:** No utilizamos entidades ricas (DDD complejo) a menos que sea estrictamente necesario.
-   **Flujo:** `store` (DB) -> `repository` (Adapter) -> `service` (Domain/Business) -> `proto` (DTO).
-   **Repository:** Devuelve structs simples del `store` o modelos de dominio básicos en `internal/models/`. No hay lógica de negocio en el repositorio.

## 5. Identificadores Únicos (TypeID)

Para mejorar la trazabilidad, depuración y ordenabilidad de los datos, adoptamos el estándar de **Identificadores Prefijados y Ordenables por Tiempo** (estilo Stripe).

-   **Estándar:** Utilizaremos **TypeID** (`github.com/jetpack-io/typeid-go`), que combina un prefijo legible con un **UUIDv7**.
-   **Formato:** `prefix_01h455vb4pex5vsknk084sn02q`.
    -   **Prefix:** Indica el tipo de entidad (ej. `user`, `role`, `org`). Máximo 8 caracteres.
    -   **Suffix:** Un UUIDv7 codificado en Base32 (Crockford), lo que lo hace lexicográficamente ordenable.
-   **Ventajas:**
    -   **Sortable:** La ordenabilidad por tiempo permite que las bases de datos (PostgreSQL) indexen de forma más eficiente que con UUIDs aleatorios.
    -   **Contextual:** Al ver un ID en un log (`user_...`), sabemos inmediatamente a qué entidad pertenece.
    -   **Seguridad:** Son globalmente únicos y difíciles de predecir.
-   **Ownership:** Los TypeIDs se generan **únicamente** en la capa de `service`. El repositorio y la base de datos son pasivos y nunca generan identificadores.
-   **Semántica:** Los prefijos son puramente informativos para humanos y trazabilidad; no deben usarse para lógica de autorización o acceso cross-domain.

> [!NOTE]
> En este documento, por simplicidad, los TypeIDs se representan y almacenan como `VARCHAR` completos. En implementaciones de alto rendimiento, se podría almacenar solo el sufijo binario como `UUID` y reconstruir el prefijo en la aplicación.

## 6. Manejo de Errores gRPC

El template proporciona un sistema de manejo de errores estandarizado en `internal/errors` que elimina el boilerplate y garantiza consistencia.

### Domain Errors con Mapeo Automático a gRPC

En lugar de mapear manualmente cada error a códigos gRPC, utilizamos errores de dominio tipados:

```go
import "github.com/cmelgarejo/go-modulith-template/internal/errors"

// En el servicio
func (s *Service) CreateUser(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    // Los errores de dominio se mapean automáticamente
    if err := s.repo.CreateUser(ctx, id, email); err != nil {
        return nil, errors.ToGRPC(errors.Internal("failed to create user", errors.WithWrappedError(err)))
    }

    return &pb.Response{Id: id}, nil
}
```

### Tipos de Errores Disponibles

El paquete `internal/errors` proporciona constructores para todos los casos comunes:

```go
// Not found (maps to codes.NotFound)
errors.NotFound("user not found")

// Validation (maps to codes.InvalidArgument)
errors.Validation("invalid email format")

// Already exists (maps to codes.AlreadyExists)
errors.AlreadyExists("user already exists")

// Unauthorized (maps to codes.Unauthenticated)
errors.Unauthorized("authentication required")

// Forbidden (maps to codes.PermissionDenied)
errors.Forbidden("access denied")

// Conflict (maps to codes.AlreadyExists)
errors.Conflict("resource conflict")

// Internal (maps to codes.Internal)
errors.Internal("internal server error")

// Unavailable (maps to codes.Unavailable)
errors.Unavailable("service temporarily unavailable")
```

### Opciones de Error

Los errores pueden incluir detalles adicionales:

```go
err := errors.NotFound("user not found",
    errors.WithDetail("user_id", userID),
    errors.WithWrappedError(sqlErr),
)
```

### Verificación de Tipos

```go
if errors.Is(err, errors.TypeNotFound) {
    // Handle not found case
}

var domainErr *errors.DomainError
if errors.As(err, &domainErr) {
    // Access error details
    log.Info("error type", "type", domainErr.Type)
}
```

### Beneficios

-   ✅ **Consistencia:** Todos los servicios usan el mismo formato de error
-   ✅ **Trazabilidad:** Los errores wrappean la cadena completa con `%w`
-   ✅ **Type-safe:** El compilador detecta errores en los tipos
-   ✅ **Menos Boilerplate:** No más `status.Error()` manual en cada servicio

## 7. Transacciones

Las transacciones deben ser controladas por la capa de negocio (`service`) pero ejecutadas por el repositorio.

-   **Patrón WithTx:** El repositorio debe ofrecer una forma de ejecutar múltiples operaciones en una transacción atómica.
-   **Ejemplo Conceptual:**

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

-   **Estructural (Proto):** Formato, longitud, obligatorios, rangos. Se valida preferiblemente en el interceptor o al inicio del `service` usando la estructura del proto.
-   **Negocio (Service):** Existencia en BD, permisos complejos, reglas de estado, lógica temporal.

## 9. Seguridad: Autenticación y Autorización

### 9.1 Autenticación (JWT)

-   **Validación de Tokens:** Se realiza centralizadamente en un **gRPC Interceptor** global.
-   **Contexto:** El interceptor extrae el `user_id` y `role` del token y los inyecta en el `context.Context` para que estén disponibles en toda la cadena de llamada.
-   **Endpoints Públicos:** Los módulos declaran sus endpoints públicos (login, registro) que no requieren autenticación.

### 9.2 Autorización (RBAC)

El template incluye helpers de autorización en `internal/authz` para implementar control de acceso basado en roles y permisos:

#### Verificación de Permisos

```go
import "github.com/cmelgarejo/go-modulith-template/internal/authz"

func (s *Service) DeleteUser(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    // Require specific permission
    if err := authz.RequirePermission(ctx, "users:delete"); err != nil {
        return nil, errors.ToGRPC(err)
    }

    // Business logic...
}
```

#### Verificación de Roles

```go
// Require one of multiple roles
if err := authz.RequireRole(ctx, authz.RoleAdmin, authz.RoleModerator); err != nil {
    return nil, errors.ToGRPC(err)
}
```

#### Verificación de Propiedad del Recurso

```go
// Ensure user owns the resource
if err := authz.RequireOwnership(ctx, req.UserId); err != nil {
    return nil, errors.ToGRPC(err)
}

// Allow ownership OR specific roles (flexible)
if err := authz.RequireOwnershipOrRole(ctx, req.UserId, authz.RoleAdmin); err != nil {
    return nil, errors.ToGRPC(err)
}
```

#### Roles y Permisos Personalizados

Registra roles personalizados durante la inicialización del módulo:

```go
func init() {
    authz.RegisterRole("moderator",
        "posts:delete",
        "comments:delete",
        "users:ban",
    )

    authz.RegisterRole("editor",
        "posts:create",
        "posts:edit",
        "posts:publish",
    )
}
```

#### Roles Predefinidos

-   **`admin`**: Tiene permiso wildcard (`*`) - acceso total
-   **`user`**: Permisos básicos (`users:read`, `profile:read`, `profile:edit`)

#### Beneficios

-   ✅ **Centralizado:** Toda la lógica de autorización en un solo lugar
-   ✅ **Reutilizable:** Los mismos helpers funcionan en todos los módulos
-   ✅ **Type-safe:** Roles y permisos como constantes tipadas
-   ✅ **Flexible:** Soporta permisos, roles y ownership

### 9.3 OAuth/Social Login

El template soporta autenticación con proveedores externos usando [markbates/goth](https://github.com/markbates/goth):

-   **Providers soportados:** Google, Facebook, GitHub, Apple, Microsoft, Twitter/X
-   **Auto-link por email:** Vincula automáticamente cuentas externas a usuarios existentes con el mismo email
-   **Linking manual:** Los usuarios pueden vincular/desvincular cuentas desde su perfil
-   **Encriptación de tokens:** Los tokens OAuth se encriptan con AES-256-GCM antes de almacenarse

Para configuración completa, ver [OAuth Integration Guide](OAUTH_INTEGRATION.md).

## 10. Configuración y Entorno (Environment)

La jerarquía de configuración favorece la flexibilidad tanto en desarrollo como en despliegues complejos de microservicios.

### Jerarquía de Carga

La aplicación carga la configuración siguiendo un orden de precedencia estricto (de menor a mayor prioridad):

1.  **Valores por Defecto:** Valores hardcodeados en `config.go` (ej. `Env: "dev"`, `HTTPPort: "8080"`).
2.  **Variables de Entorno del Sistema:** Variables definidas en el entorno donde se ejecuta la aplicación (`os.Getenv`).
3.  **Archivo `.env`:** Variables cargadas desde el archivo `.env` en la raíz del proyecto (usando `godotenv`). Sobrescribe las variables del sistema.
4.  **Archivo YAML:** Configuración ubicada en `configs/` (ej. `configs/server.yaml`). **Tiene la mayor prioridad** y sobrescribe todo lo anterior.

**Orden de Precedencia Final:** `YAML > .env > system ENV vars > defaults`

### Logging de Fuentes de Configuración

Al iniciar la aplicación, se registra un log estructurado que muestra el valor final y la fuente de cada variable de configuración:

```
Configuration sources
  ENV="dev = yaml"
  HTTP_PORT="8080 = yaml"
  DB_DSN="postgres://... = yaml"
  JWT_SECRET="[42 bytes] = yaml"
```

Esto facilita la depuración y el entendimiento de qué fuente está proporcionando cada valor.

### Agregar nueva configuración

1.  Añadir el campo al struct en `internal/config/config.go` con los tags `yaml` y `env` correspondientes.
2.  Implementar la lógica de carga en `OverrideWithEnv` y `OverrideWithEnvFromDotenv` para soportar variables de entorno.
3.  Actualizar los archivos YAML en `configs/` si el valor es específico del entorno.
4.  Inyectar el struct de configuración en la función `Initialize` del módulo correspondiente.

### Variables de Entorno Clave

Aunque residan en el YAML, estas variables son críticas para el entorno de ejecución:

-   `ENV`: `dev` o `prod`. Determina el nivel de logs y la activación de herramientas de depuración.
-   `DB_DSN`: Conexión a PostgreSQL.
-   `JWT_SECRET`: Clave secreta para tokens JWT. **Debe tener al menos 32 bytes (256 bits)** para el algoritmo HS256. Se valida automáticamente al cargar la configuración.
-   `HTTP_PORT` / `GRPC_PORT`: Puertos de escucha.

### Validación de Configuración

El sistema valida automáticamente la configuración antes de iniciar:

-   **JWT Secret:** Debe tener al menos 32 bytes para cumplir con los requisitos de seguridad de HS256.
-   **Producción:** En modo `prod`, se requiere `DB_DSN` y `JWT_SECRET` obligatoriamente.

## 11. Infraestructura Local (Docker)

Utilizamos Docker Compose para levantar dependencias (Base de Datos).

-   El puerto de PostgreSQL es configurable vía `DB_PORT` en el `.env` del host.
-   Comandos útiles en `Makefile`: `make docker-up`, `make docker-down`.

## 12. Observabilidad

La observabilidad es ciudadana de primera clase. No se debe desplegar código sin visibilidad.

### 12.1. Logs Estructurados

Usamos la librería estándar `log/slog` (Go 1.21+).

-   **Formato:** JSON en producción, Texto en desarrollo.
-   **Contexto:** Todo log debe incluir `trace_id` y `span_id` si existen en el contexto.
-   **Niveles:** INFO (flujo normal), ERROR (excepciones), DEBUG (solo dev). El nivel DEBUG está habilitado por defecto en desarrollo para facilitar la depuración.
-   **Inicialización Temprana:** El logger se inicializa en dos fases: primero con un logger básico antes de cargar la configuración (para ver logs de inicialización), y luego se re-inicializa con la configuración completa (formato, nivel) después de cargar la configuración.
-   **Privacidad (PII):** **NUNCA** loguear información sensible (emails, tokens, passwords).

```go
slog.InfoContext(ctx, "user created", "user_id", id) // Evitar loguear el email aquí
```

### 12.2. Métricas (OpenTelemetry)

Instrumentamos la aplicación usando el SDK de OpenTelemetry.

-   **Protocolo:** Prometheus (`/metrics`).
-   **Métricas Standard:**
    -   `http_request_duration_seconds` (Histograma).
    -   `grpc_server_handled_total` (Contador).
-   **Mapeo:** Middleware/Interceptores automáticos para gRPC y HTTP.

### 12.3. Health Checks

El sistema expone dos endpoints críticos para el orquestador (K8s):

-   **/healthz (Liveness):** Indica si el proceso está vivo. Retorna `200 OK`.
-   **/readyz (Readiness):** Indica si el servicio puede recibir tráfico. Valida la conexión a la base de datos usando `db.PingContext(r.Context())` para respetar los timeouts del cliente HTTP y permitir que el orquestador cancele la verificación si es necesario.

### 12.4. Tracing (OpenTelemetry)

Implementamos trazabilidad distribuida usando el exportador OTLP.

-   **Propagación:** Los traces viajan automáticamente a través de los interceptores gRPC.
-   **Contexto:** Permite ver el camino de una petición desde el gateway hasta el repositorio.

### 12.5. Helpers de Telemetría (`internal/telemetry`)

Para eliminar el boilerplate de OpenTelemetry, el template proporciona helpers que simplifican la instrumentación:

#### Spans por Capa

```go
import "github.com/cmelgarejo/go-modulith-template/internal/telemetry"

// Service layer - auto-includes module and operation attributes
func (s *Service) CreateUser(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    ctx, span := telemetry.ServiceSpan(ctx, "auth", "CreateUser")
    defer span.End()

    // Add custom attributes
    telemetry.SetAttribute(ctx, "user_email", req.Email)

    // Business logic...
    if err != nil {
        telemetry.RecordError(span, err)
        return nil, errors.ToGRPC(err)
    }

    return &pb.Response{Id: id}, nil
}

// Repository layer - includes entity name
func (r *Repo) GetUser(ctx context.Context, id string) (*User, error) {
    ctx, span := telemetry.RepositorySpan(ctx, "auth", "GetUser", "user")
    defer span.End()

    user, err := r.q.GetUserByID(ctx, id)
    if err != nil {
        telemetry.RecordError(span, err)
        return nil, fmt.Errorf("failed to get user: %w", err)
    }

    return user, nil
}
```

#### Helpers Disponibles

-   **`telemetry.StartSpan(ctx, name)`** - Span básico
-   **`telemetry.ServiceSpan(ctx, module, operation)`** - Span de capa de servicio
-   **`telemetry.RepositorySpan(ctx, module, operation, entity)`** - Span de repositorio
-   **`telemetry.SetAttribute(ctx, key, value)`** - Agregar atributo al span actual
-   **`telemetry.RecordError(span, err)`** - Registrar error en el span
-   **`telemetry.AddEvent(ctx, name, attrs)`** - Agregar evento al span

#### Beneficios

-   ✅ **Menos Boilerplate:** No más imports de múltiples paquetes de OTel
-   ✅ **Consistencia:** Todos los spans siguen la misma convención de nombres
-   ✅ **Atributos Automáticos:** Module, operation y entity se incluyen automáticamente
-   ✅ **Context Propagation:** El context se propaga correctamente entre capas

## 13. Comunicación Asíncrona (Eventos)

Para evitar acoplamiento fuerte entre módulos, disponemos de un **Bus de Eventos** interno (`internal/events`).

-   **Patrón Pub/Sub:** Los módulos se suscriben a eventos (ej. `user.created`) sin conocer quién los emite.
-   **No Bloqueante:** La publicación de eventos ocurre en goroutines separadas para no penalizar el tiempo de respuesta gRPC/HTTP.
-   **Extensibilidad:** Facilita añadir efectos secundarios (auditoría, notificaciones) sin modificar el servicio original.

### Eventos Tipados (`internal/events/types.go`)

Para evitar errores de tipeo y mejorar el autocomplete, el template incluye constantes tipadas para eventos comunes:

```go
import "github.com/cmelgarejo/go-modulith-template/internal/events"

// En el servicio - usando constantes tipadas
bus.Publish(ctx, events.Event{
    Name:    events.UserCreatedEvent,  // Autocomplete disponible!
    Payload: events.NewUserCreatedPayload(userID, email),
})

// Suscripción - usando las mismas constantes
bus.Subscribe(events.UserCreatedEvent, func(ctx context.Context, e events.Event) error {
    slog.InfoContext(ctx, "audit: logging user creation", "user_id", e.Payload["user_id"])
    return nil
})
```

#### Eventos Predefinidos

El template incluye eventos comunes del módulo auth:

```go
// Auth module events
events.UserCreatedEvent           // "user.created"
events.MagicCodeRequestedEvent    // "auth.magic_code_requested"
events.SessionCreatedEvent        // "auth.session_created"
events.ProfileUpdatedEvent        // "user.profile_updated"
events.OAuthAccountLinkedEvent    // "auth.oauth_account_linked"
events.ContactChangeRequestedEvent // "user.contact_change_requested"
```

#### Agregar Eventos de Tu Módulo

Añade tus eventos en `internal/events/types.go`:

```go
const (
    OrderCreatedEvent   = "order.created"
    OrderCancelledEvent = "order.cancelled"
    OrderShippedEvent   = "order.shipped"
)

// Helper para crear payloads type-safe
func NewOrderCreatedPayload(orderID, userID string, amount float64) (map[string]any, error) {
    if orderID == "" || userID == "" {
        return nil, Validation("order ID and user ID are required")
    }
    return map[string]any{
        "order_id": orderID,
        "user_id":  userID,
        "amount":   amount,
    }, nil
}
```

#### Beneficios

-   ✅ **Type-safe:** El compilador detecta nombres de eventos incorrectos
-   ✅ **Autocomplete:** Los IDEs sugieren eventos disponibles
-   ✅ **Validación:** Los helpers de payload validan campos requeridos
-   ✅ **Documentación:** Los eventos están centralizados y son fáciles de descubrir

## 13.1. WebSocket: Comunicación en Tiempo Real

El proyecto incluye soporte completo para **WebSocket** (`internal/websocket`), permitiendo comunicación bidireccional y en tiempo real con los clientes.

### Características

-   **Integración con Event Bus:** Los eventos publicados en el bus pueden ser enviados automáticamente a clientes WebSocket conectados.
-   **Autenticación JWT:** Las conexiones WebSocket están protegidas mediante JWT extraído del query parameter (`?token=...`).
-   **Mensajes Dirigidos:** Soporte para broadcast (todos los clientes) y mensajes específicos por `user_id`.
-   **Gestión de Ciclo de Vida:** Manejo automático de conexiones, desconexiones, heartbeat (ping/pong).

### Arquitectura

```
┌─────────────┐      ┌──────────────┐      ┌─────────────┐
│   Client    │─────▶│  WebSocket   │─────▶│     Hub     │
│  (Browser)  │      │   Handler    │      │  (Manager)  │
└─────────────┘      └──────────────┘      └─────────────┘
                                                   │
                                                   ▼
                                            ┌─────────────┐
                                            │  Event Bus  │
                                            │ Subscriber  │
                                            └─────────────┘
```

### Ejemplo de Uso

```go
// Enviar evento desde un módulo (se propagará vía WebSocket)
bus.Publish(ctx, events.Event{
    Name: "notification.new",
    Payload: map[string]any{
        "user_id": "user_123",
        "message": "Nueva notificación",
    },
})

// El subscriber de WebSocket lo captura y envía al cliente conectado
```

**Endpoint:** `ws://localhost:8080/ws?token={jwt_token}`

**Ver guía completa:** `docs/WEBSOCKET_GUIDE.md`

## 13.2. GraphQL: API Flexible (Opcional)

El proyecto soporta integración **opcional** de GraphQL usando `gqlgen`, proporcionando una alternativa flexible a gRPC/REST.

### Características

-   **Schema por Módulo:** Cada módulo define su propio schema GraphQL (`internal/graphql/schema/{module}.graphql`).
-   **Subscriptions:** Soporte para subscripciones en tiempo real vía WebSocket.
-   **Integración con Event Bus:** Las subscriptions pueden escuchar eventos del bus interno.
-   **Setup Automatizado:** Script de instalación y configuración (`scripts/add-graphql.sh`).

### Arquitectura

```
internal/graphql/
├── schema/
│   ├── schema.graphql      # Root schema (combina todos)
│   ├── auth.graphql        # Schema del módulo auth
│   └── order.graphql       # Schema del módulo order
├── resolver/
│   ├── resolver.go         # Root resolver
│   ├── auth.go             # Resolvers de auth
│   └── order.go            # Resolvers de order
└── server.go               # Setup de GraphQL
```

### Instalación y Uso

```bash
# 1. Agregar GraphQL al proyecto
make add-graphql

# 2. Definir schemas por módulo en internal/graphql/schema/

# 3. Generar código
make graphql-generate

# 4. Implementar resolvers en internal/graphql/resolver/

# 5. Validar
make graphql-validate
```

**Endpoints:**
-   GraphQL API: `POST /graphql`
-   Playground: `GET /graphql/playground` (solo dev)

**Ver guía completa:** `docs/GRAPHQL_INTEGRATION.md`

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

-- name: GetValidMagicCodeByEmail :one
SELECT * FROM magic_codes
WHERE user_email = $1 AND code = $2 AND expires_at > $3
ORDER BY created_at DESC LIMIT 1;
```

> [!NOTE]
> Para queries que involucran comparaciones de tiempo (ej. códigos mágicos con expiración), se recomienda pasar el tiempo actual como parámetro (`$3`) desde la aplicación en lugar de usar `CURRENT_TIMESTAMP` en SQL. Esto garantiza consistencia entre el tiempo de la aplicación y el tiempo de la base de datos, evitando problemas de sincronización.

**3. Configuración SQLC:**
`sqlc.yaml`:

```yaml
version: "2"
sql:
    - egine: "postgresql"
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

**5. Sistema de Migraciones Multi-Módulo (`internal/migration`):**

El template incluye un sistema automático de descubrimiento y ejecución de migraciones para todos los módulos registrados.

#### Declaración en el Módulo

Cada módulo implementa la interfaz `ModuleMigration` para declarar su ruta de migraciones:

```go
// En modules/users/module.go
func (m *Module) MigrationPath() string {
    return "modules/users/resources/db/migration"
}
```

#### Ejecución Automática

Las migraciones se ejecutan automáticamente al iniciar el servidor:

```go
// En cmd/server/main.go - ya está implementado
runner := migration.NewRunner(cfg.DBDSN, reg)
if err := runner.RunAll(); err != nil {
    return err
}
```

El sistema:
1. Descubre todos los módulos que implementan `ModuleMigration`
2. Ejecuta las migraciones en el orden de registro de módulos
3. Usa `golang-migrate` internamente para track de versiones
4. Cada módulo mantiene su propio historial de migraciones

#### Comandos de Makefile

```bash
# Ejecutar todas las migraciones de todos los módulos
make migrate-up  # o simplemente: make migrate

# Revertir la última migración de un módulo específico
make migrate-down MODULE=users

# Crear una nueva migración para un módulo
make migrate-create MODULE=users NAME=add_profile_fields

# Borrar todas las tablas y re-ejecutar migraciones
make db-down    # Solo borra las tablas
make db-reset   # Borra y re-ejecuta (db-down + migrate-up)
```

#### Ejecución Manual de Solo Migraciones

```bash
# Ejecutar solo migraciones sin levantar el servidor
go run cmd/server/main.go -migrate
# o
make migrate
```

#### Beneficios

-   ✅ **Automático:** No necesitas modificar `main.go` al agregar módulos
-   ✅ **Ordenado:** Las migraciones se ejecutan en el orden de registro
-   ✅ **Autónomo:** Cada módulo gestiona su propio esquema
-   ✅ **Portable:** Funciona tanto en monolito como en microservicios

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
  "database/sql"
  "errors"
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
        // Manejo de errores específicos: mapeo a códigos gRPC apropiados
        if errors.Is(err, sql.ErrNoRows) {
            slog.DebugContext(ctx, "user not found", "email", req.Email)
            return nil, status.Error(codes.NotFound, "user not found")
        }

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

### Mocking (gomock)

Para facilitar el testing unitario, utilizamos **gomock** (`go.uber.org/mock`) para generar mocks automáticos de interfaces.

**Filosofía:**
-   **Type-safe:** Los mocks fallan en compilación si la interfaz cambia, garantizando que los tests estén siempre sincronizados.
-   **Automático:** Generación mediante `//go:generate`, alineado con la filosofía del proyecto (sqlc, buf).
-   **Validable:** Verificaciones de expectativas en tests para asegurar que el código llama a las dependencias correctamente.

**Comandos:**

```bash
# Instalar herramienta
make install-mocks

# Generar todos los mocks
make generate-mocks

# Ejecutar tests unitarios (genera mocks automáticamente)
make test-unit
```

**Agregar mocks a una nueva interfaz:**

1.  Agregar anotación al inicio del archivo (antes del package doc):

```go
//go:generate mockgen -source=myinterface.go -destination=mocks/myinterface_mock.go -package=mocks

// Package mypackage provides...
package mypackage

type MyInterface interface {
    DoSomething(ctx context.Context, id string) error
}
```

2.  Generar: `make generate-mocks`

3.  Usar en tests:

```go
package service_test

import (
    "testing"
    "go.uber.org/mock/gomock"
    "yourproject/path/to/mocks"
)

func TestWithMock(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mock := mocks.NewMockMyInterface(ctrl)

    // Setup expectations
    mock.EXPECT().
        DoSomething(gomock.Any(), "user_123").
        Return(nil).
        Times(1)

    // Test code that uses the mock
    // ...
}
```

**Mocks vs Repository Real:**
-   **Unit Tests:** Usar mocks (rápidos, aislados, no requieren DB).
-   **Integration Tests:** Usar DB real con Testcontainers (validan queries SQL reales).

**Ejemplo Real:**

Ver `modules/auth/internal/service/service_mock_test.go` para ejemplos completos de cómo testear servicios usando mocks del repositorio.

### Hot Reload (Desarrollo Rápido)

Para una experiencia de desarrollo fluida, utilizamos **Air** para recompilar automáticamente el código al guardar:

1.  **Monolito:** `make dev`
2.  **Cualquier Módulo:** `make dev-module {nombre}` (ej. `make dev-module auth`)

> [!TIP]
> Air vigila cambios en archivos `.go`, `.yaml`, `.yml`, `.proto`, `.sql`, `.env` y archivos de configuración específicos, reiniciando el binario instantáneamente. El generador de módulos (`make new-module`) crea automáticamente el archivo `.air.{module}.toml` necesario.

### Comandos de Build Genéricos

El proyecto proporciona comandos comodín para trabajar con cualquier módulo:

```bash
# Build
make build-module auth      # Genera bin/auth
make build-module payments  # Genera bin/payments
make build-all              # Compila server + todos los módulos

# Docker
make docker-build-module auth      # Genera modulith-auth:latest
make docker-build-module payments  # Genera modulith-payments:latest

# Desarrollo con Hot Reload
make dev-module auth      # Ejecuta auth con hot reload
make dev-module payments  # Ejecuta payments con hot reload
```

> [!NOTE]
> Todos los binarios se compilan en el directorio `bin/` centralizado, ignorado por Git.

-   **Convención:** Archivos `*_test.go` al lado del código que prueban.
-   **Unit Tests:**
    -   **Enfoque:** Probar lógica de negocio pura y transformaciones.
    -   **Mocks:** **Mockear** la interfaz `repository.Repository` en los tests del Servicio. Prohibido usar DB real en unit tests.
-   **Integration Tests:**
    -   **Ubicación:** Pueden vivir dentro de cada módulo o en una carpeta `tests/integration` aparte.
    -   **Infra:** Usar `docker-compose` o **Testcontainers** para levantar una base de datos real.
    -   **Flujo:** Probar el endpoint gRPC -> Repository -> DB real y verificar efectos secundarios.

## 17. Generación Automática de Módulos (Scaffolding)

Para acelerar el inicio de nuevos módulos y asegurar que sigan los estándares definidos, disponemos de una herramienta de scaffolding robusta.

-   **Comando:** `make new-module {nombre}` (ej. `make new-module payments`)
-   **Automatización:**
    -   Genera la estructura de carpetas estándar.
    -   Crea los archivos boilerplate (`module.go`, `service.go`, `repository.go`, `proto`).
    -   **Configura automáticamente `sqlc.yaml`** añadiendo la entrada para el nuevo módulo.
    -   **Genera archivo `.air.{module}.toml`** para hot reload con Air.
    -   **Crea `cmd/{module}/main.go`** para despliegue independiente como microservicio.
    -   **Crea `configs/{module}.yaml`** con configuración específica del módulo.
    -   **Manejo de Plurales:** Detecta nombres en plural (ej. `products`) y ajusta el nombre del struct generado (ej. `Product`) en los templates para evitar errores de compilación.
-   **Archivos Generados:**
    -   `cmd/[name]/main.go`: Entrypoint para microservicio independiente.
    -   `configs/[name].yaml`: Configuración específica del módulo.
    -   `.air.[name].toml`: Configuración de hot reload.
    -   `modules/[name]/module.go`: Implementación completa de `registry.Module` con:
        -   `Name()` - Identificador del módulo
        -   `Initialize(reg)` - Inicialización con acceso al registry
        -   `RegisterGRPC(server)` - Registro de handlers gRPC
        -   `RegisterGateway(ctx, mux, conn)` - Registro de gateway HTTP
        -   `MigrationPath()` - Ruta de migraciones del módulo
        -   `PublicGRPCEndpoints()` - Endpoints públicos (sin auth)
    -   `modules/[name]/internal/service/service.go`: Servicio con:
        -   Integración con `internal/errors` para manejo de errores
        -   Integración con `internal/telemetry` para tracing
        -   Integración con `internal/events` para pub/sub
        -   Generación de TypeIDs
        -   Validación y autorización
    -   `modules/[name]/internal/repository/repository.go`:
        -   Interfaz `Repository` para testabilidad
        -   Implementación `SQLRepository` con SQLC
        -   Soporte de transacciones con `WithTx()`
    -   `modules/[name]/resources/db/migration/`: Scripts SQL iniciales (up/down)
    -   `proto/[name]/v1/`: Definición Protocol Buffer con anotaciones HTTP

**Después de generar un módulo:**
```bash
# Generar código
make proto  # Genera código gRPC
make sqlc   # Genera código de DB

# Build
make build-module payments

# Docker
make docker-build-module payments

# Desarrollo
make dev-module payments
```

### Quick Start: Creando Tu Primer Módulo

Una vez generado el módulo con `make new-module orders`, implementa la lógica de negocio:

```go
// modules/orders/internal/service/service.go
func (s *Service) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
    // 1. Telemetry (ya incluido en el template)
    ctx, span := telemetry.ServiceSpan(ctx, "orders", "CreateOrder")
    defer span.End()

    // 2. Autorización (usando helpers del template)
    if err := authz.RequirePermission(ctx, "orders:create"); err != nil {
        return nil, errors.ToGRPC(err)
    }

    // 3. Validación de negocio
    if req.Amount <= 0 {
        return nil, errors.ToGRPC(errors.Validation("amount must be positive"))
    }

    // 4. Generar TypeID (sortable, prefijado)
    tid, _ := typeid.WithPrefix("order")
    id := tid.String()

    // 5. Persistencia
    if err := s.repo.CreateOrder(ctx, id, req); err != nil {
        telemetry.RecordError(span, err)
        return nil, errors.ToGRPC(errors.Internal("failed to create order", errors.WithWrappedError(err)))
    }

    // 6. Publicar evento (typed event)
    payload, _ := events.NewOrderCreatedPayload(id, req.UserId, req.Amount)
    s.bus.Publish(ctx, events.Event{
        Name:    events.OrderCreatedEvent,
        Payload: payload,
    })

    return &pb.CreateOrderResponse{Id: id}, nil
}
```

**Todo lo anterior utiliza las abstracciones del template - tu código solo contiene lógica de negocio.**

## 18. Despliegue Granular y Configuración (Microservices Path)

Un Modulito bien diseñado permite transicionar de un único binario (Monolito) a múltiples binarios (Microservicios) sin cambiar la lógica de los módulos.

### Configuración por Módulo

Cada módulo debe definir su propio struct de configuración para evitar depender de variables globales.

```go
// modules/auth/module.go
type Config struct {
    JWTSecret string `yaml:"jwt_secret"` // Tag yaml requerido para mapeo desde YAML
}

func Initialize(db *sql.DB, grpcServer *grpc.Server, bus *events.Bus, cfg Config) error {
    // Validación temprana: verificar que la configuración requerida esté presente
    if cfg.JWTSecret == "" {
        return fmt.Errorf("JWT secret is empty, cannot initialize auth module")
    }
    // ... resto de la inicialización
}
```

### Uso de YAML y Variables de Entorno

El proyecto utiliza un cargador centralizado en `internal/config` (basado en `yaml.v3`) con la siguiente jerarquía:

1.  **Archivos por Aplicación**: Se recomienda una carpeta `configs/` con archivos YAML específicos para cada entrypoint (ej. `configs/server.yaml`, `configs/auth.yaml`).
2.  **Schema Unificado**: Aunque los archivos sean distintos, todos mapean al struct `AppConfig` central para mantener consistencia. Un microservicio simplemente ignorará las secciones YAML que no le correspondan.
3.  **Jerarquía de Precedencia**: El orden de carga es: **YAML > .env > system ENV vars > defaults**. Esto significa que los valores en el YAML tienen la máxima prioridad, seguidos por el archivo `.env`, luego las variables del sistema, y finalmente los valores por defecto.
4.  **Trazabilidad**: Al iniciar, la aplicación registra la fuente de cada variable de configuración, facilitando la depuración y el entendimiento de qué valor se está utilizando.

### De Monolito a Microservicios

La separación se logra creando diferentes puntos de entrada (`cmd/`) que apuntan a sus respectivos archivos de configuración:

1.  **Modo Monolito (`cmd/server/main.go`):** Inicia todos los módulos, una única conexión a DB y un único servidor gRPC.
2.  **Modo Microservicio (`cmd/auth/main.go`):** Solo importa e inicializa el módulo de `auth`.

### Comunicación Inter-Módulo en Microservicios

Cuando los módulos viven en binarios distintos, las llamadas gRPC que antes eran in-process (directas) ahora deben viajar por la red. Para que esto sea transparente:

-   Se utiliza un **Service Discovery** o un **Load Balancer** interno.
-   El cliente gRPC inyectado en un módulo debe apuntar a la dirección del microservicio externo en lugar de `127.0.0.1` (o usar la misma interfaz de cliente).

---

## 19. Contenerización y Despliegue en la Nube

El proyecto está preparado para ejecutarse en entornos de contenedores (Docker) y orquestadores (Kubernetes) de forma nativa, con un enfoque modular que permite evolucionar de monolito a microservicios sin fricciones.

### Dockerfile: Multi-Stage Build

Utilizamos un `Dockerfile` optimizado con dos etapas que soporta construcción dinámica de cualquier módulo:

1.  **Builder:** Compila el binario en una imagen de Go (Alpine). Usa `--build-arg TARGET={module}` para seleccionar qué construir.
2.  **Runner:** Una imagen ligera (`alpine:3.20`) que solo contiene el binario y los archivos de configuración necesarios.

**Todos los binarios se compilan en `/app/bin/` y se consolidan automáticamente.**

```bash
# Construir el servidor monolito
make docker-build
# Genera: modulith-server:latest

# Construir un módulo específico
make docker-build-module auth
# Genera: modulith-auth:latest

# Construir cualquier módulo
make docker-build-module payments
# Genera: modulith-payments:latest
```

### Helm Charts: Despliegue Flexible en Kubernetes

En `deployment/helm/modulith` se encuentra el chart estándar que soporta múltiples estrategias de despliegue.

#### Estrategia 1: Monolito (Fase Inicial)

Despliega todo como un solo deployment con autoscaling:

```bash
helm install modulith-server ./deployment/helm/modulith \
  --values ./deployment/helm/modulith/values-server.yaml \
  --namespace production
```

#### Estrategia 2: Híbrida (Transición)

Combina el monolito con módulos independientes para componentes que necesitan escalar de forma diferente:

```bash
# Servidor principal con módulos core
helm install modulith-server ./deployment/helm/modulith \
  --values values-server.yaml

# Módulo Auth separado (mayor demanda)
helm install modulith-auth ./deployment/helm/modulith \
  --values values-auth-module.yaml
```

#### Estrategia 3: Microservicios (Fase Avanzada)

Cada módulo como deployment independiente:

```bash
# Cada módulo con su propio ciclo de vida
helm install modulith-auth ./deployment/helm/modulith \
  --set deploymentType=module \
  --set moduleName=auth

helm install modulith-orders ./deployment/helm/modulith \
  --set deploymentType=module \
  --set moduleName=orders
```

#### Características del Helm Chart

-   **✅ Soporte Multi-Módulo:** Un solo chart para server y todos los módulos
-   **✅ Convención de Nombres:** Genera automáticamente `modulith-{module}:tag`
-   **✅ HPA y PDB:** Horizontal Pod Autoscaling y Pod Disruption Budgets configurables
-   **✅ Health Checks:** Liveness (`/healthz`) y Readiness (`/readyz`) probes
-   **✅ Secrets:** Gestión de configuración sensible (DB_DSN, JWT_SECRET)
-   **✅ Resource Limits:** Configuración de CPU y memoria por deployment

**Ver documentación completa en:** `deployment/helm/modulith/README.md`

## 20. Infraestructura como Código (IaC)

Manejamos la infraestructura base utilizando un enfoque modular con **OpenTofu** (Fork Open Source de Terraform) y **Terragrunt** para garantizar entornos consistentes y reproducibles.

**Nota:** La IaC gestiona la infraestructura base (VPC, EKS, RDS), mientras que los deployments de aplicaciones se manejan con Helm Charts (ver sección anterior).

### Estructura de Directorios

-   `deployment/opentofu/modules/`: Definición de componentes base (VPC, RDS, EKS).
-   `deployment/terragrunt/envs/`: Configuraciones específicas por entorno (`dev`, `prod`).

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
    -   **Complejidad Ciclomática y Cognitiva:** Evita funciones inmanejables.
    -   **Nivel de Anidación:** Máximo 5 niveles (linters `nestif`).
    -   **Documentación:** Todo elemento público **DEBE** tener comentarios de Godoc.
    -   **Seguridad:** Análisis estático con `gosec` en cada commit.
2.  **Validación de Configuración:** El cargador de configuración valida semánticamente las variables críticas antes de que la aplicación inicie (Fail-Fast).
3.  **Tests con Race Detection:** No se permite código con condiciones de carrera (`-race`).

### Cobertura de Tests

El proyecto incluye un sistema de reporting de cobertura avanzado:

```bash
# Reporte visual en terminal con estadísticas
make coverage-report

# Reporte HTML interactivo
make test-coverage
make coverage-html
```

El reporte de cobertura muestra:
-   📦 Cobertura por paquete con indicadores visuales (🟢 >95%, 🟡 80-95%, 🟠 60-80%)
-   📈 Estadísticas generales (paquetes con excelente/buena/media cobertura)
-   🎯 Top 10 archivos con mejor cobertura
-   ⚠️ Áreas que necesitan más tests

**Nota:** La cobertura total del proyecto excluye automáticamente código generado (`*.pb.go`, `sqlc`, etc.) para proporcionar métricas precisas del código escrito a mano.

### Estándares de Linting (Actualizado)

Hemos adoptado un set de reglas estricto para garantizar consistencia:

-   **wsl_v5 (Whitespace Linter):** Fuerza el uso de espacios en blanco para separar bloques lógicos (ej. antes de un `return` o `if`).
-   **wrapcheck:** Obliga a envolver errores externos con `fmt.Errorf("...: %w", err)` para mantener la cadena de trazabilidad.
-   **revive:** Reemplazo moderno de `golint` para estilo y convención de nombres.
-   **errcheck:** Verifica que todos los errores retornados sean manejados apropiadamente.
-   **goconst:** Detecta strings repetidos que deberían ser constantes.
-   **cyclop:** Limita la complejidad ciclomática de funciones (máximo 10).
-   **funlen:** Limita la longitud de funciones (máximo 60 líneas).
-   **package-comments:** Todos los paquetes deben tener documentación.

### Workflow de Linting (CRÍTICO)

**Regla de Oro:** NUNCA modificar `.golangci.yaml` para ignorar o suprimir errores. Siempre implementar fixes apropiados.

**Proceso Obligatorio:**

1.  **Ejecutar:** `make lint` después de CUALQUIER modificación a archivos `.go`.
2.  **Iterar:** Corregir todos los errores hasta alcanzar **0 issues**.
3.  **Fixes Apropiados:**
    -   `errcheck`: Agregar manejo de errores o asignar explícitamente a `_` si el error debe ser ignorado intencionalmente.
    -   `goconst`: Extraer strings repetidos a constantes con nombres descriptivos.
    -   `revive`: Renombrar parámetros no utilizados a `_`.
    -   `wsl_v5`: Agregar espacios en blanco apropiados entre declaraciones y control de flujo.
    -   `cyclop`: Reducir complejidad extrayendo lógica a funciones auxiliares.
    -   `funlen`: Dividir funciones largas en funciones más pequeñas y enfocadas.
4.  **Validación:** El CI/CD rechazará cualquier PR con errores de linting.

**Ejemplo de Refactoring (Complejidad):**

```go
// ❌ MAL: Función compleja con cyclomatic complexity > 10
func TestComplexFunction(t *testing.T) {
    // 50+ líneas de código con muchos if/else anidados
}

// ✅ BIEN: Extraer a funciones auxiliares
func TestComplexFunction(t *testing.T) {
    t.Run("case 1", func(t *testing.T) { testCase1(t) })
    t.Run("case 2", func(t *testing.T) { testCase2(t) })
}

func testCase1(t *testing.T) {
    t.Helper()
    // Lógica enfocada
}
```

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

-   **Interfaces:** Definidas en `internal/notifier/notifier.go` (`EmailProvider`, `SMSProvider`).
-   **Implementación Reactiva:** Un `notifier.Subscriber` escucha eventos globales (ej. `auth.magic_code_requested`) y despacha la notificación de forma **asíncrona** y **no bloqueante**.
-   **LogNotifier para Dev:** Imprime las notificaciones en los logs estructurados, permitiendo probar flujos como el "Magic Code" sin configurar APIs externas.
-   **Inyección y Registro:**
    -   El módulo (ej. `auth`) emite el evento al `Bus`.
    -   El `Subscriber` se registra al `Bus` en el `main.go`, garantizando que la lógica de entrega esté totalmente fuera del dominio del módulo.

---

## 24. Caching (`internal/cache`)

El sistema proporciona una abstracción de caché para session storage, rate limiting y caching general.

### Interface

```go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    Close() error
}
```

### Implementaciones

-   **MemoryCache:** Caché en memoria con limpieza automática de entradas expiradas. Ideal para desarrollo y despliegues single-instance.
-   **RedisCache:** Stub preparado para Redis. Agregar dependencia `github.com/redis/go-redis/v9` para usar.

### Ejemplo de Uso

```go
import "github.com/cmelgarejo/go-modulith-template/internal/cache"

// Crear caché en memoria
mc := cache.NewMemoryCache()

// Guardar valor con TTL
err := mc.Set(ctx, "session:123", sessionData, 30*time.Minute)

// Recuperar valor
data, err := mc.Get(ctx, "session:123")
if errors.Is(err, cache.ErrNotFound) {
    // Cache miss
}

// Helper para strings
sc := cache.NewStringCache(mc)
token, err := sc.Get(ctx, "token:456")
```

---

## 25. Resilience Patterns (`internal/resilience`)

Para proteger el sistema contra fallos en cascada, el template incluye patrones de resiliencia.

### Circuit Breaker

Implementa el patrón Circuit Breaker para servicios externos:

```go
import "github.com/cmelgarejo/go-modulith-template/internal/resilience"

// Crear circuit breaker
config := resilience.DefaultCircuitBreakerConfig()
config.MaxFailures = 5
config.Timeout = 30 * time.Second

cb := resilience.NewCircuitBreaker("payment-service", config)

// Usar para llamadas externas
err := cb.Execute(ctx, func(ctx context.Context) error {
    return paymentClient.Charge(ctx, amount)
})

if errors.Is(err, resilience.ErrCircuitOpen) {
    // Servicio está fallando, usar fallback
}
```

### Estados del Circuit Breaker

-   **Closed:** Operación normal, las llamadas pasan.
-   **Open:** Circuito abierto, rechaza llamadas inmediatamente.
-   **Half-Open:** Probando recuperación, permite algunas llamadas.

### Retry con Backoff Exponencial

```go
config := resilience.DefaultRetryConfig()
config.MaxAttempts = 3
config.InitialDelay = 100 * time.Millisecond

err := resilience.Retry(ctx, config, func(ctx context.Context) error {
    return externalService.Call(ctx)
})
```

---

## 26. Feature Flags (`internal/feature`)

Sistema de feature flags para rollouts graduales y A/B testing.

### Uso Básico

```go
import "github.com/cmelgarejo/go-modulith-template/internal/feature"

// Crear manager
fm := feature.NewInMemoryManager()

// Registrar flags
fm.RegisterFlag("new_checkout", "New checkout flow", false)
fm.RegisterFlag("dark_mode", "Enable dark mode", true)

// Verificar flag
if fm.IsEnabled(ctx, "new_checkout") {
    // Usar nuevo flujo
}
```

### Rollout por Porcentaje

```go
// Flag habilitado para 20% de usuarios
fm.SetFlag(ctx, feature.Flag{
    Name:       "experimental_feature",
    Enabled:    true,
    Percentage: 20,  // Solo 20% de usuarios
})

// Verificar para un usuario específico
featureCtx := feature.Context{
    UserID: userID,
    Email:  email,
}

if fm.IsEnabledFor(ctx, "experimental_feature", featureCtx) {
    // Usuario está en el 20%
}
```

### Reglas Condicionales

```go
fm.SetFlag(ctx, feature.Flag{
    Name:    "beta_feature",
    Enabled: true,
    Rules: []feature.Rule{
        {
            Attribute: "email",
            Operator:  "contains",
            Value:     "@beta.com",
        },
    },
})
```

---

## 27. Structured Error Codes

Los errores de dominio ahora incluyen códigos estables para clientes API.

### Formato de Respuesta

Los errores gRPC incluyen el código en el mensaje: `[ERROR_CODE] mensaje`

```
[USER_NOT_FOUND] user with email test@example.com not found
[AUTH_TOKEN_EXPIRED] session has expired, please login again
[VALIDATION_FAILED] email format is invalid
```

### Códigos Disponibles

| Código | Tipo | Descripción |
|--------|------|-------------|
| `NOT_FOUND` | NotFound | Recurso no encontrado |
| `ALREADY_EXISTS` | AlreadyExists | Recurso ya existe |
| `VALIDATION_FAILED` | Validation | Error de validación |
| `AUTH_REQUIRED` | Unauthorized | Autenticación requerida |
| `AUTH_TOKEN_EXPIRED` | Unauthorized | Token expirado |
| `FORBIDDEN` | Forbidden | Acceso denegado |
| `RATE_LIMITED` | Forbidden | Rate limit excedido |

### Uso

```go
import "github.com/cmelgarejo/go-modulith-template/internal/errors"

// Crear error con código específico
err := errors.WithCode(errors.CodeUserNotFound, "user not found")

// O usar helpers existentes (código se asigna automáticamente)
err := errors.NotFound("user not found")  // Código: NOT_FOUND

// Obtener código de un error
code := errors.GetErrorCode(err)  // "NOT_FOUND"
```

---

## 28. Request Logging Middleware

El middleware de logging registra todas las peticiones HTTP con información detallada.

### Información Registrada

-   Método HTTP y path
-   Status code y duración
-   Bytes escritos
-   Request ID (si disponible)
-   User-Agent y Remote Address

### Configuración

```go
config := middleware.LoggingConfig{
    SkipPaths: []string{"/healthz", "/readyz", "/metrics"},
    SlowRequestThreshold: 500 * time.Millisecond,
}

handler := middleware.Logging(config)(yourHandler)
```

### Niveles de Log

-   **INFO:** Peticiones exitosas (2xx, 3xx)
-   **WARN:** Errores de cliente (4xx) o peticiones lentas
-   **ERROR:** Errores de servidor (5xx)

---

## 29. Futuras Mejoras y Nota Final

Esta arquitectura favorece la seguridad en tiempo de compilación y la disciplina operativa. Go 1.24+ se elige por el soporte nativo de `slog`, mejoras en el `toolchain` y optimizaciones de performance que permiten un código más limpio y eficiente.
