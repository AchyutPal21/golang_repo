// FILE: book/part4_concurrency_systems/chapter47_context/examples/01_context_basics/main.go
// CHAPTER: 47 — context Package
// TOPIC: context.Background, context.TODO, WithCancel, WithTimeout,
//        WithDeadline, Done channel, Err(), and cancellation propagation.
//
// Run (from the chapter folder):
//   go run ./examples/01_context_basics

package main

import (
	"context"
	"fmt"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// ROOT CONTEXTS
//
// context.Background() — the root of all context trees; never cancelled.
// context.TODO()        — placeholder when the right context isn't clear yet.
// ─────────────────────────────────────────────────────────────────────────────

// ─────────────────────────────────────────────────────────────────────────────
// WithCancel — manual cancellation
// ─────────────────────────────────────────────────────────────────────────────

func demoWithCancel() {
	fmt.Println("=== context.WithCancel ===")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // always defer cancel to avoid goroutine leak

	// Worker that respects cancellation.
	done := make(chan string, 1)
	go func() {
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				done <- fmt.Sprintf("cancelled after %d iterations: %v", i, ctx.Err())
				return
			default:
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	time.Sleep(22 * time.Millisecond)
	cancel() // signal cancellation
	fmt.Println(" ", <-done)
}

// ─────────────────────────────────────────────────────────────────────────────
// WithTimeout — cancel after duration
// ─────────────────────────────────────────────────────────────────────────────

func slowOperation(ctx context.Context, delay time.Duration) error {
	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func demoWithTimeout() {
	fmt.Println()
	fmt.Println("=== context.WithTimeout ===")

	// Succeeds within timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := slowOperation(ctx, 20*time.Millisecond)
	fmt.Printf("  fast op (20ms, 100ms timeout): err=%v\n", err)

	// Exceeds timeout.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel2()
	err = slowOperation(ctx2, 200*time.Millisecond)
	fmt.Printf("  slow op (200ms, 30ms timeout): err=%v\n", err)
}

// ─────────────────────────────────────────────────────────────────────────────
// WithDeadline — cancel at absolute time
// ─────────────────────────────────────────────────────────────────────────────

func demoWithDeadline() {
	fmt.Println()
	fmt.Println("=== context.WithDeadline ===")

	deadline := time.Now().Add(50 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	fmt.Printf("  deadline in: %s\n", time.Until(deadline).Round(time.Millisecond))

	err := slowOperation(ctx, 200*time.Millisecond)
	fmt.Printf("  result: err=%v\n", err)

	dl, ok := ctx.Deadline()
	fmt.Printf("  ctx.Deadline(): %v ok=%v\n", dl.Format("15:04:05.000"), ok)
}

// ─────────────────────────────────────────────────────────────────────────────
// CANCELLATION PROPAGATION
//
// When a parent context is cancelled, all child contexts are cancelled too.
// ─────────────────────────────────────────────────────────────────────────────

func demoPropagation() {
	fmt.Println()
	fmt.Println("=== Cancellation propagation ===")

	parent, cancelParent := context.WithCancel(context.Background())
	child1, cancelChild1 := context.WithCancel(parent)
	child2, cancelChild2 := context.WithTimeout(parent, time.Minute)
	defer cancelChild2()

	grandchild, cancelGrandchild := context.WithCancel(child1)
	defer cancelGrandchild()

	// Cancel the parent — all descendants are cancelled.
	cancelParent()
	cancelChild1() // still safe to call; idempotent

	wait := func(name string, ctx context.Context) {
		<-ctx.Done()
		fmt.Printf("  %s cancelled: %v\n", name, ctx.Err())
	}

	wait("child1", child1)
	wait("child2", child2)
	wait("grandchild", grandchild)
}

// ─────────────────────────────────────────────────────────────────────────────
// ctx.Err() — distinguish cancel from timeout
// ─────────────────────────────────────────────────────────────────────────────

func demoErr() {
	fmt.Println()
	fmt.Println("=== ctx.Err() values ===")

	// Cancelled manually.
	ctx1, cancel1 := context.WithCancel(context.Background())
	cancel1()
	<-ctx1.Done()
	fmt.Printf("  after manual cancel: %v (is context.Canceled: %v)\n",
		ctx1.Err(), ctx1.Err() == context.Canceled)

	// Timed out.
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel2()
	time.Sleep(time.Millisecond)
	<-ctx2.Done()
	fmt.Printf("  after timeout:       %v (is context.DeadlineExceeded: %v)\n",
		ctx2.Err(), ctx2.Err() == context.DeadlineExceeded)
}

func main() {
	demoWithCancel()
	demoWithTimeout()
	demoWithDeadline()
	demoPropagation()
	demoErr()
}
