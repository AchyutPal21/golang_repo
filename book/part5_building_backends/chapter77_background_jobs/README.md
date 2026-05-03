# Chapter 77 — Background Jobs and Schedulers

## What you'll learn

How to build reliable background job processing in Go: typed job queues with priority, retry logic and dead-letter queues, worker pools, cron-style schedulers, one-shot delayed jobs, distributed locking to prevent duplicate execution, and job lifecycle hooks.

## Key concepts

| Concept | Description |
|---|---|
| Job queue | Buffered work list; workers drain it concurrently |
| Priority queue | Higher-priority jobs run before lower-priority ones |
| Retry | Re-enqueue failed jobs up to maxRetries; each attempt increments counter |
| Dead-letter queue | Destination for jobs that exhaust retries; enables inspection and replay |
| Worker pool | N goroutines draining one queue; concurrency = N |
| Scheduler | Fires tasks on a schedule (interval, cron, one-shot) |
| Distributed lock | Prevents duplicate execution of the same cron job across instances |
| Idempotency | Job handler produces the same result if called multiple times |
| Job middleware | Logging, timeout, tracing wrappers applied to all jobs of a type |

## Files

| File | Topic |
|---|---|
| `examples/01_job_queue/main.go` | Priority queue, retry, DLQ, worker pool, unique jobs |
| `examples/02_scheduler/main.go` | Every/Daily/At schedules, delayed jobs, distributed lock, hooks |
| `exercises/01_job_processor/main.go` | Typed jobs, middleware chain, idempotency, DLQ, observability |

## Job queue pattern

```go
// Register typed handlers.
q.Register("email", func(ctx context.Context, job *Job) error {
    ep := job.Payload.(EmailPayload)
    return sendEmail(ctx, ep.To, ep.Subject)
})

// Enqueue work.
q.Enqueue("email", priority, maxRetries, EmailPayload{To: "alice@example.com"})

// Worker pool drains the queue.
pool := NewWorkerPool(q, 5)
pool.Start(ctx)
```

## Retry and DLQ

```go
func (q *Queue) complete(job *Job, err error) {
    if err == nil { job.Status = Done; return }
    job.Attempts++
    if job.Attempts >= job.MaxRetries {
        q.dlq = append(q.dlq, job) // dead letter
        return
    }
    q.pending = append(q.pending, job) // retry
}
```

## Scheduler

```go
sched := NewScheduler()
sched.Add(&Task{
    Name:     "metrics-flush",
    Schedule: Every{5 * time.Minute},
    Run: func(ctx context.Context) error {
        return flushMetrics(ctx)
    },
})
go sched.Run(ctx)
```

## Distributed lock (prevent duplicate cron execution)

```go
// In production: use Redis SET NX EX.
if !lock.Acquire("daily-report", instanceID, 5*time.Minute) {
    return // another instance is running it
}
defer lock.Release("daily-report", instanceID)
runDailyReport(ctx)
```

## Idempotency in job handlers

```go
func handleOrder(ctx context.Context, job *Job) error {
    op := job.Payload.(OrderPayload)
    key := "process-order:" + op.OrderID
    if idempotency.Check(key) {
        return nil // already processed
    }
    return processOrder(ctx, op)
}
```

## Production libraries

| Library | Use case |
|---|---|
| `github.com/hibiken/asynq` | Redis-backed queue; priority, unique jobs, scheduled tasks |
| `github.com/riverqueue/river` | Postgres-backed; strong ACID guarantees, zero deps |
| `github.com/gocraft/work` | Redis-backed with concurrency control |
| `github.com/robfig/cron/v3` | Cron expression parsing and scheduling |

```go
// asynq example
client := asynq.NewClient(asynq.RedisClientOpt{Addr: ":6379"})
task := asynq.NewTask("email:send", payload,
    asynq.MaxRetry(5),
    asynq.TaskID("email:send:"+email), // unique by email
)
client.Enqueue(task)
```

## Production notes

- Always set a deadline on job handlers with `context.WithTimeout` — a hung job occupies a worker slot
- Use `asynq` or `river` rather than an in-memory queue in production — durable storage survives restarts
- Design all handlers to be idempotent; at-least-once delivery is unavoidable under retry
- Expose DLQ as a route in your admin panel so engineers can inspect and replay failed jobs
- Track job duration histograms and DLQ size as operational metrics
