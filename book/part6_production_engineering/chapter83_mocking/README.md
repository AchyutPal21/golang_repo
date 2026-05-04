# Chapter 83 — Mocking

Mocking replaces real dependencies with controlled substitutes so tests run fast, deterministically, and without external services. Go makes this natural through interface-based design: any type that satisfies an interface can be swapped in.

## Test double taxonomy

| Type | Records calls | Returns programmed responses | Has real logic |
|------|--------------|------------------------------|----------------|
| **Dummy** | No | No | No (satisfies compiler) |
| **Stub** | No | Yes (fixed) | No |
| **Spy** | Yes | Optional | No |
| **Mock** | Yes | Yes (per-expectation) | No |
| **Fake** | No | Yes (derived from logic) | Yes (lightweight) |

## Stub

Simplest double — returns a fixed response, records nothing.

```go
type StubEmailSender struct{ err error }
func (s *StubEmailSender) Send(_ context.Context, to, subject, body string) error {
    return s.err
}
```

Use when: you need a dependency to "not fail" and don't care what it does.

## Spy

Records all calls. You assert against the recording after the fact.

```go
type SpyEmailSender struct {
    Calls []EmailCall
    Err   error
}
func (s *SpyEmailSender) Send(_ context.Context, to, subject, body string) error {
    s.Calls = append(s.Calls, EmailCall{to, subject, body})
    return s.Err
}
// In test:
if spy.Calls[0].To != "alice@example.com" { t.Errorf(...) }
```

Use when: you need to verify side effects (emails sent, events fired, audit records).

## Fake

A working in-memory implementation with real logic. Faster and simpler than the real thing.

```go
type FakeUserRepository struct {
    users map[string]*User
}
func (f *FakeUserRepository) FindByID(_ context.Context, id string) (*User, error) {
    u, ok := f.users[id]
    if !ok { return nil, fmt.Errorf("not found") }
    return u, nil
}
```

Use when: you need the collaborator to behave correctly (queries, filtering, state updates).

## Configurable mock

Programs a sequence of responses, one per call.

```go
mock := (&MockPaymentGateway{}).
    OnCharge("", fmt.Errorf("timeout")).  // first call fails
    OnCharge("ch-123", nil)               // second call succeeds
```

Use when: testing retry logic, transient failures, or multi-step protocols.

## Call-order verification

Track the sequence of method calls to ensure operations happen in the right order (e.g. cache invalidate before write).

## Avoiding over-mocking

Signs you're over-mocking:
- Test setup is longer than the assertion
- Test breaks when you rename a private method
- Mock expectations mirror the implementation line-for-line

Rules:
- **Mock external boundaries only**: DB, HTTP clients, email/SMS, clocks
- **Use fakes for internal collaborators** with real logic (in-memory stores)
- **Test observable behaviour**: return values and state, not internal call sequences
- Prefer integration tests for wiring; keep unit tests focused on logic

## Real-world tools

```bash
# mockery (most popular, generates from interface)
go install github.com/vektra/mockery/v2@latest
mockery --name=UserRepository --dir=./internal

# gomock (Google)
go install go.uber.org/mock/mockgen@latest
mockgen -source=internal/repo.go -destination=internal/mock_repo.go
```

Both generate code you check in. The generated mocks look like hand-written spies — same concepts, less typing.

## Examples in this chapter

| File | Topic |
|------|-------|
| `examples/01_interfaces_mocks/main.go` | Stubs, spies, fakes, configurable mocks |
| `examples/02_mock_patterns/main.go` | Call ordering, argument capture, expectations |
| `exercises/01_service_mocks/main.go` | Notification service mock suite |
