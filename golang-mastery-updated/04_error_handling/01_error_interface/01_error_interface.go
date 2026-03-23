// 01_error_interface.go
//
// THE ERROR INTERFACE IN GO
// =========================
// Go's error handling is fundamentally different from most mainstream languages.
// There are no exceptions, no try/catch blocks, no stack-unwinding.
//
// Instead, errors are VALUES — ordinary values that functions return alongside
// their normal results. This is not a limitation; it is a deliberate design
// choice with profound implications for code clarity and reliability.
//
// WHY RETURN VALUES INSTEAD OF EXCEPTIONS?
// -----------------------------------------
// 1. EXPLICIT CONTROL FLOW: With exceptions, an error can bubble up invisibly
//    through many stack frames without any caller realising it. The reader of
//    the code cannot tell which calls might throw. In Go every call site that
//    can fail is visually obvious — you see the ", err" right there.
//
// 2. ERRORS ARE FIRST-CLASS: Because errors are just values, you can store
//    them in data structures, pass them to functions, inspect them, wrap them,
//    and reason about them like any other value. You cannot do this with
//    exceptions without catching them first.
//
// 3. FORCES ACKNOWLEDGEMENT: The compiler will not let you silently ignore
//    a returned error unless you explicitly write "_" (which is a code smell
//    that reviewers will flag immediately). With exceptions, forgetting a
//    try/catch is silent until production.
//
// 4. PERFORMANCE: Exception-based error handling requires setting up stack
//    frames that can be unwound. Go's approach has zero overhead on the happy
//    path — a returned nil costs nothing.
//
// THE ERROR INTERFACE
// -------------------
// The entire error system is built on one tiny interface defined in the
// builtin package:
//
//   type error interface {
//       Error() string
//   }
//
// That's it. One method. Any type that has an Error() string method satisfies
// the error interface and can be used as an error value. This simplicity is
// intentional — it makes error types easy to define and the system easy to
// understand.

package main

import (
	"errors"
	"fmt"
)

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 1: The nil error — success is represented by nil
// ─────────────────────────────────────────────────────────────────────────────

