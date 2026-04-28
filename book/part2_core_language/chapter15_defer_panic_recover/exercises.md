# Chapter 15 — Exercises

## 15.1 — Resource cleanup chain

Run [`exercises/01_cleanup`](exercises/01_cleanup/main.go).

`runPipeline` opens a variable number of resources and ensures each one opened
is closed in LIFO order even when a later open fails. Study how the slice-based
cleanup avoids the "defer in loop" trap.

Try:
- Add a fourth resource `"flaky"` that opens successfully but `Use()` returns
  an error. Verify the resources are still closed.
- Rewrite `runPipeline` using individual defer statements instead of the slice.
  Why does the slice approach generalise better?

## 15.2 ★ — Panic boundary for HTTP handlers

Write `recoverMiddleware(next http.Handler) http.Handler` that catches any panic
in `next`, logs the stack trace, and writes `500 Internal Server Error` to the
response — without crashing the server.

Use `net/http/httptest` to write a test that verifies a panicking handler returns
500 instead of crashing.

## 15.3 ★ — defer benchmark

Benchmark the cost of `defer mu.Unlock()` vs. a direct `mu.Unlock()` call in a
hot loop. Use `testing.B`. What does `go test -benchmem` reveal?
(Hint: in Go 1.14+ open-coded defer, the difference is sub-nanosecond in
non-loop, non-heap contexts.)
