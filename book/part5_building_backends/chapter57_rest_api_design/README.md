# Chapter 57 — REST API Design

## What you will learn

- The six REST constraints and why they matter (uniform interface, stateless, cacheable, layered, code-on-demand, client-server)
- Resource naming: nouns for resources, verbs for actions, plural collections (`/articles`, `/articles/{id}`)
- HTTP method semantics: safe (GET/HEAD), idempotent (GET/PUT/DELETE), non-idempotent (POST)
- Status code selection: 200, 201, 204, 400, 404, 405, 422 — when to use each
- `Location` header on 201 Created — pointing the client to the new resource
- `Allow` header on 405 Method Not Allowed — advertising supported methods
- HATEOAS (Level 3 REST): `_links` in responses with `rel`, `href`, and `method`
- State transitions via sub-resources (`POST /articles/{id}/publish`)
- API versioning strategies: URL-prefix (`/v1/`, `/v2/`) vs Accept-header content negotiation
- Pagination strategies: offset/limit vs cursor-based — trade-offs for each

---

## REST Richardson Maturity Model

| Level | What it means |
|---|---|
| **0** | Single URI, one HTTP method (RPC over HTTP) |
| **1** | Multiple URIs (resources), still one method |
| **2** | Correct HTTP methods + status codes |
| **3** | HATEOAS — hypermedia drives application state |

Most production APIs target Level 2. Level 3 adds discoverability but increases response size.

---

## HTTP method semantics

| Method | Safe? | Idempotent? | Typical use |
|---|---|---|---|
| GET | ✓ | ✓ | Read resource or collection |
| HEAD | ✓ | ✓ | Check existence / metadata |
| PUT | ✗ | ✓ | Replace resource in full |
| PATCH | ✗ | ✗ | Partial update |
| DELETE | ✗ | ✓ | Remove resource |
| POST | ✗ | ✗ | Create, or trigger action |

**Safe** = no observable side effects.  
**Idempotent** = calling N times has the same effect as calling once.

---

## Status code quick reference

```
200 OK            — read/update/action succeeded; body contains resource
201 Created       — resource was created; Location header points to it
204 No Content    — success, no body (typical for DELETE)
400 Bad Request   — malformed request (syntax error, invalid JSON)
404 Not Found     — resource does not exist
405 Method Not Allowed — include Allow: GET, POST header
422 Unprocessable Entity — valid syntax, invalid semantics (missing required field)
```

---

## Resource naming conventions

```
/articles              collection
/articles/{id}         single resource
/articles/{id}/publish sub-resource for state transition (POST)
/authors/{id}/articles nested collection (use sparingly)
```

- Use **nouns**, not verbs in paths
- Use **plural** for collections
- Keep nesting ≤ 2 levels deep

---

## HATEOAS response shape

```json
{
  "id": 1,
  "title": "Go Concurrency",
  "_links": [
    {"rel": "self",    "method": "GET",    "href": "/articles/1"},
    {"rel": "update",  "method": "PUT",    "href": "/articles/1"},
    {"rel": "delete",  "method": "DELETE", "href": "/articles/1"},
    {"rel": "publish", "method": "POST",   "href": "/articles/1/publish"}
  ]
}
```

Include conditional links (e.g., `publish` only appears when `published == false`).

---

## Versioning trade-offs

| Strategy | Pros | Cons |
|---|---|---|
| URL prefix `/v1/` | Simple, cache-friendly, easy to route | Version in URI (purists object) |
| Accept header | Stable URI, clean | Hard to cache, harder to debug |
| Query param `?v=1` | Easy to test in browser | Informal, not widely recommended |

URL-prefix versioning is the pragmatic default for most teams.

---

## Pagination trade-offs

| Strategy | Pros | Cons |
|---|---|---|
| Offset / limit | Random access, simple | Page drift on inserts; slow on large offsets |
| Cursor-based | No drift, efficient | Cannot jump to page N |

Use **cursor pagination** for feeds and real-time data. Use **offset pagination** for static reports and admin UIs where random-access is needed.

---

## Examples

| File | Demonstrates |
|---|---|
| `examples/01_rest_principles/main.go` | Resource naming, HTTP semantics, status codes, HATEOAS, idempotency |
| `examples/02_versioning_pagination/main.go` | URL-prefix versioning, Accept-header versioning, offset and cursor pagination |

## Exercise

`exercises/01_api_design/main.go` — Books API with full CRUD, HATEOAS links, cursor pagination, and URL versioning.
