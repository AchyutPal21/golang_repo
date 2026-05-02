// FILE: book/part4_concurrency_systems/chapter48_worker_pools/examples/01_basic_pool/main.go
// CHAPTER: 48 — Worker Pools
// TOPIC: Fixed-size worker pool, job/result channels, graceful shutdown,
//        context-aware pool, and dynamic sizing.
//
// Run (from the chapter folder):
//   go run ./examples/01_basic_pool

package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// PATTERN 1: SIMPLE WORKER POOL (job channel shared by N workers)
// ─────────────────────────────────────────────────────────────────────────────

type Job struct {
	ID    int
	Input int
}

type Result struct {
	JobID  int
	Output int
	Worker int
}

func startPool(ctx context.Context, workers int, jobs <-chan Job, results chan<- Result) *sync.WaitGroup {
	var wg sync.WaitGroup
	for id := range workers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobs:
					if !ok {
						return
					}
					// Simulate work.
					time.Sleep(time.Duration(job.Input%3+1) * 5 * time.Millisecond)
					results <- Result{
						JobID:  job.ID,
						Output: job.Input * job.Input,
						Worker: workerID,
					}
				}
			}
		}(id)
	}
	return &wg
}

func demoSimplePool() {
	fmt.Println("=== Simple worker pool (4 workers, 20 jobs) ===")

	ctx := context.Background()
	jobs := make(chan Job, 10)
	results := make(chan Result, 20)

	wg := startPool(ctx, 4, jobs, results)

	// Submit jobs.
	go func() {
		for i := range 20 {
			jobs <- Job{ID: i + 1, Input: i + 1}
		}
		close(jobs)
	}()

	// Close results after all workers finish.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results.
	workerCounts := make(map[int]int)
	for r := range results {
		workerCounts[r.Worker]++
	}

	total := 0
	for w, n := range workerCounts {
		fmt.Printf("  worker %d: %d jobs\n", w, n)
		total += n
	}
	fmt.Printf("  total processed: %d\n", total)
}

// ─────────────────────────────────────────────────────────────────────────────
// PATTERN 2: POOL WITH CONTEXT CANCELLATION
// ─────────────────────────────────────────────────────────────────────────────

func demoContextPool() {
	fmt.Println()
	fmt.Println("=== Pool with context cancellation ===")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()

	jobs := make(chan Job, 100)
	results := make(chan Result, 100)

	wg := startPool(ctx, 3, jobs, results)

	// Submit many jobs — more than can complete in 60ms.
	go func() {
		for i := range 50 {
			select {
			case jobs <- Job{ID: i + 1, Input: i + 1}:
			case <-ctx.Done():
				close(jobs)
				return
			}
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	processed := 0
	for range results {
		processed++
	}

	fmt.Printf("  processed %d of 50 jobs before timeout\n", processed)
}

// ─────────────────────────────────────────────────────────────────────────────
// PATTERN 3: POOL THAT COLLECTS ERRORS
// ─────────────────────────────────────────────────────────────────────────────

type WorkResult struct {
	ID  int
	Val int
	Err error
}

func demoErrorPool() {
	fmt.Println()
	fmt.Println("=== Pool with error collection ===")

	type errJob struct{ id, val int }
	jobs := make(chan errJob, 20)
	results := make(chan WorkResult, 20)

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if j.val%5 == 0 {
					results <- WorkResult{ID: j.id, Err: fmt.Errorf("job %d: val %d is divisible by 5", j.id, j.val)}
					continue
				}
				results <- WorkResult{ID: j.id, Val: j.val * 2}
			}
		}()
	}

	for i := range 20 {
		jobs <- errJob{id: i + 1, val: i + 1}
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	var errs []error
	successes := 0
	for r := range results {
		if r.Err != nil {
			errs = append(errs, r.Err)
		} else {
			successes++
		}
	}

	fmt.Printf("  successes: %d  errors: %d\n", successes, len(errs))
	for _, e := range errs {
		fmt.Printf("  error: %v\n", e)
	}
}

func main() {
	demoSimplePool()
	demoContextPool()
	demoErrorPool()
}
