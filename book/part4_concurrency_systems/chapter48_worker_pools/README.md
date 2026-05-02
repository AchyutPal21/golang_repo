# Chapter 48 — Worker Pools

## What you will learn

- The canonical N-worker pool pattern: a shared job channel drained by N goroutines
- Collecting results and errors through a dedicated output channel
- Coordinating worker shutdown with `sync.WaitGroup` and context cancellation
- `errgroup.Group` — automatic context cancellation on first error, clean `Wait()` interface
- Counting semaphores with buffered channels to cap fan-out
- Scatter-gather: launch all sub-tasks in parallel, collect when all finish
- Building a full `RunBatch` orchestrator with source / process / sink separation

---

## Core pattern: N workers sharing a job channel

```go
func startPool(ctx context.Context, n int, jobs <-chan Job, results chan<- Result) *sync.WaitGroup {
    var wg sync.WaitGroup
    for id := range n {
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
                    results <- process(job, workerID)
                }
            }
        }(id)
    }
    return &wg
}
```

- Closing `jobs` is the graceful shutdown signal.
- `ctx.Done()` provides immediate cancellation.
- The caller closes `results` after `wg.Wait()` so range-over-results works.

---

## errgroup.Group

`errgroup` (available in `golang.org/x/sync/errgroup`) pairs a `WaitGroup` with first-error collection and optional context cancellation:

```go
g, ctx := errgroup.WithContext(parent)
for _, item := range items {
    item := item
    g.Go(func() error {
        return process(ctx, item)
    })
}
err := g.Wait() // first non-nil error, or nil
```

When any goroutine returns a non-nil error, `ctx` is cancelled — all other goroutines should observe `ctx.Done()` and stop early.

---

## Semaphore via buffered channel

```go
sem := make(chan struct{}, maxConcurrent)

go func() {
    sem <- struct{}{} // acquire
    defer func() { <-sem }() // release
    doWork()
}()
```

This limits in-flight work without a fixed pool of persistent workers — good for tasks of varying duration.

---

## Scatter-gather

Fan out N tasks to N goroutines, write results into a slice indexed by position, then `wg.Wait()`:

```go
results := make([]Result, len(sources))
var wg sync.WaitGroup
for i, s := range sources {
    i, s := i, s
    wg.Add(1)
    go func() {
        defer wg.Done()
        results[i] = fetch(ctx, s)
    }()
}
wg.Wait()
```

Because each goroutine writes to a distinct index, no mutex is needed.

---

## Worker pool sizing

| Workload type | Rule of thumb |
|---|---|
| CPU-bound | `runtime.NumCPU()` workers |
| I/O-bound | 2–10× `NumCPU()` — tune by load testing |
| Mixed | Separate pools per type |

Always measure with `go test -bench` and a realistic load profile before committing to a number.

---

## Examples

| File | Demonstrates |
|---|---|
| `examples/01_basic_pool/main.go` | Fixed pool, context cancellation, error-collecting pool |
| `examples/02_errgroup_semaphore/main.go` | errgroup (all-succeed and cancel-on-error), semaphore pool, scatter-gather |

## Exercise

`exercises/01_batch_processor/main.go` — full `RunBatch` orchestrator with atomic progress, error collection, and context cancellation.
