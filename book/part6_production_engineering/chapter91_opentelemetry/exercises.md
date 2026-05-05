# Chapter 91 Exercises — OpenTelemetry

## Exercise 1 (provided): Context Propagation Pipeline

Location: `exercises/01_context_propagation/main.go`

Builds a complete in-process trace pipeline with:
- `SpanStore` — thread-safe in-memory exporter
- `Tracer` with configurable head-based sampling
- Simulated microservices: API Gateway → Order Service → DB + Cache + Payment
- Waterfall timeline renderer with ASCII bar chart
- Sampling demo: 100 requests at 10% rate

## Exercise 2 (self-directed): HTTP Trace Middleware

Build an HTTP middleware `TraceMiddleware(tracer *Tracer) func(http.Handler) http.Handler` that:
- Reads `traceparent` and `baggage` headers from the incoming request
- Creates a root span if no parent trace exists, or a child span if `traceparent` is present
- Adds `http.method`, `http.path`, `http.status_code` attributes
- Sets span status to Error for 5xx responses
- Injects `traceparent` into the response headers for downstream use

Acceptance criteria:
- Chained handlers share the same TraceID
- Status code is correctly recorded
- Missing `traceparent` header creates a new root trace

## Exercise 3 (self-directed): Metrics + Traces Correlation

Build a `MetricsTracer` that:
- Counts spans by name using `atomic.Int64` counters
- Tracks P50/P95/P99 latency per span name using a simple histogram (10 fixed buckets)
- Exposes a `Report()` method that prints per-span stats
- Integrates with the existing `SpanStore` via an export hook

## Stretch Goal: Propagation Across Goroutines

Extend the exercise tracer to support fan-out: one root span spawning N parallel child spans via goroutines, each with their own sub-spans, all sharing the same TraceID. Render the resulting waterfall sorted by start time with correct indentation showing the concurrent branches.
