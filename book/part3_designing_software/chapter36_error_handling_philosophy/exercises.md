# Chapter 36 — Exercises

## 36.1 — Three-layer error propagation

Run [`exercises/01_error_propagation`](exercises/01_error_propagation/main.go).

A simulated HTTP stack classifies errors at the boundary using `errors.Is`.

Try:
- Add a `RateLimitError` struct with a `RetryAfter time.Duration` field. Return it from the service when a user makes too many requests. Use `errors.As` in the handler to extract `RetryAfter` and include it in the response.
- Add a `DatabaseError` sentinel. Wrap it in `PostRepo.FindByID` for odd-numbered IDs. In the handler, map it to HTTP 503 with a `Retry-After` header.
- Add a `withContext(ctx string, err error) error` helper that prepends a context string to the error message. Show that `errors.Is` still works through it.

## 36.2 ★ — errWriter for CSV export

Build a `CSVWriter` using the errWriter pattern. It should have:

```go
func (w *CSVWriter) WriteHeader(cols ...string)
func (w *CSVWriter) WriteRow(vals ...string)
func (w *CSVWriter) Flush() error
```

If any write fails, subsequent calls are no-ops and `Flush` returns the first error.

## 36.3 ★★ — Error taxonomy

Design a three-level error taxonomy for an API:

```
APIError (base type, contains Code int and Message string)
├── ClientError   (4xx — caller's fault; safe to expose message)
└── ServerError   (5xx — our fault; message is internal only)
```

Implement `errors.Is` by comparing `Code`, and implement a `HTTPStatus() int` method on each type. Demonstrate wrapping and classification in a handler.
