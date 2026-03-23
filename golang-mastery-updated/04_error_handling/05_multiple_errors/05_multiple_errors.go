// 05_multiple_errors.go
//
// COLLECTING MULTIPLE ERRORS
// ==========================
// Most error handling follows the "fail fast" pattern: the moment one thing
// goes wrong, stop and return the error. This is correct for sequential
// operations where later steps depend on earlier ones.
//
// But some operations are INDEPENDENT and you want to check ALL of them
// before reporting to the caller. The canonical example is form validation:
// if a user submits a form, you want to report every validation error
// in one response, not just the first one. Forcing the user to fix one
// field, resubmit, discover the next error, repeat is terrible UX.
//
// FAIL FAST vs COLLECT ALL
// -------------------------
// Fail fast: sequential steps where step N requires step N-1 to succeed.
//   → return the first error, stop immediately.
//
// Collect all: independent checks where each can be evaluated regardless
//   of the others.
//   → run all checks, gather all errors, return them together.
//
// ERRORS.JOIN (Go 1.20+)
// -----------------------
// errors.Join(errs...) returns a single error that contains ALL the
// provided errors. It returns nil if all inputs are nil.
//
// The joined error:
//   - Has an Error() string that is the newline-separated messages of all
//     non-nil input errors.
//   - Implements Unwrap() []error (plural Unwrap — Go 1.20 addition).
//   - errors.Is and errors.As work on it: they check each sub-error.
//
// BEFORE GO 1.20 — building your own MultiError
// -----------------------------------------------
// errors.Join did not exist before Go 1.20. The community built custom
// MultiError types. We show both approaches here so you understand the
// history and can read older code.

package main

