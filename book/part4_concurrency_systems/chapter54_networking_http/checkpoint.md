# Chapter 54 — Revision Checkpoint

## Questions

1. What is the middleware pattern in Go's `net/http`, and how do you chain multiple middlewares?
2. Why must you always set a `Timeout` on `http.Client`, and what happens if you don't?
3. What are the two rules for HTTP response bodies, and what breaks if you violate either?
4. What does `srv.Shutdown(ctx)` do differently from simply calling `srv.Close()`?
5. How does `http.ServeMux` route requests, and what is the difference between a pattern ending in `/` and one without?

## Answers

1. The middleware pattern wraps an `http.Handler` with another `http.Handler`: `func(http.Handler) http.Handler`. The wrapper calls `next.ServeHTTP(w, r)` to invoke the inner handler, with before/after code for logging, auth, panic recovery, etc. Chaining is function composition: `outer(middle(inner(mux)))`. Each middleware is applied inside-out — the outermost middleware runs first on the request and last on the response. In practice, a `Chain(middlewares ...func(http.Handler) http.Handler)` helper reduces nesting.

2. `http.DefaultClient` (and any `http.Client` with no `Timeout` set) has no deadline on the full request lifecycle. A slow server that sends headers but never sends the body will hold a goroutine open forever — consuming a connection and a goroutine stack. In a high-throughput service, a handful of slow upstream servers can exhaust the connection pool and then the goroutine pool, causing all requests to hang. `Timeout` enforces the maximum duration from request start to response body close, including TCP connect, TLS handshake, response headers, and response body.

3. Rule 1: **always `defer resp.Body.Close()`** — the body must be closed even if you don't read it, to release the underlying TCP connection back to the pool. Rule 2: **always read the entire body before closing** — the transport only returns the connection to the pool if the body is fully consumed; a partial read followed by `Close()` causes the connection to be abandoned rather than reused. Violating Rule 1 leaks the connection forever. Violating Rule 2 causes connection thrashing — new TCP connections are created instead of reusing pooled ones, increasing latency and file descriptor usage.

4. `srv.Shutdown(ctx)` stops accepting new connections immediately (closes the listener), then waits for all active handlers to return — bounded by `ctx`'s deadline. In-flight requests continue processing normally. `srv.Close()` immediately closes all connections — active handlers are interrupted, potentially returning partial responses or causing `write: broken pipe` errors. For a graceful shutdown that allows in-flight requests to complete (e.g., finishing a database write or sending a response), use `Shutdown`. Use `Close` only when you need an immediate hard stop (e.g., during a fatal error).

5. `http.ServeMux` uses a longest-prefix match. A pattern `"/foo"` (no trailing slash) matches only the exact path `/foo`. A pattern `"/foo/"` (trailing slash) matches `/foo/` and any path starting with `/foo/` — it is a subtree pattern. When a request arrives, the mux finds the longest matching pattern: `/foo/bar/baz` matches `/foo/` (subtree) rather than a shorter pattern like `/foo` (exact). Special case: the mux automatically redirects `/foo` to `/foo/` if only the subtree pattern `/foo/` is registered, unless the exact `/foo` pattern is also registered. This allows both exact and subtree routing to coexist without conflicts.
