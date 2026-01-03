# Logging Standards

This document defines logging standards and best practices for the modulith template.

## Table of Contents

1. [Overview](#overview)
2. [Log Levels](#log-levels)
3. [Structured Logging](#structured-logging)
4. [Log Format](#log-format)
5. [Context Fields](#context-fields)
6. [Best Practices](#best-practices)
7. [Examples](#examples)

## Overview

The modulith template uses structured logging with `slog` (Go's structured logging package). All logs are JSON-formatted in production for easy parsing and analysis.

## Log Levels

Use the following log levels consistently:

### DEBUG
Use for detailed information that is only useful for debugging.

```go
slog.Debug("processing request", "user_id", userID, "request_id", requestID)
```

**When to use:**
- Function entry/exit points (in debug mode)
- Detailed state information
- Development-only information

### INFO
Use for informational messages about normal application flow.

```go
slog.Info("user logged in", "user_id", userID, "method", "email")
```

**When to use:**
- Successful operations
- Important state changes
- Business events
- Startup/shutdown messages

### WARN
Use for warning conditions that don't stop execution but should be noticed.

```go
slog.Warn("retry attempt failed", "attempt", attempt, "error", err)
```

**When to use:**
- Recoverable errors
- Deprecated feature usage
- Performance degradation
- Unusual but valid conditions

### ERROR
Use for error conditions that require attention but don't stop execution.

```go
slog.Error("failed to process request", "error", err, "request_id", requestID)
```

**When to use:**
- Failed operations
- External service failures
- Database errors
- Configuration issues

## Structured Logging

Always use structured logging with key-value pairs:

```go
// Good: Structured logging
slog.Info("user created",
    "user_id", userID,
    "email", email,
    "method", "oauth",
)

// Bad: String concatenation
slog.Info(fmt.Sprintf("user created: %s, email: %s", userID, email))
```

### Benefits

- **Searchable**: Easy to search/filter by field
- **Parseable**: JSON format can be parsed by log aggregation tools
- **Queryable**: Can create dashboards and alerts based on fields
- **Structured**: Consistent format across all logs

## Log Format

### Development
In development, logs use human-readable text format:

```
2024-01-15T10:30:45.123Z INFO user created user_id=user-123 email=test@example.com method=oauth
```

### Production
In production, logs use JSON format:

```json
{
  "time": "2024-01-15T10:30:45.123Z",
  "level": "INFO",
  "msg": "user created",
  "user_id": "user-123",
  "email": "test@example.com",
  "method": "oauth"
}
```

## Context Fields

Always include relevant context fields in logs:

### Request Context

```go
slog.Info("request processed",
    "request_id", requestID,
    "user_id", userID,
    "method", r.Method,
    "path", r.URL.Path,
    "status_code", statusCode,
    "duration_ms", duration.Milliseconds(),
)
```

### Error Context

```go
slog.Error("operation failed",
    "error", err,
    "operation", "create_user",
    "user_id", userID,
    "attempt", attempt,
)
```

### Module Context

```go
slog.Info("module operation",
    "module", "auth",
    "operation", "login",
    "user_id", userID,
)
```

### Database Context

```go
slog.Debug("database query",
    "query", "SELECT * FROM users",
    "duration_ms", duration.Milliseconds(),
    "rows_affected", rowsAffected,
)
```

### Event Context

```go
slog.Info("event published",
    "event_name", "user.created",
    "event_id", eventID,
    "user_id", userID,
)
```

## Best Practices

### 1. Use Appropriate Log Levels

- **DEBUG**: Development/debugging only
- **INFO**: Normal operations, business events
- **WARN**: Recoverable issues, performance warnings
- **ERROR**: Failures that need attention

### 2. Include Context

Always include relevant context fields:

```go
// Good: Includes context
slog.Error("failed to create user",
    "error", err,
    "user_id", userID,
    "email", email,
    "operation", "create_user",
)

// Bad: Missing context
slog.Error("failed to create user", "error", err)
```

### 3. Don't Log Sensitive Information

Never log:
- Passwords or password hashes
- API keys or secrets
- Credit card numbers
- Personal identification numbers (SSN, etc.)
- Authentication tokens (except for debugging)

```go
// Bad: Logging sensitive information
slog.Info("user login", "password", password, "api_key", apiKey)

// Good: Logging safe information
slog.Info("user login", "user_id", userID, "method", "password")
```

### 4. Use Consistent Field Names

Use consistent field names across the application:

- `user_id`: User identifier
- `request_id`: Request identifier (from context)
- `error`: Error object
- `duration_ms`: Duration in milliseconds
- `module`: Module name
- `operation`: Operation name
- `event_name`: Event name
- `event_id`: Event identifier

### 5. Log Errors with Context

Always log errors with sufficient context:

```go
// Good: Error with context
if err := repo.CreateUser(ctx, user); err != nil {
    slog.Error("failed to create user",
        "error", err,
        "user_id", user.ID,
        "email", user.Email,
        "operation", "create_user",
    )
    return fmt.Errorf("create user: %w", err)
}

// Bad: Error without context
if err := repo.CreateUser(ctx, user); err != nil {
    slog.Error("failed to create user", "error", err)
    return err
}
```

### 6. Use Request ID from Context

Extract request ID from context for correlation:

```go
requestID := middleware.RequestIDFromContext(ctx)
slog.Info("processing request",
    "request_id", requestID,
    "user_id", userID,
)
```

### 7. Log at Appropriate Granularity

- **Too verbose**: Logging every function call
- **Too sparse**: Only logging errors
- **Just right**: Logging important operations, state changes, and errors

### 8. Use Structured Fields for Metrics

Include fields that can be used for metrics:

```go
slog.Info("request completed",
    "request_id", requestID,
    "status_code", statusCode,
    "duration_ms", duration.Milliseconds(),
    "method", method,
    "path", path,
)
```

### 9. Don't Log in Hot Paths

Avoid logging in hot paths (frequently called code) unless necessary:

```go
// Bad: Logging in hot path
func (s *Service) ProcessItem(item Item) error {
    slog.Debug("processing item", "item_id", item.ID) // Called millions of times
    // ... process item
}

// Good: Logging only when needed
func (s *Service) ProcessItem(item Item) error {
    // ... process item
    if item.Important {
        slog.Info("processed important item", "item_id", item.ID)
    }
}
```

### 10. Use Context for Scoped Logging

Use context to carry request-scoped information:

```go
// Add to context
ctx = context.WithValue(ctx, "user_id", userID)
ctx = context.WithValue(ctx, "request_id", requestID)

// Use from context
userID := ctx.Value("user_id")
slog.Info("operation", "user_id", userID)
```

## Examples

### Service Layer

```go
func (s *AuthService) RequestLogin(ctx context.Context, req *authv1.RequestLoginRequest) (*authv1.RequestLoginResponse, error) {
    requestID := middleware.RequestIDFromContext(ctx)

    slog.Info("login request received",
        "request_id", requestID,
        "email", req.Email,
        "method", "magic_link",
    )

    code, err := s.repo.CreateMagicCode(ctx, req.Email)
    if err != nil {
        slog.Error("failed to create magic code",
            "error", err,
            "request_id", requestID,
            "email", req.Email,
        )
        return nil, fmt.Errorf("create magic code: %w", err)
    }

    slog.Info("magic code created",
        "request_id", requestID,
        "email", req.Email,
    )

    return &authv1.RequestLoginResponse{CodeSent: true}, nil
}
```

### Repository Layer

```go
func (r *Repository) CreateUser(ctx context.Context, user *User) error {
    start := time.Now()

    err := r.q.CreateUser(ctx, userParams)
    if err != nil {
        slog.Error("database error",
            "error", err,
            "operation", "create_user",
            "user_id", user.ID,
            "duration_ms", time.Since(start).Milliseconds(),
        )
        return fmt.Errorf("create user: %w", err)
    }

    slog.Debug("user created",
        "user_id", user.ID,
        "email", user.Email,
        "duration_ms", time.Since(start).Milliseconds(),
    )

    return nil
}
```

### Event Handlers

```go
func (h *UserEventHandler) HandleUserCreated(ctx context.Context, event events.Event) error {
    eventID := event.Metadata["event_id"]
    userID := event.Payload.(map[string]interface{})["user_id"]

    slog.Info("handling user created event",
        "event_id", eventID,
        "event_name", "user.created",
        "user_id", userID,
    )

    // Process event
    if err := h.processUserCreated(ctx, userID); err != nil {
        slog.Error("failed to handle user created event",
            "error", err,
            "event_id", eventID,
            "user_id", userID,
        )
        return fmt.Errorf("handle user created: %w", err)
    }

    slog.Info("user created event handled",
        "event_id", eventID,
        "user_id", userID,
    )

    return nil
}
```

### Middleware

```go
func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        requestID := middleware.RequestIDFromContext(r.Context())

        // Log request
        slog.Info("request started",
            "request_id", requestID,
            "method", r.Method,
            "path", r.URL.Path,
            "remote_addr", r.RemoteAddr,
        )

        // Process request
        next.ServeHTTP(w, r)

        // Log response
        duration := time.Since(start)
        statusCode := getStatusCode(w)

        slog.Info("request completed",
            "request_id", requestID,
            "method", r.Method,
            "path", r.URL.Path,
            "status_code", statusCode,
            "duration_ms", duration.Milliseconds(),
        )
    })
}
```

## Summary

- Use structured logging with key-value pairs
- Use appropriate log levels (DEBUG, INFO, WARN, ERROR)
- Include relevant context fields
- Don't log sensitive information
- Use consistent field names
- Log errors with sufficient context
- Extract request ID from context for correlation
- Log at appropriate granularity
- Use JSON format in production
- Follow best practices for performance and security

For more information, see:
- [Go slog documentation](https://pkg.go.dev/log/slog)
- [Observability Setup Guide](OBSERVABILITY_SETUP.md)
- [Error Handling Standards](.cursor/rules/60-errors-telemetry.mdc)

