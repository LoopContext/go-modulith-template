# WebSocket Real-Time Communication Guide

Este documento explica cómo usar la infraestructura de WebSocket para comunicación en tiempo real con clientes.

## 🎯 Arquitectura

```
┌─────────────┐
│Auth Module  │ ──┐
└─────────────┘   │
                  ├─→ Event Bus ─→ WebSocket Subscriber ─→ WebSocket Hub ─→ Clients
┌─────────────┐   │
│Order Module │ ──┘
└─────────────┘
```

**Principios clave:**
- ✅ Los módulos **NO saben** que existe WebSocket
- ✅ Los módulos solo publican eventos al bus (como siempre)
- ✅ El WebSocket subscriber escucha eventos y los reenvía automáticamente
- ✅ Desacoplamiento total = fácil testing y mantenimiento

## 📦 Componentes

### 1. Hub (`internal/websocket/hub.go`)

Gestiona todas las conexiones WebSocket activas.

**Responsabilidades:**
- Registrar/desregistrar clientes
- Broadcast a todos los clientes
- Envío dirigido a usuarios específicos
- Tracking de conexiones activas

**API Principal:**
```go
hub := websocket.NewHub(ctx)
go hub.Run()

// Broadcast a todos
hub.Broadcast(&websocket.Message{
    Type: "notification",
    Payload: data,
})

// Enviar a usuario específico
hub.SendToUser("user-123", &websocket.Message{
    Type: "alert",
    Payload: data,
})

// Métricas
connections := hub.GetTotalConnections()
users := hub.GetConnectedUsers()
```

### 2. Client (`internal/websocket/client.go`)

Representa una conexión WebSocket individual.

**Características:**
- ID único por conexión
- User ID para targeting
- Manejo de ping/pong automático
- Buffer de mensajes salientes
- Gestión de lifecycle

### 3. Subscriber (`internal/websocket/subscriber.go`)

Integra el event bus con WebSocket.

**Eventos suscritos por defecto:**
- `*.created` (user.created, order.created, etc.)
- `*.updated`
- `*.deleted`
- `notification.*`
- `alert.*`

**Extracción automática de user_id:**
```go
// Si el payload incluye user_id, el mensaje se envía solo a ese usuario
bus.Publish(ctx, events.Event{
    Name: "order.created",
    Payload: map[string]interface{}{
        "user_id": "user-123",  // ← Detectado automáticamente
        "order_id": "order-456",
        "amount": 100.50,
    },
})
```

### 4. Handler (`internal/websocket/handler.go`)

HTTP handler para upgrade de conexiones.

**Endpoint:** `/ws`

**Autenticación:**
- En desarrollo: `?user_id=xxx` en query string
- En producción: Extraer de contexto (set por middleware de auth)

## 🚀 Uso desde Módulos

### Ejemplo 1: Notificación Simple (Auth Module)

```go
// En modules/auth/internal/service/service.go

func (s *AuthService) CompleteLogin(ctx context.Context, req *pb.CompleteLoginRequest) (*pb.CompleteLoginResponse, error) {
    // ... lógica de login ...

    // Publicar evento (¡WebSocket lo maneja automáticamente!)
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

**Resultado:** Todos los clientes conectados reciben:
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

### Ejemplo 2: Notificación Dirigida (Order Module)

```go
// En un hipotético módulo de órdenes

func (s *OrderService) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
    order := s.repo.Create(...)

    // Evento con user_id → solo ese usuario lo recibe
    s.bus.Publish(ctx, events.Event{
        Name: "order.created",
        Payload: map[string]interface{}{
            "user_id": req.UserId,  // ← Solo este usuario recibe el mensaje
            "order_id": order.ID,
            "status": "pending",
            "amount": order.Amount,
        },
    })

    return &pb.CreateOrderResponse{...}, nil
}
```

**Resultado:** Solo el usuario `req.UserId` recibe el mensaje en sus conexiones WebSocket.

### Ejemplo 3: Eventos Personalizados

Si necesitas eventos que no están en los patrones por defecto:

```go
// En cmd/server/main.go (durante inicialización)

