// FILE: book/part1_foundations/chapter02_ecosystem_map/examples/03_stdlib_inventory/main.go
// CHAPTER: 02 — A Map of the Go Ecosystem
// TOPIC: How big is the standard library, and what's in it?
//
// Run (from the chapter folder):
//   go run ./examples/03_stdlib_inventory
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   Most newcomers underestimate how much ships in the box. This walks
//   $GOROOT/src and counts top-level packages by category, so you see the
//   surface area as a single screen. The categorisation is hand-rolled —
//   Go has no formal categories — but it matches how senior Go engineers
//   talk about the stdlib at a high level.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// category groups stdlib top-level packages into the buckets we use to
// reason about them. The first matching prefix wins, so order matters.
// "runtime" must come before "rune-related" packages (none currently),
// but the principle is the safer general-purpose pattern.
type category struct {
	name     string
	prefixes []string
}

var categories = []category{
	{"runtime & language", []string{"builtin", "runtime", "errors", "unsafe", "reflect", "go", "internal"}},
	{"I/O & filesystem", []string{"io", "os", "bufio", "path"}},
	{"text & strings", []string{"strings", "bytes", "strconv", "unicode", "fmt", "regexp", "text"}},
	{"time", []string{"time"}},
	{"collections & sort", []string{"sort", "slices", "maps", "container", "iter"}},
	{"concurrency", []string{"sync", "context"}},
	{"networking", []string{"net"}},
	{"crypto", []string{"crypto", "hash"}},
	{"encoding", []string{"encoding"}},
	{"templates", []string{"html", "text/template"}},
	{"database", []string{"database"}},
	{"compression & archives", []string{"compress", "archive"}},
	{"image & color", []string{"image"}},
	{"math", []string{"math"}},
	{"logging", []string{"log"}},
	{"testing", []string{"testing"}},
	{"plugins & cgo", []string{"plugin", "cgo", "syscall"}},
	{"debugging", []string{"debug"}},
	{"build & embedding", []string{"embed", "flag"}},
	{"misc", []string{"mime", "expvar"}},
}

func main() {
	root, err := goRootSrc()
	if err != nil {
		fmt.Println("could not locate Go SDK source:", err)
		os.Exit(1)
	}

	pkgs, err := walkPackages(root)
	if err != nil {
		fmt.Println("walk failed:", err)
		os.Exit(1)
	}

	// Bucket each package into its category.
	bucket := map[string][]string{}
	uncategorised := []string{}
	for _, p := range pkgs {
		c := classify(p)
		if c == "" {
			uncategorised = append(uncategorised, p)
			continue
		}
		bucket[c] = append(bucket[c], p)
	}

	// Print in the curated category order so the output is stable.
	fmt.Printf("Standard library inventory (from %s)\n", root)
	fmt.Println(strings.Repeat("─", 60))
	total := 0
	for _, c := range categories {
		ps := bucket[c.name]
		if len(ps) == 0 {
			continue
		}
		sort.Strings(ps)
		fmt.Printf("\n%s (%d):\n", c.name, len(ps))
		for _, p := range ps {
			fmt.Printf("  %s\n", p)
		}
		total += len(ps)
	}
	if len(uncategorised) > 0 {
		sort.Strings(uncategorised)
		fmt.Printf("\nuncategorised (%d):\n", len(uncategorised))
		for _, p := range uncategorised {
			fmt.Printf("  %s\n", p)
		}
		total += len(uncategorised)
	}
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Total top-level packages: %d\n", total)
	fmt.Println("Run `go doc <pkg>` on any of them.")
}

// goRootSrc returns $(go env GOROOT)/src. We shell out rather than read
// runtime.GOROOT() because the latter is the value the toolchain was
// *built* with, which can differ from the toolchain's runtime location
// in some installations.
func goRootSrc() (string, error) {
	out, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return "", err
	}
	root := strings.TrimSpace(string(out))
	return filepath.Join(root, "src"), nil
}

// walkPackages returns every top-level package directory under $GOROOT/src,
// excluding cmd/ (which holds the compiler/linker/etc.) and vendor/ (which
// holds vendored deps for the toolchain itself).
//
// "Top-level" here means "directories one or two levels below src/" that
// contain at least one .go file. We stop the descent at depth 2 because
// going deeper would conflate user-facing packages (encoding/json) with
// their internal helpers (encoding/json/internal/...).
func walkPackages(root string) ([]string, error) {
	var pkgs []string

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "cmd" || name == "vendor" || name == "internal" {
			continue
		}
		if hasGoFiles(filepath.Join(root, name)) {
			pkgs = append(pkgs, name)
		}
		// Look one level deeper, e.g. encoding/json under encoding.
		subEntries, err := os.ReadDir(filepath.Join(root, name))
		if err != nil {
			continue
		}
		for _, se := range subEntries {
			if !se.IsDir() || se.Name() == "internal" || se.Name() == "testdata" {
				continue
			}
			full := filepath.Join(root, name, se.Name())
			if hasGoFiles(full) {
				pkgs = append(pkgs, name+"/"+se.Name())
			}
		}
	}
	return pkgs, nil
}

func hasGoFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
			return true
		}
	}
	return false
}

func classify(pkg string) string {
	for _, c := range categories {
		for _, prefix := range c.prefixes {
			if pkg == prefix || strings.HasPrefix(pkg, prefix+"/") {
				return c.name
			}
		}
	}
	return ""
}
