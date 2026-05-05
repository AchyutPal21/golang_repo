# Scaling Discussion — Real-Time Chat

This document walks through every major scaling constraint you will hit when
taking the in-process hub from Capstone D to production, and the standard
engineering response to each one.

---

## 1. WebSocket Connection Limits Per Process

A single Linux process can hold roughly **65 535 simultaneous WebSocket
connections** before it runs into the default OS file-descriptor limit (`ulimit
-n`).  That ceiling can be raised (`/etc/security/limits.conf`, `fs.file-max`
sysctl) to a few hundred thousand, but each goroutine pair (reader + writer per
connection) consumes ~8–16 KB of stack at minimum.  A 64-core host with 128 GB
RAM can realistically sustain **200–400 K concurrent connections** before memory
pressure causes goroutine scheduling latency to spike.

Practical planning numbers:

| Deployment unit       | Realistic connection ceiling |
|-----------------------|------------------------------|
| Single process        | ~50 K (safe default)         |
| Tuned single process  | ~200 K                       |
| 10-instance cluster   | ~500 K–2 M                   |

**Consequence**: any architecture that stores subscriber channels purely
in-process (like `capstone_d`) cannot span multiple OS processes.  The rest of
this document describes how to break that coupling.

---

## 2. Redis Pub/Sub for Multi-Instance Fan-Out

When you deploy N chat servers behind a load balancer, user A (connected to
server 1) must be able to message user B (connected to server 3).

**Solution: use Redis as the shared message bus.**

```
 server-1              server-2              server-3
 ┌─────────┐           ┌─────────┐           ┌─────────┐
 │ Hub     │           │ Hub     │           │ Hub     │
 │ PUBLISH ├──────────►│ SUB     │           │ SUB     │
 └─────────┘  Redis    └─────────┘           └─────────┘
              channel
              "room:general"
```

Each server **subscribes** to every room channel it has active users for.
When a message arrives on a channel, the receiving server fans it out locally
to its own in-memory subscribers.

**Go pattern (stdlib `net` + Redis RESP protocol, or a thin wrapper):**

```go
// Publish — called by the server that received the WebSocket frame
rdb.Publish(ctx, "room:"+roomID, marshalMessage(msg))

// Subscribe — background goroutine on each server
pubsub := rdb.Subscribe(ctx, "room:"+roomID)
for msg := range pubsub.Channel() {
    hub.BroadcastLocal(unmarshalMessage(msg.Payload))
}
```

**Trade-offs:**

| Aspect               | Notes                                                         |
|----------------------|---------------------------------------------------------------|
| Latency              | Adds ~0.5–2 ms round-trip through Redis (usually acceptable) |
| Ordering             | Redis delivers to each subscriber in publish order            |
| At-most-once         | Redis pub/sub does not persist — if a subscriber is down      |
|                      | during publish, the message is lost                           |
| Redis cluster        | Use a single shard for pub/sub to preserve channel semantics  |

---

## 3. Sticky Sessions vs Redis

WebSocket connections are long-lived.  A standard L7 load balancer can route
the HTTP upgrade request to any backend — after that the TCP connection is
pinned.  Two strategies exist:

### 3a. Sticky Sessions (IP hash or cookie affinity)

The load balancer routes all requests from a given client to the **same
backend** for the lifetime of the connection.

**Pros:** Hub logic stays 100 % in-process; no cross-server coordination
overhead for most traffic.

**Cons:**
- A single server crash drops all its sticky connections simultaneously
  (thundering herd on reconnect).
- Uneven load if some rooms are much larger than others (one server
  gets 90 % of traffic).
- Does not work behind a multi-layer proxy that changes source IP.

### 3b. Stateless servers + Redis fan-out (recommended for > 10 K users)

No affinity needed.  Any server handles any client.  Cross-server delivery
uses Redis pub/sub (Section 2).

**Pros:** Uniform load distribution; zero-downtime rolling deploys; any
server can serve reconnecting clients.

**Cons:** Every published message travels through Redis even when sender
and receiver are on the same server (mitigated with a local short-circuit
check before publishing).

**Rule of thumb**: use sticky sessions up to ~50 K users per cluster;
switch to Redis fan-out above that.

---

## 4. Message Persistence — Postgres with COPY for Bulk Insert

Chat messages must survive server restarts and support features like
search, moderation, and audit logs.

### Why not `INSERT` row by row?

A busy room can generate 1 000 messages per second.  Individual `INSERT`
statements each pay ~0.5 ms round-trip + WAL flush overhead — that is ~500
QPS maximum on a single connection, easily saturated.

### Batch with `COPY`

Postgres `COPY` can stream thousands of rows per second over a single
connection with minimal per-row overhead.

