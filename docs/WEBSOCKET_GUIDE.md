# WebSocket Real-Time Communication Guide

This document explains how to use the WebSocket infrastructure for real-time communication with clients.

## 🎯 Architecture

```bash
┌─────────────┐
│Auth Module  │ ──┐
└─────────────┘   │
                  ├─→ Event Bus ─→ WebSocket Subscriber ─→ WebSocket Hub ─→ Clients
┌─────────────┐   │
│Order Module │ ──┘
└─────────────┘
```

**Key principles:**

-   ✅ Modules **DO NOT know** WebSocket exists
-   ✅ Modules only publish events to the bus (as always)
-   ✅ WebSocket subscriber listens to events and automatically forwards them
-   ✅ Total decoupling = easy testing and maintenance

## 📦 Components

### 1. Hub (`internal/websocket/hub.go`)

Manages all active WebSocket connections.

**Responsibilities:**

-   Register/unregister clients
-   Broadcast to all clients
-   Directed sending to specific users
-   Tracking active connections

**Main API:**

```go
hub := websocket.NewHub(ctx)
go hub.Run()

// Broadcast to all
hub.Broadcast(&websocket.Message{
    Type: "notification",
    Payload: data,
})

// Send to specific user
hub.SendToUser("user-123", &websocket.Message{
    Type: "alert",
    Payload: data,
})

// Métricas
connections := hub.GetTotalConnections()
users := hub.GetConnectedUsers()
```

### 2. Client (`internal/websocket/client.go`)

Represents an individual WebSocket connection.

**Features:**

-   Unique ID per connection
-   User ID for targeting
-   Automatic ping/pong handling
-   Outgoing message buffer
-   Lifecycle management

### 3. Subscriber (`internal/websocket/subscriber.go`)

Integrates the event bus with WebSocket.

**Events subscribed by default:**

-   `*.created` (user.created, order.created, etc.)
-   `*.updated`
-   `*.deleted`
-   `notification.*`
-   `alert.*`

**Automatic user_id extraction:**

```go
// If payload includes user_id, message is sent only to that user
bus.Publish(ctx, events.Event{
    Name: "order.created",
    Payload: map[string]interface{}{
        "user_id": "user-123",  // ← Automatically detected
        "order_id": "order-456",
        "amount": 100.50,
    },
})
```

### 4. Handler (`internal/websocket/handler.go`)

HTTP handler for connection upgrade.

**Endpoint:** `/ws`

**Authentication:**

-   In development: `?user_id=xxx` in query string
-   In production: Extract from context (set by auth middleware)

## 🚀 Usage from Modules

### Example 1: Simple Notification (Auth Module)

```go
// In modules/auth/internal/service/service.go

func (s *AuthService) CompleteLogin(ctx context.Context, req *pb.CompleteLoginRequest) (*pb.CompleteLoginResponse, error) {
    // ... login logic ...

    // Publish event (WebSocket handles it automatically!)
    s.bus.Publish(ctx, events.Event{
        Name: "user.created",
        Payload: map[string]interface{}{
            "user_id": user.ID,
            "email": user.Email,
            "created_at": time.Now(),
        },
    })

    return &pb.CompleteLoginResponse{...}, nil
}
```

**Result:** All connected clients receive:

```json
{
    "type": "user.created",
    "payload": {
        "user_id": "user-123",
        "email": "user@example.com",
        "created_at": "2025-12-31T20:00:00Z"
    }
}
```

### Example 2: Directed Notification (Order Module)

```go
// In a hypothetical orders module

func (s *OrderService) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
    order := s.repo.Create(...)

    // Event with user_id → only that user receives it
    s.bus.Publish(ctx, events.Event{
        Name: "order.created",
        Payload: map[string]interface{}{
            "user_id": req.UserId,  // ← Only this user receives the message
            "order_id": order.ID,
            "status": "pending",
            "amount": order.Amount,
        },
    })

    return &pb.CreateOrderResponse{...}, nil
}
```

**Result:** Only user `req.UserId` receives the message on their WebSocket connections.

### Example 3: Custom Events

If you need events that aren't in the default patterns:

