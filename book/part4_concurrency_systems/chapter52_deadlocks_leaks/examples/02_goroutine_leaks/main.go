// FILE: book/part4_concurrency_systems/chapter52_deadlocks_leaks/examples/02_goroutine_leaks/main.go
// CHAPTER: 52 — Deadlocks, Leaks
// TOPIC: Goroutine leak patterns — blocked channel read, blocked channel write,
//        ticker without Stop, context not propagated — and how to detect leaks
//        using runtime.NumGoroutine before/after.
//
// Run (from the chapter folder):
//   go run ./examples/02_goroutine_leaks

package main

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

// snapshot prints the current goroutine count.
func snapshot(label string) int {
	n := runtime.NumGoroutine()
	fmt.Printf("  [%s] goroutines: %d\n", label, n)
	return n
}

// ─────────────────────────────────────────────────────────────────────────────
// LEAK 1: goroutine blocked reading from a channel nobody closes
// ─────────────────────────────────────────────────────────────────────────────

func leakyReceiver() {
	ch := make(chan int)
	go func() {
		v := <-ch // blocks forever — nobody sends or closes ch
		fmt.Println("received:", v)
	}()
	// ch goes out of scope here but the goroutine is still blocked
}

func fixedReceiver(ctx context.Context) {
	ch := make(chan int)
	go func() {
		select {
		case v := <-ch:
			fmt.Println("received:", v)
		case <-ctx.Done(): // escape hatch
			return
		}
	}()
}

func demoReceiverLeak() {
	fmt.Println("=== Leak 1: blocked receiver ===")

	before := snapshot("before leak")
	leakyReceiver()
	time.Sleep(10 * time.Millisecond)
	after := snapshot("after leak")
	fmt.Printf("  leaked %d goroutine(s)\n", after-before)

	// Fixed version.
	ctx, cancel := context.WithCancel(context.Background())
	fixedReceiver(ctx)
	time.Sleep(10 * time.Millisecond)
	mid := snapshot("after fix (ctx running)")
	cancel()
	time.Sleep(10 * time.Millisecond)
	end := snapshot("after cancel")
	fmt.Printf("  goroutines cleaned up: %d (was %d)\n", mid-end, mid-before)
}

// ─────────────────────────────────────────────────────────────────────────────
// LEAK 2: goroutine blocked sending to a full channel nobody drains
// ─────────────────────────────────────────────────────────────────────────────

func leakySender() chan int {
	ch := make(chan int, 1) // buffer 1
	go func() {
		ch <- 1 // OK — buffer has space
		ch <- 2 // BLOCKS — buffer full, nobody reads
	}()
	return ch
}

func fixedSender(ctx context.Context) <-chan int {
	ch := make(chan int, 1)
	go func() {
		for i := 1; i <= 2; i++ {
			select {
			case ch <- i:
			case <-ctx.Done():
				return
			}
		}
		close(ch)
	}()
	return ch
}

func demoSenderLeak() {
	fmt.Println()
	fmt.Println("=== Leak 2: blocked sender ===")

	before := snapshot("before leak")
	_ = leakySender()
	time.Sleep(10 * time.Millisecond)
	after := snapshot("after leak")
	fmt.Printf("  leaked %d goroutine(s)\n", after-before)

	// Fixed.
	ctx, cancel := context.WithCancel(context.Background())
	ch := fixedSender(ctx)
	<-ch // drain one item
	cancel()
	time.Sleep(10 * time.Millisecond)
	end := snapshot("after fix+cancel")
	fmt.Printf("  goroutines after fix: %d (was %d)\n", end, after)
}

// ─────────────────────────────────────────────────────────────────────────────
// LEAK 3: time.Ticker not stopped
// ─────────────────────────────────────────────────────────────────────────────

func leakyTicker() {
	ticker := time.NewTicker(10 * time.Millisecond)
	// Forget to call ticker.Stop() — ticker goroutine leaks
	go func() {
		for range 3 {
			<-ticker.C
		}
		// goroutine exits but ticker is never stopped
	}()
}

func fixedTicker(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Millisecond)
	go func() {
		defer ticker.Stop() // always stop
		for {
			select {
			case <-ticker.C:
				// do work
			case <-ctx.Done():
				return
			}
		}
	}()
}

func demoTickerLeak() {
	fmt.Println()
	fmt.Println("=== Leak 3: ticker not stopped ===")

	before := snapshot("before")
	leakyTicker()
	time.Sleep(50 * time.Millisecond)
	after := snapshot("after leaky")
	fmt.Printf("  (ticker goroutine exited but timer goroutine may linger: %d)\n", after-before)

	ctx, cancel := context.WithCancel(context.Background())
	fixedTicker(ctx)
	time.Sleep(50 * time.Millisecond)
	mid := snapshot("fixed running")
	cancel()
	time.Sleep(10 * time.Millisecond)
	end := snapshot("fixed stopped")
	fmt.Printf("  goroutine cleaned up: %v\n", mid > end)
}

// ─────────────────────────────────────────────────────────────────────────────
// LEAK 4: context not propagated into blocking call
// ─────────────────────────────────────────────────────────────────────────────

func slowWork(d time.Duration) error {
	time.Sleep(d) // ignores context — leak when caller cancels
	return nil
}

func slowWorkCtx(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func demoContextLeak() {
	fmt.Println()
	fmt.Println("=== Leak 4: context-unaware blocking call ===")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	before := snapshot("before")
	done := make(chan error, 1)

	// Leaky: goroutine ignores the context, runs for full 200ms.
	go func() {
		done <- slowWork(200 * time.Millisecond)
	}()

	select {
	case <-ctx.Done():
		fmt.Printf("  caller timed out, but goroutine is still running...\n")
		time.Sleep(10 * time.Millisecond)
		snapshot("goroutine still alive")
		<-done // wait for leak to clear before next demo
	case err := <-done:
		fmt.Printf("  work completed (err=%v)\n", err)
	}
	snapshot("after leak cleared")
	fmt.Printf("  leaked goroutine count: %d\n", runtime.NumGoroutine()-before)

	// Fixed.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel2()
	before2 := runtime.NumGoroutine()
	go func() {
		done <- slowWorkCtx(ctx2, 200*time.Millisecond)
	}()
	<-done
	time.Sleep(5 * time.Millisecond)
	fmt.Printf("  fixed: goroutines after cancel = %d (change = %d)\n",
		runtime.NumGoroutine(), runtime.NumGoroutine()-before2)
}

func main() {
	demoReceiverLeak()
	demoSenderLeak()
	demoTickerLeak()
	demoContextLeak()
}
