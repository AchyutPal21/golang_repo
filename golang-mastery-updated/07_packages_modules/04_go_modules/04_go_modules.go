// 04_go_modules.go
//
// GO MODULES — Deep Dive
//
// Go Modules is the official dependency management system for Go, introduced
// in Go 1.11 and made the default in Go 1.16. It solves the reproducibility
// and versioning problems of the old $GOPATH era.
//
// This file covers:
//   - go.mod anatomy: every directive explained
//   - Semantic versioning (semver) and how Go uses it
//   - The Minimal Version Selection (MVS) algorithm
//   - v2+ module paths (the "major version suffix" rule)
//   - Key go commands: go get, go mod tidy, go mod download, go mod verify
//   - GOMODCACHE — where modules live on disk
//   - The vendor directory and when to use it
//   - go.sum and the checksum database
//   - replace directive — local development and forking
//   - exclude directive
//   - Private modules and GOPRIVATE

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ============================================================
// PART 1: go.mod ANATOMY — EVERY DIRECTIVE
// ============================================================
//
// A go.mod file is a plain text file. Here is a fully annotated example:
//
//   module github.com/acme/myapp
//   ↑ The module path. This is the "root" of all import paths in this module.
//     Convention: use the VCS hosting URL (github.com/user/repo) so the
//     go toolchain can locate the module for others importing your code.
//     For internal/private modules, you can use any path that won't conflict
//     with public module paths (e.g., "corp.internal/myapp").
//
//   go 1.22
//   ↑ The minimum Go version required to build this module.
//     Does NOT prevent building with a newer Go version.
//     As of Go 1.21, this line also controls some language semantics
//     (e.g., loop variable capture behavior changed in 1.22).
//
//   toolchain go1.22.2
//   ↑ (Go 1.21+) Specifies the exact toolchain version to use.
//     If your installed Go is older, `go` will download the right version.
//     Optional but recommended for reproducibility.
//
//   require (
//       github.com/pkg/errors v0.9.1
//       go.uber.org/zap       v1.27.0
//   )
//   ↑ Direct dependencies with their MINIMUM required versions.
//     Go uses Minimal Version Selection — it picks the minimum version that
//     satisfies ALL requirements across the dependency graph.
//
//   require (
//       go.uber.org/atomic v1.11.0 // indirect
//   )
//   ↑ Indirect dependencies: packages your direct dependencies need,
//     but that you don't import directly. Marked with "// indirect".
//     "go mod tidy" manages these automatically.
//
//   replace github.com/acme/shared v1.2.3 => ../shared
//   ↑ Replaces a module with a different path or version.
//     Used for: local development, testing a fork, patching a dependency.
//     The right-hand side can be:
//       - A local directory path (../shared)
//       - Another module@version (github.com/fork/shared v1.2.4)
//     IMPORTANT: replace directives in dependencies are IGNORED by Go.
//     Only the top-level module's replace directives take effect.
//
//   exclude github.com/bad/module v1.0.0
//   ↑ Prevents a specific version from being used.
//     If the MVS algorithm would select v1.0.0, it will use the next
//     higher version instead.
//     Use case: a dependency has a known-bad version you want to skip.

func explainGoMod() {
	fmt.Println("--- go.mod directives ---")

	directives := []struct {
		name        string
		syntax      string
		description string
	}{
		{
			"module",
			"module github.com/acme/myapp",
			"Declares the module path (root of all import paths in this module)",
		},
		{
			"go",
			"go 1.22",
			"Minimum Go language version; may affect language semantics since 1.21",
		},
		{
			"toolchain",
			"toolchain go1.22.2",
			"(Go 1.21+) Exact toolchain; downloaded automatically if needed",
		},
		{
			"require",
			"require github.com/pkg/errors v0.9.1",
			"Declares a dependency with its minimum acceptable version",
		},
		{
			"replace",
			"replace mod v1 => ../local/path",
			"Swaps a dep with a different path/version (local dev, forks)",
		},
		{
			"exclude",
			"exclude bad/mod v1.0.0",
			"Prevents a specific version from being selected by MVS",
		},
		{
			"retract",
			"retract v1.0.1 // CVE-2024-12345",
			"(Go 1.16+) Marks versions as unsafe/retracted in the registry",
		},
	}

	for _, d := range directives {
		fmt.Printf("  %-12s  %-45s  %s\n", d.name, d.syntax, d.description)
	}
	fmt.Println()
}

