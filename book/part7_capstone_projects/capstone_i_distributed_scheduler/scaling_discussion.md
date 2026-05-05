# Capstone I — Scaling Discussion

## Leader election without etcd

The in-memory cluster state works for a single process. In production, use PostgreSQL advisory locks as a lightweight coordinator:

```sql
-- Acquire leadership (non-blocking)
SELECT pg_try_advisory_lock(12345);   -- returns true if acquired
-- Release
SELECT pg_advisory_unlock(12345);
-- Heartbeat: leader must re-acquire within TTL
-- Followers poll pg_try_advisory_lock every 5s
```

Advantages over etcd for small clusters: no additional infrastructure, Postgres is already your primary data store.

## No-miss guarantee: how it works

On promotion, the new leader queries the job execution log for the last successful run of each job. It computes the gap between `lastRun` and `now` and fires any missed intervals immediately:

```
lastRun = 11:00 (recorded in DB)
now     = 11:10
gap     = 10 minutes

data-cleanup (*/5 *): should have run at 11:05 → fire now
report-gen   (0 *):   not due until 12:00 → no catch-up needed
```

This requires persisting job execution state in the database, not just in memory.

## Exactly-once vs at-least-once

The distributed lock gives **at-least-once** semantics: if the leader crashes after starting a job but before recording completion, the next leader will rerun it.

True **exactly-once** requires idempotent job handlers:
- Generate a unique execution token before running
- Record `(job, minute_slot, token)` atomically with job effects in a single transaction
- On catch-up, skip slots already recorded

## Cron expression edge cases

| Expression | Pitfall |
|------------|---------|
| `0 0 31 2 *` | Feb 31 never fires — implementation must handle this |
| `0 0 29 2 *` | Only fires in leap years |
| `*/60 * * * *` | Only fires at minute 0 — not every hour; use `0 * * * *` |
| `0 0 * * 7` | Sunday = 0 OR 7 depending on implementation |

## Kubernetes deployment

```yaml
scheduler:
  replicas: 3          # 3 nodes, 1 leader at a time
  strategy:
    type: RollingUpdate
    rollingUpdate: {maxUnavailable: 1}
  env:
    - HEARTBEAT_INTERVAL: "5s"
    - LEADER_TIMEOUT:     "15s"   # 3× heartbeat
    - NODE_ID:            valueFrom: {fieldRef: {fieldPath: metadata.name}}
```

Use `metadata.name` (pod name) as the node ID — it's stable within a StatefulSet and unique across pods.

## Time zone handling

All cron schedules should be stored and evaluated in UTC. For user-facing schedules in local time:

```go
loc, _ := time.LoadLocation("America/New_York")
localTime := t.In(loc)
if expr.Matches(localTime) { ... }
```

Store `timezone` alongside the cron expression in the DB. Never hardcode local time in scheduler logic.

## Observability

| Metric | Alert |
|--------|-------|
| `scheduler_leader_changes_total` | > 5/hour → unstable election |
| `scheduler_job_missed_total` | > 0 → catch-up ran; investigate outage |
| `scheduler_job_duration_seconds{job}` | p99 > cron interval → job overruns |
| `scheduler_lock_contention_total` | > 0 → split-brain event occurred |
