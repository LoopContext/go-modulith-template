# Environment Variables

This document describes all environment variables used by the Go Modulith template, their purposes, defaults, and configuration precedence.

## Configuration Precedence

The application loads configuration in the following order (highest to lowest priority):

1. **`PORT` environment variable** - Standard 12-factor app port binding (takes precedence over HTTP_PORT)
2. **YAML Configuration** (`configs/server.yaml` or `configs/server.prod.yaml`) - High priority
3. **`.env` file** - Overrides system environment variables
4. **System Environment Variables** - Base values
5. **Default Values** - Hardcoded in `internal/config/config.go`

**Priority:** `PORT > YAML > .env > system ENV vars > defaults`

**Note:** The `PORT` environment variable is a standard convention used by Heroku, Cloud Run, Railway, Render, and other platforms. If set, it will override `HTTP_PORT` to ensure compatibility with these platforms.

## Required Variables (Production)

These variables **must** be set in production environments:

| Variable     | Description                  | Example                                             | Notes                                                        |
| ------------ | ---------------------------- | --------------------------------------------------- | ------------------------------------------------------------ |
| `DB_DSN`     | PostgreSQL connection string | `postgres://user:pass@host:5432/db?sslmode=require` | Required for database access                                 |
| `JWT_SECRET` | JWT signing key              | `your-secret-key-at-least-32-bytes-long`            | **Must be at least 32 bytes (256 bits)** for HS256 algorithm |

## Optional Variables

### Application Configuration

| Variable       | Default           | Description                                      | Example           |
| -------------- | ----------------- | ------------------------------------------------ | ----------------- |
| `ENV`          | `dev`             | Environment name (`dev`, `staging`, `prod`)      | `prod`            |
| `LOG_LEVEL`    | `debug`           | Logging level (`debug`, `info`, `warn`, `error`) | `info`            |
| `PORT`         | (none)            | HTTP server port (12-factor standard)            | `8080`            |
| `HTTP_PORT`    | `8080`            | HTTP server port (explicit)                      | `8080`            |
| `GRPC_PORT`    | `9050`            | gRPC server port                                 | `9050`            |
| `SERVICE_NAME` | `modulith-server` | Service name for OpenTelemetry resource          | `modulith-server` |

### Database Configuration

| Variable               | Default | Description                       | Example |
| ---------------------- | ------- | --------------------------------- | ------- |
| `DB_MAX_OPEN_CONNS`    | `25`    | Maximum open database connections | `50`    |
| `DB_MAX_IDLE_CONNS`    | `25`    | Maximum idle database connections | `25`    |
| `DB_CONN_MAX_LIFETIME` | `5m`    | Maximum connection lifetime       | `15m`   |
| `DB_CONNECT_TIMEOUT`   | `10s`   | Initial connection timeout        | `10s`   |

### Observability

| Variable        | Default | Description                      | Example       |
| --------------- | ------- | -------------------------------- | ------------- |
| `OTLP_ENDPOINT` | ``      | OpenTelemetry collector endpoint | `jaeger:4317` |

### CORS Configuration

| Variable               | Default | Description                             | Example                                       |
| ---------------------- | ------- | --------------------------------------- | --------------------------------------------- |
| `CORS_ALLOWED_ORIGINS` | `*`     | Comma-separated list of allowed origins | `https://example.com,https://app.example.com` |

**Note:** In production, never use `*`. Specify exact origins separated by commas.

### Rate Limiting

| Variable             | Default | Description                | Example |
| -------------------- | ------- | -------------------------- | ------- |
| `RATE_LIMIT_ENABLED` | `false` | Enable rate limiting       | `true`  |
| `RATE_LIMIT_RPS`     | `100`   | Requests per second per IP | `1000`  |
| `RATE_LIMIT_BURST`   | `50`    | Burst size                 | `100`   |

### Timeouts

| Variable           | Default | Description                                   | Example |
| ------------------ | ------- | --------------------------------------------- | ------- |
| `READ_TIMEOUT`     | `5s`    | HTTP server read timeout                      | `10s`   |
| `WRITE_TIMEOUT`    | `10s`   | HTTP server write timeout                     | `30s`   |
| `REQUEST_TIMEOUT`  | `30s`   | Maximum request duration (middleware timeout) | `30s`   |
| `SHUTDOWN_TIMEOUT` | `30s`   | Graceful shutdown timeout                     | `60s`   |