// ============================================================
// PART 2: SEMANTIC VERSIONING (semver)
// ============================================================
//
// Go modules use semantic versioning: vMAJOR.MINOR.PATCH
//
//   PATCH increment: backward-compatible bug fixes (v1.2.3 → v1.2.4)
//   MINOR increment: backward-compatible new features (v1.2.3 → v1.3.0)
//   MAJOR increment: breaking changes (v1.2.3 → v2.0.0)
//
// Pre-release versions: v1.0.0-alpha.1, v1.0.0-rc.2
//   - Lower precedence than the release version (v1.0.0-alpha.1 < v1.0.0)
//   - Go modules do NOT automatically upgrade to pre-release versions.
//
// Pseudo-versions (for commits without a tag):
//   v0.0.0-20240101000000-abcdef012345
//   ↑ base version   timestamp  commit hash
//   Used by "go get" when you specify a branch or commit hash.
//   You'll see these in go.mod for untagged dependencies.
//
// THE v2+ MODULE PATH RULE (the "major version suffix"):
//   When a module makes breaking changes, it MUST increment the major version.
//   For v2 and above, the module path in go.mod MUST have a /v2 (or /vN) suffix:
//
//     module github.com/acme/myapp/v2
//
//   And every import of that module in consumer code must include the suffix:
//
//     import "github.com/acme/myapp/v2/pkg/auth"
//
//   WHY: This allows v1 and v2 to coexist in the same build.
//   A binary can import both v1 and v2 of the same logical library — they are
//   treated as completely different modules (different import paths).
//
//   This is called "import compatibility rule" — if a package is imported with
//   the same path, it must be backward compatible.
//
// Versions BEFORE v1.0.0:
//   v0.x.x — no stability guarantees. Breaking changes can happen at any minor/patch.
//   The v2+ rule only applies to v1.0.0 and above.
//   Many popular libraries stayed at v0.x.x for years to avoid the /vN path issue.

func explainSemver() {
	fmt.Println("--- Semantic versioning in Go modules ---")

	versions := []struct {
		version     string
		kind        string
		notes       string
	}{
		{"v1.2.3", "Release", "MAJOR.MINOR.PATCH — the standard form"},
		{"v1.2.4", "Patch", "Bug fix, backward-compatible"},
		{"v1.3.0", "Minor", "New features, backward-compatible"},
		{"v2.0.0", "Major", "Breaking change — module path gains /v2 suffix"},
		{"v1.0.0-alpha.1", "Pre-release", "Lower precedence than v1.0.0"},
		{"v0.0.0-20240101-abc123", "Pseudo", "Untagged commit; generated by go get"},
	}

	fmt.Printf("  %-28s  %-12s  %s\n", "Version", "Kind", "Notes")
	fmt.Printf("  %-28s  %-12s  %s\n", "-------", "----", "-----")
	for _, v := range versions {
		fmt.Printf("  %-28s  %-12s  %s\n", v.version, v.kind, v.notes)
	}
	fmt.Println()

	fmt.Println("  v2+ rule: module path must include /v2 (or /vN) suffix")
	fmt.Println("  go.mod:   module github.com/acme/myapp/v2")
	fmt.Println("  imports:  import \"github.com/acme/myapp/v2/config\"")
	fmt.Println()
}

// ============================================================
// PART 3: MINIMAL VERSION SELECTION (MVS)
// ============================================================
//
// Go's dependency resolution algorithm is called Minimal Version Selection (MVS).
// It was designed by Russ Cox to be simple, reproducible, and predictable.
//
// Core idea: given a set of minimum version requirements, always select
// the MINIMUM version that satisfies ALL of them. Never automatically
// upgrade to a newer version.
//
// Example:
//   Your module requires:
//     pkg/errors >= v0.8.0
//   Your other dependency requires:
//     pkg/errors >= v0.9.0
//
//   MVS selects: v0.9.0 (the minimum that satisfies BOTH requirements).
//   If v0.9.1 exists, MVS does NOT select it — you asked for minimum, you get minimum.
//
// Why MVS is better than other approaches (e.g., npm's "latest compatible"):
//   - REPRODUCIBLE: The same go.mod always produces the same build.
//   - PREDICTABLE: No surprise upgrades. You get exactly what you asked for.
//   - FAST: No SAT solver needed. MVS is O(n) in the number of modules.
//   - SAFE: A new patch release that introduces a regression won't silently sneak in.
//
// To upgrade a dependency:
//   go get github.com/pkg/errors@v0.9.1   → updates go.mod + go.sum
//
// To upgrade ALL dependencies to their latest minor/patch:
//   go get -u ./...        → upgrades everything (risky, test afterward!)
//   go get -u=patch ./...  → upgrades only patch versions (safer)

