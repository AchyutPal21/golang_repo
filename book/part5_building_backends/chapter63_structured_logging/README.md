# Chapter 63 — Structured Logging

## What you'll learn

How to produce machine-parseable log output from Go services using the standard `log/slog` package: levels, structured key-value attributes, child loggers, groups, context propagation, custom handlers, and HTTP middleware that injects a per-request logger into context.

## Why structured logging matters

Plain text logs break in production. You can't query them in Loki, CloudWatch Insights, or Datadog without expensive regex parsing. Structured logs emit JSON objects where every field is a first-class queryable attribute — filter `status=500` or join on `request_id` across services in milliseconds.

## Key concepts

| Concept | API |
|---|---|
| JSON handler | `slog.NewJSONHandler(w, opts)` |
| Text handler | `slog.NewTextHandler(w, opts)` |
| Typed attributes | `slog.String`, `slog.Int`, `slog.Bool`, `slog.Any` |
| Child logger | `logger.With(attrs...)` — pre-attaches fields to all future records |
| Group | `slog.Group("req", slog.String("method","GET"), ...)` — namespaced nested object |
| Context logging | `logger.InfoContext(ctx, msg, attrs...)` — handler can extract ctx values |
| Custom handler | Implement `slog.Handler`: `Enabled`, `Handle`, `WithAttrs`, `WithGroup` |

## Files

| File | Topic |
|---|---|
| `examples/01_slog_basics/main.go` | Handlers, levels, child loggers, groups, context, custom handlers |
| `examples/02_log_middleware/main.go` | Per-request child logger injected via context; request/response logging |
| `exercises/01_observability_logger/main.go` | Multi-handler fan-out, slow request detection, 4xx sampling, domain events |

## Child logger pattern

```go
// At request entry — attach request-level fields once
reqLogger := base.With(
    slog.String("request_id", requestID),
    slog.String("method", r.Method),
    slog.String("path", r.URL.Path),
)
// Store in context
ctx = withLogger(ctx, reqLogger)

// Inside any handler — pull from context, call normally
log := loggerFromCtx(r.Context())
log.Info("order created", slog.String("order_id", o.ID))
// ↑ also carries request_id, method, path automatically
```

## Custom handler skeleton

```go
type myHandler struct {
    inner slog.Handler
}

func (h *myHandler) Enabled(ctx context.Context, level slog.Level) bool {
    return h.inner.Enabled(ctx, level)
}

func (h *myHandler) Handle(ctx context.Context, r slog.Record) error {
    r.AddAttrs(slog.String("service", "my-svc")) // inject before delegating
    return h.inner.Handle(ctx, r)
}

func (h *myHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    return &myHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *myHandler) WithGroup(name string) slog.Handler {
    return &myHandler{inner: h.inner.WithGroup(name)}
}
```

## Production tips

- Set `Level: slog.LevelInfo` for production — `Debug` is too noisy.
- Use `io.Discard` handler in tests to silence log output without changing code.
- Keep attribute keys lowercase and `snake_case` — consistent across services.
- Use `slog.Group` to namespace related fields (e.g. `request.*`, `db.*`).
- Never log secrets, PII, or auth tokens — add a sanitizing middleware handler if needed.
