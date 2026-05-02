// FILE: book/part4_concurrency_systems/chapter49_pipelines/examples/01_pipeline_stages/main.go
// CHAPTER: 49 — Pipelines, Fan-In/Out
// TOPIC: Pipeline stage construction — generator, transform, filter, sink;
//        context-aware cancellation through the chain; bounded pipelines
//        with back-pressure.
//
// Run (from the chapter folder):
//   go run ./examples/01_pipeline_stages

package main

import (
	"context"
	"fmt"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// PIPELINE PRIMITIVES
// ─────────────────────────────────────────────────────────────────────────────

// generate emits integers 1..n into a channel and closes it.
func generate(ctx context.Context, nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			select {
			case out <- n:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// square reads from in and emits each value squared.
func square(ctx context.Context, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case n, ok := <-in:
				if !ok {
					return
				}
				select {
				case out <- n * n:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}

// filter emits only values satisfying pred.
func filter(ctx context.Context, in <-chan int, pred func(int) bool) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case n, ok := <-in:
				if !ok {
					return
				}
				if pred(n) {
					select {
					case out <- n:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return out
}

// take emits at most n values then cancels the context.
func take(ctx context.Context, cancel context.CancelFunc, in <-chan int, n int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		defer cancel()
		count := 0
		for v := range in {
			out <- v
			count++
			if count >= n {
				return
			}
		}
	}()
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 1: linear pipeline — generate → square → filter → print
// ─────────────────────────────────────────────────────────────────────────────

func demoLinearPipeline() {
	fmt.Println("=== Linear pipeline: square + filter even ===")

	ctx := context.Background()

	nums := generate(ctx, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	squares := square(ctx, nums)
	evens := filter(ctx, squares, func(n int) bool { return n%2 == 0 })

	for v := range evens {
		fmt.Printf("  %d\n", v)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 2: infinite generator with take — early cancellation
// ─────────────────────────────────────────────────────────────────────────────

func naturals(ctx context.Context) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		n := 1
		for {
			select {
			case out <- n:
				n++
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func demoInfiniteWithTake() {
	fmt.Println()
	fmt.Println("=== Infinite source + take(5) ===")

	ctx, cancel := context.WithCancel(context.Background())
	nat := naturals(ctx)
	sq := square(ctx, nat)
	first5 := take(ctx, cancel, sq, 5)

	for v := range first5 {
		fmt.Printf("  %d\n", v)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 3: bounded pipeline — back-pressure with buffered channels
// ─────────────────────────────────────────────────────────────────────────────

func slowProducer(ctx context.Context) <-chan int {
	out := make(chan int, 2) // small buffer
	go func() {
		defer close(out)
		for i := 1; i <= 6; i++ {
			select {
			case out <- i:
				fmt.Printf("  produced %d\n", i)
			case <-ctx.Done():
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()
	return out
}

func slowConsumer(ctx context.Context, in <-chan int) {
	for {
		select {
		case <-ctx.Done():
			return
		case v, ok := <-in:
			if !ok {
				return
			}
			time.Sleep(50 * time.Millisecond) // slower than producer
			fmt.Printf("  consumed %d\n", v)
		}
	}
}

func demoBackPressure() {
	fmt.Println()
	fmt.Println("=== Back-pressure: slow consumer throttles fast producer ===")

	ctx := context.Background()
	in := slowProducer(ctx)
	slowConsumer(ctx, in)
}

func main() {
	demoLinearPipeline()
	demoInfiniteWithTake()
	demoBackPressure()
}
