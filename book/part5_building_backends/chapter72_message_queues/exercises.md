# Chapter 72 Exercises — Message Queues

## Exercise 1 — Order Events (`exercises/01_order_events`)

Build an order lifecycle event system that demonstrates priority queuing, middleware chaining, deduplication, and event sourcing with replay.

### Priority Queue

Implement `PriorityQueue` with constants `PriorityLow=1`, `PriorityNormal=5`, `PriorityHigh=10`:

```go
type PriorityMessage struct {
    ID       string
    Topic    string
    Payload  any
    Priority Priority
    Attempts int
}

func (pq *PriorityQueue) Enqueue(msg *PriorityMessage)
func (pq *PriorityQueue) Dequeue() *PriorityMessage   // highest priority first; nil if empty
func (pq *PriorityQueue) Nack(msg *PriorityMessage)   // retry or DLQ
func (pq *PriorityQueue) DLQ() []*PriorityMessage
```

- `Enqueue` keeps the slice sorted descending by priority
- `Nack` increments `Attempts` and re-enqueues if below `maxRetry`, else appends to DLQ

### Middleware Pipeline

Implement a `HandlerFunc` and `Middleware` type:

```go
type HandlerFunc func(ctx context.Context, e Event) error
type Middleware func(next HandlerFunc) HandlerFunc

func Chain(h HandlerFunc, mws ...Middleware) HandlerFunc
```

Implement these middleware:

- **`LoggingMW`**: prints `[log] handling event id=X topic=Y` before, `[log] event id=X failed: err` on error
- **`RetryMW(maxAttempts int, delay time.Duration) Middleware`**: retries up to maxAttempts on error with delay between attempts; returns wrapped error on final failure
- **`Dedup.Middleware`**: tracks seen event IDs in a map; drops (returns nil) on duplicate; thread-safe

### Event Store

Implement an append-only `EventStore`:

```go
func (es *EventStore) Append(e Event)
func (es *EventStore) All() []Event
func (es *EventStore) Replay(topic string, fn func(Event))  // empty topic = all events
```

### Demonstration

1. **Priority queue**: enqueue 4 messages with mixed priorities; dequeue all and print in priority order
2. **DLQ**: enqueue one message, NACK it until it reaches maxRetry; print DLQ entry
3. **Dedup**: send the same event ID twice; verify base handler called once
4. **Retry**: use a flaky handler that fails first 2 times, succeeds on 3rd; verify success
5. **Event store + replay**: record events through a recording middleware; replay all on a topic

### Hints

- Sort the priority slice after every Enqueue (or use a heap for O(log n))
- `Chain` applies middleware right-to-left so the first middleware in the slice is outermost
- Dedup map must be protected by a mutex — handlers may run concurrently
- `RetryMW` wraps errors with attempt count; the final error should use `fmt.Errorf("all N attempts failed: %w", err)`
- Event store's `Replay` should copy the slice before iterating to avoid holding the lock during callbacks
