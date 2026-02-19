package main

// =============================================================================
// MODULE 04: ERROR HANDLING — Go's explicit, no-exception approach
// =============================================================================
// Run: go run 04_error_handling/main.go
//
// Go philosophy: errors are VALUES, not exceptions.
// This forces you to think about every failure path explicitly.
// No try/catch — errors are returned and checked inline.
// =============================================================================

import (
	"errors"
	"fmt"
	"strconv"
)

// =============================================================================
// THE error INTERFACE — built into Go
// =============================================================================
// type error interface {
//     Error() string
// }
// Any type with an Error() string method IS an error.

// =============================================================================
// CREATING ERRORS — multiple ways
// =============================================================================

// Way 1: errors.New — simple string error
var ErrDivisionByZero = errors.New("division by zero")
var ErrNegativeNumber = errors.New("negative number not allowed")

// Package-level sentinel errors (start with Err by convention)
var (
	ErrNotFound   = errors.New("not found")
	ErrPermission = errors.New("permission denied")
	ErrTimeout    = errors.New("operation timed out")
)

// Way 2: fmt.Errorf — formatted error message
func validateAge(age int) error {
	if age < 0 {
		return fmt.Errorf("age cannot be negative: got %d", age)
	}
	if age > 150 {
		return fmt.Errorf("age %d is unrealistically large", age)
	}
	return nil // nil = no error
}

// Way 3: Custom error type — carries extra context
// Implement the error interface by adding Error() string method
type ValidationError struct {
	Field   string
	Message string
	Value   interface{}
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field %q: %s (got: %v)", e.Field, e.Message, e.Value)
}

type DatabaseError struct {
	Code    int
	Message string
	Query   string
}

func (e *DatabaseError) Error() string {
	return fmt.Sprintf("database error %d: %s (query: %s)", e.Code, e.Message, e.Query)
}

// Another custom error — for network operations
type NetworkError struct {
	Host    string
	Port    int
	Timeout bool
	Err     error // wrapped underlying error
}

func (e *NetworkError) Error() string {
	if e.Timeout {
		return fmt.Sprintf("connection to %s:%d timed out", e.Host, e.Port)
	}
	return fmt.Sprintf("network error connecting to %s:%d: %v", e.Host, e.Port, e.Err)
}

// Unwrap — for error chain traversal (Go 1.13+)
func (e *NetworkError) Unwrap() error {
	return e.Err
}

// =============================================================================
// ERROR WRAPPING — Go 1.13+ feature
// =============================================================================
// Use %w verb in fmt.Errorf to WRAP an error.
// Wrapped errors preserve the chain — you can unwrap them later.

func readConfig(filename string) error {
	// Simulating a read failure
	err := fmt.Errorf("file not found: %s", filename)
	// Wrap with context — %w preserves the error for errors.Is/errors.As
	return fmt.Errorf("readConfig failed: %w", err)
}

func loadApp(configPath string) error {
	err := readConfig(configPath)
	if err != nil {
		return fmt.Errorf("loadApp: %w", err) // wrap again — builds a chain
	}
	return nil
}

// =============================================================================
// ERROR INSPECTION — errors.Is and errors.As
// =============================================================================

// errors.Is: checks if ANY error in the chain matches a target
// errors.As: checks if ANY error in the chain can be cast to a target type

func connectDB(host string) error {
	underlying := ErrTimeout // sentinel error
	return fmt.Errorf("connectDB to %s: %w", host, underlying)
}

func queryUser(id int) (*struct{ Name string }, error) {
	if id <= 0 {
		return nil, &ValidationError{
			Field:   "id",
			Message: "must be positive",
			Value:   id,
		}
	}
	if id > 100 {
		return nil, &DatabaseError{
			Code:    404,
			Message: "user not found",
			Query:   fmt.Sprintf("SELECT * FROM users WHERE id=%d", id),
		}
	}
	return &struct{ Name string }{Name: "Achyut"}, nil
}

// =============================================================================
// PANIC AND RECOVER — for truly exceptional situations
// =============================================================================
// panic: for unrecoverable errors (programming bugs, not user errors)
//   - Index out of bounds
//   - Nil pointer dereference
//   - Type assertion failure without ok check
// recover: only inside deferred functions — stops a panic

func mustParseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic(fmt.Sprintf("mustParseInt: cannot parse %q as int", s))
	}
	return n
}

// Safe wrapper using recover
func safeOperation(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// Convert panic to error
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()
	fn()
	return nil
}

// =============================================================================
// REAL-WORLD PATTERNS
// =============================================================================

// Pattern 1: Early return on error (avoid deep nesting)
func processUser(id int, name string, age int) error {
	if id <= 0 {
		return fmt.Errorf("processUser: %w", &ValidationError{"id", "must be positive", id})
	}
	if name == "" {
		return fmt.Errorf("processUser: %w", &ValidationError{"name", "cannot be empty", name})
	}
	if err := validateAge(age); err != nil {
		return fmt.Errorf("processUser: %w", err)
	}
	fmt.Printf("Processing user: id=%d, name=%s, age=%d\n", id, name, age)
	return nil
}

// Pattern 2: Multiple error collection
type MultiError struct {
	Errors []error
}

func (m *MultiError) Error() string {
	if len(m.Errors) == 0 {
		return "no errors"
	}
	msg := fmt.Sprintf("%d errors occurred:\n", len(m.Errors))
	for i, err := range m.Errors {
		msg += fmt.Sprintf("  %d: %v\n", i+1, err)
	}
	return msg
}

func (m *MultiError) Add(err error) {
	if err != nil {
		m.Errors = append(m.Errors, err)
	}
}

func (m *MultiError) HasErrors() bool {
	return len(m.Errors) > 0
}

