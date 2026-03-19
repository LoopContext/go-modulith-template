# Module Communication: Modulith vs Microservices

This document explains how communication between modules/services works in both scenarios: **Modulith** (modular monolith) and **Microservices** (independent services).

## Executive Summary

The template supports two deployment modes:

1. **Modulith (Modular Monolith):** All modules in a single process, in-process communication
2. **Microservices:** Each module as an independent service, network communication

**Key point:** The same code works in both scenarios. The difference is in how it's deployed, not in how the code is written.

---

## Scenario 1: Modulith (Modular Monolith)

### Architecture

In the modulith scenario, all modules run in a **single process** (`cmd/server/main.go`), with setup logic organized in `cmd/server/setup/`.

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

### gRPC Communication (In-Process)

**Features:**

-   ✅ **No network:** All calls are in-process (direct function calls)
-   ✅ **High performance:** No network serialization/deserialization overhead
-   ✅ **Type-safe:** Same Protobuf contracts as in microservices
-   ✅ **Same code:** Modules use the same generated gRPC clients

**How it works:**

1. **Module registration:**

    ```go
    // cmd/server/setup/registry.go
    func RegisterModules(reg *registry.Registry) {
        reg.Register(auth.NewModule())
        reg.Register(order.NewModule())
        reg.Register(payment.NewModule())
    }

    // Called from cmd/server/main.go
    setup.RegisterModules(reg)
    ```

2. **Register with gRPC Server:**

    ```go
    // All modules register with the same gRPC server
    grpcServer := grpc.NewServer(...)
    reg.RegisterGRPCAll(grpcServer)  // Registers all modules
    ```

3. **Inter-module call:**

    ```go
    // In the Order module, calling the Auth module
    // modules/order/internal/service/order_service.go

    import (
        authv1 "github.com/cmelgarejo/go-modulith-template/gen/go/proto/auth/v1"
        "google.golang.org/grpc"
    )

    func (s *OrderService) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
        // Create gRPC client for Auth
        // In modulith, this connects to the in-process server
        // Note: Default gRPC port is 9000 (configurable via GRPC_PORT or configs/server.yaml)
        conn, err := grpc.NewClient(
            "127.0.0.1:9000",  // Local gRPC server (default, configurable)
            grpc.WithTransportCredentials(insecure.NewCredentials()),
        )
        if err != nil {
            return nil, err
        }
        defer conn.Close()

        authClient := authv1.NewAuthServiceClient(conn)

        // Call Auth module (in-process, no network)
        user, err := authClient.GetUser(ctx, &authv1.GetUserRequest{
            UserId: req.UserId,
        })
        if err != nil {
            return nil, err
        }

        // Continue with business logic...
    }
    ```

**Important note:** Although technically a gRPC connection is created to `127.0.0.1`, the gRPC server is in the same process, so communication is **in-process** and very efficient.

### Event Bus (In-Memory)

**Features:**

-   ✅ **In-memory:** Events are distributed directly in memory
-   ✅ **Synchronous/Asynchronous:** Supports both modes
-   ✅ **Thread-safe:** Uses `sync.RWMutex` for concurrency

**Example:**

```go
// Auth module publishes event
func (s *AuthService) CreateUser(ctx context.Context, req *authv1.CreateUserRequest) (*authv1.CreateUserResponse, error) {
    // ... create user ...

    // Publish event (in-memory, instant)
    s.bus.Publish(ctx, events.Event{
        Name: "user.created",
        Payload: map[string]any{
            "user_id": userID,
            "email":   req.Email,
        },
    })

    return response, nil
}

// Order module subscribes to event
func (s *OrderService) Initialize(r *registry.Registry) error {
    // Subscribe to user events
    r.EventBus().Subscribe("user.created", func(ctx context.Context, event events.Event) error {
        // Process event (executed immediately, in-process)
        userID := event.Payload.(map[string]any)["user_id"].(string)
        // Create cart for new user, etc.
        return nil
    })

    return nil
}
```

### Modulith Advantages

1. **Performance:** No network latency between modules
2. **Simplicity:** Single process to deploy and monitor
3. **Transactions:** Can use shared DB transactions
4. **Debugging:** Easier to debug (everything in one process)
5. **Testing:** Simpler integration tests

### Modulith Disadvantages

