// 03_channels_patterns.go
//
// CHANNEL PATTERNS — idiomatic Go concurrency recipes
//
// This file covers the essential channel-based patterns that appear throughout
// real Go codebases. Understanding these patterns lets you recognize and
// implement concurrent designs quickly and correctly.
//
// PATTERNS COVERED:
//   1. Range over channel (until closed)
//   2. Done/quit channel for cancellation
//   3. Fan-out (one source → multiple receivers)
//   4. Fan-in (merge multiple channels → one)
//   5. Timeout with time.After
//   6. Generator pattern (function returning <-chan T)
//
// MENTAL MODEL: channels are directional pipes between goroutines.
// These patterns compose those pipes into larger data-flow graphs.

package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// =============================================================================
// SECTION 1: Range over channel
// =============================================================================
//
// "for v := range ch" loops, receiving values from ch until ch is CLOSED.
// It's syntactic sugar for:
//   for {
//       v, ok := <-ch
//       if !ok { break }
//       // use v
//   }
//
// KEY: the sender MUST close the channel or range loops forever (goroutine leak).
// KEY: only the sender knows when there are no more values — so SENDER closes.

func sendNumbers(out chan<- int, n int) {
	for i := 0; i < n; i++ {
		out <- i
	}
	close(out) // signals "no more values" to the range loop
}

func demoRangeOverChannel() {
	fmt.Println("=== Range Over Channel ===")

	ch := make(chan int, 5)
	go sendNumbers(ch, 7)

	// This loop automatically exits when ch is closed and drained.
	for val := range ch {
		fmt.Printf("  received: %d\n", val)
	}
	fmt.Println("  loop done (channel was closed)")
	fmt.Println()
}

// =============================================================================
// SECTION 2: Done channel — cooperative cancellation
// =============================================================================
//
// A "done" channel is a channel of struct{} (zero bytes) that is CLOSED to
// signal all interested goroutines to stop. It's a broadcast mechanism.
//
// WHY CLOSE INSTEAD OF SEND?
//   - close(done) unblocks ALL goroutines waiting on <-done simultaneously.
//   - Sending a value (done <- struct{}{}) only unblocks ONE waiter.
//   - For broadcast cancellation, close is the correct tool.
//
// This is the pattern that the context package formalizes (see 10_context).

func worker(id int, jobs <-chan int, done <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case job, ok := <-jobs:
			if !ok {
				// jobs channel was closed — no more work
				fmt.Printf("  worker %d: jobs channel closed, exiting\n", id)
				return
			}
			fmt.Printf("  worker %d: processing job %d\n", id, job)
			time.Sleep(10 * time.Millisecond)

		case <-done:
			// done was closed — stop immediately regardless of pending jobs
			fmt.Printf("  worker %d: received cancel signal, stopping\n", id)
			return
		}
	}
}

func demoDoneChannel() {
	fmt.Println("=== Done Channel (cancellation) ===")

	jobs := make(chan int, 20)
	done := make(chan struct{})
	var wg sync.WaitGroup

	// Start 3 workers
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go worker(i, jobs, done, &wg)
	}

	// Send some jobs
	for i := 0; i < 10; i++ {
		jobs <- i
	}

	// After a short time, cancel all workers
	time.Sleep(35 * time.Millisecond)
	fmt.Println("  main: sending cancel signal (closing done)")
	close(done) // broadcasts to ALL workers simultaneously

	wg.Wait()
	fmt.Println("  all workers stopped")
	fmt.Println()
}

// =============================================================================
// SECTION 3: Fan-out — distributing work across multiple goroutines
// =============================================================================
//
// Fan-out: one input channel feeds N worker goroutines.
// Workers compete to receive from the shared input channel.
// This is how you parallelize CPU-bound or I/O-bound work in Go.
//
// The Go runtime ensures no two goroutines receive the same value from a channel —
// channel receives are atomic at the application level.
//
//           ┌─ worker 1
// input ────┼─ worker 2
//           └─ worker 3
//
// WHEN TO FAN OUT:
//   - Processing items is slow (I/O, computation)
//   - Items are independent (order doesn't matter)
//   - You want to use multiple CPU cores

func demoFanOut() {
	fmt.Println("=== Fan-Out (one source, multiple receivers) ===")

	work := make(chan int, 20)
	results := make(chan string, 20)
	var wg sync.WaitGroup

	// Fan-out: 3 workers all read from the SAME work channel
	numWorkers := 3
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for job := range work { // each worker competes for jobs
				time.Sleep(5 * time.Millisecond) // simulate work
				results <- fmt.Sprintf("worker-%d processed job-%d", id, job)
			}
		}(w)
	}

	// Send 9 jobs (3 per worker on average, but distribution varies)
	for i := 1; i <= 9; i++ {
		work <- i
	}
	close(work) // closing signals workers: no more jobs coming

	// Wait for all workers, then close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for result := range results {
		fmt.Printf("  %s\n", result)
	}
	fmt.Println()
}

