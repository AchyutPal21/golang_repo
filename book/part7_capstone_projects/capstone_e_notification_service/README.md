# Capstone E — Multi-Channel Notification Service

## What It Builds

A production-grade notification dispatcher that delivers messages across **email**,
**SMS**, and **push** channels. The service handles the full life-cycle of a
notification: routing, multi-attempt retry with exponential backoff, automatic
provider failover, dead-letter queueing for exhausted messages, and per-channel
delivery tracking — all without any external dependencies.

---

## Architecture Diagram

```
                         ┌──────────────────────────────────┐
                         │       NotificationDispatcher      │
                         │                                   │
  Notification ─────────►  Route by Channel                 │
                         │         │                         │
                         │         ├─── Email ───────────────┼──► ProviderWithFallback
                         │         │                         │       ├─ EmailProvider (primary)
                         │         │                         │       └─ SMSProvider   (fallback)
                         │         │                         │
                         │         ├─── SMS ─────────────────┼──► ProviderWithFallback
                         │         │                         │       ├─ SMSProvider  (primary)
                         │         │                         │       └─ EmailProvider (fallback)
                         │         │                         │
                         │         └─── Push ────────────────┼──► ProviderWithFallback
                         │                                   │       ├─ PushProvider  (primary)
                         │                                   │       └─ SMSProvider   (fallback)
                         │                                   │
                         │   RetryPolicy (exp. backoff)      │
                         │     MaxAttempts / BaseDelay /     │
                         │     Multiplier                    │
                         │                                   │
                         │   DeadLetterQueue  ◄──────────────┼── exhausted notifications
                         │                                   │
                         │   DeliveryTracker  ◄──────────────┼── sent / failed / dlq
                         └──────────────────────────────────┘
```

**Flow per notification**

1. Dispatcher looks up the `ProviderWithFallback` registered for the channel.
2. The retry loop calls the fallback provider up to `RetryPolicy.MaxAttempts` times.
3. Each successive attempt waits `BaseDelay × Multiplier^(attempt-1)` (simulated
   with counters so the binary runs instantly).
4. If every attempt fails the notification is pushed to the `DeadLetterQueue` and
   the tracker records a DLQ hit.
5. A single success at any attempt records a "sent" counter increment and breaks
   the loop.

---

## Key Components

| Component | Role | Chapter refs |
|---|---|---|
| `Notification` | Value type carrying channel, priority, subject/body | Ch 3 (structs) |
| `NotificationProvider` | Interface: `Send(Notification) error` | Ch 5 (interfaces) |
| `EmailProvider` / `SMSProvider` / `PushProvider` | Simulated providers with configurable failure rate | Ch 5, Ch 9 (testing with fakes) |
| `ProviderWithFallback` | Wraps two providers; delegates to secondary on error | Ch 5 (composition) |
| `RetryPolicy` | Exponential-backoff metadata; `NextDelay(attempt)` | Ch 7 (error handling) |
| `DeadLetterQueue` | Thread-safe slice; `Push` / `Drain` | Ch 8 (sync) |
| `DeliveryTracker` | `sync/atomic` counters per channel | Ch 8 (sync), Ch 66 (observability) |
| `NotificationDispatcher` | Orchestrates routing + retry + DLQ | Ch 10 (concurrency patterns) |

---

## Running

```bash
# From the repo root
go run ./part7_capstone_projects/capstone_e_notification_service

# Or build and run
go build ./part7_capstone_projects/capstone_e_notification_service
./capstone_e_notification_service
```

Expected output includes a per-notification delivery log, a channel-level delivery
summary, and a dump of every notification that landed in the dead-letter queue.

---

## What It Tests

| Capability | How it is exercised |
|---|---|
| Interface-based provider abstraction | Three concrete providers satisfy one interface |
| Provider failover | `ProviderWithFallback` transparently switches to secondary |
| Exponential backoff without sleeping | `RetryPolicy.NextDelay` validated by logged attempt counts |
| Dead-letter queue | High-failure-rate scenario forces notifications past `MaxAttempts` |
| Atomic delivery statistics | `DeliveryTracker` is read after all goroutines finish |
| Channel routing | Dispatcher sends each notification to the correct provider pair |
| Priority handling | `Priority` field preserved through the dispatch pipeline |

---

## Extending to Production

See `scaling_discussion.md` for async queuing, idempotency keys, provider rate
limits, priority queues, unsubscribe management, DLQ retry strategy, and
Kubernetes CronJob reprocessing.
