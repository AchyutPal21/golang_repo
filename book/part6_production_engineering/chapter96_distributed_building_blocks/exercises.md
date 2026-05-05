# Chapter 96 Exercises — Distributed Building Blocks

## Exercise 1 (provided): Raft Basics

Location: `exercises/01_raft_basics/main.go`

Simplified Raft simulation:
- Node states: Follower, Candidate, Leader
- Term-based leader election with simulated votes
- Log entry append and replication
- Quorum-based commit index advancement
- Leader heartbeat and follower timeout

## Exercise 2 (self-directed): etcd-Style Leader Election

Simulate etcd's leader election using Go channels:
- Each node periodically tries to `PUT /election/leader` with a TTL
- Whoever succeeds becomes leader and sends heartbeats
- Other nodes watch the key and campaign when it disappears
- On network partition: old leader loses TTL, new election occurs
- Verify: only one leader at a time per epoch

## Exercise 3 (self-directed): Distributed Counter

Build a distributed counter simulation:
- N counter nodes each holding a partial count
- `Increment(nodeID string)` — adds 1 to that node's partial count
- `Read() int64` — returns the sum across all nodes (eventual read)
- `StrongRead() int64` — waits for all nodes to sync before reading
- Simulate a "partition" where some increments are lost, then recovered

## Stretch Goal: Two-Phase Commit

Implement a simplified 2PC coordinator:
- `Prepare(txID string, participants []string) bool` — sends prepare, collects votes
- `Commit(txID string, participants []string)` — if all voted yes
- `Abort(txID string, participants []string)` — if any voted no
- Each participant has a `P(abortProbability float64)` that sometimes votes no
- Demonstrate: if one participant crashes after prepare, the coordinator handles it
