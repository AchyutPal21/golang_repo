// FILE: book/part1_foundations/chapter01_why_go_exists/exercises/01_verify/main.go
// EXERCISE 1.1 — Verify your install.
//
// Run (from the chapter folder):
//   go run ./exercises/01_verify
//
// Expected output (your numbers will differ):
//
//   Go version : go1.22.5
//   GOOS/GOARCH: linux / amd64
//   NumCPU     : 16
//   Startup    : 73.421µs
//
// If any of these look wrong, run `go env` and check your install.

package main

import (
	"fmt"
	"runtime"
	"time"
)

func main() {
	t := time.Now()
	fmt.Printf("Go version : %s\n", runtime.Version())
	fmt.Printf("GOOS/GOARCH: %s / %s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("NumCPU     : %d\n", runtime.NumCPU())
	fmt.Printf("Startup    : %s\n", time.Since(t))
}
