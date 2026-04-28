# Chapter 15 — defer, panic, recover

> **Part II · Core Language** | Estimated reading time: 24 min | Runnable examples: 3 | Exercises: 1

---

## Why this chapter matters

`defer`, `panic`, and `recover` are Go's three mechanisms for managing cleanup and unexpected failures. They are simpler than exceptions, but require care: `defer` has surprising argument-evaluation semantics; `recover` only works in exactly one context; and `panic` is rarely the right answer when `error` will do. Every production Go program relies on `defer` for resource cleanup and mutex unlocking. Understanding the execution model prevents bugs and lets you write correct cleanup code the first time.

---

## 15.1 — defer

`defer` registers a function call to run when the surrounding function returns. The call runs whether the function returns normally, via an explicit `return`, or due to a panic.

```go
func readFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer f.Close() // registered here; runs when readFile returns

    // ... use f
}
```

`defer` is the idiomatic way to express "clean up X when I am done with it". It keeps the acquisition and release code adjacent, reducing the chance of leaks.

---

## 15.2 — Execution order: LIFO

Deferred calls execute in last-in, first-out order (stack). Multiple defers in the same function run in reverse registration order:

```go
defer fmt.Println("A") // runs 3rd
defer fmt.Println("B") // runs 2nd
defer fmt.Println("C") // runs 1st
```

Output: `C B A`. This mirrors resource acquisition order: if you acquire A then B, you typically release B then A.

---

## 15.3 — Argument evaluation timing

Deferred **arguments** are evaluated when the `defer` statement is reached, not when the call eventually runs:

```go
x := 10
defer fmt.Println(x) // x evaluated NOW → 10 will be printed
x = 99
// function returns, deferred call prints 10
```

This catches many people out. If you need the deferred call to see the value at return time, use a closure:

```go
defer func() { fmt.Println(x) }() // x captured by reference → sees 99
```

---

## 15.4 — defer and named return values

Named return values are variables in the function scope. Deferred closures can read and modify them:

```go
func loadUser(id int) (user User, err error) {
    defer func() {
        if err != nil {
            err = fmt.Errorf("loadUser(%d): %w", id, err)
        }
    }()
    // ... err may be set here
    return
}
```

This pattern lets you annotate errors with context at the boundary without modifying every `return` statement.

---

## 15.5 — defer in loops

`defer` inside a loop registers one call per iteration. All of them run when the **function** returns, not when each iteration ends:

```go
for _, f := range files {
    defer f.Close() // NOT called per-iteration!
}
```

If you need per-iteration cleanup, factor the loop body into a function:

```go
for _, f := range files {
    func() {
        defer f.Close()
        process(f)
    }()
}
```

---

## 15.6 — panic

`panic` stops normal execution, begins unwinding the call stack, and runs all deferred functions. If nothing recovers the panic, the program crashes with a stack trace.

When to panic:
- The program is in a state from which it cannot safely continue (nil pointer, out-of-bounds, impossible branch)
- Initialisation invariants are violated (in `init` or package-level `var`)
- The caller has broken an explicit contract (the function is documented to require non-nil input)

When **not** to panic: when the caller can reasonably handle the condition. Return an `error` instead.

```go
// DO: return an error the caller can handle
func Divide(a, b float64) (float64, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}

// DO: panic when the invariant violation is the caller's fault
func mustNonNil[T any](v *T) *T {
    if v == nil {
        panic("mustNonNil: nil pointer")
    }
    return v
}
```

---

## 15.7 — recover

`recover` stops a panic and returns the value passed to `panic`. It **only works inside a deferred function**:

```go
func safe(f func()) (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("panic: %v", r)
        }
    }()
    f()
    return nil
}
```

If `recover` is called outside a deferred function, or if no panic is in progress, it returns `nil`.

The contract:
1. Call `recover` inside `defer func() { ... }()`
2. Check the return value — `nil` means no panic
3. The function returns whatever the deferred code sets (typically an error)

---

## 15.8 — Re-panicking

If a recover function sees a panic it doesn't own, it should re-panic:

