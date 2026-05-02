# Chapter 56 — Production HTTP Server

## What you will learn

- Complete middleware stack: recovery, logging, rate limiting, per-request timeout
- Structured logging with `log/slog` — JSON or text format, request metadata
- Panic recovery with `debug.Stack()` — never crash the process on handler panics
- Health (`/health`) and readiness (`/ready`) probes — the distinction between "alive" and "ready to serve"
- Graceful shutdown: `srv.Shutdown(ctx)` triggered by OS signals (`SIGTERM`, `SIGINT`)
- In-memory Prometheus-style metrics: counters, histograms, `/metrics` endpoint
- Correlation IDs: inject a UUID into context, propagate in response headers
- Consistent JSON error responses with correlation ID

---

## Middleware order matters

```
Request →  recovery → logging → rate-limit → timeout → mux → handler
Response ← recovery ← logging ← rate-limit ← timeout ← mux ← handler
```

- **Recovery** is outermost — it catches panics from all inner layers
- **Logging** is second — it records the final status after the inner chain runs
- **Rate limit** is before business logic — reject early, log the rejection
- **Timeout** wraps just the mux — health probes can bypass it

---

## OS signal handling

```go
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
<-quit

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
srv.Shutdown(ctx)
```

Kubernetes sends `SIGTERM` before killing a pod. The 30-second window lets in-flight requests drain before the process exits.

---

## Health vs readiness

| Probe | Path | Returns OK when |
|---|---|---|
| **Liveness** | `/health` | Process is running (always 200 unless dead) |
| **Readiness** | `/ready` | Server has finished startup and is ready for traffic |

Kubernetes restarts pods that fail liveness; it removes pods from the load balancer rotation when they fail readiness. Never return a 5xx from `/health`.

---

## `log/slog` structured logging

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger.Info("request",
    "method", r.Method,
    "path",   r.URL.Path,
    "status", rec.status,
    "latency_ms", time.Since(start).Milliseconds(),
)
```

Output: `{"time":"...","level":"INFO","msg":"request","method":"GET","path":"/","status":200,"latency_ms":3}`

---

## Production `http.Server` timeouts

```go
srv := &http.Server{
    ReadTimeout:  10 * time.Second,  // max time to read request headers + body
    WriteTimeout: 15 * time.Second,  // max time to write response
    IdleTimeout:  120 * time.Second, // max time between requests on keep-alive conn
}
```

---

## Examples

| File | Demonstrates |
|---|---|
| `examples/01_production_server/main.go` | Recovery, structured logging, rate limit, timeout, health/ready, OS signal shutdown |

## Exercise

`exercises/01_observability/main.go` — Prometheus-style registry (Counter + Histogram), correlation-ID middleware, structured JSON errors, `/metrics` endpoint.
