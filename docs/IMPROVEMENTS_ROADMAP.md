# Improvements Roadmap

This document tracks potential improvements and gaps that would enhance the template to a perfect 10/10 score. These items are identified based on real-world production usage patterns and common challenges teams face when scaling modulith applications.

## Priority Ranking

### Must Have (To Reach 10/10)

1. Performance/caching patterns documentation
2. Development workflow enhancements (deps-graph, etc.)

### Should Have

3. Production deployment patterns
4. Testing patterns for cross-module interactions

### Nice to Have

5. Security hardening examples
6. Multi-tenancy patterns (if applicable)

---

## 1. Performance & Caching Patterns (High Priority)

### Issue

Caching infrastructure exists but usage patterns and best practices are unclear.

### Real-World Scenarios

-   Cache-aside pattern examples
-   Cache invalidation strategies (when user data changes in one module, invalidate cache in another)
-   Cache warming strategies
-   Distributed cache coordination (Valkey vs in-memory decisions)

### What's Missing

-   Examples of caching at different layers (service, repository, HTTP)
-   Cache key naming conventions and standards
-   TTL strategies and cache stampede prevention
-   Cache invalidation patterns across modules

### Impact

Teams will either over-cache (wasting memory) or under-cache (poor performance).

### Implementation Notes

-   Add caching patterns to documentation
-   Provide examples in `examples/` directory
-   Document cache key naming conventions
-   Add cache invalidation strategies

---



## 3. Testing Patterns for Cross-Module (Medium Priority)

### Issue

Unit and integration tests are well-documented, but cross-module testing patterns are unclear.

### Real-World Scenarios

-   Testing event-driven workflows (publish event, verify handler executed)
-   Testing gRPC interactions between modules (module A calls module B)
-   Contract testing between modules
-   End-to-end test patterns

### What's Missing

-   Examples of testing module A calling module B via gRPC
-   Event bus testing patterns (verifying events published)
-   Contract testing setup (Pact or similar)
-   E2E test structure and examples

### Impact

Teams won't know how to properly test cross-module interactions.

### Implementation Notes

-   Add cross-module testing examples to `examples/`
-   Document gRPC client testing patterns
-   Provide event bus testing utilities
-   Add contract testing setup guide

---

## 5. Development Workflow Enhancements (Low Priority)

### Real-World Friction Points

-   Module dependency visualization (which modules depend on which)
-   Breaking change detection (proto changes affecting other modules)
-   Module hot-reload performance (Air reloads everything when one module changes)
-   Development data management (reset, fixtures, test data)

### What's Missing

-   `just deps-graph` - Visualize module dependencies
-   Proto compatibility checking tools
-   Selective module reloading (only reload changed module)
-   Better test data management tools

### Impact

Development experience could be smoother with better tooling.

### Implementation Notes

-   Add dependency graph visualization script
-   Create proto compatibility checker
-   Enhance Air configuration for selective reloading
-   Add test data management tools

---

## 6. Production Deployment Patterns (Low Priority)

### Real-World Scenarios

-   Database migration strategies (zero-downtime, rollback)
-   Feature flag patterns in production
-   A/B testing infrastructure
-   Blue-green deployment examples

### What's Missing

-   Migration runbook/templates
-   Feature flag usage examples in production
-   Canary deployment setup
-   Rollback procedures documentation

### Impact

Teams might struggle with production deployments and migrations.

### Implementation Notes

-   Create deployment runbook in `docs/DEPLOYMENT.md`
-   Add feature flag usage examples
-   Document canary deployment setup
-   Add rollback procedures

---

## 7. Security Hardening (Low Priority)

### Real-World Concerns

-   Rate limiting per user/endpoint (not just per IP)
-   API key management
-   Secret rotation strategies
-   Audit logging

### What's Missing

-   Advanced rate limiting examples (per-user, per-endpoint)
-   API key service pattern
-   Secret rotation automation
-   Audit log infrastructure

### Impact

Security could be enhanced with more advanced patterns.

### Implementation Notes

-   Add advanced rate limiting examples
-   Create API key management pattern
-   Document secret rotation strategies
-   Add audit logging infrastructure

---

## 8. Multi-Tenancy Patterns (If Applicable)

### Real-World Scenarios

-   Data isolation strategies
-   Tenant context propagation
-   Tenant-specific configuration

### What's Missing

-   Multi-tenant architecture guidance
-   Row-level security patterns
-   Tenant context helpers

### Impact

Only relevant if multi-tenancy is a requirement.

### Implementation Notes

-   Only add if multi-tenancy is a common requirement
-   Document tenant isolation patterns
-   Add tenant context propagation utilities

---

## Recommended Implementation Order

To reach a perfect 10/10, prioritize:

1. **Performance/Caching Patterns**
    - `CACHING_PATTERNS.md` to explain how to use Valkey/In-memory cache.

2. **Development Workflow Enhancements**
    - `just deps-graph` rule to automatically generate module visualization graphs.

These areas address the remaining common production challenges.

---

## Notes

-   This roadmap is based on real-world production usage patterns
-   Items are prioritized by impact and frequency of need
-   Each item should include:
    -   Documentation
    -   Code examples
    -   Test examples (where applicable)
    -   Integration with existing infrastructure

---

**Last Updated**: 2026-01-03