```go
// In cmd/server/setup/registry.go (during registry creation)

wsSubscriber := websocket.NewSubscriber(wsHub, ebus)
wsSubscriber.Subscribe()  // Default patterns

// Add custom events
wsSubscriber.SubscribeToEvent("payment.processed")
wsSubscriber.SubscribeToEvent("inventory.low")
wsSubscriber.SubscribeToEvent("admin.alert")
```

## 🔌 JavaScript/TypeScript Client

### Basic Connection

**Development Mode:**

```javascript
// Development - use user_id query parameter (dev only)
const ws = new WebSocket("ws://localhost:8000/ws?user_id=user-123");
```

**Production Mode - Option 1: Token in Query String**

```javascript
// Pass JWT token in query parameter
const authToken = localStorage.getItem("auth_token");
const ws = new WebSocket(`wss://api.example.com/ws?token=${authToken}`);
```

**Production Mode - Option 2: Cookie (Recommended for Web Apps)**

```javascript
// Cookie is automatically sent by browser if SameSite is not Strict
// Set cookie on login:
document.cookie = `auth_token=${token}; path=/; secure; samesite=lax`;

// Then connect (cookie sent automatically)
const ws = new WebSocket("wss://api.example.com/ws");
```

**Production Mode - Option 3: Custom Protocol (Advanced)**

```javascript
// Some WebSocket libraries support custom headers/protocols
// Check your library's documentation
const ws = new WebSocket("wss://api.example.com/ws", ["bearer", authToken]);
```

### Message Handling

```javascript
ws.onopen = () => {
    console.log("WebSocket connected");
};

ws.onmessage = (event) => {
    const message = JSON.parse(event.data);

    switch (message.type) {
        case "user.created":
            console.log("New user:", message.payload);
            break;

        case "order.created":
            showNotification("New order created!", message.payload);
            break;

        case "notification.sent":
            displayNotification(message.payload);
            break;

        default:
            console.log("Unknown message type:", message.type);
    }
};

ws.onerror = (error) => {
    console.error("WebSocket error:", error);
};

ws.onclose = () => {
    console.log("WebSocket disconnected");
    // Implement automatic reconnection
    setTimeout(() => connectWebSocket(), 5000);
};
```

### React Hook Example

```typescript
import { useEffect, useState } from "react";

interface WebSocketMessage {
    type: string;
    payload: any;
}

export function useWebSocket(userId: string) {
    const [messages, setMessages] = useState<WebSocketMessage[]>([]);
    const [isConnected, setIsConnected] = useState(false);

    useEffect(() => {
        const ws = new WebSocket(`ws://localhost:8080/ws?user_id=${userId}`);

        ws.onopen = () => setIsConnected(true);
        ws.onclose = () => setIsConnected(false);

        ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
            setMessages((prev) => [...prev, message]);
        };

        return () => ws.close();
    }, [userId]);

    return { messages, isConnected };
}

