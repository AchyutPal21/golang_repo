# Chapter 35 — Revision Checkpoint

## Questions

1. List the six responsibilities of a service method in order.
2. What is the difference between input validation and domain validation? Give one example of each.
3. Why must domain events be published *after* persistence, not before?
4. How does an idempotency key prevent double-charges?
5. What is a compensating transaction and when is it needed?

## Answers

1. Six responsibilities in order:
   1. Validate inputs (missing fields, out-of-range values)
   2. Load domain objects from repositories
   3. Invoke domain logic (call entity methods; they enforce business rules)
   4. Persist results (save changed entities back to repositories)
   5. Coordinate side effects (events, audit logs, notifications)
   6. Return results to the caller

2. **Input validation** (service): checks that the data received from the transport
   layer is structurally correct before any DB call is made. Example: "AmountCents
   must be positive." This does not require loading any domain object.
   **Domain validation** (entity): enforces business invariants on the state of
   domain objects. Example: `account.Debit` returns `ErrInsufficientFunds` if
   `balance - amount < 0`. This rule is defined on the entity because the entity
   owns the data it protects.

3. If the event is published before persistence and then the `Save` call fails, the
   event describes a state change that never happened. Consumers (email senders,
   analytics pipelines, downstream services) would act on a lie. Publishing after a
   successful save ensures the event faithfully represents durable state. The
   tradeoff is that a crash between `Save` succeeding and `Publish` being called can
   cause a missed event — the accepted fix is an outbox pattern or at-least-once
   delivery, not publishing before persistence.

4. Before executing a mutating operation, the service calls `idem.Check(key)`. If
   the key was already marked (from a previous successful execution), the service
   returns `ErrDuplicateKey` immediately — no payment is attempted. Only after all
   steps succeed does the service call `idem.Mark(key)`. A client that retries (e.g.,
   due to a network timeout) sends the same key; the service recognises it and
   refuses the duplicate. Even if the first request timed out *after* charging but
   *before* responding, the key is already marked and the retry is blocked.

5. A compensating transaction is an operation that undoes the effect of a previously
   completed step when a later step in a multi-step operation fails. Go has no
   distributed transaction support, so compensation must be explicit. Pattern: collect
   each completed step in a slice; on error, iterate the slice in reverse and call the
   compensating action for each (release reserved stock, refund a charge, delete a
   created record). Compensation is not always perfectly reversible — a refund takes
   time, and a sent email cannot be unsent — but it is the best available mechanism
   for maintaining consistency across independent systems.
