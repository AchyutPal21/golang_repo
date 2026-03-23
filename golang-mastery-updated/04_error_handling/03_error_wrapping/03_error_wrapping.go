// 03_error_wrapping.go
//
// ERROR WRAPPING AND THE ERROR CHAIN
// ===================================
// When an error occurs deep in a call stack, the caller that first detects it
// knows the low-level detail ("permission denied", "connection reset"). But the
// caller two levels up needs CONTEXT: "what were we trying to do when this
// happened?" ("failed to load user profile").
//
// Error wrapping is Go's answer: each layer adds its own context and passes the
// original error along INSIDE the new error. The result is an error CHAIN
// (also called an error tree in Go 1.20+).
//
// THE %w VERB
// -----------
// fmt.Errorf("...: %w", err) wraps err. The returned error:
//   - Has an Error() string that includes the original message (via %w's formatting).
//   - Has an Unwrap() error method that returns the original err.
//
// CONTRAST WITH %v
// ----------------
// fmt.Errorf("...: %v", err) embeds the original message as a STRING ONLY.
// The original error is lost — errors.Is / errors.As cannot find it.
// Use %v when you intentionally want to sever the chain.
//
// ERRORS.IS — matching by identity in the chain
// -----------------------------------------------
// errors.Is(err, target) returns true if ANY error in the chain is
// identical to target (by value or via an Is(error) bool method).
// It traverses the chain via Unwrap() until it finds a match or nil.
//
// ERRORS.AS — extracting a typed error from the chain
// ----------------------------------------------------
// errors.As(err, &target) traverses the chain and, for the first error
// assignable to target's type, sets *target and returns true.
// This replaces direct type assertion (which cannot traverse the chain).
//
// WHY ADD CONTEXT AT EACH LAYER?
// --------------------------------
// Each layer should add:
//   - What it was trying to do ("loadUserProfile")
//   - Any relevant identifier ("userID=42")
//   - Leave the original error accessible for programmatic inspection
//
// Convention for error message format: "verb noun: reason"
//   e.g., "load user profile: query users: connection refused"
// This reads naturally as a chain of failed operations.

package main

