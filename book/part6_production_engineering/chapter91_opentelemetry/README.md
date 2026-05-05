# Chapter 91 — OpenTelemetry

OpenTelemetry (OTel) is the CNCF standard for distributed tracing, metrics, and logs. Go's SDK lets you instrument services without vendor lock-in, export to Jaeger/Tempo/Grafana, and propagate trace context across HTTP and gRPC boundaries.

## Topics

| # | Topic | Key ideas |
|---|-------|-----------|
| 1 | Traces & Spans | TraceID, SpanID, parent-child, attributes, events, status |
| 2 | Baggage & Sampling | W3C traceparent, baggage header, head-based ratio sampling |
| E | Context Propagation | Full in-process pipeline: root → child spans → waterfall view |

## Examples

### `examples/01_traces_spans`

Simulates the OTel trace/span API without an OTLP collector:

- `Tracer.Start` — creates child or root spans from context
- `Span.SetAttribute`, `AddEvent`, `SetStatus`, `End`
- Parent-child depth rendering
- Concurrent traces with unique TraceIDs

### `examples/02_baggage_sampling`

Context propagation headers and sampling:

- `Baggage` — cross-cutting key-value map (user_id, tenant_id)
- W3C `traceparent` encode/parse: `00-<traceID>-<spanID>-<flags>`
- `AlwaysOn`, `AlwaysOff`, `RatioSampler`, `ParentBasedSampler`
- Sampling simulation: 10 000 traces at various rates

### `exercises/01_context_propagation`

Full trace pipeline with in-memory exporter:

- `SpanStore` — thread-safe span collector
- `Tracer` with configurable sample rate
- Simulated microservices: API Gateway → Order Service → DB + Cache + Payment
- Waterfall timeline renderer

## Key Concepts

**Trace anatomy**
```
Trace (shared TraceID)
  └─ root span: HTTP GET /api/orders        ← entry point
       └─ child: service.processOrder       ← service layer
            ├─ child: db.query              ← leaf span
            └─ child: cache.get             ← leaf span
```

**W3C traceparent header**
```
traceparent: 00-4bf92f3577b34da6a3ce929d-00f067aa0ba902b7-01
             ^^  ^^^^^^^^^^^^^^^^^^^^^^^^  ^^^^^^^^^^^^^^^^  ^^
         version       trace ID (128-bit)  span ID (64-bit)  flags
flags: 0x01 = sampled, 0x00 = not sampled
```

**Sampling strategies**

| Strategy | When to use |
|----------|-------------|
| AlwaysOn | Dev/test — 100% capture |
| AlwaysOff | Disable tracing |
| TraceIDRatioBased(0.01) | 1% production sampling |
| ParentBased(ratio) | Respect upstream decision (recommended) |
| Tail-based | Sample after seeing full trace (Jaeger/Tempo) |

**Span semantic conventions**
```
http.method, http.status_code, http.url
db.system, db.statement, db.name
messaging.system, messaging.destination
rpc.system, rpc.method, rpc.service
```

## Real OTel SDK Setup

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

exporter, _ := otlptracegrpc.New(ctx,
    otlptracegrpc.WithEndpoint("collector:4317"),
)
tp := sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter),
    sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.01))),
)
otel.SetTracerProvider(tp)
defer tp.Shutdown(ctx)

tracer := otel.Tracer("my-service")
ctx, span := tracer.Start(ctx, "operation-name")
defer span.End()
```

## Running

```bash
go run ./book/part6_production_engineering/chapter91_opentelemetry/examples/01_traces_spans
go run ./book/part6_production_engineering/chapter91_opentelemetry/examples/02_baggage_sampling
go run ./book/part6_production_engineering/chapter91_opentelemetry/exercises/01_context_propagation
```
