# Chapter 56 — Exercises

## 56.1 — Observability

Run [`exercises/01_observability`](exercises/01_observability/main.go).

In-memory metrics registry with Counter and Histogram types exposed at `/metrics` in Prometheus text format. Correlation-ID middleware injects a UUID into the request context and response header. Consistent JSON error responses include the correlation ID. 23 requests (20 work + 3 error) produce the histogram and counter output.

Try:
- Add a `Gauge` type that supports `Set(v float64)` and `Inc()`/`Dec()`. Track active connections.
- Change the `/metrics` handler to return JSON instead of Prometheus text format.
- Add a `Summary` (p50, p90, p99) using a sliding window of the last 1000 observations.

## 56.2 ★ — Circuit breaker

Build a `CircuitBreaker` middleware:

```go
type State int
const (Closed State = iota; Open; HalfOpen)

type CircuitBreaker struct {
    failures   atomic.Int64
    threshold  int64
    timeout    time.Duration
    state      atomic.Int64 // State
    openedAt   time.Time
    mu         sync.Mutex
}
```

- **Closed**: requests pass through normally
- **Open**: after `threshold` consecutive failures, return `503` immediately for `timeout` duration
- **Half-Open**: after `timeout`, allow one probe request; if it succeeds → Closed; if it fails → Open again

Test with a backend that fails 5 times then recovers.

## 56.3 ★★ — Full production server

Combine everything from Ch53–56 into a single binary:

- TLS with a self-signed cert (Ch55)
- HTTP/2 auto-upgrade (Ch55)
- REST API for a resource of your choice (Ch54)
- Rate limiting + timeout middleware (Ch56)
- Structured logging + correlation IDs (Ch56)
- `/health`, `/ready`, `/metrics` endpoints (Ch56)
- Graceful SIGTERM shutdown (Ch56)

Write a test harness that starts the server, sends 50 concurrent requests, and asserts the metrics endpoint shows the correct request counts.
