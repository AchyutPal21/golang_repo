// FILE: book/part6_production_engineering/chapter92_dockerizing/examples/01_multistage_build/main.go
// CHAPTER: 92 — Dockerizing Go Services
// TOPIC: Multi-stage Dockerfile patterns, build metadata injection,
//        image size analysis, layer cache strategy.
//
// Run:
//   go run ./book/part6_production_engineering/chapter92_dockerizing/examples/01_multistage_build

package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// BUILD METADATA — injected at build time via -ldflags
// ─────────────────────────────────────────────────────────────────────────────

// These are overridden by the linker:
//   go build -ldflags="-X main.Version=1.2.3 -X main.Commit=$(git rev-parse --short HEAD) -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func printBuildInfo() {
	fmt.Println("--- Build metadata ---")
	fmt.Printf("  Version:   %s\n", Version)
	fmt.Printf("  Commit:    %s\n", Commit)
	fmt.Printf("  BuildTime: %s\n", BuildTime)
	fmt.Printf("  Go:        %s\n", runtime.Version())
	fmt.Printf("  OS/Arch:   %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

// ─────────────────────────────────────────────────────────────────────────────
// DOCKERFILE PATTERNS
// ─────────────────────────────────────────────────────────────────────────────

const dockerfileOptimal = `# OPTIMAL: copy go.mod/go.sum first to cache dependency layer
FROM golang:1.24-alpine AS builder
WORKDIR /app

# Layer 1: dependency cache (invalidated only when go.mod/go.sum change)
COPY go.mod go.sum ./
RUN go mod download

# Layer 2: source code (invalidated on every code change)
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT}" \
    -o /bin/app ./cmd/app

# Final stage: minimal runtime image
FROM gcr.io/distroless/static-debian12
COPY --from=builder /bin/app /app
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app"]`

const dockerfileNaive = `# NAIVE: copies everything before downloading — cache busted on every change
FROM golang:1.24-alpine
WORKDIR /app
COPY . .                          # ← every code change invalidates all layers below
RUN go mod download               # ← re-downloads all deps on every change
RUN CGO_ENABLED=0 go build -o app .`

// ─────────────────────────────────────────────────────────────────────────────
// IMAGE LAYER SIMULATION
// ─────────────────────────────────────────────────────────────────────────────

type Layer struct {
	Label string
	SizeB int64
}

func (l Layer) humanSize() string {
	switch {
	case l.SizeB >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(l.SizeB)/(1024*1024))
	case l.SizeB >= 1024:
		return fmt.Sprintf("%.1f KB", float64(l.SizeB)/1024)
	default:
		return fmt.Sprintf("%d B", l.SizeB)
	}
}

type ImageSpec struct {
	Name   string
	Layers []Layer
}

func (img ImageSpec) totalSize() int64 {
	var total int64
	for _, l := range img.Layers {
		total += l.SizeB
	}
	return total
}

func (img ImageSpec) largestLayer() Layer {
	var largest Layer
	for _, l := range img.Layers {
		if l.SizeB > largest.SizeB {
			largest = l
		}
	}
	return largest
}

func printImageAnalysis(img ImageSpec) {
	fmt.Printf("  Image: %s\n", img.Name)
	fmt.Printf("  %-30s  %8s  %10s\n", "Layer", "Size", "Cumulative")
	fmt.Printf("  %s\n", strings.Repeat("-", 55))
	var cumulative int64
	for _, l := range img.Layers {
		cumulative += l.SizeB
		cumL := Layer{SizeB: cumulative}
		fmt.Printf("  %-30s  %8s  %10s\n", l.Label, l.humanSize(), cumL.humanSize())
	}
	total := Layer{SizeB: img.totalSize()}
	largest := img.largestLayer()
	fmt.Printf("  Total: %s\n", total.humanSize())
	fmt.Printf("  Largest layer: %q (%s)\n", largest.Label, largest.humanSize())
}

// ─────────────────────────────────────────────────────────────────────────────
// IMAGE COMPARISON
// ─────────────────────────────────────────────────────────────────────────────

var imageComparisons = []struct {
	base        string
	baseSize    string
	totalTyp    string
	tradeoff    string
}{
	{"ubuntu:22.04", "77 MB", "~85 MB", "Full OS; largest attack surface"},
	{"alpine:3.19", "7 MB", "~15 MB", "Shell + package manager; good for debugging"},
	{"distroless/static", "2 MB", "~9 MB", "CA certs + tzdata; no shell (recommended)"},
	{"scratch", "0 B", "~7 MB", "Binary only; no CA certs unless copied in"},
}

func printComparison() {
	fmt.Printf("  %-25s  %10s  %12s  %s\n", "Base image", "Base size", "With Go app", "Trade-off")
	fmt.Printf("  %s\n", strings.Repeat("-", 85))
	for _, c := range imageComparisons {
		fmt.Printf("  %-25s  %10s  %12s  %s\n", c.base, c.baseSize, c.totalTyp, c.tradeoff)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// BUILD FLAGS REFERENCE
// ─────────────────────────────────────────────────────────────────────────────

const buildFlagsRef = `  Build flag reference:
    CGO_ENABLED=0          → static binary; no libc dependency
    GOOS=linux             → cross-compile target OS
    GOARCH=amd64           → cross-compile target arch
    -ldflags="-s -w"       → strip symbol table (-s) and DWARF (-w)
    -ldflags="-X pkg.Var=v"→ inject variable value at link time
    -trimpath              → remove local paths from binary (reproducible builds)

  Full production command:
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
      go build -trimpath \
      -ldflags="-s -w -X main.Version=1.2.3 -X main.Commit=$(git rev-parse --short HEAD)" \
      -o /bin/app ./cmd/app`

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 92: Multi-Stage Docker Builds ===")
	fmt.Println()

	printBuildInfo()
	fmt.Println()

	// ── DOCKERFILE PATTERNS ───────────────────────────────────────────────────
	fmt.Println("--- Optimal Dockerfile (dependency layer cached) ---")
	fmt.Println(dockerfileOptimal)
	fmt.Println()

	fmt.Println("--- Naive Dockerfile (cache busted on every change) ---")
	fmt.Println(dockerfileNaive)
	fmt.Println()

	// ── IMAGE LAYER ANALYSIS ──────────────────────────────────────────────────
	fmt.Println("--- Image layer analysis (simulated) ---")
	distrolessImage := ImageSpec{
		Name: "myapp:latest (distroless)",
		Layers: []Layer{
			{"distroless/static base", 2 * 1024 * 1024},
			{"CA certificates", 300 * 1024},
			{"Go binary (stripped)", 6 * 1024 * 1024},
			{"Config files", 4 * 1024},
		},
	}
	printImageAnalysis(distrolessImage)
	fmt.Println()

	// ── BASE IMAGE COMPARISON ─────────────────────────────────────────────────
	fmt.Println("--- Base image comparison ---")
	printComparison()
	fmt.Println()

	// ── BUILD FLAGS REFERENCE ─────────────────────────────────────────────────
	fmt.Println("--- Build flags ---")
	fmt.Println(buildFlagsRef)
	fmt.Println()

	// ── ENVIRONMENT INFO ──────────────────────────────────────────────────────
	fmt.Println("--- Runtime environment ---")
	if v := os.Getenv("VERSION"); v != "" {
		fmt.Printf("  VERSION env: %s\n", v)
	} else {
		fmt.Println("  VERSION env: (not set — use -ldflags to inject)")
	}
}
