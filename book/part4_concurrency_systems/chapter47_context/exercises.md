# Chapter 47 — Exercises

## 47.1 — Request pipeline

Run [`exercises/01_request_pipeline`](exercises/01_request_pipeline/main.go).

4-stage order pipeline: authenticate → checkInventory → calculatePrice → createOrder. Demonstrates timeout mid-pipeline, manual cancel, and business logic errors distinct from context errors.

Try:
- Add a `withSpan(ctx, name)` helper that attaches a start time and stage name to the context. On each stage completion, compute and print elapsed time for that span.
- Add `WithCause` (Go 1.20+): wrap `context.WithCancelCause(parent)` so stages can call `cancel(err)` with a specific reason, and the caller can retrieve it with `context.Cause(ctx)`.
- Run `go run -race ./exercises/01_request_pipeline` and confirm no races.

## 47.2 ★ — Context-aware retry

Build `RetryWithContext(ctx context.Context, fn func(context.Context) error, maxAttempts int, backoff time.Duration) error` that:
- Passes the same `ctx` to each `fn` call
- Stops retrying if `ctx` is cancelled (returns `ctx.Err()`)
- Waits `backoff` between attempts, but exits early if `ctx` is cancelled during the wait

Test: fn succeeds on attempt 3, ctx cancelled during attempt 2 wait.

## 47.3 ★★ — Request tracing middleware

Build a middleware stack for a simulated HTTP server:

1. `TraceMiddleware` — attaches a trace ID and start time via `WithValue`
2. `AuthMiddleware` — reads a `X-User-ID` header and attaches user to context, returns 401 if missing
3. `TimeoutMiddleware(d)` — wraps the request context with `WithTimeout`

Chain them and run 5 simulated requests: 2 normal, 1 missing auth, 1 slow (timeout), 1 cancelled. Print trace ID and latency for each.