// =============================================================================
// SECTION 4: Fan-in — merging multiple channels into one
// =============================================================================
//
// Fan-in (also called "merge"): multiple goroutines each produce on their own
// channel; a fan-in goroutine reads from all of them and writes to one output.
//
// Use case: aggregate results from multiple parallel workers.
//
// worker 1 ─┐
// worker 2 ─┼─ fanIn ──► merged output
// worker 3 ─┘

// fanIn merges any number of input channels into a single output channel.
// It starts one goroutine per input channel and a coordinator goroutine.
func fanIn(done <-chan struct{}, channels ...<-chan int) <-chan int {
	merged := make(chan int, 10)
	var wg sync.WaitGroup

	// For each input channel, start a goroutine that forwards values to merged
	forward := func(ch <-chan int) {
		defer wg.Done()
		for {
			select {
			case val, ok := <-ch:
				if !ok {
					return // this input channel is closed
				}
				select {
				case merged <- val:
				case <-done: // respect cancellation
					return
				}
			case <-done:
				return
			}
		}
	}

	wg.Add(len(channels))
	for _, ch := range channels {
		go forward(ch)
	}

	// Close merged when all forwarders are done
	go func() {
		wg.Wait()
		close(merged)
	}()

	return merged
}

// generate produces n values on a channel and closes it.
func generate(done <-chan struct{}, label string, n int) <-chan int {
	out := make(chan int, n)
	go func() {
		defer close(out)
		for i := 0; i < n; i++ {
			select {
			case out <- rand.Intn(100):
				fmt.Printf("  [%s] generated %d\n", label, i)
				time.Sleep(time.Duration(rand.Intn(20)) * time.Millisecond)
			case <-done:
				fmt.Printf("  [%s] cancelled\n", label)
				return
			}
		}
	}()
	return out
}

func demoFanIn() {
	fmt.Println("=== Fan-In (merge multiple channels into one) ===")

	done := make(chan struct{})
	defer close(done)

	// Three independent producers
	ch1 := generate(done, "producer-A", 3)
	ch2 := generate(done, "producer-B", 3)
	ch3 := generate(done, "producer-C", 3)

	// Merge all three into one channel
	merged := fanIn(done, ch1, ch2, ch3)

	// Consume from a single channel — values arrive interleaved from all producers
	count := 0
	for val := range merged {
		fmt.Printf("  merged received: %d\n", val)
		count++
	}
	fmt.Printf("  total values received: %d\n", count)
	fmt.Println()
}

// =============================================================================
// SECTION 5: Timeout with time.After
// =============================================================================
//
// time.After(d) returns a <-chan time.Time that receives the current time
// after duration d. Used in select to implement timeouts.
//
// IMPORTANT LEAK NOTE: time.After creates a timer that the GC won't collect
// until it fires. In tight loops, use time.NewTimer with t.Stop() instead.
// For illustrative select timeouts, time.After is fine.

func slowOperation(out chan<- string, delay time.Duration) {
	time.Sleep(delay)
	out <- "operation complete"
}

func demoTimeout() {
	fmt.Println("=== Timeout with time.After ===")

	result := make(chan string, 1)

	// Case 1: operation completes before timeout
	go slowOperation(result, 30*time.Millisecond)
	select {
	case res := <-result:
		fmt.Printf("  fast operation: %s\n", res)
	case <-time.After(100 * time.Millisecond):
		fmt.Println("  fast operation: timed out!")
	}

	// Case 2: operation is too slow — timeout fires first
	go slowOperation(result, 200*time.Millisecond)
	select {
	case res := <-result:
		fmt.Printf("  slow operation: %s\n", res)
	case <-time.After(50 * time.Millisecond):
		fmt.Println("  slow operation: timed out! (expected)")
	}

	// -------------------------------------------------------------------------
	// Per-attempt timeout in a retry loop (the correct pattern):
	// -------------------------------------------------------------------------
	fmt.Println("  Retry with per-attempt timeout:")
	attempts := 0
	for attempts < 3 {
		timer := time.NewTimer(30 * time.Millisecond)
		select {
		case res := <-result:
			timer.Stop() // IMPORTANT: stop timer to free resources
			fmt.Printf("  attempt %d: got result: %s\n", attempts+1, res)
			goto done
		case <-timer.C:
			fmt.Printf("  attempt %d: timed out, retrying...\n", attempts+1)
			attempts++
		}
	}
	fmt.Println("  all attempts exhausted")
done:
	fmt.Println()
}

