# Scaling Discussion — Notification Service

The `main.go` simulation dispatches notifications synchronously in a tight loop.
In production every design choice here would change. This document covers the
seven most important gaps between the capstone and a real service.

---

## 1. Async Delivery via Buffered Channels / Queues

Sending an HTTP request to SES or Twilio inside a request handler blocks the
caller for tens to hundreds of milliseconds. Any spike in notification volume
stalls API response times.

**Solution — decouple ingestion from delivery:**

```
HTTP Handler
    │  enqueue(notification)
    ▼
Buffered channel / persistent queue (Redis Streams, SQS, Kafka)
    │  consume
    ▼
Worker pool  ──► provider
```

In Go, a buffered channel provides a lightweight in-process queue:

```go
type Dispatcher struct {
    queue chan Notification // e.g. make(chan Notification, 10_000)
}

// Enqueue never blocks the caller for more than the channel capacity.
func (d *Dispatcher) Enqueue(n Notification) error {
    select {
    case d.queue <- n:
        return nil
    default:
        return ErrQueueFull
    }
}

// Workers drain the channel concurrently.
func (d *Dispatcher) startWorkers(n int) {
    for i := 0; i < n; i++ {
        go func() {
            for n := range d.queue {
                d.deliver(n)
            }
        }()
    }
}
```

For durability across restarts the channel is replaced by a persistent broker
(Redis Streams `XADD` / `XREADGROUP`, or SQS). The queue also acts as a natural
buffer during provider outages — messages accumulate rather than being lost.

---

## 2. Idempotency Keys to Prevent Duplicate Sends

Network failures are ambiguous: a timeout does not tell you whether the provider
received the request or not. A naive retry will send the same SMS twice.

**Solution — idempotency key per notification attempt:**

Most providers accept an idempotency/deduplication key in the HTTP header or
request body:

```
POST https://api.twilio.com/2010-04-01/Accounts/.../Messages
X-Twilio-Idempotency-Token: notif-08-attempt-2
```

Generate the key as `<notificationID>-attempt-<n>`. The provider discards the
second request with the same key and returns the outcome of the first.

On the service side, track completion state in Redis before and after each
attempt:

```
SET idempotency:<notifID> "pending"   EX 86400  # 24-hour TTL
SET idempotency:<notifID> "delivered" EX 86400
```

A worker that restarts mid-flight checks this key first and skips re-delivery
if the state is already `"delivered"`.

---

## 3. Provider Rate Limits

Every provider publishes hard send limits. Exceeding them causes 429 errors and
temporary bans.

| Provider | Default rate limit |
|---|---|
| Amazon SES | 14 messages / second (sandbox); up to 1 000/s after limit increase |
| Twilio SMS | 1 message / second per originating phone number (long code) |
| Firebase Cloud Messaging (FCM) | 600 000 messages / minute per project |
| Apple APNs | No published per-second limit, but token errors accumulate |

**Solution — token-bucket rate limiter per provider:**

```go
import "golang.org/x/time/rate"

emailLimiter := rate.NewLimiter(rate.Limit(14), 14)  // 14/s, burst 14
smsLimiter   := rate.NewLimiter(rate.Limit(1), 1)    // 1/s, burst 1

func (p *EmailProvider) Send(ctx context.Context, n Notification) error {
    if err := emailLimiter.Wait(ctx); err != nil {
        return err
    }
    // ... make HTTP request
}
```

For multi-instance deployments a single Go limiter is insufficient because each
pod has its own counter. Use Redis to share state:

```
INCR rate:<provider>:<second>
EXPIRE rate:<provider>:<second> 2
```

If the counter exceeds the limit, sleep until the next window or push the
notification back to the queue.

---

## 4. Priority Queues for Transactional vs. Marketing

An OTP for a login attempt must not queue behind a weekly newsletter.

**Solution — separate queues per priority tier, polled in order:**

