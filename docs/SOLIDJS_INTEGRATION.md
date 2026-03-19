# SolidJS Integration Guide

This guide explains how to connect a SolidJS frontend application to the Go Modulith backend.

## Architecture Overview

The backend provides several entry points for frontend clients:
- **gRPC-Gateway (REST)**: Best for standard CRUD operations and when using lightweight HTTP clients.
- **GraphQL**: Best for complex data fetching and minimizing network requests.
- **WebSockets**: Used for real-time notifications and reactive updates.

## 1. Connecting via gRPC-Gateway (REST)

The gRPC-Gateway translates your gRPC services into standard RESTful JSON endpoints.

### Fetching Data
You can use the native `fetch` API or any HTTP client like `axios`.

```typescript
import { createResource } from "solid-js";

const fetchUsers = async () => {
  const response = await fetch("http://localhost:8080/v1/users");
  return response.json();
};

const [users] = createResource(fetchUsers);
```

### Sending Data (POST/PUT)
```typescript
const createUser = async (userData) => {
  const response = await fetch("http://localhost:8080/v1/users", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(userData),
  });
  return response.json();
};
```

## 2. Using GraphQL

The backend exposes a GraphQL endpoint at `/graphql`. For SolidJS, we recommend using `@urql/solid` or `solid-apollo`.

### Setup with URQL
```typescript
import { createClient, Provider } from "@urql/solid";

const client = createClient({
  url: "http://localhost:8080/graphql",
});

// Wrap your app with the Provider
<Provider value={client}>
  <App />
</Provider>
```

## 3. Real-time Updates via WebSockets

The WebSocket endpoint is available at `/ws`. It supports authentication via the `access_token` cookie.

```typescript
import { onMount, onCleanup } from "solid-js";

const setupWS = () => {
  const ws = new WebSocket("ws://localhost:8080/ws");

  ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log("Real-time update:", data);
  };

  onCleanup(() => ws.close());
};
```

## 4. Authentication (JWT Cookies)

The backend uses `HttpOnly` and `Secure` cookies for session management. 
- **Auto-propagation**: Browsers automatically send these cookies with requests to the same domain.
- **CORS**: If your frontend is on a different port (e.g., `:3000`), ensure `CORS_ALLOWED_ORIGINS` in `.env` includes your frontend URL and `credentials: "include"` is set in your fetch/GraphQL client.

### Fetch with Credentials
```typescript
const response = await fetch("http://localhost:8080/v1/auth/me", {
  credentials: "include",
});
```

## 5. TypeScript Type Safety

For full type safety, you can generate TypeScript clients from the `.proto` definitions using `buf`.

### Recommendation
Add the following to your `buf.gen.yaml` to generate Connect-ES or gRPC-Web clients:

```yaml
  - plugin: es
    out: gen/es
  - plugin: connect-es
    out: gen/es
```

Then install the dependencies in your SolidJS project:
```bash
pnpm add @bufbuild/protobuf @connectrpc/connect @connectrpc/connect-web
```

---
Refer to [WEBSOCKET_GUIDE.md](WEBSOCKET_GUIDE.md) for more details on event formats.
