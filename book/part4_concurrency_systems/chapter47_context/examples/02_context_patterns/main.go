// FILE: book/part4_concurrency_systems/chapter47_context/examples/02_context_patterns/main.go
// CHAPTER: 47 — context Package
// TOPIC: context.WithValue, request-scoped values, context keys,
//        passing context through call chains, and anti-patterns.
//
// Run (from the chapter folder):
//   go run ./examples/02_context_patterns

package main

import (
	"context"
	"fmt"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// context.WithValue — attach request-scoped values
//
// Use unexported key types to prevent collisions across packages.
// ─────────────────────────────────────────────────────────────────────────────

// Unexported key types prevent collision with keys from other packages.
type contextKey string

const (
	keyRequestID contextKey = "request_id"
	keyUserID    contextKey = "user_id"
	keyTraceID   contextKey = "trace_id"
)

func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyRequestID, id)
}

func requestIDFrom(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(keyRequestID).(string)
	return id, ok
}

func withUserID(ctx context.Context, uid int64) context.Context {
	return context.WithValue(ctx, keyUserID, uid)
}

func userIDFrom(ctx context.Context) (int64, bool) {
	uid, ok := ctx.Value(keyUserID).(int64)
	return uid, ok
}

func demoWithValue() {
	fmt.Println("=== context.WithValue ===")

	ctx := context.Background()
	ctx = withRequestID(ctx, "req-abc-123")
	ctx = withUserID(ctx, 42)
	ctx = context.WithValue(ctx, keyTraceID, "trace-xyz")

	if rid, ok := requestIDFrom(ctx); ok {
		fmt.Printf("  request_id: %s\n", rid)
	}
	if uid, ok := userIDFrom(ctx); ok {
		fmt.Printf("  user_id: %d\n", uid)
	}

	// Missing key returns zero value + false.
	_, ok := ctx.Value(contextKey("missing")).(string)
	fmt.Printf("  missing key: ok=%v\n", ok)
}

// ─────────────────────────────────────────────────────────────────────────────
// CONTEXT THROUGH A CALL CHAIN
//
// Every function that can be cancelled or timed out must accept ctx as its
// first parameter and pass it to all sub-calls.
// ─────────────────────────────────────────────────────────────────────────────

func fetchUser(ctx context.Context, id int64) (string, error) {
	// Simulate DB call that respects cancellation.
	select {
	case <-time.After(10 * time.Millisecond):
		if id, ok := userIDFrom(ctx); ok {
			return fmt.Sprintf("User-%d", id), nil
		}
		return fmt.Sprintf("User-%d", id), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func fetchPermissions(ctx context.Context, userID int64) ([]string, error) {
	select {
	case <-time.After(10 * time.Millisecond):
		return []string{"read", "write"}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func handleRequest(ctx context.Context, userID int64) error {
	rid, _ := requestIDFrom(ctx)
	fmt.Printf("  [%s] handling request for user %d\n", rid, userID)

	user, err := fetchUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("fetchUser: %w", err)
	}

	perms, err := fetchPermissions(ctx, userID)
	if err != nil {
		return fmt.Errorf("fetchPermissions: %w", err)
	}

	fmt.Printf("  [%s] user=%s perms=%v\n", rid, user, perms)
	return nil
}

func demoCallChain() {
	fmt.Println()
	fmt.Println("=== Context through call chain ===")

	ctx := context.Background()
	ctx = withRequestID(ctx, "req-001")
	ctx = withUserID(ctx, 7)
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	if err := handleRequest(ctx, 7); err != nil {
		fmt.Printf("  error: %v\n", err)
	}

	// Now with a very short timeout — cancellation propagates through chain.
	ctx2 := context.Background()
	ctx2 = withRequestID(ctx2, "req-002")
	ctx2, cancel2 := context.WithTimeout(ctx2, 5*time.Millisecond)
	defer cancel2()

	if err := handleRequest(ctx2, 99); err != nil {
		fmt.Printf("  req-002 error (expected): %v\n", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CONTEXT ANTI-PATTERNS
// ─────────────────────────────────────────────────────────────────────────────

func demoAntiPatterns() {
	fmt.Println()
	fmt.Println("=== Anti-patterns (what NOT to do) ===")

	// ANTI-PATTERN 1: storing context in a struct field.
	// Don't do this — context should be per-call, not per-object.
	fmt.Println("  BAD:  type Server struct { ctx context.Context }")
	fmt.Println("  GOOD: func (s *Server) Handle(ctx context.Context, req Request)")

	// ANTI-PATTERN 2: passing nil context.
	// Always use context.Background() or context.TODO() as roots.
	fmt.Println("  BAD:  doSomething(nil)")
	fmt.Println("  GOOD: doSomething(context.Background())")

	// ANTI-PATTERN 3: using context.WithValue for optional parameters.
	// context.Value is for request-scoped metadata (trace IDs, auth tokens),
	// not for passing optional function arguments.
	fmt.Println("  BAD:  ctx = context.WithValue(ctx, \"maxRetries\", 5)")
	fmt.Println("  GOOD: func Fetch(ctx context.Context, maxRetries int)")

	// ANTI-PATTERN 4: ignoring cancellation in a loop.
	fmt.Println("  BAD:  for { process(item) }  // no ctx.Done() check")
	fmt.Println("  GOOD: for { select { case <-ctx.Done(): return; default: process(item) } }")
}

// ─────────────────────────────────────────────────────────────────────────────
// CONTEXT IN HTTP MIDDLEWARE (simulation)
// ─────────────────────────────────────────────────────────────────────────────

type Request struct {
	Path   string
	UserID int64
}

type Handler func(ctx context.Context, req Request) error

func withRequestIDMiddleware(next Handler) Handler {
	return func(ctx context.Context, req Request) error {
		reqID := fmt.Sprintf("req-%d", time.Now().UnixNano()%10000)
		ctx = withRequestID(ctx, reqID)
		return next(ctx, req)
	}
}

func withTimeoutMiddleware(d time.Duration, next Handler) Handler {
	return func(ctx context.Context, req Request) error {
		ctx, cancel := context.WithTimeout(ctx, d)
		defer cancel()
		return next(ctx, req)
	}
}

func demoMiddleware() {
	fmt.Println()
	fmt.Println("=== Context in middleware chain ===")

	handler := Handler(func(ctx context.Context, req Request) error {
		rid, _ := requestIDFrom(ctx)
		fmt.Printf("  [%s] %s for user %d\n", rid, req.Path, req.UserID)
		return nil
	})

	chain := withTimeoutMiddleware(
		100*time.Millisecond,
		withRequestIDMiddleware(handler),
	)

	reqs := []Request{
		{Path: "/api/users", UserID: 1},
		{Path: "/api/orders", UserID: 2},
	}
	for _, r := range reqs {
		if err := chain(context.Background(), r); err != nil {
			fmt.Printf("  error: %v\n", err)
		}
	}
}

func main() {
	demoWithValue()
	demoCallChain()
	demoAntiPatterns()
	demoMiddleware()
}
