# Chapter 58 — Routing Options

## What you will learn

- Go 1.22 `net/http.ServeMux` enhancements: method-qualified patterns, `{name}` path parameters, `{name...}` wildcard tails
- `r.PathValue("name")` — extracting typed path segments at the handler level
- Route specificity: longest-match wins (exact > wildcard > subtree)
- Automatic 405 Method Not Allowed from the stdlib mux (Go 1.22+)
- Middleware chaining: `chain(handler, mw1, mw2, ...)` — right-to-left application
- Route groups: shared prefix and middleware via a `Group` wrapper
- Named routes and reverse URL generation
- When to reach for a third-party router (chi, gin, echo) vs staying in stdlib

---

## Go 1.22 routing syntax

```go
mux := http.NewServeMux()

// Method + path: only matches GET requests to /users
mux.HandleFunc("GET /users", listUsers)

// Path parameter: {id} captures one path segment
mux.HandleFunc("GET /users/{id}", getUser)

// Wildcard tail: {path...} captures the rest of the URL
mux.HandleFunc("GET /files/{path...}", serveFile)

// Extract in handler:
id := r.PathValue("id")
```

Before Go 1.22, `ServeMux` matched only by path prefix, had no path parameters, and did not distinguish methods. Every handler had to manually check `r.Method`.

---

## Route specificity (longest match wins)

```
GET /files/index.html       ← exact match
GET /files/{path...}        ← wildcard (fallback)
```

If both are registered, `GET /files/index.html` always routes to the exact handler.

---

## Middleware chain

```go
type middleware func(http.Handler) http.Handler

func chain(h http.Handler, mws ...middleware) http.Handler {
    for i := len(mws) - 1; i >= 0; i-- {
        h = mws[i](h)
    }
    return h
}

// Usage: first middleware is outermost (sees request first).
mux.Handle("GET /users", chain(handler, logging, authRequired))
```

---

## When to use a third-party router

| Need | stdlib 1.22 | Third-party (chi/gin/echo) |
|---|---|---|
| Method routing | ✓ | ✓ |
| Path parameters | ✓ | ✓ |
| Route groups | Manual | Built-in |
| Middleware per group | Manual | Built-in |
| Named routes + URL generation | Manual | Built-in |
| Regex constraints on params | ✗ | chi: ✓ |
| Request validation / binding | ✗ | gin/echo: ✓ |
| Performance (10k+ routes) | Good | Slightly better (trie) |

**Recommendation**: use stdlib `net/http` for APIs with ≤ ~20 routes. Add a router library when you need route groups with per-group middleware, regex constraints, or automatic request binding.

---

## Third-party routers at a glance

| Library | Approach | Best for |
|---|---|---|
| **chi** | Stdlib-compatible, composable middleware | APIs that want to stay close to stdlib |
| **gin** | Fast radix-tree router, binding, validation | High-throughput APIs with rich binding |
| **echo** | Similar to gin, clean middleware API | Full-featured web APIs |
| **gorilla/mux** | Mature, feature-rich, slower | Legacy projects |

All of these are wrappers over `http.Handler` — middleware and handlers written for stdlib work with all of them.

---

## Examples

| File | Demonstrates |
|---|---|
| `examples/01_stdlib_routing/main.go` | Go 1.22 method patterns, `{id}`, `{path...}`, auto-405, middleware chaining |
| `examples/02_router_patterns/main.go` | Custom router: route groups, named routes, URL generation, per-group auth middleware |

## Exercise

`exercises/01_route_table/main.go` — Orders + Items nested API using Go 1.22 ServeMux with structured logging middleware, auth middleware on write routes, and a `GET /routes` introspection endpoint.
