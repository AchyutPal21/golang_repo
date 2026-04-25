// FILE: book/part1_foundations/chapter01_why_go_exists/examples/01_hello/main.go
// CHAPTER: 01 — Why Go Exists
// TOPIC: The smallest meaningful Go program.
//
// Run (from the chapter folder):
//   go run ./examples/01_hello
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   To prove, in six lines of code, that Go has very little ceremony. Every
//   token in this file is doing real work. There is no boilerplate to skip,
//   no framework to configure, no class to declare. This is the entire shape
//   of "a Go program that does something."
// ─────────────────────────────────────────────────────────────────────────────

// ─── 1. The package declaration ─────────────────────────────────────────────
//
// Every Go source file begins with `package <name>`. Files in the same
// directory must declare the same package name. The special package name
// `main` marks this file as belonging to an *executable* program — i.e.,
// when you run `go build` in this directory, you'll get a binary you can
// run, not just a library other code can import.
package main

// ─── 2. The import block ────────────────────────────────────────────────────
//
// Imports name standard-library or third-party packages used by this file.
// Unused imports are a *compile error* in Go — not a warning. This is
// deliberate. The authors believed that "unused imports" was a code-rot
// signal worth fighting in the compiler, not the linter. You'll find this
// strict in the first month and grateful for it after that.
import "fmt"

// ─── 3. main: the entry point ───────────────────────────────────────────────
//
// `main` takes no arguments and returns no value. The runtime calls it
// after package initialization (covered in Chapters 4 and 8). When `main`
// returns, the program exits with status 0; if it panics and is not
// recovered, the program exits with status 2 and prints a stack trace.
//
// To exit with a non-zero status code without panicking: call os.Exit(n).
// To get command-line arguments: read os.Args. We'll use both in Chapter 7.
func main() {
	// fmt.Println writes to standard output, followed by a newline.
	// It is not a macro, not a keyword — just a function in the `fmt`
	// package of the standard library. Type `go doc fmt.Println` in your
	// terminal to read the contract directly. Reading stdlib docs from
	// the command line is one of the niceties of Go's tooling.
	fmt.Println("hello, world")
}
