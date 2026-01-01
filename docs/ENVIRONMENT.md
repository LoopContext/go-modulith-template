# Environment Variables

This document describes all environment variables used by the Go Modulith template, their purposes, defaults, and configuration precedence.

## Configuration Precedence

The application loads configuration in the following order (highest to lowest priority):

1. **YAML Configuration** (`configs/server.yaml` or `configs/server.prod.yaml`) - Highest priority
2. **`.env` file** - Overrides system environment variables
3. **System Environment Variables** - Base values
4. **Default Values** - Hardcoded in `internal/config/config.go`

**Priority:** `YAML > .env > system ENV vars > defaults`

## Required Variables (Production)

These variables **must** be set in production environments:

| Variable | Description | Example | Notes |
|----------|-------------|---------|-------|
| `DB_DSN` | PostgreSQL connection string | `postgres://user:pass@host:5432/db?sslmode=require` | Required for database access |
| `JWT_SECRET` | JWT signing key | `your-secret-key-at-least-32-bytes-long` | **Must be at least 32 bytes (256 bits)** for HS256 algorithm |

## Optional Variables

### Application Configuration

| Variable | Default | Description | Example |
|----------|---------|-------------|---------|
| `ENV` | `dev` | Environment name (`dev`, `staging`, `prod`) | `prod` |
| `LOG_LEVEL` | `debug` | Logging level (`debug`, `info`, `warn`, `error`) | `info` |
| `HTTP_PORT` | `8080` | HTTP server port | `8080` |
| `GRPC_PORT` | `9050` | gRPC server port | `9050` |
| `SERVICE_NAME` | `modulith-server` | Service name for OpenTelemetry resource | `modulith-server` |

### Database Configuration

| Variable | Default | Description | Example |
|----------|---------|-------------|---------|
| `DB_MAX_OPEN_CONNS` | `25` | Maximum open database connections | `50` |
| `DB_MAX_IDLE_CONNS` | `25` | Maximum idle database connections | `25` |
| `DB_CONN_MAX_LIFETIME` | `5m` | Maximum connection lifetime | `15m` |
| `DB_CONNECT_TIMEOUT` | `10s` | Initial connection timeout | `10s` |

### Observability

| Variable | Default | Description | Example |
|----------|---------|-------------|---------|
| `OTLP_ENDPOINT` | `` | OpenTelemetry collector endpoint | `jaeger:4317` |

### Rate Limiting

| Variable | Default | Description | Example |
|----------|---------|-------------|---------|
| `RATE_LIMIT_ENABLED` | `false` | Enable rate limiting | `true` |
| `RATE_LIMIT_RPS` | `100` | Requests per second per IP | `1000` |
| `RATE_LIMIT_BURST` | `50` | Burst size | `100` |

### Timeouts

| Variable | Default | Description | Example |
|----------|---------|-------------|---------|
| `READ_TIMEOUT` | `5s` | HTTP server read timeout | `10s` |
| `WRITE_TIMEOUT` | `10s` | HTTP server write timeout | `30s` |
| `REQUEST_TIMEOUT` | `30s` | Maximum request duration (middleware timeout) | `30s` |
| `SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown timeout | `60s` |

### OAuth Configuration

| Variable | Default | Description | Example |
|----------|---------|-------------|---------|
| `OAUTH_ENABLED` | `false` | Enable OAuth authentication | `true` |
| `OAUTH_AUTO_LINK_BY_EMAIL` | `true` | Auto-link accounts by email | `true` |
| `OAUTH_BASE_URL` | `` | Base URL for OAuth callbacks | `https://yourdomain.com` |
| `OAUTH_TOKEN_ENCRYPTION_KEY` | `` | 32-byte key for token encryption | `your-32-byte-key-here!!` |

#### OAuth Provider Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `GOOGLE_CLIENT_ID` | Google OAuth client ID | `xxx.apps.googleusercontent.com` |
| `GOOGLE_CLIENT_SECRET` | Google OAuth client secret | `GOCSPX-xxx` |
| `FACEBOOK_CLIENT_ID` | Facebook App ID | `123456789` |
| `FACEBOOK_CLIENT_SECRET` | Facebook App Secret | `xxx` |
| `GITHUB_CLIENT_ID` | GitHub OAuth client ID | `Iv1.xxx` |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth client secret | `xxx` |
| `MICROSOFT_CLIENT_ID` | Microsoft Azure AD client ID | `xxx` |
| `MICROSOFT_CLIENT_SECRET` | Microsoft Azure AD client secret | `xxx` |
| `TWITTER_CLIENT_ID` | Twitter API Key | `xxx` |
| `TWITTER_CLIENT_SECRET` | Twitter API Secret | `xxx` |
| `APPLE_CLIENT_ID` | Apple Services ID | `com.example.service` |
| `APPLE_TEAM_ID` | Apple Team ID | `ABC123DEF4` |
| `APPLE_KEY_ID` | Apple Key ID | `XYZ123ABC4` |
| `APPLE_PRIVATE_KEY_PATH` | Path to Apple private key file | `/path/to/AuthKey.p8` |

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

- **Minimum length:** 32 bytes (256 bits) for HS256 algorithm
- **Recommended:** Use a cryptographically secure random generator
- **Example generation:**
  ```bash
  openssl rand -base64 32
  ```

### Database Connection Strings

- Use `sslmode=require` or `sslmode=verify-full` in production
- Never log connection strings
- Rotate credentials regularly

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
HTTP_PORT=8080
GRPC_PORT=9050

# Database
DB_DSN=postgres://postgres:postgres@localhost:5432/modulith_demo?sslmode=disable

# Security
JWT_SECRET=your-secret-key-at-least-32-bytes-long-for-production

# Observability (optional)
OTLP_ENDPOINT=localhost:4317

# OAuth (optional)
OAUTH_ENABLED=false
GOOGLE_CLIENT_ID=your-client-id
GOOGLE_CLIENT_SECRET=your-client-secret
```

## Production Checklist

Before deploying to production:

- [ ] Set `ENV=prod`
- [ ] Set `LOG_LEVEL=info` (or `warn`/`error`)
- [ ] Configure `DB_DSN` with production database
- [ ] Set strong `JWT_SECRET` (32+ bytes)
- [ ] Enable rate limiting (`RATE_LIMIT_ENABLED=true`)
- [ ] Restrict CORS origins (never use `*`)
- [ ] Configure `OTLP_ENDPOINT` for distributed tracing
- [ ] Use secrets manager for sensitive values
- [ ] Review and adjust timeout values
- [ ] Test configuration loading and validation

