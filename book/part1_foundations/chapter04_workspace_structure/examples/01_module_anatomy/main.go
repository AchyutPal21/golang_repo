// FILE: book/part1_foundations/chapter04_workspace_structure/examples/01_module_anatomy/main.go
// CHAPTER: 04 — The Go Workspace and Project Structure
// TOPIC: Surface a Go file's module path, package path, and exports.
//
// Run (from the chapter folder):
//   go run ./examples/01_module_anatomy <path-to-go-file>
//   go run ./examples/01_module_anatomy ../03_install_setup/examples/02_editor_smoketest/main.go
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   Beginners often guess at the relationship between a file's location, its
//   module, its package, and its import path. This program *computes* all
//   four for any .go file you point at, so you can see the rules in action.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: 01_module_anatomy <go-file>")
		os.Exit(2)
	}
	target, err := filepath.Abs(os.Args[1])
	if err != nil {
		die("absolute path:", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		die("stat:", err)
	}
	if info.IsDir() || !strings.HasSuffix(target, ".go") {
		die("not a .go file:", target)
	}

	// ─── Find the enclosing module ─────────────────────────────────────
	//
	// We shell out to `go env GOMOD` from the file's *directory*, since
	// that env var is computed relative to the working directory. This is
	// the toolchain's own answer; we don't reinvent module discovery.
	dir := filepath.Dir(target)
	gomod, err := goEnvFrom(dir, "GOMOD")
	if err != nil || gomod == "" || gomod == "/dev/null" {
		fmt.Println("This file is NOT inside a Go module.")
		fmt.Println("Move it under a directory containing go.mod, or run 'go mod init'.")
		return
	}

	moduleRoot := filepath.Dir(gomod)
	modulePath, err := readModulePath(gomod)
	if err != nil {
		die("read go.mod:", err)
	}

	// ─── Compute the package's import path ─────────────────────────────
	//
	// The toolchain rule: package import path = module path + the relative
	// directory from the module root. Slashes are forward slashes on every
	// platform; filepath.ToSlash normalises Windows backslashes.
	rel, err := filepath.Rel(moduleRoot, dir)
	if err != nil {
		die("relpath:", err)
	}
	importPath := modulePath
	if rel != "." {
		importPath = modulePath + "/" + filepath.ToSlash(rel)
	}

	// ─── Parse the file and inspect declarations ───────────────────────
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, target, nil, parser.ParseComments)
	if err != nil {
		die("parse:", err)
	}

	exported, unexported := countNames(f)

	// ─── Print the breakdown ───────────────────────────────────────────
	fmt.Printf("File:               %s\n", target)
	fmt.Printf("Module root:        %s\n", moduleRoot)
	fmt.Printf("Module path:        %s\n", modulePath)
	fmt.Printf("Package directory:  %s\n", rel)
	fmt.Printf("Package name:       %s\n", f.Name.Name)
	fmt.Printf("Import path:        %s\n", importPath)
	fmt.Printf("Top-level decls:    %d (%d exported, %d unexported)\n",
		exported+unexported, exported, unexported)

	if strings.Contains(filepath.ToSlash(rel), "/internal/") || rel == "internal" || strings.HasPrefix(filepath.ToSlash(rel), "internal/") {
		fmt.Println("Privacy:            INTERNAL — only importable within this module")
	} else {
		fmt.Println("Privacy:            public — importable by other modules")
	}
}

// goEnvFrom runs `go env <name>` with cwd set to dir. Useful so module
// resolution works as if the user had cd'd into dir before invoking go.
func goEnvFrom(dir, name string) (string, error) {
	cmd := exec.Command("go", "env", name)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// readModulePath extracts the module path declared in a go.mod file. We
// don't use the official golang.org/x/mod/modfile parser because we want
// the example to run with stdlib only — string-based parsing is fine for
// this case.
func readModulePath(gomod string) (string, error) {
	data, err := os.ReadFile(gomod)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}
	return "", fmt.Errorf("no module directive in %s", gomod)
}

// countNames walks the top-level declarations and counts how many have
// exported (uppercase first rune) names vs. unexported. Useful as a sanity
// check on package surface area.
func countNames(f *ast.File) (exported, unexported int) {
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if isExported(d.Name.Name) {
				exported++
			} else {
				unexported++
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if isExported(s.Name.Name) {
						exported++
					} else {
						unexported++
					}
				case *ast.ValueSpec:
					for _, n := range s.Names {
						if isExported(n.Name) {
							exported++
						} else {
							unexported++
						}
					}
				}
			}
		}
	}
	return
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	r := []rune(name)[0]
	return unicode.IsUpper(r)
}

func die(args ...any) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}
