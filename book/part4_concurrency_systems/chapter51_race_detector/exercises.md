# Chapter 51 — Exercises

## 51.1 — Race audit

Run [`exercises/01_audit`](exercises/01_audit/main.go).

Five components — `RequestCounter`, `Cache`, `Collector`, `Config`, `Broadcaster` — each have one hidden data race marked with `// TODO RACE N`. Your task:

1. Run `go run -race ./exercises/01_audit` and read the reports.
2. Fix each `TODO RACE` comment using the appropriate synchronisation primitive.
3. Run `go run -race ./exercises/01_audit` again and confirm zero race reports.

Expected fixes:
- Race 1: replace `int64` fields with `atomic.Int64`
- Race 2: add `sync.RWMutex` to `Cache`
- Race 3: protect `Collector.results` with a `sync.Mutex`
- Race 4: replace `bool+string` with `sync.Once`
- Race 5: add `sync.RWMutex` to `Broadcaster`

## 51.2 ★ — Race-free cache with expiry and concurrent purge

Write a `TTLCache[K comparable, V any]` that:
- `Set(key K, value V, ttl time.Duration)` stores a value with expiry
- `Get(key K) (V, bool)` returns the value if not expired
- `Purge()` removes all expired entries
- Passes `go test -race` when all methods are called from concurrent goroutines

Use a `sync.RWMutex`: read lock for `Get`, write lock for `Set` and `Purge`.

## 51.3 ★★ — Instrument with GORACE

Build a binary from `examples/01_race_patterns` with the race detector enabled:
```bash
go build -race -o /tmp/rp ./examples/01_race_patterns
```

Then run it with:
```bash
GORACE="halt_on_error=0 log_path=/tmp/race" /tmp/rp
```

Read each file `/tmp/race.pid.*` and for each race: (a) identify the conflicting goroutines, (b) identify the file and line, and (c) name the synchronisation primitive you would add to fix it.
