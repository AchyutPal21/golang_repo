// FILE: book/part3_designing_software/chapter37_custom_error_types/examples/02_error_interfaces/main.go
// CHAPTER: 37 — Custom Error Types
// TOPIC: Error interfaces (Temporary, Timeout, Retryable), error trees,
//        and a real-world domain error taxonomy.
//
// Run (from the chapter folder):
//   go run ./examples/02_error_interfaces

package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// ERROR BEHAVIOUR INTERFACES
//
// Instead of checking concrete types, check for behaviour.
// Any error that implements Temporary() bool is a temporary error.
// This is the net.Error pattern from the standard library.
// ─────────────────────────────────────────────────────────────────────────────

type Retryable interface {
	Retryable() bool
	RetryAfter() time.Duration
}

type Categorised interface {
	Category() string // "client" | "server" | "network"
}

// NetworkError — temporary, retryable.
type NetworkError struct {
	Op      string
	Reason  string
	Backoff time.Duration
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error during %s: %s", e.Op, e.Reason)
}
func (e *NetworkError) Retryable() bool           { return true }
func (e *NetworkError) RetryAfter() time.Duration { return e.Backoff }
func (e *NetworkError) Category() string          { return "network" }

// RateLimitError — retryable after a fixed window.
type RateLimitError struct {
	Limit  int
	Window time.Duration
	After  time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit: %d requests per %s (retry after %s)",
		e.Limit, e.Window, e.After)
}
func (e *RateLimitError) Retryable() bool           { return true }
func (e *RateLimitError) RetryAfter() time.Duration { return e.After }
func (e *RateLimitError) Category() string          { return "client" }

// AuthError — not retryable.
type AuthError struct{ Message string }

func (e *AuthError) Error() string    { return "auth: " + e.Message }
func (e *AuthError) Category() string { return "client" }

// DatabaseError — not retryable; server-side.
type DatabaseError struct {
	Op    string
	Cause error
}

func (e *DatabaseError) Error() string {
	return fmt.Sprintf("database error during %s: %v", e.Op, e.Cause)
}
func (e *DatabaseError) Unwrap() error   { return e.Cause }
func (e *DatabaseError) Category() string { return "server" }

// ─── Helper: check retryability generically ───────────────────────────────────

func shouldRetry(err error) (bool, time.Duration) {
	var r Retryable
	if errors.As(err, &r) && r.Retryable() {
		return true, r.RetryAfter()
	}
	return false, 0
}

func category(err error) string {
	var c Categorised
	if errors.As(err, &c) {
		return c.Category()
	}
	return "unknown"
}

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN ERROR TAXONOMY — hierarchical errors for an e-commerce API
// ─────────────────────────────────────────────────────────────────────────────

// DomainError is the base type.
type DomainError struct {
	Code    string
	Message string
	Cause   error
}

func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error { return e.Cause }

