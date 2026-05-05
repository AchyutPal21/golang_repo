// FILE: book/part6_production_engineering/chapter92_dockerizing/exercises/01_production_image/main.go
// CHAPTER: 92 — Dockerizing Go Services
// EXERCISE: Complete production-ready service — build metadata, env config,
//           graceful shutdown, secrets detection, and health probes.
//
// Run:
//   go run ./book/part6_production_engineering/chapter92_dockerizing/exercises/01_production_image

package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// BUILD METADATA
// ─────────────────────────────────────────────────────────────────────────────

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// ─────────────────────────────────────────────────────────────────────────────
// CONFIGURATION
// ─────────────────────────────────────────────────────────────────────────────

type Config struct {
	Port        string
	LogLevel    string
	DatabaseURL string // secret — redacted in String()
	AppEnv      string
	DrainSecs   string
}

func loadConfig() Config {
	return Config{
		Port:        getEnv("PORT", "8080"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost/app"),
		AppEnv:      getEnv("APP_ENV", "development"),
		DrainSecs:   getEnv("DRAIN_SECONDS", "30"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func (c Config) String() string {
	return fmt.Sprintf("port=%s log_level=%s app_env=%s database_url=[REDACTED]",
		c.Port, c.LogLevel, c.AppEnv)
}

// ─────────────────────────────────────────────────────────────────────────────
// SECRETS DETECTION
// ─────────────────────────────────────────────────────────────────────────────

var secretKeywords = []string{
	"password", "passwd", "secret", "token", "key", "auth",
	"credential", "private", "cert", "api_key",
}

func detectSecretEnv() []string {
	var found []string
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		lower := strings.ToLower(parts[0])
		for _, kw := range secretKeywords {
			if strings.Contains(lower, kw) {
				found = append(found, parts[0]+"=[REDACTED]")
				break
			}
		}
	}
	return found
}

// ─────────────────────────────────────────────────────────────────────────────
// SERVICE LIFECYCLE
// ─────────────────────────────────────────────────────────────────────────────

var serviceReady atomic.Int32

type ServiceStats struct {
	StartTime    time.Time
	RequestCount atomic.Int64
}

func (s *ServiceStats) uptime() time.Duration {
	return time.Since(s.StartTime).Round(time.Second)
}

func (s *ServiceStats) handleRequest(path string) {
	s.RequestCount.Add(1)
	fmt.Printf("  [req] %s (total: %d)\n", path, s.RequestCount.Load())
}

// ─────────────────────────────────────────────────────────────────────────────
// DOCKERFILE REFERENCE
// ─────────────────────────────────────────────────────────────────────────────

const productionDockerfile = `# Production Dockerfile for this service
# ─────────────────────────────────────────

# Stage 1: Build
FROM golang:1.24-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata

# Cache dependencies separately from source
COPY go.mod go.sum ./
RUN go mod download

# Build with metadata injection
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath \
    -ldflags="-s -w \
      -X main.Version=${VERSION} \
      -X main.Commit=${COMMIT} \
      -X main.BuildTime=${BUILD_TIME}" \
    -o /bin/app .

# Stage 2: Minimal runtime
FROM gcr.io/distroless/static-debian12
# Copy CA certs for HTTPS outbound calls
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
# Copy binary
COPY --from=builder /bin/app /app

# Run as non-root (UID 65534 = nobody in distroless)
USER nonroot:nonroot

EXPOSE 8080
ENTRYPOINT ["/app"]`

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 92 Exercise: Production Image Patterns ===")
	fmt.Println()

	// ── CONFIGURATION ─────────────────────────────────────────────────────────
	cfg := loadConfig()
	fmt.Println("--- Configuration ---")
	fmt.Printf("  %s\n", cfg)
	fmt.Printf("  Build: version=%s commit=%s built=%s\n", Version, Commit, BuildTime)
	fmt.Printf("  Runtime: go=%s os=%s arch=%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Println()

	// ── SECRETS DETECTION ─────────────────────────────────────────────────────
	fmt.Println("--- Secrets detection ---")
	secrets := detectSecretEnv()
	if len(secrets) == 0 {
		fmt.Println("  No secret-like environment variables detected.")
	} else {
		fmt.Printf("  WARNING: %d secret-like env vars found (never log values):\n", len(secrets))
		for _, s := range secrets {
			fmt.Printf("    %s\n", s)
		}
	}
	fmt.Println()

	// ── SERVICE LIFECYCLE ─────────────────────────────────────────────────────
	fmt.Println("--- Service lifecycle ---")
	stats := &ServiceStats{StartTime: time.Now()}

	// Simulate startup delay
	fmt.Println("  [startup] Connecting to database...")
	time.Sleep(50 * time.Millisecond)
	fmt.Println("  [startup] Running migrations...")
	time.Sleep(50 * time.Millisecond)
	serviceReady.Store(1)
	fmt.Printf("  [ready] Service ready after %v\n", stats.uptime())
	fmt.Println()

	// Simulate request handling
	fmt.Println("--- Handling requests ---")
	for _, path := range []string{"/api/orders", "/api/users/42", "/api/health"} {
		stats.handleRequest(path)
	}
	fmt.Println()

	// ── GRACEFUL SHUTDOWN ─────────────────────────────────────────────────────
	fmt.Println("--- Graceful shutdown simulation ---")
	fmt.Println("  [signal] SIGTERM received")
	serviceReady.Store(0)
	fmt.Println("  [shutdown] readyz → 503 (draining connections)")
	fmt.Printf("  [shutdown] Waiting up to %s seconds for in-flight requests...\n", cfg.DrainSecs)
	time.Sleep(50 * time.Millisecond) // simulated drain
	fmt.Printf("  [shutdown] Clean exit after %v uptime, %d requests served\n",
		stats.uptime(), stats.RequestCount.Load())
	fmt.Println()

	// ── DOCKERFILE ────────────────────────────────────────────────────────────
	fmt.Println("--- Production Dockerfile ---")
	fmt.Println(productionDockerfile)
}
