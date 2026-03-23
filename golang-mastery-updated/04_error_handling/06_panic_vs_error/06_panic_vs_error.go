// 06_panic_vs_error.go
//
// PANIC vs ERROR IN GO
// ====================
// Go has two distinct mechanisms for dealing with failure:
//
//   1. ERRORS  — returned values representing expected failure conditions.
//   2. PANICS  — an abrupt stop of the current goroutine for unexpected,
//                unrecoverable situations (programmer bugs).
//
// Understanding the difference — and knowing which to use — is one of the
// most important skills for writing idiomatic, robust Go.
//
// THE GO PHILOSOPHY
// -----------------
// "Errors are for expected failure conditions that callers can and should
//  handle. Panics are for programmer errors — things that should never
//  happen if the code is correct."
//
// WHEN TO USE ERRORS (almost always)
// -----------------------------------
//   - File not found
//   - Network timeout
//   - Invalid user input
//   - Database constraint violation
//   - JSON unmarshal failure
//   - Any condition the CALLER should know about and handle
//
// WHEN TO USE PANIC (rarely — programmer bugs only)
// --------------------------------------------------
//   - Index out of bounds (Go runtime does this automatically)
//   - Nil pointer dereference (Go runtime does this automatically)
//   - Invariant violation that means the program state is fundamentally broken
//     e.g., a required configuration is nil after initialisation
//   - Programming contract violations: "you MUST call Init() before Use()"
//   - Unreachable code paths in a switch that exhausted all known types
//
// LIBRARY CODE RULE: Never let a panic escape a library.
// -------------------------------------------------------
// If you are writing a library (any package imported by others), panics
// are unacceptable. Your callers have no way to handle a panic short of
// a recover() they must set up defensively. Library code MUST convert
// all panics to errors at its API boundary.
//
// THE "MUST" PATTERN
// ------------------
// Some operations WILL panic on invalid input by design — because they are
// only called with static, compile-time-known values that must be correct.
// The convention is to name these functions "Must" + something:
//
//   regexp.MustCompile("(invalid") // panics — use only with static patterns
//   template.Must(...)
//   http.ListenAndServe(...)       // different pattern — but same spirit
//
// Must functions are used at program initialisation (package-level var or
// init()). If a regex pattern is wrong, you want to find out at startup,
// not at the first request.

package main

import (
	"errors"
	"fmt"
	"regexp"
)

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 1: How panic works
// ─────────────────────────────────────────────────────────────────────────────
// panic(v) stops the current function immediately, runs all deferred
// functions in the current goroutine (in LIFO order), and then crashes
// the program (printing the panic value and a stack trace) — UNLESS
// a deferred recover() intercepts it.
//
// panic accepts any value (interface{}), but convention is to use:
//   - a string: panic("invariant violated: ...")
//   - an error:  panic(err)
//   - a custom struct for richer info

// safeDiv panics on divide-by-zero (wrong approach — illustrative only).
// Compare with the error-returning version above.
func safeDiv(a, b int) int {
	if b == 0 {
		// A panic here is WRONG for a general-purpose function.
		// Callers may legitimately pass b=0 (e.g., user input).
		// This demonstrates the anti-pattern; real code should return error.
		panic("safeDiv: division by zero — b must not be zero")
	}
	return a / b
}

// errorDiv is the CORRECT version — returns an error instead of panicking.
func errorDiv(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("errorDiv: division by zero")
	}
	return a / b, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 2: recover() — intercepting a panic
// ─────────────────────────────────────────────────────────────────────────────
// recover() can only be called inside a deferred function. It stops the
// panic propagation and returns the value passed to panic(). Outside a
// deferred function, recover() always returns nil and does nothing.
//
// Use cases for recover():
//   1. Library API boundaries — convert panics to errors for callers.
//   2. Server goroutines — catch unexpected panics to keep the server alive.
//   3. Test helpers — assert that code panics as expected.

