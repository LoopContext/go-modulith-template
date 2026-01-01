# Module Communication: Modulith vs Microservices

Este documento explica cómo funciona la comunicación entre módulos/servicios en ambos escenarios: **Modulith** (monolito modular) y **Microservicios** (servicios independientes).

## Resumen Ejecutivo

El template soporta dos modos de deployment:

1. **Modulith (Monolito Modular):** Todos los módulos en un solo proceso, comunicación in-process
2. **Microservicios:** Cada módulo como servicio independiente, comunicación vía red

**Punto clave:** El mismo código funciona en ambos escenarios. La diferencia está en cómo se despliega, no en cómo se escribe el código.

---

## Escenario 1: Modulith (Monolito Modular)

### Arquitectura

En el escenario modulith, todos los módulos se ejecutan en un **único proceso** (`cmd/server/main.go`).

```
┌─────────────────────────────────────────────────┐
│           Modulith Process (cmd/server)         │
│                                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │  Auth    │  │  Order   │  │ Payment  │       │
│  │  Module  │  │  Module  │  │ Module   │       │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘       │
│       │             │             │             │
│       └─────────────┼─────────────┘             │
│                     │                           │
│              ┌──────▼──────--┐                  │
│              │  gRPC Server  │                  │
│              │  (in-process) │                  │
│              └──────┬──────--┘                  │
│                     │                           │
│              ┌──────▼──────┐                    │
│              │ Event Bus   │                    │
│              │ (in-memory) │                    │
│              └─────────────┘                    │
│                                                 │
│  ┌─────────────────────────────────────────┐    │
│  │  Shared: DB, Redis, WebSocket Hub       │    │
│  └─────────────────────────────────────────┘    │
└─────────────────────────────────────────────────┘
```

### Comunicación gRPC (In-Process)

**Características:**

-   ✅ **Sin red:** Todas las llamadas son in-process (llamadas a función directas)
-   ✅ **Alta performance:** Sin overhead de serialización/deserialización de red
-   ✅ **Type-safe:** Mismos contratos Protobuf que en microservicios
-   ✅ **Mismo código:** Los módulos usan los mismos clientes gRPC generados

**Cómo funciona:**

1. **Registro de módulos:**

    ```go
    // cmd/server/main.go
    func registerModules(reg *registry.Registry) {
        reg.Register(auth.NewModule())
        reg.Register(order.NewModule())
        reg.Register(payment.NewModule())
    }
    ```

2. **Registro con gRPC Server:**

    ```go
    // Todos los módulos se registran en el mismo servidor gRPC
    grpcServer := grpc.NewServer(...)
    reg.RegisterGRPCAll(grpcServer)  // Registra todos los módulos
    ```

3. **Llamada entre módulos:**

    ```go
    // En el módulo Order, llamando al módulo Auth
    // modules/order/internal/service/order_service.go

    import (
        authv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/auth/v1"
        "google.golang.org/grpc"
    )

    func (s *OrderService) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
        // Crear cliente gRPC para Auth
        // En modulith, esto se conecta al servidor in-process
        conn, err := grpc.NewClient(
            "127.0.0.1:9050",  // gRPC server local
            grpc.WithTransportCredentials(insecure.NewCredentials()),
        )
        if err != nil {
            return nil, err
        }
        defer conn.Close()

        authClient := authv1.NewAuthServiceClient(conn)

        // Llamar al módulo Auth (in-process, sin red)
        user, err := authClient.GetUser(ctx, &authv1.GetUserRequest{
            UserId: req.UserId,
        })
        if err != nil {
            return nil, err
        }

        // Continuar con lógica de negocio...
    }
    ```

**Nota importante:** Aunque técnicamente se crea una conexión gRPC a `127.0.0.1`, el servidor gRPC está en el mismo proceso, por lo que la comunicación es **in-process** y muy eficiente.

### Event Bus (In-Memory)

**Características:**

-   ✅ **In-memory:** Eventos se distribuyen directamente en memoria
-   ✅ **Síncrono/Asíncrono:** Soporta ambos modos
-   ✅ **Thread-safe:** Usa `sync.RWMutex` para concurrencia

**Ejemplo:**

