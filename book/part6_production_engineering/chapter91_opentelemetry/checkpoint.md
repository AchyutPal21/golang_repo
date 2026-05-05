# Chapter 91 Checkpoint — OpenTelemetry

## Concepts to know

- [ ] What are the three OTel signal types? Which chapter covers each?
- [ ] What fields are in a W3C `traceparent` header? What does the flags byte mean?
- [ ] What is the difference between a trace and a span?
- [ ] What is baggage? How does it differ from span attributes?
- [ ] Name four head-based sampling strategies and when to use each.
- [ ] What is tail-based sampling? What infrastructure does it require?
- [ ] How does `ParentBased` sampling differ from `TraceIDRatioBased`?
- [ ] What are span semantic conventions? Give three examples.
- [ ] What does `defer span.End()` guarantee?

## Code exercises

### 1. Trace context propagation

Write a function `injectHeaders(ctx context.Context) map[string]string` that extracts the current `traceparent` and `baggage` headers from context and returns them as a map, simulating what an HTTP client interceptor would do before sending a request.

### 2. Sampling math

Given `TraceIDRatioBased(0.05)` — 5% sampling — and 1 000 RPS:
- How many traces per second are recorded?
- If each trace has 8 spans averaging 2 KB, what is the storage rate per minute?

### 3. Span status

Write a wrapper `WithSpan(ctx, tracer, name, fn)` that starts a span, calls `fn(ctx)`, sets status to Error if fn returns an error, and always calls `span.End()`. Use it to wrap a database call.

## Quick reference

```bash
# Run with OTEL_EXPORTER_OTLP_ENDPOINT pointed at a collector
OTEL_SERVICE_NAME=my-svc \
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 \
go run ./...

# Jaeger all-in-one (docker)
docker run -p 16686:16686 -p 4317:4317 jaegertracing/all-in-one

# View traces at http://localhost:16686
```

## Expected answers

1. Traces (ch 91), Metrics (ch 90), Logs (ch 89).
2. `version (2 hex) - traceID (32 hex) - spanID (16 hex) - flags (2 hex)`. Flags `0x01` = sampled.
3. A trace is the entire tree of spans sharing a TraceID; a span is one unit of work within that tree.
4. Baggage is cross-cutting business context (user_id, tenant_id) propagated through every service. Attributes belong only to one span.
5. AlwaysOn (dev), AlwaysOff (disabled), RatioSampler (deterministic fraction), ParentBased (respect parent decision).
6. Tail-based sampling waits to see the full trace before deciding. Requires a collector like Jaeger or Grafana Tempo with enough buffer.
7. ParentBased follows the upstream sampling flag; if the parent was sampled, this span is too. RatioSampled decides independently per root trace.
8. `http.method`, `db.system`, `rpc.service` — defined in OTel semantic conventions.
9. `defer span.End()` ensures the span is exported even on early returns or panics.
