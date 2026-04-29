# Chapter 32 — Exercises

## 32.1 — Middleware chain

Run [`exercises/01_middleware_chain`](exercises/01_middleware_chain/main.go).

An HTTP-style middleware stack: `RateLimit → Auth → Logging → Router`.

Try:
- Add a `TracingMiddleware` that assigns a unique request ID and includes it in every log line.
- Add a `TimeoutMiddleware` that returns a 408 if `Handle` does not return within a duration (simulate with a slow handler that `time.Sleep`s).
- Reorder the middlewares so Auth runs before RateLimit. Observe the difference when an unauthenticated request hits the rate limiter.

## 32.2 ★ — Caching composite

Build a `Node` interface with `Get(key string) (string, bool)` and `Set(key, value string)`. Implement:
- `MemNode` — in-memory map store
- `CompositeNode` — holds a slice of `Node`; `Get` tries each in order (first hit wins); `Set` writes to all

Use this to model a multi-layer cache: L1 (small map) → L2 (larger map) → miss.

## 32.3 ★★ — Transparent proxy with metrics

Implement a `MetricsProxy` that wraps any `DataLoader`. It records call count, hit/miss ratio (if the inner loader is also a `cachingProxy`), and total latency. Print a summary table after all calls complete.