// safeCall runs f and converts any panic into a returned error.
// This is the canonical pattern for libraries that must not let panics escape.
func safeCall(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// r can be anything panic was called with.
			switch v := r.(type) {
			case error:
				err = fmt.Errorf("recovered panic: %w", v)
			case string:
				err = fmt.Errorf("recovered panic: %s", v)
			default:
				err = fmt.Errorf("recovered panic: %v", v)
			}
		}
	}()
	f()
	return nil // only reached if f() does not panic
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 3: A correct use of panic — invariant violation
// ─────────────────────────────────────────────────────────────────────────────

// Config holds application configuration.
type Config struct {
	DBHost string
	Port   int
}

// NewApp is called once at startup. If Config is nil, the program is
// misconfigured. There is no sensible way to proceed; panic is correct here.
func NewApp(cfg *Config) string {
	if cfg == nil {
		// This is a programmer error: the caller must not pass nil.
		// A nil Config is not a runtime condition; it is a coding mistake.
		panic("NewApp: cfg must not be nil — check your initialisation code")
	}
	if cfg.DBHost == "" {
		panic("NewApp: cfg.DBHost must not be empty")
	}
	return fmt.Sprintf("App running on %s:%d", cfg.DBHost, cfg.Port)
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 4: The "Must" pattern
// ─────────────────────────────────────────────────────────────────────────────
// Functions named Must* panic when they receive an error.
// They are designed for use ONLY with values known at compile time
// (static regexes, static templates, static configurations loaded at startup).

// mustCompile wraps regexp.Compile: panics if the pattern is invalid.
// This mirrors what regexp.MustCompile does.
func mustCompile(pattern string) *regexp.Regexp {
	re, err := regexp.Compile(pattern)
	if err != nil {
		// The panic message should clearly indicate this is a programming error.
		panic(fmt.Sprintf("mustCompile: invalid pattern %q: %v", pattern, err))
	}
	return re
}

// Package-level regex — evaluated once at program start.
// If the pattern is wrong, you discover it IMMEDIATELY at startup, not at
// the first request (which might happen hours later in production).
var emailRegex = mustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// mustParseInt is another Must example — for configuration parsing.
func mustParseInt(s string) int {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		panic(fmt.Sprintf("mustParseInt: %q is not an integer: %v", s, err))
	}
	return n
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 5: Library boundary pattern
// ─────────────────────────────────────────────────────────────────────────────
// A simulated library that internally might panic (e.g., due to a bug or
// a third-party dependency). The exported API must never let panics escape.

// ProcessData is an exported library function. It uses recover to ensure
// panics from internal code are converted to errors.
func ProcessData(data []byte) (result string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("ProcessData: internal error: %v", r)
			result = ""
		}
	}()

	if data == nil {
		panic("internal bug: processData called with nil data") // internal panic
	}
	if len(data) == 0 {
		return "", errors.New("ProcessData: data must not be empty")
	}
	return fmt.Sprintf("processed %d bytes", len(data)), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 6: When panic propagation crashes the program
// ─────────────────────────────────────────────────────────────────────────────
// If a panic is not recovered, it propagates up through the goroutine's
// stack, runs all deferred functions, and terminates the program with a
// message and stack trace. We demonstrate this contained in safeCall.

