# Chapter 79 Exercises — Idempotency

## Exercise 1 — Payment Service (`exercises/01_payment_service`)

Build an idempotent payment processor that combines: idempotency key deduplication with TTL, race-safe concurrent request handling, outbox publishing, and TTL-based re-execution.

### Components

**`IdempStore`** (with TTL):
```go
func NewIdempStore(ttl time.Duration) *IdempStore
func (s *IdempStore) Do(key string, fn func() (any, error)) (any, error)
func (s *IdempStore) Count() int
```

- TTL=0 means keys never expire.
- When the TTL has elapsed, delete the stored result and re-execute `fn`.
- Concurrent requests on the same inflight key wait and share the result.

**`PaymentProcessor`**:
```go
func NewPaymentProcessor() *PaymentProcessor
func (pp *PaymentProcessor) Charge(ctx context.Context, req ChargeRequest) (*ChargeResponse, error)
func (pp *PaymentProcessor) PublishOutbox() []*OutboxEvent
```

- `Charge` wraps `doCharge` with the idempotency store.
- `doCharge` creates a `Payment` record and appends an `OutboxEvent` atomically.
- `PublishOutbox` marks all unpublished events as published and returns them.

### Behaviour rules

- Calling `Charge` with the same `IdempotencyKey` N times executes the charge exactly once.
- 5 concurrent goroutines with the same key produce exactly 1 charge and all receive the same payment ID.
- After TTL expires, a new call with the same key creates a new payment (intended re-execution).
- `PublishOutbox()` called twice publishes events only on the first call (idempotent relay).

### Demonstration

1. **Basic idempotency**: same key called 3 times → 1 actual charge, same payment ID in all responses
2. **Different keys**: 2 new keys → 2 additional charges (total=3)
3. **Concurrent safety**: 5 goroutines on same key → exactly 1 charge, 1 unique payment ID
4. **TTL expiry**: 50ms TTL, wait 60ms, retry → new charge created (total=2 for this processor)
5. **Outbox publishing**: run relay once → 3 events published; run again → 0 published

### Hints

- Track inflight keys with `map[string]chan struct{}` — first caller creates the channel, others wait on it
- Close the channel **after** storing the result to avoid a race where a waiter reads before the result is stored
- For TTL check: `time.Since(r.At) > s.ttl` — delete the entry and fall through to re-execute
- The `Cached` field in `ChargeResponse` is informational; the key invariant is that `Charges` counter stays at 1 for repeated calls with the same key
