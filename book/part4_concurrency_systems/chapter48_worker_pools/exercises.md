# Chapter 48 — Exercises

## 48.1 — Batch processor

Run [`exercises/01_batch_processor`](exercises/01_batch_processor/main.go).

Full `RunBatch` orchestrator: a source goroutine pushes `Record` values into a buffered input channel; N workers process them concurrently via a `process` function; a sink goroutine collects `ProcessedRecord` results. Atomic counters track progress and print every N records.

Scenarios:
1. 50 records with 10% error rate (IDs divisible by 10 fail)
2. 200 slow records cancelled after 100ms

Try:
- Change `ProgressEvery` to 1 and observe per-record progress output.
- Add a `MaxErrors` field to `BatchConfig`; cancel the context when the error count exceeds the threshold.
- Replace `atomic.Int64` with a `sync.Mutex`-guarded struct and compare the code complexity.

## 48.2 ★ — Priority queue pool

Build a worker pool that accepts two job channels: `highPriority <-chan Job` and `lowPriority <-chan Job`. Each worker preferentially drains high-priority jobs using a nested `select`:

```go
select {
case job := <-highPriority:
    // process
default:
    select {
    case job := <-highPriority:
    case job := <-lowPriority:
    }
}
```

Submit 50 high-priority and 100 low-priority jobs; verify that most high-priority jobs are processed before most low-priority ones.

## 48.3 ★★ — Dynamic pool

Build a `DynamicPool` that adjusts its worker count at runtime:

- `New(min, max int, jobs <-chan Job) *DynamicPool`
- `Scale(n int)` — add or remove workers to reach target `n` (clamped to [min, max])
- `Stats() PoolStats` — current workers, jobs queued, jobs processed

Grow when queue depth exceeds a threshold; shrink when workers are idle. Workers that receive the scale-down signal exit after finishing their current job.
