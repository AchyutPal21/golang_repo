// FILE: book/part4_concurrency_systems/chapter42_goroutines_internals/exercises/01_goroutine_pool/main.go
// CHAPTER: 42 — Goroutines: Internals
// EXERCISE: Build a bounded goroutine pool that processes jobs concurrently
//           but caps the number of live goroutines, collects errors, and
//           shuts down cleanly when a done channel is closed.
//
// Run (from the chapter folder):
//   go run ./exercises/01_goroutine_pool

package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// JOB / RESULT TYPES
// ─────────────────────────────────────────────────────────────────────────────

type Job struct {
	ID      int
	Payload int
}

type Result struct {
	JobID  int
	Output int
	Err    error
}

// ─────────────────────────────────────────────────────────────────────────────
// POOL
// ─────────────────────────────────────────────────────────────────────────────

type Pool struct {
	workers int
	jobs    chan Job
	results chan Result
	done    chan struct{}
	wg      sync.WaitGroup
}

func NewPool(workers int) *Pool {
	return &Pool{
		workers: workers,
		jobs:    make(chan Job, workers*2),
		results: make(chan Result, workers*2),
		done:    make(chan struct{}),
	}
}

// Start launches worker goroutines. Each worker processes jobs until the jobs
// channel is closed or the done channel is signalled.
func (p *Pool) Start(process func(Job) Result) {
	for range p.workers {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-p.done:
					return
				case job, ok := <-p.jobs:
					if !ok {
						return
					}
					p.results <- process(job)
				}
			}
		}()
	}
}

// Submit sends a job to the pool. Returns false if the pool is shutting down.
func (p *Pool) Submit(j Job) bool {
	select {
	case <-p.done:
		return false
	case p.jobs <- j:
		return true
	}
}

// Close signals workers to stop after draining submitted jobs.
func (p *Pool) Close() {
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
}

// Shutdown immediately cancels all workers. Workers that are blocked sending
// to results will unblock once the goroutine below drains the channel.
func (p *Pool) Shutdown() {
	close(p.done)
	// Drain results so blocked workers can exit.
	go func() {
		for range p.results {
		}
	}()
	p.wg.Wait()
	close(p.results)
}

// Results returns the results channel for reading.
func (p *Pool) Results() <-chan Result { return p.results }

// ─────────────────────────────────────────────────────────────────────────────
// PROCESS FUNCTION — simulates work with occasional errors
// ─────────────────────────────────────────────────────────────────────────────

func process(j Job) Result {
	time.Sleep(time.Duration(j.Payload) * time.Millisecond)
	if j.ID%7 == 0 {
		return Result{JobID: j.ID, Err: fmt.Errorf("job %d: simulated failure", j.ID)}
	}
	return Result{JobID: j.ID, Output: j.Payload * j.Payload}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	const (
		numWorkers = 4
		numJobs    = 20
	)

	fmt.Printf("Pool: %d workers, %d jobs\n\n", numWorkers, numJobs)

	pool := NewPool(numWorkers)
	pool.Start(process)

	// Submit jobs in a separate goroutine so main can concurrently collect results.
	go func() {
		for i := range numJobs {
			pool.Submit(Job{ID: i + 1, Payload: (i % 5) + 1})
		}
		pool.Close()
	}()

	var (
		successes []Result
		failures  []error
	)
	for r := range pool.Results() {
		if r.Err != nil {
			failures = append(failures, r.Err)
		} else {
			successes = append(successes, r)
		}
	}

	fmt.Printf("Successes: %d\n", len(successes))
	for _, r := range successes {
		fmt.Printf("  job %2d → %d\n", r.JobID, r.Output)
	}

	fmt.Printf("\nFailures: %d\n", len(failures))
	for _, e := range failures {
		fmt.Printf("  %v\n", e)
	}

	// Demonstrate early shutdown.
	fmt.Println()
	fmt.Println("=== Early shutdown demo ===")
	pool2 := NewPool(2)
	pool2.Start(process)

	submitted := 0
	for i := range 30 {
		if !pool2.Submit(Job{ID: i + 1, Payload: 5}) {
			break
		}
		submitted++
		if i == 9 {
			// Cancel after 10 jobs submitted.
			pool2.Shutdown()
			break
		}
	}

	processed := 0
	var shutdownErrs []error
	for r := range pool2.Results() {
		if r.Err != nil {
			shutdownErrs = append(shutdownErrs, r.Err)
		} else {
			processed++
		}
	}
	fmt.Printf("  submitted %d, processed %d before shutdown\n", submitted, processed)
	fmt.Printf("  errors during shutdown: %v\n", errors.Join(shutdownErrs...))
}
