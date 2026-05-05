# Chapter 95 Checkpoint — Reliability Engineering

## Concepts to know

- [ ] What is the difference between SLA, SLO, and SLI?
- [ ] How do you calculate error budget from an SLO target?
- [ ] What is burn rate? What burn rate triggers a page?
- [ ] What are the three states of a circuit breaker? Describe the transitions.
- [ ] Why does a circuit breaker help during cascading failures?
- [ ] What is the difference between retry and circuit breaker? When do you use each?
- [ ] What is an error budget policy? Give two examples.
- [ ] What is multi-window, multi-burn-rate alerting?
- [ ] Name three reliability patterns beyond circuit breaker.

## Code exercises

### 1. Error budget calculator

Write a function `errorBudgetMinutes(sloPercent float64, windowDays int) float64` that returns how many minutes of downtime are allowed in the given window.

### 2. Circuit breaker

Write a `CircuitBreaker` that:
- Opens after 5 consecutive failures
- Waits 10 seconds before attempting recovery (HalfOpen)
- Requires 2 consecutive successes to close again
- Returns `ErrCircuitOpen` immediately when open

### 3. Burn rate alert

Given: 99.9% SLO, 30-day window, and observed error counts per hour for 24 hours.
Calculate: the 1-hour burn rate and determine if a page should fire (threshold: 14.4×).

## Quick reference

```
# 30-day error budget at various SLOs
SLO 99.0%  → 432 min / month (7h 12m)
SLO 99.5%  → 216 min / month (3h 36m)
SLO 99.9%  → 43.2 min / month
SLO 99.95% → 21.6 min / month
SLO 99.99% → 4.32 min / month

# Burn rate at SLO 99.9%: error budget rate = 0.1% / 30d = 0.00139%/h
# If actual error rate = 1%/h: burn rate = 1 / 0.00139 ≈ 720× (catastrophic)
```

## Expected answers

1. SLA: legal contract with customers (weaker). SLO: internal target (stricter). SLI: actual measurement.
2. Error budget = (1 - SLO%) × window duration. E.g., 99.9% SLO over 30 days = 0.001 × 43200 min = 43.2 min.
3. Burn rate = actual error rate / error budget rate. Typical page threshold: 14.4× (budget exhausted in 1h) or 6× (in 6h).
4. Closed (normal) → Open (too many errors) → HalfOpen (probe attempt) → Closed (probe succeeds) or Open (probe fails).
5. Open circuit stops forwarding requests to an unhealthy dependency, giving it time to recover and preventing thread/goroutine exhaustion.
6. Retry: for transient network errors, with exponential backoff. Circuit breaker: when the downstream is persistently unhealthy — retries would make it worse.
7. Error budget policy examples: "Feature freeze when budget < 10%" and "Rollback changes when budget burn > 6× for 1h".
8. Multi-window alerting uses short windows (1h, 6h) for fast detection and long windows (24h, 3d) for slow burns, reducing false positive rate.
9. Bulkhead (isolate resources), retry with jitter, timeout, rate limiter, health check, graceful degradation.
