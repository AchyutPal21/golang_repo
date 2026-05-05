# Capstone I — Distributed Scheduler

A distributed cron scheduler with leader election, no-miss guarantees, and at-least-once job execution — all without an external coordinator like ZooKeeper or etcd.

## What you build

- `Schedule(spec, handler)` — register a cron job (e.g. `"*/5 * * * *"`)
- Leader election: only the leader node runs jobs; followers watch and take over on leader failure
- No-miss guarantee: on leader failover, any job that was due during the gap fires immediately
- Exactly-once semantics: distributed lock prevents two nodes running the same job simultaneously
- Job history: last N executions per job (start time, duration, success/error)
- Heartbeat: leader publishes a heartbeat; followers promote if heartbeat expires

## Architecture

```
Node A (leader)            Node B (follower)         Node C (follower)
  │                          │                          │
  ├─ Scheduler loop          ├─ Watch heartbeat         ├─ Watch heartbeat
  ├─ Heartbeat publisher     └─ Promote on timeout      └─ Promote on timeout
  └─ Job executor
       │
       └─ DistributedLock.Acquire(jobID)
             ├─ success → run handler
             └─ already held → skip (another node got it)
```

## Key components

| Component | Pattern | Chapter ref |
|-----------|---------|-------------|
| Leader election | Epoch-based, quorum | Ch 96 |
| Distributed lock | TTL-based, fencing token | Ch 96 |
| Cron parser | Minute/hour/day/month/weekday fields | Ch 13 |
| No-miss catchup | Missed-intervals scan on startup | Ch 44 |
| Heartbeat | Goroutine + context cancel | Ch 47 |
| Job history | Ring buffer per job | Ch 18 |

## Running

```bash
go run ./book/part7_capstone_projects/capstone_i_distributed_scheduler
```

## What this capstone tests

- Can you implement leader election without etcd or Consul?
- Can you write a cron expression parser from scratch?
- Can you guarantee a job runs at least once across a failover?
- Can you prevent two nodes from executing the same job simultaneously?
