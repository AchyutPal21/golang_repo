# Chapter 78 — Rate Limiting, Circuit Breakers, Retries

## What you'll learn

How to build resilient systems: four rate-limiting algorithms, the circuit breaker state machine, exponential backoff with jitter, hedged requests, retry budgets, and a combined resilience policy.

## Key concepts

| Concept | Description |
|---|---|
| Token bucket | Burst up to capacity, refill at fixed rate/s |
| Sliding window | Accurate per-key count over rolling time window |
| Fixed window | Simple counter; resets at boundary; burst edge risk |
| Leaky bucket | Smooths output to constant rate; drops excess |
| Circuit breaker | Closed → Open (failure threshold) → Half-Open → Closed |
| Exponential backoff | Delay doubles each retry; reduces thundering herd |
| Jitter | Random ±% added to delay; prevents synchronised retry waves |
| Hedged request | Send duplicate after delay; use first response; cancel loser |
| Retry budget | Global token bucket capping total retries across all requests |

## Files

| File | Topic |
|---|---|
| `examples/01_rate_limiters/main.go` | Token bucket, sliding window, fixed window, leaky bucket, per-key |
| `examples/02_circuit_breaker/main.go` | Circuit breaker state machine, backoff+jitter, hedged requests, retry budget |
| `exercises/01_api_resilience/main.go` | Rate limiter + CB + retry combined policy, failure mode summary |

## Token bucket

```go
// Allow burst up to capacity; refill at ratePerSec tokens/second.
func (tb *TokenBucket) Allow() bool {
    now := time.Now()
    tb.tokens += now.Sub(tb.lastFill).Seconds() * tb.rate
    if tb.tokens > tb.capacity { tb.tokens = tb.capacity }
    tb.lastFill = now
    if tb.tokens < 1 { return false }
    tb.tokens--
    return true
}
```

## Circuit breaker

```go
cb := NewCircuitBreaker(CBConfig{
    FailureThreshold: 5,    // 5 consecutive failures → Open
    SuccessThreshold: 2,    // 2 successes in HalfOpen → Closed
    OpenTimeout:      30 * time.Second,
})

err := cb.Call(func() error {
    return downstreamService.Call(ctx)
})
if errors.Is(err, ErrCircuitOpen) {
    // Return cached data or 503
}
```

## Exponential backoff with jitter

```go
delay := initialDelay
for attempt := range maxAttempts {
    err := call()
    if err == nil { return nil }
    jitter := delay * jitterFraction * (rand.Float64()*2 - 1)
    time.Sleep(delay + time.Duration(jitter))
    delay = min(delay * multiplier, maxDelay)
}
```

## Retry policy — what to retry

```go
RetryOn: func(err error) bool {
    // Retry on transient errors only.
    return errors.Is(err, ErrUnavailable) || errors.Is(err, ErrTimeout)
    // Do NOT retry: 400 Bad Request, 401 Unauthorized, 404 Not Found
}
```

## HTTP middleware pattern

```go
func rateLimitMiddleware(limiter *TokenBucket) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !limiter.Allow() {
                w.Header().Set("Retry-After", "1")
                http.Error(w, "rate limited", http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

## Production notes

- Use `golang.org/x/time/rate` for a production-grade token bucket
- Use `github.com/sony/gobreaker` for a well-tested circuit breaker
- Always add jitter to backoff — without it, all retries arrive simultaneously
- Set a `Retry-After` header when returning 429 — give the client the right delay
- Export circuit breaker state as a metric — alerts when state = Open > 30s
- Retry budget prevents cascade: if 10% of requests fail, retries at 3x amplify load by 30%
