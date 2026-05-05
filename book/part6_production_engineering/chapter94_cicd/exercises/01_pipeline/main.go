// FILE: book/part6_production_engineering/chapter94_cicd/exercises/01_pipeline/main.go
// CHAPTER: 94 — CI/CD for Go Services
// EXERCISE: Simulate a complete CI/CD pipeline with sequential stages,
//           build matrix, timing, and failure handling.
//
// Run:
//   go run ./book/part6_production_engineering/chapter94_cicd/exercises/01_pipeline

package main

import (
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// PIPELINE STAGE
// ─────────────────────────────────────────────────────────────────────────────

type StageResult struct {
	Name     string
	Passed   bool
	Duration time.Duration
	Output   string
}

type Stage struct {
	Name string
	Run  func() (string, error)
}

func runStage(s Stage) StageResult {
	start := time.Now()
	output, err := s.Run()
	dur := time.Since(start)
	passed := err == nil
	msg := output
	if err != nil {
		msg = err.Error()
	}
	return StageResult{s.Name, passed, dur, msg}
}

// ─────────────────────────────────────────────────────────────────────────────
// PIPELINE RUNNER
// ─────────────────────────────────────────────────────────────────────────────

type Pipeline struct {
	Name   string
	Stages []Stage
}

func (p Pipeline) Run() []StageResult {
	fmt.Printf("Pipeline: %s\n", p.Name)
	fmt.Println(strings.Repeat("=", 60))
	var results []StageResult
	for _, s := range p.Stages {
		fmt.Printf("  ▶ %-25s ... ", s.Name)
		r := runStage(s)
		results = append(results, r)
		if r.Passed {
			fmt.Printf("PASS (%v)\n", r.Duration.Round(time.Millisecond))
		} else {
			fmt.Printf("FAIL (%v)\n    Error: %s\n", r.Duration.Round(time.Millisecond), r.Output)
			fmt.Println("\n  Pipeline stopped (fast-fail).")
			return results
		}
	}
	return results
}

func printSummary(results []StageResult) {
	fmt.Println()
	fmt.Println("Summary:")
	fmt.Println(strings.Repeat("-", 50))
	passed, failed := 0, 0
	var total time.Duration
	for _, r := range results {
		status := "PASS"
		if !r.Passed {
			status = "FAIL"
			failed++
		} else {
			passed++
		}
		total += r.Duration
		fmt.Printf("  %-25s  %4s  %v\n", r.Name, status, r.Duration.Round(time.Millisecond))
	}
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("  Passed: %d  Failed: %d  Total: %v\n", passed, failed, total.Round(time.Millisecond))
}

// ─────────────────────────────────────────────────────────────────────────────
// SIMULATED STAGE IMPLEMENTATIONS
// ─────────────────────────────────────────────────────────────────────────────

func stageVet() (string, error) {
	time.Sleep(200 * time.Millisecond)
	return "go vet: no issues found", nil
}

func stageLint() (string, error) {
	time.Sleep(400 * time.Millisecond)
	return "golangci-lint: no issues found", nil
}

func stageTest() (string, error) {
	time.Sleep(800 * time.Millisecond)
	return "PASS: 142 tests in 0.8s, coverage: 84.2%", nil
}

func stageVulnCheck() (string, error) {
	time.Sleep(300 * time.Millisecond)
	return "govulncheck: No vulnerabilities found.", nil
}

func stageBuild() (string, error) {
	time.Sleep(600 * time.Millisecond)
	return "Built app-linux-amd64 (7.2MB), app-linux-arm64 (6.8MB)", nil
}

func stageDockerBuild() (string, error) {
	time.Sleep(1200 * time.Millisecond)
	return "Built ghcr.io/myorg/app:2.4.1 (linux/amd64, linux/arm64)", nil
}

func stageDockerPush() (string, error) {
	time.Sleep(800 * time.Millisecond)
	return "Pushed ghcr.io/myorg/app:2.4.1", nil
}

func stageDeployStaging() (string, error) {
	time.Sleep(500 * time.Millisecond)
	return "Deployed to staging, rollout complete (3/3 pods ready)", nil
}

func stageIntegrationTests() (string, error) {
	time.Sleep(600 * time.Millisecond)
	return "Integration tests: 18/18 passed", nil
}

func stageDeployProduction() (string, error) {
	time.Sleep(700 * time.Millisecond)
	return "Deployed to production, rollout complete (5/5 pods ready)", nil
}

// ─────────────────────────────────────────────────────────────────────────────
// BUILD MATRIX
// ─────────────────────────────────────────────────────────────────────────────

type MatrixTarget struct {
	GOOS   string
	GOARCH string
}

func runBuildMatrix(targets []MatrixTarget) {
	fmt.Println()
	fmt.Println("Build matrix:")
	fmt.Println(strings.Repeat("-", 50))
	allPassed := true
	for _, t := range targets {
		time.Sleep(50 * time.Millisecond) // simulated build time
		// darwin/arm64 always passes; everything else passes in this simulation
		passed := true
		status := "PASS"
		if !passed {
			status = "FAIL"
			allPassed = false
		}
		fmt.Printf("  GOOS=%-10s GOARCH=%-8s  %s\n", t.GOOS, t.GOARCH, status)
	}
	if allPassed {
		fmt.Println("  All targets built successfully.")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// COVERAGE GATE
// ─────────────────────────────────────────────────────────────────────────────

type PackageCoverage struct {
	Package    string
	Statements int
	Percent    float64
}

func checkCoverage(packages []PackageCoverage, threshold float64) error {
	fmt.Printf("\nCoverage gate (threshold: %.0f%%):\n", threshold)
	fmt.Printf("  %-45s  %10s  %8s\n", "Package", "Statements", "Coverage")
	fmt.Printf("  %s\n", strings.Repeat("-", 70))
	var totalStmts, coveredStmts int
	for _, p := range packages {
		mark := ""
		if p.Percent < threshold {
			mark = " ⚠"
		}
		covered := int(float64(p.Statements) * p.Percent / 100)
		totalStmts += p.Statements
		coveredStmts += covered
		fmt.Printf("  %-45s  %10d  %7.1f%%%s\n", p.Package, p.Statements, p.Percent, mark)
	}
	totalPct := 100 * float64(coveredStmts) / float64(totalStmts)
	fmt.Printf("  %-45s  %10d  %7.1f%%\n", "TOTAL", totalStmts, totalPct)
	if totalPct < threshold {
		return fmt.Errorf("coverage %.1f%% is below threshold %.0f%%", totalPct, threshold)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 94 Exercise: CI/CD Pipeline Simulator ===")
	fmt.Println()

	// ── FULL PIPELINE ─────────────────────────────────────────────────────────
	pipeline := Pipeline{
		Name: "order-service v2.4.1 (tag push)",
		Stages: []Stage{
			{"go vet", stageVet},
			{"golangci-lint", stageLint},
			{"go test -race", stageTest},
			{"govulncheck", stageVulnCheck},
			{"go build", stageBuild},
			{"docker build", stageDockerBuild},
			{"docker push", stageDockerPush},
			{"deploy → staging", stageDeployStaging},
			{"integration tests", stageIntegrationTests},
			{"deploy → production", stageDeployProduction},
		},
	}

	results := pipeline.Run()
	printSummary(results)

	// ── BUILD MATRIX ──────────────────────────────────────────────────────────
	targets := []MatrixTarget{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"windows", "amd64"},
	}
	runBuildMatrix(targets)

	// ── COVERAGE GATE ─────────────────────────────────────────────────────────
	packages := []PackageCoverage{
		{"github.com/myorg/app/internal/order", 142, 91.5},
		{"github.com/myorg/app/internal/payment", 89, 88.7},
		{"github.com/myorg/app/internal/cache", 54, 75.9},
		{"github.com/myorg/app/pkg/config", 38, 97.3},
		{"github.com/myorg/app/pkg/middleware", 67, 82.1},
	}
	if err := checkCoverage(packages, 80.0); err != nil {
		fmt.Printf("  GATE FAILED: %v\n", err)
	} else {
		fmt.Println("  Coverage gate: PASSED")
	}
}