```go
// Accumulate messages in a ring buffer, flush every 100 ms or 500 messages.
func (s *Store) FlushBatch(ctx context.Context, msgs []Message) error {
    _, err := s.db.CopyFrom(
        ctx,
        pgx.Identifier{"messages"},
        []string{"id", "room_id", "user_id", "body", "created_at"},
        pgx.CopyFromSlice(len(msgs), func(i int) ([]any, error) {
            m := msgs[i]
            return []any{m.ID, m.RoomID, m.UserID, m.Body, m.Timestamp}, nil
        }),
    )
    return err
}
```

**Schema recommendation:**

```sql
CREATE TABLE messages (
    id         TEXT        PRIMARY KEY,
    room_id    TEXT        NOT NULL,
    user_id    TEXT        NOT NULL,
    body       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
) PARTITION BY LIST (room_id);
```

Partition by `room_id` so that fetching a room's history hits a single
partition.  Add a `BRIN` index on `created_at` within each partition for
efficient time-range queries.

---

## 5. Presence with Redis EXPIRE

The in-process `PresenceTracker` in Capstone D loses all state on restart.
In production, use Redis keys with a TTL:

```
SET presence:alice 1 EX 30        # mark alice online, TTL = 30 s
```

Each client sends a **heartbeat** (WebSocket ping or application-level
keepalive) every 15 seconds.  The server resets the TTL on each heartbeat.
If the TTL expires (no heartbeat for 30 s), Redis automatically removes the
key — the user is implicitly offline.

**Checking presence:**

```go
func IsOnline(ctx context.Context, rdb *redis.Client, userID string) bool {
    val, err := rdb.Exists(ctx, "presence:"+userID).Result()
    return err == nil && val == 1
}
```

**Presence fan-out:** when a user comes online or goes offline, publish a
`presence` event to a Redis channel so all servers can update their local
caches and notify rooms the user belongs to.

---

## 6. Horizontal Scaling with a Fan-Out Relay

For very large deployments (millions of connections), a two-tier architecture
reduces load on the pub/sub bus:

```
                     ┌─────────────┐
                     │  Fan-out    │
                     │  Relay      │  ◄── subscribes to Redis once per room
                     │  (1 process)│
                     └──────┬──────┘
              ┌─────────────┼─────────────┐
              ▼             ▼             ▼
         server-1       server-2       server-3
         (~50 K WS)    (~50 K WS)    (~50 K WS)
```

The **Fan-out Relay** holds a single Redis subscription per active room
and re-broadcasts to the edge servers over an internal gRPC stream or
Unix socket.  This prevents N server processes each opening their own
Redis subscription for the same room (which is wasteful at high room
counts).

Each edge server only subscribes to rooms that have at least one local
user, further reducing fan-out overhead.

---

## 7. Kubernetes StatefulSet vs Deployment for WebSocket Servers

### Deployment (standard)

Kubernetes `Deployment` with `RollingUpdate` strategy works well when:
- Servers are stateless (Redis fan-out model, Section 3b).
- Sessions can be gracefully drained before a pod is replaced.
- You set `terminationGracePeriodSeconds` long enough (e.g. 60 s) to let
  existing connections finish.

```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxUnavailable: 0
    maxSurge: 1
```

### StatefulSet (sticky-session model)

A `StatefulSet` gives each pod a **stable network identity**
(`chat-0.chat-svc`, `chat-1.chat-svc`, …) which lets the load balancer
implement sticky routing by pod name rather than IP (IPs change on
rescheduling).  Use this when:

- You are using the sticky-session model (Section 3a) and need a stable
  hostname to encode in the session cookie.
- Room state is partially in-process and sharded by consistent hashing
  over pod names.

**Trade-off summary:**

| Criterion              | Deployment          | StatefulSet               |
|------------------------|---------------------|---------------------------|
| Pod identity           | Ephemeral           | Stable (`name-N`)         |
| Rolling update         | Automatic           | Manual order control      |
| Stateless fan-out      | Preferred           | Works but adds complexity |
| Sticky-session sharding| Awkward             | Natural                   |
| Scale-down safety      | Immediate           | Ordered (highest N first) |

**Recommendation for most teams**: start with a `Deployment` + Redis
fan-out.  Migrate to `StatefulSet` only if you need consistent hashing
for room affinity at scale (>1 M concurrent users per cluster).

---

## Summary Checklist for Production

- [ ] Raise `ulimit -n` and `fs.file-max` on every chat server host.
- [ ] Use Redis pub/sub for cross-server message delivery.
- [ ] Use Redis `EXPIRE` keys for presence; clients send keepalive heartbeats.
- [ ] Batch-insert messages with Postgres `COPY`; flush every 100 ms or 500 rows.
- [ ] Add a fan-out relay tier once active room count exceeds ~10 K.
- [ ] Choose `Deployment` + stateless routing for simplicity; `StatefulSet` only
      for consistent-hashing sharding scenarios.
- [ ] Set `terminationGracePeriodSeconds` generously to drain WebSocket
      connections before pod replacement.