import (
	"errors"
	"fmt"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 1: Pre-1.20 MultiError implementation
// ─────────────────────────────────────────────────────────────────────────────

// MultiError holds a slice of errors.
// Before errors.Join, every project that needed this had its own version.
type MultiError struct {
	Errors []error
}

// Error produces a human-readable joined message.
// We separate errors with "; " so the result fits on one log line.
// (errors.Join uses "\n" which is better for structured output.)
func (m *MultiError) Error() string {
	if len(m.Errors) == 0 {
		return ""
	}
	msgs := make([]string, 0, len(m.Errors))
	for _, e := range m.Errors {
		msgs = append(msgs, e.Error())
	}
	return strings.Join(msgs, "; ")
}

// Unwrap returns the slice — this is the Go 1.20 interface for multi-wrapping.
// errors.Is / errors.As iterate over this slice when they find it.
// (The method signature must be exactly `Unwrap() []error`.)
func (m *MultiError) Unwrap() []error {
	return m.Errors
}

// OrNil returns nil if there are no errors, otherwise returns m.
// This is an important pattern: callers expect nil to mean "no error",
// not an empty MultiError. An empty MultiError is truthy (non-nil interface).
func (m *MultiError) OrNil() error {
	if len(m.Errors) == 0 {
		return nil
	}
	return m
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 2: Validation use case
// ─────────────────────────────────────────────────────────────────────────────

// UserInput represents a form submission.
type UserInput struct {
	Username string
	Email    string
	Age      int
	Password string
}

// Sentinel errors for field validation — callers can use errors.Is.
var (
	ErrUsernameEmpty   = errors.New("username: must not be empty")
	ErrUsernameTooShort = errors.New("username: must be at least 3 characters")
	ErrEmailInvalid    = errors.New("email: must contain @")
	ErrAgeTooYoung     = errors.New("age: must be 18 or older")
	ErrPasswordWeak    = errors.New("password: must be at least 8 characters")
)

// validateInput runs ALL validations and collects every error.
// Uses our hand-rolled MultiError.
func validateInput(input UserInput) error {
	var me MultiError

	if input.Username == "" {
		me.Errors = append(me.Errors, ErrUsernameEmpty)
	} else if len(input.Username) < 3 {
		me.Errors = append(me.Errors, ErrUsernameTooShort)
	}

	if !strings.Contains(input.Email, "@") {
		me.Errors = append(me.Errors, ErrEmailInvalid)
	}

	if input.Age < 18 {
		me.Errors = append(me.Errors, ErrAgeTooYoung)
	}

	if len(input.Password) < 8 {
		me.Errors = append(me.Errors, ErrPasswordWeak)
	}

	return me.OrNil() // returns nil if Errors is empty
}

// validateInputJoin does the same thing using errors.Join (Go 1.20+).
// It is functionally equivalent but uses the standard library.
func validateInputJoin(input UserInput) error {
	var errs []error

	if input.Username == "" {
		errs = append(errs, ErrUsernameEmpty)
	} else if len(input.Username) < 3 {
		errs = append(errs, ErrUsernameTooShort)
	}

	if !strings.Contains(input.Email, "@") {
		errs = append(errs, ErrEmailInvalid)
	}

	if input.Age < 18 {
		errs = append(errs, ErrAgeTooYoung)
	}

	if len(input.Password) < 8 {
		errs = append(errs, ErrPasswordWeak)
	}

	// errors.Join returns nil when all errs are nil (or slice is empty).
	return errors.Join(errs...)
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 3: errors.Is works on joined errors
// ─────────────────────────────────────────────────────────────────────────────
// Both MultiError (with Unwrap() []error) and errors.Join's result
// support errors.Is traversal over each contained error.

func checkSpecificError(err error) {
	if err == nil {
		return
	}
	fmt.Println("  Checking specific errors in multi-error:")
	fmt.Printf("    errors.Is(ErrEmailInvalid):  %v\n", errors.Is(err, ErrEmailInvalid))
	fmt.Printf("    errors.Is(ErrAgeTooYoung):   %v\n", errors.Is(err, ErrAgeTooYoung))
	fmt.Printf("    errors.Is(ErrUsernameEmpty): %v\n", errors.Is(err, ErrUsernameEmpty))
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 4: Collecting errors from concurrent operations
// ─────────────────────────────────────────────────────────────────────────────
// A common pattern: run a batch of independent operations, collect all
// failures, continue with the successes.

type Result struct {
	Input  string
	Output string
	Err    error
}

// processBatch processes items independently, collecting all errors.
// This is the "collect, don't fail-fast" pattern for batch operations.
func processBatch(items []string) ([]Result, error) {
	results := make([]Result, len(items))
	var errs []error

	for i, item := range items {
		r := Result{Input: item}
		if item == "" {
			r.Err = fmt.Errorf("item[%d]: empty input not allowed", i)
			errs = append(errs, r.Err)
		} else if strings.HasPrefix(item, "bad_") {
			r.Err = fmt.Errorf("item[%d] %q: rejected by policy", i, item)
			errs = append(errs, r.Err)
		} else {
			r.Output = strings.ToUpper(item)
		}
		results[i] = r
	}

	// Return ALL results (successful ones too) AND a combined error.
	// The caller decides what to do — maybe retry just the failures.
	return results, errors.Join(errs...)
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 5: Unwrapping a joined error to get individual errors
// ─────────────────────────────────────────────────────────────────────────────
// errors.Join returns a type that implements Unwrap() []error.
// You can access the individual errors by type-asserting to the interface.

type multiUnwrapper interface {
	Unwrap() []error
}

// extractAll extracts individual errors from a joined error.
func extractAll(err error) []error {
	if err == nil {
		return nil
	}
	if mu, ok := err.(multiUnwrapper); ok {
		return mu.Unwrap()
	}
	return []error{err}
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 6: The OrNil trap — NEVER return an empty custom error type as nil
// ─────────────────────────────────────────────────────────────────────────────
//
// This is a classic Go pitfall. An interface value is nil only if both
// its type and value are nil. A *MultiError with zero errors is NOT nil
// when stored in an error interface — it has a type (*MultiError).
//
// The OrNil() method on MultiError (above) solves this.

func badReturn() error {
	var me *MultiError // nil pointer
	// WRONG: returning a typed nil as an error interface.
	// The interface is NOT nil — it holds (*MultiError, nil pointer).
	// Callers who check "if err != nil" will think there is an error!
	return me // DO NOT do this
}

func goodReturn() error {
	var me MultiError
	if len(me.Errors) == 0 {
		return nil // return untyped nil — the interface IS nil
	}
	return &me
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== 05: Multiple Errors ===")
	fmt.Println()

	// ── 1. validateInput with MultiError ─────────────────────────────────────
	fmt.Println("── validateInput (MultiError, pre-1.20 style) ──")

	bad := UserInput{Username: "x", Email: "notanemail", Age: 15, Password: "abc"}
	good := UserInput{Username: "alice", Email: "alice@example.com", Age: 25, Password: "securepass123"}

	if err := validateInput(bad); err != nil {
		fmt.Println("  bad input errors:", err)
		checkSpecificError(err)
	}
	if err := validateInput(good); err == nil {
		fmt.Println("  good input: valid")
	}
	fmt.Println()

	// ── 2. validateInputJoin with errors.Join ────────────────────────────────
	fmt.Println("── validateInputJoin (errors.Join, Go 1.20+) ──")

	if err := validateInputJoin(bad); err != nil {
		fmt.Println("  bad input errors:")
		// errors.Join uses newline separator — each error on its own line.
		for i, line := range strings.Split(err.Error(), "\n") {
			fmt.Printf("    [%d] %s\n", i, line)
		}
		checkSpecificError(err)
	}
	if err := validateInputJoin(good); err == nil {
		fmt.Println("  good input: valid")
	}
	fmt.Println()

	// ── 3. processBatch ───────────────────────────────────────────────────────
	fmt.Println("── processBatch ──")

	items := []string{"hello", "", "world", "bad_item", "foo"}
	results, err := processBatch(items)
	for _, r := range results {
		if r.Err != nil {
			fmt.Printf("  [FAIL] %q → %v\n", r.Input, r.Err)
		} else {
			fmt.Printf("  [OK]   %q → %q\n", r.Input, r.Output)
		}
	}
	if err != nil {
		fmt.Printf("  Combined error: %v\n", err)
		individal := extractAll(err)
		fmt.Printf("  Individual errors (%d):\n", len(individal))
		for i, e := range individal {
			fmt.Printf("    [%d] %v\n", i, e)
		}
	}
	fmt.Println()

	// ── 4. The OrNil trap ────────────────────────────────────────────────────
	fmt.Println("── OrNil trap demonstration ──")

	badErr := badReturn()
	goodErr := goodReturn()
	fmt.Printf("  badReturn()  != nil: %v  ← typed nil interface is non-nil!\n", badErr != nil)
	fmt.Printf("  goodReturn() != nil: %v  ← untyped nil is correctly nil\n", goodErr != nil)
	fmt.Println()

	// ── 5. errors.Join returns nil when all inputs are nil ───────────────────
	fmt.Println("── errors.Join nil behavior ──")
	j1 := errors.Join(nil, nil, nil)
	j2 := errors.Join(errors.New("a"), nil, errors.New("b"))
	fmt.Printf("  Join(nil,nil,nil) == nil: %v\n", j1 == nil)
	fmt.Printf("  Join(a, nil, b)   = %q\n", j2.Error())
	fmt.Println()

	fmt.Println("Key takeaways:")
	fmt.Println("  1. Collect all errors when checks are independent (validation)")
	fmt.Println("  2. Fail fast when steps are sequential and dependent")
	fmt.Println("  3. errors.Join (Go 1.20) is the standard way to join errors")
	fmt.Println("  4. Implement Unwrap() []error on custom multi-error types")
	fmt.Println("  5. errors.Is/As traverse multi-error slices automatically")
	fmt.Println("  6. OrNil pattern: return nil (untyped), not an empty custom type")
}
