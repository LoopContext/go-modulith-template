# Proposal: Frontend Integration with Go Templates + HTMX

## Executive Summary

This proposal documents the architectural decision to include the web frontend in the same modulith repository, using **Go Templates** for server-side rendering and **HTMX** for dynamic interactions. This architecture maintains consistency with the modular monolith design and significantly simplifies deployment and development.

## Justification

### Integration Advantages

1. **Coherent Architecture**: The project already exposes HTTP via `grpc-gateway`; serving HTML with templates is a natural extension of the same HTTP server.

2. **Single Binary**: Simplifies deployment by eliminating the need to manage multiple services, builds, and separate configurations.

3. **Code Co-location**: Templates and handlers live alongside business code, facilitating development and maintenance.

4. **HTMX as Perfect Complement**:
   - No build step required (just HTML + inline JavaScript)
   - Small and efficient requests
   - Server already handles HTTP robustly
   - Enables dynamic interactions without SPA complexity

5. **Consistency with Modulith**: Maintains the "everything in one place" principle while allowing internal modular separation.

6. **Simplified Development**:
   - Hot reload with Air works for templates too
   - No synchronization between repositories
   - Simpler testing (everything in one process)

### Real-Time Communication

The backend already includes integrated **WebSocket** (`/ws`) for bidirectional real-time communication:

- **JWT Authentication**: WebSocket connections are protected
- **Event Bus Integration**: Backend events automatically propagate to clients
- **Broadcast and Directed Messages**: Support for global and user-specific notifications

**See complete documentation:** `docs/WEBSOCKET_GUIDE.md`

### Flexible API (Optional)

If you need a more flexible API than REST, the backend supports optional **GraphQL**:

- **Schema per Module**: Each module defines its own GraphQL schema
- **Subscriptions**: Real-time subscriptions via WebSocket
- **Event Bus Integration**: Subscriptions can listen to internal events

**See complete documentation:** `docs/GRAPHQL_INTEGRATION.md`

### Ideal Use Cases

- **Dashboards administrativos**: Interfaces de gestión internas
- **Aplicaciones B2B**: Portales para clientes empresariales
- **Herramientas internas**: Paneles de control, configuraciones
- **Applications with low UI traffic**: Where simplicity outweighs the need for extreme optimization

## Proposed Structure

```
project/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   ├── handlers/          # New: HTTP handlers for HTML
│   │   ├── web.go        # Main handlers that render templates
│   │   └── middleware.go # Middleware for web (CSRF, auth, etc.)
│   └── templates/         # New: HTML templates
│       ├── base.html      # Base template with common layout
│       ├── pages/         # Complete pages
│       │   ├── login.html
│       │   ├── dashboard.html
│       │   └── profile.html
│       └── components/    # Reusable components (partials)
│           ├── navbar.html
│           ├── footer.html
│           └── form.html
├── static/                # New: Static assets
│   ├── css/
│   │   └── main.css
│   ├── js/
│   │   └── htmx.min.js   # HTMX from CDN or local
│   └── images/
│       └── logo.svg
├── modules/
│   └── auth/
│       └── internal/
│           └── handlers/  # Auth module-specific handlers
│               └── web.go # Login, logout, etc.
└── configs/
    └── server.yaml
```

## Technical Implementation

### 1. Handler Structure

Web handlers can be organized in two ways:

#### Option A: Centralized Handlers (Recommended for start)

```go
// internal/handlers/web.go
package handlers

import (
    "html/template"
    "net/http"
    "embed"
)

//go:embed templates/*
var templatesFS embed.FS

type WebHandler struct {
    tmpl *template.Template
    // Dependencies: auth service, etc.
}

func NewWebHandler() (*WebHandler, error) {
    tmpl, err := template.ParseFS(templatesFS, "templates/**/*.html")
    if err != nil {
        return nil, err
    }
    return &WebHandler{tmpl: tmpl}, nil
}

func (h *WebHandler) Home(w http.ResponseWriter, r *http.Request) {
    data := map[string]interface{}{
        "Title": "Dashboard",
        "User":  getUserFromContext(r.Context()),
    }
    h.tmpl.ExecuteTemplate(w, "pages/dashboard.html", data)
}
```

#### Option B: Handlers per Module (Recommended for scaling)

```go
// modules/auth/internal/handlers/web.go
package handlers

type AuthWebHandler struct {
    tmpl    *template.Template
    authSvc *service.AuthService
}

func (h *AuthWebHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
    h.tmpl.ExecuteTemplate(w, "pages/login.html", nil)
}

func (h *AuthWebHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
    // Process login via HTMX
    email := r.FormValue("email")
    // ... login logic

    // HTMX response (HTML fragment or redirect)
    w.Header().Set("HX-Redirect", "/dashboard")
    w.WriteHeader(http.StatusOK)
}
```

### 2. Integration in main.go

```go
// In cmd/server/main.go, after setupGateway:

// Setup web handlers (HTML + HTMX)
webHandler, err := handlers.NewWebHandler()
if err != nil {
    slog.Error("Failed to setup web handlers", "error", err)
    return
}

webMux := http.NewServeMux()
webMux.HandleFunc("/", webHandler.Home)
webMux.HandleFunc("/login", webHandler.LoginPage)
webMux.HandleFunc("/api/auth/login", webHandler.HandleLogin) // HTMX endpoint

// Serve static files
webMux.Handle("/static/", http.StripPrefix("/static/",
    http.FileServer(http.Dir("static"))))

// Mount web routes on HTTP server
mux := http.NewServeMux()
mux.Handle("/", webMux)              // Web routes (HTML)
mux.Handle("/v1/", grpcGatewayMux)   // gRPC Gateway (REST API)
mux.Handle("/metrics", metricsHandler)
mux.HandleFunc("/healthz", healthz)
mux.HandleFunc("/readyz", readyz)
```

