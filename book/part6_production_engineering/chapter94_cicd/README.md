# Chapter 94 — CI/CD for Go Services

A CI/CD pipeline for Go should: lint, test, build, scan for vulnerabilities, build a Docker image, and deploy — all in a reproducible, fast pipeline.

## Topics

| # | Topic | Key ideas |
|---|-------|-----------|
| 1 | GitHub Actions | Workflow YAML, matrix builds, caching, secrets |
| 2 | goreleaser | Cross-platform binaries, changelogs, container publishing |
| E | Pipeline simulator | End-to-end pipeline stages with timing and failure handling |

## Examples

### `examples/01_github_actions`

GitHub Actions workflow patterns for Go:
- Lint with `golangci-lint`
- Test with race detector and coverage
- Build multi-arch Docker image
- Deploy to Kubernetes on tag push
- Workflow matrix: Go 1.23, 1.24 × linux/amd64, linux/arm64

### `examples/02_goreleaser`

GoReleaser configuration and cross-compilation:
- `.goreleaser.yaml` structure
- Cross-compilation targets
- SBOM generation
- Docker image publishing
- Changelog from git commits

### `exercises/01_pipeline`

Simulated CI/CD pipeline:
- Sequential stages with pass/fail
- Build matrix (OS × arch)
- Timing measurements per stage
- Failure handling and fast-fail logic

## Key Concepts

**Pipeline stages (fast → slow)**
1. `go vet` + `golangci-lint` (< 60s)
2. `go test -race ./...` with coverage gate
3. `go build` (multi-arch)
4. `docker build` + image scan
5. Push to registry
6. Deploy to staging
7. Integration tests
8. Deploy to production

**Go-specific CI tricks**
```yaml
# Cache Go module download cache
- uses: actions/cache@v4
  with:
    path: ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}

# Race detector (use in CI, not prod builds)
go test -race -count=1 -timeout=5m ./...

# Coverage gate
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total | awk '{print $3}'
```

**goreleaser minimal config**
```yaml
builds:
  - env: [CGO_ENABLED=0]
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags: ["-s -w -X main.Version={{.Version}}"]
dockers:
  - image_templates: ["ghcr.io/org/app:{{.Tag}}"]
    use: buildx
    build_flag_templates: ["--platform=linux/amd64,linux/arm64"]
```

## Running

```bash
go run ./book/part6_production_engineering/chapter94_cicd/examples/01_github_actions
go run ./book/part6_production_engineering/chapter94_cicd/examples/02_goreleaser
go run ./book/part6_production_engineering/chapter94_cicd/exercises/01_pipeline
```