func explainMVS() {
	fmt.Println("--- Minimal Version Selection (MVS) ---")
	fmt.Println("  Principle: select the MINIMUM version satisfying ALL requirements.")
	fmt.Println("  Never auto-upgrade beyond the minimum.")
	fmt.Println()

	scenario := `
  Module A requires: libX >= v1.1.0
  Module B requires: libX >= v1.3.0
  Available:         v1.1.0, v1.2.0, v1.3.0, v1.4.0

  MVS selects: v1.3.0 (minimum satisfying both A and B)
  npm/bundler might select: v1.4.0 (latest compatible) — surprise upgrades!
`
	fmt.Println(scenario)

	fmt.Println("  Upgrade commands:")
	cmds := []struct{ cmd, effect string }{
		{"go get pkg@v1.3.0", "Pin to exactly v1.3.0"},
		{"go get pkg@latest", "Upgrade to latest tagged release"},
		{"go get -u=patch ./...", "Upgrade all deps to latest patch only"},
		{"go get -u ./...", "Upgrade all deps to latest minor+patch (risky)"},
		{"go mod tidy", "Add missing, remove unused deps from go.mod/go.sum"},
	}
	for _, c := range cmds {
		fmt.Printf("  %-35s → %s\n", c.cmd, c.effect)
	}
	fmt.Println()
}

// ============================================================
// PART 4: KEY go MODULE COMMANDS
// ============================================================

func explainGoCommands() {
	fmt.Println("--- Key go module commands ---")

	commands := []struct {
		cmd  string
		desc string
	}{
		{
			"go mod init github.com/acme/myapp",
			"Create go.mod in current directory",
		},
		{
			"go mod tidy",
			"Add missing + remove unused deps; update go.sum",
		},
		{
			"go mod download",
			"Download all deps to GOMODCACHE (no build)",
		},
		{
			"go mod verify",
			"Verify cached modules match go.sum hashes",
		},
		{
			"go mod vendor",
			"Copy all deps into ./vendor/ directory",
		},
		{
			"go mod graph",
			"Print the module dependency graph",
		},
		{
			"go mod why -m github.com/pkg/errors",
			"Explain why a module is needed",
		},
		{
			"go mod edit -replace old=new",
			"Add/edit a replace directive programmatically",
		},
		{
			"go list -m all",
			"List all modules in the build",
		},
		{
			"go list -m -versions github.com/pkg/errors",
			"List all available versions of a module",
		},
		{
			"go get github.com/pkg/errors@v0.9.1",
			"Add or upgrade a specific dependency version",
		},
		{
			"go clean -modcache",
			"Delete the entire module cache (use with care!)",
		},
	}

	for _, c := range commands {
		fmt.Printf("  %-45s  %s\n", c.cmd, c.desc)
	}
	fmt.Println()
}

// ============================================================
// PART 5: GOMODCACHE
// ============================================================
//
// $GOMODCACHE (default: $GOPATH/pkg/mod) is where the go toolchain stores
// downloaded module source code.
//
// Structure inside GOMODCACHE:
//   $GOPATH/pkg/mod/
//     github.com/pkg/errors@v0.9.1/
//       ├── errors.go
//       ├── go.mod
//       └── ...
//     go.uber.org/zap@v1.27.0/
//       └── ...
//
// Key properties:
//   - Module directories are READ-ONLY (go chmod's them after download).
//     This prevents accidental modification.
//   - A specific version is only downloaded ONCE, regardless of how many
//     modules in your build require it.
//   - The cache persists across projects on the same machine.
//   - "go clean -modcache" deletes everything — you'll re-download on next build.
//   - In CI, caching $GOPATH/pkg/mod between runs speeds up builds significantly.
//
// GOMODCACHE vs GOPATH:
//   GOPATH was the workspace root in the old era.
//   GOMODCACHE is just the download cache — you no longer develop IN it.

