# Chapter 92 Checkpoint — Dockerizing Go Services

## Concepts to know

- [ ] Why use a multi-stage Dockerfile instead of a single stage?
- [ ] What do `-ldflags="-s -w"` do? What is the trade-off?
- [ ] What is `CGO_ENABLED=0`? When would you need CGO in production?
- [ ] What is the difference between `scratch`, `distroless`, and `alpine`?
- [ ] What is the difference between a liveness probe and a readiness probe?
- [ ] Why should containers run as non-root?
- [ ] What is the graceful shutdown sequence in Go? Why does SIGTERM handling matter?
- [ ] How do you embed build metadata (version, commit) into a Go binary?
- [ ] Name three ways secrets should NOT reach a container image.

## Code exercises

### 1. Graceful shutdown

Write a Go HTTP server that:
- Listens on `:8080`
- Handles `SIGTERM` and `SIGINT` with `signal.NotifyContext`
- Calls `server.Shutdown(ctx)` with a 30-second timeout
- Returns 503 from `/readyz` during the shutdown window

### 2. Build metadata

Write a `version` package with `var (Version, Commit, BuildTime string)` and a `func Print()`. Show the `go build -ldflags` invocation that injects the values.

### 3. Layer cache optimization

Given a Dockerfile that copies all source before `go mod download`, explain what cache is invalidated on every code change and rewrite it to maximize cache hits.

## Quick reference

```bash
# Build production image
docker build --target builder -t myapp:latest .

# Inspect layers
docker history myapp:latest

# Check binary for CGO dependencies
file ./myapp          # should say "statically linked"
ldd ./myapp           # should say "not a dynamic executable"

# Run with non-root user
docker run --user 65534:65534 myapp:latest

# Test health check
curl -s http://localhost:8080/healthz
curl -s http://localhost:8080/readyz
```

## Expected answers

1. Multi-stage keeps build tools (Go toolchain, git) out of the final image, reducing attack surface and image size.
2. `-s` strips the symbol table; `-w` removes DWARF debug info. Trade-off: harder to symbolicate crash stacks.
3. `CGO_ENABLED=0` produces a static binary with no libc linkage. You need CGO for packages like `sqlite3`, `libssl`, or system call wrappers.
4. `scratch` is empty (smallest, no shell or CA certs). `distroless` adds CA certs + tzdata. `alpine` adds a full shell and package manager.
5. Liveness: kill and restart the container if it fails. Readiness: stop routing traffic but keep the container alive.
6. Non-root limits blast radius of a container escape — attacker can't write to `/` or bind to ports < 1024.
7. `signal.NotifyContext` → `server.Shutdown(ctx)` → wait for in-flight requests to drain. SIGTERM is sent by Kubernetes before pod deletion.
8. `go build -ldflags="-X main.Version=1.2.3 -X main.Commit=$(git rev-parse --short HEAD)"`.
9. Secrets must not be: baked into image layers, in ENV in the Dockerfile, or in source code. Use runtime env vars, Kubernetes Secrets, or a secrets manager.
