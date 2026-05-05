# Chapter 96 — Distributed Building Blocks

Distributed systems require coordination primitives that work across processes and machines: leader election to avoid split-brain, distributed locks for mutual exclusion, and consensus for consistent state changes.

## Topics

| # | Topic | Key ideas |
|---|-------|-----------|
| 1 | Leader Election | Heartbeat-based, epoch fencing, split-brain prevention |
| 2 | Distributed Lock | TTL-based, fencing tokens, lock stealing |
| E | Raft Basics | Leader election, log replication, commit quorum |

## Examples

### `examples/01_leader_election`

Leader election simulation:
- Heartbeat-based leader detection
- Epoch/term numbering for fencing
- Candidate promotion on heartbeat timeout
- Split-brain detection
- Graceful leader handoff

### `examples/02_distributed_lock`

Distributed lock with fencing tokens:
- TTL-based lock expiry
- Fencing token (monotonically increasing version)
- Lock stealing detection
- Deadlock prevention via timeout

### `exercises/01_raft_basics`

Simplified Raft consensus implementation:
- Single-node leader election with term numbers
- Log entry replication simulation
- Quorum calculation
- Commit index advancement

## Key Concepts

**Leader election invariants**
1. At most one leader per epoch (safety)
2. A leader is elected eventually if a majority is reachable (liveness)
3. Old leaders are fenced: they must check their epoch before acting

**Fencing token**
A monotonically increasing token attached to each lock acquisition. Storage systems reject writes with a stale token, preventing split-brain writes.

```
Client A acquires lock → token=42
Network partition
Client A still thinks it holds lock
Client B acquires lock → token=43
Client A writes with token=42 → storage rejects (42 < 43)
```

**Raft in one paragraph**
Raft elects a leader that replicates a log to followers. A log entry is committed when a majority of nodes have written it. Leaders send heartbeats; followers that don't hear from a leader start an election with an incremented term.

## Running

```bash
go run ./book/part6_production_engineering/chapter96_distributed_building_blocks/examples/01_leader_election
go run ./book/part6_production_engineering/chapter96_distributed_building_blocks/examples/02_distributed_lock
go run ./book/part6_production_engineering/chapter96_distributed_building_blocks/exercises/01_raft_basics
```
