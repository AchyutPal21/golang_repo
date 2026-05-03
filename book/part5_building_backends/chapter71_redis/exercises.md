# Chapter 71 Exercises — Redis

## Exercise 1 — Cache Store (`exercises/01_cache_store`)

Build a Redis-backed cache with JSON serialization, bulk invalidation, and a transparent in-memory fallback.

### Cache interface

```go
type Cache interface {
    Set(ctx, key string, value any, ttl time.Duration) error
    Get(ctx, key string, dest any) error   // dest is a pointer; returns ErrCacheMiss
    Delete(ctx, key string) error
    DeleteByPrefix(ctx, prefix string) (int, error)
    Ping(ctx) error
}
```

### Implementations

**RedisCache** (`NewRedisCache(rdb *redis.Client)`)
- `Set`: JSON marshal, `SETEX key ttl jsonBytes`
- `Get`: `GET`, unmarshal; return `ErrCacheMiss` on `redis.Nil`
- `DeleteByPrefix`: `SCAN cursor prefix* 100` loop + `DEL` batches

**MemCache** (`NewMemCache()`)
- Backed by `map[string]*memEntry{data []byte, expiresAt time.Time}`
- `DeleteByPrefix`: iterate map, delete keys with `strings.HasPrefix`

**ResilientCache** (`NewResilientCache(primary, fallback Cache)`)
- On each operation: `Ping` primary; if err → use fallback
- Transparent to callers

### Demonstration

1. Create 3 products in Redis cache
2. GetByID hits, cache miss on non-existent key
3. DeleteByPrefix removes all `product:*` keys → verify miss
4. TTL: set 2s TTL, advance miniredis clock, verify expiry
5. Resilient fallback: point at unreachable Redis → writes/reads use in-memory
6. Complex JSON struct with nested slices round-trips correctly

### Hints

- `mr.FastForward(duration)` advances miniredis's internal clock for TTL testing
- `json.Marshal` / `json.Unmarshal` into/from `[]byte` for any `any` value
- Use `redis.Nil` sentinel (not `nil`) to detect cache misses
- For prefix scan loop: continue while `nextCursor != 0`
