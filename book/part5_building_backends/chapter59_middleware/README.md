# Chapter 59 — Middleware

## What you will learn

- The middleware adapter pattern: `func(http.Handler) http.Handler`
- Middleware execution order and how `chain()` composes them
- Why the `ResponseWriter` must be wrapped to capture the status code for logging
- Recovery middleware: catching panics from any inner layer, writing 500, continuing
- Per-request timeout middleware using `context.WithTimeout`
- Token bucket rate limiting: burst, refill rate, per-IP variants
- CORS middleware: preflight `OPTIONS` handling, allowed origins, `Access-Control-*` headers
- Secure headers: `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`
- Passing values through context: typed keys, accessor functions, collision prevention
- Correlation ID: generating, propagating, and reading from context and response headers
- Authentication via context: `bearerAuth` populates context; `requireAuth` gates access
- Role-based access: `requireRole("admin")` reads from context set by auth middleware

---

## The adapter pattern

```go
type MW = func(http.Handler) http.Handler

func logging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        rec := &recorder{ResponseWriter: w, status: 200}
        next.ServeHTTP(rec, r)  // ← call inner chain
        log.Printf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(start))
    })
}
```

Code before `next.ServeHTTP` runs on the **way in**. Code after runs on the **way out**.

---

## Execution order

```
Request → [MW1] → [MW2] → [MW3] → handler
                                        ↓
Response ← [MW1] ← [MW2] ← [MW3] ←────┘
```

```go
func chain(h http.Handler, mws ...MW) http.Handler {
    for i := len(mws) - 1; i >= 0; i-- {
        h = mws[i](h)
    }
    return h
}

// chain(handler, recovery, logging, rateLimit)
// → recovery wraps (logging wraps (rateLimit wraps handler))
// → recovery sees request first; sees response last
```

---

## Standard middleware stack order

```
corrID → recovery → logging → rateLimit → timeout → auth → handler
```

- **corrID** is outermost — ID must be in context before recovery fires (so the error body can include it)
- **recovery** is before logging — logs the final status even when a panic was caught
- **logging** wraps the rate limiter — logs 429 responses
- **timeout** wraps just the handler — health probes can bypass it
- **auth** is inner — extracts user from token; runs after rate limiting

---

## Context value pattern

```go
// 1. Define an unexported typed key to prevent collisions.
type ctxKey int
const keyUser ctxKey = iota

// 2. Write a setter — returns a new *http.Request with the value in context.
func withUser(r *http.Request, u *User) *http.Request {
    return r.WithContext(context.WithValue(r.Context(), keyUser, u))
}

// 3. Write a typed getter — callers don't touch context directly.
func currentUser(r *http.Request) (*User, bool) {
    u, ok := r.Context().Value(keyUser).(*User)
    return u, ok
}

// 4. Middleware sets it; handlers read it.
func authMW(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if u := lookupToken(r.Header.Get("Authorization")); u != nil {
            r = withUser(r, u)
        }
        next.ServeHTTP(w, r)
    })
}
```

---

## Token bucket rate limiter

```go
type bucket struct {
    tokens, max, rate float64
    lastRefill        time.Time
    mu                sync.Mutex
}

func (b *bucket) allow() bool {
    b.mu.Lock()
    defer b.mu.Unlock()
    elapsed := time.Since(b.lastRefill).Seconds()
    b.tokens = min(b.max, b.tokens + elapsed*b.rate)
    b.lastRefill = time.Now()
    if b.tokens < 1 { return false }
    b.tokens--
    return true
}
```

For per-IP limiting, use a `map[string]*bucket` protected by a mutex.

---

## Examples

| File | Demonstrates |
|---|---|
| `examples/01_middleware_stack/main.go` | Recovery, logging, rate limiting, timeout, CORS, secure headers, request counter |
| `examples/02_context_values/main.go` | Typed context keys, correlation ID, auth middleware, role gate, timing |

## Exercise

`exercises/01_middleware_suite/main.go` — Full middleware suite: corrID (outermost), recovery, JSON logger, per-IP rate limiter, bearer auth, require-auth, require-role. Demonstrates correct middleware ordering so correlation ID appears in recovery error bodies.
