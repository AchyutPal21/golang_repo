# Chapter 54 — Exercises

## 54.1 — REST API

Run [`exercises/01_rest_api`](exercises/01_rest_api/main.go).

Full CRUD API for a Todo resource. 11 test cases exercise create, list, get, update, delete, not-found, empty-title validation, and method-not-allowed. Request-ID and logging middleware wrap all routes.

Try:
- Add a `GET /todos?done=true` filter that returns only completed todos.
- Add a `Content-Type` middleware that returns `415 Unsupported Media Type` for non-JSON POST/PUT bodies.
- Run with `-race` to confirm no data races in the store.

## 54.2 ★ — HTTP proxy

Build a reverse proxy that forwards requests to a backend, adds a `X-Proxy-Via: golang-proxy` response header, and logs each proxied request. Use `httputil.ReverseProxy` or implement manually with `http.Client`.

Test by running a real httptest.Server as the backend and asserting the custom header appears on every response.

## 54.3 ★★ — HTTP file server with range support

Build an HTTP file server that:
- Serves files from a directory
- Supports `Range: bytes=0-1023` for partial content (`206 Partial Content`)
- Returns `304 Not Modified` when `If-None-Match` matches the file's ETag (MD5 hash of content)
- Returns `404` for missing files and `400` for malformed range headers

Use only `net/http` and `os` — no `http.FileServer`.