// =============================================================================
// SECTION 6: Generator pattern — function returning <-chan T
// =============================================================================
//
// A generator is a function that launches a goroutine and returns a channel
// through which the goroutine produces values. The caller drives the pace
// (the generator blocks when the channel buffer is full or when the caller
// doesn't read).
//
// This is the idiomatic way to represent lazy sequences or streams in Go.
// It's Go's version of Python generators or Rust iterators.
//
// SIGNATURE CONVENTION: always return <-chan T (receive-only) so the caller
// cannot accidentally send on the producer's channel.

// fibonacci returns a receive-only channel that produces Fibonacci numbers.
// It respects a done channel for cancellation.
func fibonacci(n int, done <-chan struct{}) <-chan int {
	out := make(chan int) // unbuffered: producer blocks until consumer is ready
	go func() {
		defer close(out)
		a, b := 0, 1
		for i := 0; i < n; i++ {
			select {
			case out <- a:
				a, b = b, a+b
			case <-done:
				return // stop producing if cancelled
			}
		}
	}()
	return out
}

// counter is a simple infinite generator (stopped via done channel).
func counter(start int, done <-chan struct{}) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		n := start
		for {
			select {
			case out <- n:
				n++
			case <-done:
				return
			}
		}
	}()
	return out
}

// squares wraps a number generator and squares each value — composable generators.
func squares(in <-chan int, done <-chan struct{}) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for {
			select {
			case v, ok := <-in:
				if !ok {
					return
				}
				select {
				case out <- v * v:
				case <-done:
					return
				}
			case <-done:
				return
			}
		}
	}()
	return out
}

func demoGeneratorPattern() {
	fmt.Println("=== Generator Pattern ===")

	done := make(chan struct{})

	// Fibonacci generator
	fmt.Println("  Fibonacci sequence (first 10):")
	for fib := range fibonacci(10, done) {
		fmt.Printf("  %d", fib)
	}
	fmt.Println()

	// Composed generators: counter → squares (pipeline of generators)
	fmt.Println("\n  First 5 perfect squares via composed generators:")
	nums := counter(1, done)
	sqrs := squares(nums, done)
	for i := 0; i < 5; i++ {
		fmt.Printf("  %d", <-sqrs)
	}
	close(done) // cancel both goroutines
	fmt.Println()

	fmt.Println()
}

// =============================================================================
// SECTION 7: Combining patterns — pipeline with cancellation
// =============================================================================
//
// This example shows a realistic mini-pipeline:
//   generate → filter → transform → collect
// With a done channel flowing through all stages for cancellation.

func generateInts(done <-chan struct{}, vals ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, v := range vals {
			select {
			case out <- v:
			case <-done:
				return
			}
		}
	}()
	return out
}

func filterEven(done <-chan struct{}, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			if v%2 == 0 {
				select {
				case out <- v:
				case <-done:
					return
				}
			}
		}
	}()
	return out
}

func doubleValues(done <-chan struct{}, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			select {
			case out <- v * 2:
			case <-done:
				return
			}
		}
	}()
	return out
}

func demoCombinedPipeline() {
	fmt.Println("=== Combined Pipeline: generate → filter(even) → double ===")

	done := make(chan struct{})
	defer close(done)

	source := generateInts(done, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	evens := filterEven(done, source)
	doubled := doubleValues(done, evens)

	fmt.Print("  results: ")
	for v := range doubled {
		fmt.Printf("%d ", v)
	}
	fmt.Println()
	fmt.Println("  (even numbers from 1-10, doubled: 4 8 12 16 20)")
	fmt.Println()
}

// =============================================================================
// MAIN
// =============================================================================

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║         CHANNEL PATTERNS — Deep Dive                 ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	fmt.Println()

	demoRangeOverChannel()
	demoDoneChannel()
	demoFanOut()
	demoFanIn()
	demoTimeout()
	demoGeneratorPattern()
	demoCombinedPipeline()

	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("KEY TAKEAWAYS:")
	fmt.Println("  1. range ch: loops until channel is closed (sender must close)")
	fmt.Println("  2. done channel: close(done) broadcasts cancel to ALL goroutines")
	fmt.Println("  3. Fan-out: N workers share one input channel (they compete)")
	fmt.Println("  4. Fan-in: merge N channels into 1 (one goroutine per input)")
	fmt.Println("  5. time.After: creates a one-shot timeout channel")
	fmt.Println("  6. Generator: function returns <-chan T; goroutine closes when done")
	fmt.Println("  7. Pipelines: chain generators via channels; pass done for cancel")
	fmt.Println("═══════════════════════════════════════════════════════")
}