```go
// Módulo Auth publica evento
func (s *AuthService) CreateUser(ctx context.Context, req *authv1.CreateUserRequest) (*authv1.CreateUserResponse, error) {
    // ... crear usuario ...

    // Publicar evento (in-memory, instantáneo)
    s.bus.Publish(ctx, events.Event{
        Name: "user.created",
        Payload: map[string]any{
            "user_id": userID,
            "email":   req.Email,
        },
    })

    return response, nil
}

// Módulo Order se suscribe al evento
func (s *OrderService) Initialize(r *registry.Registry) error {
    // Suscribirse a eventos de usuario
    r.EventBus().Subscribe("user.created", func(ctx context.Context, event events.Event) error {
        // Procesar evento (ejecutado inmediatamente, in-process)
        userID := event.Payload.(map[string]any)["user_id"].(string)
        // Crear carrito para nuevo usuario, etc.
        return nil
    })

    return nil
}
```

### Ventajas del Modulith

1. **Performance:** Sin latencia de red entre módulos
2. **Simplicidad:** Un solo proceso para desplegar y monitorear
3. **Transacciones:** Puede usar transacciones de DB compartidas
4. **Debugging:** Más fácil de depurar (todo en un proceso)
5. **Testing:** Tests de integración más simples

### Desventajas del Modulith

1. **Escalado:** No puede escalar módulos independientemente
2. **Acoplamiento:** Todos los módulos deben desplegarse juntos
3. **Tecnología:** Todos los módulos deben usar el mismo stack (Go)

---

## Escenario 2: Microservicios

### Arquitectura

En el escenario de microservicios, cada módulo se ejecuta como un **proceso independiente** (`cmd/{module}/main.go`).

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Auth Service│     │ Order Service│     │Payment Service│
│  (cmd/auth)  │     │(cmd/order)   │     │(cmd/payment) │
│              │     │              │     │              │
│  ┌────────┐  │     │  ┌────────┐  │     │  ┌────────┐  │
│  │ gRPC   │  │     │  │ gRPC   │  │     │  │ gRPC   │  │
│  │ Server │  │     │  │ Server │  │     │  │ Server │  │
│  └───┬────┘  │     │  └───┬────┘  │     │  └───┬────┘  │
│      │       │     │      │       │     │      │       │
└──────┼───────┘     └──────┼───────┘     └──────┼───────┘
       │                    │                     │
       │                    │                     │
       └────────────────────┼─────────────────────┘
                            │
                    ┌───────▼───────┐
                    │  Load Balancer │
                    │   / Service    │
                    │    Discovery   │
                    └───────┬───────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
  ┌─────▼─────┐      ┌─────▼─────┐      ┌─────▼─────┐
  │ PostgreSQL │      │   Redis   │      │  Event Bus │
  │  (Shared)  │      │  (Shared) │      │ (Kafka/...) │
  └────────────┘      └───────────┘      └────────────┘
