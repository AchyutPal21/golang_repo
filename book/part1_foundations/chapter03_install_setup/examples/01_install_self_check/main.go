// FILE: book/part1_foundations/chapter03_install_setup/examples/01_install_self_check/main.go
// CHAPTER: 03 — Installing Go and Setting Up Your Environment
// TOPIC: A self-check that validates the install end-to-end.
//
// Run (from the chapter folder):
//   go run ./examples/01_install_self_check
//
// Exits non-zero if anything is misconfigured. Use it as the last step of
// onboarding and as a daily diagnostic.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// minGoMinor is the minimum Go minor version we accept. The book targets
// 1.22+; tighten this in your team's fork to whatever your floor is.
const minGoMinor = 22

// check is a single diagnostic step: a label, a function that returns
// (ok, detail), and whether the failure should fail the whole run.
type check struct {
	label  string
	run    func() (ok bool, detail string)
	hard   bool // hard failures cause a non-zero exit code
}

func main() {
	checks := []check{
		{"Go toolchain is on $PATH", goOnPath, true},
		{"Go version is recent enough", goVersionOK, true},
		{"$GOPATH is set or defaults sensibly", gopathOK, false},
		{"$GOBIN (or $GOPATH/bin) is on $PATH", gobinOnPath, true},
		{"gopls is installed", toolPresent("gopls"), false},
		{"golangci-lint is installed", toolPresent("golangci-lint"), false},
		{"govulncheck is installed", toolPresent("govulncheck"), false},
		{"GOPROXY is set to something sensible", goproxyOK, false},
		{"GOSUMDB is enabled", gosumdbOK, false},
	}

	hardFails := 0
	for _, c := range checks {
		ok, detail := c.run()
		mark := "✓"
		if !ok {
			if c.hard {
				mark = "✗"
				hardFails++
			} else {
				mark = "•"
			}
		}
		fmt.Printf("  %s %s\n      %s\n", mark, c.label, detail)
	}

	fmt.Println()
	if hardFails > 0 {
		fmt.Printf("%d hard failure(s). Fix them before continuing.\n", hardFails)
		os.Exit(1)
	}
	fmt.Println("Install looks good.")
}

func goOnPath() (bool, string) {
	path, err := exec.LookPath("go")
	if err != nil {
		return false, "go binary not found on $PATH"
	}
	return true, path
}

func goVersionOK() (bool, string) {
	major, minor, raw, err := goMinor()
	if err != nil {
		return false, "could not determine Go version: " + err.Error()
	}
	if major != 1 {
		return false, fmt.Sprintf("unexpected major Go version: %s", raw)
	}
	if minor < minGoMinor {
		return false, fmt.Sprintf("Go %s is older than required 1.%d", raw, minGoMinor)
	}
	return true, raw
}

func gopathOK() (bool, string) {
	gopath, err := goEnv("GOPATH")
	if err != nil {
		return false, err.Error()
	}
	if gopath == "" {
		return false, "GOPATH is empty"
	}
	return true, gopath
}

func gobinOnPath() (bool, string) {
	gobin, _ := goEnv("GOBIN")
	if gobin == "" {
		gp, _ := goEnv("GOPATH")
		gobin = filepath.Join(gp, "bin")
	}
	pathEntries := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	for _, p := range pathEntries {
		if p == gobin {
			return true, gobin + " is on $PATH"
		}
	}
	return false, gobin + " is NOT on $PATH (add it to your shell rc)"
}

func toolPresent(name string) func() (bool, string) {
	return func() (bool, string) {
		path, err := exec.LookPath(name)
		if err != nil {
			return false, fmt.Sprintf("%s not found (try `go install ...`)", name)
		}
		return true, path
	}
}

func goproxyOK() (bool, string) {
	v, err := goEnv("GOPROXY")
	if err != nil || v == "" {
		return false, "GOPROXY unset"
	}
	if v == "off" {
		return false, "GOPROXY=off — module fetches will fail"
	}
	return true, v
}

func gosumdbOK() (bool, string) {
	v, err := goEnv("GOSUMDB")
	if err != nil {
		return false, err.Error()
	}
	if v == "off" {
		return false, "GOSUMDB=off — module integrity NOT verified"
	}
	if v == "" {
		v = "sum.golang.org (default)"
	}
	return true, v
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func goEnv(name string) (string, error) {
	out, err := exec.Command("go", "env", name).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// goMinor returns (major, minor, raw, err). It parses runtime.Version, which
// looks like "go1.22.5". We don't shell out — runtime.Version is reliable
// for the program's own binary, and that's what we care about.
func goMinor() (int, int, string, error) {
	raw := runtime.Version()
	v := strings.TrimPrefix(raw, "go")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return 0, 0, raw, fmt.Errorf("unparseable: %s", raw)
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0, raw, fmt.Errorf("unparseable: %s", raw)
	}
	return major, minor, raw, nil
}
