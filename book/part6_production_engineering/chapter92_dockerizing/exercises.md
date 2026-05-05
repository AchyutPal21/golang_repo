# Chapter 92 Exercises — Dockerizing Go Services

## Exercise 1 (provided): Production Image Patterns

Location: `exercises/01_production_image/main.go`

A complete production-ready HTTP service demonstrating:
- Build metadata injection via `-ldflags`
- Environment-based configuration with defaults
- Graceful shutdown on SIGTERM/SIGINT
- Liveness and readiness probes
- Secrets detection (warns when secret-like names are logged)
- Dockerfile patterns embedded as comments

## Exercise 2 (self-directed): Health Check Server

Build an HTTP server with production-grade health checks:
- `/healthz` — always 200 once the server is running
- `/readyz` — 503 during startup (first 5s) and during shutdown; 200 otherwise
- `/version` — returns JSON with version, commit, buildTime
- `/metrics` — returns plaintext: `goroutines`, `heap_alloc_bytes`, `gc_count`

The server must handle SIGTERM gracefully with a configurable drain window (env `DRAIN_SECONDS`, default 30).

Acceptance criteria:
- `curl /readyz` returns 503 in the first 5 seconds
- `curl /readyz` returns 200 after warmup
- Sending SIGTERM causes `/readyz` to return 503 while in-flight requests complete

## Exercise 3 (self-directed): Image Size Analyser

Write a Go program that simulates image layer analysis:
- Define a `Layer` struct: `{Digest, CreatedBy string, Size int64}`
- Build a `[]Layer` representing: base image, dependencies, app binary
- Print a table: layer, size (human-readable), cumulative size
- Identify the largest layer and suggest an optimization

## Stretch Goal: Config Validation at Startup

Build a `Config` struct loaded from environment variables with:
- `RequiredEnv(key string) string` — panics with a clear message if missing
- `OptionalEnv(key, defaultVal string) string`
- `SecretEnv(key string) string` — reads the value but returns `[REDACTED]` from `String()`
- A `Validate() error` that checks all required fields before the server starts
