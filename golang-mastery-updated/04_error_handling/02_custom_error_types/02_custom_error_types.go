// 02_custom_error_types.go
//
// CUSTOM ERROR TYPES IN GO
// ========================
// errors.New and fmt.Errorf give you string-based errors. They are perfect
// when the only information a caller needs is "did this succeed?" plus a
// human-readable explanation.
//
// But sometimes callers need to PROGRAMMATICALLY inspect the error — not just
// print it, but branch on its fields, retry with different parameters, or log
// structured metadata. For these cases you implement the error interface on
// your own struct.
//
// WHEN TO USE A CUSTOM TYPE vs errors.New
// ----------------------------------------
// Use errors.New / fmt.Errorf when:
//   - The caller only needs to know that something went wrong.
//   - The message is enough to log and move on.
//   - You control both the producer and the consumer and they don't need
//     structured inspection.
//
// Use a custom type when:
//   - The caller needs to read structured fields (HTTP status code, field name,
//     retry-after duration, etc.)
//   - You want to group errors into categories (all ValidationErrors, all
//     NetworkErrors) so callers can handle them uniformly.
//   - You are building a library and callers must be able to type-assert or
//     use errors.As on your errors without depending on string matching.
//
// IMPLEMENTING THE ERROR INTERFACE ON A STRUCT
// --------------------------------------------
// Any struct that has a method with the signature
//
//   func (e *YourType) Error() string
//
// satisfies the error interface. The receiver must be a POINTER (*YourType)
// if the struct is used via pointer (which is almost always — it avoids
// copying the struct when the error passes through function returns).

package main

import (
	"fmt"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 1: A basic custom error with fields
// ─────────────────────────────────────────────────────────────────────────────

// AppError is a simple domain error carrying a code, message, and timestamp.
// The timestamp is useful: when you see an error in logs, you know exactly
// when it happened without having to correlate with timestamps elsewhere.
type AppError struct {
	Code      int
	Message   string
	OccurredAt time.Time
}

// Error implements the error interface. The method must be on the POINTER
// receiver so that *AppError is what satisfies error (not AppError).
// If you return an AppError (value, not pointer) as an error, a non-nil
// interface wrapping a zero AppError would compare non-nil — subtle bug.
// Always return *AppError from functions, never AppError.
func (e *AppError) Error() string {
	return fmt.Sprintf("[%d] %s (at %s)",
		e.Code, e.Message, e.OccurredAt.Format(time.RFC3339))
}

// newAppError is a constructor. Constructors for error types are idiomatic
// because they set mandatory fields (like OccurredAt) so callers cannot forget.
func newAppError(code int, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		OccurredAt: time.Now(),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 2: Domain-specific error types
// ─────────────────────────────────────────────────────────────────────────────
// Real applications have distinct error categories that need to be handled
// differently by callers. Define a separate type for each category.

// ValidationError is returned when user-supplied input fails validation rules.
// Fields: Field (which input), Value (what was supplied), Message (why it failed).
// The caller (e.g., an HTTP handler) can use Field and Value to return a
// structured JSON error response instead of a generic 500.
type ValidationError struct {
	Field   string
	Value   interface{} // the actual value that was rejected
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed on field %q (value=%v): %s",
		e.Field, e.Value, e.Message)
}

// DatabaseError is returned when a database operation fails.
// Query carries the SQL (sanitised — never log raw queries with user data
// in production). Op identifies the operation kind.
type DatabaseError struct {
	Op      string // "SELECT", "INSERT", etc.
	Table   string
	Message string
	Cause   error // the underlying driver error, if any
}

func (e *DatabaseError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("database error [op=%s table=%s]: %s: %v",
			e.Op, e.Table, e.Message, e.Cause)
	}
	return fmt.Sprintf("database error [op=%s table=%s]: %s",
		e.Op, e.Table, e.Message)
}

// Unwrap lets errors.Is / errors.As look through DatabaseError to its Cause.
// This is the "wrapping" pattern — covered deeply in 03_error_wrapping.go.
// Adding Unwrap here makes DatabaseError a first-class member of error chains.
func (e *DatabaseError) Unwrap() error {
	return e.Cause
}