### 3. Base Template with HTMX

```html
<!-- internal/templates/base.html -->
<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Modulith App</title>
    <link rel="stylesheet" href="/static/css/main.css">
    <script src="/static/js/htmx.min.js"></script>
</head>
<body>
    {{template "navbar" .}}

    <main>
        {{block "content" .}}{{end}}
    </main>

    {{template "footer" .}}
</body>
</html>
```

### 4. Ejemplo de Página con HTMX

```html
<!-- internal/templates/pages/login.html -->
{{define "content"}}
<div class="login-container">
    <form hx-post="/api/auth/login"
          hx-target="#login-result"
          hx-swap="innerHTML">
        <input type="email" name="email" placeholder="Email" required>
        <button type="submit">Send Magic Code</button>
    </form>

    <div id="login-result"></div>
</div>
{{end}}
```

## Routing and Middleware

### Suggested Route Structure

```
/                    → Home/Dashboard (HTML)
/login               → Login page (HTML)
/api/auth/login      → HTMX endpoint for login
/api/auth/complete   → HTMX endpoint to complete login
/static/*            → Static assets (CSS, JS, images)
/v1/*                → gRPC Gateway (REST API for external clients)
/metrics             → Prometheus metrics
/healthz             → Liveness probe
/readyz              → Readiness probe
```

### Recommended Middleware

```go
// internal/handlers/middleware.go
package handlers

func RequireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify JWT from cookie or header
        // If not authenticated, redirect to /login
        next.ServeHTTP(w, r)
    })
}

func CSRFProtection(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Validate CSRF token for POST/PUT/DELETE
        next.ServeHTTP(w, r)
    })
}
```

## Static Assets

### Option 1: FileServer (Development)

```go
webMux.Handle("/static/", http.StripPrefix("/static/",
    http.FileServer(http.Dir("static"))))
```

### Option 2: embed (Production - Recommended)

```go
//go:embed static/*
var staticFS embed.FS

webMux.Handle("/static/", http.StripPrefix("/static/",
    http.FileServer(http.FS(staticFS))))
```

**embed advantage**: All frontend is packaged in the binary, eliminating external file dependencies in production.

## Security Considerations

1. **CSRF Protection**: Implement CSRF tokens for all forms
2. **XSS Prevention**: Use `html/template` (not `text/template`) which automatically escapes
3. **Content Security Policy**: Appropriate CSP headers
4. **Authentication**: HTTP-only cookies for JWT tokens
5. **HTTPS**: Mandatory in production

## Future Scalability

### Gradual Separation

If you need to separate frontend/backend in the future:

1. **Phase 1 (Current)**: Everything in one binary
2. **Phase 2**: Extract web handlers to an independent module
3. **Phase 3**: Move to separate microservice maintaining the same API

The modulith design facilitates this transition without drastic changes.

### Scaling Alternatives

- **CDN for Assets**: Serve CSS/JS from CDN in production
- **Caching**: Appropriate cache headers for static assets
- **SSR Caching**: Cache rendered templates if necessary

## Testing

### Unit Tests for Handlers

```go
func TestWebHandler_Home(t *testing.T) {
    handler := setupTestHandler(t)

    req := httptest.NewRequest("GET", "/", nil)
    w := httptest.NewRecorder()

    handler.Home(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
    assert.Contains(t, w.Body.String(), "Dashboard")
}
```

### Integration Tests

```go
func TestLoginFlow(t *testing.T) {
    // Complete test: login page → HTMX request → redirect
    // Similar to gRPC tests but with HTTP
}
```

## Hot Reload

Air is already configured to monitor `.html` files. Template changes will automatically reflect when the server restarts.

**Recommendation**: Add `"html"` to `include_ext` in `.air.toml` (already included according to diff).

## Advantages vs Disadvantages

### Advantages ✅

- Operational simplicity (single binary)
- Faster development (no synchronization between repos)
- Lower infrastructure complexity
- Co-location of related code
- HTMX is lightweight and efficient
- Easy to test (everything in one process)

### Disadvantages ⚠️

- Frontend/backend teams work in the same repo (can also be an advantage)
- If you need a complex SPA, this architecture is not ideal
- Static assets increase binary size (mitigated with CDN)
- Less flexibility to deploy frontend/backend separately (mitigated by modulith design)

## Final Recommendation

**✅ YES, include frontend in the same repository** if:

- You use Go Templates + HTMX (traditional server architecture)
- You prioritize simplicity over strict separation
- The team is small or the same team works on both
- You don't need a complex SPA with React/Vue/etc.

**❌ DON'T include** if:

- You need a complex SPA with modern framework (React, Vue, Svelte)
- You have completely separate frontend/backend teams
- You require deploying frontend/backend independently from the start

## Next Steps

1. Create folder structure (`internal/templates/`, `static/`)
2. Implement base handler with template parsing
3. Integrate in `cmd/server/main.go`
4. Create base template and first example page
5. Configure authentication middleware for web
6. Document HTMX patterns in the project

---

**Proposal Date**: 2025-01-XX
**Estado**: Propuesto
**Decisión**: Pendiente de aprobación

