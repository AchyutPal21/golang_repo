// FILE: book/part3_designing_software/chapter37_custom_error_types/examples/01_error_types/main.go
// CHAPTER: 37 — Custom Error Types
// TOPIC: Implementing Error(), Unwrap(), custom Is(), and custom As().
//        When to use a struct vs a sentinel vs a string error.
//
// Run (from the chapter folder):
//   go run ./examples/01_error_types

package main

import (
	"errors"
	"fmt"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// SIMPLE STRUCT ERROR
//
// Carries metadata beyond a plain string.
// Implement Error() to satisfy the error interface.
// ─────────────────────────────────────────────────────────────────────────────

type NotFoundError struct {
	Resource string
	ID       any
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with id %v not found", e.Resource, e.ID)
}

// ─────────────────────────────────────────────────────────────────────────────
// WRAPPING ERROR TYPE
//
// Wraps an underlying cause. Implement Unwrap() so errors.Is / errors.As
// can walk the chain.
// ─────────────────────────────────────────────────────────────────────────────

type OperationError struct {
	Op    string
	Cause error
}

func (e *OperationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("operation %q failed: %v", e.Op, e.Cause)
	}
	return fmt.Sprintf("operation %q failed", e.Op)
}

func (e *OperationError) Unwrap() error { return e.Cause }

// ─────────────────────────────────────────────────────────────────────────────
// CUSTOM Is() METHOD
//
// Allows errors.Is to match on field values rather than pointer identity.
// Useful when errors carry a code or category that identifies the "kind".
// ─────────────────────────────────────────────────────────────────────────────

type APIError struct {
	Code    int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.Code, e.Message)
}

// Is reports whether this error matches target — matches if target is an *APIError
// with the same Code (ignores Message, making code the identity).
func (e *APIError) Is(target error) bool {
	t, ok := target.(*APIError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// Package-level "sentinel" API errors — compared by code, not pointer.
var (
	ErrUnauthorised = &APIError{Code: 401, Message: "unauthorised"}
	ErrForbidden    = &APIError{Code: 403, Message: "forbidden"}
	ErrRateLimit    = &APIError{Code: 429, Message: "rate limit exceeded"}
)

// ─────────────────────────────────────────────────────────────────────────────
// CUSTOM As() METHOD (rare, but sometimes needed)
//
// Allows errors.As to cast to a different type from the chain.
// ─────────────────────────────────────────────────────────────────────────────

type TemporaryError struct {
	Underlying error
	RetryAfter time.Duration
}

func (e *TemporaryError) Error() string {
	return fmt.Sprintf("temporary error (retry after %s): %v", e.RetryAfter, e.Underlying)
}

func (e *TemporaryError) Unwrap() error { return e.Underlying }

// Temporary is a helper sentinel for use with errors.Is.
var ErrTemporary = errors.New("temporary error")

func (e *TemporaryError) Is(target error) bool { return target == ErrTemporary }

// ─────────────────────────────────────────────────────────────────────────────
// TYPED NIL TRAP REMINDER
//
// A non-nil interface wrapping a nil pointer is non-nil.
// Return bare nil, not a typed nil pointer.
// ─────────────────────────────────────────────────────────────────────────────

func buggyFind(fail bool) error {
	var err *NotFoundError // typed nil
	if fail {
		err = &NotFoundError{Resource: "user", ID: 42}
	}
	return err // WRONG: if !fail, returns non-nil interface wrapping nil *NotFoundError
}

func correctFind(fail bool) error {
	if fail {
		return &NotFoundError{Resource: "user", ID: 42}
	}
	return nil // correct: bare nil interface
}

func main() {
	fmt.Println("=== Simple struct error ===")
	err := &NotFoundError{Resource: "product", ID: "sku-123"}
	fmt.Println("  error:", err)
	var nfe *NotFoundError
	if errors.As(err, &nfe) {
		fmt.Printf("  resource=%s  id=%v\n", nfe.Resource, nfe.ID)
	}

	fmt.Println()
	fmt.Println("=== Wrapping error with Unwrap() ===")
	cause := &NotFoundError{Resource: "order", ID: 99}
	opErr := &OperationError{Op: "checkout", Cause: cause}
	wrapped := fmt.Errorf("handler: %w", opErr)
	fmt.Println("  error:", wrapped)
	fmt.Println("  errors.As(*NotFoundError):", errors.As(wrapped, &nfe))
	fmt.Printf("  extracted: resource=%s id=%v\n", nfe.Resource, nfe.ID)

	fmt.Println()
	fmt.Println("=== Custom Is() — match by code ===")
	// Wrap an API error through multiple layers.
	apiErr := &APIError{Code: 401, Message: "token expired"}
	deepWrapped := fmt.Errorf("request: %w", fmt.Errorf("auth: %w", apiErr))
	fmt.Println("  error:", deepWrapped)
	fmt.Println("  Is ErrUnauthorised (code 401):", errors.Is(deepWrapped, ErrUnauthorised))
	fmt.Println("  Is ErrForbidden (code 403):   ", errors.Is(deepWrapped, ErrForbidden))

	fmt.Println()
	fmt.Println("=== TemporaryError with Is() + Unwrap() ===")
	netErr := errors.New("connection reset by peer")
	tempErr := &TemporaryError{Underlying: netErr, RetryAfter: 5 * time.Second}
	wrapped2 := fmt.Errorf("fetchData: %w", tempErr)

	fmt.Println("  error:", wrapped2)
	fmt.Println("  Is ErrTemporary:", errors.Is(wrapped2, ErrTemporary))
	fmt.Println("  Underlying errors.Is(netErr):", errors.Is(wrapped2, netErr))

	var te *TemporaryError
	if errors.As(wrapped2, &te) {
		fmt.Printf("  retry after: %s\n", te.RetryAfter)
	}

	fmt.Println()
	fmt.Println("=== Typed nil trap ===")
	buggy := buggyFind(false)
	correct := correctFind(false)
	fmt.Printf("  buggy == nil:   %v  (should be true, but is false!)\n", buggy == nil)
	fmt.Printf("  correct == nil: %v  (correct)\n", correct == nil)
}
