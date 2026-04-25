// FILE: book/part1_foundations/chapter05_go_run_build_install/examples/03_ldflags_version/main.go
// CHAPTER: 05 — How `go run`, `go build`, `go install` Work
// TOPIC: Stamping a version string at link time with -ldflags -X.
//
// Run as-is (default values):
//   go run ./examples/03_ldflags_version
//
// Run with custom values:
//   go build -ldflags="-X main.version=1.2.3 -X main.commit=abc1234" \
//       -o /tmp/vapp ./examples/03_ldflags_version
//   /tmp/vapp
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   The minimal demo of -ldflags -X. Three things to notice:
//     1. version, commit, date are package-level VARIABLES (not const).
//        -X cannot set consts.
//     2. They have default values, so `go run` without flags still works.
//     3. The full path of the variable is "main.version" — package name
//        plus identifier, separated by a dot.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"runtime/debug"
)

// version, commit, date are stamped at link time via:
//
//   -ldflags "-X main.version=1.2.3 -X main.commit=abc1234 -X main.date=2026-04-25"
//
// They MUST be `var`, not `const`, and must be of type string. The Go
// linker does a textual substitution; it cannot evaluate expressions or
// touch typed constants.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	fmt.Println("=== version info ===")
	fmt.Printf("version : %s\n", version)
	fmt.Printf("commit  : %s\n", commit)
	fmt.Printf("date    : %s\n", date)

	// Since Go 1.18, the toolchain auto-embeds VCS info if -buildvcs=true
	// (the default). We can read it without -ldflags. This is the
	// preferred source for commit hash.
	if info, ok := debug.ReadBuildInfo(); ok {
		fmt.Println("\n=== auto-embedded VCS info ===")
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs", "vcs.revision", "vcs.time", "vcs.modified",
				"GOOS", "GOARCH", "CGO_ENABLED", "-buildmode", "-trimpath":
				fmt.Printf("  %-15s %s\n", s.Key, s.Value)
			}
		}
		fmt.Printf("\nGo version: %s\n", info.GoVersion)
	}
}
