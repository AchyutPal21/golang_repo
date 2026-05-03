// FILE: book/part5_building_backends/chapter78_rate_limiting/exercises/01_api_resilience/main.go
// CHAPTER: 78 — Rate Limiting, Circuit Breakers, Retries
// TOPIC: Resilient API client — per-endpoint rate limiting, circuit breaker,
//        retry with backoff, and a combined resilience middleware.
//
// Run (from the chapter folder):
//   go run ./exercises/01_api_resilience

package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// TOKEN BUCKET (reused from examples)
// ─────────────────────────────────────────────────────────────────────────────

type TokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	capacity float64
	rate     float64
	lastFill time.Time
}

func NewTokenBucket(capacity, ratePerSec float64) *TokenBucket {
	return &TokenBucket{tokens: capacity, capacity: capacity, rate: ratePerSec, lastFill: time.Now()}
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(tb.lastFill).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}
	tb.lastFill = now
	if tb.tokens < 1 {
		return false
	}
	tb.tokens--
	return true
}

// ─────────────────────────────────────────────────────────────────────────────
// CIRCUIT BREAKER
// ─────────────────────────────────────────────────────────────────────────────

type CBState int

const (
	StateClosed   CBState = 0
	StateOpen     CBState = 1
	StateHalfOpen CBState = 2
)

var ErrOpen = errors.New("circuit open")

type CircuitBreaker struct {
	mu               sync.Mutex
	state            CBState
	failThreshold    int
	successThreshold int
	openTimeout      time.Duration
	failures         int
	successes        int
	openedAt         time.Time
}

func NewCB(failThreshold, successThreshold int, openTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failThreshold:    failThreshold,
		successThreshold: successThreshold,
		openTimeout:      openTimeout,
	}
}

func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	switch cb.state {
	case StateOpen:
		if time.Since(cb.openedAt) < cb.openTimeout {
			cb.mu.Unlock()
			return ErrOpen
		}
		cb.state = StateHalfOpen
		cb.failures = 0
		cb.successes = 0
	case StateHalfOpen:
		cb.mu.Unlock()
		return ErrOpen // one probe at a time
	}
	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()
	if err != nil {
		cb.failures++
		cb.successes = 0
		if cb.state == StateHalfOpen || cb.failures >= cb.failThreshold {
			cb.state = StateOpen
			cb.openedAt = time.Now()
		}
	} else {
		cb.failures = 0
		cb.successes++
		if cb.state == StateHalfOpen && cb.successes >= cb.successThreshold {
			cb.state = StateClosed
		}
	}
	return err
}

func (cb *CircuitBreaker) State() CBState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// ─────────────────────────────────────────────────────────────────────────────
// RETRY WITH BACKOFF
// ─────────────────────────────────────────────────────────────────────────────

type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	RetryOn      func(err error) bool
}