func explainGOMODCACHE() {
	fmt.Println("--- GOMODCACHE ---")

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		// Default GOPATH if not set
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}
	gomodcache := os.Getenv("GOMODCACHE")
	if gomodcache == "" {
		gomodcache = filepath.Join(gopath, "pkg", "mod")
	}

	fmt.Printf("  GOPATH      = %s\n", gopath)
	fmt.Printf("  GOMODCACHE  = %s\n", gomodcache)
	fmt.Printf("  GOROOT      = %s\n", runtime.GOROOT())
	fmt.Println()
	fmt.Println("  Module cache layout:")
	fmt.Println("    $GOMODCACHE/")
	fmt.Println("      github.com/pkg/errors@v0.9.1/  ← read-only after download")
	fmt.Println("      go.uber.org/zap@v1.27.0/")
	fmt.Println()
	fmt.Println("  CI tip: cache $GOPATH/pkg/mod between pipeline runs to avoid re-downloading.")
	fmt.Println()
}

// ============================================================
// PART 6: THE vendor DIRECTORY
// ============================================================
//
// "go mod vendor" copies all dependency source code into ./vendor/
//
// Structure:
//   vendor/
//     modules.txt          ← metadata listing all vendored modules
//     github.com/
//       pkg/errors/
//         v0.9.1/
//           errors.go
//
// When to use vendor/:
//   1. Air-gapped environments: no internet access during build.
//   2. Regulatory requirements: some industries require all source in the repo.
//   3. Build reproducibility without relying on the module proxy.
//   4. Faster CI builds (no network I/O needed).
//
// When NOT to use vendor/:
//   - General development: vendor/ bloats your repository significantly.
//   - When you trust the module proxy + go.sum for reproducibility.
//
// Build with vendored deps:
//   go build -mod=vendor ./...
//
// Or set GOFLAGS=-mod=vendor globally.
//
// Since Go 1.14, if vendor/ exists and go.mod says go 1.14+,
// go build uses vendor/ automatically. Set -mod=mod to override.

func explainVendor() {
	fmt.Println("--- vendor/ directory ---")
	fmt.Println("  go mod vendor  →  copies all deps into ./vendor/")
	fmt.Println()
	fmt.Println("  Use vendor/ when:")
	reasons := []string{
		"Air-gapped/offline builds (no internet access)",
		"Regulatory: all source code must be in the repository",
		"Faster CI (skip network I/O)",
		"Pinning exact source for audit purposes",
	}
	for _, r := range reasons {
		fmt.Printf("    + %s\n", r)
	}
	fmt.Println()
	fmt.Println("  Build with vendor/:")
	fmt.Println("    go build -mod=vendor ./...")
	fmt.Println("    GOFLAGS=-mod=vendor go build ./...")
	fmt.Println()
}

// ============================================================
// PART 7: replace DIRECTIVE — LOCAL DEVELOPMENT AND FORKS
// ============================================================
//
// The replace directive swaps one module (or version) for another.
//
// Use case 1 — Local development of multiple modules simultaneously:
//   You're developing myapp and mylib at the same time. Instead of pushing
//   every change to mylib and running "go get" in myapp, use:
//
//     replace github.com/acme/mylib => ../mylib
//
//   Now myapp uses the local directory ../mylib as the source.
//
// Use case 2 — Forking a dependency to apply a patch:
//   You found a bug in upstream/lib. You forked it and fixed it.
//   Until your PR is merged, use:
//
//     replace upstream/lib v1.0.0 => github.com/you/lib v1.0.1-patched
//
// Use case 3 — Redirecting to a mirror:
//   replace golang.org/x/net => github.com/mirror/net v0.20.0
//
// IMPORTANT RESTRICTIONS:
//   - replace directives only take effect in the MAIN MODULE (top-level go.mod).
//   - If a dependency uses a replace directive in its go.mod, Go IGNORES it.
//   - This is by design: you don't want your dependencies secretly replacing modules.
//   - Replace directives involving local paths make it harder to run "go get" from CI.
//     Remember to remove or comment them before releasing.

func explainReplace() {
	fmt.Println("--- replace directive ---")

	examples := []struct {
		scenario string
		syntax   string
	}{
		{
			"Local development (relative path)",
			"replace github.com/acme/mylib => ../mylib",
		},
		{
			"Fork with patch applied",
			"replace upstream/lib v1.0.0 => github.com/you/lib v1.0.1-patched",
		},
		{
			"Specific version replaced by specific version",
			"replace bad/pkg v1.2.0 => good/pkg v1.2.1",
		},
		{
			"All versions of a module replaced",
			"replace bad/pkg => good/pkg v1.2.1",
		},
	}

	for _, ex := range examples {
		fmt.Printf("  Scenario: %s\n", ex.scenario)
		fmt.Printf("  Syntax:   %s\n", ex.syntax)
		fmt.Println()
	}

	fmt.Println("  WARNING: replace only works in the main module's go.mod.")
	fmt.Println("  Dependencies' replace directives are silently ignored.")
	fmt.Println()
}

