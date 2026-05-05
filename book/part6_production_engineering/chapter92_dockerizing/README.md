# Chapter 92 — Dockerizing Go Services

Packaging a Go binary into a container is straightforward but the details matter: image size, build reproducibility, non-root users, graceful shutdown, and health checks all affect production reliability.

## Topics

| # | Topic | Key ideas |
|---|-------|-----------|
| 1 | Multi-stage Dockerfile | Scratch/distroless final image, build args, layer caching |
| 2 | Health checks | `/healthz` liveness, `/readyz` readiness, startup delay |
| E | Production image | Distroless, non-root, secrets via env, graceful SIGTERM |

## Examples

### `examples/01_multistage_build`

Demonstrates multi-stage Docker patterns in Go code:
- Build metadata embedding via `-ldflags`
- Dockerfile layer cache analysis
- Image size comparison: scratch vs distroless vs alpine
- Build reproducibility checks

### `examples/02_health_checks`

HTTP health check server:
- `/healthz` — liveness probe (always returns 200 once up)
- `/readyz` — readiness probe (waits for dependencies)
- `/metrics` — basic process metrics
- Graceful shutdown on SIGTERM with 30s drain window

### `exercises/01_production_image`

Complete production-ready service:
- `Dockerfile` patterns embedded as constants
- Non-root UID/GID in container
- Environment-based configuration
- Secrets detection (warns if secret-like env vars are logged)

## Key Concepts

**Multi-stage build pattern**
```dockerfile
# Stage 1: build
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/app ./cmd/app

# Stage 2: runtime
FROM gcr.io/distroless/static-debian12
COPY --from=builder /bin/app /app
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app"]
```

**Build flags**
- `-ldflags="-s -w"` — strip debug info (-s) and DWARF (-w) → ~30% smaller
- `CGO_ENABLED=0` — static binary, no libc dependency
- `GOOS=linux GOARCH=amd64` — cross-compile from macOS/Windows

**Image size targets**

| Base image | Typical size |
|-----------|-------------|
| ubuntu:22.04 | 80 MB |
| alpine:3.19 | 7 MB |
| gcr.io/distroless/static | 2 MB |
| scratch | 0 MB (binary only) |

**Health check types**

| Probe | Purpose | Failure action |
|-------|---------|----------------|
| Liveness | Is the process alive? | Restart container |
| Readiness | Can it serve traffic? | Remove from LB pool |
| Startup | Has it finished initializing? | Wait before liveness |

## Running

```bash
go run ./book/part6_production_engineering/chapter92_dockerizing/examples/01_multistage_build
go run ./book/part6_production_engineering/chapter92_dockerizing/examples/02_health_checks
go run ./book/part6_production_engineering/chapter92_dockerizing/exercises/01_production_image
```