```

### Comunicación gRPC (Network)

**Características:**

-   ✅ **Vía red:** Llamadas gRPC sobre TCP/IP
-   ✅ **Service Discovery:** Requiere descubrimiento de servicios (Kubernetes DNS, Consul, etc.)
-   ✅ **Resiliencia:** Necesita circuit breakers, retries, timeouts
-   ✅ **Observabilidad:** Trazabilidad distribuida esencial

**Cómo funciona:**

1. **Deployment independiente:**

    ```bash
    # Cada módulo se despliega como servicio separado
    kubectl apply -f deployment/helm/modulith/values-auth-module.yaml
    kubectl apply -f deployment/helm/modulith/values-order-module.yaml
    kubectl apply -f deployment/helm/modulith/values-payment-module.yaml
    ```

2. **Service Discovery:**

    ```yaml
    # Kubernetes Service para Auth
    apiVersion: v1
    kind: Service
    metadata:
        name: auth-service
    spec:
        selector:
            app: auth
        ports:
            - port: 9050
              targetPort: 9050
    ```

3. **Llamada entre servicios:**

    ```go
    // En el módulo Order, llamando al servicio Auth
    // modules/order/internal/service/order_service.go

    func (s *OrderService) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
        // Crear cliente gRPC para Auth (vía red)
        // En microservicios, esto se conecta a otro servicio
        conn, err := grpc.NewClient(
            "auth-service:9050",  // Service discovery (Kubernetes DNS)
            grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),  // TLS en prod
        )
        if err != nil {
            return nil, err
        }
        defer conn.Close()

        authClient := authv1.NewAuthServiceClient(conn)

        // Llamar al servicio Auth (vía red, con latencia)
        user, err := authClient.GetUser(ctx, &authv1.GetUserRequest{
            UserId: req.UserId,
        })
        if err != nil {
            // Manejar errores de red, timeouts, etc.
            return nil, err
        }

        // Continuar con lógica de negocio...
    }
    ```

**Consideraciones importantes:**

1. **Resiliencia:**

    ```go
    // Usar circuit breaker y retries
    import "github.com/cmelgarejo/go-modulith-template/internal/resilience"

    cb := resilience.NewCircuitBreaker("auth-service", ...)
    retry := resilience.NewRetry(3, time.Second)

    user, err := retry.Do(ctx, func() (interface{}, error) {
        return cb.Execute(func() (interface{}, error) {
            return authClient.GetUser(ctx, req)
        })
    })
    ```

2. **Timeouts:**

    ```go
    // Context con timeout para llamadas de red
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    user, err := authClient.GetUser(ctx, req)
    ```

3. **TLS/Security:**
    ```go
    // En producción, usar TLS
    creds := credentials.NewTLS(&tls.Config{
        ServerName: "auth-service",
    })
    conn, err := grpc.NewClient("auth-service:9050", grpc.WithTransportCredentials(creds))
    ```

### Event Bus (Distributed)

**Características:**

-   ✅ **Distribuido:** Eventos viajan por red (Kafka, RabbitMQ, Redis Pub/Sub)
-   ✅ **Asíncrono:** Eventos se procesan de forma asíncrona
-   ✅ **Durabilidad:** Eventos persisten en el broker
-   ✅ **Escalable:** Múltiples consumidores pueden procesar eventos

**Implementación:**

El template incluye una interfaz para event bus distribuido:

```go
// internal/events/distributed.go
// Implementación futura con Kafka, RabbitMQ, etc.

// Ejemplo con Kafka (futuro)
type KafkaEventBus struct {
    producer *kafka.Writer
    consumer *kafka.Reader
}

func (k *KafkaEventBus) Publish(ctx context.Context, event Event) error {
    // Publicar a Kafka topic
    return k.producer.WriteMessages(ctx, kafka.Message{
        Key:   []byte(event.Name),
        Value: marshalEvent(event),
    })
}
```

**Ejemplo de uso:**

```go
// Módulo Auth (servicio independiente) publica evento
func (s *AuthService) CreateUser(ctx context.Context, req *authv1.CreateUserRequest) (*authv1.CreateUserResponse, error) {
    // ... crear usuario ...

    // Publicar evento (distribuido, vía Kafka/RabbitMQ)
    s.bus.Publish(ctx, events.Event{
        Name: "user.created",
        Payload: map[string]any{
            "user_id": userID,
            "email":   req.Email,
        },
    })

    return response, nil
}

