# Chapter 59 — Middleware

## Questions

1. What is the middleware adapter pattern in Go, and why is the `ResponseWriter` wrapped before calling `next.ServeHTTP`?
2. Explain what "outermost middleware executes first on the request and last on the response" means, and give a concrete example of why ordering matters.
3. Why should correlation ID middleware be placed _before_ (i.e., wrap) recovery middleware, rather than after it?
4. What is a token bucket rate limiter, and how does it differ from a fixed window (request counter) rate limiter?
5. Why must context keys be unexported custom types rather than strings?

## Answers

1. The **middleware adapter pattern** is `func(http.Handler) http.Handler` — a function that takes a handler, wraps it, and returns a new handler. The returned handler executes setup code, calls `next.ServeHTTP`, and then executes teardown code. The `ResponseWriter` must be wrapped (e.g., in a `statusRecorder` struct) because `http.ResponseWriter` does not expose the status code after `WriteHeader` is called — the wrapper captures it in a field so logging middleware can read it after the inner chain completes.

2. In a chain `[corrID, recovery, logging, rateLimit, handler]`, `corrID` wraps everything: its code before `next.ServeHTTP` runs first (generates the ID, sets header), and its code after (if any) runs last. `handler`'s code runs last on the way in and first on the way out. Ordering matters critically for recovery: if logging is _outside_ recovery, the logging middleware sees the final status code including 500 responses from panics. If logging were _inside_ recovery, a panic in the handler would unwind through recovery (which writes 500) but logging would never see the status because its `next.ServeHTTP` line panicked.

3. If recovery is outermost, it catches panics and writes a 500 error body — but the correlation ID has not yet been set in the context (correlation ID middleware never ran). The error body cannot include the correlation ID. By placing `corrIDMW` outermost (wrapping recovery), the correlation ID is set in context _first_, then recovery runs _inside_ it. When a panic occurs, recovery catches it and can read `getCorrID(r)` from context to include in the error response. This is the key: the outermost middleware's `before` code always runs before any inner code, including recovery's deferred function.

4. A **token bucket** maintains a pool of tokens that refills at a constant rate (e.g., 2 tokens/second) up to a maximum burst size. Each request consumes one token; if the bucket is empty, the request is rejected. This allows controlled bursting — a client can fire `burst` requests immediately before the limiter kicks in, then sustains at the refill rate. A **fixed window** counter resets to zero at each window boundary (e.g., "100 requests per minute"). The problem with fixed windows is the "double burst" — a client can fire 100 requests at the end of minute N and 100 at the start of minute N+1, sending 200 requests in under two seconds. Token buckets don't have this vulnerability because the bucket capacity bounds the instantaneous burst regardless of clock alignment.

5. Context uses `any` keys — if two packages both use `"userID"` as a key, they silently overwrite each other. Unexported custom types (e.g., `type ctxKey int`) are guaranteed unique per package: `ctxKey(0)` from package A and `ctxKey(0)` from package B are different runtime types, so `ctx.Value(ctxKey(0))` from A cannot be read by B. This prevents accidental cross-package collisions in any middleware chain that composes code from multiple packages.
