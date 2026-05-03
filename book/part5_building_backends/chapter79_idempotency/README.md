# Chapter 79 — Idempotency at the API Boundary

Idempotency means that calling an operation once produces the same result as calling it N times. This is essential for safe retries: network failures and client timeouts lead to duplicate requests; without idempotency, the server processes each one, potentially charging a customer twice or booking a seat twice.

## Why idempotency matters

HTTP clients and load balancers retry on timeout. The server may have processed the request before the timeout — the client just didn't receive the response. Without idempotency keys, this causes double execution.

## Idempotency key

The client generates a unique key (UUID) per logical operation and includes it in the request (usually as an `Idempotency-Key` header or request field). The server:

1. Checks if it has already processed this key.
2. If yes: returns the stored result (no re-execution).
3. If no: executes, stores the result, returns it.

```go
func (s *IdempStore) Do(key string, fn func() (any, error)) (any, error) {
    s.mu.Lock()
    if result, ok := s.results[key]; ok {
        s.mu.Unlock()
        return result.Response, result.Error  // replay
    }
    // mark inflight, unlock, execute, store, close channel
}
```

## Race-safe first write

When multiple goroutines send the same key simultaneously, only the first executes `fn`. Others wait on a channel and receive the same result when it completes.

```
goroutine A: key not seen → mark inflight → unlock → execute fn
goroutine B: sees inflight → wait on channel → read result from store
goroutine C: sees inflight → wait on channel → read result from store
```

## TTL expiry

Idempotency keys expire after a configured duration (e.g. 24 hours). Expired keys allow re-execution — a new request with the same key gets a fresh result.

## Natural idempotency

Some operations are inherently idempotent:
- `PUT /users/u-1` with a full body — same input, same state, no key needed.
- `DELETE /orders/ord-1` — deleting a non-existent resource is not an error.
- `GET`/`HEAD` — read-only, always idempotent.

## Optimistic locking

For conditional updates, include a version number. The server only applies the change if the version matches, preventing double-application of the same delta.

```go
func (s *AccountStore) Debit(id string, amount, expectedVersion int) (*Account, error) {
    if acc.Version != expectedVersion {
        return nil, fmt.Errorf("version mismatch") // reject duplicate
    }
    // apply
}
```

## Transactional inbox

Consumer-side deduplication: maintain a `processed` set keyed by event ID. If the event ID is already present, skip processing. TTL-clean to bound memory growth.

## Transactional outbox

Producer-side guarantee: write the event to an outbox table in the same transaction as the business change. A relay process publishes unpublished events. Since the relay can restart, marking an event published must also be idempotent.

## Saga with compensation

Long-running distributed transactions that span services are modelled as sagas. Each step is compensatable. If a later step fails, completed steps are undone in reverse order.

## Delivery semantics

| Mode | Guarantee | Implication |
|------|-----------|-------------|
| At-most-once | May lose events | Acceptable for metrics, logs |
| At-least-once | May duplicate events | Consumer must be idempotent |
| Exactly-once | At-least-once + inbox dedup | Highest cost; strongest guarantee |

## Examples in this chapter

| File | Topic |
|------|-------|
| `examples/01_idempotency_keys/main.go` | IdempStore, race-safe first write, PUT semantics, optimistic locking |
| `examples/02_idempotency_patterns/main.go` | Inbox dedup, outbox relay, saga with compensation, delivery semantics |
| `exercises/01_payment_service/main.go` | Payment processor with TTL, concurrent dedup, outbox publishing |
