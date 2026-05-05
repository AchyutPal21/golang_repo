# Chapter 94 Exercises — CI/CD for Go Services

## Exercise 1 (provided): Pipeline Simulator

Location: `exercises/01_pipeline/main.go`

Simulates a complete CI/CD pipeline with:
- Sequential stages: lint → test → build → scan → push → deploy
- Pass/fail for each stage with simulated durations
- Fast-fail on first error
- Build matrix: linux/amd64 × linux/arm64
- Total pipeline time measurement

## Exercise 2 (self-directed): Coverage Gate

Build a coverage gate checker:
- Parse `go tool cover -func` output (provided as a string)
- Extract per-package and total coverage percentages
- Return an error if total is below a configurable threshold
- Print a sorted table: package, statements covered, percentage
- Highlight packages below threshold

## Exercise 3 (self-directed): Dependency Vulnerability Scanner

Simulate `govulncheck` output parsing:
- Define a `Vulnerability` struct: `{ID, Package, Severity, FixedIn string}`
- Parse a mock vulnerability report (JSON string)
- Exit with code 1 if any CRITICAL vulnerabilities are found
- Print a formatted report grouped by severity

## Stretch Goal: GitHub Actions YAML Generator

Write a Go program that generates a `.github/workflows/ci.yml` file for a given Go service configuration:
- Input: service name, Go version, coverage threshold, Kubernetes namespace
- Output: complete workflow YAML with lint, test, build, push, deploy stages
- Include matrix build for the specified GOOS/GOARCH targets
- Include proper caching with `hashFiles`
