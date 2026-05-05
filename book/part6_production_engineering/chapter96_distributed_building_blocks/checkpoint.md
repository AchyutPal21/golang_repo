# Chapter 96 Checkpoint — Distributed Building Blocks

## Concepts to know

- [ ] What is the split-brain problem in leader election?
- [ ] What is a fencing token? How does it prevent split-brain writes?
- [ ] What is an epoch or term in distributed consensus? Why does it increment?
- [ ] What is the Raft consensus algorithm? Name its three sub-problems.
- [ ] What is a quorum? Why does Raft require a majority?
- [ ] What is the difference between a distributed lock and a mutex?
- [ ] What is lock expiry (TTL)? What problem does it solve?
- [ ] Name two production systems that implement distributed locks.
- [ ] What is the difference between CP and AP in the CAP theorem?

## Code exercises

### 1. Quorum calculator

Write `quorum(n int) int` that returns the minimum number of nodes needed for a majority quorum (n/2 + 1). Write `canCommit(n, votes int) bool`.

### 2. Fencing token checker

Write a `FencingStore` that:
- Tracks a `currentToken int64`
- `Write(token int64, data string) error` — rejects if `token < currentToken`
- `CurrentToken() int64`

### 3. Heartbeat detector

Write a `HeartbeatMonitor` that:
- Receives heartbeats via `Beat(nodeID string)`
- Returns `Alive(nodeID string, timeout time.Duration) bool`

## Quick reference

```
# Production distributed lock implementations
etcd:  lease-based, strongly consistent (Raft-backed)
Redis: SETNX + TTL (not strictly correct without Redlock)
ZooKeeper: ephemeral nodes

# Raft log safety invariant
A log entry is committed only when stored on a majority of nodes.
Committed entries are never overridden.

# Common distributed primitives
Leader election: etcd elections API, Consul sessions
Distributed lock: etcd, Redis (Redlock), Postgres advisory locks
Distributed counter: Redis INCR, DynamoDB atomic updates
Distributed queue: Kafka, SQS, NATS JetStream
```

## Expected answers

1. Split-brain: two nodes each believe they are the leader simultaneously, making conflicting decisions.
2. A fencing token is a monotonically increasing number given to each lock holder. Storage rejects writes with a lower token than it last saw.
3. Epoch/term is an integer that increases on every new election. It allows nodes to reject messages from stale leaders.
4. Raft solves: leader election, log replication, safety (committed entries never overwritten).
5. Quorum = majority (n/2 + 1). Requires majority so that any two quorums always overlap, preventing conflicting commits.
6. A mutex serializes goroutines within one process; a distributed lock serializes across processes/machines.
7. TTL (time-to-live) automatically releases a lock when the holder crashes, preventing deadlock from dead processes.
8. etcd (lease-based), Redis with Redlock.
9. CP: consistent and partition-tolerant (may be unavailable). AP: available and partition-tolerant (may return stale data). etcd is CP; DynamoDB is AP.
