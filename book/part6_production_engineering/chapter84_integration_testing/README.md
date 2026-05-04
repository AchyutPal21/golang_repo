# Chapter 84 — Integration Testing

Integration tests verify that components work together correctly. They test the seams between layers: handler → service → repository → database.

## The testing pyramid

```
        ▲
       /E2E\          few, slow, test full system
      /------\
     / Integ  \       moderate, test component wiring
    /----------\
   /  Unit      \     many, fast, test logic in isolation
  /--------------\
```

Unit tests test logic; integration tests test wiring. Both are necessary.

## httptest — fast HTTP handler tests

Use `httptest.NewRecorder` to test a handler directly (no network, no server):

```go
req := httptest.NewRequest("POST", "/products", strings.NewReader("name=Widget&price=999"))
req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
rr := httptest.NewRecorder()
handler.ServeHTTP(rr, req)
if rr.Code != http.StatusCreated { t.Errorf(...) }
```

Use `httptest.NewServer` when you need the full HTTP stack (middleware, routing):

```go
srv := httptest.NewServer(router)
defer srv.Close()
resp, _ := http.Post(srv.URL+"/products", ...)
```

## testcontainers-go — real databases in tests

```go
container, _ := postgres.Run(ctx, "postgres:16-alpine",
    postgres.WithDatabase("testdb"),
    postgres.WithUsername("test"),
    postgres.WithPassword("test"),
)
t.Cleanup(func() { container.Terminate(ctx) })
connStr, _ := container.ConnectionString(ctx, "sslmode=disable")
db, _ := sql.Open("pgx", connStr)
```

Start one container per test package in `TestMain`, not per test:

```go
var testDB *sql.DB

func TestMain(m *testing.M) {
    // start container, run migrations
    code := m.Run()
    // stop container
    os.Exit(code)
}
```

## Transaction-based test isolation

Each test opens a transaction and rolls back at the end. The table stays clean:

```go
func TestCreate(t *testing.T) {
    tx, _ := db.BeginTx(ctx, nil)
    t.Cleanup(func() { tx.Rollback() })
    repo := NewRepository(tx)
    id, err := repo.Create(ctx, "Widget", 999)
    // ... assertions
}
```

This is fast (no truncate/re-seed) and safe (each test is fully isolated).

## Seeding strategies

| Strategy | When to use |
|----------|-------------|
| Seed in each test | Best for isolation; each test is self-contained |
| SQL seed file in `TestMain` | Read-only reference data that never changes |
| Factory helpers | Reduce repetition across many tests |

## Parallel test isolation

For parallel tests that all write to the same DB, use per-test schemas or transactions:

```go
func newTestSchema(t *testing.T, db *sql.DB) string {
    schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
    db.Exec("CREATE SCHEMA " + schema)
    t.Cleanup(func() { db.Exec("DROP SCHEMA " + schema + " CASCADE") })
    return schema
}
```

## Examples in this chapter

| File | Topic |
|------|-------|
| `examples/01_testcontainers/main.go` | httptest recorder + server, testcontainers reference |
| `examples/02_db_tests/main.go` | Transaction isolation, rollback, seeding |
| `exercises/01_postgres_suite/main.go` | Full CRUD + concurrency integration suite |
