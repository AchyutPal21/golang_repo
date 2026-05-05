# Capstone F — Job Queue

A production-grade, SQS-inspired job queue built entirely in Go with no external dependencies. This project ties together concurrency primitives, priority scheduling, visibility timeouts, dead-letter queues, and worker-pool management — patterns covered in Parts III–VI.

## What you build

A self-contained job queue system where:
- Producers `Enqueue` jobs with a priority (High / Normal / Low) and a job type
- `Dequeue` always returns the highest-priority available job and marks it **invisible** for the duration of its visibility timeout
- Workers call `Ack` on success (permanent removal) or `Nack` on failure (retry or dead-letter)
- Jobs that exhaust `MaxAttempts` are moved to the **Dead-Letter Queue (DLQ)**
- The DLQ supports `List` and `Requeue` so operators can inspect and replay failures
- A `WorkerPool` runs N goroutines, each continuously dequeuing and dispatching to registered `JobHandler` implementations
- Atomic counters track processed, acked, nacked, and dead-lettered totals

## Architecture

```
Producers
    │
    │  Enqueue(job)
    ▼
┌─────────────────────────────────────┐
│               Queue                 │
│                                     │
│  ┌──────────┐ ┌────────┐ ┌───────┐ │
│  │ High (1) │ │ Normal │ │Low (3)│ │  <── priority lanes (sorted slice)
│  └──────────┘ │  (2)   │ └───────┘ │
│               └────────┘           │
│                                     │
│  invisibilityMap: jobID → deadline  │  <── visibility timeout tracking
│  inFlight:        jobID → Job       │  <── jobs currently being processed
└──────────────┬──────────────────────┘
               │  Dequeue()  (highest priority, timeout not expired)
               ▼
    ┌─────────────────────┐
    │     WorkerPool      │
    │  goroutine × N      │
    │                     │
    │  Dequeue → dispatch │
    │  Handler.Handle()   │
    │  Ack  /  Nack       │
    └──────────┬──────────┘
               │
        ┌──────┴──────┐
        │             │
    success        failure
        │             │
      Ack()         Nack()
        │             │
   removed      attempts++
                  │
          maxAttempts reached?
                  │ yes
                  ▼
        ┌─────────────────┐
        │  DeadLetterQueue│
        │  Add / List     │
        │  Requeue        │
        └─────────────────┘
```

## Key components

| Component | Responsibility |
|---|---|
| `Priority` | Typed constant: `High=1`, `Normal=2`, `Low=3` |
| `Job` | Unit of work: ID, Type, Payload, Priority, Attempts, MaxAttempts, VisibilityTimeout, EnqueuedAt |
| `Queue` | Thread-safe store; Enqueue, Dequeue (priority + visibility), Ack, Nack |
| `DeadLetterQueue` | Stores exhausted jobs; supports List and Requeue back to main Queue |
| `JobHandler` | Interface `Handle(Job) error`; concrete handlers registered by job type |
| `HandlerRegistry` | Maps job type string → JobHandler |
| `WorkerPool` | Launches N goroutines; each runs the dequeue-dispatch-ack/nack loop |
| `Stats` | Atomic counters: enqueued, processed, acked, nacked, dead-lettered |

## Running

```bash
# From the repo root
go run ./part7_capstone_projects/capstone_f_job_queue

# Or build first
go build ./part7_capstone_projects/capstone_f_job_queue
./capstone_f_job_queue
```

Expected output shows:
1. 10 jobs enqueued with mixed priorities and types
2. 2 workers processing them concurrently
3. High-priority jobs completing before Low-priority jobs
4. The `poison-pill` job type failing on every attempt and landing in the DLQ
5. Final stats: total processed, acked, nacked, dead-lettered, and DLQ contents

## What it tests

| Concept | Where demonstrated |
|---|---|
| `sync.Mutex` for shared queue state | `Queue`, `DeadLetterQueue` |
| `sync/atomic` for lock-free counters | `Stats` fields |
| Goroutines + `sync.WaitGroup` | `WorkerPool.Start` / `Stop` |
| Priority scheduling without a heap | sorted-insertion by `Priority` value |
| Visibility timeout (re-appear after deadline) | `Queue.Dequeue` checks `invisibilityMap` |
| At-least-once delivery semantics | Nack re-enqueues; only Ack removes |
| Dead-letter queue + requeue | `DeadLetterQueue.Requeue` moves job back |
| Interface-driven dispatch | `JobHandler` registered by type string |
| Context-based graceful shutdown | `WorkerPool.Stop` drains workers cleanly |
