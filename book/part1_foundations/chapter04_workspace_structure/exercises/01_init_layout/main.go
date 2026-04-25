// FILE: book/part1_foundations/chapter04_workspace_structure/exercises/01_init_layout/main.go
// EXERCISE 4.3 — Initialize a layout.
//
// Run (from the chapter folder):
//   go run ./exercises/01_init_layout
//
// Prompts for a project type and module path, then prints the directory
// tree you should `mkdir -p` to set it up. We deliberately don't actually
// create the directories — making layout decisions in your head, then
// committing them, is the skill the exercise is training.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type plan struct {
	name      string
	directory []string
	files     []string
	notes     string
}

var plans = map[string]plan{
	"library": {
		name: "Flat library (Pattern A)",
		directory: []string{
			"",
		},
		files: []string{
			"go.mod",
			"README.md",
			"doc.go",
			"<package>.go",
			"<package>_test.go",
			"internal/<helper>/",
		},
		notes: "Single import path == module path. internal/ for private helpers.",
	},
	"service": {
		name: "Service / multi-binary (Pattern B)",
		directory: []string{
			"cmd/<binary>/",
			"internal/<domain>/",
			"internal/<domain>/<storage>/",
		},
		files: []string{
			"go.mod",
			"README.md",
			"cmd/<binary>/main.go",
			"internal/<domain>/<domain>.go",
			"internal/<domain>/<storage>/<storage>.go",
		},
		notes: "cmd/<name>/main.go is tiny — flag parsing + call internal/<name>.Run(ctx).",
	},
	"monorepo": {
		name: "Multi-module monorepo (Pattern C)",
		directory: []string{
			"<lib1>/",
			"<lib2>/",
			"<svc1>/",
			"<svc2>/",
		},
		files: []string{
			"go.work          # GITIGNORED",
			"<lib1>/go.mod",
			"<lib1>/<lib1>.go",
			"<svc1>/go.mod",
			"<svc1>/main.go",
			"<svc2>/go.mod",
			"<svc2>/main.go",
		},
		notes: "Each subdir is its own module with its own version. go.work coordinates local dev.",
	},
}

func main() {
	in := bufio.NewReader(os.Stdin)

	fmt.Println("Layout planner")
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println("1) library")
	fmt.Println("2) service (cmd/ + internal/)")
	fmt.Println("3) monorepo (multi-module)")
	fmt.Print("\nChoose 1/2/3: ")
	choice, _ := in.ReadString('\n')
	choice = strings.TrimSpace(choice)

	var p plan
	switch choice {
	case "1":
		p = plans["library"]
	case "2":
		p = plans["service"]
	case "3":
		p = plans["monorepo"]
	default:
		fmt.Println("invalid choice")
		os.Exit(2)
	}

	fmt.Print("Module path (e.g. github.com/yourname/yourtool): ")
	module, _ := in.ReadString('\n')
	module = strings.TrimSpace(module)
	if module == "" {
		fmt.Println("module path required")
		os.Exit(2)
	}

	fmt.Println()
	fmt.Println(p.name)
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("Module: %s\n\n", module)
	fmt.Println("Directories to mkdir -p:")
	for _, d := range p.directory {
		full := strings.TrimSuffix(module, "/") + "/" + d
		fmt.Printf("  %s\n", full)
	}
	fmt.Println("\nFiles to seed:")
	for _, f := range p.files {
		fmt.Printf("  %s\n", f)
	}
	fmt.Println("\nNotes:")
	fmt.Println("  " + p.notes)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. mkdir the directories.")
	fmt.Printf("  2. cd into the root and: go mod init %s\n", module)
	fmt.Println("  3. Add a top-level README.md.")
	fmt.Println("  4. Write a doc.go with the package overview.")
	fmt.Println("  5. Run `go build ./...` to confirm it compiles.")
}
