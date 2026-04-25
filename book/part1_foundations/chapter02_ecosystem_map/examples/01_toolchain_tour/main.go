// FILE: book/part1_foundations/chapter02_ecosystem_map/examples/01_toolchain_tour/main.go
// CHAPTER: 02 — A Map of the Go Ecosystem
// TOPIC: The toolchain, surfaced as data: env vars, caches, paths.
//
// Run (from the chapter folder):
//   go run ./examples/01_toolchain_tour
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   The shape of the Go ecosystem is mostly visible through environment
//   variables and well-known directories. This program prints them in a
//   readable form, so you can see where modules live, where the build cache
//   lives, and where installed tools land. Use it as a one-shot health
//   check on a new machine.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// envVar is a single entry in the toolchain's environment surface.
type envVar struct {
	name string
	desc string
}

// vars is the curated list of GOxxx variables we care about. The list is
// intentionally short — the full surface is `go env`, but most of it is
// rarely interesting. We pick the ones a senior Go engineer reads first.
var vars = []envVar{
	{"GOROOT", "where the Go toolchain itself is installed"},
	{"GOPATH", "where modules and tool binaries live by default"},
	{"GOBIN", "where `go install` writes binaries (defaults to $GOPATH/bin)"},
	{"GOMODCACHE", "where downloaded module versions are cached"},
	{"GOOS", "target operating system for builds"},
	{"GOARCH", "target CPU architecture for builds"},
	{"CGO_ENABLED", "whether C interop is permitted (1 enables, 0 disables)"},
	{"GOPROXY", "module proxy URL list, comma-separated"},
	{"GOSUMDB", "checksum database for module integrity"},
	{"GOPRIVATE", "patterns that bypass the public proxy and sumdb"},
	{"GOFLAGS", "default flags applied to every `go` command"},
	{"GOTOOLCHAIN", "toolchain selection policy (auto, local, or pinned)"},
}

func main() {
	fmt.Println("Go toolchain tour")
	fmt.Println(strings.Repeat("─", 60))

	// runtime.Version reports the Go release we were *compiled* with;
	// `go env` reports what the *currently-installed* toolchain is. They
	// usually match; when they don't, it's because GOTOOLCHAIN is doing
	// version switching.
	fmt.Printf("\nRuntime build:    %s on %s/%s\n",
		runtime.Version(), runtime.GOOS, runtime.GOARCH)

	// ─── Print the env surface ─────────────────────────────────────────
	fmt.Println("\nEnvironment:")
	for _, v := range vars {
		val, err := goEnv(v.name)
		switch {
		case err != nil:
			fmt.Printf("  %-12s %-50s [unavailable]\n", v.name, v.desc)
		case val == "":
			fmt.Printf("  %-12s %-50s (unset)\n", v.name, v.desc)
		default:
			fmt.Printf("  %-12s %-50s %s\n", v.name, v.desc, val)
		}
	}

	// ─── Cache sizes ───────────────────────────────────────────────────
	fmt.Println("\nCaches:")
	for _, key := range []string{"GOMODCACHE", "GOCACHE"} {
		path, err := goEnv(key)
		if err != nil || path == "" {
			fmt.Printf("  %-12s [unknown]\n", key)
			continue
		}
		size, files, ok := dirSize(path)
		if !ok {
			fmt.Printf("  %-12s %s (not yet present)\n", key, path)
			continue
		}
		fmt.Printf("  %-12s %s — %s, %d files\n",
			key, path, humanBytes(size), files)
	}

	// ─── Tools installed in $GOBIN ─────────────────────────────────────
	fmt.Println("\nInstalled tools (binaries in $GOBIN):")
	gobin := firstNonEmpty(must(goEnv("GOBIN")), filepath.Join(must(goEnv("GOPATH")), "bin"))
	if entries, err := os.ReadDir(gobin); err == nil {
		if len(entries) == 0 {
			fmt.Printf("  %s — empty\n", gobin)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			fmt.Printf("  %s\n", filepath.Join(gobin, e.Name()))
		}
	} else {
		fmt.Printf("  %s — not present (this is fine if you haven't `go install`-ed any tool yet)\n", gobin)
	}

	fmt.Println("\nTip: run `go env` for the full surface.")
}

// goEnv shells out to `go env <NAME>` and returns the trimmed value.
// We use the actual `go` binary rather than os.Getenv because some of
// these values (GOPROXY, GOMODCACHE) are computed by the toolchain and
// don't always live in the shell environment.
func goEnv(name string) (string, error) {
	out, err := exec.Command("go", "env", name).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// dirSize returns the total byte size and file count of a directory tree,
// or (0, 0, false) if the tree doesn't exist. We treat missing as "not yet
// populated" rather than as an error, since that's the common case on a
// fresh install.
func dirSize(root string) (int64, int, bool) {
	if _, err := os.Stat(root); err != nil {
		return 0, 0, false
	}
	var total int64
	var n int
	_ = filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if !info.IsDir() {
			total += info.Size()
			n++
		}
		return nil
	})
	return total, n, true
}

// humanBytes renders a byte count in IEC units. We use binary (1024)
// units because that's what file-system tools (du, ls -lh) typically
// report. Nothing in Go forces this choice; it's a stylistic call.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}

// must is a tiny helper that turns "(value, error)" into "value", swallowing
// the error. It's only safe here because every call site is to goEnv, and
// goEnv's only failure mode is "go binary not found" — in which case nothing
// in this program could work anyway.
func must(v string, _ error) string { return v }
