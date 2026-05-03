# Chapter 63 Exercises — Structured Logging

## Exercise 1 — Observability Logger (`exercises/01_observability_logger`)

Build a production-grade observability logging layer for an HTTP order API.

### Requirements

**Multi-handler setup**
- Write `DEBUG+` records to `os.Stderr` in text format (developer console)
- Write `INFO+` records to `os.Stdout` in JSON format (log aggregator)
- Implement a `multiHandler` that fans records out to all registered handlers

**Observability middleware**
- Inject a child logger per request with: `trace_id`, `method`, `path`, `user_agent`
- Set an `X-Trace-Id` response header with the same trace ID
- Log "request started" at INFO on entry
- On exit, log "request completed" with `status`, `latency_ms`, `bytes`
- Warn if handler duration exceeds 5ms (`"slow request detected"` with threshold)
- Log 100% of 5xx responses at ERROR; sample ~10% of 4xx at WARN (reduce noise)

**Domain events**
- `order created` — log `order_id`, `item`, `amount_cents`, `status`
- `order fetched` — same fields
- `order shipped` — include `prev_status` alongside current status

**Discard logger**
- Expose a `newDiscardLogger()` function that returns a logger writing to `io.Discard`
- Demonstrate it in `main()` to show test usage pattern

### Endpoints to implement

| Method | Path | Response |
|---|---|---|
| `GET` | `/orders/{id}` | 200 order JSON / 404 |
| `POST` | `/orders` | 201 created order / 422 if item missing |
| `POST` | `/orders/{id}/ship` | 200 shipped / 409 if already shipped / 404 |
| `GET` | `/slow` | 200 — sleeps 10ms to trigger slow warning |

### Expected output characteristics

- DEBUG text lines (slow handler start) appear on stderr only
- JSON lines appear on stdout for all INFO+ events
- Each request's log lines share the same `trace_id`
- The `/slow` request emits a WARN "slow request detected" line
- State-transition conflict emits WARN "invalid state transition"

### Hints

- Use `r.Clone()` inside `multiHandler.Handle` — `slog.Record` is not safe to share across handlers
- `rand.Intn(10) == 0` gives ~10% probability for 4xx sampling
- Use `time.Since(start)` after `next.ServeHTTP` returns for latency measurement
