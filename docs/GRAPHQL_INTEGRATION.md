# GraphQL Integration Guide (Optional)

Esta guía explica cómo agregar GraphQL opcionalmente a tu proyecto usando [gqlgen](https://github.com/99designs/gqlgen), manteniendo la arquitectura modular y desacoplada.

## 🎯 ¿Por qué GraphQL Opcional?

- ✅ **Flexibilidad**: Los clientes pueden elegir entre gRPC (eficiente) o GraphQL (flexible)
- ✅ **Frontend-friendly**: GraphQL es ideal para aplicaciones web/móviles
- ✅ **Subscriptions**: Integración nativa con WebSocket para tiempo real
- ✅ **Desacoplado**: Los módulos siguen usando el event bus, GraphQL solo expone

## 📦 Instalación Rápida

### Opción 1: Script Automático (Recomendado)

```bash
make add-graphql
```

Este comando:
- ✅ Instala gqlgen y dependencias
- ✅ Crea estructura base de GraphQL
- ✅ Genera código inicial
- ✅ Integra con el servidor existente
- ✅ Configura subscriptions con WebSocket

### Opción 2: Manual

```bash
# 1. Instalar gqlgen
go install github.com/99designs/gqlgen@latest

# 2. Inicializar GraphQL
make graphql-init

# 3. Generar código
make graphql-generate

# 4. Integrar en servidor
# (Ver sección de integración)
```

## 🏗️ Arquitectura

```
┌─────────────────┐
│  GraphQL API    │  ← Opcional, expone módulos
│  (gqlgen)       │
└────────┬────────┘
         │
         ↓
┌─────────────────┐
│  Modules        │  ← Sin cambios
│  (gRPC + Bus)   │
└─────────────────┘
```

**Principio clave:** GraphQL es una **capa de exposición**, no reemplaza gRPC ni el event bus.

## 📁 Estructura de Archivos

Después de la instalación:

```
go-modulith-template/
├── internal/
│   └── graphql/          # ← Nuevo (opcional)
│       ├── schema/
│       │   ├── schema.graphql    # Root schema (combina todos)
│       │   ├── auth.graphql      # Schema del módulo auth
│       │   ├── order.graphql     # Schema del módulo order
│       │   └── payment.graphql   # Schema del módulo payment
│       ├── resolver/
│       │   ├── resolver.go       # Root resolver
│       │   ├── auth.go           # Resolvers del módulo auth
│       │   ├── order.go          # Resolvers del módulo order
│       │   └── payment.go        # Resolvers del módulo payment
│       ├── generated/
│       │   └── (código generado)
│       └── server.go
├── gqlgen.yml           # ← Configuración gqlgen
└── cmd/server/main.go   # ← Integración opcional
```

## 🎯 Estrategia: Schema por Módulo

**Recomendamos schema por módulo** por las siguientes razones:

### ✅ Ventajas

1. **Desacoplamiento**: Cada módulo mantiene su propio schema
2. **Evolución Independiente**: Los módulos pueden cambiar sin afectar otros
3. **Escalabilidad**: Si un módulo se separa a microservicio, su schema va con él
4. **Mantenibilidad**: Más fácil encontrar y modificar código relacionado
5. **Alineado con Modulith**: Respeta la filosofía de módulos independientes

### 📝 Cómo Funciona

gqlgen **combina automáticamente** todos los schemas en `schema/*.graphql`:

```graphql
# schema/schema.graphql (root)
type Query {
  _empty: String
}

type Mutation {
  _empty: String
}

type Subscription {
  _empty: String
}

# schema/auth.graphql (módulo auth)
extend type Query {
  me: User
}

extend type Mutation {
  requestLogin(email: String): Boolean!
}

# schema/order.graphql (módulo order)
extend type Query {
  orders(userId: ID): [Order!]!
}

extend type Mutation {
  createOrder(input: CreateOrderInput!): Order!
}
```

**Resultado final combinado:**
```graphql
type Query {
  me: User              # ← De auth.graphql
  orders(userId: ID): [Order!]!  # ← De order.graphql
}
```

## 🔧 Configuración

### gqlgen.yml

```yaml
# Configuración base generada automáticamente
schema:
  - internal/graphql/schema/*.graphql

exec:
  filename: internal/graphql/generated/generated.go
  package: generated

model:
  filename: internal/graphql/generated/models_gen.go
  package: generated

resolver:
  layout: follow-schema
  dir: internal/graphql/resolver
  package: resolver
```

## 📝 Ejemplo: Exponer Módulo Auth

### 1. Crear Schema del Módulo

**Estrategia: Un archivo por módulo**

```graphql
# internal/graphql/schema/auth.graphql
# Schema específico del módulo auth

# Extender el Query root (definido en schema.graphql)
extend type Query {
  me: User
}

# Extender el Mutation root
extend type Mutation {
  requestLogin(email: String, phone: String): Boolean!
  completeLogin(email: String, phone: String, code: String!): AuthPayload!
}

# Extender el Subscription root
extend type Subscription {
  userEvents: UserEvent!
}

# Tipos específicos del módulo auth
type User {
  id: ID!
  email: String
  phone: String
  createdAt: String!
}

type AuthPayload {
  token: String!
  user: User!
}

type UserEvent {
  type: String!
  user: User!
}
```

**Nota:** Usa `extend type` para agregar campos a los tipos root definidos en `schema.graphql`.

### 2. Implementar Resolver del Módulo

**Estrategia: Un resolver por módulo**

```go
// internal/graphql/resolver/auth.go
// Resolvers específicos del módulo auth

package resolver

import (
    "context"

    "github.com/cmelgarejo/go-modulith-template/internal/graphql/generated"
    pb "github.com/cmelgarejo/go-modulith-template/gen/go/proto/auth/v1"
    "github.com/cmelgarejo/go-modulith-template/internal/events"
)

// authResolver contiene los resolvers del módulo auth
type authResolver struct {
    authClient pb.AuthServiceClient
    eventBus   *events.Bus
}

// Agregar al queryResolver en resolver.go:
func (r *queryResolver) Me(ctx context.Context) (*generated.User, error) {
    // Implementación aquí
}

func (r *authResolver) RequestLogin(ctx context.Context, email *string, phone *string) (bool, error) {
    req := &pb.RequestLoginRequest{}
    if email != nil {
        req.Email = *email
    }
    if phone != nil {
        req.Phone = *phone
    }

    _, err := r.authClient.RequestLogin(ctx, req)
    return err == nil, err
}

func (r *authResolver) CompleteLogin(ctx context.Context, email *string, phone *string, code string) (*generated.AuthPayload, error) {
    req := &pb.CompleteLoginRequest{
        Code: code,
    }
    if email != nil {
        req.Email = *email
    }
    if phone != nil {
        req.Phone = *phone
    }

    resp, err := r.authClient.CompleteLogin(ctx, req)
    if err != nil {
        return nil, err
    }

    return &generated.AuthPayload{
        Token: resp.Token,
        User: &generated.User{
            ID:    resp.User.Id,
            Email: &resp.User.Email,
            Phone: &resp.User.Phone,
        },
    }, nil
}
```

### 3. Subscription con Event Bus

```go
// internal/graphql/resolver/subscription.go

func (r *authResolver) UserEvents(ctx context.Context) (<-chan *generated.UserEvent, error) {
    ch := make(chan *generated.UserEvent)

    // Suscribirse al event bus
    handler := func(ctx context.Context, event events.Event) error {
        if event.Name == "user.created" || event.Name == "user.updated" {
            payload, ok := event.Payload.(map[string]interface{})
            if !ok {
                return nil
            }

            userID, _ := payload["user_id"].(string)

            ch <- &generated.UserEvent{
                Type: event.Name,
                User: &generated.User{
                    ID: userID,
                    // ... mapear más campos
                },
            }
        }
        return nil
    }

    r.eventBus.Subscribe("user.created", handler)
    r.eventBus.Subscribe("user.updated", handler)

    // Cleanup cuando el contexto se cancela
    go func() {
        <-ctx.Done()
        close(ch)
    }()

    return ch, nil
}
```

## 🔌 Integración con WebSocket Existente

gqlgen soporta WebSocket para subscriptions. Puedes usar el hub WebSocket existente:

```go
// internal/graphql/server.go

import (
    "github.com/99designs/gqlgen/graphql/handler"
    "github.com/99designs/gqlgen/graphql/handler/transport"
    "github.com/99designs/gqlgen/graphql/playground"
)

func NewGraphQLServer(schema generated.ExecutableSchema, wsHub *websocket.Hub) http.Handler {
    srv := handler.NewDefaultServer(schema)

    // Agregar transport WebSocket (usa el hub existente)
    srv.AddTransport(transport.Websocket{
        KeepAlivePingInterval: 10 * time.Second,
        // Opcional: usar el hub existente para gestión de conexiones
    })

    // Otros transports
    srv.AddTransport(transport.Options{})
    srv.AddTransport(transport.GET{})
    srv.AddTransport(transport.POST{})

    return srv
}
```

## 🚀 Integración en cmd/server/main.go

### Opción A: Siempre Habilitado (si GraphQL está instalado)

```go
// cmd/server/main.go

import (
    graphqlServer "github.com/cmelgarejo/go-modulith-template/internal/graphql"
)

func setupGateway(ctx context.Context, cfg *config.AppConfig, db *sql.DB, wsHub *websocket.Hub) (*http.ServeMux, *grpc.ClientConn, error) {
    // ... código existente ...

    mux := http.NewServeMux()
    setupHealthChecks(mux, db, wsHub)
    mux.Handle("/", rmux)

    // GraphQL (opcional, solo si existe)
    if graphqlHandler := graphqlServer.Setup(ctx, db, ebus, wsHub); graphqlHandler != nil {
        mux.Handle("/graphql", graphqlHandler)
        mux.Handle("/graphql/playground", playground.Handler("GraphQL Playground", "/graphql"))
        slog.Info("GraphQL enabled", "endpoint", "/graphql", "playground", "/graphql/playground")
    }

    // ... resto del código ...
}
```

### Opción B: Feature Flag

```go
// configs/server.yaml
graphql:
  enabled: true  # o false para deshabilitar

// cmd/server/main.go
if cfg.GraphQL.Enabled {
    graphqlHandler := graphqlServer.Setup(ctx, db, ebus, wsHub)
    mux.Handle("/graphql", graphqlHandler)
}
```

## 📊 Ejemplo Completo: Query + Mutation + Subscription

### Schema

```graphql
type Query {
  orders(userId: ID): [Order!]!
}

type Mutation {
  createOrder(input: CreateOrderInput!): Order!
}

type Subscription {
  orderUpdates: OrderUpdate!
}

type Order {
  id: ID!
  userId: ID!
  amount: Float!
  status: String!
}

input CreateOrderInput {
  userId: ID!
  amount: Float!
}

type OrderUpdate {
  order: Order!
  event: String!
}
```

### Resolver

```go
// internal/graphql/resolver/order.go

func (r *queryResolver) Orders(ctx context.Context, userID *string) ([]*generated.Order, error) {
    // Llamar al módulo order vía gRPC
    req := &pb.ListOrdersRequest{}
    if userID != nil {
        req.UserId = *userID
    }

    resp, err := r.orderClient.ListOrders(ctx, req)
    if err != nil {
        return nil, err
    }

    // Convertir a tipos GraphQL
    orders := make([]*generated.Order, len(resp.Orders))
    for i, o := range resp.Orders {
        orders[i] = &generated.Order{
            ID:     o.Id,
            UserID: o.UserId,
            Amount: float64(o.Amount),
            Status: o.Status,
        }
    }

    return orders, nil
}

func (r *mutationResolver) CreateOrder(ctx context.Context, input generated.CreateOrderInput) (*generated.Order, error) {
    req := &pb.CreateOrderRequest{
        UserId: input.UserID,
        Amount: int64(input.Amount),
    }

    resp, err := r.orderClient.CreateOrder(ctx, req)
    if err != nil {
        return nil, err
    }

    // El módulo publica evento automáticamente
    // La subscription lo captura

    return &generated.Order{
        ID:     resp.Order.Id,
        UserID: resp.Order.UserId,
        Amount: float64(resp.Order.Amount),
        Status: resp.Order.Status,
    }, nil
}

func (r *subscriptionResolver) OrderUpdates(ctx context.Context) (<-chan *generated.OrderUpdate, error) {
    ch := make(chan *generated.OrderUpdate)

    handler := func(ctx context.Context, event events.Event) error {
        if strings.HasPrefix(event.Name, "order.") {
            payload, _ := event.Payload.(map[string]interface{})

            ch <- &generated.OrderUpdate{
                Event: event.Name,
                Order: &generated.Order{
                    ID:     payload["order_id"].(string),
                    UserID: payload["user_id"].(string),
                    // ...
                },
            }
        }
        return nil
    }

    r.eventBus.Subscribe("order.created", handler)
    r.eventBus.Subscribe("order.updated", handler)

    go func() {
        <-ctx.Done()
        close(ch)
    }()

    return ch, nil
}
```

## 🧪 Testing

```go
// internal/graphql/resolver/auth_test.go

func TestRequestLogin(t *testing.T) {
    mockClient := &mockAuthClient{}
    resolver := &authResolver{authClient: mockClient}

    result, err := resolver.RequestLogin(context.Background(), stringPtr("test@example.com"), nil)

    assert.NoError(t, err)
    assert.True(t, result)
    assert.Equal(t, "test@example.com", mockClient.lastRequest.Email)
}
```

## 📚 Comandos Makefile

```bash
# Inicializar GraphQL en el proyecto
make graphql-init

# Generar código desde schema
make graphql-generate

# Validar schema
make graphql-validate

# Ver playground (requiere servidor corriendo)
# http://localhost:8080/graphql/playground
```

## 🔄 Flujo de Desarrollo

1. **Definir Schema** (`internal/graphql/schema/*.graphql`)
2. **Generar Código** (`make graphql-generate`)
3. **Implementar Resolvers** (`internal/graphql/resolver/*.go`)
4. **Conectar con Módulos** (vía gRPC clients)
5. **Agregar Subscriptions** (vía event bus)
6. **Probar en Playground** (`/graphql/playground`)

## 🎯 Ventajas de esta Arquitectura

### ✅ Desacoplamiento Total

- Los módulos **NO saben** que existe GraphQL
- GraphQL solo **expone** lo que ya existe
- Fácil de agregar/quitar sin afectar módulos

### ✅ Reutilización

- Mismo event bus para WebSocket y GraphQL subscriptions
- Mismo WebSocket hub para ambos
- Módulos siguen usando gRPC internamente

### ✅ Flexibilidad

- Clientes pueden elegir: gRPC (eficiente) o GraphQL (flexible)
- GraphQL opcional: no afecta si no lo usas
- Fácil de escalar horizontalmente

## 📖 Referencias

- [gqlgen Documentation](https://gqlgen.com/)
- [gqlgen Examples](https://github.com/99designs/gqlgen/tree/master/_examples)
- [GraphQL Subscriptions](https://gqlgen.com/reference/subscriptions/)
- [WebSocket Transport](https://gqlgen.com/reference/transports/)

## 🐛 Troubleshooting

### Error: "schema not found"

**Solución:** Ejecuta `make graphql-generate` después de crear/modificar schemas.

### Subscriptions no funcionan

**Verifica:**
1. WebSocket transport está agregado al handler
2. Resolver retorna un channel
3. Event bus está suscrito correctamente

### Tipos no coinciden

**Solución:** Regenera código con `make graphql-generate` después de cambios en schema.

---

**¿Listo para agregar GraphQL?** Ejecuta `make add-graphql` y sigue las instrucciones! 🚀

