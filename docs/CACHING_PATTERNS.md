# Caching Patterns Guide

Caching is a critical component for maintaining high performance in a Modulith architecture. This guide outlines the caching strategies available in this template, when to use them, and the correct patterns to follow.

## Overview

We provide a unified `cache.Cache` interface (located in `internal/cache/cache.go`) to interact with cache providers. The available implementations are:
1. **MemoryCache (`internal/cache/memory.go`)**: Suitable for single-instance deployments, local development, or data that does not need to be synchronized across instances.
2. **ValkeyCache (`internal/cache/valkey.go`)**: The distributed cache choice for production environments running multiple instances.

By depending only on `cache.Cache`, your modules remain agnostic to the underlying cache implementation.

## 1. When to use Caching

**DO USE CACHING FOR:**
- Slow database queries that don't need real-time consistency.
- Computed data that is read heavily but changes rarely (e.g., Configuration data, User profiles).
- API Rate Limiting windows.
- Resolving identical requests across module boundaries (e.g. JWT validations or API keys).

**DO NOT USE CACHING FOR:**
- Data that requires strong transactional consistency (use Database instead).
- Ephemeral workflow statuses that change constantly.
- Cross-module coordination that requires guaranteed delivery (use Event Bus instead).

## 2. Using the Cache Interface

Access cache through the central registry:
```go
memCache := registry.Cache() // Retrieves the injected cache instance
```

A common pattern is the **Cache-Aside** strategy. Here is an example of querying the cache before falling back to the database:

```go
func (s *Service) GetUser(ctx context.Context, userID string) (*User, error) {
    cacheKey := fmt.Sprintf("user_profile:%s", userID)
    
    // 1. Try to fetch from cache
    if cachedData, err := s.cache.Get(ctx, cacheKey); err == nil {
        var user User
        if err := json.Unmarshal(cachedData, &user); err == nil {
            return &user, nil // Cache Hit
        }
    }
    
    // 2. Cache Miss - Fetch from Database
    user, err := s.repo.FindUserByID(ctx, userID)
    if err != nil {
         return nil, err
    }
    
    // 3. Serialize and save to cache asynchronously
    if userData, err := json.Marshal(user); err == nil {
         _ = s.cache.Set(ctx, cacheKey, userData, 15 * time.Minute)
    }

    return user, nil
}
```

## 3. Cache Invalidation Strategies

Invalidating cache across module lines can be tricky.
When a record changes inside a module, the cache must be updated.

### Active Invalidation
When an update goes through your service, immediately call `s.cache.Delete(ctx, cacheKey)`.

### Event-Driven Invalidation
If Module A caches a user profile, and Module B updates the user avatar, Module A's cache becomes stale.
Use the outbox/event-bus to publish an event (`user.avatar_updated`). Module A listens to this event and invalidates the `user_profile:*` cache key associated with that user ID.

```go
func (s *UserCacheInvalidator) HandleUserUpdated(ctx context.Context, payload []byte) error {
    var event struct { UserID string }
    _ = json.Unmarshal(payload, &event)

    return s.cache.Delete(ctx, fmt.Sprintf("user_profile:%s", event.UserID))
}
```

## 4. Key Naming Conventions

Always namespace your cache keys to avoid collisions between modules. Use the pattern:
`<module_name>:<domain_model>:<identifier>:<optional_suffix>`.

**Good:**
- `auth:session:usr_1234:permissions`
- `orders:summary:ord_9999`
- `inventory:stock_count:item_55`

**Bad:**
- `usr_1234`
- `session`
- `data:1:full`

## 5. Switching between Memory and Valkey

In `templates/module/cmd/main.go.tmpl`, you define the injected registry dependencies.
During development, you may instantiate `cache.NewMemoryCache()`. 

To migrate to Valkey:
1. Ensure the `github.com/valkey-io/valkey-go` dependency is satisfied.
2. Implement the methods inside `internal/cache/valkey.go`.
3. Swap `cache.NewMemoryCache()` with `cache.NewValkeyCache()` in `createRegistry()`.

> [!WARNING]
> The `ValkeyCache` included in this template is currently a stub for you to complete depending on your `valkey-go` or `go-redis` version preference. All logic methods must be mapped to your chosen Valkey client.
