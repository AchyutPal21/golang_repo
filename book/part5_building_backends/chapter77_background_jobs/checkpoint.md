# Chapter 77 Checkpoint — Background Jobs and Schedulers

## Self-assessment questions

1. Why must background job handlers be idempotent? What could go wrong if they're not?
2. What is the purpose of a dead-letter queue? How should it be operationally managed?
3. Why do distributed cron jobs need a locking mechanism? What happens without one?
4. What is the difference between a priority queue and a FIFO queue for job processing? When does priority matter?
5. How do you prevent a slow or hung job handler from blocking all workers?

## Checklist

- [ ] Can implement a job queue with Enqueue, dequeue, Ack/Nack, and DLQ
- [ ] Can implement retry logic with attempt counting and DLQ routing
- [ ] Can implement a priority queue (higher priority dequeued first)
- [ ] Can implement a worker pool with N concurrent goroutines
- [ ] Can implement a scheduler with configurable schedules (interval, one-shot)
- [ ] Can implement a distributed lock for cron job deduplication
- [ ] Can implement idempotency in a job handler using a seen-key store
- [ ] Can implement job middleware (logging, timeout) composable with Chain

## Answers

1. With at-least-once delivery, a job may be executed more than once (network timeout before commit, worker crash after processing but before ack). If the handler is not idempotent, a charge-card job could charge twice, a send-email job could send two emails. Idempotency means: check a deduplication key (e.g., `"charge:order-123"`) before doing the work; if already processed, return success without re-executing.

2. The DLQ holds jobs that have exhausted their retry budget. It serves two purposes: (1) prevent the bad job from blocking the queue indefinitely, and (2) give operators a way to inspect why the job failed and replay it after fixing the root cause. Without a DLQ, failed jobs either loop forever (burning retries) or are silently discarded (data loss). Operational practice: alert when DLQ size exceeds a threshold; provide an admin UI to inspect payloads and replay.

3. Without a distributed lock, every instance running the cron job fires at the same scheduled time — if 5 instances are deployed, the daily report generates 5 times. The lock ensures only one instance runs the job at a time. The lock TTL must exceed the expected job duration; if the owner crashes, the lock auto-expires and another instance can take over.

4. A priority queue ensures high-importance jobs (fraud alerts, payment confirmations) are processed before low-importance ones (analytics, newsletter). Without priority, a burst of low-priority jobs delays critical high-priority work that arrived later. Priority matters when job types have different SLAs.

5. Apply a context timeout via middleware: `tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)`. If the handler exceeds the deadline, ctx cancellation propagates into any downstream calls. The handler returns an error, the job is retried (or DLQed). Each worker goroutine exits `handler(tCtx, job)` and is free to pick up the next job — a hung handler never blocks other workers.