### OAuth Configuration

| Variable                     | Default | Description                      | Example                   |
| ---------------------------- | ------- | -------------------------------- | ------------------------- |
| `OAUTH_ENABLED`              | `false` | Enable OAuth authentication      | `true`                    |
| `OAUTH_AUTO_LINK_BY_EMAIL`   | `true`  | Auto-link accounts by email      | `true`                    |
| `OAUTH_BASE_URL`             | ``      | Base URL for OAuth callbacks     | `https://yourdomain.com`  |
| `OAUTH_TOKEN_ENCRYPTION_KEY` | ``      | 32-byte key for token encryption | `your-32-byte-key-here!!` |

#### OAuth Provider Variables

| Variable                  | Description                      | Example                          |
| ------------------------- | -------------------------------- | -------------------------------- |
| `GOOGLE_CLIENT_ID`        | Google OAuth client ID           | `xxx.apps.googleusercontent.com` |
| `GOOGLE_CLIENT_SECRET`    | Google OAuth client secret       | `GOCSPX-xxx`                     |
| `FACEBOOK_CLIENT_ID`      | Facebook App ID                  | `123456789`                      |
| `FACEBOOK_CLIENT_SECRET`  | Facebook App Secret              | `xxx`                            |
| `GITHUB_CLIENT_ID`        | GitHub OAuth client ID           | `Iv1.xxx`                        |
| `GITHUB_CLIENT_SECRET`    | GitHub OAuth client secret       | `xxx`                            |
| `MICROSOFT_CLIENT_ID`     | Microsoft Azure AD client ID     | `xxx`                            |
| `MICROSOFT_CLIENT_SECRET` | Microsoft Azure AD client secret | `xxx`                            |
| `TWITTER_CLIENT_ID`       | Twitter API Key                  | `xxx`                            |
| `TWITTER_CLIENT_SECRET`   | Twitter API Secret               | `xxx`                            |
| `APPLE_CLIENT_ID`         | Apple Services ID                | `com.example.service`            |
| `APPLE_TEAM_ID`           | Apple Team ID                    | `ABC123DEF4`                     |
| `APPLE_KEY_ID`            | Apple Key ID                     | `XYZ123ABC4`                     |
| `APPLE_PRIVATE_KEY_PATH`  | Path to Apple private key file   | `/path/to/AuthKey.p8`            |

## Security Considerations

### Secrets Management

**Never commit secrets to version control!** Use one of these approaches:

1. **Environment Variables** (Development)

    - Use `.env` file (gitignored)
    - Set in your shell or IDE

2. **Secrets Manager** (Production)

    - AWS Secrets Manager
    - HashiCorp Vault
    - Kubernetes Secrets
    - Cloud provider secret stores

3. **Configuration Files** (Development only)
    - Use `configs/server.yaml` for non-sensitive config
    - Never commit actual secrets

### JWT Secret Requirements

-   **Minimum length:** 32 bytes (256 bits) for HS256 algorithm
-   **Recommended:** Use a cryptographically secure random generator
-   **Example generation:**
    ```bash
    openssl rand -base64 32
    ```

### Database Connection Strings

-   Use `sslmode=require` or `sslmode=verify-full` in production
-   Never log connection strings
-   Rotate credentials regularly

## Configuration Source Logging

When the application starts, it logs the source of each configuration value:

```
Configuration sources
  ENV="prod = yaml"
  HTTP_PORT="8080 = yaml"
  DB_DSN="postgres://... = .env"
  JWT_SECRET="[42 bytes] = system"
```

This helps debug configuration issues and understand which source provided each value.

## Example `.env` File

```bash
# Application
ENV=dev
LOG_LEVEL=debug
# Use PORT for 12-factor compliance (Heroku, Cloud Run, etc.)
# or HTTP_PORT for explicit control
PORT=8080
# HTTP_PORT=8080  # Alternative to PORT
GRPC_PORT=9050

# Database
DB_DSN=postgres://postgres:postgres@localhost:5432/modulith_demo?sslmode=disable

# Security
JWT_SECRET=your-secret-key-at-least-32-bytes-long-for-production

# Observability (optional)
OTLP_ENDPOINT=localhost:4317

# CORS (optional)
CORS_ALLOWED_ORIGINS=*

# Rate limiting (optional)
RATE_LIMIT_ENABLED=false
RATE_LIMIT_RPS=100
RATE_LIMIT_BURST=50

# OAuth (optional)
OAUTH_ENABLED=false
GOOGLE_CLIENT_ID=your-client-id
GOOGLE_CLIENT_SECRET=your-client-secret
```

