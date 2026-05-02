# Chapter 59 — Exercises

## 59.1 — Middleware Suite

Run [`exercises/01_middleware_suite`](exercises/01_middleware_suite/main.go).

Full composable middleware stack: correlation ID (outermost), panic recovery, JSON structured logger, per-IP token bucket rate limiter, bearer auth, require-auth gate, and role gate. Four API routes exercise all layers. Tests demonstrate correct ordering — correlation ID appears in recovery error bodies because `corrIDMW` wraps `panicRecovery`.

Try:
- Move `corrIDMW` inside `panicRecovery` and observe that correlation IDs disappear from panic error responses.
- Add a `timeoutMW(200ms)` middleware and a slow handler; observe the 504 in the log output.
- Extend per-IP rate limiter to evict stale buckets (last seen > 5 minutes) using a background goroutine.

## 59.2 ★ — Conditional middleware

Build a `Conditional` middleware that applies a middleware only when a predicate is true:

```go
// Apply authMW only for non-GET requests:
Conditional(authMW, func(r *http.Request) bool {
    return r.Method != http.MethodGet
})
```

Use it to build an API where `GET /items` is public, but `POST /items` requires auth — without registering two separate route handlers.

## 59.3 ★★ — Request deduplication

Implement a deduplication middleware using an in-memory store:

- Clients include an `Idempotency-Key: <uuid>` header on `POST` and `PUT` requests
- On first request: process normally, cache `(key → status + body)` for 24 hours
- On repeat with the same key: return the cached response immediately (200 or 201 as original) with an `X-Idempotent-Replay: true` header
- Requests without an `Idempotency-Key` are never deduplicated

Test by sending the same POST twice and asserting the second response is identical to the first without the handler running again (track handler invocation with an atomic counter).
