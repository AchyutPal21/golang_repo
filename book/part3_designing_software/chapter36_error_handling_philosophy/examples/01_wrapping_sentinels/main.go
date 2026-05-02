// FILE: book/part3_designing_software/chapter36_error_handling_philosophy/examples/01_wrapping_sentinels/main.go
// CHAPTER: 36 — Error Handling Philosophy
// TOPIC: Sentinel errors, wrapping with %w, errors.Is, errors.As,
//        and the golden rule: handle OR propagate, never both.
//
// Run (from the chapter folder):
//   go run ./examples/01_wrapping_sentinels

package main

import (
	"errors"
	"fmt"
	"strconv"
)

// ─────────────────────────────────────────────────────────────────────────────
// SENTINEL ERRORS
//
// Sentinel errors are package-level values. Callers compare with errors.Is.
// Use them for conditions the caller needs to distinguish programmatically.
// ─────────────────────────────────────────────────────────────────────────────

var (
	ErrNotFound       = errors.New("not found")
	ErrPermissionDenied = errors.New("permission denied")
	ErrTimeout        = errors.New("timeout")
)

// ─────────────────────────────────────────────────────────────────────────────
// ERROR WRAPPING WITH %w
//
// fmt.Errorf("op: %w", err) adds context while preserving the original error.
// errors.Is unwraps the chain to find the sentinel.
// errors.As unwraps the chain to find a value of a specific type.
// ─────────────────────────────────────────────────────────────────────────────

func fetchUser(id int) (string, error) {
	if id <= 0 {
		return "", fmt.Errorf("fetchUser: invalid id %d: %w", id, ErrNotFound)
	}
	users := map[int]string{1: "alice", 2: "bob"}
	name, ok := users[id]
	if !ok {
		return "", fmt.Errorf("fetchUser: user %d: %w", id, ErrNotFound)
	}
	return name, nil
}

func fetchProfile(id int) (string, error) {
	name, err := fetchUser(id)
	if err != nil {
		return "", fmt.Errorf("fetchProfile: %w", err) // wrap, don't handle
	}
	return "profile:" + name, nil
}

func handleRequest(id int) error {
	profile, err := fetchProfile(id)
	if err != nil {
		return fmt.Errorf("handleRequest: %w", err)
	}
	fmt.Println(" ", profile)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// ERRORS.IS — checks the entire unwrap chain for a sentinel match
// ─────────────────────────────────────────────────────────────────────────────

func demoIs() {
	fmt.Println("=== errors.Is ===")

	err := handleRequest(99) // user not found
	fmt.Println("  raw error:", err)
	fmt.Println("  errors.Is(err, ErrNotFound):", errors.Is(err, ErrNotFound))

	err2 := handleRequest(1) // success
	fmt.Println("  success err:", err2)

	// Direct sentinel comparison FAILS for wrapped errors.
	wrapped := fmt.Errorf("layer: %w", ErrNotFound)
	fmt.Println("  direct == (wrong):", wrapped == ErrNotFound)
	fmt.Println("  errors.Is (correct):", errors.Is(wrapped, ErrNotFound))
}

// ─────────────────────────────────────────────────────────────────────────────
// ERRORS.AS — extracts a specific error type from the chain
// ─────────────────────────────────────────────────────────────────────────────

// ValidationError carries field-level detail.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation: %s: %s", e.Field, e.Message)
}

func parseAge(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, &ValidationError{Field: "age", Message: "must be a number"}
	}
	if n < 0 || n > 150 {
		return 0, &ValidationError{Field: "age", Message: fmt.Sprintf("must be 0–150, got %d", n)}
	}
	return n, nil
}

func processAge(s string) (int, error) {
	age, err := parseAge(s)
	if err != nil {
		return 0, fmt.Errorf("processAge: %w", err) // wrap preserves type
	}
	return age, nil
}

func demoAs() {
	fmt.Println()
	fmt.Println("=== errors.As ===")

	for _, input := range []string{"25", "abc", "-5", "200"} {
		age, err := processAge(input)
		if err != nil {
			var ve *ValidationError
			if errors.As(err, &ve) {
				fmt.Printf("  input=%q  field=%s  msg=%s\n", input, ve.Field, ve.Message)
			} else {
				fmt.Printf("  input=%q  unexpected error: %v\n", input, err)
			}
		} else {
			fmt.Printf("  input=%q  age=%d  ok\n", input, age)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// THE GOLDEN RULE: handle OR propagate, never both
// ─────────────────────────────────────────────────────────────────────────────

func badHandler(id int) error {
	name, err := fetchUser(id)
	if err != nil {
		fmt.Println("ERROR:", err) // log AND return — caller logs again
		return err
	}
	_ = name
	return nil
}

func goodHandler(id int) error {
	name, err := fetchUser(id)
	if err != nil {
		return fmt.Errorf("goodHandler: %w", err) // propagate only
	}
	_ = name
	return nil
}

func goodCaller(id int) {
	if err := goodHandler(id); err != nil {
		// Handle here — only one place logs.
		fmt.Println("  handled at boundary:", err)
	}
}

func demoGoldenRule() {
	fmt.Println()
	fmt.Println("=== Golden rule: handle OR propagate ===")
	fmt.Println("  bad (logs twice, suppressed in example):")
	fmt.Println("  good (single log at boundary):")
	goodCaller(99)
}

// ─────────────────────────────────────────────────────────────────────────────
// errors.Join — combine multiple errors (Go 1.20+)
// ─────────────────────────────────────────────────────────────────────────────

type FieldError struct {
	Field   string
	Message string
}

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }

func validateForm(name, email string, age int) error {
	var errs []error
	if name == "" {
		errs = append(errs, &FieldError{"name", "is required"})
	}
	if email == "" {
		errs = append(errs, &FieldError{"email", "is required"})
	}
	if age < 18 {
		errs = append(errs, &FieldError{"age", fmt.Sprintf("must be ≥18, got %d", age)})
	}
	return errors.Join(errs...) // nil if errs is empty
}

func demoJoin() {
	fmt.Println()
	fmt.Println("=== errors.Join (multi-error) ===")

	err := validateForm("", "", 15)
	if err != nil {
		fmt.Println("  all errors:")
		// errors.Join wraps each error; unwrap yields all.
		for _, e := range err.(interface{ Unwrap() []error }).Unwrap() {
			var fe *FieldError
			if errors.As(e, &fe) {
				fmt.Printf("    field=%s  msg=%s\n", fe.Field, fe.Message)
			}
		}
	}

	err2 := validateForm("Alice", "alice@example.com", 25)
	fmt.Println("  valid form error:", err2)
}

func main() {
	demoIs()
	demoAs()
	demoGoldenRule()
	demoJoin()
}
