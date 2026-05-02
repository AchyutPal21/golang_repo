// FILE: book/part4_concurrency_systems/chapter48_worker_pools/exercises/01_batch_processor/main.go
// CHAPTER: 48 — Worker Pools
// EXERCISE: Batch processor that reads records from a source, processes them
//           in parallel with a worker pool, and writes results to a sink —
//           with context cancellation, error collection, and progress reporting.
//
// Run (from the chapter folder):
//   go run ./exercises/01_batch_processor

package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN TYPES
// ─────────────────────────────────────────────────────────────────────────────

type Record struct {
	ID   int
	Data string
}

type ProcessedRecord struct {
	ID     int
	Result string
	Err    error
}

// ─────────────────────────────────────────────────────────────────────────────
// BATCH PROCESSOR
// ─────────────────────────────────────────────────────────────────────────────

type BatchConfig struct {
	Workers       int
	InputBuf      int
	OutputBuf     int
	ProgressEvery int // report progress every N records
}

type BatchStats struct {
	Total     int64
	Succeeded int64
	Failed    int64
	Duration  time.Duration
}

func RunBatch(
	ctx context.Context,
	cfg BatchConfig,
	source func(ctx context.Context, out chan<- Record),
	process func(ctx context.Context, r Record) ProcessedRecord,
	sink func(r ProcessedRecord),
) BatchStats {
	start := time.Now()

	input := make(chan Record, cfg.InputBuf)
	output := make(chan ProcessedRecord, cfg.OutputBuf)

	var (
		total     atomic.Int64
		succeeded atomic.Int64
		failed    atomic.Int64
	)

	// Source goroutine.
	go func() {
		defer close(input)
		source(ctx, input)
	}()

	// Worker pool.
	var wg sync.WaitGroup
	for range cfg.Workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case rec, ok := <-input:
					if !ok {
						return
					}
					output <- process(ctx, rec)
					total.Add(1)
					if total.Load()%int64(cfg.ProgressEvery) == 0 {
						fmt.Printf("  [progress] processed %d records\n", total.Load())
					}
				}
			}
		}()
	}

	// Close output when all workers finish.
	go func() {
		wg.Wait()
		close(output)
	}()

	// Sink.
	for r := range output {
		if r.Err != nil {
			failed.Add(1)
		} else {
			succeeded.Add(1)
		}
		sink(r)
	}

	return BatchStats{
		Total:     total.Load(),
		Succeeded: succeeded.Load(),
		Failed:    failed.Load(),
		Duration:  time.Since(start),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SCENARIOS
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	cfg := BatchConfig{
		Workers:       5,
		InputBuf:      20,
		OutputBuf:     20,
		ProgressEvery: 10,
	}

	// Scenario 1: process 50 records, 10% fail.
	fmt.Println("=== Scenario 1: 50 records, 10% error rate ===")

	source := func(ctx context.Context, out chan<- Record) {
		for i := range 50 {
			select {
			case <-ctx.Done():
				return
			case out <- Record{ID: i + 1, Data: fmt.Sprintf("item-%d", i+1)}:
			}
		}
	}

	process := func(ctx context.Context, r Record) ProcessedRecord {
		select {
		case <-ctx.Done():
			return ProcessedRecord{ID: r.ID, Err: ctx.Err()}
		case <-time.After(5 * time.Millisecond):
		}
		if r.ID%10 == 0 {
			return ProcessedRecord{ID: r.ID, Err: fmt.Errorf("record %d: processing failed", r.ID)}
		}
		return ProcessedRecord{ID: r.ID, Result: r.Data + "-processed"}
	}

	var errors []error
	var mu sync.Mutex
	sink := func(r ProcessedRecord) {
		if r.Err != nil {
			mu.Lock()
			errors = append(errors, r.Err)
			mu.Unlock()
		}
	}

	ctx := context.Background()
	stats := RunBatch(ctx, cfg, source, process, sink)

	fmt.Printf("  total=%d succeeded=%d failed=%d duration=%s\n",
		stats.Total, stats.Succeeded, stats.Failed, stats.Duration.Round(time.Millisecond))
	for _, e := range errors {
		fmt.Printf("  error: %v\n", e)
	}

	// Scenario 2: context cancelled mid-batch.
	fmt.Println()
	fmt.Println("=== Scenario 2: cancelled after 100ms ===")

	ctx2, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 200 slow records.
	source2 := func(ctx context.Context, out chan<- Record) {
		for i := range 200 {
			select {
			case <-ctx.Done():
				return
			case out <- Record{ID: i + 1}:
			}
		}
	}

	process2 := func(ctx context.Context, r Record) ProcessedRecord {
		select {
		case <-ctx.Done():
			return ProcessedRecord{ID: r.ID, Err: ctx.Err()}
		case <-time.After(15 * time.Millisecond):
			return ProcessedRecord{ID: r.ID, Result: "ok"}
		}
	}

	stats2 := RunBatch(ctx2, cfg, source2, process2, func(ProcessedRecord) {})
	fmt.Printf("  processed %d of 200 before cancellation\n", stats2.Total)
}
