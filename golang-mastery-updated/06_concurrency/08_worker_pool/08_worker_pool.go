// FILE: 06_concurrency/08_worker_pool.go
// TOPIC: Worker Pool — bounded concurrency, job queues, result collection
//
// Run: go run 06_concurrency/08_worker_pool.go

package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Job struct {
	ID    int
	Value int
}

type Result struct {
	JobID  int
	Output int
}

// processJob simulates CPU work
func processJob(j Job) Result {
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
	return Result{JobID: j.ID, Output: j.Value * j.Value}
}

// worker reads from jobs channel, writes to results channel
func worker(id int, jobs <-chan Job, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {  // range blocks until jobs channel is closed
		result := processJob(job)
		fmt.Printf("  worker %d processed job %d → %d\n", id, result.JobID, result.Output)
		results <- result
	}
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Worker Pool")
	fmt.Println("════════════════════════════════════════")

	const numWorkers = 3
	const numJobs = 9

	jobs := make(chan Job, numJobs)       // buffered: all jobs fit, no blocking on send
	results := make(chan Result, numJobs) // buffered: collect all results

	// Start workers BEFORE sending jobs
	var wg sync.WaitGroup
	fmt.Printf("\n── Starting %d workers ──\n", numWorkers)
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go worker(w, jobs, results, &wg)
	}

	// Send jobs
	fmt.Printf("── Sending %d jobs ──\n", numJobs)
	for i := 1; i <= numJobs; i++ {
		jobs <- Job{ID: i, Value: i}
	}
	close(jobs)  // closing jobs tells workers: no more jobs, exit range loop

	// Wait for all workers to finish, then close results
	go func() {
		wg.Wait()
		close(results)  // safe to close now — all workers done
	}()

	// Collect results
	fmt.Println("\n── Results ──")
	var total int
	for r := range results {
		total += r.Output
	}
	fmt.Printf("  Total of all squares: %d\n", total)

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  Worker pool: N workers, M jobs via buffered channel")
	fmt.Println("  close(jobs) signals workers to stop (range exits)")
	fmt.Println("  WaitGroup tracks when all workers finish")
	fmt.Println("  close(results) only after all workers done")
	fmt.Println("  Tune numWorkers to match CPU cores or I/O concurrency")
}
