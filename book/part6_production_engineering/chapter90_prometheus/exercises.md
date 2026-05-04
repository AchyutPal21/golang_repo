# Chapter 90 Exercises — Prometheus Metrics

## Exercise 1 (provided): Label Cardinality

Location: `exercises/01_cardinality/main.go`

Implements `LabelPolicy`, `SafeCounter`, and demonstrates how 100k users
collapse to 2 series with an overflow sentinel.

## Exercise 2 (self-directed): Apdex Score

Implement an Apdex score calculator from a Histogram:

```
Apdex(T) = (Satisfied + Tolerating/2) / Total
  Satisfied  = count with latency <= T
  Tolerating = count with T < latency <= 4T
  Frustrated = count with latency > 4T (implicit)
```

Target T = 100ms. Build a `func Apdex(h *Histogram, targetSec float64) float64`
and verify it produces values in [0, 1].

## Exercise 3 (self-directed): Metrics Middleware

Build an `HTTPMetricsMiddleware` that:
- Instruments every handler with `http_requests_total` (counter with method/path/status)
- Records `http_request_duration_seconds` (histogram with default buckets)
- Normalises paths with IDs to parameterised form: `/api/users/42` → `/api/users/:id`
- Uses the in-process metric types from Example 1

Test by calling 1000 simulated requests and verifying the rendered metrics match.

## Stretch Goal: Push-based Metrics Gateway

Implement a simple in-process "Pushgateway" that:
- Accepts `Metric` structs via a channel
- Merges them into a registry (label-based dedup)
- Exposes a `Snapshot()` method returning the current text-format exposition
- Handles counter resets (detects decrease, adds offset)
