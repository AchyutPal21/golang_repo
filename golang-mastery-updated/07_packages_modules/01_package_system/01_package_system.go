// 01_package_system.go
//
// THE GO PACKAGE SYSTEM — Deep Dive
//
// A package is Go's fundamental unit of code organization and encapsulation.
// Understanding the package system is essential to writing idiomatic Go code
// that scales from small scripts to large distributed systems.
//
// This file covers:
//   - What a package actually is (a directory of .go files)
//   - package main vs library packages
//   - Import paths and how Go resolves them
//   - The history of $GOPATH and the transition to Go Modules
//   - Package-level variables and init() order
//   - Why Go forbids circular imports — and how to design around them
//   - Package documentation conventions (godoc)

package main

import (
	"fmt"
	"math"
)

// ============================================================
// PART 1: WHAT IS A PACKAGE?
// ============================================================
//
// A Go package is simply a DIRECTORY of .go files that all share
// the same "package <name>" declaration at the top.
//
// Key rules:
//  1. Every .go file must belong to exactly one package.
//  2. All .go files in the same directory MUST declare the same package name
//     (exception: _test.go files can use "package foo_test" for black-box tests).
//  3. The package name (what you write in code) does NOT have to match the
//     directory name — but by convention it should (except "main").
//  4. The import PATH is the directory path; the identifier you use in code
//     is the package NAME.
//
// Example layout:
//
//   myproject/
//   ├── go.mod              ← module root
//   ├── main.go             ← package main
//   └── internal/
//       └── mathutil/
//           ├── add.go      ← package mathutil
//           └── mul.go      ← package mathutil  (same dir → same package)
//
// When you write:
//   import "myproject/internal/mathutil"
//
// Go finds the directory "internal/mathutil/" relative to the module root.
// You refer to it in code as "mathutil.Add()" — using the PACKAGE NAME, not
// the last directory segment (they happen to match here, but need not).
//
// Counterexample of name ≠ directory:
//   Directory: "gopkg.in/yaml.v3"
//   Package name declared inside: "yaml"
//   Usage: yaml.Unmarshal(...)   (not v3.Unmarshal)

// ============================================================
// PART 2: package main — THE ENTRY POINT PACKAGE
// ============================================================
//
// "package main" is special. It is the ONLY package that produces an
// executable binary when you run "go build" or "go run".
//
// Rules for package main:
//  - Must contain a function named "main" with signature: func main()
//  - "main" is the entry point; the runtime calls it after all init() functions run.
//  - You can have multiple files in a package main (all in the same directory).
//  - A package main cannot be imported by other packages.
//    (The toolchain enforces this — importing "main" is a compile error.)
//
// Library packages (everything else):
//  - Named anything except "main" (by convention: short, lowercase, no underscores).
//  - Produce a .a archive (static library) that the linker embeds into binaries.
//  - They ARE importable.
//  - They must NOT define func main() as an entry point (though they can define
//    a function called "main" for other purposes — but this is confusing, avoid it).
//
// WHY the distinction?
//   The Go toolchain needs to know which package to use as the program's root.
//   "main" is an unambiguous signal: "start here, this is the binary."

// ============================================================
// PART 3: IMPORT PATHS — HOW GO FINDS PACKAGES
// ============================================================
//
// An import path is a string like:
//   "fmt"                          — stdlib (resolved from GOROOT)
//   "golang.org/x/sync/errgroup"   — external module (resolved from module cache)
//   "myapp/internal/config"        — internal package (resolved from module root)
//
// Resolution algorithm (simplified):
//   1. Is it a standard library package? → use $GOROOT/src/<path>
//   2. Otherwise, look at go.mod: find which module provides this path,
//      download it if necessary, use the cached copy in $GOMODCACHE.
//
// The module path in go.mod (e.g., "module myapp") defines the ROOT of all
// import paths for that module. A file at myapp/foo/bar.go is imported as
// "myapp/foo/bar".
//
// IMPORTANT: You never import a package by file path (no .go extension,
// no OS-specific slashes in your code). Import paths use forward slashes
// regardless of operating system.

