# Chapter 57 — Exercises

## 57.1 — Books API

Run [`exercises/01_api_design`](exercises/01_api_design/main.go).

Full CRUD REST API for a Books resource with HATEOAS `_links`, cursor-based pagination, and URL versioning (`/v1` returns title+author, `/v2` adds ISBN and year). 13 test assertions cover all HTTP method semantics, status codes (200/201/204/400/404/405/422), idempotent DELETE, and the `Location` header on creation.

Try:
- Add a `GET /v2/books/{id}` endpoint that returns a richer `BookV2` response.
- Add a `PATCH /v1/books/{id}` endpoint — how does its semantics differ from `PUT`?
- Extend the `_links` to include a `reviews` rel pointing to a `GET /v1/books/{id}/reviews` sub-resource.

## 57.2 ★ — Content negotiation + deprecation headers

Extend the Books API with full content negotiation:

- `Accept: application/vnd.books.v1+json` → v1 response (title, author)
- `Accept: application/vnd.books.v2+json` → v2 response (title, author, isbn, year)
- When v1 is requested, include a `Deprecation: true` response header and a `Sunset: <date>` header announcing when v1 will be removed.
- When an unsupported media type is requested, return `406 Not Acceptable`.

## 57.3 ★★ — Rate-limited, self-documented API

Build a fully self-documented REST API:

- `GET /` returns a JSON index of all available endpoints (path, method, description, version)
- `GET /openapi.json` returns a minimal OpenAPI 3.0 document generated at startup from your route table
- Add per-IP rate limiting (20 req/min) using a token bucket; return `429 Too Many Requests` with a `Retry-After` header when the bucket is empty
- Add an `ETag` header to single-resource GET responses (hash of the serialized body); handle `If-None-Match` and return `304 Not Modified` when the ETag matches

Verify with a harness that fires 25 requests from the same IP and asserts 20 succeed (200) and 5 are rejected (429).
