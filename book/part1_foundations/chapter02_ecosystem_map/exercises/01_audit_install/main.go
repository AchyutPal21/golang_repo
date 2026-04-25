// FILE: book/part1_foundations/chapter02_ecosystem_map/exercises/01_audit_install/main.go
// EXERCISE 2.1 — Audit your install.
//
// Run (from the chapter folder):
//   go run ./exercises/01_audit_install
//
// This is a richer version of examples/01_toolchain_tour: it adds a few
// derived checks ($GOBIN on $PATH? GOSUMDB enabled? GOPRIVATE set?) and
// flags anomalies. Use it to verify a fresh machine before starting
// real work.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	fmt.Printf("Go install audit — %s on %s/%s\n",
		runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Println(strings.Repeat("─", 60))

	keys := []string{
		"GOROOT", "GOPATH", "GOBIN", "GOMODCACHE", "GOCACHE",
		"GOPROXY", "GOSUMDB", "GOPRIVATE", "GOFLAGS", "CGO_ENABLED",
	}
	env := map[string]string{}
	for _, k := range keys {
		val, _ := goEnv(k)
		env[k] = val
		printRow(k, val)
	}

	fmt.Println()

	// ─── Derived checks ────────────────────────────────────────────────
	gobin := env["GOBIN"]
	if gobin == "" {
		gobin = filepath.Join(env["GOPATH"], "bin")
	}
	pathEntries := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	if !contains(pathEntries, gobin) {
		fmt.Printf("⚠  $GOBIN (%s) is not on $PATH — installed tools won't be found.\n", gobin)
	} else {
		fmt.Printf("✓  $GOBIN (%s) is on $PATH.\n", gobin)
	}

	switch env["GOSUMDB"] {
	case "off":
		fmt.Println("⚠  GOSUMDB is off — module integrity is NOT verified.")
	case "":
		fmt.Println("✓  GOSUMDB is at default (sum.golang.org).")
	default:
		fmt.Printf("ℹ  GOSUMDB is set to %q — make sure your team agrees.\n", env["GOSUMDB"])
	}

	if env["GOPROXY"] == "off" {
		fmt.Println("⚠  GOPROXY=off — module fetches will fail.")
	} else {
		fmt.Printf("✓  GOPROXY = %s\n", env["GOPROXY"])
	}

	if env["CGO_ENABLED"] == "0" {
		fmt.Println("ℹ  CGO_ENABLED=0 — fully static builds, but no cgo deps.")
	}

	// ─── Tools in $GOBIN ───────────────────────────────────────────────
	fmt.Println()
	fmt.Printf("Tools installed in %s:\n", gobin)
	if entries, err := os.ReadDir(gobin); err == nil {
		count := 0
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			fmt.Printf("  %s\n", e.Name())
			count++
		}
		if count == 0 {
			fmt.Println("  (none — try `go install golang.org/x/tools/gopls@latest`)")
		}
	} else {
		fmt.Println("  (directory not present)")
	}
}

func goEnv(name string) (string, error) {
	out, err := exec.Command("go", "env", name).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func printRow(k, v string) {
	if v == "" {
		v = "(unset)"
	}
	fmt.Printf("  %-12s %s\n", k, v)
}

func contains(xs []string, target string) bool {
	for _, x := range xs {
		if x == target {
			return true
		}
	}
	return false
}