```go
defer func() {
    r := recover()
    if r == nil {
        return
    }
    if _, ok := r.(*myError); ok {
        // our panic type — handle it
        return
    }
    panic(r) // not ours — re-panic
}()
```

Unconditional `recover()` silences all panics indiscriminately, which hides bugs. Always type-check before deciding to absorb a panic.

---

## 15.9 — Production defer patterns

### Timer / trace

```go
func elapsed(name string) func() {
    start := time.Now()
    return func() {
        fmt.Printf("[trace] %s took %v\n", name, time.Since(start))
    }
}

func doWork() {
    defer elapsed("doWork")() // note the double ()
    // ...
}
```

The outer call (`elapsed("doWork")`) runs at `defer` registration and captures `start`. The returned function runs at return and prints elapsed time.

### Transaction rollback

```go
tx := db.Begin()
defer tx.Rollback() // safe no-op after Commit

// ... do work
return tx.Commit()  // Rollback is a no-op after this
```

Register `Rollback` before any work. If anything fails and the function returns early, rollback runs automatically.

### Mutex unlock

```go
func (c *Cache) Set(k, v string) {
    c.mu.Lock()
    defer c.mu.Unlock() // unlock on all exit paths, including panic
    c.data[k] = v
}
```

`defer` after `Lock()` is idiomatic Go; it prevents the common lock-leak bug.

### Error annotation

```go
func load(path string) (cfg Config, err error) {
    defer func() {
        if err != nil {
            err = fmt.Errorf("load(%q): %w", path, err)
        }
    }()
    // ...
}
```

Annotate at the boundary without modifying every `return` site.

---

## 15.10 — defer cost

`defer` has a runtime cost. As of Go 1.14, inlined defers (those in non-looping code paths) are nearly free due to the "open-coded defer" optimisation. Defers in loops or behind runtime-unknown conditions are heap-allocated and slower.

For hot paths (called millions of times per second), profile before using `defer`. For the vast majority of code, the cost is irrelevant.

---

## Running the examples

```bash
cd book/part2_core_language/chapter15_defer_panic_recover

go run ./examples/01_defer_mechanics  # LIFO, arg eval, named return, loop
go run ./examples/02_panic_recover    # panic unwind, recover, re-panic
go run ./examples/03_patterns         # timer, tx rollback, annotation, cache

go run ./exercises/01_cleanup         # cleanup chain + safeParseInt
```

---

## Examples

### [examples/01_defer_mechanics/main.go](examples/01_defer_mechanics/main.go)

LIFO execution order, argument evaluation at registration time, defer modifying named return values, defer in a loop.

### [examples/02_panic_recover/main.go](examples/02_panic_recover/main.go)

Panic stack unwind, `safeDiv` recovery, `safeCall` general wrapper, `mustGetKey` panic-on-missing, selective recovery with re-panic.

### [examples/03_patterns/main.go](examples/03_patterns/main.go)

`elapsed` timer, transaction rollback pattern, error annotation via named return, `sync.Mutex` unlock via defer.

---

## Exercises

### [exercises.md](exercises.md)

---

## Revision checkpoint

### [checkpoint.md](checkpoint.md)

---

## Key takeaways

1. **defer runs on all exit paths** — normal return, early return, and panic.
2. **LIFO**: last registered, first to run.
3. **Arguments** to a deferred call are evaluated at the `defer` statement, not at call time. Use a closure if you need late binding.
4. **Named return values** can be modified by deferred closures — the canonical error-annotation pattern.
5. **panic** is for programming errors and unrecoverable states; return `error` for expected failures.
6. **recover** only works inside a deferred function. Always type-check the recovered value; re-panic if it is not yours.
7. **defer mutex unlock** immediately after `Lock()` is idiomatic and prevents lock leaks.

---

## Cross-references

- **Chapter 13** — Functions: function values and the deferred call mechanism
- **Chapter 14** — Closures: deferred `func()` literals capture variables
- **Chapter 41** — Errors as Values: when to return error vs. panic
- **Chapter 36** — sync package: `sync.Mutex` and correct locking patterns
