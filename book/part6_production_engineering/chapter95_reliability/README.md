# Chapter 95 — Reliability Engineering

Reliability is not an afterthought — it is designed into the service from day one through SLOs, error budgets, and patterns like circuit breakers that prevent cascading failures.

## Topics

| # | Topic | Key ideas |
|---|-------|-----------|
| 1 | SLOs & SLIs | Availability, latency, error rate targets and measurement |
| 2 | Circuit Breaker | Closed/Open/HalfOpen states, error threshold, backoff |
| E | Error Budget | SLO tracking, burn rate alert, budget exhaustion |

## Examples

### `examples/01_slo_sli`

SLO measurement framework:
- `SLI` types: availability, latency, error rate
- Rolling window measurement (5-minute, 1-hour windows)
- SLO compliance tracking
- Burn rate calculation

### `examples/02_circuit_breaker`

Circuit breaker state machine:
- `Closed` → `Open` on threshold breach
- `Open` → `HalfOpen` after timeout
- `HalfOpen` → `Closed` on successful probe
- Configurable: error threshold, open duration, probe count

### `exercises/01_slo_service`

Complete SLO-aware HTTP service:
- SLI collection from request metrics
- Error budget dashboard
- Burn rate alerts (1h, 6h windows)
- Policy: freeze changes when budget is < 10%

## Key Concepts

**SLO hierarchy**
```
SLA (legal commitment)
 └─ SLO (internal target, stricter than SLA)
      └─ SLI (measurement: availability, latency p99, error rate)
```

**Error budget**
```
Error budget = 1 - SLO target
  e.g. 99.9% SLO → 0.1% error budget = 43.8 min/month allowed downtime

Burn rate = actual error rate / error budget rate
  burn rate 1.0 = consuming at exactly SLO rate
  burn rate 2.0 = will exhaust budget in half the window
```

**Alert windows (Google SRE Book)**
| Window | Burn rate | Alert when |
|--------|-----------|-----------|
| 1h | > 14.4× | Severe: budget gone in < 1h |
| 6h | > 6× | Fast: gone in < 6h |
| 24h | > 3× | Slow: investigate |
| 3d | > 1× | Trend: planning needed |

**Circuit breaker states**
```
[Closed] → error rate > threshold → [Open]
[Open]   → after timeout          → [HalfOpen]
[HalfOpen] → success              → [Closed]
[HalfOpen] → failure              → [Open]
```

## Running

```bash
go run ./book/part6_production_engineering/chapter95_reliability/examples/01_slo_sli
go run ./book/part6_production_engineering/chapter95_reliability/examples/02_circuit_breaker
go run ./book/part6_production_engineering/chapter95_reliability/exercises/01_slo_service
```
