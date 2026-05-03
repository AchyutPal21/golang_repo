# Chapter 72 — Message Queues

## What you'll learn

How to build reliable in-memory message queues and event buses in Go: at-least-once delivery with ack/nack, dead-letter queues, worker pools, fan-out pub/sub, async dispatch, priority queues, deduplication middleware, and append-only event stores for replay.

## Key concepts

| Concept | Description | Use case |
|---|---|---|
| Message queue | Buffered channel + inflight map | Task distribution, async jobs |
| Ack / Nack | Explicit success / failure signal | At-least-once delivery |
| Dead-letter queue | Messages that exceed maxRetries | Poison pill isolation |
| Worker pool | N goroutines draining one queue | Parallel task processing |
| Pub/Sub bus | Topic → []handlers fan-out | Domain event propagation |
| Unsubscribe func | Returned by Subscribe | Dynamic listener management |
| Async dispatch | Goroutine per subscriber + WaitGroup | Concurrent event handling |
| Priority queue | Sorted slice, highest priority first | Fraud alerts before analytics |
| Middleware chain | Wrapping HandlerFunc | Logging, retry, dedup, tracing |
| Event store | Append-only log with Replay | Event sourcing, audit trail |

## Files

| File | Topic |
|---|---|
| `examples/01_queue_patterns/main.go` | Queue, Ack/Nack, DLQ, worker pool, at-least-once delivery |
| `examples/02_pub_sub/main.go` | EventBus, Subscribe/Publish, fan-out, unsubscribe, async |
| `exercises/01_order_events/main.go` | Priority queue, DLQ, middleware pipeline, dedup, event store |

## Queue patterns

### Basic publish / receive / ack

```go
q := NewQueue("orders", 100, 3) // name, size, maxRetries

q.Publish(&Message{ID: "m-1", Payload: []byte(`{}`)})

msg, _ := q.Receive(ctx)
if err := process(msg); err != nil {
    q.Nack(msg.ID) // retry or DLQ
} else {
    q.Ack(msg.ID)  // remove from inflight
}
```

### Dead-letter queue

```go
// After maxRetries NACKs, message moves to DLQ channel.
dlqMsg := <-q.DLQ()
log.Printf("poisoned: %s after %d attempts", dlqMsg.ID, dlqMsg.Attempts)
```

### Worker pool

```go
pool := NewWorkerPool(q, 5, func(msg *Message) error {
    return processOrder(msg)
})
pool.Start(ctx, &wg)
wg.Wait()
```

## Pub/Sub patterns

### Subscribe and publish

```go
bus := NewEventBus()

unsub := bus.Subscribe("orders.created", func(e Event) {
    fmt.Println("got order:", e.ID)
})
defer unsub()

bus.Publish(Event{ID: "e-1", Topic: "orders.created", Payload: order})
```

### Fan-out to multiple services

```go
bus.Subscribe("orders.created", inventoryHandler)
bus.Subscribe("orders.created", emailHandler)
bus.Subscribe("orders.created", analyticsHandler)

// All three receive every publish.
bus.Publish(Event{Topic: "orders.created", Payload: order})
```

### Async dispatch

```go
// PublishAsync spawns one goroutine per subscriber, waits for all.
bus.PublishAsync(ctx, Event{Topic: "work", Payload: job})
```

## Middleware pipeline

```go
base := func(ctx context.Context, e Event) error { ... }

dedup := NewDedup()
handler := Chain(base,
    LoggingMW,
    dedup.Middleware,
    RetryMW(3, 10*time.Millisecond),
)

handler(ctx, event)
```

## At-least-once delivery guarantees

- Message stays in `inflight` map until explicitly Acked or Nacked
- If a worker crashes without Acking, message can be redelivered on restart
- Consumers must be **idempotent**: deduplicate by `msg.ID`
- Exactly-once requires a distributed dedup store (Redis SET NX) or distributed transaction

## Production notes

- Replace the in-memory channel with Redis BRPOP / Streams or Kafka for persistence
- Set `maxRetries` per topic: high for critical payments, low for analytics
- DLQ messages should be inspected and replayed; never silently discard
- Use `context.WithTimeout` on Receive to prevent goroutine leaks on shutdown
- For pub/sub fan-out, consider backpressure: slow subscribers block sync Publish
