# Chapter 45 — Exercises

## 45.1 — Thread-safe TTL cache

Run [`exercises/01_cache`](exercises/01_cache/main.go).

Generic `Cache[K,V]` with TTL expiry using `RWMutex`, `LazyLoader` with `sync.Once` per key, and a `Purge` method.

Try:
- Add a background goroutine that calls `Purge` every 30 seconds (started lazily with `sync.Once` on first `Set`). Use a `done` channel to stop it.
- Add a `GetOrSet(k K, fn func() V, ttl time.Duration) V` method that atomically reads the cache and, if missing, calls `fn` under the write lock and stores the result.
- Run `go run -race ./exercises/01_cache` and confirm no races are detected.

## 45.2 ★ — Bounded semaphore with Cond

Implement `BoundedSemaphore` using `sync.Mutex` + `sync.Cond` (not a channel):
- `Acquire(n int)` — waits until `n` slots are available, then reserves them
- `Release(n int)` — returns `n` slots and signals waiters
- `Available() int` — returns current available count

Test with 10 goroutines each acquiring 3 of 10 total slots. Verify peak concurrent holders never exceeds capacity.

## 45.3 ★★ — Object pool with stats

Extend `sync.Pool` with a wrapper `StatsPool[T]` that tracks:
- `gets int64` — total Get calls
- `puts int64` — total Put calls
- `news int64` — times New was called (cache miss)

Use it for a `bytes.Buffer` pool in a benchmark that formats 10,000 JSON strings concurrently. Report pool hit rate.