// Módulo Order (servicio independiente) consume evento
func (s *OrderService) StartEventConsumer(ctx context.Context) error {
    // Suscribirse a eventos (vía Kafka consumer, etc.)
    s.bus.Subscribe("user.created", func(ctx context.Context, event events.Event) error {
        // Procesar evento (asíncrono, puede tardar)
        userID := event.Payload.(map[string]any)["user_id"].(string)
        // Crear carrito para nuevo usuario
        return nil
    })

    return nil
}
```

### Ventajas de Microservicios

1. **Escalado independiente:** Cada servicio escala según necesidad
2. **Despliegue independiente:** Cambios en un servicio no afectan otros
3. **Tecnología:** Cada servicio puede usar diferentes stacks (si es necesario)
4. **Aislamiento:** Fallos en un servicio no afectan otros
5. **Equipos:** Equipos pueden trabajar independientemente

### Desventajas de Microservicios

1. **Complejidad:** Más servicios para gestionar y monitorear
2. **Latencia:** Llamadas de red añaden latencia
3. **Resiliencia:** Necesita circuit breakers, retries, timeouts
4. **Testing:** Tests de integración más complejos
5. **Transacciones:** No puede usar transacciones de DB compartidas fácilmente

---

## Comparación: Modulith vs Microservicios

| Aspecto               | Modulith                | Microservicios             |
| --------------------- | ----------------------- | -------------------------- |
| **Comunicación gRPC** | In-process (sin red)    | Network (vía red)          |
| **Performance**       | Muy alta (sin latencia) | Menor (latencia de red)    |
| **Escalado**          | Todo junto              | Independiente por servicio |
| **Despliegue**        | Un solo artefacto       | Múltiples artefactos       |
| **Event Bus**         | In-memory               | Distribuido (Kafka, etc.)  |
| **Transacciones**     | DB compartida (fácil)   | Saga pattern (complejo)    |
| **Testing**           | Más simple              | Más complejo               |
| **Debugging**         | Un proceso              | Múltiples procesos         |
| **Service Discovery** | No necesario            | Requerido                  |
| **Resiliencia**       | Menos crítico           | Crítico (circuit breakers) |

---

## Patrón de Comunicación Recomendado

### Para Modulith

**gRPC In-Process:**

```go
// Helper para crear cliente gRPC (reutilizable)
func newAuthClient(grpcAddr string) (authv1.AuthServiceClient, error) {
    conn, err := grpc.NewClient(
        grpcAddr,  // "127.0.0.1:9050" en modulith
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        return nil, err
    }
    return authv1.NewAuthServiceClient(conn), nil
}
```

**Event Bus In-Memory:**

```go
// Los eventos se procesan inmediatamente, in-process
bus.Publish(ctx, events.Event{...})
```

### Para Microservicios

**gRPC Network con Resiliencia:**

```go
// Helper con circuit breaker y retries
func newAuthClientWithResilience(serviceAddr string) (authv1.AuthServiceClient, error) {
    conn, err := grpc.NewClient(
        serviceAddr,  // "auth-service:9050" en microservicios
        grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
    )
    if err != nil {
        return nil, err
    }

    // Envolver con circuit breaker
    // (implementación futura)
    return authv1.NewAuthServiceClient(conn), nil
}
```

**Event Bus Distribuido:**

```go
// Los eventos viajan por red, se procesan asíncronamente
// Requiere implementación con Kafka, RabbitMQ, etc.
```

---

## Ejemplo Completo: Order Module llamando a Auth Module

### Escenario Modulith

```go
// modules/order/internal/service/order_service.go

package service

import (
    "context"
    authv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/auth/v1"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

type OrderService struct {
    repo repository.Repository
    bus  *events.Bus
    // Cliente gRPC para Auth (in-process)
    authClient authv1.AuthServiceClient
}

func NewOrderService(repo repository.Repository, bus *events.Bus, grpcAddr string) (*OrderService, error) {
    // Crear cliente gRPC (in-process en modulith)
    conn, err := grpc.NewClient(
        grpcAddr,  // "127.0.0.1:9050"
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        return nil, err
    }

    return &OrderService{
        repo:       repo,
        bus:        bus,
        authClient: authv1.NewAuthServiceClient(conn),
    }, nil
}

func (s *OrderService) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
    // Llamar a Auth (in-process, sin latencia de red)
    user, err := s.authClient.GetUser(ctx, &authv1.GetUserRequest{
        UserId: req.UserId,
    })
    if err != nil {
        return nil, err
    }

    // Validar que el usuario existe y está activo
    if user.Status != "active" {
        return nil, status.Error(codes.PermissionDenied, "user is not active")
    }

    // Crear orden...
    order, err := s.repo.CreateOrder(ctx, ...)

    // Publicar evento (in-memory, instantáneo)
    s.bus.Publish(ctx, events.Event{
        Name: "order.created",
        Payload: map[string]any{
            "order_id": order.ID,
            "user_id":  req.UserId,
        },
    })

    return &orderv1.CreateOrderResponse{OrderId: order.ID}, nil
}
```

### Escenario Microservicios

```go
// modules/order/internal/service/order_service.go
// (Mismo código, diferente configuración)

package service

import (
    "context"
    "crypto/tls"
    authv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/auth/v1"
    "github.com/cmelgarejo/go-modulith-template/internal/resilience"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
)

type OrderService struct {
    repo       repository.Repository
    bus        *events.Bus
    authClient authv1.AuthServiceClient
    cb         *resilience.CircuitBreaker  // Circuit breaker para resiliencia
}

