# Chapter 80 — Event-Driven Architecture in Go

Event-driven architecture (EDA) decouples services by having them communicate through events rather than direct calls. Producers emit events without knowing who will consume them; consumers react independently.

## Core concepts

**Domain event** — something that happened in the business domain. Immutable, past-tense, named after the fact: `order.placed`, `payment.charged`. Events carry the data needed for downstream processing.

**Transactional outbox** — prevents lost events by writing the event to a local database table in the same transaction as the business change. A relay process polls the outbox and publishes pending events to the message broker.

**Event bus** — routes events from producers to subscribers. In-process buses use handler maps; production buses use message brokers (Kafka, RabbitMQ, SNS/SQS).

**Event sourcing** — stores the full history of state changes as an immutable event log. Current state is rebuilt by replaying events. Enables audit trails, temporal queries, and projection rebuilds.

**CQRS** (Command Query Responsibility Segregation) — separates write (command) and read (query) models. The write side records events; projections listen and build read-optimised models (tables, caches).

**Choreography-based saga** — distributed transaction executed through event chains. Each service reacts to events and emits new ones. Compensation events undo completed steps when a later step fails.

## The transactional outbox pattern

```
Business TX:
  1. Write entity to DB
  2. Write event to outbox table (same TX)

Relay (separate process):
  3. Poll outbox for unpublished events
  4. Publish to message broker
  5. Mark event as published
```

This guarantees at-least-once delivery even if the application crashes between writing and publishing. The relay makes publishing idempotent by checking the `published_at` column.

## Event sourcing

```go
// Write side: append events
store.Append("acc-1", "account.deposited", version+1, DepositedEvent{Amount: 100})

// Read side: rebuild by replaying
acc := &BankAccount{}
for _, e := range store.Load("acc-1") {
    acc.Apply(e)
}
```

Snapshots optimise rebuild performance: take a snapshot at version N, then replay only events after N.

## Choreography-based saga

```
order.placed
  → inventory.reserved (or inventory.failed)
    → payment.charged (or payment.failed)
      → order.confirmed
    ↗ payment.failed
      → order.cancelled
        → inventory released (compensation)
```

Each service subscribes to the events it cares about and publishes the result. No central coordinator; services are decoupled.

## Key invariants

- Events are **immutable** — never update or delete an event; append a correcting event instead.
- Events carry **all required data** — downstream services should not need to call back to get context.
- Relay publishing is **idempotent** — track `published_at`; replay is safe.
- Handlers must be **idempotent** — at-least-once delivery means duplicates will arrive.
- Always **release locks before publishing** — the event bus may invoke callbacks synchronously, causing re-entrant deadlocks if you hold a mutex.

## Examples in this chapter

| File | Topic |
|------|-------|
| `examples/01_outbox_pattern/main.go` | Domain events, outbox relay, fan-out, event versioning |
| `examples/02_event_sourcing/main.go` | Event store, aggregate rebuild, snapshots, CQRS read model |
| `exercises/01_order_saga/main.go` | Choreography saga with compensation across inventory/payment |
