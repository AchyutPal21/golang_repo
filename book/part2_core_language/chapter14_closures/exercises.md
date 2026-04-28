# Chapter 14 — Exercises

## 14.1 — Thread-safe counter and rate limiter

Run [`exercises/01_counter`](exercises/01_counter/main.go).

Study the implementation of `makeThreadSafeCounter` and `makeRateLimiter`.
Note how `sync.Mutex` is captured in the closure environment.

Try:
- Add a `reset()` function to `makeThreadSafeCounter` that sets the count to 0.
- Modify `makeRateLimiter` to accept a `window time.Duration` and reset automatically.

## 14.2 ★ — Once with error

The standard `sync.Once` does not propagate errors. Implement:

```go
func onceWithError[T any](compute func() (T, error)) func() (T, error)
```

The first call executes `compute`. Subsequent calls return the cached result
(or the cached error if compute failed). It must be concurrency-safe.

## 14.3 ★ — Middleware chain

Implement `compose(middlewares ...Middleware) Middleware` where
`type Middleware func(Handler) Handler`.

The composed middleware should apply them left-to-right:
`compose(A, B, C)(h)` is equivalent to `C(B(A(h)))`.

Test it with a logging middleware, a timing middleware, and a caching middleware.
