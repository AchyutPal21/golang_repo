// FILE: book/part1_foundations/chapter05_go_run_build_install/examples/01_build_inspector/main.go
// CHAPTER: 05 — How `go run`, `go build`, `go install` Work
// TOPIC: Inspect any Go binary's build info, including this one.
//
// Run (from the chapter folder):
//   go run ./examples/01_build_inspector
//   go run ./examples/01_build_inspector $(which gopls)
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   Production binaries should be self-describing. Since Go 1.18, every
//   binary embeds module + VCS info that you can read at runtime via
//   debug.ReadBuildInfo. This program shows two surfaces of the same data:
//   the running binary (no args) or any other Go binary on disk (with arg).
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"debug/buildinfo"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
)

func main() {
	if len(os.Args) >= 2 {
		// Inspect a binary from disk. debug/buildinfo handles ELF, Mach-O,
		// and PE; it returns the same BuildInfo type as the runtime
		// surface, so the rest of the code is shared.
		path := os.Args[1]
		info, err := buildinfo.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not read build info from %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("Inspecting: %s\n", path)
		render(info)
		return
	}

	// No args — inspect ourselves. debug.ReadBuildInfo() returns nil if the
	// build info is not available (e.g. on very old toolchains or with
	// -buildvcs=false).
	info, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("build info not available for this binary")
		os.Exit(1)
	}
	fmt.Println("Inspecting: (this binary)")
	render(info)
}

// render prints a human-readable view of a BuildInfo value. The shape of
// BuildInfo: GoVersion, Path (main module path), Main (Module), Deps
// (transitive deps), Settings (key/value pairs from the build).
func render(info *debug.BuildInfo) {
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Go version : %s\n", info.GoVersion)
	fmt.Printf("Module path: %s\n", info.Path)
	if info.Main.Path != "" {
		fmt.Printf("Main module: %s @ %s\n", info.Main.Path, info.Main.Version)
	}

	// Build settings: build flags, VCS info, GOOS/GOARCH at build time.
	// These are what you'd otherwise have to capture via -ldflags.
	if len(info.Settings) > 0 {
		fmt.Println("\nBuild settings:")
		for _, s := range info.Settings {
			fmt.Printf("  %-22s %s\n", s.Key, s.Value)
		}
	}

	// Dependencies. Truncated to the first 25 to keep output readable; bump
	// the cap if you're investigating a bloated binary.
	if len(info.Deps) > 0 {
		fmt.Printf("\nDirect + transitive deps (%d total):\n", len(info.Deps))
		shown := info.Deps
		if len(shown) > 25 {
			shown = shown[:25]
		}
		for _, d := range shown {
			line := fmt.Sprintf("  %s @ %s", d.Path, d.Version)
			if d.Replace != nil {
				line += fmt.Sprintf("  (replaced by %s @ %s)", d.Replace.Path, d.Replace.Version)
			}
			fmt.Println(line)
		}
		if len(info.Deps) > 25 {
			fmt.Printf("  ... and %d more\n", len(info.Deps)-25)
		}
	}
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println("Tip: try `go version -m <binary>` for the same data from the CLI.")
}