func NewOrderService(repo repository.Repository, bus *events.Bus, authServiceAddr string) (*OrderService, error) {
    // Crear cliente gRPC (vía red en microservicios)
    creds := credentials.NewTLS(&tls.Config{
        ServerName: "auth-service",
    })

    conn, err := grpc.NewClient(
        authServiceAddr,  // "auth-service:9050" (Kubernetes DNS)
        grpc.WithTransportCredentials(creds),
    )
    if err != nil {
        return nil, err
    }

    return &OrderService{
        repo:       repo,
        bus:        bus,
        authClient: authv1.NewAuthServiceClient(conn),
        cb:         resilience.NewCircuitBreaker("auth-service", ...),
    }, nil
}

func (s *OrderService) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
    // Llamar a Auth (vía red, con circuit breaker)
    var user *authv1.User
    err := s.cb.Execute(func() error {
        ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
        defer cancel()

        var err error
        user, err = s.authClient.GetUser(ctx, &authv1.GetUserRequest{
            UserId: req.UserId,
        })
        return err
    })
    if err != nil {
        return nil, err
    }

    // Validar que el usuario existe y está activo
    if user.Status != "active" {
        return nil, status.Error(codes.PermissionDenied, "user is not active")
    }

    // Crear orden...
    order, err := s.repo.CreateOrder(ctx, ...)

    // Publicar evento (distribuido, vía Kafka/RabbitMQ)
    s.bus.Publish(ctx, events.Event{
        Name: "order.created",
        Payload: map[string]any{
            "order_id": order.ID,
            "user_id":  req.UserId,
        },
    })

    return &orderv1.CreateOrderResponse{OrderId: order.ID}, nil
}
```

**Nota:** El código es muy similar. La diferencia principal está en:

-   La dirección del servicio (localhost vs service discovery)
-   Las credenciales (insecure vs TLS)
-   La adición de circuit breakers y timeouts

---

## Configuración por Escenario

### Modulith Configuration

```yaml
# configs/server.yaml
env: prod
http_port: 8080
grpc_port: 9050

# Todos los módulos usan la misma DB
db_dsn: postgres://user:pass@db:5432/modulith
```

**Deployment:**

```bash
# Un solo proceso
make build
./bin/server
```

### Microservicios Configuration

```yaml
# configs/auth.yaml (Auth Service)
env: prod
http_port: 8080
grpc_port: 9050
db_dsn: postgres://user:pass@db:5432/modulith

# configs/order.yaml (Order Service)
env: prod
http_port: 8080
grpc_port: 9050
db_dsn: postgres://user:pass@db:5432/modulith

# Service discovery config
auth_service_addr: auth-service:9050
```

**Deployment:**

```bash
# Múltiples procesos
make build-module auth
make build-module order
./bin/auth &
./bin/order &
```

---

## Migración de Modulith a Microservicios

### Paso 1: Identificar Módulos a Extraer

Decide qué módulos extraer basándote en:

-   Escalado independiente necesario
-   Equipos diferentes
-   Diferentes SLAs
-   Diferentes tecnologías (futuro)

### Paso 2: Crear Entrypoint de Microservicio

```bash
# Ya existe para auth
make new-module order  # Crea cmd/order/main.go
```

### Paso 3: Actualizar Configuración

```yaml
# configs/order.yaml
# Agregar service discovery
auth_service_addr: auth-service:9050 # En lugar de 127.0.0.1:9050
```

### Paso 4: Agregar Resiliencia

```go
// Agregar circuit breakers, retries, timeouts
// Usar internal/resilience
```

### Paso 5: Migrar Event Bus

```go
// Cambiar de in-memory a distribuido (Kafka, RabbitMQ)
// Implementar internal/events/distributed.go
```

### Paso 6: Deploy

```bash
# Deploy como servicios independientes
helm install auth-service ./deployment/helm/modulith --values values-auth-module.yaml
helm install order-service ./deployment/helm/modulith --values values-order-module.yaml
```

---

## Mejores Prácticas

### 1. Abstraer la Creación de Clientes

```go
// internal/clients/auth_client.go
package clients

type AuthClient interface {
    GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.User, error)
}

type authClientImpl struct {
    client authv1.AuthServiceClient
}

