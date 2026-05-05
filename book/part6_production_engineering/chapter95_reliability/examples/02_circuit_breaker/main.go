// FILE: book/part6_production_engineering/chapter95_reliability/examples/02_circuit_breaker/main.go
// CHAPTER: 95 — Reliability Engineering
// TOPIC: Circuit breaker state machine — Closed/Open/HalfOpen transitions,
//        error threshold, recovery probe, and cascading failure prevention.
//
// Run:
//   go run ./book/part6_production_engineering/chapter95_reliability/examples/02_circuit_breaker

package main

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// CIRCUIT BREAKER
// ─────────────────────────────────────────────────────────────────────────────

type CBState int

const (
	CBClosed   CBState = iota // Normal operation — requests pass through
	CBOpen                    // Failing — requests rejected immediately
	CBHalfOpen                // Probe mode — limited requests allowed
)

func (s CBState) String() string {
	switch s {
	case CBClosed:
		return "CLOSED"
	case CBOpen:
		return "OPEN"
	case CBHalfOpen:
		return "HALF-OPEN"
	default:
		return "UNKNOWN"
	}
}

var ErrCircuitOpen = errors.New("circuit breaker: open")

type CBConfig struct {
	ErrorThreshold    int           // consecutive failures to open
	OpenDuration      time.Duration // time in open state before half-open
	HalfOpenSuccesses int           // successes needed to close from half-open
}

type CircuitBreaker struct {
	config       CBConfig
	mu           sync.Mutex
	state        CBState
	failures     int
	successes    int
	lastOpenTime time.Time

	// stats
	totalRequests atomic.Int64
	totalRejected atomic.Int64
	totalSuccess  atomic.Int64
	totalFailure  atomic.Int64
}

func NewCircuitBreaker(cfg CBConfig) *CircuitBreaker {
	return &CircuitBreaker{config: cfg, state: CBClosed}
}

func (cb *CircuitBreaker) State() CBState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.maybeTransitionToHalfOpen()
	return cb.state
}

func (cb *CircuitBreaker) maybeTransitionToHalfOpen() {
	if cb.state == CBOpen && time.Since(cb.lastOpenTime) >= cb.config.OpenDuration {
		cb.state = CBHalfOpen
		cb.successes = 0
		fmt.Printf("    [CB] state: OPEN → HALF-OPEN (timeout expired)\n")
	}
}

func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.totalRequests.Add(1)
	cb.mu.Lock()
	cb.maybeTransitionToHalfOpen()
	state := cb.state

	if state == CBOpen {
		cb.mu.Unlock()
		cb.totalRejected.Add(1)
		return ErrCircuitOpen
	}
	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.totalFailure.Add(1)
		cb.onFailure()
	} else {
		cb.totalSuccess.Add(1)
		cb.onSuccess()
	}
	return err
}

func (cb *CircuitBreaker) onFailure() {
	cb.successes = 0
	switch cb.state {
	case CBClosed:
		cb.failures++
		if cb.failures >= cb.config.ErrorThreshold {
			cb.state = CBOpen
			cb.lastOpenTime = time.Now()
			fmt.Printf("    [CB] state: CLOSED → OPEN (failures=%d)\n", cb.failures)
		}
	case CBHalfOpen:
		cb.state = CBOpen
		cb.lastOpenTime = time.Now()
		fmt.Printf("    [CB] state: HALF-OPEN → OPEN (probe failed)\n")
	}
}

func (cb *CircuitBreaker) onSuccess() {
	cb.failures = 0
	if cb.state == CBHalfOpen {
		cb.successes++
		if cb.successes >= cb.config.HalfOpenSuccesses {
			cb.state = CBClosed
			fmt.Printf("    [CB] state: HALF-OPEN → CLOSED (probe succeeded)\n")
		}
	}
}

func (cb *CircuitBreaker) Stats() string {
	return fmt.Sprintf("total=%d  success=%d  failure=%d  rejected=%d",
		cb.totalRequests.Load(), cb.totalSuccess.Load(),
		cb.totalFailure.Load(), cb.totalRejected.Load())
}

// ─────────────────────────────────────────────────────────────────────────────
// SCENARIO SIMULATION
// ─────────────────────────────────────────────────────────────────────────────

type callResult struct {
	label string
	err   error
}

func simulateCall(cb *CircuitBreaker, label string, fn func() error) callResult {
	err := cb.Call(fn)
	return callResult{label, err}
}

func printResult(r callResult, state CBState) {
	status := "OK"
	if r.err != nil {
		if errors.Is(r.err, ErrCircuitOpen) {
			status = "REJECTED (circuit open)"
		} else {
			status = "ERROR: " + r.err.Error()
		}
	}
	fmt.Printf("    %-20s  state=%-10s  %s\n", r.label, state, status)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 95: Circuit Breaker ===")
	fmt.Println()

	cfg := CBConfig{
		ErrorThreshold:    3,
		OpenDuration:      200 * time.Millisecond, // short for demo
		HalfOpenSuccesses: 2,
	}
	cb := NewCircuitBreaker(cfg)

	always := func() error { return nil }
	fail := func() error { return errors.New("downstream timeout") }

	fmt.Println("--- Phase 1: Normal operation (all succeed) ---")
	for i := 0; i < 3; i++ {
		r := simulateCall(cb, fmt.Sprintf("request %d", i+1), always)
		printResult(r, cb.State())
	}
	fmt.Println()

	fmt.Println("--- Phase 2: Failures — circuit opens ---")
	for i := 0; i < 5; i++ {
		r := simulateCall(cb, fmt.Sprintf("request %d", i+4), fail)
		printResult(r, cb.State())
	}
	fmt.Println()

	fmt.Println("--- Phase 3: Open circuit rejects immediately ---")
	for i := 0; i < 3; i++ {
		r := simulateCall(cb, fmt.Sprintf("request %d", i+9), always)
		printResult(r, cb.State())
	}
	fmt.Println()

	fmt.Println("--- Phase 4: Wait for half-open transition ---")
	fmt.Printf("    Sleeping %v for open duration...\n", cfg.OpenDuration)
	time.Sleep(cfg.OpenDuration + 10*time.Millisecond)
	fmt.Printf("    State after sleep: %s\n", cb.State())
	fmt.Println()

	fmt.Println("--- Phase 5: Probe succeeds — circuit closes ---")
	for i := 0; i < 3; i++ {
		r := simulateCall(cb, fmt.Sprintf("probe %d", i+1), always)
		printResult(r, cb.State())
	}
	fmt.Println()

	fmt.Printf("Final stats: %s\n\n", cb.Stats())

	// ── DESIGN NOTES ─────────────────────────────────────────────────────────
	fmt.Println("--- Circuit breaker design notes ---")
	fmt.Println(`  When to use:
    - Calling a synchronous downstream (HTTP, gRPC, DB, cache)
    - Downstream has a history of becoming slow or returning errors under load
    - You want fail-fast instead of goroutine accumulation

  When NOT to use:
    - Async message consumers (Kafka, SQS) — use dead letter queues instead
    - Database migrations — use health checks + draining instead
    - Internal in-process calls — overhead is not justified

  Combine with:
    - Timeout: every outgoing call needs a deadline
    - Retry with jitter: for transient errors (before circuit opens)
    - Bulkhead: limit concurrent calls per downstream
    - Fallback: return cached or degraded response when circuit is open`)
}