// ============================================================
// PART 8: PRIVATE MODULES — GOPRIVATE, GONOSUMDB, GONOPROXY
// ============================================================
//
// By default, go fetches modules through proxy.golang.org (the public proxy)
// and verifies checksums against sum.golang.org (the checksum DB).
//
// For PRIVATE repositories (e.g., internal company code):
//   - The public proxy cannot access private GitHub/GitLab repos.
//   - You don't want private module paths leaked to the public checksum DB.
//
// Environment variables:
//
//   GOPRIVATE=corp.internal/*,github.com/acme/*
//     Comma-separated glob patterns. Matching modules skip BOTH the proxy
//     AND the checksum database. Equivalent to setting GONOSUMDB + GONOPROXY
//     to the same pattern.
//
//   GONOPROXY=corp.internal/*
//     Skip the proxy for matching modules (fetch directly from VCS).
//
//   GONOSUMDB=corp.internal/*
//     Skip checksum DB verification for matching modules.
//
//   GOPROXY=https://proxy.company.com,direct
//     Use a corporate proxy first, fall back to direct VCS fetch.
//     "direct" means fetch from VCS without a proxy.
//     "off" means fail if module not in cache.
//
// Typical setup for mixed public/private:
//   GOPROXY=https://proxy.golang.org,direct
//   GOPRIVATE=github.com/mycompany/*,corp.internal/*
//
// For CI/CD in a corporate environment, many teams run Athens (an open-source
// Go module proxy) to cache public modules and serve private ones.

func explainPrivateModules() {
	fmt.Println("--- Private modules ---")

	envVars := []struct {
		name    string
		example string
		purpose string
	}{
		{
			"GOPRIVATE",
			"corp.internal/*,github.com/acme/*",
			"Skip proxy AND checksum DB for matching modules",
		},
		{
			"GONOPROXY",
			"corp.internal/*",
			"Skip proxy for matching modules (fetch direct from VCS)",
		},
		{
			"GONOSUMDB",
			"corp.internal/*",
			"Skip checksum DB for matching modules",
		},
		{
			"GOPROXY",
			"https://proxy.golang.org,direct",
			"Module proxy chain; 'direct' = bypass proxy; 'off' = no fetch",
		},
	}

	for _, e := range envVars {
		fmt.Printf("  %-12s = %-45s  %s\n", e.name, e.example, e.purpose)
	}
	fmt.Println()
}

// ============================================================
// PART 9: LIVE DEMO — INSPECT go env
// ============================================================

func demoGoEnv() {
	fmt.Println("--- go env (relevant module-related variables) ---")

	keys := []string{
		"GOPATH", "GOROOT", "GOMODCACHE", "GOMODULE",
		"GOPROXY", "GONOSUMDB", "GOPRIVATE", "GOFLAGS",
	}

	for _, key := range keys {
		val := os.Getenv(key)
		if val == "" {
			// Try via "go env" command for values with defaults
			cmd := exec.Command("go", "env", key)
			out, err := cmd.Output()
			if err == nil {
				val = strings.TrimSpace(string(out))
			}
		}
		if val == "" {
			val = "(not set / using default)"
		}
		fmt.Printf("  %-15s = %s\n", key, val)
	}
	fmt.Println()
}

// ============================================================
// MAIN
// ============================================================

func main() {
	fmt.Println("=== 04: Go Modules ===")
	fmt.Println()

	explainGoMod()
	explainSemver()
	explainMVS()
	explainGoCommands()
	explainGOMODCACHE()
	explainVendor()
	explainReplace()
	explainPrivateModules()
	demoGoEnv()

	// Final checklist
	fmt.Println("--- Module workflow checklist ---")
	checklist := []string{
		"go mod init <module-path>   → start a new module",
		"Write your code, add imports",
		"go mod tidy                 → resolve dependencies, update go.sum",
		"go build ./...              → verify everything compiles",
		"git add go.mod go.sum       → commit BOTH files",
		"go get pkg@version          → update a specific dep",
		"go mod tidy                 → clean up after any changes",
		"go mod verify               → verify cache integrity",
	}
	for i, step := range checklist {
		fmt.Printf("  %d. %s\n", i+1, step)
	}
	fmt.Println()

	fmt.Println("=== End of 04_go_modules.go ===")
}