// Uso en componente
function Dashboard() {
    const { messages, isConnected } = useWebSocket("user-123");

    return (
        <div>
            <div>Status: {isConnected ? "🟢 Connected" : "🔴 Disconnected"}</div>
            {messages.map((msg, i) => (
                <div key={i}>
                    {msg.type}: {JSON.stringify(msg.payload)}
                </div>
            ))}
        </div>
    );
}
```

## 🧪 Testing

Los tests están en `internal/websocket/*_test.go`.

```bash
# Ejecutar tests de WebSocket
go test ./internal/websocket/... -v

# Con coverage
go test ./internal/websocket/... -cover
```

**Cobertura actual:** ~60% (hub y subscriber completamente testeados)

## 📊 Monitoreo

### Health Check Endpoint

```bash
# Check general
curl http://localhost:8080/healthz

# Check específico de WebSocket
curl http://localhost:8080/healthz/ws
```

**Respuesta:**

```json
{
    "status": "ok",
    "connections": 42,
    "users": 38
}
```

### Logs

El hub y subscriber loguean eventos importantes:

```
INFO WebSocket hub initialized
INFO Client registered client_id=abc-123 user_id=user-456 total_connections=1
INFO Client unregistered client_id=abc-123 user_id=user-456 total_connections=0
DEBUG Event sent to user event=order.created user_id=user-456
DEBUG Event broadcasted event=notification.sent
```

## 🔐 Security

### Authentication

The WebSocket handler now supports multiple authentication methods:

1. **JWT Token in Authorization Header** (Recommended for production)

    ```javascript
    // Note: Standard WebSocket API doesn't support custom headers
    // Use a library that supports it, or use one of the other methods
    ```

2. **JWT Token in Query Parameter**

    ```javascript
    const ws = new WebSocket(`wss://api.example.com/ws?token=${authToken}`);
    ```

3. **JWT Token in Cookie** (Recommended for web apps)

    ```javascript
    // Cookie is automatically sent by browser
    const ws = new WebSocket("wss://api.example.com/ws");
    ```

4. **Development Mode: user_id Query Parameter** (Dev only)
    ```javascript
    // Only works when ENV=dev
    const ws = new WebSocket("ws://localhost:8000/ws?user_id=test-user");
    ```

**Configuration:**

The handler automatically uses the JWT secret from your configuration. Authentication is enabled by default when a JWT secret is configured.

**Example:**

```go
// In cmd/server/setup/gateway.go - automatically configured
wsHandler := websocket.NewHandler(websocket.HandlerConfig{
    Hub:            wsHub,
    Verifier:      jwtVerifier,  // Created from JWT_SECRET
    AllowedOrigins: cfg.CORSAllowedOrigins,
    Env:            cfg.Env,
})
```

### Origin Checking

Origin checking is automatically configured based on your `CORSAllowedOrigins` setting:

**Configuration in `configs/server.yaml`:**

```yaml
cors_allowed_origins:
    - "https://app.example.com"
    - "https://admin.example.com"
    # Or use "*" to allow all origins (not recommended for production)
```

**Behavior:**

-   **Development mode (`ENV=dev`)**: If no origins configured, allows all origins
-   **Production mode (`ENV=prod`)**: If no origins configured, denies all connections (fail-secure)
-   **Wildcard (`*`)**: Allows all origins (use with caution)

**Example:**

```yaml
# configs/server.yaml
env: prod
cors_allowed_origins:
    - "https://app.example.com"
    - "https://admin.example.com"
```

The WebSocket handler will automatically reject connections from origins not in this list.

## 🚀 Escalabilidad

### Fase 1: Monolito (Actual)

```
┌─────────────────────┐
│  Server Process     │
│  ┌───────────────┐  │
│  │ WebSocket Hub │  │
│  │ (in-memory)   │  │
│  └───────────────┘  │
└─────────────────────┘
```

**Límite:** ~10,000 conexiones por instancia

### Fase 2: Múltiples Instancias + Valkey

```
┌─────────────┐    ┌─────────────┐
│  Server 1   │    │  Server 2   │
│  WS Hub     │    │  WS Hub     │
└──────┬──────┘    └──────┬──────┘
       │                  │
       └────────┬─────────┘
                ↓
         ┌─────────────┐
         │ Valkey Pub/Sub│
         └─────────────┘
```

**Cambios necesarios:**

1. Reemplazar `internal/events/bus.go` con Valkey Pub/Sub
2. Mantener el resto del código sin cambios
3. Escalar horizontalmente

## 📚 Referencias

-   [Gorilla WebSocket Documentation](https://pkg.go.dev/github.com/gorilla/websocket)
-   [WebSocket RFC 6455](https://datatracker.ietf.org/doc/html/rfc6455)
-   [Event Bus Pattern](../internal/events/bus.go)
-   [Architecture Guide](./MODULITH_ARCHITECTURE.md)

## 🐛 Troubleshooting

### Conexión se cierra inmediatamente

**Causa:** Falta de ping/pong
**Solución:** El cliente debe responder a pings automáticamente (navegadores lo hacen por defecto)

### Mensajes no llegan a usuarios específicos

**Causa:** Payload no incluye `user_id`
**Solución:** Agregar `user_id` al payload del evento:

```go
Payload: map[string]interface{}{
    "user_id": "user-123",  // ← Requerido para targeting
    // ... resto del payload
}
```

### Cliente no recibe ciertos eventos

**Causa:** Evento no está suscrito
**Solución:** Verificar que el evento esté en los patrones suscritos o agregarlo manualmente:

```go
wsSubscriber.SubscribeToEvent("tu.evento.personalizado")
```

---

**¿Preguntas?** Consulta el código en `internal/websocket/` o los tests para ejemplos completos.
