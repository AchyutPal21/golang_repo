# Chapter 80 Checkpoint — Event-Driven Architecture

## Concepts to know

- [ ] What is a domain event, and why is it immutable and past-tense?
- [ ] Explain the transactional outbox pattern. What problem does it solve?
- [ ] Why can't you just publish an event directly from within a DB transaction?
- [ ] What is the difference between event sourcing and a traditional state store?
- [ ] What is a snapshot in event sourcing, and when is it worth taking one?
- [ ] What is CQRS? What are the consistency trade-offs?
- [ ] Explain choreography-based vs orchestration-based sagas.
- [ ] What is a compensating transaction? Give an example.
- [ ] How do you handle event schema evolution (3 strategies)?
- [ ] Why must event handlers be idempotent in an at-least-once system?

## Code exercises

### 1. Outbox relay

Given:
```go
outbox := &Outbox{}
bus := NewEventBus()
```

- Write an `Order.Cancel(reason string)` method that appends an `order.cancelled` event.
- Subscribe a handler that prints the cancellation reason.
- Run the relay and verify the handler is called.

### 2. Event sourcing rebuild

Starting from an empty `EventStore`:
- Append `account.opened` (balance=500), `account.deposited` (200), `account.withdrawn` (100).
- Rebuild the account by replaying all three events.
- Take a snapshot at version 3, then append `account.deposited` (50).
- Rebuild from snapshot and verify balance=650.

### 3. Saga compensation

- Set up inventory (5 items), payment (card always declined), and order services wired via an `EventBus`.
- Place an order for 3 items.
- Verify: inventory was first reserved then released; order status is cancelled.

## Quick reference

```go
// Outbox: write with the business change, publish separately
outbox.Append("order.placed", aggregateID, payload)
// Relay: poll and publish
for _, e := range outbox.Pending() {
    bus.Publish(ctx, e)
    outbox.MarkPublished(e.EventID)
}

// Event sourcing: rebuild
for _, e := range store.Load(aggregateID) {
    aggregate.Apply(e)
}

// Snapshot + delta
snapCopy := *rebuilt
snapshots.Save(&Snapshot{Version: rebuilt.Version, State: &snapCopy})
// Later:
acc := *snap.State
for _, e := range store.LoadFrom(id, snap.Version) { acc.Apply(e) }

// Choreography saga: publish outside locks
s.mu.Lock(); s.orders[id].Status = Confirmed; s.mu.Unlock()
bus.Publish(ctx, "order.confirmed", id, payload) // outside lock
```

## What to remember

- Transactional outbox = write event in the same DB TX as the entity, publish separately.
- Relay is idempotent: marking published prevents duplicate delivery on restart.
- Event sourcing: state is derived — never update events, only append.
- Snapshots reduce replay cost: rebuild from snapshot + events since snapshot version.
- CQRS read models are eventually consistent with the write side.
- Choreography sagas: each service reacts to events, emits results; compensation triggered by failure events.
- **Always publish outside locks** to avoid re-entrant mutex deadlocks.
