// FILE: book/part3_designing_software/chapter36_error_handling_philosophy/examples/02_error_handling_patterns/main.go
// CHAPTER: 36 — Error Handling Philosophy
// TOPIC: Error reduction patterns — errWriter, must helpers, early return,
//        error groups, and when NOT to use panic.
//
// Run (from the chapter folder):
//   go run ./examples/02_error_handling_patterns

package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// ERRWRITER PATTERN
//
// When writing to a stream produces many errors, track the first one
// in a struct so callers can defer the check.
// ─────────────────────────────────────────────────────────────────────────────

type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) Write(s string) {
	if ew.err != nil {
		return // short-circuit after first error
	}
	_, ew.err = fmt.Fprint(ew.w, s)
}

func (ew *errWriter) Writef(format string, args ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintf(ew.w, format, args...)
}

// Without errWriter: every write needs an explicit check.
func buildHTMLBad(w io.Writer, title, body string) error {
	if _, err := fmt.Fprintf(w, "<html><head><title>%s</title></head>", title); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "<body>%s</body>", body); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "</html>"); err != nil {
		return err
	}
	return nil
}

// With errWriter: check once at the end.
func buildHTMLGood(w io.Writer, title, body string) error {
	ew := &errWriter{w: w}
	ew.Writef("<html><head><title>%s</title></head>", title)
	ew.Writef("<body>%s</body>", body)
	ew.Write("</html>")
	return ew.err
}

// ─────────────────────────────────────────────────────────────────────────────
// MUST HELPER
//
// For initialisation-time operations that must succeed.
// Never use Must in request-handling paths — panics on every request.
// ─────────────────────────────────────────────────────────────────────────────

func Must[T any](v T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("must: %v", err))
	}
	return v
}

func parsePositive(s string) (int, error) {
	n := 0
	_, err := fmt.Sscan(s, &n)
	if err != nil {
		return 0, fmt.Errorf("not a number: %q", s)
	}
	if n <= 0 {
		return 0, fmt.Errorf("must be positive, got %d", n)
	}
	return n, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// EARLY RETURN / GUARD CLAUSES
//
// Reject invalid inputs at the top of a function rather than nesting.
// ─────────────────────────────────────────────────────────────────────────────

// Bad: deeply nested.
func processFileBad(content string, maxLines int) ([]string, error) {
	if content != "" {
		if maxLines > 0 {
			lines := strings.Split(content, "\n")
			if len(lines) > 0 {
				var result []string
				for i, line := range lines {
					if i >= maxLines {
						break
					}
					if strings.TrimSpace(line) != "" {
						result = append(result, line)
					}
				}
				return result, nil
			}
			return nil, fmt.Errorf("no lines")
		}
		return nil, fmt.Errorf("maxLines must be positive")
	}
	return nil, fmt.Errorf("content is empty")
}

// Good: guard clauses at top, happy path runs straight.
func processFileGood(content string, maxLines int) ([]string, error) {
	if content == "" {
		return nil, fmt.Errorf("content is empty")
	}
	if maxLines <= 0 {
		return nil, fmt.Errorf("maxLines must be positive")
	}

	lines := strings.Split(content, "\n")
	var result []string
	for i, line := range lines {
		if i >= maxLines {
			break
		}
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return result, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// PANIC / RECOVER — when panic is appropriate
//
// Use panic only for:
//   1. Programmer errors (assertion failure, impossible state)
//   2. Initialisation failures (Must pattern at startup)
//   3. Propagating errors across goroutine boundaries (rare; use channels instead)
//
// Never use panic for expected errors (not found, validation failed, network error).
// ─────────────────────────────────────────────────────────────────────────────

// safeDiv panics on division by zero — a programmer error (precondition violation).
func safeDiv(a, b int) int {
	if b == 0 {
		panic("safeDiv: divisor must be non-zero")
	}
	return a / b
}

// withRecovery converts a panic into an error — used at layer boundaries (e.g., HTTP handler).
func withRecovery(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered panic: %v", r)
		}
	}()
	fn()
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// COLLECTING MULTIPLE ERRORS
// ─────────────────────────────────────────────────────────────────────────────

type multiErr struct{ errs []error }

func (m *multiErr) Add(err error) {
	if err != nil {
		m.errs = append(m.errs, err)
	}
}

func (m *multiErr) Err() error {
	return errors.Join(m.errs...)
}

func validateUser(name, email string, age int) error {
	var me multiErr
	if strings.TrimSpace(name) == "" {
		me.Add(fmt.Errorf("name: required"))
	}
	if !strings.Contains(email, "@") {
		me.Add(fmt.Errorf("email: invalid format"))
	}
	if age < 0 || age > 150 {
		me.Add(fmt.Errorf("age: must be 0–150, got %d", age))
	}
	return me.Err()
}

func main() {
	fmt.Println("=== errWriter pattern ===")
	var sb1, sb2 strings.Builder
	_ = buildHTMLBad(&sb1, "Test Page", "<p>Hello</p>")
	_ = buildHTMLGood(&sb2, "Test Page", "<p>Hello</p>")
	fmt.Println("  bad  output:", sb1.String())
	fmt.Println("  good output:", sb2.String())
	same := sb1.String() == sb2.String()
	fmt.Println("  outputs equal:", same)

	fmt.Println()
	fmt.Println("=== Must helper ===")
	n := Must(parsePositive("42"))
	fmt.Println("  Must(parsePositive(\"42\")):", n)

	err := withRecovery(func() {
		_ = Must(parsePositive("abc"))
	})
	fmt.Println("  Must(parsePositive(\"abc\")) panicked, recovered:", err)

	fmt.Println()
	fmt.Println("=== Early return (guard clauses) ===")
	content := "  line one  \n\nline two\n  line three  \n\nline four"
	resBad, _ := processFileBad(content, 3)
	resGood, _ := processFileGood(content, 3)
	fmt.Println("  bad result: ", resBad)
	fmt.Println("  good result:", resGood)

	fmt.Println()
	fmt.Println("=== panic / recover ===")
	fmt.Println("  safeDiv(10, 2):", safeDiv(10, 2))
	err2 := withRecovery(func() {
		_ = safeDiv(10, 0)
	})
	fmt.Println("  safeDiv(10, 0) recovered:", err2)

	fmt.Println()
	fmt.Println("=== Multi-error collection ===")
	err3 := validateUser("", "not-an-email", 200)
	if err3 != nil {
		fmt.Println("  all validation errors:")
		for _, e := range err3.(interface{ Unwrap() []error }).Unwrap() {
			fmt.Println("   ", e)
		}
	}
	err4 := validateUser("Alice", "alice@example.com", 30)
	fmt.Println("  valid user error:", err4)
}
