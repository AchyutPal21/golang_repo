// FILE: book/part4_concurrency_systems/chapter49_pipelines/examples/02_fan_in_out/main.go
// CHAPTER: 49 — Pipelines, Fan-In/Out
// TOPIC: Fan-out (one source → N workers), fan-in (N channels → one),
//        the merge pattern, ordered fan-out via indexing, and a complete
//        parallel-download analogue.
//
// Run (from the chapter folder):
//   go run ./examples/02_fan_in_out

package main

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// MERGE (fan-in): combine N input channels into one output channel
// ─────────────────────────────────────────────────────────────────────────────

func merge(ctx context.Context, channels ...<-chan int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup

	forward := func(ch <-chan int) {
		defer wg.Done()
		for {
			select {
			case v, ok := <-ch:
				if !ok {
					return
				}
				select {
				case out <- v:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}

	wg.Add(len(channels))
	for _, ch := range channels {
		go forward(ch)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 1: fan-out → merge (unordered results)
// ─────────────────────────────────────────────────────────────────────────────

func generator(ctx context.Context, nums ...int) <-chan int {
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

func slowSquare(ctx context.Context, in <-chan int, workerID int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for {
			select {
			case n, ok := <-in:
				if !ok {
					return
				}
				delay := time.Duration(n%3+1) * 10 * time.Millisecond
				select {
				case <-time.After(delay):
					select {
					case out <- n * n:
						fmt.Printf("  worker %d: %d² = %d\n", workerID, n, n*n)
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func demoFanOutMerge() {
	fmt.Println("=== Fan-out → merge (unordered) ===")

	ctx := context.Background()
	in := generator(ctx, 1, 2, 3, 4, 5, 6)

	// Fan-out to 3 workers sharing the same input channel.
	c1 := slowSquare(ctx, in, 1)
	c2 := slowSquare(ctx, in, 2)
	c3 := slowSquare(ctx, in, 3)

	results := merge(ctx, c1, c2, c3)
	var vals []int
	for v := range results {
		vals = append(vals, v)
	}
	sort.Ints(vals)
	fmt.Printf("  sorted results: %v\n", vals)
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 2: ordered fan-out via result struct (preserve input order)
// ─────────────────────────────────────────────────────────────────────────────

type indexedResult struct {
	Index int
	Value int
}

func orderedFanOut(ctx context.Context, inputs []int, workers int, fn func(int) int) []int {
	type task struct {
		index int
		value int
	}

	jobs := make(chan task, len(inputs))
	for i, v := range inputs {
		jobs <- task{index: i, value: v}
	}
	close(jobs)

	results := make(chan indexedResult, len(inputs))
	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
					results <- indexedResult{Index: j.index, Value: fn(j.value)}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	out := make([]int, len(inputs))
	for r := range results {
		out[r.Index] = r.Value
	}
	return out
}

func demoOrderedFanOut() {
	fmt.Println()
	fmt.Println("=== Ordered fan-out: results preserve input order ===")

	ctx := context.Background()
	inputs := []int{3, 1, 4, 1, 5, 9, 2, 6}
	results := orderedFanOut(ctx, inputs, 4, func(n int) int {
		time.Sleep(time.Duration(n%3+1) * 5 * time.Millisecond)
		return n * n
	})

	fmt.Printf("  inputs:  %v\n", inputs)
	fmt.Printf("  squared: %v\n", results)
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 3: parallel fetch (download analogue) with context timeout
// ─────────────────────────────────────────────────────────────────────────────

type FetchResult struct {
	URL  string
	Body string
	Err  error
}

func parallelFetch(ctx context.Context, urls []string) []FetchResult {
	results := make([]FetchResult, len(urls))
	chans := make([]<-chan FetchResult, len(urls))

	for i, url := range urls {
		ch := make(chan FetchResult, 1)
		chans[i] = ch
		go func(u string, out chan<- FetchResult) {
			delay := time.Duration(len(u)%5+1) * 20 * time.Millisecond
			select {
			case <-time.After(delay):
				out <- FetchResult{URL: u, Body: "content of " + u}
			case <-ctx.Done():
				out <- FetchResult{URL: u, Err: ctx.Err()}
			}
		}(url, ch)
	}

	for i, ch := range chans {
		results[i] = <-ch
	}
	return results
}

func demoParallelFetch() {
	fmt.Println()
	fmt.Println("=== Parallel fetch with 150ms timeout ===")

	urls := []string{
		"/api/users",
		"/api/orders",
		"/api/products",
		"/api/inventory",
		"/api/reports", // longest name → slowest simulated fetch
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	results := parallelFetch(ctx, urls)
	for _, r := range results {
		if r.Err != nil {
			fmt.Printf("  %-20s ERROR: %v\n", r.URL, r.Err)
		} else {
			fmt.Printf("  %-20s OK: %s\n", r.URL, r.Body)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 4: multi-stage pipeline — generate → fan-out → merge → print
// ─────────────────────────────────────────────────────────────────────────────

func square(ctx context.Context, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for {
			select {
			case n, ok := <-in:
				if !ok {
					return
				}
				select {
				case out <- n * n:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func demoMultiStagePipeline() {
	fmt.Println()
	fmt.Println("=== Multi-stage: generate → 3×square → merge ===")

	ctx := context.Background()
	in := generator(ctx, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12)

	// Fan-out: three workers each drain the shared input.
	workers := make([]<-chan int, 3)
	for i := range 3 {
		workers[i] = square(ctx, in)
	}

	merged := merge(ctx, workers...)
	var results []int
	for v := range merged {
		results = append(results, v)
	}
	sort.Ints(results)
	fmt.Printf("  results (%d values): %v\n", len(results), results)
}

func main() {
	demoFanOutMerge()
	demoOrderedFanOut()
	demoParallelFetch()
	demoMultiStagePipeline()
}
