// FILE: book/part6_production_engineering/chapter94_cicd/examples/01_github_actions/main.go
// CHAPTER: 94 — CI/CD for Go Services
// TOPIC: GitHub Actions workflow patterns for Go — lint, test, build,
//        Docker image, matrix builds, caching, and deployment.
//
// Run:
//   go run ./book/part6_production_engineering/chapter94_cicd/examples/01_github_actions

package main

import (
	"fmt"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// WORKFLOW TEMPLATES
// ─────────────────────────────────────────────────────────────────────────────

const ciWorkflow = `# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main, 'release/**']
  pull_request:
    branches: [main]

jobs:
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.24'
        cache: true
    - name: golangci-lint
      uses: golangci/golangci-lint-action@v6
      with:
        version: latest
        args: --timeout=5m

  test:
    name: Test
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go: ['1.23', '1.24']
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: ${{ matrix.go }}
    - name: Cache modules
      uses: actions/cache@v4
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ matrix.go }}-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-${{ matrix.go }}-
    - name: Download deps
      run: go mod download
    - name: Vet
      run: go vet ./...
    - name: Test with race detector
      run: go test -race -count=1 -timeout=5m -coverprofile=coverage.out ./...
    - name: Coverage gate (80%)
      run: |
        COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
        echo "Coverage: ${COVERAGE}%"
        awk "BEGIN {exit ($COVERAGE < 80)}" || (echo "FAIL: coverage below 80%"; exit 1)
    - name: govulncheck
      run: |
        go install golang.org/x/vuln/cmd/govulncheck@latest
        govulncheck ./...

  build:
    name: Build
    needs: [lint, test]
    runs-on: ubuntu-latest
    if: github.event_name == 'push'
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.24'
        cache: true
    - name: Build (linux/amd64)
      run: CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/app-amd64 .
    - name: Build (linux/arm64)
      run: CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o bin/app-arm64 .

  docker:
    name: Docker
    needs: build
    runs-on: ubuntu-latest
    if: startsWith(github.ref, 'refs/tags/v')
    steps:
    - uses: actions/checkout@v4
    - uses: docker/setup-buildx-action@v3
    - uses: docker/login-action@v3
      with:
        registry: ghcr.io
        username: ${{ github.actor }}
        password: ${{ secrets.GITHUB_TOKEN }}
    - name: Build and push multi-arch image
      uses: docker/build-push-action@v5
      with:
        context: .
        platforms: linux/amd64,linux/arm64
        push: true
        tags: ghcr.io/${{ github.repository }}:${{ github.ref_name }}`

const deployWorkflow = `# .github/workflows/deploy.yml
name: Deploy

on:
  push:
    tags: ['v*']

jobs:
  deploy-staging:
    runs-on: ubuntu-latest
    environment: staging
    steps:
    - uses: actions/checkout@v4
    - name: Deploy to staging
      run: |
        kubectl set image deployment/my-app \
          app=ghcr.io/${{ github.repository }}:${{ github.ref_name }} \
          -n staging
        kubectl rollout status deployment/my-app -n staging

  deploy-production:
    runs-on: ubuntu-latest
    environment: production
    needs: deploy-staging
    steps:
    - name: Deploy to production
      run: |
        kubectl set image deployment/my-app \
          app=ghcr.io/${{ github.repository }}:${{ github.ref_name }} \
          -n production
        kubectl rollout status deployment/my-app -n production`

// ─────────────────────────────────────────────────────────────────────────────
// GOLANGCI-LINT CONFIG
// ─────────────────────────────────────────────────────────────────────────────

const golangciConfig = `# .golangci.yml
linters:
  enable:
    - staticcheck    # comprehensive static analysis
    - errcheck       # ensure all errors are handled
    - gosec          # security issues
    - exhaustive     # exhaustive enum switches
    - noctx          # HTTP requests must use context
    - bodyclose      # HTTP response bodies must be closed
    - gocritic       # opinionated style improvements
    - misspell       # common spelling mistakes
    - unconvert      # unnecessary type conversions
    - unparam        # unused function parameters

linters-settings:
  errcheck:
    check-type-assertions: true
  gosec:
    excludes: [G401, G501]   # exclude weak crypto in tests only

issues:
  exclude-rules:
    - path: _test\.go
      linters: [gosec, errcheck]`

// ─────────────────────────────────────────────────────────────────────────────
// BUILD MATRIX ANALYSIS
// ─────────────────────────────────────────────────────────────────────────────

type BuildTarget struct {
	GOOS   string
	GOARCH string
}

var supportedTargets = []BuildTarget{
	{"linux", "amd64"},
	{"linux", "arm64"},
	{"darwin", "amd64"},
	{"darwin", "arm64"},
	{"windows", "amd64"},
}

func printBuildMatrix() {
	fmt.Printf("  %-10s  %-8s  %s\n", "GOOS", "GOARCH", "Binary name")
	fmt.Printf("  %s\n", strings.Repeat("-", 40))
	for _, t := range supportedTargets {
		ext := ""
		if t.GOOS == "windows" {
			ext = ".exe"
		}
		name := fmt.Sprintf("app-%s-%s%s", t.GOOS, t.GOARCH, ext)
		fmt.Printf("  %-10s  %-8s  %s\n", t.GOOS, t.GOARCH, name)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 94: GitHub Actions for Go ===")
	fmt.Println()

	fmt.Println("--- CI workflow (.github/workflows/ci.yml) ---")
	fmt.Println(ciWorkflow)
	fmt.Println()

	fmt.Println("--- Deploy workflow (.github/workflows/deploy.yml) ---")
	fmt.Println(deployWorkflow)
	fmt.Println()

	fmt.Println("--- golangci-lint config (.golangci.yml) ---")
	fmt.Println(golangciConfig)
	fmt.Println()

	fmt.Println("--- Build matrix targets ---")
	printBuildMatrix()
	fmt.Println()

	fmt.Println("--- Pipeline summary ---")
	fmt.Println(`  PR workflow:   lint → test (matrix) → [done]
  Tag workflow:  lint → test → build → docker → deploy-staging → deploy-prod

  Cache strategy:
    go.sum hash    → module download cache (hits on every run with same deps)
    go-build cache → build cache (hits on every run with same source)
    golangci-lint  → lint result cache (hits when files unchanged)

  Secrets best practices:
    - GITHUB_TOKEN: auto-injected for pushing to ghcr.io
    - KUBECONFIG: store in GitHub environment secrets (not repo secrets)
    - Never print secrets: use ::add-mask:: or GitHub's automatic masking`)
}
