# Chapter 64 — Exercises

## 64.1 — Error Catalog

Run [`exercises/01_error_catalog`](exercises/01_error_catalog/main.go).

Orders API with a 9-type error catalog covering every common HTTP error scenario: 404, 422, 409 (conflict + inventory), 401, 403, 402, 429, 503, 500. All errors are RFC 7807-compliant with `type`, `title`, `status`, `detail`, `instance`, and `correlation_id`. Internal errors log their cause with `[ERROR]` but send only `"an unexpected error occurred"` to clients. Retry-After header is set on 429 responses.

Try:
- Add an `errors` extension field to the 422 response containing per-field validation errors.
- Add a `POST /orders/{id}/cancel` endpoint that returns 409 if the order is already cancelled.
- Implement `errors.Is`-based routing in `handleErr` using sentinel errors instead of a type switch.

## 64.2 ★ — Error documentation endpoint

Add a `GET /errors` endpoint to the API that returns the full error catalog as JSON:

```json
[
  {
    "type": "https://api.example.com/errors/not-found",
    "title": "Not Found",
    "status": 404,
    "description": "The requested resource does not exist",
    "example_detail": "article '42' does not exist"
  },
  ...
]
```

This turns the `type` URIs into actual dereferenceable documentation endpoints — a client can `GET` the `type` URI and learn about the error.

## 64.3 ★★ — Error aggregation and observability

Extend the error handler to publish structured error metrics:

- Track per-type error counts in an `atomic.Int64` map
- Expose `GET /metrics/errors` returning error counts per type and per status code
- Add a `X-Error-Count` response header to every 5xx response showing how many 5xx errors have occurred since startup
- Implement an error sampling policy: log 100% of 5xx errors, 10% of 4xx errors (to avoid log flooding on bad-client attacks)

Test with a harness that fires 50 requests with mixed outcomes and verifies the metrics endpoint reflects correct counts.
