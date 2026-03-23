// FILE: 06_concurrency/10_context_package.go
// TOPIC: context Package — cancellation, timeouts, deadlines, values
//
// Run: go run 06_concurrency/10_context_package.go

package main

import (
	"context"
	"fmt"
	"time"
)

// simulateWork does work but respects context cancellation
func simulateWork(ctx context.Context, name string, duration time.Duration) error {
	select {
	case <-time.After(duration):
		fmt.Printf("  [%s] completed successfully\n", name)
		return nil
	case <-ctx.Done():
		fmt.Printf("  [%s] cancelled: %v\n", name, ctx.Err())
		return ctx.Err()
	}
}

// fetchUser simulates a DB call that respects context
func fetchUser(ctx context.Context, id int) (string, error) {
	// Check context BEFORE doing expensive work:
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("fetchUser: context already done: %w", err)
	}
	// Simulate DB latency:
	select {
	case <-time.After(50 * time.Millisecond):
		return fmt.Sprintf("user-%d", id), nil
	case <-ctx.Done():
		return "", fmt.Errorf("fetchUser: %w", ctx.Err())
	}
}

// getUserPrefs calls fetchUser and adds its own step — context propagates through
func getUserPrefs(ctx context.Context, userID int) (string, error) {
	user, err := fetchUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("getUserPrefs: %w", err)
	}
	// Continue work with same context:
	select {
	case <-time.After(30 * time.Millisecond):
		return fmt.Sprintf("prefs for %s", user), nil
	case <-ctx.Done():
		return "", fmt.Errorf("getUserPrefs prefs fetch: %w", ctx.Err())
	}
}

// contextKeyType is unexported to prevent collisions in context values
type contextKeyType string
const requestIDKey contextKeyType = "requestID"

func middleware(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, requestIDKey, reqID)
}

func handler(ctx context.Context) {
	reqID := ctx.Value(requestIDKey)
	fmt.Printf("  Handler: request ID = %v\n", reqID)
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: context Package")
	fmt.Println("════════════════════════════════════════")

	// ── context.Background and context.TODO ───────────────────────────────
	// context.Background() — root context for the entire program / top-level main
	// context.TODO()       — placeholder when you haven't decided on context yet
	// Both are non-nil, non-cancellable, empty contexts.
	bg := context.Background()
	fmt.Printf("\n── Root contexts ──\n")
	fmt.Printf("  Background: %v\n", bg)

	// ── context.WithCancel ────────────────────────────────────────────────
	// Creates a child context + a cancel function.
	// Calling cancel() closes ctx.Done() channel — signals all watchers to stop.
	// ALWAYS defer cancel() to avoid resource leaks.
	fmt.Println("\n── WithCancel ──")
	ctx, cancel := context.WithCancel(bg)
	defer cancel()

	go func() {
		time.Sleep(30 * time.Millisecond)
		fmt.Println("  Cancelling context...")
		cancel()
	}()

	err := simulateWork(ctx, "job-A", 100*time.Millisecond)
	fmt.Printf("  job-A result: %v\n", err)

	// ── context.WithTimeout ───────────────────────────────────────────────
	// Creates a context that auto-cancels after a duration.
	// Returns a cancel function — still call it to free resources early.
	fmt.Println("\n── WithTimeout ──")
	ctx2, cancel2 := context.WithTimeout(bg, 80*time.Millisecond)
	defer cancel2()

	// This should succeed (50ms < 80ms timeout):
	user, err := fetchUser(ctx2, 1)
	fmt.Printf("  fetchUser: %q, err=%v\n", user, err)

	// Now try with very short timeout:
	ctx3, cancel3 := context.WithTimeout(bg, 10*time.Millisecond)
	defer cancel3()
	user2, err2 := fetchUser(ctx3, 2)
	fmt.Printf("  fetchUser(10ms timeout): %q, err=%v\n", user2, err2)

	// ── context.WithDeadline ──────────────────────────────────────────────
	// Like WithTimeout but takes an absolute time.Time instead of duration.
	fmt.Println("\n── WithDeadline ──")
	deadline := time.Now().Add(100 * time.Millisecond)
	ctx4, cancel4 := context.WithDeadline(bg, deadline)
	defer cancel4()
	fmt.Printf("  Deadline set to: %v\n", deadline.Format("15:04:05.000"))
	fmt.Printf("  ctx.Deadline(): %v\n", func() string {
		d, ok := ctx4.Deadline()
		if !ok { return "no deadline" }
		return d.Format("15:04:05.000") + fmt.Sprintf(" (ok=%v)", ok)
	}())

	// ── Context propagation through call chain ────────────────────────────
	fmt.Println("\n── Context propagation through call chain ──")
	ctx5, cancel5 := context.WithTimeout(bg, 200*time.Millisecond)
	defer cancel5()

	prefs, err := getUserPrefs(ctx5, 42)
	fmt.Printf("  getUserPrefs: %q, err=%v\n", prefs, err)

	// With timeout too short for the chain:
	ctx6, cancel6 := context.WithTimeout(bg, 20*time.Millisecond)
	defer cancel6()
	prefs2, err2 := getUserPrefs(ctx6, 99)
	fmt.Printf("  getUserPrefs(20ms): %q, err=%v\n", prefs2, err2)

	// ── context.WithValue ─────────────────────────────────────────────────
	// Pass request-scoped data (request ID, user auth) through the call chain.
	// RULES:
	//   - Use a private key type to avoid collisions
	//   - Only for request-scoped data (logging, tracing, auth) — NOT config
	//   - Never store mutable state in context values
	fmt.Println("\n── WithValue (request-scoped data) ──")
	ctx7 := middleware(bg, "req-abc-123")
	handler(ctx7)

	// ── ctx.Err() — why was it cancelled? ────────────────────────────────
	fmt.Println("\n── ctx.Err() ──")
	ctxCancelled, cancelFn := context.WithCancel(bg)
	cancelFn()
	fmt.Printf("  After cancel: ctx.Err() = %v\n", ctxCancelled.Err())

	ctxTimeout, cancelT := context.WithTimeout(bg, 1*time.Millisecond)
	defer cancelT()
	time.Sleep(5 * time.Millisecond)
	fmt.Printf("  After timeout: ctx.Err() = %v\n", ctxTimeout.Err())

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  context.Background() → root, use at top of call chain")
	fmt.Println("  WithCancel  → manual cancellation (defer cancel()!)")
	fmt.Println("  WithTimeout → auto-cancel after duration")
	fmt.Println("  WithDeadline → auto-cancel at absolute time")
	fmt.Println("  WithValue   → request-scoped data (not config!)")
	fmt.Println("  ALWAYS pass ctx as FIRST argument in every function")
	fmt.Println("  NEVER store ctx in a struct field")
	fmt.Println("  ctx.Err() → context.Canceled or context.DeadlineExceeded")
}
