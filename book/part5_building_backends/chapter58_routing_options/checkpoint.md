# Chapter 58 — Routing Options

## Questions

1. What two capabilities did Go 1.22 add to `net/http.ServeMux` that previously required a third-party router?
2. How does Go 1.22's `ServeMux` resolve a conflict when two patterns both match the same URL, one exact and one wildcard?
3. Why is middleware applied in reverse order when building a chain? What does "outermost" middleware mean in practice?
4. What does a route group provide, and how would you implement one without a third-party library?
5. When should you prefer a third-party router like chi or gin over the stdlib `ServeMux`?

## Answers

1. Go 1.22 added **(1) method-qualified patterns** — you can write `"GET /users"` as the pattern string, and the mux matches only that HTTP method, automatically returning `405 Method Not Allowed` with an `Allow` header for other methods. **(2) Named path parameters** — `{id}` in a pattern captures one path segment, extractable at the handler with `r.PathValue("id")`. Before 1.22, every handler had to parse `r.URL.Path` manually and check `r.Method` itself.

2. `ServeMux` uses **longest-match wins**: a more specific (longer) pattern takes precedence. An exact pattern like `GET /files/index.html` is longer than the wildcard `GET /files/{path...}`, so the exact handler is called for that specific path, and the wildcard handles all other paths under `/files/`. If two patterns are the same length and both match, the mux panics at registration time to prevent ambiguity.

3. Middleware is applied in reverse order because each `mws[i](h)` wraps `h` — the last call returns the outermost wrapper. If you apply `[logging, auth]` in order from index 0 down, `auth` would be outer and `logging` inner. By iterating from `len(mws)-1` to `0`, `logging` ends up as the outermost layer, meaning it executes first on the way in (wraps `auth` which wraps the handler). "Outermost" means it receives the raw request before any other middleware and can intercept the response after all inner layers have run.

4. A **route group** is a prefix + shared middleware that applies to a set of routes without repeating the prefix on every `HandleFunc` call. Implementation without a library: create a `Group` struct that holds a pointer to the mux, the shared prefix string, and a slice of middlewares. Its `GET`/`POST`/etc. methods prepend the prefix and wrap the handler with the group's middlewares before delegating to the underlying mux. This is exactly what libraries like chi expose as `r.Route("/api/v1", func(r chi.Router) {...})`.

5. Reach for a third-party router when you need: **(1) nested route groups** with per-group middleware applied declaratively, **(2) regex constraints on path parameters** (e.g., `{id:[0-9]+}`), **(3) automatic request binding and validation** (gin/echo decode JSON/form into structs and run validators), or **(4) a large route table** (50+ routes) where the trie-based radix router gives measurably better dispatch performance. For an API with 10–20 routes and no special constraints, stdlib 1.22 is sufficient and avoids a dependency.