// ============================================================
// PART 4: $GOPATH HISTORY vs GO MODULES
// ============================================================
//
// The OLD world ($GOPATH, before Go 1.11):
//   - All Go code lived in a single workspace: $GOPATH/src/
//   - Import paths were relative to $GOPATH/src/
//   - e.g., $GOPATH/src/github.com/user/project/...
//   - Problems:
//       • Only one version of any dependency could exist in $GOPATH at a time.
//       • No reproducible builds — "go get" always fetched HEAD.
//       • Sharing code required everyone to have the same $GOPATH layout.
//       • Large teams hit constant version conflicts.
//
// The NEW world (Go Modules, introduced Go 1.11, default since Go 1.16):
//   - A module is defined by a go.mod file at its root.
//   - The module can live ANYWHERE on your filesystem — $GOPATH is no longer
//     required (though $GOMODCACHE still lives inside $GOPATH by default).
//   - Each module has explicit, versioned dependencies in go.mod + go.sum.
//   - Multiple versions of the same library can coexist (via replace directives
//     or different modules importing different versions — the Minimal Version
//     Selection algorithm resolves conflicts).
//   - Reproducible builds: go.sum contains cryptographic hashes of every
//     dependency, so "go build" always produces the same binary.
//
// Key environment variables today:
//   $GOPATH     — still exists; $GOPATH/pkg/mod is the module cache (GOMODCACHE)
//   $GOROOT     — Go installation directory (stdlib lives here)
//   $GOMODCACHE — where downloaded modules are cached (default: $GOPATH/pkg/mod)
//   $GOFLAGS    — default flags for go commands
//   $GOPROXY    — proxy server for fetching modules (default: proxy.golang.org)
//   $GONOSUMDB  — pattern of modules to skip checksum verification
//   $GOPRIVATE  — pattern of private modules (skips proxy + sum DB)

// ============================================================
// PART 5: PACKAGE-LEVEL VARIABLES AND INIT ORDER
// ============================================================
//
// Go initializes package-level variables in DEPENDENCY ORDER before main() runs.
// If variable A depends on variable B, B is initialized first.
//
// After all package-level variables are initialized, Go calls all init()
// functions in the order they appear in source files (files processed
// alphabetically by filename within a package).
//
// Key facts about init():
//   - init() takes no arguments and returns nothing: func init()
//   - A single file can have MULTIPLE init() functions (unusual but valid).
//   - init() cannot be called directly from user code.
//   - init() runs before main() in package main, and before any exported
//     function is called in library packages.
//   - If package A imports package B, package B's init() runs BEFORE package A's init().
//   - The same package imported by multiple packages is only initialized ONCE.
//
// Common uses of init():
//   - Registering codecs, drivers, or plugins (e.g., database/sql drivers use
//     blank import + init() to self-register).
//   - Validating configuration that requires runtime checks.
//   - Setting up package-level caches or lookup tables.
//
// Warning: Heavy use of init() makes code harder to test and reason about.
//          Prefer explicit initialization (constructor functions) when possible.

// Package-level variables — initialized before main().
// The initializer expressions are evaluated in dependency order.
var (
	// baseRadius is initialized to a constant directly.
	baseRadius = 5.0

	// circleArea depends on baseRadius — Go ensures baseRadius is set first.
	circleArea = math.Pi * baseRadius * baseRadius

	// greeting is initialized using a function call.
	greeting = buildGreeting("Go Package System")
)

func buildGreeting(topic string) string {
	return fmt.Sprintf("Welcome to: %s", topic)
}

// init() runs after ALL package-level vars are initialized.
// This file has two init() functions — both will run, in order.
func init() {
	fmt.Println("[init #1] Package-level vars are ready.")
	fmt.Printf("[init #1] baseRadius=%.1f, circleArea=%.4f\n", baseRadius, circleArea)
}

