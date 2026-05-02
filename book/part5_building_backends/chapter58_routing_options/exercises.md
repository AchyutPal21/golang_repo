# Chapter 58 — Exercises

## 58.1 — Route Table API

Run [`exercises/01_route_table`](exercises/01_route_table/main.go).

Orders + Items nested REST API using Go 1.22 `net/http.ServeMux`. Structured JSON logging middleware wraps all routes. Auth middleware protects write operations. A `GET /routes` introspection endpoint returns the full route table as JSON. 14 test assertions cover CRUD, nested resources, error codes (400/401/404/405/422), and idempotent DELETE.

Try:
- Add a `PATCH /v1/orders/{id}` route that updates only the `status` field.
- Add a `GET /v1/orders/{id}/items/{itemID}` route for fetching a single item.
- Add a `?status=pending` query parameter filter to `GET /v1/orders`.

## 58.2 ★ — Parameterized route constraints

Extend the route table with typed constraints at the router level:

```go
// Register a constraint: {id} must match [0-9]+
register(mux, "GET", "/v1/orders/{id:[0-9]+}", ...)

// Non-matching segment falls through to the next handler or 404:
// GET /v1/orders/abc → 404 (not found, not 400)
// GET /v1/orders/42  → routes correctly
```

Implement constraint checking in the dispatch layer. If a path segment does not match the constraint regex, treat it as a non-match (not a 400 — the URL simply does not match this route).

## 58.3 ★★ — Versioned subrouters

Build an API that serves two complete versions simultaneously:

- `/v1/products/{id}` returns `{id, name, price_cents}`
- `/v2/products/{id}` returns `{id, name, price_dollars, currency}`

Use a `SubRouter` abstraction that mounts an entire handler tree under a path prefix. Each version's subrouter has its own middleware chain (e.g., v1 logs "deprecated" warning, v2 logs normally). A `GET /` endpoint returns a discovery document listing both API versions, their base paths, and their sunset dates.
