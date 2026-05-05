# Scaling Discussion — Job Queue

## 1. Visibility timeout vs ACK: why not just delete on dequeue?

The naive approach is to delete a job the moment a worker picks it up. The problem is that workers crash. If a worker dequeues job J and then panics, OOM-kills, or loses its network connection before finishing, J is gone forever with no record of what happened.

Visibility timeout solves this with a **two-phase commit at the message layer**:

1. Worker dequeues → job becomes *invisible* (not deleted) for a configurable window (e.g., 30 s).
2. If the worker calls `Ack` in time → job is permanently deleted.
3. If the worker crashes or times out → the broker's reaper notices the deadline has passed and makes the job *visible again*. Another worker picks it up.

This gives you **at-least-once delivery** with zero coordination: the broker is the sole source of truth, and no separate "heartbeat" or lock-renewal protocol is needed. The timeout window should be set to slightly longer than the 99th-percentile processing time for that job type, so normal jobs ack before timeout while genuinely stuck jobs are reclaimed promptly.

---

## 2. At-least-once vs exactly-once delivery

| Property | At-least-once | Exactly-once |
|---|---|---|
| Delivery guarantee | Every job is processed ≥ 1 time | Every job is processed exactly 1 time |
| Failure model | Retries → possible duplicates | Requires distributed transaction or idempotency token |
| Complexity | Low | High |
| Common systems | SQS standard, Kafka, RabbitMQ | SQS FIFO (deduplication window), Kafka transactions |

**At-least-once** is almost always the right default. The burden of handling duplicates is pushed to the handler via **idempotency**: each job carries a unique ID, and handlers check "have I already processed this ID?" (e.g., via a `processed_jobs` table with a unique index). Writing this check once per handler type is far cheaper than building exactly-once infrastructure.

**Exactly-once** requires the broker and the handler's side-effect store to participate in the same atomic transaction, which is impractical across a network boundary. Kafka transactions can give exactly-once delivery inside the Kafka ecosystem (producer + consumer + Kafka Streams), but the moment you write to Postgres or send an email the guarantee evaporates unless you implement the idempotency check yourself anyway.

---

## 3. Postgres as a queue: SELECT FOR UPDATE SKIP LOCKED

Postgres can serve as a surprisingly capable job queue for moderate workloads (up to ~10 k jobs/s on a single node):

```sql
-- Schema
CREATE TABLE jobs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type        TEXT        NOT NULL,
    payload     JSONB       NOT NULL,
    priority    SMALLINT    NOT NULL DEFAULT 2,   -- 1 High, 2 Normal, 3 Low
    attempts    INT         NOT NULL DEFAULT 0,
    max_attempts INT        NOT NULL DEFAULT 3,
    visible_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX jobs_dequeue ON jobs (priority, created_at)
    WHERE visible_at <= NOW();
```

```sql
-- Dequeue (runs inside a transaction held by the worker)
WITH next AS (
    SELECT id FROM jobs
    WHERE  visible_at <= NOW()
    ORDER  BY priority ASC, created_at ASC
    LIMIT  1
    FOR UPDATE SKIP LOCKED
)
UPDATE jobs
SET    visible_at = NOW() + INTERVAL '30 seconds'
FROM   next
WHERE  jobs.id = next.id
RETURNING jobs.*;
```

`SKIP LOCKED` is the key primitive: it lets many workers run this statement concurrently and each gets a *different* row without blocking each other. Without it, all workers would queue up on the same row lock.

**ACK** — delete the row.
**NACK** — increment attempts; if `attempts >= max_attempts`, move to a `dead_jobs` table and delete from `jobs`; otherwise set `visible_at = NOW()` (immediate retry) or apply exponential back-off.

**Pros:** transactions, SQL joins for monitoring, no extra infra.
**Cons:** write amplification (every dequeue is an UPDATE), table bloat from dead tuples, max ~5–10 k jobs/s before Postgres I/O becomes the bottleneck.

---

## 4. Redis LPUSH / BRPOP pattern

Redis lists are the simplest distributed queue primitive:

```
Producer:   LPUSH queue:high   <job-json>
            LPUSH queue:normal <job-json>
            LPUSH queue:low    <job-json>

Worker:     BRPOP queue:high queue:normal queue:low 0
            -- blocks until an item is available;
            -- BRPOP checks keys left-to-right so queue:high wins
```

`BRPOP` with multiple keys provides **multi-lane priority**: the worker always drains `queue:high` before touching `queue:normal`. This is more robust than sorting inside a single list.

