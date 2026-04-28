# Chapter 23 — Exercises

## 23.1 — Middleware stack

Run [`exercises/01_middleware_stack`](exercises/01_middleware_stack/main.go).

The `Chain` function wraps a handler with a list of middleware factories.
Logging runs outermost (first in, last out), Auth runs before the core handler.

Try:
- Add a `RateLimitMiddleware` that rejects more than 3 requests per second.
- Add a `CachingMiddleware` that returns a cached response on subsequent identical requests.
- Rewrite `Chain` to accept `...func(Handler) Handler` (already done) — explain why this is more composable than embedding middleware structs directly.

## 23.2 ★ — Embedded interface for partial implementation

Create a `Storage` interface with `Get`, `Put`, `Delete`, and `List`.
Implement a `ReadOnlyStorage` that embeds `Storage` but overrides `Put` and `Delete` to return `ErrReadOnly`.
A concrete `MemoryStorage` provides the full implementation.
Use `ReadOnlyStorage` to wrap `MemoryStorage` and verify that writes are rejected.
