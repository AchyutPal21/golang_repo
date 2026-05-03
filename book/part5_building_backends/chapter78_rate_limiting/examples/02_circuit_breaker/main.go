// FILE: book/part5_building_backends/chapter78_rate_limiting/examples/02_circuit_breaker/main.go
// CHAPTER: 78 — Rate Limiting, Circuit Breakers, Retries
// TOPIC: Circuit breaker state machine, exponential backoff with jitter,
//        hedged requests, and retry budgets.
//
// Run (from the chapter folder):
//   go run ./examples/02_circuit_breaker

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
// CIRCUIT BREAKER
// States: Closed → Open (on failure threshold) → Half-Open → Closed/Open
// ─────────────────────────────────────────────────────────────────────────────

type CBState int

const (
	StateClosed   CBState = 0 // normal operation
	StateOpen     CBState = 1 // failing — reject all requests
	StateHalfOpen CBState = 2 // testing recovery — allow one probe
)

func (s CBState) String() string {
	switch s {
	case StateClosed:
		return "Closed"
	case StateOpen:
		return "Open"
	case StateHalfOpen:
		return "HalfOpen"
	default:
		return "Unknown"
	}
}

type CBConfig struct {
	FailureThreshold int           // consecutive failures before opening
	SuccessThreshold int           // consecutive successes in HalfOpen to close
	OpenTimeout      time.Duration // how long to stay Open before probing
}

type CircuitBreaker struct {
	mu               sync.Mutex
	state            CBState
	cfg              CBConfig
	consecutiveFails int
	consecutiveOK    int
	openedAt         time.Time
	Transitions      []string
}

var ErrCircuitOpen = errors.New("circuit breaker: open")

func NewCircuitBreaker(cfg CBConfig) *CircuitBreaker {
	return &CircuitBreaker{cfg: cfg}
}

func (cb *CircuitBreaker) State() CBState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	switch cb.state {
	case StateOpen:
		if time.Since(cb.openedAt) < cb.cfg.OpenTimeout {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
		// Transition to half-open to probe.
		cb.transition(StateHalfOpen)
	case StateHalfOpen:
		// Only one probe at a time — reject others.
		cb.mu.Unlock()
		return ErrCircuitOpen
	}
	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()
	if err != nil {
		cb.consecutiveFails++
		cb.consecutiveOK = 0
		if cb.state == StateHalfOpen || cb.consecutiveFails >= cb.cfg.FailureThreshold {
			cb.transition(StateOpen)
		}
		return err
	}
	cb.consecutiveFails = 0
	cb.consecutiveOK++
	if cb.state == StateHalfOpen && cb.consecutiveOK >= cb.cfg.SuccessThreshold {
		cb.transition(StateClosed)
	}
	return nil
}

func (cb *CircuitBreaker) transition(next CBState) {
	prev := cb.state
	cb.state = next
	if next == StateOpen {
		cb.openedAt = time.Now()
		cb.consecutiveFails = 0
	}
	if next == StateClosed {
		cb.consecutiveOK = 0
	}
	cb.Transitions = append(cb.Transitions, fmt.Sprintf("%s→%s", prev, next))
}

// ─────────────────────────────────────────────────────────────────────────────
// EXPONENTIAL BACKOFF WITH JITTER
// ─────────────────────────────────────────────────────────────────────────────

type BackoffConfig struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       float64 // fraction of delay to randomize (0–1)
	MaxAttempts  int
}

func DefaultBackoff() BackoffConfig {
	return BackoffConfig{
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.2,
		MaxAttempts:  5,
	}
}

func (b BackoffConfig) Delay(attempt int) time.Duration {
	delay := float64(b.InitialDelay)
	for i := 0; i < attempt; i++ {
		delay *= b.Multiplier
	}
	if delay > float64(b.MaxDelay) {
		delay = float64(b.MaxDelay)
	}
	// Add ±jitter*delay.
	jitter := delay * b.Jitter * (rand.Float64()*2 - 1)
	d := time.Duration(delay + jitter)
	if d < 0 {
		d = 0
	}
	return d
}

func WithRetry(ctx context.Context, cfg BackoffConfig, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if attempt == cfg.MaxAttempts-1 {
			break
		}
		delay := cfg.Delay(attempt)
		fmt.Printf("  [retry] attempt %d failed (%v), backoff %s\n", attempt+1, lastErr, delay.Round(time.Millisecond))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return fmt.Errorf("all %d attempts failed: %w", cfg.MaxAttempts, lastErr)
}

// ─────────────────────────────────────────────────────────────────────────────
// HEDGED REQUESTS (send duplicate request after a delay; use first response)
// ─────────────────────────────────────────────────────────────────────────────