// divide returns a result and an error. The error is nil on success.
// CONVENTION: error is ALWAYS the last return value. The Go community treats
// this as a hard rule. Breaking it makes your code feel foreign and makes
// tooling (like errcheck) less effective.
func divide(a, b float64) (float64, error) {
	if b == 0 {
		// errors.New creates the simplest possible error: a static string
		// wrapped in a private errorString struct that implements error.
		// Use it when the message completely describes the problem and you
		// do not need to carry extra structured data.
		return 0, errors.New("division by zero")
	}
	return a / b, nil // nil means "no error occurred"
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 2: errors.New vs fmt.Errorf
// ─────────────────────────────────────────────────────────────────────────────

// errors.New
// ----------
// Creates a simple error with a static message.
// The resulting value is a *errors.errorString (unexported type).
// Two calls to errors.New with the same string produce DIFFERENT values:
//
//   errors.New("x") != errors.New("x")   // different pointers
//
// This matters for sentinel errors (covered in 04_sentinel_errors.go).

// fmt.Errorf
// ----------
// Creates an error whose message is built with Printf-style formatting.
// Use it when you need to include runtime values in the message.
//
// IMPORTANT: fmt.Errorf with %v embeds a string — the original error is lost.
//            fmt.Errorf with %w WRAPS the error — errors.Is/As can unwrap it.
//            (Wrapping is covered in depth in 03_error_wrapping.go.)

func openFile(path string) (string, error) {
	if path == "" {
		// fmt.Errorf lets us inject the problematic value into the message.
		// This makes the error message immediately actionable for callers.
		return "", fmt.Errorf("openFile: path must not be empty, got %q", path)
	}
	if path == "/etc/shadow" {
		return "", fmt.Errorf("openFile: permission denied for path %q", path)
	}
	return "file content of " + path, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 3: The idiomatic error check pattern
// ─────────────────────────────────────────────────────────────────────────────
//
// The single most common Go pattern is:
//
//   result, err := someCall()
//   if err != nil {
//       // handle or return
//   }
//   // use result safely
//
// WHY "if err != nil" and not something nicer?
// The Go authors considered alternatives (Result types, monadic chaining)
// and chose explicit checks deliberately. The verbosity is a feature: it
// forces you to think at each call site about what to do when things go wrong.
// It also makes control flow completely obvious to anyone reading the code.
//
// COMMON MISTAKE — ignoring errors with _:
//
//   result, _ := someCall()   // DO NOT DO THIS in production code
//
// There are extremely rare legitimate uses (closing a file you already read
// successfully, or ignoring errors in defer), but they should be accompanied
// by a comment explaining why the error is intentionally ignored.

func processFile(path string) error {
	// The short variable declaration ":=" in the if initialiser is idiomatic.
	// It scopes err to the if block, avoiding variable shadowing across checks.
	content, err := openFile(path)
	if err != nil {
		// When you cannot handle the error here, WRAP it with context and
		// return it upward. More on wrapping in 03_error_wrapping.go.
		// For now we just return it as-is to keep this example simple.
		return err
	}

	fmt.Printf("processFile: successfully read %d bytes\n", len(content))
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 4: Multiple return values — the full pattern in action
// ─────────────────────────────────────────────────────────────────────────────

// parseAge converts a string age to an integer with full validation.
// Notice: the zero value is returned for the integer when an error occurs.
// Convention: return the zero value of non-error return values on failure.
// This prevents callers from accidentally using a partial result.
func parseAge(s string) (int, error) {
	if s == "" {
		return 0, errors.New("parseAge: age string must not be empty")
	}

	var age int
	// Manually "parse" for demonstration — real code would use strconv.Atoi
	switch s {
	case "25":
		age = 25
	case "30":
		age = 30
	case "150":
		// Valid parse but invalid domain value
		return 0, fmt.Errorf("parseAge: age %q is out of valid range [0, 120]", s)
	default:
		return 0, fmt.Errorf("parseAge: %q is not a valid integer", s)
	}

	return age, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 5: Demonstrating what errors.New returns
// ─────────────────────────────────────────────────────────────────────────────

// To understand the interface mechanically, here is a manual re-implementation
// of errors.New. The real one is almost exactly this.
type errorString struct {
	s string
}

func (e *errorString) Error() string {
	return e.s
}

// myNew mirrors errors.New. Returns a *errorString which satisfies error.
func myNew(text string) error {
	return &errorString{s: text}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN — run all demonstrations
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== 01: The error Interface ===")
	fmt.Println()

	// ── 1. divide: success and failure ──────────────────────────────────────
	fmt.Println("── divide ──")

	result, err := divide(10, 2)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("10 / 2 = %.2f\n", result)
	}

	result, err = divide(10, 0)
	if err != nil {
		// err.Error() returns the string. fmt verbs like %v and %s call it
		// automatically, so you rarely need to call .Error() directly.
		fmt.Println("Error:", err)
		fmt.Println("Error string explicitly:", err.Error())
	} else {
		fmt.Printf("10 / 0 = %.2f\n", result)
	}
	fmt.Println()

	// ── 2. errors.New vs fmt.Errorf ─────────────────────────────────────────
	fmt.Println("── errors.New vs fmt.Errorf ──")

	e1 := errors.New("something went wrong")
	e2 := fmt.Errorf("something went wrong for user %d", 42)
	fmt.Printf("errors.New  → %v  (type: %T)\n", e1, e1)
	fmt.Printf("fmt.Errorf  → %v  (type: %T)\n", e2, e2)

	// Two errors.New with the same text are NOT equal (different pointers)
	ea := errors.New("same text")
	eb := errors.New("same text")
	fmt.Printf("\nea == eb: %v  (pointer identity, not string equality)\n", ea == eb)
	fmt.Println()

	// ── 3. openFile ──────────────────────────────────────────────────────────
	fmt.Println("── openFile ──")

	cases := []string{"", "/etc/shadow", "/home/user/data.txt"}
	for _, path := range cases {
		content, err := openFile(path)
		if err != nil {
			fmt.Printf("  path=%q  → ERROR: %v\n", path, err)
		} else {
			fmt.Printf("  path=%q  → OK: %q\n", path, content)
		}
	}
	fmt.Println()

	// ── 4. processFile ───────────────────────────────────────────────────────
	fmt.Println("── processFile ──")

	if err := processFile(""); err != nil {
		fmt.Println("processFile(\"\") error:", err)
	}
	if err := processFile("/home/user/notes.txt"); err != nil {
		fmt.Println("processFile error:", err)
	}
	fmt.Println()

	// ── 5. parseAge ──────────────────────────────────────────────────────────
	fmt.Println("── parseAge ──")

	ages := []string{"25", "30", "150", "abc", ""}
	for _, s := range ages {
		age, err := parseAge(s)
		if err != nil {
			fmt.Printf("  parseAge(%q)  → ERROR: %v\n", s, err)
		} else {
			fmt.Printf("  parseAge(%q)  → age=%d\n", s, age)
		}
	}
	fmt.Println()

	// ── 6. Manual errorString (our re-implementation of errors.New) ──────────
	fmt.Println("── manual errorString ──")

	myErr := myNew("this is a custom error")
	fmt.Printf("myErr type: %T\n", myErr)
	fmt.Printf("myErr message: %v\n", myErr)

	// It satisfies the error interface — we can assign it to error
	var e error = myErr
	fmt.Printf("as error interface: %v\n", e)
	fmt.Println()

	// ── 7. The if err != nil check with initialiser ──────────────────────────
	fmt.Println("── idiomatic if-err check ──")

	// Single-line initialiser + check — preferred style when you do not need
	// the result outside the if block. Keeps scope tight.
	if _, err := divide(5, 0); err != nil {
		fmt.Println("Caught in initialiser form:", err)
	}

	fmt.Println()
	fmt.Println("Key takeaways:")
	fmt.Println("  1. error is just an interface: Error() string")
	fmt.Println("  2. nil error means success; non-nil means failure")
	fmt.Println("  3. error is always the last return value by convention")
	fmt.Println("  4. errors.New for static messages; fmt.Errorf for dynamic ones")
	fmt.Println("  5. Never ignore errors with _ without a justifying comment")
	fmt.Println("  6. Check errors immediately after the call that produced them")
}