func validateForm(name string, age int, email string) error {
	var errs MultiError
	if name == "" {
		errs.Add(&ValidationError{"name", "required", name})
	}
	if age < 0 || age > 150 {
		errs.Add(&ValidationError{"age", "must be 0-150", age})
	}
	if email == "" {
		errs.Add(&ValidationError{"email", "required", email})
	}
	if errs.HasErrors() {
		return &errs
	}
	return nil
}

// Pattern 3: Result type simulation using closure/struct
type Result[T any] struct {
	value T
	err   error
}

func Ok[T any](v T) Result[T]       { return Result[T]{value: v} }
func Err[T any](e error) Result[T]  { return Result[T]{err: e} }
func (r Result[T]) Unwrap() (T, error) { return r.value, r.err }
func (r Result[T]) IsOk() bool      { return r.err == nil }

// =============================================================================
// MAIN
// =============================================================================

func main() {
	fmt.Println("=== MODULE 04: ERROR HANDLING ===")

	// -------------------------------------------------------------------------
	// SECTION 1: Basic error handling
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Basic Error Handling ---")

	// The idiomatic Go pattern: if err != nil
	result, err := strconv.Atoi("42")
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Parsed:", result)
	}

	// Error case
	result2, err2 := strconv.Atoi("not a number")
	if err2 != nil {
		fmt.Println("Parse error:", err2)
	} else {
		fmt.Println("Parsed:", result2)
	}

	// -------------------------------------------------------------------------
	// SECTION 2: errors.New and fmt.Errorf
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Creating Errors ---")

	if err := validateAge(25); err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Age 25 is valid")
	}

	if err := validateAge(-5); err != nil {
		fmt.Println("Error:", err)
	}

	if err := validateAge(200); err != nil {
		fmt.Println("Error:", err)
	}

	// -------------------------------------------------------------------------
	// SECTION 3: Custom error types
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Custom Error Types ---")

	_, err3 := queryUser(-1)
	if err3 != nil {
		fmt.Println("Error:", err3)
	}

	_, err4 := queryUser(500)
	if err4 != nil {
		fmt.Println("Error:", err4)
	}

	user, err5 := queryUser(1)
	if err5 == nil {
		fmt.Println("Found user:", user.Name)
	}

	// -------------------------------------------------------------------------
	// SECTION 4: Error wrapping and unwrapping
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Error Wrapping ---")

	err6 := loadApp("config.yaml")
	if err6 != nil {
		fmt.Println("Error:", err6)
		// The full chain is preserved:
		// loadApp: readConfig failed: file not found: config.yaml
	}

	// errors.Is — check if error chain contains a specific sentinel
	fmt.Println("\n--- errors.Is ---")

	err7 := connectDB("localhost")
	fmt.Println("err7:", err7)
	fmt.Println("Is ErrTimeout:", errors.Is(err7, ErrTimeout))  // true (wrapped)
	fmt.Println("Is ErrNotFound:", errors.Is(err7, ErrNotFound)) // false

	// errors.As — check if error chain contains a specific type
	fmt.Println("\n--- errors.As ---")

	_, err8 := queryUser(-5)
	var valErr *ValidationError
	if errors.As(err8, &valErr) {
		fmt.Printf("Validation error! Field: %s, Message: %s\n", valErr.Field, valErr.Message)
	}

	_, err9 := queryUser(500)
	var dbErr *DatabaseError
	if errors.As(err9, &dbErr) {
		fmt.Printf("Database error! Code: %d, Message: %s\n", dbErr.Code, dbErr.Message)
	}

	// -------------------------------------------------------------------------
	// SECTION 5: Panic and recover
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Panic and Recover ---")

	// Safe call — no panic
	err10 := safeOperation(func() {
		fmt.Println("Safe operation succeeded")
	})
	fmt.Println("safeOperation error:", err10)

	// Panic recovered
	err11 := safeOperation(func() {
		mustParseInt("not a number")
	})
	fmt.Println("safeOperation with panic:", err11)

	// -------------------------------------------------------------------------
	// SECTION 6: Real-world patterns
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Early Return Pattern ---")

	if err := processUser(1, "Achyut", 25); err != nil {
		fmt.Println("Error:", err)
	}

	if err := processUser(0, "Achyut", 25); err != nil {
		fmt.Println("Error:", err)
	}

	if err := processUser(1, "", 25); err != nil {
		fmt.Println("Error:", err)
	}

	fmt.Println("\n--- Multiple Error Collection ---")

	if err := validateForm("", -1, ""); err != nil {
		fmt.Println("Form errors:", err)
	}

	if err := validateForm("Achyut", 25, "achyut@test.com"); err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Form is valid!")
	}

	// -------------------------------------------------------------------------
	// SECTION 7: Result type (generics pattern)
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Result Type Pattern ---")

	r := Ok(42)
	val, err12 := r.Unwrap()
	fmt.Println("Result Ok:", val, err12)

	r2 := Err[int](errors.New("something went wrong"))
	val2, err13 := r2.Unwrap()
	fmt.Println("Result Err:", val2, err13)

	// -------------------------------------------------------------------------
	// SECTION 8: Error handling best practices
	// -------------------------------------------------------------------------
	fmt.Println("\n--- Best Practices Summary ---")
	fmt.Println(`
1. Always check errors — never ignore with _
2. Add context when wrapping: fmt.Errorf("context: %w", err)
3. Use sentinel errors (var ErrX = errors.New(...)) for expected conditions
4. Use custom error types when callers need to inspect fields
5. Use errors.Is for sentinel matching, errors.As for type matching
6. panic only for unrecoverable programmer errors
7. recover only in library code to convert panics to errors
8. Never let panic propagate across API boundaries
`)

	fmt.Println("=== MODULE 04 COMPLETE ===")
}
