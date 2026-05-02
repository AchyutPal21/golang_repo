# Chapter 35 — Service Layer

> **Part III · Designing Software** | Estimated reading time: 22 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

The service layer (application layer) is where use cases live. It is the orchestrator: it loads domain objects, invokes domain logic, persists results, and coordinates side effects. Getting the boundaries right — what belongs in the service vs. the domain vs. the transport layer — is one of the most important skills in Go backend development.

---

## 35.1 — What the service layer does

A service method implements exactly one use case. Its responsibilities, in order:

1. **Validate inputs** — check types, ranges, required fields. This is *not* domain validation.
2. **Load domain objects** — call repositories.
3. **Invoke domain logic** — call methods on domain entities. The service does not contain business rules.
4. **Persist results** — save changed entities back to repositories.
5. **Coordinate side effects** — publish events, write audit logs, send notifications.
6. **Return results** — a DTO or a domain entity, depending on what the transport layer needs.

---

## 35.2 — Input validation vs. domain rules

| | Input validation (service) | Domain rule (entity) |
|---|---|---|
| Where | `req.Validate()` in the service | Domain method (`account.Debit`) |
| What | Empty strings, missing IDs, out-of-range ints | Business invariants (sufficient funds, valid state) |
| Error type | `fmt.Errorf("field X is required")` | Domain sentinel (`ErrInsufficientFunds`) |
| Why | Fail before loading anything from DB | Enforce rule close to data it protects |

---

## 35.3 — Idempotency

Operations that mutate state should accept an `IdempotencyKey`. Before executing, the service checks if the key was already used. If yes, return `ErrDuplicateKey` immediately — no double charges, no double subscriptions:

```go
used, _ := s.idem.Check(req.IdempotencyKey)
if used { return Subscription{}, ErrDuplicateKey }

// ... do the work ...

_ = s.idem.Mark(req.IdempotencyKey)
```

---

## 35.4 — Compensating transactions

Go does not have distributed transactions. When a multi-step operation fails partway through, earlier steps must be compensated manually:

```go
for _, item := range items {
    if err := inventory.Reserve(item); err != nil {
        for _, r := range reserved { inventory.Release(r) } // compensate
        return err
    }
    reserved = append(reserved, item)
}
```

Pattern: collect what was done; on error, undo in reverse order.

---

## 35.5 — Event publishing

After a successful state change, publish a domain event. Events are published *after* all persistence succeeds — never before:

```go
saved, err := s.subs.Save(sub)
if err != nil { return err }
s.events.Publish("subscription.created", payload) // only here
```

Events are fire-and-forget from the service's perspective. Failure to publish is logged, not returned as an error to the caller.

---

## 35.6 — Service boundaries

Services must not:
- Call other services directly (use events or orchestrate via a higher-level coordinator)
- Import infrastructure packages (HTTP clients, SQL drivers)
- Embed business rules — those belong on domain entities

Services may:
- Use multiple repositories
- Call ports (payment gateway, email sender, event bus)
- Coordinate multiple domain operations in one transaction

---

## Running the examples

```bash
cd book/part3_designing_software/chapter35_service_layer

go run ./examples/01_service_patterns  # BankingService: transfer, validation, audit
go run ./examples/02_cross_cutting     # SubscriptionService: idempotency, events, compensation

go run ./exercises/01_checkout_service # CheckoutService: 3-port orchestration with rollback
```

---

## Key takeaways

1. **Six steps** — validate inputs → load → invoke domain logic → persist → side effects → return.
2. **Business rules belong on entities**, not in services.
3. **Idempotency** — check-before-execute + mark-after-succeed prevents duplicate operations.
4. **Compensating transactions** — collect what was done; undo in reverse on failure.
5. **Events after persistence** — never publish before the state change is durable.

---

## Cross-references

- **Chapter 28** — Dependency Injection: all ports are injected via constructor
- **Chapter 30** — Clean Architecture: service layer is the application layer
- **Chapter 34** — Repository Pattern: the primary persistence mechanism used by services
- **Chapter 36** — Error Handling Philosophy: wrapping and sentinel errors in services