wsSubscriber := websocket.NewSubscriber(wsHub, ebus)
wsSubscriber.Subscribe()  // Patrones por defecto

// Agregar eventos personalizados
wsSubscriber.SubscribeToEvent("payment.processed")
wsSubscriber.SubscribeToEvent("inventory.low")
wsSubscriber.SubscribeToEvent("admin.alert")
```

## 🔌 Cliente JavaScript/TypeScript

### Conexión Básica

```javascript
// Desarrollo
const ws = new WebSocket('ws://localhost:8080/ws?user_id=user-123');

// Producción (con auth token en header no es posible con WebSocket estándar)
// Opción 1: Pasar token en query string (menos seguro)
const ws = new WebSocket(`wss://api.example.com/ws?token=${authToken}`);

// Opción 2: Usar protocolo personalizado (recomendado)
const ws = new WebSocket('wss://api.example.com/ws', [authToken]);
```

### Manejo de Mensajes

```javascript
ws.onopen = () => {
  console.log('WebSocket connected');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);

  switch(message.type) {
    case 'user.created':
      console.log('New user:', message.payload);
      break;

    case 'order.created':
      showNotification('New order created!', message.payload);
      break;

    case 'notification.sent':
      displayNotification(message.payload);
      break;

    default:
      console.log('Unknown message type:', message.type);
  }
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('WebSocket disconnected');
  // Implementar reconexión automática
  setTimeout(() => connectWebSocket(), 5000);
};
```

### React Hook Example

```typescript
import { useEffect, useState } from 'react';

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
      setMessages(prev => [...prev, message]);
    };

    return () => ws.close();
  }, [userId]);

  return { messages, isConnected };
}

// Uso en componente
function Dashboard() {
  const { messages, isConnected } = useWebSocket('user-123');

  return (
    <div>
      <div>Status: {isConnected ? '🟢 Connected' : '🔴 Disconnected'}</div>
      {messages.map((msg, i) => (
        <div key={i}>{msg.type}: {JSON.stringify(msg.payload)}</div>
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

## 🔐 Seguridad

### Autenticación (Producción)

**TODO:** Implementar en `internal/websocket/handler.go`:

```go
func getUserIDFromContext(r *http.Request) string {
    // Opción 1: Token en query string (menos seguro)
    token := r.URL.Query().Get("token")
    claims, err := verifier.VerifyToken(token)
    if err != nil {
        return ""
    }
    return claims.Subject

    // Opción 2: Cookie (recomendado)
    cookie, err := r.Cookie("auth_token")
    if err != nil {
        return ""
    }
    claims, err := verifier.VerifyToken(cookie.Value)
    if err != nil {
        return ""
    }
    return claims.Subject
}
```

### Origin Checking

**TODO:** Implementar en `internal/websocket/handler.go`:

```go
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        origin := r.Header.Get("Origin")
        allowedOrigins := []string{
            "https://app.example.com",
            "https://admin.example.com",
        }

        for _, allowed := range allowedOrigins {
            if origin == allowed {
                return true
            }
        }

        return false
    },
}
```

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

### Fase 2: Múltiples Instancias + Redis

```
┌─────────────┐    ┌─────────────┐
│  Server 1   │    │  Server 2   │
│  WS Hub     │    │  WS Hub     │
└──────┬──────┘    └──────┬──────┘
       │                  │
       └────────┬─────────┘
                ↓
         ┌─────────────┐
         │ Redis Pub/Sub│
         └─────────────┘
```

**Cambios necesarios:**
1. Reemplazar `internal/events/bus.go` con Redis Pub/Sub
2. Mantener el resto del código sin cambios
3. Escalar horizontalmente

## 📚 Referencias

- [Gorilla WebSocket Documentation](https://pkg.go.dev/github.com/gorilla/websocket)
- [WebSocket RFC 6455](https://datatracker.ietf.org/doc/html/rfc6455)
- [Event Bus Pattern](../internal/events/bus.go)
- [Architecture Guide](./MODULITH_ARCHITECTURE.md)

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

