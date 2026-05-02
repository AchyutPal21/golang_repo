# Chapter 37 — Exercises

## 37.1 — HTTP error system

Run [`exercises/01_http_errors`](exercises/01_http_errors/main.go).

`HTTPError` with custom `Is()`, `Unwrap()`, and `HTTPStatus()`. Handlers classify errors using `errors.Is` and `errors.As`.

Try:
- Add a `Details map[string]string` field to `HTTPError`. Populate it in `createUser` with field-level validation details. Extract it in the handler and include it in the JSON response.
- Add a `loggable() bool` method to `HTTPError`. Return true for 5xx, false for 4xx. Use it in the handler to decide whether to log to stderr.
- Add `ErrHTTP503` with a `RetryAfter time.Duration`. Implement `Retryable` from example 02 on `HTTPError` when Status == 503.

## 37.2 ★ — Error code registry

Build an `ErrorRegistry` that maps string codes to human-readable messages:

```go
type ErrorRegistry struct { codes map[string]string }
func (r *ErrorRegistry) Register(code, message string)
func (r *ErrorRegistry) New(code string, cause error) *CodedError
```

`CodedError` implements `Error()`, `Unwrap()`, and a custom `Is()` that matches by code. Demonstrate that `errors.Is(wrappedErr, registry.New("E404", nil))` returns true when the chain contains a `CodedError` with code `"E404"`.

## 37.3 ★★ — Aggregate validation error

Design an `AggregateError` that:
- Holds `[]FieldError` (field + message pairs)
- Implements `Error()` that lists all field errors
- Implements `Unwrap() []error` (Go 1.20 multi-unwrap)
- Has `Fields() []FieldError` for structured access

Demonstrate that `errors.As` can extract individual `*FieldError` values from a deeply-wrapped `AggregateError`.