func NewAuthClient(addr string, tls bool) (AuthClient, error) {
    var creds credentials.TransportCredentials
    if tls {
        creds = credentials.NewTLS(&tls.Config{ServerName: "auth-service"})
    } else {
        creds = insecure.NewCredentials()
    }

    conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(creds))
    if err != nil {
        return nil, err
    }

    return &authClientImpl{
        client: authv1.NewAuthServiceClient(conn),
    }, nil
}
```

### 2. Usar Context Propagation

```go
// Siempre pasar el contexto para trazabilidad
func (s *OrderService) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) {
    // El contexto se propaga automáticamente con trace_id/span_id
    user, err := s.authClient.GetUser(ctx, &authv1.GetUserRequest{...})
}
```

### 3. Manejar Errores Consistentemente

```go
// Usar el sistema de errores del template
import "github.com/cmelgarejo/go-modulith-template/internal/errors"

if err != nil {
    // Mapear errores de red a errores de dominio
    if status.Code(err) == codes.Unavailable {
        return nil, errors.Unavailable("auth service unavailable", err)
    }
    return nil, errors.Internal("failed to get user", err)
}
```

### 4. Eventos para Desacoplamiento

```go
// Preferir eventos para comunicación asíncrona
// En lugar de llamadas síncronas cuando sea posible

// ❌ Evitar: Llamada síncrona para notificación
authClient.SendNotification(ctx, ...)

// ✅ Preferir: Evento asíncrono
bus.Publish(ctx, events.Event{
    Name: "order.created",
    Payload: ...,
})
// El módulo de notificaciones se suscribe y envía
```

---

## Diagramas de Flujo

### Modulith: Request Flow

```
Client Request
    │
    ▼
HTTP Gateway (grpc-gateway)
    │
    ▼
gRPC Server (in-process)
    │
    ├─→ Auth Module (in-process call)
    │       │
    │       └─→ Order Module (in-process call)
    │               │
    │               └─→ Payment Module (in-process call)
    │
    ▼
Response (sin latencia de red entre módulos)
```

### Microservicios: Request Flow

```
Client Request
    │
    ▼
Load Balancer
    │
    ├─→ Order Service (Pod 1)
    │       │
    │       ├─→ Auth Service (network call, ~1-5ms)
    │       │       │
    │       │       └─→ Response
    │       │
    │       ├─→ Payment Service (network call, ~1-5ms)
    │       │       │
    │       │       └─→ Response
    │       │
    │       └─→ Response
    │
    └─→ Order Service (Pod 2)  # Load balanced
```

### Event Flow: Modulith vs Microservicios

**Modulith:**

```
Auth Module
    │
    │ Publish("user.created")
    ▼
Event Bus (in-memory)
    │
    ├─→ Order Module (handler ejecutado inmediatamente)
    └─→ Notification Module (handler ejecutado inmediatamente)
```

**Microservicios:**

```
Auth Service
    │
    │ Publish("user.created")
    ▼
Kafka Topic: "user.created"
    │
    ├─→ Order Service Consumer (procesa asíncronamente)
    └─→ Notification Service Consumer (procesa asíncronamente)
```

---

## Checklist de Implementación

### Para Modulith

-   [ ] Todos los módulos registrados en `cmd/server/main.go`
-   [ ] Clientes gRPC apuntan a `127.0.0.1:9050`
-   [ ] Event bus in-memory configurado
-   [ ] Un solo proceso para desplegar

### Para Microservicios

-   [ ] Cada módulo tiene su propio `cmd/{module}/main.go`
-   [ ] Service discovery configurado (Kubernetes DNS, etc.)
-   [ ] Clientes gRPC apuntan a service names (`auth-service:9050`)
-   [ ] TLS habilitado para comunicación entre servicios
-   [ ] Circuit breakers implementados
-   [ ] Retries y timeouts configurados
-   [ ] Event bus distribuido (Kafka, RabbitMQ, etc.)
-   [ ] Trazabilidad distribuida (OpenTelemetry)

---

## Referencias

-   [Modulith Architecture](MODULITH_ARCHITECTURE.md) - Arquitectura completa
-   [Event Bus Guide](MODULITH_ARCHITECTURE.md#12-events) - Sistema de eventos
-   [Deployment Guide](DEPLOYMENT_SYNC.md) - Guía de despliegue
-   [gRPC Best Practices](https://grpc.io/docs/guides/best-practices/)

---

**Última actualización:** Enero 2026
**Mantenido por:** Go Modulith Template Team