func Retry(ctx context.Context, cfg RetryConfig, fn func() error) error {
	delay := cfg.InitialDelay
	var lastErr error
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if cfg.RetryOn != nil && !cfg.RetryOn(lastErr) {
			return lastErr
		}
		if attempt == cfg.MaxAttempts {
			break
		}
		// Add 10% jitter.
		jitter := time.Duration(float64(delay) * 0.1 * rand.Float64())
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay + jitter):
		}
		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}
	return fmt.Errorf("after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

// ─────────────────────────────────────────────────────────────────────────────
// RESILIENCE POLICY — combines rate limiter + circuit breaker + retry
// ─────────────────────────────────────────────────────────────────────────────

type ResiliencePolicy struct {
	RateLimiter    *TokenBucket
	CB             *CircuitBreaker
	RetryConfig    RetryConfig
	Stats          struct {
		Allowed   atomic.Int64
		RateLimited atomic.Int64
		CBOpen    atomic.Int64
		Retries   atomic.Int64
		Errors    atomic.Int64
	}
}

func (p *ResiliencePolicy) Execute(ctx context.Context, fn func() error) error {
	if p.RateLimiter != nil && !p.RateLimiter.Allow() {
		p.Stats.RateLimited.Add(1)
		return fmt.Errorf("rate limited")
	}
	p.Stats.Allowed.Add(1)

	err := Retry(ctx, p.RetryConfig, func() error {
		if p.CB != nil {
			return p.CB.Call(fn)
		}
		return fn()
	})

	if err != nil {
		if errors.Is(err, ErrOpen) {
			p.Stats.CBOpen.Add(1)
		} else {
			p.Stats.Errors.Add(1)
		}
	}
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// EXTERNAL SERVICE MOCK
// ─────────────────────────────────────────────────────────────────────────────

type MockService struct {
	FailRate    float64
	Latency     time.Duration
	CallCount   atomic.Int64
	ErrorCount  atomic.Int64
}

func (s *MockService) Call(ctx context.Context) error {
	s.CallCount.Add(1)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(s.Latency):
	}
	if rand.Float64() < s.FailRate {
		s.ErrorCount.Add(1)
		return fmt.Errorf("service error")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== API Resilience Exercise ===")
	fmt.Println()

	ctx := context.Background()

	// ── RATE LIMITER ONLY ─────────────────────────────────────────────────────
	fmt.Println("--- Rate limiter (5 burst, 10/s) ---")
	policy := &ResiliencePolicy{
		RateLimiter: NewTokenBucket(5, 10),
		RetryConfig: RetryConfig{MaxAttempts: 1},
	}
	svc := &MockService{FailRate: 0, Latency: time.Millisecond}

	for i := 0; i < 8; i++ {
		err := policy.Execute(ctx, func() error {
			return svc.Call(ctx)
		})
		status := "ok"
		if err != nil {
			status = err.Error()
		}
		fmt.Printf("  call %d: %s\n", i+1, status)
	}
	fmt.Printf("  allowed=%d rate_limited=%d\n", policy.Stats.Allowed.Load(), policy.Stats.RateLimited.Load())

	// ── CIRCUIT BREAKER + RETRY ───────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Circuit breaker (threshold=3) + retry (max=2) ---")

	cb := NewCB(3, 2, 80*time.Millisecond)
	policy2 := &ResiliencePolicy{
		CB: cb,
		RetryConfig: RetryConfig{
			MaxAttempts:  2,
			InitialDelay: 5 * time.Millisecond,
			MaxDelay:     20 * time.Millisecond,
			Multiplier:   2,
			RetryOn:      func(err error) bool { return !errors.Is(err, ErrOpen) },
		},
	}
	failingSvc := &MockService{FailRate: 1.0, Latency: time.Millisecond}

	// Trip the circuit.
	for i := 1; i <= 5; i++ {
		err := policy2.Execute(ctx, func() error {
			return failingSvc.Call(ctx)
		})
		fmt.Printf("  call %d: state=%d err=%v\n", i, cb.State(), err != nil)
	}
	fmt.Printf("  cb state after failures: %d (1=open)\n", cb.State())

	// ── FULL RESILIENCE POLICY ────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Full policy: rate limiter + CB + retry ---")

	cb3 := NewCB(2, 1, 30*time.Millisecond)
	policy3 := &ResiliencePolicy{
		RateLimiter: NewTokenBucket(10, 20),
		CB:          cb3,
		RetryConfig: RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 2 * time.Millisecond,
			MaxDelay:     10 * time.Millisecond,
			Multiplier:   2,
			RetryOn:      func(err error) bool { return !errors.Is(err, ErrOpen) },
		},
	}

	var callNum atomic.Int32
	mixedSvc := func() error {
		n := callNum.Add(1)
		if n <= 4 {
			return fmt.Errorf("transient error")
		}
		return nil // recovers after 4 failures
	}

	results := make(map[string]int)
	for i := 0; i < 10; i++ {
		err := policy3.Execute(ctx, mixedSvc)
		if err == nil {
			results["ok"]++
		} else if errors.Is(err, ErrOpen) {
			results["cb_open"]++
		} else {
			results["error"]++
		}
		time.Sleep(10 * time.Millisecond) // allow CB to recover
	}
	fmt.Printf("  results: %v\n", results)
	fmt.Printf("  policy3 stats: allowed=%d rate_limited=%d cb_open=%d errors=%d\n",
		policy3.Stats.Allowed.Load(),
		policy3.Stats.RateLimited.Load(),
		policy3.Stats.CBOpen.Load(),
		policy3.Stats.Errors.Load())

	// ── FAILURE MODES SUMMARY ─────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Failure mode summary ---")
	fmt.Println(`  Rate limited  → 429 Too Many Requests; client should back off
  Circuit open  → 503 Service Unavailable; stop sending; wait for recovery
  Retry exhausted → 503 or upstream error; log + alert
  Timeout       → 504 Gateway Timeout; surface to user with retry suggestion`)
}
