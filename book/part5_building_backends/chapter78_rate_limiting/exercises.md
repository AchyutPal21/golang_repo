# Chapter 78 Exercises — Rate Limiting, Circuit Breakers, Retries

## Exercise 1 — API Resilience (`exercises/01_api_resilience`)

Build a resilience policy that combines a token-bucket rate limiter, a circuit breaker, and a retry-with-backoff strategy into a single `Execute` method.

### Components

**`TokenBucket`**:
```go
func NewTokenBucket(capacity, ratePerSec float64) *TokenBucket
func (tb *TokenBucket) Allow() bool
```

**`CircuitBreaker`**:
```go
func NewCB(failThreshold, successThreshold int, openTimeout time.Duration) *CircuitBreaker
func (cb *CircuitBreaker) Call(fn func() error) error
func (cb *CircuitBreaker) State() CBState  // 0=Closed, 1=Open, 2=HalfOpen
```

**`RetryConfig`**:
```go
type RetryConfig struct {
    MaxAttempts  int
    InitialDelay time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
    RetryOn      func(err error) bool
}
func Retry(ctx context.Context, cfg RetryConfig, fn func() error) error
```

**`ResiliencePolicy`**:
```go
type ResiliencePolicy struct {
    RateLimiter *TokenBucket
    CB          *CircuitBreaker
    RetryConfig RetryConfig
    Stats       struct {
        Allowed      atomic.Int64
        RateLimited  atomic.Int64
        CBOpen       atomic.Int64
        Errors       atomic.Int64
    }
}
func (p *ResiliencePolicy) Execute(ctx context.Context, fn func() error) error
```

`Execute` applies: rate check → retry wrapper → circuit breaker → fn.

### Behaviour rules

- If rate limited, return error immediately (don't retry)
- If circuit open, return `ErrOpen` immediately (don't consume retry budget)
- `RetryOn` should NOT retry `ErrOpen` — circuit open means stop sending
- `RetryConfig.MaxAttempts=1` means no retry (call fn exactly once)

### Demonstration

1. **Rate limiting only**: send 8 requests; bucket allows first 5; verify 3 rate-limited
2. **Circuit breaker + retry**: point at a 100%-failing service; verify CB trips after failThreshold; subsequent calls rejected as `ErrOpen`
3. **Full policy**: mixed service (fails first N calls, then succeeds); verify eventual success after CB recovers

### Hints

- `Retry` should use `time.After(delay + jitter)` in a `select` with `ctx.Done()`
- Circuit breaker's HalfOpen state allows only ONE probe at a time; all others return `ErrOpen`
- `ResiliencePolicy.Execute` wraps the user's fn: rate_check → retry(cb.Call(fn))
- Track `CBOpen` by checking `errors.Is(err, ErrOpen)` after `Retry` returns
- Use `atomic.Int64` for all stats to support concurrent `Execute` calls