1. **Scaling:** Cannot scale modules independently
2. **Coupling:** All modules must be deployed together
3. **Technology:** All modules must use the same stack (Go)

---

## Scenario 2: Microservices

### Architecture

In the microservices scenario, each module runs as an **independent process** (`cmd/{module}/main.go`).

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

### gRPC Communication (Network)

**Features:**

-   ✅ **Via network:** gRPC calls over TCP/IP
-   ✅ **Service Discovery:** Requires service discovery (Kubernetes DNS, Consul, etc.)
-   ✅ **Resilience:** Needs circuit breakers, retries, timeouts
-   ✅ **Observability:** Distributed tracing essential

**How it works:**

1. **Independent deployment:**

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
            - port: 9000  # Default gRPC port (configurable)
              targetPort: 9000
    ```

3. **Inter-service call:**

    ```go
    // In the Order module, calling the Auth service
    // modules/order/internal/service/order_service.go

    func (s *OrderService) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
        // Create gRPC client for Auth (via network)
        // In microservices, this connects to another service
        conn, err := grpc.NewClient(
            "auth-service:9000",  // Service discovery (Kubernetes DNS, port configurable)
            grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),  // TLS in prod
        )
        if err != nil {
            return nil, err
        }
        defer conn.Close()

        authClient := authv1.NewAuthServiceClient(conn)

        // Call Auth service (via network, with latency)
        user, err := authClient.GetUser(ctx, &authv1.GetUserRequest{
            UserId: req.UserId,
        })
        if err != nil {
            // Handle network errors, timeouts, etc.
            return nil, err
        }

        // Continue with business logic...
    }
    ```

**Important considerations:**

1. **Resilience:**

    ```go
    // Use circuit breaker and retries
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
    // Context with timeout for network calls
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    user, err := authClient.GetUser(ctx, req)
    ```

3. **TLS/Security:**
    ```go
    // In production, use TLS
    creds := credentials.NewTLS(&tls.Config{
        ServerName: "auth-service",
    })
    conn, err := grpc.NewClient("auth-service:9000", grpc.WithTransportCredentials(creds))  // Port configurable
    ```

### Event Bus (Distributed)

**Features:**

-   ✅ **Distributed:** Events travel over network (Kafka, RabbitMQ, Redis Pub/Sub)
-   ✅ **Asynchronous:** Events are processed asynchronously
-   ✅ **Durability:** Events persist in the broker
-   ✅ **Scalable:** Multiple consumers can process events

**For complete implementation guide, see [Distributed Events Documentation](DISTRIBUTED_EVENTS.md).**

**Implementation:**

The template includes an interface for distributed event bus:

```go
// internal/events/distributed.go
// Future implementation with Kafka, RabbitMQ, etc.

// Example with Kafka (future)
type KafkaEventBus struct {
    producer *kafka.Writer
    consumer *kafka.Reader
}

func (k *KafkaEventBus) Publish(ctx context.Context, event Event) error {
    // Publish to Kafka topic
    return k.producer.WriteMessages(ctx, kafka.Message{
        Key:   []byte(event.Name),
        Value: marshalEvent(event),
    })
}
```

**Usage example:**

```go
// Auth module (independent service) publishes event
func (s *AuthService) CreateUser(ctx context.Context, req *authv1.CreateUserRequest) (*authv1.CreateUserResponse, error) {
    // ... create user ...

    // Publish event (distributed, via Kafka/RabbitMQ)
    s.bus.Publish(ctx, events.Event{
        Name: "user.created",
        Payload: map[string]any{
            "user_id": userID,
            "email":   req.Email,
        },
    })

    return response, nil
}

