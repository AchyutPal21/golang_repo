// FILE: book/part4_concurrency_systems/chapter44_select_timeouts_cancel/examples/02_timeout_patterns/main.go
// CHAPTER: 44 — select / Timeouts / Cancel
// TOPIC: time.After, time.NewTimer (and Reset), time.NewTicker,
//        per-call timeout, per-loop timeout, and heartbeat pattern.
//
// Run (from the chapter folder):
//   go run ./examples/02_timeout_patterns

package main

import (
	"fmt"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// time.After — simplest per-call timeout
//
// WARNING: time.After leaks the underlying Timer until it fires.
// In tight loops, prefer time.NewTimer + Reset (see below).
// ─────────────────────────────────────────────────────────────────────────────

func fetchSlow(delay time.Duration) <-chan string {
	ch := make(chan string, 1)
	go func() {
		time.Sleep(delay)
		ch <- "result"
	}()
	return ch
}

func demoTimeAfter() {
	fmt.Println("=== time.After (per-call timeout) ===")

	// Fetch completes within timeout.
	select {
	case v := <-fetchSlow(20 * time.Millisecond):
		fmt.Printf("  fast fetch: %s\n", v)
	case <-time.After(100 * time.Millisecond):
		fmt.Println("  fast fetch: TIMEOUT")
	}

	// Fetch exceeds timeout.
	select {
	case v := <-fetchSlow(200 * time.Millisecond):
		fmt.Printf("  slow fetch: %s\n", v)
	case <-time.After(50 * time.Millisecond):
		fmt.Println("  slow fetch: TIMEOUT (expected)")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// time.NewTimer + Reset — reusable timer (no leak in loops)
// ─────────────────────────────────────────────────────────────────────────────

func demoTimerReset() {
	fmt.Println()
	fmt.Println("=== time.NewTimer + Reset ===")

	timer := time.NewTimer(0) // create once
	defer timer.Stop()

	delays := []time.Duration{10 * time.Millisecond, 200 * time.Millisecond, 10 * time.Millisecond, 200 * time.Millisecond}
	timeout := 50 * time.Millisecond

	for i, delay := range delays {
		work := fetchSlow(delay)

		// Safe Reset: stop first, drain channel, then reset.
		timer.Stop()
		select {
		case <-timer.C:
		default:
		}
		timer.Reset(timeout)

		select {
		case v := <-work:
			fmt.Printf("  call %d (%dms): ok — %s\n", i, delay/time.Millisecond, v)
		case <-timer.C:
			fmt.Printf("  call %d (%dms): TIMEOUT\n", i, delay/time.Millisecond)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// time.NewTicker — repeated work at fixed intervals
// ─────────────────────────────────────────────────────────────────────────────

func demoTicker() {
	fmt.Println()
	fmt.Println("=== time.NewTicker (heartbeat) ===")

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	done := time.After(90 * time.Millisecond)
	beats := 0

	for {
		select {
		case <-ticker.C:
			beats++
			fmt.Printf("  tick %d\n", beats)
		case <-done:
			fmt.Printf("  stopped after %d beats\n", beats)
			return
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// OVERALL DEADLINE — one timeout shared across multiple operations
// ─────────────────────────────────────────────────────────────────────────────

func demoOverallDeadline() {
	fmt.Println()
	fmt.Println("=== Overall deadline across multiple ops ===")

	deadline := time.After(120 * time.Millisecond)
	ops := []struct {
		name  string
		delay time.Duration
	}{
		{"op-A", 30 * time.Millisecond},
		{"op-B", 30 * time.Millisecond},
		{"op-C", 30 * time.Millisecond}, // this one will exceed the deadline
		{"op-D", 30 * time.Millisecond},
	}

	for _, op := range ops {
		ch := fetchSlow(op.delay)
		select {
		case <-ch:
			fmt.Printf("  %s: done\n", op.name)
		case <-deadline:
			fmt.Printf("  %s: overall deadline exceeded — aborting\n", op.name)
			return
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// HEARTBEAT — worker signals liveness on a regular interval
// ─────────────────────────────────────────────────────────────────────────────

func workerWithHeartbeat(done <-chan struct{}) (<-chan int, <-chan time.Time) {
	results := make(chan int, 5)
	heartbeat := make(chan time.Time, 1)

	go func() {
		defer close(results)
		ticker := time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				select {
				case heartbeat <- t: // non-blocking: drop if nobody reading
				default:
				}
				results <- i
				i++
			}
		}
	}()
	return results, heartbeat
}

func demoHeartbeat() {
	fmt.Println()
	fmt.Println("=== Heartbeat pattern ===")

	done := make(chan struct{})
	results, heartbeat := workerWithHeartbeat(done)

	timeout := time.After(120 * time.Millisecond)
	for {
		select {
		case v, ok := <-results:
			if !ok {
				return
			}
			fmt.Printf("  result: %d\n", v)
		case t := <-heartbeat:
			fmt.Printf("  heartbeat at %s\n", t.Format("15:04:05.000"))
		case <-timeout:
			fmt.Println("  timeout — closing worker")
			close(done)
			// drain remaining results
			for v := range results {
				fmt.Printf("  drained: %d\n", v)
			}
			return
		}
	}
}

func main() {
	demoTimeAfter()
	demoTimerReset()
	demoTicker()
	demoOverallDeadline()
	demoHeartbeat()
}
