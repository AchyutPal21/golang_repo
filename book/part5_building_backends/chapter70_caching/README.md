# Chapter 70 — Caching

## What you'll learn

How to build caching from first principles in Go: a generic LRU cache backed by a doubly-linked list and hash map, TTL expiry, and the three classic write strategies. You'll also build an HTTP-level response cache middleware with ETag/304 support, Vary-header awareness, and Cache-Control parsing, and wrap a repository with a transparent LRU+TTL cache that emits hit/miss/eviction metrics.

## Key concepts

| Concept | Description | File |
|---|---|---|
| **LRU eviction** | Least-recently-used item removed when capacity exceeded | `01_cache_patterns` |
| **TTL expiry** | Per-item time-to-live via wrapper around LRU | `01_cache_patterns` |
| **Cache-aside** | App checks cache first, populates on miss | `01_cache_patterns` |
| **Write-through** | Write to both store and cache synchronously | `01_cache_patterns` |
| **Write-behind** | Write to cache immediately, flush to store async | `01_cache_patterns` |
| **ETag / 304** | Hash-based fingerprint enables conditional requests | `02_http_cache` |
| **Cache-Control** | max-age, no-cache, no-store parsed from headers | `02_http_cache` |
| **Vary header** | Different Accept headers get separate cache entries | `02_http_cache` |
| **Cache invalidation** | Evict entry on update/delete to prevent stale reads | `01_product_cache` |
| **Stale-while-revalidate** | Serve stale data while re-fetching in background | `01_product_cache` |

## Files

| File | Topic |
|---|---|
| `examples/01_cache_patterns/main.go` | Generic `LRUCache[K,V]`, `TTLCache`, cache-aside, write-through, write-behind |
| `examples/02_http_cache/main.go` | `ResponseCache` middleware with ETag, Cache-Control, Vary |
| `exercises/01_product_cache/main.go` | `CachedProductStore` with metrics, TTL, invalidation |

## Generic LRU Cache

```go
type LRUCache[K comparable, V any] struct {
    capacity int
    items    map[K]*list.Element
    order    *list.List          // front = most-recent, back = LRU
}

// Get is O(1): map lookup + list move-to-front.
func (c *LRUCache[K, V]) Get(key K) (V, bool) { ... }

// Put is O(1): insert at front, evict back when over capacity.
func (c *LRUCache[K, V]) Put(key K, value V) { ... }
```

## TTL cache pattern

```go
type TTLCache[K comparable, V any] struct {
    inner *LRUCache[K, ttlEntry[V]]
    ttl   time.Duration
}

func (c *TTLCache[K, V]) Get(key K) (V, bool) {
    te, ok := c.inner.Get(key)
    if !ok { return zero, false }
    if time.Now().After(te.expiresAt) {
        c.inner.Delete(key)   // lazy expiry — remove on access
        return zero, false
    }
    return te.value, true
}
```

## Cache-aside (lazy loading)

```go
func (s *CacheAsideService) Get(key string) (string, bool) {
    if v, ok := s.cache.Get(key); ok { return v, true }  // cache hit
    v, ok := s.store.Load(key)                            // cache miss
    if ok { s.cache.Put(key, v) }                         // populate
    return v, ok
}
```

## HTTP caching — ETag and 304

```go
// Server: hash response body to form ETag.
etag := fmt.Sprintf(`"%x"`, sha256.Sum256(body))
w.Header().Set("ETag", etag)

// Client: send ETag back on next request.
req.Header.Set("If-None-Match", etag)

// Server: if ETag matches, return 304 — no body sent.
if r.Header.Get("If-None-Match") == cached.ETag {
    w.WriteHeader(http.StatusNotModified)
    return
}
```

## Cache-Control directives

| Directive | Meaning |
|---|---|
| `max-age=N` | Cache the response for N seconds |
| `no-cache` | Revalidate with server before using cached copy |
| `no-store` | Do not cache at all (sensitive data) |
| `Vary: Accept` | Separate cache entries per Accept header value |

## Cache invalidation strategies

```go
// Invalidate on write — prevents stale reads.
func (c *CachedProductStore) Update(ctx context.Context, p Product) error {
    if err := c.inner.Update(ctx, p); err != nil { return err }
    c.invalidate(p.ID)  // remove from cache so next read re-fetches
    return nil
}
```

## Write strategies compared

| Strategy | Write path | Read path | Consistency | Use case |
|---|---|---|---|---|
| Cache-aside | Store only | Check cache → on miss, load & populate | Eventual (on miss) | General-purpose reads |
| Write-through | Store + Cache | Cache always has latest | Strong | Frequently-read, rarely-written |
| Write-behind | Cache only; async flush | Cache always has latest | Eventual (on flush) | Write-heavy, tolerate delay |

## Production tips

- **Choose capacity by working-set size** — caching 10 % of the hottest keys often yields 90 % of the hit-rate benefit.
- **TTL should match staleness tolerance** — user sessions: minutes; product prices: seconds to minutes; realtime data: avoid caching.
- **Always invalidate on writes** — a stale cache is worse than no cache for correctness-sensitive data.
- **Emit metrics** — track hits, misses, evictions, and TTL-expiries to detect cache thrashing.
- **Use `crypto/sha256` for ETags** — deterministic, fast, and collision-resistant enough for HTTP caching.
- **Vary: header doubles cache entries** — use it only for headers that genuinely change the response shape.