## Dev/Prod Parity (12-Factor App: Factor X)

El template está diseñado para mantener **paridad entre desarrollo y producción**, siguiendo el principio 12-factor app.

### Principios de Paridad

**1. Mismas Dependencias:**

-   ✅ **Base de datos:** PostgreSQL 18 (misma versión en dev y prod)
-   ✅ **Redis:** Redis 7 (opcional, misma versión)
-   ✅ **Go:** Go 1.24+ (misma versión de compilación)
-   ✅ **Herramientas:** Versiones fijas en `go.mod` y `buf.lock`

**2. Mismas Herramientas:**

-   ✅ **Docker Compose:** Mismas imágenes que producción
-   ✅ **Migraciones:** `golang-migrate` (misma herramienta)
-   ✅ **SQLC:** Misma versión para generación de código
-   ✅ **Buf:** Misma versión para generación de protobuf

**3. Mínimas Diferencias de Tiempo:**

-   ✅ **Deploy rápido:** Código en producción minutos después de desarrollo
-   ✅ **CI/CD:** Pipelines automatizados para validación
-   ✅ **Testing:** Tests ejecutados en ambiente similar a producción

### Verificación de Paridad

**Versiones en `docker-compose.yaml`:**

```yaml
db:
    image: postgres:18-alpine # ✅ Misma versión que producción

redis:
    image: redis:7-alpine # ✅ Misma versión que producción
```

**Recomendación:** Usar las mismas versiones de imágenes en producción (Kubernetes/Helm).

### Diferencias Aceptables

Algunas diferencias son aceptables y necesarias:

1. **Configuración:**

    - Dev: `ENV=dev`, `LOG_LEVEL=debug`
    - Prod: `ENV=prod`, `LOG_LEVEL=info`

2. **Recursos:**

    - Dev: Recursos limitados (CPU/memoria)
    - Prod: Recursos escalables según demanda

3. **Observabilidad:**

    - Dev: Jaeger/Prometheus local (opcional)
    - Prod: Observabilidad centralizada (requerido)

4. **Secrets:**
    - Dev: Variables de entorno o `.env`
    - Prod: Secrets manager (Vault, AWS Secrets Manager, etc.)

### Mejores Prácticas

**1. Usar Mismas Versiones:**

```bash
# En desarrollo
docker-compose up db  # postgres:18-alpine

# En producción (Kubernetes)
# Usar la misma versión en Helm values
```

**2. Testing en Ambiente Similar:**

-   Ejecutar tests de integración con Docker Compose
-   Usar `testcontainers` para tests automatizados
-   Validar migraciones en ambiente de staging

**3. Documentar Diferencias:**

-   Mantener `configs/server.yaml` para dev
-   Usar `configs/server.prod.yaml` para producción
-   Documentar cualquier diferencia necesaria

### Checklist de Paridad

Antes de desplegar a producción:

-   [ ] Verificar que las versiones de DB/Redis coinciden con producción
-   [ ] Validar que las migraciones funcionan en staging
-   [ ] Ejecutar tests de integración con Docker Compose
-   [ ] Verificar que la configuración de producción está documentada
-   [ ] Asegurar que los secrets se gestionan correctamente
-   [ ] Validar que los timeouts y límites son apropiados para producción

## Production Checklist

Before deploying to production:

-   [ ] Set `ENV=prod`
-   [ ] Set `LOG_LEVEL=info` (or `warn`/`error`)
-   [ ] Configure `DB_DSN` with production database
-   [ ] Set strong `JWT_SECRET` (32+ bytes)
-   [ ] Enable rate limiting (`RATE_LIMIT_ENABLED=true`)
-   [ ] Restrict CORS origins (never use `*`)
-   [ ] Configure `OTLP_ENDPOINT` for distributed tracing
-   [ ] Use secrets manager for sensitive values
-   [ ] Review and adjust timeout values
-   [ ] Test configuration loading and validation
-   [ ] Verify dev/prod parity (same DB/Redis versions)
