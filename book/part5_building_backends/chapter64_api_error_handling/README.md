# Chapter 64 — API Error Handling

## What you will learn

- RFC 7807 "Problem Details for HTTP APIs" — the standard error envelope for JSON APIs
- `Content-Type: application/problem+json` — the IANA-registered media type for problem details
- Problem Detail fields: `type` (URI), `title`, `status`, `detail`, `instance`
- Extension fields: `errors` array for validation problems, `correlation_id` for tracing
- Building an error catalog with stable `type` URIs that clients can program against
- The error-middleware pattern: handlers return `error`, a central adapter converts to Problem Details
- Typed API errors carrying status code + problem type — inspected with `errors.As`
- Wrapping internal errors (`cause`) so they are logged but not exposed to clients
- Source location capture with `runtime.Caller` for internal error debugging

---

## RFC 7807 format

```json
Content-Type: application/problem+json

{
  "type":     "https://api.example.com/errors/not-found",
  "title":    "Resource Not Found",
  "status":   404,
  "detail":   "Article with id 42 does not exist",
  "instance": "/articles/42"
}
```

- `type` — a stable URI identifying the problem class (clients check this, not `title`)
- `title` — human-readable summary; stable for a given `type`
- `status` — matches the HTTP status code
- `detail` — specific to this occurrence (may change between requests)
- `instance` — the request URI that triggered this error

---

## Error catalog pattern

```go
const typeBase = "https://api.example.com/errors/"

type APIError struct {
    Status  int
    ErrType string // appended to typeBase
    Title   string
    Detail  string
    Cause   error  // internal; never sent to client
}

func (e *APIError) Error() string { return e.Detail }
func (e *APIError) Unwrap() error { return e.Cause }
```

Define constructors for each error type:

```go
func ErrNotFound(resource, id string) *APIError { ... }
func ErrValidation(detail string) *APIError      { ... }
func ErrInternal(cause error) *APIError          { ... }
```

---

## Handler adapter (return-error pattern)

```go
type Handler func(w http.ResponseWriter, r *http.Request) error

func Adapt(h Handler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if err := h(w, r); err != nil {
            handleErr(w, r, err)  // central error dispatcher
        }
    }
}
```

Handlers focus on the happy path; all error conversion lives in `handleErr`.

---

## Internal vs client-visible errors

```
5xx errors: log cause internally, send generic message to client
4xx errors: send specific detail to client (it's their fault)

if ae.Status >= 500 && ae.Cause != nil {
    log.Printf("internal error: %v at %s:%d", ae.Cause, ae.file, ae.line)
}
// Problem sent to client:
{
  "type": "https://api.example.com/errors/internal-error",
  "detail": "an unexpected error occurred"  ← generic
}
```

Never expose database errors, stack traces, or internal hostnames to clients.

---

## Validation extension

```go
type ValidationProblem struct {
    Problem
    Errors []FieldError `json:"errors"`
}

type FieldError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}
```

Clients receive per-field validation errors in a single response rather than fail-fast.

---

## Examples

| File | Demonstrates |
|---|---|
| `examples/01_rfc7807/main.go` | Problem Details format, error catalog, domain error → Problem conversion |
| `examples/02_error_middleware/main.go` | Handler-returns-error pattern, typed APIError, central handleErr, source location logging |

## Exercise

`exercises/01_error_catalog/main.go` — Orders API with 9-type error catalog (404, 422, 409, 401, 403, 402, 429, 503, 500), correlation IDs in all error responses, and internal errors logged but not exposed.
