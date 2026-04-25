// FILE: book/part1_foundations/chapter05_go_run_build_install/examples/02_cross_compile_demo/main.go
// CHAPTER: 05 — How `go run`, `go build`, `go install` Work
// TOPIC: Cross-compilation as data — print the commands you'd actually run.
//
// Run (from the chapter folder):
//   go run ./examples/02_cross_compile_demo
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   Cross-compilation looks intimidating but is actually two env vars. This
//   program prints the running platform, then prints the GOOS/GOARCH command
//   for every common target — copy/paste-ready. Run `go tool dist list` for
//   the exhaustive list (~50 combos).
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"runtime"
	"strings"
)

// target is one cross-compilation target: a GOOS, a GOARCH, and a short
// human-readable name for the use case. The list is curated; it is *not*
// the full output of `go tool dist list`.
type target struct {
	goos, goarch, label string
}

var common = []target{
	{"linux", "amd64", "most servers"},
	{"linux", "arm64", "AWS Graviton, RPi 4+"},
	{"darwin", "arm64", "Apple Silicon Mac"},
	{"darwin", "amd64", "Intel Mac"},
	{"windows", "amd64", "most Windows"},
	{"windows", "arm64", "Windows on ARM"},
	{"freebsd", "amd64", "FreeBSD server"},
	{"js", "wasm", "WebAssembly for browsers"},
}

func main() {
	fmt.Printf("You are currently building on: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println("To build for another target, set GOOS and GOARCH:")
	fmt.Println()

	for _, t := range common {
		current := t.goos == runtime.GOOS && t.goarch == runtime.GOARCH
		marker := "   "
		if current {
			marker = " * "
		}
		out := outputName("app", t)
		fmt.Printf("%s%-12s GOOS=%s GOARCH=%s go build -o %s ./cmd/app\n",
			marker, t.label+":", t.goos, t.goarch, out)
	}

	fmt.Println()
	fmt.Println("(*) is your current platform. Pure-Go binaries cross-compile")
	fmt.Println("    with no extra toolchain. Add CGO_ENABLED=0 if your imports")
	fmt.Println("    bring in cgo dependencies.")
	fmt.Println()
	fmt.Println("For the exhaustive list of supported combinations:")
	fmt.Println("    go tool dist list")
}

// outputName builds a sensible -o filename for a given target. Convention:
// app-<os>-<arch>, with .exe suffix on Windows.
func outputName(base string, t target) string {
	name := fmt.Sprintf("%s-%s-%s", base, t.goos, t.goarch)
	if t.goos == "windows" {
		name += ".exe"
	}
	return name
}