func init() {
	fmt.Println("[init #2] A second init() in the same file — perfectly legal.")
}

// ============================================================
// PART 6: WHY GO FORBIDS CIRCULAR IMPORTS
// ============================================================
//
// Go's compiler REJECTS circular imports. If package A imports B, and B
// imports A, compilation fails immediately.
//
// WHY is this a feature, not a limitation?
//
//   1. Fast compilation: Go compiles packages in dependency order. Circular
//      deps make this impossible — you cannot compile A without B, but cannot
//      compile B without A.
//
//   2. Clean architecture: Circular deps are almost always a sign of poor
//      separation of concerns. Removing them forces better design.
//
//   3. Predictable initialization: The init() order described above works
//      because the dependency graph is a DAG (Directed Acyclic Graph). Cycles
//      would make initialization order undefined.
//
// How to BREAK circular imports (common patterns):
//
//   Pattern 1 — Extract a shared types package:
//     If A and B both need type Foo, create package "types" or "model" that
//     defines Foo. Both A and B import "types"; neither imports the other.
//
//   Pattern 2 — Use an interface:
//     If A calls a function in B, define an interface in A that B satisfies.
//     A depends on the interface; B depends on A's interface type.
//     This inverts the dependency (Dependency Inversion Principle).
//
//   Pattern 3 — Merge packages:
//     If two packages are so tightly coupled they need each other, they
//     probably belong in the same package. Merge them.
//
//   Pattern 4 — Introduce a third "coordinator" package:
//     A and B both stay pure. A new package C imports both A and B and
//     orchestrates the interaction.
//
// Architectural principle: dependencies should flow in ONE direction.
//   e.g.,  main → service → repository → model
//   Never:  repository → service (that's a cycle waiting to happen)

// ============================================================
// PART 7: PACKAGE NAMING CONVENTIONS
// ============================================================
//
// Go has strong conventions (enforced by community, not compiler):
//
//   - Short, lowercase, single word: "json", "http", "sync", "io"
//   - No underscores, no mixedCase: NOT "my_utils", NOT "myUtils"
//   - The name should describe what the package PROVIDES, not what it IS.
//     Good: "parser", "auth", "config"
//     Bad:  "utils", "helpers", "common", "misc"  ← these say nothing
//   - Avoid stutter: if the package is "bytes", don't name a function
//     "bytes.BytesReader" — "bytes.Reader" is better.
//   - test files: use "package foo_test" for black-box tests (external test package).
//
// The "internal" directory:
//   A package at path "a/b/internal/c" can ONLY be imported by code rooted
//   at "a/b". External modules cannot import it. This is enforced by the compiler.
//   Use internal/ to hide implementation details that you want to refactor freely
//   without worrying about external consumers breaking.

// ============================================================
// PART 8: PACKAGE DOCUMENTATION CONVENTIONS (godoc)
// ============================================================
//
// godoc extracts documentation from comments. Rules:
//
//   - A package doc comment goes immediately before the "package" declaration,
//     with NO blank line between the comment and "package".
//   - A declaration doc comment goes immediately before the declaration.
//   - Start the comment with the name of the thing being documented:
//       // Parse parses a JSON value from r.
//       func Parse(r io.Reader) (Value, error)
//   - Use full sentences. Godoc renders the first sentence as a summary.
//   - For packages, the top comment describes the whole package:
//       // Package json implements encoding and decoding of JSON.
//   - Doc comments support a lightweight markup:
//       • Paragraphs: blank line between them
//       • Code blocks: indented lines
//       • Links: [text](url) or [PackageName.Func]
//       • Lists: lines starting with " - " or " * "
//   - The "Deprecated:" prefix signals deprecation to tooling.
//
// Run "go doc <package>" or "godoc -http=:6060" to browse docs locally.
// pkg.go.dev is the public documentation site.