func demonstratePropagation() {
	fmt.Println("  About to call safeDiv(10, 0) via safeCall...")
	err := safeCall(func() {
		result := safeDiv(10, 0) // will panic
		fmt.Println("  (this line never runs):", result)
	})
	if err != nil {
		fmt.Println("  Caught panic as error:", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 7: Guidelines decision table
// ─────────────────────────────────────────────────────────────────────────────

func printGuidelines() {
	fmt.Println("── Panic vs Error decision table ──")
	rows := []struct{ condition, approach string }{
		{"User provided invalid input", "Return error — expected condition"},
		{"File does not exist", "Return error — expected condition"},
		{"Network request timed out", "Return error — expected condition"},
		{"Nil pointer passed to a required parameter", "Panic — programmer bug"},
		{"Required config missing at startup", "Panic — programmer bug"},
		{"Index out of bounds in algorithm", "Panic (runtime does it anyway)"},
		{"Unreachable default in exhaustive switch", "Panic — invariant violated"},
		{"Static regex pattern is wrong (package init)", "Must* + panic"},
		{"Panic inside a library function", "recover() → return error"},
		{"Panic inside a server handler goroutine", "recover() to keep server alive"},
	}
	for _, r := range rows {
		fmt.Printf("  %-50s → %s\n", r.condition, r.approach)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== 06: Panic vs Error ===")
	fmt.Println()

	// ── 1. errorDiv vs panic ──────────────────────────────────────────────────
	fmt.Println("── errorDiv (correct) vs safeDiv (wrong) ──")
	if result, err := errorDiv(10, 2); err == nil {
		fmt.Printf("  errorDiv(10,2) = %d\n", result)
	}
	if _, err := errorDiv(10, 0); err != nil {
		fmt.Printf("  errorDiv(10,0) = error: %v\n", err)
	}
	fmt.Println()

	// ── 2. safeDiv panic intercepted by safeCall ──────────────────────────────
	fmt.Println("── panic in safeDiv (intercepted by safeCall) ──")
	demonstratePropagation()
	fmt.Println()

	// ── 3. NewApp — invariant panic ───────────────────────────────────────────
	fmt.Println("── NewApp invariant panic ──")
	err := safeCall(func() {
		_ = NewApp(nil) // nil config — programmer error
	})
	fmt.Println("  nil config:", err)

	err = safeCall(func() {
		_ = NewApp(&Config{DBHost: ""}) // missing required field
	})
	fmt.Println("  empty DBHost:", err)

	app := NewApp(&Config{DBHost: "db.internal", Port: 5432})
	fmt.Println("  valid config:", app)
	fmt.Println()

	// ── 4. Must pattern ────────────────────────────────────────────────────────
	fmt.Println("── Must pattern ──")

	// emailRegex is already compiled at package init — no error possible here.
	emails := []string{"user@example.com", "invalid-email", "a@b.io"}
	for _, e := range emails {
		fmt.Printf("  %q matches email regex: %v\n", e, emailRegex.MatchString(e))
	}

	// mustCompile with a bad pattern — intercept the panic to show it.
	err = safeCall(func() {
		_ = mustCompile("((broken pattern")
	})
	fmt.Println("  bad regex panic:", err)

	// mustParseInt
	n := mustParseInt("42")
	fmt.Printf("  mustParseInt(\"42\") = %d\n", n)
	err = safeCall(func() { _ = mustParseInt("abc") })
	fmt.Println("  mustParseInt(\"abc\") panic:", err)
	fmt.Println()

	// ── 5. Library boundary ────────────────────────────────────────────────────
	fmt.Println("── ProcessData (library boundary) ──")

	r1, err1 := ProcessData([]byte("hello world"))
	fmt.Printf("  valid data: result=%q err=%v\n", r1, err1)

	r2, err2 := ProcessData(nil) // triggers internal panic
	fmt.Printf("  nil data:   result=%q err=%v\n", r2, err2)

	r3, err3 := ProcessData([]byte{}) // returns error (not panic)
	fmt.Printf("  empty data: result=%q err=%v\n", r3, err3)
	fmt.Println()

	// ── 6. Guidelines ──────────────────────────────────────────────────────────
	printGuidelines()
	fmt.Println()

	fmt.Println("Key takeaways:")
	fmt.Println("  1. Errors for expected conditions; panic for programmer bugs only")
	fmt.Println("  2. Library code: recover() at API boundary → convert panic to error")
	fmt.Println("  3. Must* pattern: panic on static invalid input (discovered at startup)")
	fmt.Println("  4. recover() only works inside a deferred function")
	fmt.Println("  5. Goroutine panics only affect that goroutine; recover per goroutine")
	fmt.Println("  6. Ask: 'is this a caller mistake or a programmer mistake?' → error vs panic")
}
