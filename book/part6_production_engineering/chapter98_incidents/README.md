# Chapter 98 — Incident Management & Debugging

When production breaks, you need tools and processes that let you diagnose quickly, minimize blast radius, and learn from what happened. This chapter covers Go-specific debugging tools and incident management practices.

## Topics

| # | Topic | Key ideas |
|---|-------|-----------|
| 1 | Goroutine dumps | `SIGQUIT`, `/debug/pprof/goroutine`, deadlock detection |
| 2 | Panic recovery & postmortem | Structured panics, recovery middleware, incident timeline |
| E | Debugging toolkit | Combined: health, dumps, profiling, incident timeline |

## Examples

### `examples/01_goroutine_dumps`

Goroutine leak and deadlock debugging:
- `runtime.Stack` — capture all goroutine stacks
- `/debug/pprof/goroutine?debug=2` reference
- Goroutine count tracking over time
- Leak detection pattern: compare before/after counts

### `examples/02_panic_recovery`

Panic handling and postmortem patterns:
- Recovery middleware that captures panic + stack trace
- Structured incident record (time, stack, context)
- Postmortem template (5 whys, timeline, action items)
- `defer` + `recover` best practices

### `exercises/01_incident_timeline`

Incident timeline and debugging toolkit:
- Event timeline builder with severity levels
- Automated health snapshot (goroutines, heap, GC)
- Incident report generator
- MTTR (Mean Time To Recovery) calculator

## Key Concepts

**Goroutine dump**
```bash
# In production (always-on):
kill -QUIT <pid>  # dumps all goroutine stacks to stderr

# Via pprof endpoint:
curl http://localhost:6060/debug/pprof/goroutine?debug=2

# In tests:
if after := runtime.NumGoroutine(); after > before {
    t.Errorf("goroutine leak: %d → %d", before, after)
}
```

**Panic recovery pattern**
```go
defer func() {
    if r := recover(); r != nil {
        stack := make([]byte, 64*1024)
        n := runtime.Stack(stack, false)
        log.Errorf("panic: %v\n%s", r, stack[:n])
        // Report to error tracking (Sentry, etc.)
    }
}()
```

**Incident severity levels**

| Level | Impact | Response |
|-------|--------|----------|
| SEV-1 | Total outage | Immediate page, all hands |
| SEV-2 | Partial outage / data loss risk | Page primary oncall |
| SEV-3 | Degraded performance | Next business day |
| SEV-4 | Minor issue | Backlog |

**Postmortem structure**
1. Incident summary (date, duration, impact)
2. Timeline (what happened when)
3. Root cause analysis (5 whys)
4. Contributing factors
5. Action items (with owners and due dates)
6. Lessons learned

## Running

```bash
go run ./book/part6_production_engineering/chapter98_incidents/examples/01_goroutine_dumps
go run ./book/part6_production_engineering/chapter98_incidents/examples/02_panic_recovery
go run ./book/part6_production_engineering/chapter98_incidents/exercises/01_incident_timeline
```
