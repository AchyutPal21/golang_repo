# Chapter 84 Checkpoint — Integration Testing

## Concepts to know

- [ ] What is the difference between a unit test and an integration test?
- [ ] What is `httptest.NewRecorder`? What does it replace?
- [ ] When should you use `httptest.NewServer` instead of `httptest.NewRecorder`?
- [ ] What does testcontainers-go do? Why is it better than `docker-compose up` in CI?
- [ ] Why should you start one container per package in `TestMain` rather than one per test?
- [ ] What is the transaction rollback pattern for test isolation?
- [ ] What are the three seeding strategies and when is each appropriate?
- [ ] How do you isolate parallel tests that all write to the same DB?
- [ ] What is `t.Cleanup`? How is it better than `defer` in integration tests?

## Code exercises

### 1. Handler integration test

Write an integration test for a `POST /users` handler that:
- Returns 201 on valid input
- Returns 400 when `email` field is missing
- Returns 409 when email already exists (simulate with a fake store)

### 2. Transaction rollback isolation

Write two tests that use the same `*TxDB`. Verify that:
- A row inserted and committed in test 1 is visible in test 2
- A row inserted but rolled back in test 1 is NOT visible in test 2

### 3. TestMain container pattern

Write the structure (code comments, no real Docker) for a `TestMain` that:
- Starts a Postgres container
- Runs migrations
- Provides a `*sql.DB` to all tests
- Terminates the container after all tests complete

## Quick reference

```go
// httptest.NewRecorder
req := httptest.NewRequest("GET", "/users/1", nil)
rr := httptest.NewRecorder()
handler.ServeHTTP(rr, req)
// assert rr.Code, rr.Body.String()

// httptest.NewServer
srv := httptest.NewServer(handler)
defer srv.Close()
resp, _ := http.Get(srv.URL + "/users/1")

// testcontainers-go (real)
container, _ := postgres.Run(ctx, "postgres:16-alpine", ...)
t.Cleanup(func() { container.Terminate(ctx) })
connStr, _ := container.ConnectionString(ctx)

// Transaction isolation
tx, _ := db.BeginTx(ctx, nil)
t.Cleanup(func() { tx.Rollback() })
repo := NewRepo(tx)
```

## What to remember

- Use `httptest.NewRecorder` for handler logic; use `httptest.NewServer` for end-to-end HTTP.
- Rollback after every test — it's the fastest way to keep tests isolated.
- `TestMain` container pattern: one container startup per `go test ./pkg` invocation.
- Never rely on test execution order — each test must set up its own state.
- Tag integration tests with `//go:build integration` to skip them in unit-test runs.