func (e *DomainError) Is(target error) bool {
	t, ok := target.(*DomainError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// Constructor helpers.
func newDomainErr(code, msg string, cause error) *DomainError {
	return &DomainError{Code: code, Message: msg, Cause: cause}
}

// Sentinel domain errors — matched by code via custom Is().
var (
	ErrOrderNotFound    = &DomainError{Code: "ORDER_NOT_FOUND"}
	ErrProductSoldOut   = &DomainError{Code: "PRODUCT_SOLD_OUT"}
	ErrPaymentDeclined  = &DomainError{Code: "PAYMENT_DECLINED"}
	ErrInvalidCoupon    = &DomainError{Code: "INVALID_COUPON"}
)

// placeOrder simulates a service that produces domain errors.
func placeOrder(productID string, qty int, coupon string) error {
	if productID == "SOLD_OUT" {
		return newDomainErr("PRODUCT_SOLD_OUT",
			fmt.Sprintf("product %s is sold out", productID), nil)
	}
	if productID == "MISSING" {
		dbErr := fmt.Errorf("SELECT failed: no rows")
		return newDomainErr("ORDER_NOT_FOUND",
			fmt.Sprintf("product %s not found", productID), dbErr)
	}
	if coupon == "EXPIRED" {
		return newDomainErr("INVALID_COUPON", "coupon EXPIRED has expired", nil)
	}
	if qty > 100 {
		return newDomainErr("PAYMENT_DECLINED",
			fmt.Sprintf("order total exceeds limit for qty %d", qty), nil)
	}
	fmt.Printf("  order placed: product=%s qty=%d\n", productID, qty)
	return nil
}

// HTTP handler — classifies domain errors into HTTP status codes.
func handleOrder(productID string, qty int, coupon string) (int, string) {
	err := placeOrder(productID, qty, coupon)
	if err == nil {
		return 200, `{"status":"ok"}`
	}

	switch {
	case errors.Is(err, ErrOrderNotFound):
		return 404, `{"error":"product not found"}`
	case errors.Is(err, ErrProductSoldOut):
		return 409, `{"error":"product sold out"}`
	case errors.Is(err, ErrPaymentDeclined):
		return 402, `{"error":"payment declined"}`
	case errors.Is(err, ErrInvalidCoupon):
		return 422, `{"error":"invalid coupon"}`
	default:
		fmt.Println("  [LOG] unhandled error:", err)
		return 500, `{"error":"internal error"}`
	}
}

func main() {
	fmt.Println("=== Error behaviour interfaces ===")
	errs := []error{
		&NetworkError{Op: "dial", Reason: "connection refused", Backoff: 2 * time.Second},
		&RateLimitError{Limit: 100, Window: time.Minute, After: 30 * time.Second},
		&AuthError{Message: "token expired"},
		&DatabaseError{Op: "INSERT", Cause: fmt.Errorf("deadlock detected")},
	}

	for _, err := range errs {
		retry, after := shouldRetry(err)
		cat := category(err)
		fmt.Printf("  %-55s  category=%-8s  retry=%-5v  after=%s\n",
			err.Error(), cat, retry, after)
	}

	fmt.Println()
	fmt.Println("=== Domain error taxonomy ===")
	cases := []struct {
		product, coupon string
		qty             int
	}{
		{"WIDGET", "", 2},
		{"SOLD_OUT", "", 1},
		{"MISSING", "", 1},
		{"GADGET", "EXPIRED", 1},
		{"EXPENSIVE", "", 150},
	}

	for _, c := range cases {
		status, body := handleOrder(c.product, c.qty, c.coupon)
		label := fmt.Sprintf("product=%-10s qty=%3d coupon=%-8s",
			c.product, c.qty, c.coupon)
		fmt.Printf("  %-45s  → HTTP %d  %s\n", label, status, body)
	}

	fmt.Println()
	fmt.Println("=== errors.Is through wrapping chain ===")
	err := fmt.Errorf("service: %w",
		newDomainErr("PRODUCT_SOLD_OUT", "widget-a sold out", nil))
	fmt.Println("  Is ErrProductSoldOut:", errors.Is(err, ErrProductSoldOut))
	fmt.Println("  Is ErrOrderNotFound: ", errors.Is(err, ErrOrderNotFound))

	var de *DomainError
	if errors.As(err, &de) {
		fmt.Printf("  extracted: code=%s msg=%s\n", de.Code, de.Message)
	}

	fmt.Println()
	fmt.Println("=== Retryable wrapped inside domain error ===")
	netErr := &NetworkError{Op: "charge", Reason: "timeout", Backoff: 3 * time.Second}
	payErr := newDomainErr("PAYMENT_DECLINED", "payment gateway unreachable", netErr)
	wrapped := fmt.Errorf("checkout: %w", payErr)

	retry, after := shouldRetry(wrapped)
	fmt.Printf("  retry=%v  after=%s\n", retry, after)

	// Separate string just to avoid a very long single format string
	parts := []string{"Is ErrPaymentDeclined:", fmt.Sprintf("%v", errors.Is(wrapped, ErrPaymentDeclined))}
	fmt.Println(" ", strings.Join(parts, " "))
}
