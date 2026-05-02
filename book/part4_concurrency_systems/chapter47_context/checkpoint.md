# Chapter 47 — Revision Checkpoint

## Questions

1. What are the four methods on `context.Context` and what does each return?
2. Why must you always `defer cancel()` even when you cancel the context manually before the defer fires?
3. How does cancellation propagate through a context tree, and what is the direction of propagation?
4. When should you use `context.WithValue` and what type should the key be?
5. What is the difference between `context.Canceled` and `context.DeadlineExceeded`, and how do you distinguish them?

## Answers

1. The four methods are: (1) `Done() <-chan struct{}` — returns a channel that is closed when the context is cancelled or times out; returns `nil` for non-cancellable contexts (like `Background`). (2) `Err() error` — returns `nil` while the context is active; returns `context.Canceled` after manual cancellation; returns `context.DeadlineExceeded` after a timeout. (3) `Deadline() (time.Time, bool)` — returns the absolute deadline and whether one was set. (4) `Value(key any) any` — returns the value associated with `key`, searching up the context tree; returns `nil` if not found.

2. `WithCancel`, `WithTimeout`, and `WithDeadline` all register the new context with its parent so that parent cancellation propagates down. This registration involves a goroutine slot in the parent's internal cancel list and, for timeouts, a timer goroutine. Calling `cancel()` removes the context from this list and stops the timer, freeing those resources. If you never call `cancel()`, the timer goroutine and the parent registration persist until the parent itself is cancelled — which may never happen if it is `context.Background()`. `defer cancel()` guarantees cleanup on all return paths, including early returns and panics.

3. Cancellation propagates **downward only** — from parent to children, never upward. When a parent context is cancelled (whether by calling its cancel function, by a timeout, or by its own parent being cancelled), all descendant contexts are immediately cancelled. Cancelling a child context has no effect on the parent or sibling contexts. This design allows a top-level request context to cancel an entire tree of goroutines with a single call, while individual sub-operations can be cancelled independently without affecting the broader request.

4. Use `context.WithValue` for **request-scoped metadata** that needs to flow implicitly through the call chain without being passed as explicit function arguments — examples: request IDs, trace IDs, authenticated user objects, A/B test flags. The key must be a value of an **unexported custom type** (not `string` or `int`) defined in your package: `type contextKey string; const reqIDKey contextKey = "req_id"`. Using an unexported type prevents key collisions with other packages that might use the same string value. Do not use `WithValue` for optional function parameters or configuration — those should be explicit function arguments.

5. `context.Canceled` is returned by `ctx.Err()` when the context was explicitly cancelled by calling the cancel function returned by `WithCancel`, `WithTimeout`, or `WithDeadline`. `context.DeadlineExceeded` is returned when the context's deadline or timeout elapsed before the context was manually cancelled. You distinguish them with direct equality: `if errors.Is(err, context.Canceled)` or `if errors.Is(err, context.DeadlineExceeded)`. In practice, `Canceled` usually means an upstream caller decided to stop (e.g., HTTP client disconnected), while `DeadlineExceeded` means the operation took too long — the response to each may differ (log warning vs log error, retry vs not).
