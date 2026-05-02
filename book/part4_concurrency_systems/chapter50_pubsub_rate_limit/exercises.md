# Chapter 50 — Exercises

## 50.1 — Event bus

Run [`exercises/01_event_bus`](exercises/01_event_bus/main.go).

Event bus with three scenarios:
1. Wildcard topic routing — exact `orders.created`, prefix `orders.*`, and unmatched `inventory.updated`
2. Rate-limited publisher (10 rps) + slow subscriber + dead-letter queue
3. Three subscribers with 1ms / 10ms / 30ms processing speeds; drop counts show the effect of back-pressure

Try:
- Change the subscriber buffer in scenario 3 to 1 and observe how drop counts change.
- Add a `ReplayDLQ(sub *Subscription)` method that feeds all dead-letter events into the subscriber's channel.
- Add a `Metrics() BusMetrics` method that atomically tracks total published, total dropped, and active subscriptions.

## 50.2 ★ — Adaptive rate limiter

Build a rate limiter that adjusts its rate based on downstream feedback:

```go
type AdaptiveLimiter struct { ... }
func (a *AdaptiveLimiter) Allow() bool
func (a *AdaptiveLimiter) RecordSuccess()
func (a *AdaptiveLimiter) RecordFailure() // caller signals downstream overload
```

On `RecordFailure`, halve the allowed rate (down to a minimum). On a streak of `RecordSuccess` calls, gradually increase back toward the original rate. This models the AIMD (Additive Increase / Multiplicative Decrease) algorithm used in TCP congestion control.

## 50.3 ★★ — Ordered pub/sub

Modify the broker so that messages are delivered to each subscriber in the order they were published, even when multiple goroutines call `Publish` concurrently.

Hint: assign a monotonically increasing sequence number to each message inside `Publish` (under the write lock), then buffer and re-order at the subscriber level using a min-heap keyed on sequence number.
