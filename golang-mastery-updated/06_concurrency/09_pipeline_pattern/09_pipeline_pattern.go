// FILE: 06_concurrency/09_pipeline_pattern.go
// TOPIC: Pipeline Pattern — stage-connected channels, fan-out, cancellation
//
// Run: go run 06_concurrency/09_pipeline_pattern.go

package main

import (
	"fmt"
	"sync"
)

// ── STAGE FUNCTIONS ───────────────────────────────────────────────────────────
// Each stage: receives from an input channel, processes, sends to output channel.
// Returns <-chan T so the caller gets a read-only channel.

// generate: source stage — emits numbers
func generate(done <-chan struct{}, nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			select {
			case out <- n:
			case <-done:  // cancelled — stop early
				return
			}
		}
	}()
	return out
}

// square: transform stage — squares each number
func square(done <-chan struct{}, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			select {
			case out <- n * n:
			case <-done:
				return
			}
		}
	}()
	return out
}

// filter: filter stage — only passes values > threshold
func filter(done <-chan struct{}, in <-chan int, threshold int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			if n > threshold {
				select {
				case out <- n:
				case <-done:
					return
				}
			}
		}
	}()
	return out
}

// ── FAN-OUT / FAN-IN ─────────────────────────────────────────────────────────
// Fan-out: split one channel into N parallel workers
// Fan-in: merge N channels into one

func merge(done <-chan struct{}, channels ...<-chan int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup
	output := func(c <-chan int) {
		defer wg.Done()
		for n := range c {
			select {
			case out <- n:
			case <-done:
				return
			}
		}
	}
	wg.Add(len(channels))
	for _, c := range channels {
		go output(c)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Pipeline Pattern")
	fmt.Println("════════════════════════════════════════")

	// ── Simple linear pipeline ─────────────────────────────────────────
	fmt.Println("\n── Linear pipeline: generate → square → filter(>10) ──")
	done := make(chan struct{})
	defer close(done)

	// Connect stages: generate → square → filter
	nums := generate(done, 1, 2, 3, 4, 5, 6)
	squares := square(done, nums)
	filtered := filter(done, squares, 10)

	fmt.Print("  Results: ")
	for v := range filtered {
		fmt.Printf("%d ", v)
	}
	fmt.Println()

	// ── Fan-out / Fan-in ───────────────────────────────────────────────
	fmt.Println("\n── Fan-out / Fan-in ──")
	done2 := make(chan struct{})
	defer close(done2)

	source := generate(done2, 1, 2, 3, 4, 5, 6, 7, 8)
	// Fan-out: two parallel square stages
	sq1 := square(done2, source)
	// (In real fan-out, you'd fan out source to multiple workers.
	//  Here we demo merge with two independent pipelines.)
	source2 := generate(done2, 10, 20, 30)
	sq2 := square(done2, source2)
	// Fan-in: merge both into one output channel
	merged := merge(done2, sq1, sq2)

	fmt.Print("  Merged results: ")
	for v := range merged {
		fmt.Printf("%d ", v)
	}
	fmt.Println()

	// ── Early cancellation ─────────────────────────────────────────────
	fmt.Println("\n── Early cancellation via done channel ──")
	done3 := make(chan struct{})
	bigSource := generate(done3, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	bigSquares := square(done3, bigSource)

	count := 0
	for v := range bigSquares {
		fmt.Printf("  got %d\n", v)
		count++
		if count == 3 {
			close(done3)  // cancel pipeline — all stages will stop
			break
		}
	}
	fmt.Println("  Pipeline cancelled after 3 values")

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  Pipeline: stages connected by channels")
	fmt.Println("  Each stage: goroutine reading input, writing output channel")
	fmt.Println("  Done channel: propagate cancellation through all stages")
	fmt.Println("  Fan-out: one source → multiple parallel workers")
	fmt.Println("  Fan-in: merge multiple channels → one (merge function)")
	fmt.Println("  close(done) cancels everything — clean shutdown")
}