func HedgedRequest(ctx context.Context, hedgeDelay time.Duration, fn func(ctx context.Context, attempt int) (string, error)) (string, error) {
	type result struct {
		val string
		err error
	}
	results := make(chan result, 2)
	hCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for attempt := 0; attempt < 2; attempt++ {
		attempt := attempt
		go func() {
			if attempt == 1 {
				select {
				case <-hCtx.Done():
					return
				case <-time.After(hedgeDelay):
				}
			}
			val, err := fn(hCtx, attempt)
			select {
			case results <- result{val, err}:
			default:
			}
		}()
	}

	// Return first successful response.
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err == nil {
			cancel() // cancel the other request
			return r.val, nil
		}
	}
	return "", errors.New("all hedged attempts failed")
}

// ─────────────────────────────────────────────────────────────────────────────
// RETRY BUDGET
// A global token bucket that limits how many retries the system issues in total.
// Prevents retry storms.
// ─────────────────────────────────────────────────────────────────────────────

type RetryBudget struct {
	mu        sync.Mutex
	tokens    float64
	capacity  float64
	rate      float64
	lastFill  time.Time
	Consumed  atomic.Int64
	Rejected  atomic.Int64
}

func NewRetryBudget(capacity, ratePerSec float64) *RetryBudget {
	return &RetryBudget{tokens: capacity, capacity: capacity, rate: ratePerSec, lastFill: time.Now()}
}

func (rb *RetryBudget) TryRetry() bool {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(rb.lastFill).Seconds()
	rb.tokens += elapsed * rb.rate
	if rb.tokens > rb.capacity {
		rb.tokens = rb.capacity
	}
	rb.lastFill = now
	if rb.tokens < 1 {
		rb.Rejected.Add(1)
		return false
	}
	rb.tokens--
	rb.Consumed.Add(1)
	return true
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Circuit Breaker and Retry Patterns ===")
	fmt.Println()

	// ── CIRCUIT BREAKER ───────────────────────────────────────────────────────
	fmt.Println("--- Circuit breaker (threshold=3, openTimeout=50ms) ---")
	cb := NewCircuitBreaker(CBConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		OpenTimeout:      50 * time.Millisecond,
	})

	failingService := func() error {
		return fmt.Errorf("connection refused")
	}

	// Trip the breaker.
	for i := 1; i <= 5; i++ {
		err := cb.Call(failingService)
		fmt.Printf("  call %d: state=%s err=%v\n", i, cb.State(), err)
	}

	// Wait for open timeout, then probe.
	fmt.Println("  waiting 55ms for open timeout...")
	time.Sleep(55 * time.Millisecond)

	// Probe succeeds → half-open; success threshold needs 2 successes.
	successService := func() error { return nil }
	for i := 6; i <= 9; i++ {
		err := cb.Call(successService)
		fmt.Printf("  call %d: state=%s err=%v\n", i, cb.State(), err)
	}

	fmt.Printf("  transitions: %v\n", cb.Transitions)

	// ── EXPONENTIAL BACKOFF WITH JITTER ───────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Exponential backoff with jitter ---")

	cfg := BackoffConfig{
		InitialDelay: 5 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0.3,
		MaxAttempts:  4,
	}
	for i := 0; i < 4; i++ {
		d := cfg.Delay(i)
		fmt.Printf("  attempt %d delay: %s\n", i+1, d.Round(time.Millisecond))
	}

	var callCount atomic.Int32
	ctx := context.Background()
	err := WithRetry(ctx, cfg, func() error {
		n := callCount.Add(1)
		if n < 3 {
			return fmt.Errorf("transient error (call %d)", n)
		}
		fmt.Printf("  succeeded on call %d\n", n)
		return nil
	})
	if err != nil {
		fmt.Printf("  error: %v\n", err)
	}

	// ── HEDGED REQUESTS ───────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Hedged requests (hedge after 10ms) ---")

	var hedgeAttempts atomic.Int32
	val, err := HedgedRequest(ctx, 10*time.Millisecond,
		func(hCtx context.Context, attempt int) (string, error) {
			hedgeAttempts.Add(1)
			if attempt == 0 {
				// Slow primary — takes 30ms.
				select {
				case <-hCtx.Done():
					return "", hCtx.Err()
				case <-time.After(30 * time.Millisecond):
					return "primary response", nil
				}
			}
			// Fast hedge — takes 5ms.
			select {
			case <-hCtx.Done():
				return "", hCtx.Err()
			case <-time.After(5 * time.Millisecond):
				return "hedge response", nil
			}
		})
	fmt.Printf("  result: %q (from %d attempts)\n", val, hedgeAttempts.Load())
	if err != nil {
		fmt.Printf("  error: %v\n", err)
	}

	// ── RETRY BUDGET ──────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Retry budget (cap=3 retries globally) ---")

	budget := NewRetryBudget(3, 10) // 3 burst, 10 refill/s
	consumed, rejected := 0, 0
	for i := 0; i < 8; i++ {
		if budget.TryRetry() {
			consumed++
			fmt.Printf("  retry %d: allowed\n", i+1)
		} else {
			rejected++
			fmt.Printf("  retry %d: budget exhausted — fail fast\n", i+1)
		}
	}
	fmt.Printf("  budget: consumed=%d rejected=%d\n", consumed, rejected)
}
