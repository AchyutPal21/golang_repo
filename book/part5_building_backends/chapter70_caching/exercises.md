# Chapter 70 Exercises — Caching

## Exercise 1 — Product Cache (`exercises/01_product_cache`)

Implement a caching layer over any `ProductRepository` using an LRU+TTL cache with observable metrics and proper write-invalidation.

### Types

```go
type CacheConfig struct {
    MaxSize int
    TTL     time.Duration
}

type CacheMetrics struct {
    Hits        int
    Misses      int
    Evictions   int
    TTLExpiries int
}

type CachedProductStore struct {
    inner   ProductRepository
    cfg     CacheConfig
    // ... LRU internals
}
```

### Required behaviour

| Operation | Cache behaviour |
|---|---|
| `Create` | Write to inner; pre-populate cache entry |
| `GetByID` — cache hit, TTL valid | Return from cache; increment Hits |
| `GetByID` — cache miss | Load from inner; populate cache; increment Misses |
| `GetByID` — TTL expired | Re-fetch from inner; increment TTLExpiries |
| `Update` | Write to inner; invalidate cache entry |
| `Delete` | Write to inner; invalidate cache entry |
| Capacity exceeded | Evict LRU entry; increment Evictions |

### Stale-while-revalidate

When a cache entry's TTL expires, the current implementation removes the stale entry and returns the freshly fetched value synchronously. The concept of stale-while-revalidate means you could instead serve the stale value immediately and refresh asynchronously in a goroutine. Implement the simpler synchronous refresh for this exercise but add a comment explaining how to extend it to async.

### Hints

- Use `container/list` from the standard library for O(1) LRU ordering.
- Store `expiresAt time.Time` inside each list element to check TTL without a separate map.
- Call `c.mu.Lock()` before any cache read/write, but release before calling `c.inner.*` to avoid holding the lock during slow I/O.
- `Metrics()` should return a value copy (not a pointer) so the caller gets a snapshot.
