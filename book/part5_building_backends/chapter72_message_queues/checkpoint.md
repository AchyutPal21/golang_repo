# Chapter 72 Checkpoint — Message Queues

## Self-assessment questions

1. What is the purpose of the "inflight" map in an at-least-once queue, and what happens if a consumer crashes without calling Ack?
2. How does a dead-letter queue differ from simply discarding a message after maxRetries?
3. What is the difference between synchronous `Publish` and `PublishAsync` in a pub/sub bus? When would you choose each?
4. Why must consumers be idempotent when using at-least-once delivery? Give a concrete example of what breaks without idempotency.
5. How does the unsubscribe function returned by `Subscribe` work internally? Why is it returned as a closure rather than a separate `Unsubscribe(id)` method?
6. How do you prevent a slow subscriber from blocking the entire fan-out in a synchronous pub/sub system?

## Checklist

- [ ] Can implement an in-memory queue with Publish, Receive, Ack, and Nack
- [ ] Can implement retry logic with a max-retry counter and DLQ routing
- [ ] Can build a worker pool that drains a queue with N concurrent goroutines
- [ ] Can implement a pub/sub EventBus with topic subscriptions and fan-out
- [ ] Can implement Subscribe returning an unsubscribe closure
- [ ] Can implement async dispatch (one goroutine per subscriber, WaitGroup)
- [ ] Can build a priority queue that dequeues by highest priority first
- [ ] Can build a middleware chain (logging, retry, dedup) wrapping a HandlerFunc
- [ ] Can implement a Dedup middleware that drops duplicate event IDs
- [ ] Can implement an append-only EventStore with Replay over a topic

## Answers

1. The inflight map tracks messages currently being processed. If a consumer crashes (panics, loses connection) before calling Ack, the message remains in the inflight map. On consumer restart or queue drain, that message must be requeued and redelivered — this guarantees no message is silently dropped. Without the inflight map, a crashed worker would simply lose the message.

2. A DLQ preserves the message and its metadata (ID, attempts, payload) for later inspection, alerting, and manual replay. Discarding silently makes debugging impossible — you lose the message and never know processing failed. DLQ lets operators investigate why a message failed and fix the root cause before replaying.

3. Synchronous `Publish` calls each handler in the same goroutine, in series — callers block until all handlers return, and output order is deterministic. `PublishAsync` spawns one goroutine per subscriber and waits for all via WaitGroup — handlers run concurrently and may finish in any order. Use sync for ordered, auditable flows; async when handlers are independent and latency matters.

4. Without idempotency, a message processed twice causes double side effects: two charge attempts, two emails sent, two inventory deductions. An idempotent consumer checks whether it already processed `msg.ID` (e.g., via a Redis SET NX or DB unique constraint) and skips if so. Example: if a payment service NACKs after charging but before Acking, at-least-once redelivery would charge the customer twice.

5. The closure captures the subscription ID at Subscribe time via a local variable. When called, it acquires the bus mutex and removes the entry with that ID from the slice. Using a closure avoids exposing internal subscription IDs in the public API — callers simply hold a `func()` and call it; the bus handles cleanup internally.

6. Buffer each subscriber's events via a per-subscriber channel (buffered): the Publish path sends to the channel non-blocking, dropping the event if the buffer is full (lossy) or blocking the publisher (backpressure). Alternatively, use a timeout on the channel send and log/DLQ dropped events. In practice, slow subscribers are best separated into their own worker pool with a dedicated queue.