**Visibility timeout** is harder in raw Redis. The standard approach:

1. `BRPOP` the job from the pending list.
2. Atomically `SET job:<id>:inFlight 1 EX 30` (a TTL key acting as the lease).
3. On `Ack`: `LREM` from a separate processing set + `DEL` the TTL key.
4. A separate reaper script (or a Lua script on a cron) finds jobs whose TTL key has expired and re-pushes them.

Purpose-built libraries (Sidekiq, BullMQ, Asynq) handle this bookkeeping with Lua scripts and a `ZSET` sorted by re-queue time so the reaper is O(log n).

**Throughput:** Redis can handle hundreds of thousands of `LPUSH/BRPOP` operations per second on a single node. Use Redis Cluster or stream sharding when you need horizontal scale.

---

## 5. Priority implementation strategies

| Strategy | How it works | Trade-offs |
|---|---|---|
| Multiple queues per priority | `queue:1`, `queue:2`, `queue:3`; BRPOP checks in order | Simple; starvation of low-priority if high is always full |
| Weighted fair queuing | Worker round-robins lanes with weights (e.g., High:Normal:Low = 5:3:1) | Prevents starvation; adds scheduling logic |
| Min-heap / priority queue | Single heap ordered by `(priority, enqueued_at)` | O(log n) insert & extract; requires single-writer or fine-grained locking |
| Delayed priority boost | Jobs that wait too long automatically promoted | Prevents indefinite starvation of Low-priority jobs |

The simulation in this capstone uses sorted-slice insertion (O(n)) which is fine for a teaching example but should be replaced with a heap (`container/heap`) or separate per-priority channels for production use.

---

## 6. Worker autoscaling based on queue depth

A simple feedback controller:

```
every 10 seconds:
  depth = queue.Len()
  in_flight = queue.InFlightLen()

  desired_workers = clamp(
      ceil(depth / target_depth_per_worker),
      min_workers,
      max_workers,
  )

  if desired_workers > current_workers:
      spin up (desired_workers - current_workers) goroutines / containers
  elif desired_workers < current_workers:
      signal (current_workers - desired_workers) workers to drain and exit
```

In a container orchestration layer (Kubernetes) this translates to a `HorizontalPodAutoscaler` driven by a custom metric exported from the queue service (e.g., via Prometheus). The metric is the **queue depth** (pending + in-flight), not CPU, because a queue consumer's CPU may stay low even when the queue is backed up.

**Cooldown periods** prevent thrashing: don't scale down within 3 minutes of the last scale-up. **Scale-up** should be aggressive (double capacity), **scale-down** conservative (reduce by 10% per interval).

For in-process worker pools (like `WorkerPool` in this capstone), a goroutine can watch a `chan struct{}` for shutdown signals, and a supervisor goroutine adjusts pool size by launching or cancelling goroutines based on `atomic.LoadInt64(&stats.Enqueued) - atomic.LoadInt64(&stats.Processed)`.

---

## 7. DLQ alerting strategy

The DLQ is a **canary for systemic failures**. Any growth in DLQ depth should trigger an alert.

**Tiered alerts:**

| Condition | Severity | Action |
|---|---|---|
| DLQ depth > 0 for first time | Warning | Notify on-call; auto-create incident ticket |
| DLQ depth > 10 within 5 min | High | Page on-call immediately |
| DLQ depth growing at > 1/min for 10 min | Critical | Escalate; consider pausing producers |
| Specific job type in DLQ > 5 times | Warning | Tag handler team for review |

**Runbook items for each DLQ alert:**

1. Inspect `dlq.List()` (or query `dead_jobs` table) — what job types failed? What is the error?
2. Check application logs for the handler error at the time of failure.
3. If the failure was transient (downstream API timeout, DB blip): `Requeue` the jobs and monitor.
4. If the failure was a code bug: fix the handler, deploy, then `Requeue`.
5. If the payload itself is corrupt: discard and audit upstream producer.

**Metrics to export:**

- `queue_dlq_depth` — current count of dead-lettered jobs (gauge)
- `queue_dlq_added_total` — lifetime counter of jobs sent to DLQ (counter)
- `queue_dlq_requeued_total` — how often operators replay from DLQ (counter)

Keeping DLQ depth as a Prometheus gauge means Grafana can alert on `rate(queue_dlq_added_total[5m]) > 0` and display a dashboard tile that turns red the moment any job fails permanently.
