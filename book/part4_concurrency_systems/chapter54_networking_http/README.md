# Chapter 54 — Networking II: HTTP/1.1

## What you will learn

- `net/http` server: `ServeMux`, `HandleFunc`, `Handle`
- Request anatomy: method, URL, query params, headers, body
- JSON request/response with `encoding/json`
- Middleware pattern: `func(http.Handler) http.Handler`
- Logging middleware, auth middleware, request-ID injection via context
- Graceful shutdown: `srv.Shutdown(ctx)` drains in-flight requests
- `http.Client` configuration: `Transport`, timeouts, connection pool
- Retry with exponential backoff on 5xx responses
- Concurrent client requests and response body rules

---

## Minimal server

```go
mux := http.NewServeMux()
mux.HandleFunc("/greet", func(w http.ResponseWriter, r *http.Request) {
    name := r.URL.Query().Get("name")
    fmt.Fprintf(w, "Hello, %s!\n", name)
})
http.ListenAndServe(":8080", mux)
```

---

## JSON handler pattern

```go
func createHandler(w http.ResponseWriter, r *http.Request) {
    var req CreateRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    result := process(req)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(result)
}
```

---

## Middleware

```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
    })
}

// Chain: outer(inner(handler))
http.ListenAndServe(":8080", logging(auth(mux)))
```

---

## Graceful shutdown

```go
srv := &http.Server{Handler: mux}
go srv.ListenAndServe()

// On OS signal:
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
srv.Shutdown(ctx)  // stops accepting new connections; waits for existing ones
```

---

## Custom client

```go
client := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConnsPerHost:   10,
        ResponseHeaderTimeout: 5 * time.Second,
    },
}
```

**Always** specify a `Timeout` — the default client has no timeout.

---

## Response body rules

1. **Always** `defer resp.Body.Close()` after a successful `client.Do`.
2. **Always** read the entire body before closing: `io.ReadAll(resp.Body)` or `io.Copy(io.Discard, resp.Body)`. This allows connection reuse.

---

## Examples

| File | Demonstrates |
|---|---|
| `examples/01_http_server/main.go` | ServeMux, JSON handlers, logging+auth middleware, graceful shutdown |
| `examples/02_http_client/main.go` | Custom transport, retry backoff, concurrent requests, body handling |

## Exercise

`exercises/01_rest_api/main.go` — CRUD Todo API with `GET/POST/PUT/DELETE /todos/{id}`, request-ID middleware, logging middleware, 11-case test harness.
