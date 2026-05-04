# Chapter 83 Exercises — Mocking

## Exercise 1 — Notification Service Mocks (`exercises/01_service_mocks`)

Build a full mock suite for a multi-channel notification service that dispatches via SMS and push providers based on each user's preferences.

### System under test

```go
func (ns *NotificationService) Notify(ctx context.Context, userID, title, message string) (*SendResult, error)
```

Behaviour:
- Looks up user preferences via `UserStore`
- For each channel in `user.Channels`, dispatches to the matching provider
- Channel failures are non-fatal — collect in `SendResult.Errors`, continue
- `Sent` counter tracks successful dispatches

### Interfaces to mock

```go
type SMSProvider interface {
    SendSMS(ctx context.Context, phone, message string) error
}
type PushProvider interface {
    SendPush(ctx context.Context, deviceToken, title, body string) error
}
type UserStore interface {
    GetUser(ctx context.Context, userID string) (*UserPrefs, error)
}
```

### Test cases to implement

1. **User not found** — `UserStore` returns error → `Notify` returns error, no provider called
2. **SMS-only user** — single channel; verify phone number and message content via spy
3. **Both channels** — verify both SMS and push called; `Sent` counter = 2
4. **SMS failure, push succeeds** — `SendResult.Errors` has 1 entry; `SendResult.Channels = ["push"]`
5. **Push argument capture** — verify title and body forwarded correctly to provider
6. **Concurrent safety** — 5 goroutines each notify a different user; total `Sent` = 5

### Implementation hints

- `SpySMSProvider.Errs []error` — per-call errors: index 0 applies to first call, index 1 to second, etc.
- `FakeUserStore` pre-seeded with test users avoids DB dependency
- Guard spy slices with `sync.Mutex` for concurrency test
- Test the "both channels, SMS fails" case by pre-programming `Errs[0] = someError`