// Calc is an example of a well-documented exported type.
// It demonstrates how doc comments attach to declarations.
//
// A Calc performs basic geometric calculations. Create one with NewCalc.
type Calc struct {
	// Precision is the number of decimal places used in string output.
	// It is exported so callers can adjust formatting.
	Precision int

	// radius is unexported — callers cannot set it directly.
	// They must use SetRadius, which validates the input.
	radius float64
}

// NewCalc returns a Calc initialized with the given radius and default precision of 2.
// It returns an error if radius is not positive.
func NewCalc(radius float64) (*Calc, error) {
	if radius <= 0 {
		return nil, fmt.Errorf("NewCalc: radius must be positive, got %f", radius)
	}
	return &Calc{Precision: 2, radius: radius}, nil
}

// Area returns the area of a circle with the receiver's radius.
// Uses the formula: π × r²
func (c *Calc) Area() float64 {
	return math.Pi * c.radius * c.radius
}

// SetRadius updates the radius. Returns an error if r ≤ 0.
//
// Deprecated: Use NewCalc to create a fresh Calc instead.
func (c *Calc) SetRadius(r float64) error {
	if r <= 0 {
		return fmt.Errorf("SetRadius: radius must be positive, got %f", r)
	}
	c.radius = r
	return nil
}

// ============================================================
// MAIN — Demonstrates everything above
// ============================================================

func main() {
	fmt.Println("=== 01: The Go Package System ===")
	fmt.Println()

	// init() functions already ran before main() — their output appeared first.
	fmt.Printf("greeting (package-level var): %q\n", greeting)
	fmt.Printf("circleArea (package-level var): %.4f\n", circleArea)
	fmt.Println()

	// --- Package documentation demo ---
	fmt.Println("--- Well-documented Calc type ---")
	c, err := NewCalc(7.5)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Calc{radius: 7.5} → Area = %.4f\n", c.Area())

	_, err = NewCalc(-3)
	fmt.Println("NewCalc(-3) error:", err)
	fmt.Println()

	// --- Naming convention illustration ---
	fmt.Println("--- Package naming recap ---")
	namingExamples := []struct{ bad, good string }{
		{"myUtils", "util"},
		{"helpers", "parse"},
		{"CommonTypes", "types"},
		{"http_client", "httpclient"},
	}
	for _, ex := range namingExamples {
		fmt.Printf("  Bad: %-15s → Good: %s\n", ex.bad, ex.good)
	}
	fmt.Println()

	// --- Circular import avoidance strategies ---
	fmt.Println("--- Circular import avoidance strategies ---")
	strategies := []string{
		"1. Extract shared types into a dedicated 'types' or 'model' package",
		"2. Define interfaces in the consumer package (Dependency Inversion)",
		"3. Merge tightly coupled packages into one",
		"4. Introduce a coordinator package that imports both",
	}
	for _, s := range strategies {
		fmt.Println(" ", s)
	}
	fmt.Println()

	// --- $GOPATH vs Modules summary ---
	fmt.Println("--- GOPATH era vs Modules era ---")
	comparison := [][2]string{
		{"Code location", "$GOPATH/src/... (fixed)  vs  anywhere on disk"},
		{"Versioning", "no versions (always HEAD)  vs  semantic versions in go.mod"},
		{"Reproducibility", "not guaranteed  vs  cryptographic hashes in go.sum"},
		{"Multi-version", "impossible  vs  possible via MVS algorithm"},
		{"Private code", "hard (GOPATH layout required)  vs  GOPRIVATE env var"},
	}
	fmt.Printf("  %-20s  %s\n", "Aspect", "Comparison")
	fmt.Printf("  %-20s  %s\n", "------", "----------")
	for _, row := range comparison {
		fmt.Printf("  %-20s  %s\n", row[0], row[1])
	}
	fmt.Println()

	fmt.Println("=== End of 01_package_system.go ===")
}
