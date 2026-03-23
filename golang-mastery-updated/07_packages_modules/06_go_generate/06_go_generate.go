// FILE: 07_packages_modules/06_go_generate.go
// TOPIC: go:generate, embed, compiler directives
//
// Run: go run 07_packages_modules/06_go_generate.go

package main

import (
	_ "embed"
	"fmt"
	"strings"
)

// ── EMBED — include files in the binary at compile time ──────────────────────
// //go:embed embeds a file or directory into the binary.
// The file content becomes a compile-time constant (string or []byte).
// No runtime file I/O needed — the data is baked into the binary.
//
// We embed this source file itself as a demo (always exists):

//go:embed 06_go_generate.go
var sourceCode string

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: go generate, embed, directives")
	fmt.Println("════════════════════════════════════════")

	// ── //go:embed demo ─────────────────────────────────────────────────
	fmt.Println("\n── //go:embed ──")
	lines := strings.Split(sourceCode, "\n")
	fmt.Printf("  Embedded this file: %d lines, %d bytes\n", len(lines), len(sourceCode))
	fmt.Printf("  First line: %q\n", lines[0])

	// ── go:generate ─────────────────────────────────────────────────────
	fmt.Println(`
── //go:generate directive ──

  A comment that tells 'go generate' to run a command.
  Run all generators in a package with: go generate ./...

  Examples:

  //go:generate go run golang.org/x/tools/cmd/stringer -type=Color
  // → generates string methods for an enum type

  //go:generate protoc --go_out=. proto/service.proto
  // → generates Go code from .proto files

  //go:generate mockery --name=UserRepository
  // → generates mock implementations for testing

  //go:generate go-bindata -o bindata.go assets/
  // → embeds static files (replaced by //go:embed in modern Go)

  WHY go:generate?
    - Code generation keeps repetitive code out of your codebase
    - Generated files are committed to git so build doesn't require tools
    - Run generator on code change, commit the result
    - Stringer is the most common: avoids writing String() for every enum

── Compiler directives ──

  //go:noinline
  func expensiveFunc() { ... }
  // Prevents inlining — useful for benchmarks (prevents optimization away)

  //go:nosplit
  func lowLevelFunc() { ... }
  // Prevents stack growth — only for very low-level runtime code

  //go:noescape
  // Used in assembly stubs — pointer doesn't escape to heap

  //go:linkname localName importPath.remoteName
  // Access unexported functions in other packages (dangerous, internal use)
  // Example (runtime internal use): //go:linkname nanotime runtime.nanotime

  //go:build (covered in 05_build_tags.go)

── Escape analysis ──

  go build -gcflags="-m" .
  // Shows which variables escape to the heap vs stay on stack.
  // Heap allocation is slower. Use this to optimize hot paths.
`)

	// ── embed types ─────────────────────────────────────────────────────
	fmt.Println("── embed types ──")
	fmt.Println(`
  // Embed as string (UTF-8 text):
  //go:embed config.yaml
  var config string

  // Embed as []byte (binary files, images):
  //go:embed logo.png
  var logo []byte

  // Embed a whole directory as embed.FS:
  //go:embed static/*
  var staticFiles embed.FS
  // Use: staticFiles.Open("static/index.html")
  // Use with http.FileServer for serving static assets from binary.

  // Multiple files:
  //go:embed templates/*.html
  var templates embed.FS
`)
}
