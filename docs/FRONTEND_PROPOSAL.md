# Propuesta: Integración de Frontend con Go Templates + HTMX

## Resumen Ejecutivo

Esta propuesta documenta la decisión arquitectónica de incluir el frontend web en el mismo repositorio del modulith, utilizando **Go Templates** para renderizado del servidor y **HTMX** para interacciones dinámicas. Esta arquitectura mantiene la coherencia con el diseño monolítico modular y simplifica significativamente el despliegue y desarrollo.

## Justificación

### Ventajas de la Integración

1. **Arquitectura Coherente**: El proyecto ya expone HTTP vía `grpc-gateway`; servir HTML con templates es una extensión natural del mismo servidor HTTP.

2. **Un Solo Binario**: Simplifica el despliegue eliminando la necesidad de gestionar múltiples servicios, builds y configuraciones separadas.

3. **Co-locación de Código**: Templates y handlers viven junto al código de negocio, facilitando el desarrollo y mantenimiento.

4. **HTMX como Complemento Perfecto**:
   - No requiere build step (solo HTML + JavaScript inline)
   - Peticiones pequeñas y eficientes
   - El servidor ya maneja HTTP de forma robusta
   - Permite interacciones dinámicas sin complejidad de SPA

5. **Consistencia con Modulith**: Mantiene el principio de "todo en un lugar" mientras permite separación modular interna.

6. **Desarrollo Simplificado**:
   - Hot reload con Air funciona para templates también
   - No hay sincronización entre repositorios
   - Testing más simple (todo en un proceso)

### Casos de Uso Ideales

- **Dashboards administrativos**: Interfaces de gestión internas
- **Aplicaciones B2B**: Portales para clientes empresariales
- **Herramientas internas**: Paneles de control, configuraciones
- **Aplicaciones con bajo tráfico de UI**: Donde la simplicidad supera la necesidad de optimización extrema

## Estructura Propuesta

```
proyecto/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   ├── handlers/          # Nuevo: HTTP handlers para HTML
│   │   ├── web.go        # Handlers principales que renderizan templates
│   │   └── middleware.go # Middleware para web (CSRF, auth, etc.)
│   └── templates/         # Nuevo: Templates HTML
│       ├── base.html      # Template base con layout común
│       ├── pages/         # Páginas completas
│       │   ├── login.html
│       │   ├── dashboard.html
│       │   └── profile.html
│       └── components/    # Componentes reutilizables (partials)
│           ├── navbar.html
│           ├── footer.html
│           └── form.html
├── static/                # Nuevo: Assets estáticos
│   ├── css/
│   │   └── main.css
│   ├── js/
│   │   └── htmx.min.js   # HTMX desde CDN o local
│   └── images/
│       └── logo.svg
├── modules/
│   └── auth/
│       └── internal/
│           └── handlers/  # Handlers específicos del módulo auth
│               └── web.go # Login, logout, etc.
└── configs/
    └── server.yaml
```

## Implementación Técnica

### 1. Estructura de Handlers

Los handlers web pueden organizarse de dos formas:

#### Opción A: Handlers Centralizados (Recomendado para inicio)

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
    // Dependencias: auth service, etc.
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

#### Opción B: Handlers por Módulo (Recomendado para escalar)

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
    // Procesar login vía HTMX
    email := r.FormValue("email")
    // ... lógica de login

    // Respuesta HTMX (fragmento HTML o redirect)
    w.Header().Set("HX-Redirect", "/dashboard")
    w.WriteHeader(http.StatusOK)
}
```

### 2. Integración en main.go

```go
// En cmd/server/main.go, después de setupGateway:

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
mux.Handle("/v1/", grpcGatewayMux)   // gRPC Gateway (API REST)
mux.Handle("/metrics", metricsHandler)
mux.HandleFunc("/healthz", healthz)
mux.HandleFunc("/readyz", readyz)
```

### 3. Template Base con HTMX

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
        <button type="submit">Enviar Código Mágico</button>
    </form>

    <div id="login-result"></div>
</div>
{{end}}
```

## Routing y Middleware

### Estructura de Rutas Sugerida