// Order module (independent service) consumes event
func (s *OrderService) StartEventConsumer(ctx context.Context) error {
    // Subscribe to events (via Kafka consumer, etc.)
    s.bus.Subscribe("user.created", func(ctx context.Context, event events.Event) error {
        // Process event (asynchronous, may take time)
        userID := event.Payload.(map[string]any)["user_id"].(string)
        // Create cart for new user
        return nil
    })

    return nil
}
```

### Microservices Advantages

1. **Independent scaling:** Each service scales according to need
2. **Independent deployment:** Changes in one service don't affect others
3. **Technology:** Each service can use different stacks (if necessary)
4. **Isolation:** Failures in one service don't affect others
5. **Teams:** Teams can work independently

### Microservices Disadvantages

1. **Complexity:** More services to manage and monitor
2. **Latency:** Network calls add latency
3. **Resilience:** Needs circuit breakers, retries, timeouts
4. **Testing:** More complex integration tests
5. **Transactions:** Cannot easily use shared DB transactions

---

## Comparison: Modulith vs Microservices

| Aspect                 | Modulith                | Microservices               |
| ---------------------- | ----------------------- | --------------------------- |
| **gRPC Communication** | In-process (no network) | Network (via network)       |
| **Performance**        | Very high (no latency)  | Lower (network latency)     |
| **Scaling**            | All together            | Independent per service     |
| **Deployment**         | Single artifact         | Multiple artifacts          |
| **Event Bus**          | In-memory               | Distributed (Kafka, etc.)   |
| **Transactions**       | Shared DB (easy)        | Saga pattern (complex)      |
| **Testing**            | Simpler                 | More complex                |
| **Debugging**          | Single process          | Multiple processes          |
| **Service Discovery**  | Not required            | Required                    |
| **Resilience**         | Less critical           | Critical (circuit breakers) |

---

## Recommended Communication Pattern

### For Modulith

**gRPC In-Process:**

```go
// Helper to create gRPC client (reusable)
func newAuthClient(grpcAddr string) (authv1.AuthServiceClient, error) {
    conn, err := grpc.NewClient(
        grpcAddr,  // "127.0.0.1:9000" in modulith (default, configurable)
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
// Events are processed immediately, in-process
bus.Publish(ctx, events.Event{...})
```

### For Microservices

**gRPC Network with Resilience:**

```go
// Helper con circuit breaker y retries
func newAuthClientWithResilience(serviceAddr string) (authv1.AuthServiceClient, error) {
    conn, err := grpc.NewClient(
        serviceAddr,  // "auth-service:9000" en microservicios (port configurable)
        grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
    )
    if err != nil {
        return nil, err
    }

    // Wrap with circuit breaker
    // (implementación futura)
    return authv1.NewAuthServiceClient(conn), nil
}
```

**Distributed Event Bus:**

```go
// Events travel over network, processed asynchronously
// Requires implementation with Kafka, RabbitMQ, etc.
```

---

## Complete Example: Order Module calling Auth Module

### Modulith Scenario

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
        grpcAddr,  // "127.0.0.1:9000" (default, configurable)
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
    // Call Auth (in-process, no network latency)
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

### Microservices Scenario

```go
// modules/order/internal/service/order_service.go
// (Same code, different configuration)

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
    cb         *resilience.CircuitBreaker  // Circuit breaker for resilience
}

func NewOrderService(repo repository.Repository, bus *events.Bus, authServiceAddr string) (*OrderService, error) {
    // Create gRPC client (via network in microservices)
    creds := credentials.NewTLS(&tls.Config{
        ServerName: "auth-service",
    })

    conn, err := grpc.NewClient(
        authServiceAddr,  // "auth-service:9000" (Kubernetes DNS, port configurable)
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
    // Call Auth (via network, with circuit breaker)
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

    // Validate that user exists and is active
    if user.Status != "active" {
        return nil, status.Error(codes.PermissionDenied, "user is not active")
    }

    // Create order...
    order, err := s.repo.CreateOrder(ctx, ...)

    // Publish event (distributed, via Kafka/RabbitMQ)
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

**Note:** The code is very similar. The main difference is in:

-   Service address (localhost vs service discovery)
-   Credentials (insecure vs TLS)
-   Addition of circuit breakers and timeouts

---

## Configuration by Scenario

### Modulith Configuration

```yaml
# configs/server.yaml
env: prod
http_port: 8080
grpc_port: 9000  # Default port (configurable)

# All modules use the same DB
db_dsn: postgres://user:pass@db:5432/modulith
```

**Deployment:**

```bash
# Single process
just build
./bin/server
```

### Microservices Configuration

```yaml
# configs/auth.yaml (Auth Service)
env: prod
http_port: 8080
grpc_port: 9000  # Default port (configurable)
db_dsn: postgres://user:pass@db:5432/modulith

# configs/order.yaml (Order Service)
env: prod
http_port: 8080
grpc_port: 9000  # Default port (configurable)
db_dsn: postgres://user:pass@db:5432/modulith

# Service discovery config
auth_service_addr: auth-service:9050
```

**Deployment:**

```bash
# Multiple processes
just build-module auth
just build-module order
./bin/auth &
./bin/order &
```

---

## Migration from Modulith to Microservices

### Step 1: Identify Modules to Extract

Decide which modules to extract based on:

-   Independent scaling needed
-   Different teams
-   Different SLAs
-   Different technologies (future)

### Step 2: Create Microservice Entrypoint

```bash
# Already exists for auth
just new-module order  # Creates cmd/order/main.go
```

### Step 3: Update Configuration

```yaml
# configs/order.yaml
# Add service discovery
auth_service_addr: auth-service:9000 # Instead of 127.0.0.1:9000 (port configurable)
```

### Step 4: Add Resilience

```go
// Add circuit breakers, retries, timeouts
// Use internal/resilience
```

### Step 5: Migrate Event Bus

```go
// Change from in-memory to distributed (Kafka, RabbitMQ)
// Implement internal/events/distributed.go
```

### Step 6: Deploy

```bash
# Deploy as independent services
helm install auth-service ./deployment/helm/modulith --values values-auth-module.yaml
helm install order-service ./deployment/helm/modulith --values values-order-module.yaml
```

---

## Best Practices

### 1. Abstract Client Creation

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

### 2. Use Context Propagation

```go
// Always pass context for traceability
func (s *OrderService) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) {
    // Context automatically propagates with trace_id/span_id
    user, err := s.authClient.GetUser(ctx, &authv1.GetUserRequest{...})
}
```

### 3. Handle Errors Consistently

```go
// Use the template's error system
import "github.com/cmelgarejo/go-modulith-template/internal/errors"

if err != nil {
    // Map network errors to domain errors
    if status.Code(err) == codes.Unavailable {
        return nil, errors.Unavailable("auth service unavailable", err)
    }
    return nil, errors.Internal("failed to get user", err)
}
```

### 4. Events for Decoupling

```go
// Prefer events for asynchronous communication
// Instead of synchronous calls when possible

// ❌ Avoid: Synchronous call for notification
authClient.SendNotification(ctx, ...)

// ✅ Prefer: Asynchronous event
bus.Publish(ctx, events.Event{
    Name: "order.created",
    Payload: ...,
})
// The notification module subscribes and sends
```

---

## Flow Diagrams

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
Response (no network latency between modules)
```

### Microservices: Request Flow

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

### Event Flow: Modulith vs Microservices

**Modulith:**

```
Auth Module
    │
    │ Publish("user.created")
    ▼
Event Bus (in-memory)
    │
    ├─→ Order Module (handler executed immediately)
    └─→ Notification Module (handler executed immediately)
```

**Microservices:**

```
Auth Service
    │
    │ Publish("user.created")
    ▼
Kafka Topic: "user.created"
    │
    ├─→ Order Service Consumer (processes asynchronously)
    └─→ Notification Service Consumer (processes asynchronously)
```

---

## Implementation Checklist

### For Modulith

-   [ ] All modules registered in `cmd/server/setup/registry.go` via `RegisterModules()`
-   [ ] gRPC clients point to `127.0.0.1:9000` (default, configurable via `GRPC_PORT`)
-   [ ] In-memory event bus configured
-   [ ] Single process to deploy

### For Microservices

-   [ ] Each module has its own `cmd/{module}/main.go`
-   [ ] Service discovery configured (Kubernetes DNS, etc.)
-   [ ] gRPC clients point to service names (`auth-service:9000`, port configurable)
-   [ ] TLS enabled for inter-service communication
-   [ ] Circuit breakers implemented
-   [ ] Retries and timeouts configured
-   [ ] Distributed event bus (Kafka, RabbitMQ, etc.)
-   [ ] Distributed tracing (OpenTelemetry)

---

## References

-   [Modulith Architecture](MODULITH_ARCHITECTURE.md) - Complete architecture
-   [Event Bus Guide](MODULITH_ARCHITECTURE.md#12-events) - Sistema de eventos
-   [Deployment Guide](DEPLOYMENT_SYNC.md) - Guía de despliegue
-   [gRPC Best Practices](https://grpc.io/docs/guides/best-practices/)

---

**Last updated:** January 2026
**Maintained by:** Go Modulith Template Team