import (
	"errors"
	"fmt"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 1: A typed error to use in the chain
// ─────────────────────────────────────────────────────────────────────────────

// DBError represents a database-level failure.
type DBError struct {
	Op      string
	Message string
}

func (e *DBError) Error() string {
	return fmt.Sprintf("db[%s]: %s", e.Op, e.Message)
}

// NetworkError is a low-level network failure.
type NetworkError struct {
	Host    string
	Timeout bool
}

func (e *NetworkError) Error() string {
	if e.Timeout {
		return fmt.Sprintf("network: timeout connecting to %s", e.Host)
	}
	return fmt.Sprintf("network: connection refused to %s", e.Host)
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 2: A multi-layer call stack — building the chain
// ─────────────────────────────────────────────────────────────────────────────
// We simulate: handleRequest → loadUserProfile → queryDatabase → connectDB
// Each layer wraps with context using %w.

// connectDB — deepest layer, produces the root error.
func connectDB(host string) error {
	if host == "bad-host" {
		// The root cause: a raw typed error with no wrapping yet.
		return &NetworkError{Host: host, Timeout: true}
	}
	return nil
}

// queryDatabase — calls connectDB, wraps its error with DB-level context.
func queryDatabase(host, query string) (string, error) {
	if err := connectDB(host); err != nil {
		// %w wraps err inside a new error. The Error() string will be:
		//   "db query users: network: timeout connecting to bad-host"
		// Unwrap() on the result returns the *NetworkError.
		return "", fmt.Errorf("db query %s: %w", query, err)
	}
	if query == "SELECT * FROM deleted_table" {
		return "", &DBError{Op: "SELECT", Message: "table does not exist"}
	}
	return `[{"id":1,"name":"Alice"}]`, nil
}

// loadUserProfile — calls queryDatabase, wraps its error with service context.
func loadUserProfile(userID int) (string, error) {
	host := "db.internal"
	if userID == 0 {
		host = "bad-host" // triggers the network error path
	}

	data, err := queryDatabase(host, "users")
	if err != nil {
		// Adding the userID is crucial: without it, the error message would
		// not tell us WHICH user caused the failure.
		return "", fmt.Errorf("load user profile userID=%d: %w", userID, err)
	}
	return data, nil
}

// handleRequest — top level, wraps with HTTP request context.
func handleRequest(requestID string, userID int) (string, error) {
	profile, err := loadUserProfile(userID)
	if err != nil {
		return "", fmt.Errorf("handle request %s: %w", requestID, err)
	}
	return profile, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 3: Manual chain traversal via Unwrap()
// ─────────────────────────────────────────────────────────────────────────────
// errors.Is and errors.As use Unwrap() internally. Here we do it manually
// to make the mechanics visible.

func printChain(err error) {
	fmt.Println("  Error chain (outermost → innermost):")
	for i := 0; err != nil; i++ {
		fmt.Printf("    [%d] %T: %v\n", i, err, err)
		err = errors.Unwrap(err) // nil if the error does not implement Unwrap()
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 4: errors.Is — sentinel matching in the chain
// ─────────────────────────────────────────────────────────────────────────────

// Define sentinel errors to use as targets for errors.Is.
var (
	ErrNotFound   = errors.New("not found")
	ErrPermission = errors.New("permission denied")
)

// fetchRecord wraps ErrNotFound so callers can detect it via errors.Is.
func fetchRecord(id int) error {
	if id == 0 {
		// Wrap with %w so errors.Is(err, ErrNotFound) returns true even
		// though the returned error is a different value than ErrNotFound.
		return fmt.Errorf("fetchRecord id=%d: %w", id, ErrNotFound)
	}
	if id < 0 {
		return fmt.Errorf("fetchRecord id=%d: %w", id, ErrPermission)
	}
	return nil
}

// processRecord calls fetchRecord and wraps again — two layers of wrapping.
func processRecord(id int) error {
	if err := fetchRecord(id); err != nil {
		return fmt.Errorf("processRecord: %w", err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 5: errors.As — typed error extraction from the chain
// ─────────────────────────────────────────────────────────────────────────────

// findNetworkError uses errors.As to extract *NetworkError even when it is
// buried two levels deep in the chain.
func findNetworkError(err error) (*NetworkError, bool) {
	var ne *NetworkError
	if errors.As(err, &ne) {
		return ne, true
	}
	return nil, false
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 6: Custom Unwrap — adding Unwrap to your own error types
// ─────────────────────────────────────────────────────────────────────────────

// ServiceError is a custom error that wraps another error.
// To participate in the error chain, it must implement Unwrap().
type ServiceError struct {
	Layer   string
	Message string
	Cause   error
	At      time.Time
}

func (e *ServiceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Layer, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Layer, e.Message)
}

// Unwrap is what makes errors.Is / errors.As able to peer through ServiceError.
// Without this method, the chain would stop here.
func (e *ServiceError) Unwrap() error {
	return e.Cause
}

func newServiceError(layer, message string, cause error) *ServiceError {
	return &ServiceError{Layer: layer, Message: message, Cause: cause, At: time.Now()}
}

// buildServiceChain constructs a three-deep ServiceError chain manually.
func buildServiceChain() error {
	root := &DBError{Op: "INSERT", Message: "unique constraint violated"}
	repo := newServiceError("repository", "create user failed", root)
	svc := newServiceError("service", "register user failed", repo)
	return svc
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== 03: Error Wrapping ===")
	fmt.Println()

	// ── 1. Build and print the full error chain ───────────────────────────────
	fmt.Println("── multi-layer call stack (handleRequest) ──")
	_, err := handleRequest("req-abc", 0) // 0 triggers network error
	if err != nil {
		fmt.Println("  Top-level error:")
		fmt.Printf("    %v\n\n", err)
		printChain(err)
	}
	fmt.Println()

	// ── 2. errors.Is traversing the chain ────────────────────────────────────
	fmt.Println("── errors.Is ──")

	ids := []int{1, 0, -1}
	for _, id := range ids {
		err := processRecord(id) // may be wrapped twice
		if err != nil {
			notFound := errors.Is(err, ErrNotFound)
			permDenied := errors.Is(err, ErrPermission)
			fmt.Printf("  id=%d: err=%q\n", id, err)
			fmt.Printf("    errors.Is(ErrNotFound)=%v  errors.Is(ErrPermission)=%v\n",
				notFound, permDenied)
		} else {
			fmt.Printf("  id=%d: success\n", id)
		}
	}
	fmt.Println()

	// ── 3. errors.As extracting *NetworkError from deep in the chain ──────────
	fmt.Println("── errors.As ──")

	_, chainErr := handleRequest("req-xyz", 0)
	if ne, ok := findNetworkError(chainErr); ok {
		fmt.Printf("  Found *NetworkError deep in chain:\n")
		fmt.Printf("    Host=%q  Timeout=%v\n", ne.Host, ne.Timeout)
	}

	// errors.As with *DBError — try on the service chain
	svcErr := buildServiceChain()
	var dbe *DBError
	if errors.As(svcErr, &dbe) {
		fmt.Printf("  Found *DBError in ServiceError chain:\n")
		fmt.Printf("    Op=%q  Message=%q\n", dbe.Op, dbe.Message)
	}
	fmt.Println()

	// ── 4. Manual chain for ServiceError ─────────────────────────────────────
	fmt.Println("── manual Unwrap chain for ServiceError ──")
	printChain(buildServiceChain())
	fmt.Println()

	// ── 5. %w vs %v — the difference ─────────────────────────────────────────
	fmt.Println("── %w vs %v ──")

	root := ErrNotFound

	// %w: wraps — errors.Is can find the original
	wrapped := fmt.Errorf("outer: %w", root)
	// %v: embeds string — chain is severed
	embedded := fmt.Errorf("outer: %v", root)

	fmt.Printf("  errors.Is(wrapped,  ErrNotFound) = %v  ← %w preserved chain\n",
		errors.Is(wrapped, root))
	fmt.Printf("  errors.Is(embedded, ErrNotFound) = %v  ← %%v severed chain\n",
		errors.Is(embedded, root))
	fmt.Printf("  errors.Unwrap(wrapped)  = %v\n", errors.Unwrap(wrapped))
	fmt.Printf("  errors.Unwrap(embedded) = %v  (nil: no Unwrap method)\n",
		errors.Unwrap(embedded))
	fmt.Println()

	fmt.Println("Key takeaways:")
	fmt.Println("  1. %w wraps: preserves the error in the chain; %v severs it")
	fmt.Println("  2. errors.Is traverses the chain looking for identity match")
	fmt.Println("  3. errors.As traverses the chain looking for type match")
	fmt.Println("  4. Custom types join the chain by implementing Unwrap() error")
	fmt.Println("  5. Add context at each layer: 'operation identifier: %w'")
	fmt.Println("  6. The full error string reads like a stack trace of operations")
}
