// FILE: 07_packages_modules/05_build_tags.go
// TOPIC: Build Constraints — platform-specific code, custom tags
//
// Run: go run 07_packages_modules/05_build_tags.go
//
// ─────────────────────────────────────────────────────────────────────────────
// Build constraints let you include/exclude files from compilation based on:
//   - Operating system (linux, darwin, windows)
//   - CPU architecture (amd64, arm64, 386)
//   - Go version (go1.18, go1.21)
//   - Custom tags you define (debug, integration, cgo)
//
// This file itself has NO build constraint — it always compiles.
// Real constraint files would use: //go:build linux  (at very top of file)
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Build Constraints")
	fmt.Println("════════════════════════════════════════")

	// ── HOW BUILD CONSTRAINTS WORK ─────────────────────────────────────────
	fmt.Println(`
── Build constraint syntax (//go:build) ──

  Place at the TOP of the file, before package declaration.
  A blank line MUST separate it from the package line.

  // File compiled ONLY on Linux:
  //go:build linux
  package main

  // File compiled on Linux OR macOS:
  //go:build linux || darwin

  // File compiled on amd64 AND Linux:
  //go:build linux && amd64

  // File compiled when NOT on Windows:
  //go:build !windows

  // Custom tag (user-defined):
  //go:build integration
  // Run with: go test -tags integration ./...

── File naming conventions (implicit constraints) ──

  Go also infers constraints from file names:
    file_linux.go       → only on Linux
    file_windows.go     → only on Windows
    file_darwin.go      → only on macOS
    file_linux_amd64.go → Linux + amd64 only
    file_test.go        → only included in test builds

  These are IMPLICIT constraints — no //go:build needed.
  Explicit //go:build is more readable for complex cases.
`)

	// ── READING RUNTIME OS/ARCH ────────────────────────────────────────────
	fmt.Println("── Current build environment ──")
	fmt.Printf("  GOOS:   %s\n", runtime.GOOS)
	fmt.Printf("  GOARCH: %s\n", runtime.GOARCH)
	fmt.Printf("  NumCPU: %d\n", runtime.NumCPU())

	// ── COMMON USE CASES ───────────────────────────────────────────────────
	fmt.Println(`
── Common use cases ──

  1. Platform-specific syscalls:
     net_linux.go    → uses epoll
     net_darwin.go   → uses kqueue
     net_windows.go  → uses IOCP

  2. Integration tests (don't run in normal CI):
     //go:build integration
     go test -tags integration -run TestRealDatabase ./...

  3. Debug builds with extra logging:
     //go:build debug
     go run -tags debug .

  4. Excluding cgo on platforms that don't have it:
     //go:build cgo

  5. Go version gating (use newer APIs safely):
     //go:build go1.21

── Running with tags ──

  go build -tags integration .
  go test -tags "integration debug" ./...
  go run -tags linux .

── Checking which files will be compiled ──

  go list -f "{{.GoFiles}}" .
  go list -tags integration -f "{{.GoFiles}}" .
`)
}
