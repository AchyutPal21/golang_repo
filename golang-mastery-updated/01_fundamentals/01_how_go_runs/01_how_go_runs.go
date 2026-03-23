// FILE: 01_fundamentals/01_how_go_runs/01_how_go_runs.go
// TOPIC: How Go Programs Run — Entry Point, Compilation, Init Order
//
// Run: go run 01_fundamentals/01_how_go_runs/01_how_go_runs.go
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS MATTERS:
//   Before writing a single line of logic, you must understand what happens
//   when you hit "go run". Many bugs and design mistakes come from not knowing
//   the execution model of the language you're using.
//   Go's model is deliberately simple compared to Java/C++ — but there are
//   specific rules about initialization order that every senior Go dev knows.
// ─────────────────────────────────────────────────────────────────────────────

package main

import "fmt"

// ─────────────────────────────────────────────────────────────────────────────
// STEP 1: COMPILATION MODEL
// ─────────────────────────────────────────────────────────────────────────────
//
// Go is a COMPILED language. When you run `go run file.go`, Go:
//   1. Compiles the source to a temporary binary
//   2. Runs that binary
//   3. Deletes the binary after the process exits
//
// To keep the binary: use `go build -o myprogram .`
//
// Unlike Python/JS (interpreted), your Go code becomes MACHINE CODE.
// There is NO virtual machine, NO bytecode, NO garbage-collected runtime
// in the traditional sense — just a native binary with a tiny Go runtime
// embedded inside it.
//
// The Go runtime handles:
//   - Goroutine scheduling (M:N threading)
//   - Garbage collection
//   - Memory allocation (stack growth)
//   - Channel operations
//   - Panic/recover unwinding
//
// Binary size: even a "hello world" is ~2MB because the runtime is statically
// linked. The upside: you ship ONE file with zero external dependencies.

// ─────────────────────────────────────────────────────────────────────────────
// STEP 2: PACKAGE SYSTEM BASICS
// ─────────────────────────────────────────────────────────────────────────────
//
// Every Go file starts with:  package <name>
//
// The ONLY package that can have a main() function is "package main".
// The ONLY package the Go runtime knows how to execute is "package main".
//
// A package is a directory of .go files. All files in the same directory
// must have the same package name. The package name and directory name
// can differ (but by convention they match, except for "main").
//
// `package main` is special: it marks this as an EXECUTABLE program.
// Any other package name means it's a LIBRARY (importable, not runnable).

// ─────────────────────────────────────────────────────────────────────────────
// STEP 3: PACKAGE-LEVEL VARIABLES — initialized BEFORE main()
// ─────────────────────────────────────────────────────────────────────────────
//
// Variables declared at the package level (outside any function) are
// initialized in ORDER OF DECLARATION, BUT with one exception: if variable B
// depends on variable A, Go guarantees A is initialized first — even if B
// appears first in the source. The compiler resolves the dependency graph.
//
// Package-level vars are initialized BEFORE any init() function runs,
// and BEFORE main() is called.

var (
	// These are package-level variables.
	// They are initialized once, when the program starts.
	// They live for the entire lifetime of the program (global scope).
	programName = "Go Mastery"        // string, inferred
	version     = "1.0.0"             // string, inferred
	maxWorkers  = computeMaxWorkers() // initialized by calling a function!
)

// computeMaxWorkers is called during package initialization.
// This is a valid and common pattern for complex initialization.
func computeMaxWorkers() int {
	// In real code you might read from env vars, config files, etc.
	return 8
}

// ─────────────────────────────────────────────────────────────────────────────
// STEP 4: init() FUNCTIONS
// ─────────────────────────────────────────────────────────────────────────────
//
// init() is a special function in Go:
//   - Called AUTOMATICALLY by the runtime, you never call it manually
//   - Runs AFTER all package-level variables are initialized
//   - Runs BEFORE main()
//   - You can have MULTIPLE init() functions in the same file (unlike most languages!)
//   - You can have init() functions across multiple files in the same package
//   - They run in the order: dependency packages first, then current package,
//     within a file top-to-bottom, across files in the order the compiler
//     processes them (alphabetical by filename is the typical order)
//
// USE init() FOR:
//   - One-time setup that can't be done with a simple var initializer
//   - Registering things (database drivers, codec formats, etc.)
//   - Validating configuration
//
// AVOID init() FOR:
//   - Complex logic that could fail silently
//   - Things that should be testable (init() is hard to test in isolation)
//   - Side effects that surprise other devs

var initOrder []string // we'll record what ran when

func init() {
	// This runs FIRST (it's declared first in the file)
	initOrder = append(initOrder, "init() #1 ran")
	// Note: programName, version, maxWorkers are already set here
}

func init() {
	// This runs SECOND — yes, two init() in the same file is valid!
	initOrder = append(initOrder, "init() #2 ran")
}

// ─────────────────────────────────────────────────────────────────────────────
// STEP 5: main() — The Entry Point
// ─────────────────────────────────────────────────────────────────────────────
//
// main() is where YOUR code begins executing.
// It takes no arguments and returns nothing.
//
// To access command-line arguments: use os.Args (covered in Module 08)
// To return an exit code: use os.Exit(code) — but this skips deferred calls!
//
// The program exits when main() returns, OR when os.Exit() is called.
// If any goroutine panics and the panic is not recovered, the entire program
// crashes — it does NOT continue running other goroutines.

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: How Go Programs Run")
	fmt.Println("════════════════════════════════════════")

	// ── Package-level vars are ready ──────────────────────────────────────
	fmt.Printf("\nProgram: %s  v%s\n", programName, version)
	fmt.Printf("Max workers computed at startup: %d\n", maxWorkers)

	// ── init() functions already ran ──────────────────────────────────────
	fmt.Println("\nInitialization order recorded:")
	for i, entry := range initOrder {
		fmt.Printf("  [%d] %s\n", i+1, entry)
	}

	// ── Execution order summary ───────────────────────────────────────────
	fmt.Println(`
FULL EXECUTION ORDER:
  1. Import all dependent packages (recursively)
  2. Initialize package-level variables (respecting dependency graph)
  3. Run all init() functions (in declaration order, then file order)
  4. Call main()
  5. Program exits when main() returns
`)

	// ── Key insight: Go's minimal surface area ────────────────────────────
	fmt.Println("KEY INSIGHT:")
	fmt.Println("  Go deliberately has NO constructors, NO class initializers,")
	fmt.Println("  NO module-level __init__.py style code.")
	fmt.Println("  Just: package vars → init() → main()")
	fmt.Println("  Simple, predictable, debuggable.")
}
