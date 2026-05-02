# Chapter 51 — Race Detector

## What you will learn

- What a data race is and why it causes silent corruption rather than crashes
- How to enable the race detector: `go run -race`, `go test -race`, `go build -race`
- Reading and interpreting a race detector report
- The five most common race patterns in Go code
- Fixes for each pattern: `sync/atomic`, `sync.Mutex`/`sync.RWMutex`, channel ownership, explicit capture, `sync.Once`
- Why concurrent map writes always panic and how `sync.Map` avoids the issue
- CI integration: always run tests with `-race`

---

## Enabling the race detector

```bash
go run -race ./cmd/server         # run with race detection
go test -race ./...                # test all packages with race detection
go build -race -o bin/server ./cmd/server  # build instrumented binary
```

The race detector adds ~5–10× memory overhead and ~2× CPU overhead. It is not suitable for production, but it should always be used in CI.

---

## Reading a race detector report

```
WARNING: DATA RACE
Write at 0x00c0001a4018 by goroutine 7:
  main.increment()
      /app/main.go:23 +0x28

Previous read at 0x00c0001a4018 by goroutine 8:
  main.increment()
      /app/main.go:23 +0x1c

Goroutine 7 (running) created at:
  main.main()
      /app/main.go:35 +0x68
```

The report tells you:
1. The **address** that was accessed concurrently
2. The **goroutine and stack** of each conflicting access
3. **Where each goroutine was created** (the creation stack)

---

## The five common races and their fixes

| Pattern | Root cause | Fix |
|---|---|---|
| Unsynchronised counter | `n++` is three instructions | `atomic.Int64.Add` |
| Shared struct fields | Multiple goroutines write different fields | `sync.Mutex` wrapping both fields |
| Concurrent map writes | Go map is not concurrency-safe | `sync.RWMutex` or `sync.Map` |
| Concurrent slice append | Slice header (ptr/len/cap) is not atomic | Channel-based single owner |
| Closure over loop variable | All goroutines share the same variable | Pass as function argument |
| DIY bool-flag once-init | Read-check-write is not atomic | `sync.Once` |

---

## Why `go test -race` belongs in CI

The race detector is a dynamic analysis tool — it only reports races that actually occur during a run. Tests exercise more code paths with concurrent access than `main` alone. Running `go test -race ./...` in CI converts probabilistic, hard-to-reproduce bugs into deterministic test failures.

```yaml
# .github/workflows/ci.yml
- run: go test -race ./...
```

---

## GORACE environment variables

```
GORACE="log_path=/tmp/race.log"         # write to file instead of stderr
GORACE="halt_on_error=0"               # continue after first race report
GORACE="max_goroutines=1000"           # cap tracked goroutines (memory)
```

---

## Examples

| File | Demonstrates |
|---|---|
| `examples/01_race_patterns/main.go` | Counter, struct, append, closure, init races |
| `examples/02_fixing_races/main.go` | Atomic, mutex, channel, capture, sync.Once fixes |

## Exercise

`exercises/01_audit/main.go` — five components with labelled TODO RACE comments; fix each and verify with `go run -race`.
