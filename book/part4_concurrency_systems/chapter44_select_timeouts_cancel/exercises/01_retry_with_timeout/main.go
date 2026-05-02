// FILE: book/part4_concurrency_systems/chapter44_select_timeouts_cancel/exercises/01_retry_with_timeout/main.go
// CHAPTER: 44 — select / Timeouts / Cancel
// EXERCISE: Retry loop with per-attempt timeout, overall deadline, and
//           exponential back-off. Uses select for all timing.
//
// Run (from the chapter folder):
//   go run ./exercises/01_retry_with_timeout

package main

import (
	"errors"
	"fmt"
	"math/rand"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// SIMULATED UNRELIABLE OPERATION
// ─────────────────────────────────────────────────────────────────────────────

var ErrTransient = errors.New("transient error")
var ErrPermanent = errors.New("permanent error")

// call simulates an operation that:
//   - takes latency time to complete
//   - fails with ErrTransient with probability failRate
func call(latency time.Duration, failRate float64) <-chan error {
	ch := make(chan error, 1)
	go func() {
		time.Sleep(latency)
		if rand.Float64() < failRate {
			ch <- ErrTransient
		} else {
			ch <- nil
		}
	}()
	return ch
}

// ─────────────────────────────────────────────────────────────────────────────
// RETRY WITH TIMEOUT AND EXPONENTIAL BACK-OFF
// ─────────────────────────────────────────────────────────────────────────────

type RetryConfig struct {
	MaxAttempts    int
	PerAttemptTimeout time.Duration
	OverallTimeout time.Duration
	BaseBackoff    time.Duration
	MaxBackoff     time.Duration
}

type AttemptResult struct {
	Attempt  int
	Outcome  string // "ok", "timeout", "transient", "permanent", "deadline"
	Duration time.Duration
}

func retryWithTimeout(
	op func() <-chan error,
	cfg RetryConfig,
) ([]AttemptResult, error) {
	overallDeadline := time.After(cfg.OverallTimeout)
	timer := time.NewTimer(0)
	defer timer.Stop()

	var results []AttemptResult
	backoff := cfg.BaseBackoff

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		t0 := time.Now()
		ch := op()

		// Per-attempt timeout.
		timer.Stop()
		select { case <-timer.C: default: }
		timer.Reset(cfg.PerAttemptTimeout)

		var outcome string
		var err error

		select {
		case opErr := <-ch:
			elapsed := time.Since(t0)
			if opErr == nil {
				outcome = "ok"
			} else if errors.Is(opErr, ErrPermanent) {
				outcome = "permanent"
				err = opErr
			} else {
				outcome = "transient"
				err = opErr
			}
			results = append(results, AttemptResult{attempt, outcome, elapsed.Round(time.Millisecond)})

		case <-timer.C:
			results = append(results, AttemptResult{attempt, "timeout", cfg.PerAttemptTimeout})
			err = fmt.Errorf("attempt %d timed out", attempt)

		case <-overallDeadline:
			results = append(results, AttemptResult{attempt, "deadline", time.Since(t0).Round(time.Millisecond)})
			return results, fmt.Errorf("overall deadline exceeded after %d attempts", attempt)
		}

		if outcome == "ok" {
			return results, nil
		}
		if outcome == "permanent" {
			return results, err
		}

		// Back-off before next attempt (but respect overall deadline).
		if attempt < cfg.MaxAttempts {
			select {
			case <-time.After(backoff):
			case <-overallDeadline:
				return results, fmt.Errorf("overall deadline exceeded during backoff")
			}
			backoff *= 2
			if backoff > cfg.MaxBackoff {
				backoff = cfg.MaxBackoff
			}
		}
	}

	return results, fmt.Errorf("all %d attempts failed", cfg.MaxAttempts)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func printResults(results []AttemptResult, err error) {
	for _, r := range results {
		fmt.Printf("    attempt %d: %-10s (%s)\n", r.Attempt, r.Outcome, r.Duration)
	}
	if err != nil {
		fmt.Printf("    final error: %v\n", err)
	} else {
		fmt.Println("    success!")
	}
}

func main() {
	rand.New(rand.NewSource(42))

	cfg := RetryConfig{
		MaxAttempts:       5,
		PerAttemptTimeout: 80 * time.Millisecond,
		OverallTimeout:    500 * time.Millisecond,
		BaseBackoff:       10 * time.Millisecond,
		MaxBackoff:        50 * time.Millisecond,
	}

	// Scenario 1: fast operation, low failure rate → likely succeeds.
	fmt.Println("=== Scenario 1: fast op, 30% fail rate ===")
	results, err := retryWithTimeout(
		func() <-chan error { return call(20*time.Millisecond, 0.3) },
		cfg,
	)
	printResults(results, err)

	// Scenario 2: slow operation — per-attempt timeouts fire.
	fmt.Println()
	fmt.Println("=== Scenario 2: slow op (120ms), 50ms per-attempt timeout ===")
	cfg2 := cfg
	cfg2.PerAttemptTimeout = 50 * time.Millisecond
	results, err = retryWithTimeout(
		func() <-chan error { return call(120*time.Millisecond, 0.0) },
		cfg2,
	)
	printResults(results, err)

	// Scenario 3: overall deadline exceeded.
	fmt.Println()
	fmt.Println("=== Scenario 3: always fails, short overall deadline ===")
	cfg3 := cfg
	cfg3.OverallTimeout = 80 * time.Millisecond
	results, err = retryWithTimeout(
		func() <-chan error { return call(30*time.Millisecond, 1.0) },
		cfg3,
	)
	printResults(results, err)

	// Scenario 4: eventually succeeds after retries.
	fmt.Println()
	fmt.Println("=== Scenario 4: fails twice, succeeds on 3rd ===")
	attempt := 0
	results, err = retryWithTimeout(
		func() <-chan error {
			attempt++
			ch := make(chan error, 1)
			go func() {
				time.Sleep(10 * time.Millisecond)
				if attempt < 3 {
					ch <- ErrTransient
				} else {
					ch <- nil
				}
			}()
			return ch
		},
		cfg,
	)
	printResults(results, err)
}
