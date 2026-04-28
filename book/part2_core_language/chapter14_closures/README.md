# Chapter 14 — Closures and the Capture Model

> **Part II · Core Language** | Estimated reading time: 22 min | Runnable examples: 3 | Exercises: 1

---

## Why this chapter matters

A closure is what happens when a function literal captures variables from the surrounding scope. Go's closure model is pervasive: every time you write `func()` inside another function, you are creating a closure. Understanding how variables are captured — by reference, not by value — is the key to using goroutines, deferred functions, functional options, middleware, iterators, and the `once` pattern correctly. It is also the explanation for one of the most common Go bugs (the loop-variable capture trap), which Go 1.22 fixed at the language level.

---

## 14.1 — What is a closure?

A closure is a function value that *closes over* variables from its surrounding scope. The function can read and write those variables after the enclosing function has returned.

```go
func makeCounter() func() int {
    count := 0          // lives on the heap, not the stack
    return func() int {
        count++         // captures count — same variable
        return count
    }
}

c := makeCounter()
c() // 1
c() // 2
c() // 3
```

`count` is captured *by reference*: the closure holds a pointer to `count`, not a snapshot of its value at creation time. Modifying `count` inside the closure modifies the same variable the outer function created.

> **Escape analysis**: Go's compiler detects that `count` is referenced by a closure that outlives `makeCounter`'s stack frame, and allocates `count` on the heap automatically. You do not manage this; the compiler handles it.

---

## 14.2 — Independent environments

Each call to the factory produces a fresh closure with its own captured variables:

```go
c1 := makeCounter()
c2 := makeCounter()

c1() // 1
c1() // 2
c2() // 1  — independent; c2 has its own count
```

This is what makes closures useful as stateful objects: they encapsulate state without any struct or global variable.

---

## 14.3 — Capture is by reference

Because capture is by reference, **the closure always sees the current value of the captured variable**, not the value at closure creation time.

```go
x := 10
read  := func() int  { return x }
write := func(v int) { x = v }

read()    // 10
write(42)
read()    // 42 — same x
```

Multiple closures capturing the same variable share it:

```go
count := 0
inc := func() { count++ }
dec := func() { count-- }
get := func() int { return count }

inc(); inc(); inc(); dec()
get() // 2
```

This is intentional in many patterns (shared counter, shared cache). It becomes a bug when it is unintentional — the classic case is the loop-variable capture trap.

---

## 14.4 — The loop-variable capture trap

**Pre-Go 1.22** — all loop iterations shared a single loop variable:

```go
funcs := make([]func(), 3)
for i := 0; i < 3; i++ {
    funcs[i] = func() { fmt.Println(i) } // captures the *same* i
}
// loop ends: i == 3
funcs[0]() // 3 — not 0!
funcs[1]() // 3
funcs[2]() // 3
```

**Fix in all Go versions** — shadow with a new variable:

```go
for i := 0; i < 3; i++ {
    i := i  // new i scoped to loop body
    funcs[i] = func() { fmt.Println(i) }
}
```

**Go 1.22+** — each iteration creates a fresh loop variable automatically:

```go
for i := range 3 {         // Go 1.22: i is a new variable each iteration
    funcs[i] = func() { fmt.Println(i) }
}
funcs[0]() // 0
funcs[1]() // 1
funcs[2]() // 2
```

The Go 1.22 change is the most significant behavioral change in a Go release since 1.0. The old shadowing workaround still works and is the correct approach when targeting pre-1.22.

> See `GODEBUG=loopvar=1` for a per-binary opt-in in Go 1.21, and the Go 1.22 release notes for the full context.

---

## 14.5 — Goroutines and loop capture

The trap is most dangerous with goroutines, which run after the loop finishes:

```go
// Pre-1.22 bug: all goroutines print the same final value of n
for n := 0; n < 5; n++ {
    go func() {
        fmt.Println(n) // captures shared n — likely prints 5 five times
    }()
}
```

**Safe pattern 1** — Go 1.22+ (loop variable is per-iteration).

**Safe pattern 2** — pass as argument (works in all versions):

```go
for n := 0; n < 5; n++ {
    go func(n int) { // n is a parameter, not a captured variable
        fmt.Println(n)
    }(n)
}
```

Passing as an argument copies the value at the point of the goroutine launch. This is explicit and correct in all Go versions.

---

## 14.6 — Lazy evaluation

A closure can defer computation until the result is actually needed:

```go
func lazy[T any](compute func() T) func() T {
    var value T
    computed := false
    return func() T {
        if !computed {
            value = compute()
            computed = true
        }
        return value
    }
}
```

The first call executes `compute`; subsequent calls return the cached result. For a concurrency-safe version, use `sync.Once`:

```go
func onceSafe[T any](compute func() T) func() T {
    var once sync.Once
    var value T
    return func() T {
        once.Do(func() { value = compute() })
        return value
    }
}
```

`sync.Once` guarantees the function runs exactly once even under concurrent calls, with no race condition. This is the standard pattern for lazy singleton initialisation in Go.

---

## 14.7 — Functional options (closure as configuration)

Rob Pike's *functional options* pattern uses closures to build flexible constructors without long parameter lists:

