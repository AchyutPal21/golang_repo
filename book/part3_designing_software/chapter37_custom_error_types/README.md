# Chapter 37 — Custom Error Types

> **Part III · Designing Software** | Estimated reading time: 20 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Custom error types let callers inspect, classify, and extract structured information from errors — not just read a string. This chapter covers the complete toolkit: `Error()`, `Unwrap()`, custom `Is()`, custom `As()`, error behaviour interfaces, and domain error taxonomies.

---

## 37.1 — When to use a custom error type

| Use | Type |
|---|---|
| Simple "this thing happened" | `errors.New("...")` sentinel |
| Carries metadata (field name, resource ID) | struct with `Error()` |
| Wraps a cause | struct with `Error()` + `Unwrap()` |
| Matched by category/code, not pointer | struct with custom `Is()` |
| Multiple errors in one | `errors.Join(...)` |

---

## 37.2 — Implementing the error interface

```go
type NotFoundError struct {
    Resource string
    ID       any
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s with id %v not found", e.Resource, e.ID)
}
```

Always use a **pointer receiver** for error types — `errors.As` looks for `*T`, not `T`.

---

## 37.3 — Unwrap() — enabling the error chain

```go
type OperationError struct {
    Op    string
    Cause error
}

func (e *OperationError) Unwrap() error { return e.Cause }
```

With `Unwrap()`, `errors.Is` and `errors.As` walk through `OperationError` to find inner types.

---

## 37.4 — Custom Is() — match by value, not pointer

```go
type APIError struct{ Code int; Message string }

func (e *APIError) Is(target error) bool {
    t, ok := target.(*APIError)
    if !ok { return false }
    return e.Code == t.Code
}

var ErrUnauthorised = &APIError{Code: 401}

// Now two different *APIError values with Code 401 match:
errors.Is(wrappedErr, ErrUnauthorised) // true if chain contains any *APIError{Code:401}
```

---

## 37.5 — Error behaviour interfaces

Define interfaces for error behaviour rather than checking concrete types:

```go
type Retryable interface {
    Retryable() bool
    RetryAfter() time.Duration
}

func shouldRetry(err error) (bool, time.Duration) {
    var r Retryable
    if errors.As(err, &r) && r.Retryable() {
        return true, r.RetryAfter()
    }
    return false, 0
}
```

`NetworkError`, `RateLimitError`, and any future error type can implement `Retryable` without modifying `shouldRetry`.

---

## 37.6 — The typed nil trap

```go
func find(fail bool) error {
    var err *NotFoundError    // typed nil
    if fail { err = &NotFoundError{...} }
    return err                // WRONG if !fail: non-nil interface wrapping nil pointer
}
```

Fix: `return nil` explicitly; never return a typed nil pointer as an `error`.

---

## Running the examples

```bash
cd book/part3_designing_software/chapter37_custom_error_types

go run ./examples/01_error_types       # struct errors, Unwrap, custom Is, typed nil trap
go run ./examples/02_error_interfaces  # behaviour interfaces, domain taxonomy, code-based matching

go run ./exercises/01_http_errors      # HTTPError with Is(), As(), HTTPStatus(); handler classification
```

---

## Key takeaways

1. **Pointer receiver** on `Error()` — `errors.As` finds `*T`, not `T`.
2. **`Unwrap()`** lets `errors.Is`/`errors.As` look through your type.
3. **Custom `Is()`** enables matching by code/category rather than pointer identity.
4. **Behaviour interfaces** (`Retryable`, `Categorised`) decouple retry logic from error types.
5. **Never return a typed nil** as `error` — always return bare `nil`.

---

## Cross-references

- **Chapter 36** — Error Handling Philosophy: `errors.Is`, `errors.As`, golden rule
- **Chapter 22** — Interfaces: the typed nil trap lives here
- **Chapter 35** — Service Layer: domain errors propagate up through services
