# Chapter 50 — Pub/Sub, Rate Limit, Throttle

## What you will learn

- In-process pub/sub broker: subscribe, publish, unsubscribe
- Topic-based routing — exact match and wildcard prefix patterns
- Non-blocking publish with configurable subscriber buffer sizes
- Drop-on-slow-subscriber vs block-on-slow-subscriber trade-offs
- Dead-letter queue (DLQ) for dropped messages
- Token-bucket rate limiter: burst capacity + steady refill rate
- Sliding-window rate limiter: max N requests per time window
- Throttle (leading-edge debounce): at most once per interval
- Leaky bucket: constant-rate output from a buffered queue

---

## Pub/Sub: the broker pattern

```
Publisher ──publish(topic, msg)──► Broker ──fan-out──► [sub-1 chan]
                                          ──fan-out──► [sub-2 chan]
```

Each subscriber gets its own buffered channel. The broker holds a `sync.RWMutex`-guarded map from topic to `[]chan Message`.

**Non-blocking fan-out** (drop-on-slow):

```go
select {
case ch <- msg:
default:
    // subscriber too slow — drop the message
}
```

**Blocking fan-out** (backpressure): remove the `default` branch — publishers block until all slow subscribers catch up.

---

## Token bucket

Allows short bursts (up to `cap` tokens) then enforces a steady `rate` tokens/second:

```go
tokens += elapsed * rate       // refill
if tokens > cap { tokens = cap }
if tokens < 1 { return false } // deny
tokens--
return true                    // allow
```

Use when you want to permit occasional spikes while keeping the long-run average bounded.

---

## Sliding window

Keeps a timestamp list of recent requests; evicts entries older than the window:

```go
cutoff := now.Add(-window)
evict everything before cutoff
if len(timestamps) >= limit { return false }
timestamps = append(timestamps, now)
return true
```

More precise than fixed windows at boundary edges; slightly higher memory cost.

---

## Throttle (leading-edge)

```go
if time.Since(lastRun) >= interval {
    lastRun = time.Now()
    fn()
}
```

The first call in any interval fires immediately; subsequent calls within the interval are silently dropped.

---

## Leaky bucket

A buffered channel `in` accepts arriving requests; a ticker-driven drainer reads one item per tick and forwards to `out`. Overflow (full `in` channel) is discarded. Unlike the token bucket, the output rate is fixed — it cannot burst.

---

## Choosing a rate limiter

| Scenario | Use |
|---|---|
| Allow occasional bursts, cap average | Token bucket |
| Hard "no more than N per window" | Sliding window |
| UI event handler, cache invalidation | Throttle |
| Constant output rate to a downstream | Leaky bucket |

---

## Examples

| File | Demonstrates |
|---|---|
| `examples/01_pubsub/main.go` | Broker, subscribe/publish/unsubscribe, drop-on-slow, context-aware sub |
| `examples/02_rate_limiter/main.go` | Token bucket, sliding window, throttle, leaky bucket |

## Exercise

`exercises/01_event_bus/main.go` — event bus with wildcard topics, rate-limited publisher, DLQ, per-subscriber drop stats.
