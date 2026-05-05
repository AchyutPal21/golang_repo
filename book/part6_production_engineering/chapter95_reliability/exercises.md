# Chapter 95 Exercises — Reliability Engineering

## Exercise 1 (provided): SLO Service

Location: `exercises/01_slo_service/main.go`

Complete SLO-aware service demonstrating:
- SLI collection (availability, latency p99, error rate)
- Error budget calculation and tracking
- Burn rate alerts at 1h and 6h windows
- Policy enforcement: feature freeze when budget < 10%
- Dashboard-style status report

## Exercise 2 (self-directed): Multi-Window Burn Rate

Build a burn rate alerting system:
- Input: SLO target (e.g. 99.9%), rolling error counts per minute
- Compute burn rate over: 5m, 1h, 6h, 24h windows
- Fire a `CRITICAL` alert when 1h burn rate > 14.4×
- Fire a `WARNING` alert when 6h burn rate > 6×
- Print a timeline showing alerts as they fire

## Exercise 3 (self-directed): Resilient HTTP Client

Build a `ResilientClient` wrapping `net/http` with:
- Automatic retry (max 3) with exponential backoff + jitter for 5xx/timeout errors
- Circuit breaker (open after 5 failures in 30s, recover after 10s)
- Per-request timeout (configurable)
- Context cancellation support
- Metrics: success count, failure count, circuit open duration

## Stretch Goal: Bulkhead Pattern

Implement a bulkhead that:
- Limits concurrent calls to a dependency to N (configurable)
- Maintains separate concurrency pools for "critical" and "non-critical" callers
- Returns `ErrBulkheadFull` immediately when the limit is reached (no blocking)
- Tracks queue depth and rejection rate as metrics