// NetworkError is returned when a remote call fails.
// RetryAfter allows callers to implement back-off without parsing the message.
type NetworkError struct {
	URL        string
	StatusCode int           // 0 if the request never reached the server
	RetryAfter time.Duration // 0 if the server did not specify
	Message    string
}

func (e *NetworkError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("network error [url=%s status=%d retry-after=%s]: %s",
			e.URL, e.StatusCode, e.RetryAfter, e.Message)
	}
	return fmt.Sprintf("network error [url=%s status=%d]: %s",
		e.URL, e.StatusCode, e.Message)
}

// IsRetryable is a domain-specific helper method. Because NetworkError is a
// concrete type, you can add methods beyond Error(). Callers that have a
// *NetworkError can call IsRetryable() directly.
func (e *NetworkError) IsRetryable() bool {
	return e.StatusCode == 503 || e.StatusCode == 429 || e.StatusCode == 0
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 3: Functions that return custom error types
// ─────────────────────────────────────────────────────────────────────────────

// validateUsername checks that a username is acceptable.
// It returns a *ValidationError so that callers who need structured info
// can type-assert, while callers who just want to print can treat it as error.
func validateUsername(username string) error {
	if username == "" {
		return &ValidationError{
			Field:   "username",
			Value:   username,
			Message: "must not be empty",
		}
	}
	if len(username) < 3 {
		return &ValidationError{
			Field:   "username",
			Value:   username,
			Message: fmt.Sprintf("must be at least 3 characters, got %d", len(username)),
		}
	}
	if len(username) > 32 {
		return &ValidationError{
			Field:   "username",
			Value:   username,
			Message: fmt.Sprintf("must be at most 32 characters, got %d", len(username)),
		}
	}
	return nil
}

// queryUser simulates a database lookup.
func queryUser(id int) (string, error) {
	if id <= 0 {
		return "", &DatabaseError{
			Op:      "SELECT",
			Table:   "users",
			Message: fmt.Sprintf("invalid user id: %d", id),
		}
	}
	if id == 999 {
		// Simulate a connection error at the driver level
		driverErr := fmt.Errorf("connection reset by peer")
		return "", &DatabaseError{
			Op:      "SELECT",
			Table:   "users",
			Message: "connection failed during query",
			Cause:   driverErr,
		}
	}
	return fmt.Sprintf("user_%d", id), nil
}

// fetchRemote simulates an HTTP call.
func fetchRemote(url string) ([]byte, error) {
	switch url {
	case "http://overloaded.example.com":
		return nil, &NetworkError{
			URL:        url,
			StatusCode: 503,
			RetryAfter: 30 * time.Second,
			Message:    "service temporarily unavailable",
		}
	case "http://ratelimited.example.com":
		return nil, &NetworkError{
			URL:        url,
			StatusCode: 429,
			RetryAfter: 60 * time.Second,
			Message:    "rate limit exceeded",
		}
	case "http://unreachable.example.com":
		return nil, &NetworkError{
			URL:        url,
			StatusCode: 0,
			Message:    "dial tcp: connection refused",
		}
	}
	return []byte(`{"ok": true}`), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 4: Type asserting on errors
// ─────────────────────────────────────────────────────────────────────────────
// When a function returns error (the interface), callers can recover the
// concrete type using a type assertion or errors.As (preferred — see
// 03_error_wrapping.go for errors.As).
//
// Type assertion: e.(*ValidationError)
//   - Direct. Works only if the error is exactly *ValidationError (no wrapping).
//   - Panics on failure unless you use the two-value form: v, ok := e.(*ValidationError)
//
// errors.As: preferred because it traverses the error chain.

func handleValidation(username string) {
	err := validateUsername(username)
	if err == nil {
		fmt.Printf("  username %q is valid\n", username)
		return
	}

	// Two-value type assertion — safe, never panics
	if ve, ok := err.(*ValidationError); ok {
		fmt.Printf("  VALIDATION ERROR — field=%q value=%v message=%q\n",
			ve.Field, ve.Value, ve.Message)
		// We could now return a structured JSON error to an HTTP client
		// without parsing the error string.
	} else {
		fmt.Println("  unexpected error:", err)
	}
}

func handleDatabase(id int) {
	name, err := queryUser(id)
	if err == nil {
		fmt.Printf("  found user: %q\n", name)
		return
	}

	if dbe, ok := err.(*DatabaseError); ok {
		fmt.Printf("  DATABASE ERROR — op=%s table=%s message=%q\n",
			dbe.Op, dbe.Table, dbe.Message)
		if dbe.Cause != nil {
			fmt.Printf("    caused by: %v\n", dbe.Cause)
		}
	} else {
		fmt.Println("  unexpected error:", err)
	}
}

func handleNetwork(url string) {
	data, err := fetchRemote(url)
	if err == nil {
		fmt.Printf("  fetched %d bytes from %q\n", len(data), url)
		return
	}

	if ne, ok := err.(*NetworkError); ok {
		fmt.Printf("  NETWORK ERROR — status=%d retryable=%v message=%q\n",
			ne.StatusCode, ne.IsRetryable(), ne.Message)
		if ne.RetryAfter > 0 {
			fmt.Printf("    retry after: %s\n", ne.RetryAfter)
		}
	} else {
		fmt.Println("  unexpected error:", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 5: Error type switch — handling multiple error categories
// ─────────────────────────────────────────────────────────────────────────────
// A type switch is syntactic sugar for a chain of type assertions.
// It is useful when a function could return several different error types.

func describeError(err error) {
	if err == nil {
		fmt.Println("  no error")
		return
	}
	switch e := err.(type) {
	case *ValidationError:
		fmt.Printf("  → ValidationError: field=%s\n", e.Field)
	case *DatabaseError:
		fmt.Printf("  → DatabaseError: op=%s table=%s\n", e.Op, e.Table)
	case *NetworkError:
		fmt.Printf("  → NetworkError: status=%d\n", e.StatusCode)
	case *AppError:
		fmt.Printf("  → AppError: code=%d\n", e.Code)
	default:
		fmt.Printf("  → unknown error type %T: %v\n", err, err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== 02: Custom Error Types ===")
	fmt.Println()

	// ── 1. AppError ──────────────────────────────────────────────────────────
	fmt.Println("── AppError (basic custom type) ──")
	ae := newAppError(404, "resource not found")
	fmt.Println(" ", ae)
	// Access fields directly
	fmt.Printf("  code=%d message=%q\n", ae.Code, ae.Message)
	fmt.Println()

	// ── 2. ValidationError ───────────────────────────────────────────────────
	fmt.Println("── validateUsername ──")
	usernames := []string{"", "ab", "alice", "this_username_is_way_too_long_to_be_valid_in_our_system"}
	for _, u := range usernames {
		handleValidation(u)
	}
	fmt.Println()

	// ── 3. DatabaseError ─────────────────────────────────────────────────────
	fmt.Println("── queryUser ──")
	for _, id := range []int{1, -1, 999} {
		fmt.Printf("  id=%d:\n", id)
		handleDatabase(id)
	}
	fmt.Println()

	// ── 4. NetworkError ───────────────────────────────────────────────────────
	fmt.Println("── fetchRemote ──")
	urls := []string{
		"http://api.example.com",
		"http://overloaded.example.com",
		"http://ratelimited.example.com",
		"http://unreachable.example.com",
	}
	for _, url := range urls {
		fmt.Printf("  url=%q:\n", url)
		handleNetwork(url)
	}
	fmt.Println()

	// ── 5. Type switch ────────────────────────────────────────────────────────
	fmt.Println("── type switch ──")
	errors := []error{
		nil,
		&ValidationError{Field: "email", Value: "not-an-email", Message: "invalid format"},
		&DatabaseError{Op: "INSERT", Table: "orders", Message: "duplicate key"},
		&NetworkError{URL: "http://x.example.com", StatusCode: 500, Message: "internal server error"},
		newAppError(503, "service unavailable"),
		fmt.Errorf("some unrecognised error"),
	}
	for _, e := range errors {
		describeError(e)
	}
	fmt.Println()

	fmt.Println("Key takeaways:")
	fmt.Println("  1. Implement error interface: add Error() string on a *Struct")
	fmt.Println("  2. Custom types carry structured data callers can inspect")
	fmt.Println("  3. Type assertion (v, ok := err.(*T)) — safe two-value form")
	fmt.Println("  4. Type switch handles multiple error categories cleanly")
	fmt.Println("  5. Add domain methods (IsRetryable) beyond the error interface")
	fmt.Println("  6. Use errors.As (not type assertion) when errors may be wrapped")
}