```
/                    → Home/Dashboard (HTML)
/login               → Página de login (HTML)
/api/auth/login      → Endpoint HTMX para login
/api/auth/complete   → Endpoint HTMX para completar login
/static/*            → Assets estáticos (CSS, JS, imágenes)
/v1/*                → gRPC Gateway (API REST para clientes externos)
/metrics             → Prometheus metrics
/healthz             → Liveness probe
/readyz              → Readiness probe
```

### Middleware Recomendado

```go
// internal/handlers/middleware.go
package handlers

func RequireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verificar JWT del cookie o header
        // Si no autenticado, redirect a /login
        next.ServeHTTP(w, r)
    })
}

func CSRFProtection(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Validar CSRF token para POST/PUT/DELETE
        next.ServeHTTP(w, r)
    })
}
```

## Assets Estáticos

### Opción 1: FileServer (Desarrollo)

```go
webMux.Handle("/static/", http.StripPrefix("/static/",
    http.FileServer(http.Dir("static"))))
```

### Opción 2: embed (Producción - Recomendado)

```go
//go:embed static/*
var staticFS embed.FS

webMux.Handle("/static/", http.StripPrefix("/static/",
    http.FileServer(http.FS(staticFS))))
```

**Ventaja de embed**: Todo el frontend queda empaquetado en el binario, eliminando dependencias de archivos externos en producción.

## Consideraciones de Seguridad

1. **CSRF Protection**: Implementar tokens CSRF para todas las formas
2. **XSS Prevention**: Usar `html/template` (no `text/template`) que escapa automáticamente
3. **Content Security Policy**: Headers CSP apropiados
4. **Authentication**: Cookies HTTP-only para tokens JWT
5. **HTTPS**: Obligatorio en producción

## Escalabilidad Futura

### Separación Gradual

Si en el futuro necesitas separar frontend/backend:

1. **Fase 1 (Actual)**: Todo en un binario
2. **Fase 2**: Extraer handlers web a un módulo independiente
3. **Fase 3**: Mover a microservicio separado manteniendo la misma API

El diseño modulith facilita esta transición sin cambios drásticos.

### Alternativas para Escalar

- **CDN para Assets**: Servir CSS/JS desde CDN en producción
- **Caching**: Headers de cache apropiados para assets estáticos
- **SSR Caching**: Cachear templates renderizados si es necesario

## Testing

### Unit Tests para Handlers

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
    // Test completo: página login → HTMX request → redirect
    // Similar a los tests de gRPC pero con HTTP
}
```

## Hot Reload

Air ya está configurado para monitorear archivos `.html`. Los cambios en templates se reflejarán automáticamente al reiniciar el servidor.

**Recomendación**: Agregar `"html"` a `include_ext` en `.air.toml` (ya está incluido según el diff).

## Ventajas vs Desventajas

### Ventajas ✅

- Simplicidad operativa (un solo binario)
- Desarrollo más rápido (sin sincronización entre repos)
- Menor complejidad de infraestructura
- Co-locación de código relacionado
- HTMX es ligero y eficiente
- Fácil de testear (todo en un proceso)

### Desventajas ⚠️

- Equipos frontend/backend trabajan en el mismo repo (puede ser ventaja también)
- Si necesitas SPA complejo, esta arquitectura no es ideal
- Assets estáticos aumentan el tamaño del binario (mitigado con CDN)
- Menos flexibilidad para deployar frontend/backend por separado (mitigado por diseño modulith)

## Recomendación Final

**✅ SÍ, incluir el frontend en el mismo repositorio** si:

- Usas Go Templates + HTMX (arquitectura tradicional de servidor)
- Priorizas simplicidad sobre separación estricta
- El equipo es pequeño o el mismo equipo trabaja en ambos
- No necesitas un SPA complejo con React/Vue/etc.

**❌ NO incluir** si:

- Necesitas un SPA complejo con framework moderno (React, Vue, Svelte)
- Tienes equipos completamente separados frontend/backend
- Requieres deployar frontend/backend independientemente desde el inicio

## Próximos Pasos

1. Crear estructura de carpetas (`internal/templates/`, `static/`)
2. Implementar handler base con template parsing
3. Integrar en `cmd/server/main.go`
4. Crear template base y primera página de ejemplo
5. Configurar middleware de autenticación para web
6. Documentar patrones de HTMX en el proyecto

---

**Fecha de Propuesta**: 2025-01-XX
**Estado**: Propuesto
**Decisión**: Pendiente de aprobación

