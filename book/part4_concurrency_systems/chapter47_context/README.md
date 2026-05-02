# Chapter 47 — context Package

> **Part IV · Concurrency & Systems** | Estimated reading time: 18 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

`context.Context` is how Go propagates cancellation, deadlines, and request-scoped metadata through a call tree. Every library that does I/O — `database/sql`, `net/http`, `grpc` — accepts a context as its first argument. Writing correct concurrent server code requires understanding context deeply.

---

## 47.1 — The Context interface

```go
type Context interface {
    Deadline() (deadline time.Time, ok bool)
    Done() <-chan struct{}
    Err() error
    Value(key any) any
}
```

- `Done()` — closed when the context is cancelled or times out; nil for non-cancellable contexts.
- `Err()` — `nil` while active; `context.Canceled` after cancel; `context.DeadlineExceeded` after timeout.
- `Deadline()` — returns the absolute deadline, if any.
- `Value(key)` — retrieves a value stored with `WithValue`.

---

## 47.2 — Root contexts

```go
context.Background()  // never cancelled; top of every tree
context.TODO()        // placeholder when the right context isn't determined yet
```

---

## 47.3 — Cancellation

```go
ctx, cancel := context.WithCancel(parent)
defer cancel()  // always defer — frees resources even if not cancelled early

go func() {
    select {
    case <-ctx.Done():
        return  // cancelled
    case result := <-work:
        process(result)
    }
}()

cancel()  // signal all goroutines in this context tree
```

---

## 47.4 — Timeout and Deadline

```go
ctx, cancel := context.WithTimeout(parent, 5*time.Second)
defer cancel()

ctx, cancel := context.WithDeadline(parent, time.Now().Add(5*time.Second))
defer cancel()
```

`WithTimeout` is shorthand for `WithDeadline(parent, time.Now().Add(d))`. Both produce a context that is cancelled automatically when the duration elapses — or earlier if the parent is cancelled or `cancel()` is called manually. Always `defer cancel()` to release the timer.

---

## 47.5 — Cancellation propagation

When a parent context is cancelled, **all descendant contexts are cancelled immediately**:

```
Background
  └─ parent (WithCancel) ← cancel()
        ├─ child1 (WithTimeout) ← also cancelled
        └─ child2 (WithCancel)
              └─ grandchild ← also cancelled
```

This propagates through the entire goroutine tree with zero extra code.

---

## 47.6 — context.WithValue

Attach request-scoped metadata (not function parameters):

```go
type key string  // unexported type prevents collision across packages
const reqIDKey key = "request_id"

ctx = context.WithValue(ctx, reqIDKey, "req-abc")
id := ctx.Value(reqIDKey).(string)
```

**Use for:** trace IDs, auth tokens, request IDs — values that flow down the call chain implicitly.  
**Do NOT use for:** optional function arguments, configuration, dependencies.

---

## 47.7 — Idiomatic usage rules

1. **First parameter** — `ctx context.Context` is always the first parameter of any function that accepts it.
2. **Never store in struct** — pass context through the call chain, don't cache it in a struct field.
3. **Never pass nil** — use `context.Background()` or `context.TODO()` as the root.
4. **Always defer cancel** — even if you cancel early; the second `cancel()` is a no-op.
5. **Check Done in loops** — `select { case <-ctx.Done(): return; default: }` at the top of each iteration.

---

## Running the examples

```bash
cd book/part4_concurrency_systems/chapter47_context

go run ./examples/01_context_basics    # WithCancel, WithTimeout, WithDeadline, propagation, Err
go run ./examples/02_context_patterns  # WithValue, call chain, anti-patterns, middleware

go run ./exercises/01_request_pipeline # 4-stage order pipeline with timeout, cancel, business errors
```

---

## Key takeaways

1. **context.Done()** — select on it in every goroutine that should be cancellable.
2. **Always defer cancel()** — forgetting leaks a timer goroutine.
3. **Propagation is automatic** — cancel a parent, all children are cancelled.
4. **ctx.Err()** — `context.Canceled` vs `context.DeadlineExceeded` distinguishes why.
5. **WithValue** — for metadata (trace IDs), not parameters; use unexported key types.
6. **First parameter convention** — `func Do(ctx context.Context, ...)` — context is always first.

---

## Cross-references

- **Chapter 44** — select/Timeouts: the `select` + `ctx.Done()` pattern
- **Chapter 42** — Goroutines: done channels — context replaces hand-rolled done channels
- **Chapter 56** — Production HTTP Server: `http.Request.Context()` and request-scoped cancellation
