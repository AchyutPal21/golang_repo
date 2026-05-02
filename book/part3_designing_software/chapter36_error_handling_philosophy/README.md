# Chapter 36 — Error Handling Philosophy

> **Part III · Designing Software** | Estimated reading time: 20 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Go's explicit error handling is not a flaw — it is a feature that forces the programmer to think about every failure path. The patterns in this chapter eliminate the common anti-patterns (double logging, swallowed errors, opaque strings) without adding boilerplate.

---

## 36.1 — Sentinel errors and wrapping

**Sentinel errors** are package-level `var` declarations. Callers compare with `errors.Is`:

```go
var ErrNotFound = errors.New("not found")

// Wrap with context, preserving the sentinel:
return fmt.Errorf("fetchUser id=%d: %w", id, ErrNotFound)

// Unwrap anywhere in the chain:
errors.Is(err, ErrNotFound) // true, even through layers of wrapping
```

**Never** compare wrapped errors with `==` — it misses wrapping.

---

## 36.2 — errors.As — extracting typed errors

`errors.As` walks the unwrap chain and finds the first error of a target type:

```go
var ve *ValidationError
if errors.As(err, &ve) {
    fmt.Println("field:", ve.Field)
}
```

Wrapping with `%w` preserves the type through all layers, so `errors.As` works regardless of how deeply the error was wrapped.

---

## 36.3 — The golden rule

**Handle OR propagate. Never both.**

```go
// BAD: logs AND returns — the caller logs again
if err != nil {
    log.Println("error:", err)
    return err
}

// GOOD: add context and propagate upward
if err != nil {
    return fmt.Errorf("loadUser: %w", err)
}

// GOOD: handle at the boundary (top-level handler, main, HTTP handler)
if err != nil {
    log.Println("request failed:", err)
    http.Error(w, "internal error", 500)
}
```

---

## 36.4 — errWriter pattern

When many writes can fail, track the first error and short-circuit:

```go
type errWriter struct{ w io.Writer; err error }

func (ew *errWriter) Write(s string) {
    if ew.err != nil { return }
    _, ew.err = fmt.Fprint(ew.w, s)
}
```

Check `ew.err` once at the end. Used extensively in `encoding/binary`, standard library formatting, and any code that serialises structured data.

---

## 36.5 — Must helper

For initialisation-time operations that cannot fail at runtime:

```go
func Must[T any](v T, err error) T {
    if err != nil { panic(err) }
    return v
}

// At startup: panic is appropriate — the app cannot start without this
cfg := Must(config.Load("config.yaml"))
```

**Never** use `Must` in request-handling paths — a panic crashes the goroutine (and, without recovery, the entire server).

---

## 36.6 — When panic is appropriate

| Situation | Use panic? |
|---|---|
| Programmer error (nil dereference, impossible state) | Yes — it surfaces bugs early |
| Initialisation failure (missing required config) | Yes — with `Must` |
| Expected runtime error (not found, network timeout) | No — return error |
| Request handler error | No — return error; recover at the framework boundary |

---

## 36.7 — Multi-error collection

Collect all errors before returning (e.g., form validation):

```go
var errs []error
if name == "" { errs = append(errs, fmt.Errorf("name: required")) }
if !validEmail(email) { errs = append(errs, fmt.Errorf("email: invalid")) }
return errors.Join(errs...) // nil if empty; unwrappable with Unwrap() []error
```

---

## Running the examples

```bash
cd book/part3_designing_software/chapter36_error_handling_philosophy

go run ./examples/01_wrapping_sentinels     # sentinel errors, %w, errors.Is, errors.As, errors.Join
go run ./examples/02_error_handling_patterns # errWriter, Must, guard clauses, panic/recover, multiErr

go run ./exercises/01_error_propagation     # three-layer stack with boundary classification
```

---

## Key takeaways

1. **Wrap with `%w`** to add context and preserve the sentinel for `errors.Is`/`errors.As`.
2. **Handle OR propagate** — never log and return; pick one site to handle.
3. **`errors.Is`** walks the chain; `==` does not.
4. **`errors.As`** extracts a typed error anywhere in the chain.
5. **errWriter** reduces boilerplate when many sequential writes can fail.
6. **`Must`** is for startup only — never in request paths.
7. **`errors.Join`** collects multiple errors; `Unwrap() []error` iterates them.

---

## Cross-references

- **Chapter 15** — Defer, Panic, Recover: the mechanics underlying panic/recover
- **Chapter 37** — Custom Error Types: implementing `Error()`, `Unwrap()`, `Is()`, `As()`
- **Chapter 35** — Service Layer: error propagation patterns across layers
