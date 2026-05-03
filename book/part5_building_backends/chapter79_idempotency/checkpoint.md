# Chapter 79 Checkpoint — Idempotency

## Concepts to know

- [ ] What is idempotency and why is it essential for safe retries?
- [ ] How does an idempotency key prevent double charges?
- [ ] What race condition occurs when two requests arrive with the same key simultaneously? How do you fix it?
- [ ] What is a TTL on an idempotency key and when should you use it?
- [ ] Which HTTP methods are naturally idempotent and which are not?
- [ ] What is optimistic locking and how does it provide idempotency for conditional updates?
- [ ] What is the transactional inbox pattern? Where is the dedup table?
- [ ] What is the transactional outbox pattern? What happens if the relay crashes and restarts?
- [ ] What is a saga? What is a compensating transaction?
- [ ] Compare at-most-once, at-least-once, and exactly-once delivery semantics.

## Code exercises

### 1. IdempStore with TTL

- Create an `IdempStore` with a 100ms TTL.
- Call `Do("k", fn)` three times in rapid succession — verify `fn` executes once.
- Sleep 150ms, call again — verify `fn` executes a second time (key expired).

### 2. Concurrent idempotency

- Spin up 10 goroutines all calling `store.Do("same-key", fn)` simultaneously.
- Verify `fn` executes exactly once and all goroutines receive the same result.

### 3. Inbox dedup

- Create an `Inbox` with a 1 minute TTL.
- Process events: `["e1", "e2", "e1", "e3", "e2"]`.
- Verify `Applied=3`, `Rejected=2`.

### 4. Outbox relay

- Write three events to an `Outbox`.
- Run the relay once — verify all three are published.
- Run the relay a second time — verify zero are published (already marked).

## Quick reference

```go
// Idempotency store
store := NewIdempStore(24 * time.Hour) // ttl=0 means no expiry
resp, err := store.Do("idemp-key-xyz", func() (any, error) {
    return doExpensiveOperation()
})

// Inbox deduplication
inbox := NewInbox(time.Minute)
if inbox.TryProcess(eventID) {
    // handle event
} else {
    // duplicate, skip
}

// Optimistic locking
acc, err := store.Debit("acc-1", 500, expectedVersion)
// err != nil if version mismatch → safe to retry with new version

// Saga compensation
saga.Add(SagaStep{
    Name:       "charge-card",
    Execute:    func() error { return chargeCard() },
    Compensate: func() error { return refundCard() },
})
if err := saga.Run(); err != nil {
    // completed steps have been compensated in reverse order
}
```

## What to remember

- Store result before returning — if the server crashes after executing but before storing, the next retry re-executes (accept-once, store-before-reply).
- Inflight tracking prevents duplicate execution when concurrent requests race on the same key.
- TTL expiry allows long-lived systems to reuse keys without unbounded memory growth.
- `PUT` is naturally idempotent; `POST` requires an explicit idempotency key; `PATCH` is not idempotent by default.
- Outbox relay must be idempotent: marking published before crashing means the relay skips already-published events on restart.
- Exactly-once = at-least-once delivery + inbox deduplication on the consumer side.