```
critical queue     ──► worker pool (largest)
transactional queue ──► worker pool (medium)
marketing queue    ──► worker pool (smallest, rate-throttled)
```

In Redis Streams this is three separate streams. Workers always drain
`critical` first, then `transactional`, then `marketing`. A Go select with
weighted channels achieves the same effect in-process:

```go
for {
    select {
    case n := <-criticalQ:
        deliver(n)
    case n := <-transactionalQ:
        deliver(n)
    default:
        select {
        case n := <-criticalQ:
            deliver(n)
        case n := <-transactionalQ:
            deliver(n)
        case n := <-marketingQ:
            deliver(n)
        }
    }
}
```

Marketing notifications can also be subject to send-time optimisation (deliver
at a time likely to result in a high open rate) without impacting the latency
of transactional messages.

---

## 5. Unsubscribe and Preference Management

Sending to a user who has unsubscribed is a legal violation in most jurisdictions
(CAN-SPAM, GDPR, TCPA) and immediately harms sender reputation.

**Solution — preference lookup before dispatch:**

```go
type PreferenceStore interface {
    IsSubscribed(userID string, channel Channel, category string) (bool, error)
}

func (d *Dispatcher) deliver(n Notification) {
    ok, err := d.prefs.IsSubscribed(n.UserID, n.Channel, n.Category)
    if err != nil || !ok {
        d.tracker.RecordSuppressed(n.Channel)
        return
    }
    // ... proceed with delivery
}
```

Store preferences in a low-latency read path (Redis hash per user). Honour
unsubscribe requests within seconds by evicting the cache entry. Always provide
a one-click unsubscribe link (`List-Unsubscribe` header in email). Transactional
messages (OTPs, receipts) are exempt from marketing unsubscribes but must still
respect channel-level blocks (e.g. a user who has never provided a phone number).

---

## 6. Dead-Letter Queue Retry Strategy

Notifications land in the DLQ because every retry attempt failed within the
immediate dispatch window. They must not be silently discarded.

**Graduated retry strategy:**

| Reprocessing pass | Delay after DLQ arrival | Action on continued failure |
|---|---|---|
| Pass 1 | 5 minutes | Retry with same policy |
| Pass 2 | 1 hour | Retry, escalate to ops alert |
| Pass 3 | 24 hours | Retry, notify engineering |
| Pass 4 | — | Move to permanent dead store, generate incident |

Before reprocessing, check whether the underlying issue is likely resolved:

- **Provider outage**: poll provider status page API or circuit-breaker state.
- **Invalid address**: validate email/phone format before retrying; skip if
  invalid rather than burning retry budget.
- **Expired content**: discard OTPs or time-sensitive offers that are older
  than their validity window.

---

## 7. Kubernetes CronJob for DLQ Reprocessing

In a Kubernetes deployment, DLQ reprocessing runs as a `CronJob` rather than
inside the main service, so it does not consume worker capacity during peak load.

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: dlq-reprocessor
spec:
  schedule: "*/5 * * * *"     # run every 5 minutes
  concurrencyPolicy: Forbid   # never run two instances simultaneously
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
            - name: dlq-reprocessor
              image: ghcr.io/example/notification-service:latest
              args: ["--mode=dlq-reprocess", "--max-age=24h"]
              env:
                - name: REDIS_URL
                  valueFrom:
                    secretKeyRef:
                      name: notification-secrets
                      key: redis-url
```

The job reads up to N entries from the DLQ stream
(`XREADGROUP GROUP dlq-consumers reprocessor COUNT 100 STREAMS dlq >`),
attempts delivery using the normal dispatcher, and acknowledges (`XACK`) each
entry on success. Entries that fail again are left in the stream for the next
scheduled run, honouring the graduated delay table above by checking a
`next_attempt_at` field stored alongside each DLQ entry.

`concurrencyPolicy: Forbid` prevents two reprocessors from racing to deliver the
same notification when a job run takes longer than the schedule interval.
