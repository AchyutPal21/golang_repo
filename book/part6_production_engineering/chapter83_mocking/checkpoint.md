# Chapter 83 Checkpoint — Mocking

## Concepts to know

- [ ] Name the 5 types of test doubles and describe when to use each.
- [ ] What is the difference between a spy and a mock?
- [ ] Why does Go's interface system make mocking straightforward compared to dynamic languages?
- [ ] What is a fake? Give an example of when you'd use a fake instead of a spy.
- [ ] What is "over-mocking" and why is it harmful?
- [ ] Should you mock internal functions or packages? Why or why not?
- [ ] Why should you use `sync.Mutex` in spy implementations?
- [ ] What is `testing.TB` and why write mock helpers against it?
- [ ] When would you use a generated mock (mockery/gomock) vs. a hand-written one?

## Code exercises

### 1. Stub clock

Write a `Clock` interface with `Now() time.Time`. Implement a `StubClock` that returns a fixed time. Use it to test a function that formats "Today is Monday" based on the current day.

### 2. Spy logger

Write a `Logger` interface with `Log(level, message string)`. Implement a `SpyLogger` that records all calls. Test that an error-handling code path calls `Log("error", ...)` exactly once.

### 3. Configurable HTTP client

Write a `HTTPClient` interface with `Do(req *http.Request) (*http.Response, error)`. Write a `MockHTTPClient` that returns programmed `(*http.Response, error)` pairs in sequence. Test a function that retries on 500 status.

### 4. Fake event store

Write an `EventStore` interface with `Append(event Event)` and `Load(aggregateID string) []Event`. Implement `FakeEventStore` with an in-memory map. Test that aggregate rebuilding produces the expected state.

## Quick reference

```go
// Stub
type StubEmailer struct{ err error }
func (s *StubEmailer) Send(_ context.Context, _, _, _ string) error { return s.err }

// Spy
type SpyEmailer struct {
    mu    sync.Mutex
    Calls []EmailCall
}
func (s *SpyEmailer) Send(_ context.Context, to, subject, body string) error {
    s.mu.Lock(); defer s.mu.Unlock()
    s.Calls = append(s.Calls, EmailCall{to, subject, body})
    return nil
}

// Fake
type FakeDB struct{ rows map[string]Record }
func (f *FakeDB) Find(id string) (Record, error) {
    r, ok := f.rows[id]
    if !ok { return Record{}, ErrNotFound }
    return r, nil
}

// Configurable mock
mock := (&MockGateway{}).OnCharge("", errTimeout).OnCharge("ch-1", nil)
```

## What to remember

- Mock external I/O (DB, HTTP, email, time); use real logic for internal helpers.
- Spies need a mutex — tests may call the production code from goroutines.
- `t.Helper()` in assertion helpers makes failure lines point to the test, not the helper.
- Generated mocks are fine for large interfaces; hand-written doubles are clearer for narrow ones.
- If your mock has more lines than your test, you're probably mocking too much.