```go
type Option func(*Config)

func WithTimeout(ms int) Option {
    return func(c *Config) { c.timeout = ms }
}

func NewServer(opts ...Option) Server {
    cfg := Config{timeout: 30} // defaults
    for _, opt := range opts {
        opt(&cfg)
    }
    return Server{cfg: cfg}
}

srv := NewServer(WithTimeout(100), WithHost("api.example.com"))
```

Each `WithXxx` function returns a closure that modifies a `*Config`. Callers compose the options they need; the constructor applies them in order. This pattern is used throughout the Go standard library (`http.Server`, `grpc.Dial`, `slog.HandlerOptions`) and third-party packages.

---

## 14.8 — Middleware (closure as wrapper)

Middleware is a closure that wraps a handler and adds cross-cutting behaviour:

```go
type Handler func(string) string

func withLogging(h Handler) Handler {
    return func(input string) string {
        result := h(input)
        fmt.Printf("[log] %q → %q\n", input, result)
        return result
    }
}
```

`withLogging` takes a `Handler` and returns a new `Handler` that logs. The original handler is captured in the closure. Middleware chains are built by composition:

```go
logged := withLogging(withPrefix(">> ", handler))
```

This pattern underlies every Go HTTP middleware package (`net/http`, `gorilla/mux`, `echo`, etc.).

---

## 14.9 — Closure-based iterators

Before Go 1.23's range-over-function (`iter.Seq`), closures were the idiomatic way to build stateful iterators:

```go
func lines(s string) func() (string, bool) {
    parts := strings.Split(s, "\n")
    idx := 0
    return func() (string, bool) {
        for idx < len(parts) {
            line := parts[idx]; idx++
            if line != "" { return line, true }
        }
        return "", false
    }
}

next := lines("alpha\nbeta\ngamma")
for line, ok := next(); ok; line, ok = next() {
    fmt.Println(line)
}
```

Go 1.23 formalises this with `iter.Seq[V]` (`func(yield func(V) bool)`), but the underlying closure mechanics are identical.

---

## 14.10 — The fibonacci generator

Closures can hold multi-variable state:

```go
func fibonacci() func() int {
    a, b := 0, 1
    return func() int {
        v := a
        a, b = b, a+b
        return v
    }
}

fib := fibonacci()
// 0 1 1 2 3 5 8 13 21 34 ...
```

Each call advances the sequence. Two independent generators have independent state.

---

## 14.11 — When not to use closures

Closures have costs:

| Issue | Detail |
|---|---|
| **Heap allocation** | Captured variables escape to the heap, adding GC pressure |
| **Indirection** | Closure calls are indirect function calls — not inlined |
| **Shared mutation** | Multiple closures sharing a variable create implicit coupling |
| **Race conditions** | Captured variables shared across goroutines need synchronisation |

Prefer a struct with methods when:
- The state is complex (multiple fields with relationships)
- The type needs to implement an interface
- The type is exposed in a public API
- The behaviour needs to be tested independently

Use closures when:
- The state is simple (one or two variables)
- The lifetime is short (a request, a test)
- The behaviour is not part of an API contract
- You are building generic utilities (middleware, lazy, once)

---

## Running the examples

```bash
cd book/part2_core_language/chapter14_closures

go run ./examples/01_closure_basics   # counter, accumulator, greeter, toggle
go run ./examples/02_capture_model    # loop capture, Go 1.22, goroutines
go run ./examples/03_closure_patterns # lazy, middleware, functional options, iterator

go run ./exercises/01_counter         # thread-safe counter + rate limiter
```

---

## Examples

### [examples/01_closure_basics/main.go](examples/01_closure_basics/main.go)

Demonstrates independent closure environments via `makeCounter`, `makeAccumulator`, `makeGreeter`, and a generic `toggleFactory`.

### [examples/02_capture_model/main.go](examples/02_capture_model/main.go)

Demonstrates the loop-variable capture trap and its fixes, goroutine capture with `sync.WaitGroup`, shared mutable state, and pointer capture.

### [examples/03_closure_patterns/main.go](examples/03_closure_patterns/main.go)

Demonstrates production patterns: `lazy`, `onceSafe`, middleware composition, functional options, a closure-based line iterator, and a Fibonacci generator.

---

## Exercises

### [exercises.md](exercises.md)

---

## Revision checkpoint

### [checkpoint.md](checkpoint.md)

---

## Key takeaways

1. A closure captures variables **by reference** — it shares the variable with the outer scope, not a copy of its value.
2. Each factory call returns a closure with its **own independent environment**.
3. The **loop-variable trap** is the #1 Go gotcha: pre-1.22, all loop iterations shared one variable. Go 1.22 fixes it per-iteration. The `i := i` shadow workaround works in all versions.
4. **Goroutine capture** is the dangerous form: pass loop variables as arguments to goroutines when targeting pre-1.22.
5. **`sync.Once`** is the concurrency-safe lazy initialisation primitive; use it instead of a home-grown `computed bool`.
6. **Functional options** and **middleware** are idiomatic production patterns that are entirely built on closures.

---

## Cross-references

- **Chapter 13** — Functions: First-Class Citizens: function values and factories
- **Chapter 15** — defer, panic, recover: deferred functions are closures
- **Chapter 21** — Methods: when to use a struct instead of closures
- **Chapter 36** — sync package: `sync.Once`, `sync.Mutex`, and concurrency-safe state
