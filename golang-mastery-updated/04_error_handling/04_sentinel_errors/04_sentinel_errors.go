// 04_sentinel_errors.go
//
// SENTINEL ERRORS
// ===============
// A sentinel error is a pre-declared package-level error variable that serves
// as a well-known signal. The name "sentinel" comes from the idea of a guard
// standing watch — these errors are recognised by their IDENTITY, not by
// parsing their message.
//
// THE CANONICAL EXAMPLE — io.EOF
// --------------------------------
// var EOF = errors.New("EOF")
//
// Every reader in the standard library returns io.EOF to signal "there is
// no more data to read". Callers check:
//
//   if err == io.EOF { ... }    // old style (pre Go 1.13)
//   if errors.Is(err, io.EOF)   // correct style (works with wrapping)
//
// Nobody checks if err.Error() == "EOF" — that would be fragile (any package
// could produce a string "EOF" and be confused with io.EOF).
//
// WHY IDENTITY NOT MESSAGE?
// --------------------------
// errors.New("EOF") returns a *errorString at a specific memory address.
// Two calls return two different pointers. So:
//
//   io.EOF == io.EOF               // true  (same pointer)
//   io.EOF == errors.New("EOF")    // false (different pointer)
//
// This uniqueness is the whole point. It means you can export a variable,
// users compare with errors.Is, and there is zero chance of a collision
// with any other package's errors.
//
// NAMING CONVENTION
// -----------------
// By Go convention, sentinel errors are named with the Err prefix:
//   var ErrNotFound = errors.New("record not found")
//   var ErrTimeout  = errors.New("operation timed out")
//
// (Exception: io.EOF breaks this convention for historical reasons.)
//
// WHEN TO USE SENTINEL vs TYPED ERRORS
// --------------------------------------
// Use SENTINEL errors when:
//   - The error is a well-known, named condition (EOF, not-found, timeout).
//   - There is no extra structured data the caller needs to inspect.
//   - The error is part of a stable public API and callers compare with errors.Is.
//
// Use TYPED errors (custom struct) when:
//   - You need to carry structured fields (HTTP code, field name, etc.).
//   - You want callers to extract data with errors.As.
//
// You can combine both: a sentinel for identity + a typed wrapper that contains
// the sentinel so errors.Is still works (shown in Section 5 below).

package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 1: Defining sentinel errors
// ─────────────────────────────────────────────────────────────────────────────

// Package-level sentinel errors for an imaginary "store" package.
// They are unexported here (lower-case) for demo purposes; in a real package
// they would be exported so callers can reference them.
var (
	// ErrNotFound signals that the requested resource does not exist.
	// Callers use this to distinguish "not found" from "broken" —
	// they can return HTTP 404 instead of 500.
	ErrNotFound = errors.New("record not found")

	// ErrConflict signals that the operation would violate a uniqueness rule.
	ErrConflict = errors.New("record already exists")

	// ErrUnauthorised signals that the caller lacks permission.
	ErrUnauthorised = errors.New("unauthorised")

	// ErrStoreDown signals that the backing store is unavailable.
	ErrStoreDown = errors.New("store unavailable")
)

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 2: Functions returning sentinel errors
// ─────────────────────────────────────────────────────────────────────────────

// in-memory "database"
var store = map[int]string{
	1: "Alice",
	2: "Bob",
}

// getUser returns ErrNotFound if the user is absent.
// It wraps ErrNotFound with %w so errors.Is still works but the message
// also carries the ID — best of both worlds.
func getUser(id int) (string, error) {
	if id == -1 {
		// Simulate store being down — no wrapping, return sentinel directly.
		return "", ErrStoreDown
	}
	name, ok := store[id]
	if !ok {
		// Wrap with context. errors.Is(err, ErrNotFound) still returns true.
		return "", fmt.Errorf("getUser id=%d: %w", id, ErrNotFound)
	}
	return name, nil
}

// createUser returns ErrConflict if the user already exists.
func createUser(id int, name string) error {
	if _, exists := store[id]; exists {
		return fmt.Errorf("createUser id=%d: %w", id, ErrConflict)
	}
	store[id] = name
	return nil
}

// adminAction returns ErrUnauthorised for non-admin users.
func adminAction(role string) error {
	if role != "admin" {
		return fmt.Errorf("adminAction role=%q: %w", role, ErrUnauthorised)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 3: io.EOF — the canonical sentinel in action
// ─────────────────────────────────────────────────────────────────────────────

// readAll reads all lines from a strings.Reader, using io.EOF to detect end.
// This is how every scanner/reader loop in Go works.
func readAll(input string) []string {
	reader := strings.NewReader(input)
	var lines []string
	buf := make([]byte, 4) // tiny buffer to force multiple reads

	var result []byte
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				// io.EOF is NOT really an error; it is a signal.
				// We break cleanly without reporting it as a failure.
				break
			}
			// Any other error IS a real failure.
			fmt.Println("unexpected read error:", err)
			break
		}
	}
	lines = strings.Split(string(result), "\n")
	return lines
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 4: errors.Is — the right way to compare sentinels
// ─────────────────────────────────────────────────────────────────────────────
//
// Old style (pre-1.13): err == ErrNotFound
//   PROBLEM: fails if the error is wrapped with fmt.Errorf("%w", ErrNotFound)
//
// New style: errors.Is(err, ErrNotFound)
//   CORRECT: traverses the Unwrap() chain and finds ErrNotFound anywhere in it.
//
// ALWAYS use errors.Is for sentinel comparison.

func demonstrateIs() {
	// Direct sentinel — both == and errors.Is work
	direct := ErrNotFound
	fmt.Printf("  direct == ErrNotFound:        %v\n", direct == ErrNotFound)
	fmt.Printf("  errors.Is(direct):             %v\n", errors.Is(direct, ErrNotFound))

	// Wrapped sentinel — == fails, errors.Is works
	wrapped := fmt.Errorf("outer: %w", ErrNotFound)
	fmt.Printf("  wrapped == ErrNotFound:        %v  ← wrapping breaks ==\n", wrapped == ErrNotFound)
	fmt.Printf("  errors.Is(wrapped):            %v  ← errors.Is traverses chain\n", errors.Is(wrapped, ErrNotFound))

	// Double-wrapped — errors.Is still finds it
	doubleWrapped := fmt.Errorf("another layer: %w", wrapped)
	fmt.Printf("  doubleWrapped errors.Is:       %v  ← traverses multiple layers\n",
		errors.Is(doubleWrapped, ErrNotFound))
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 5: Combining sentinel + typed error
// ─────────────────────────────────────────────────────────────────────────────
// You can have the best of both worlds: a typed error that carries structured
// fields AND embeds a sentinel so errors.Is works.

// NotFoundError is a typed error that wraps ErrNotFound.
// Callers can use errors.Is(err, ErrNotFound) OR errors.As(err, &NotFoundError).
type NotFoundError struct {
	ResourceType string
	ID           int
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with id=%d: %v", e.ResourceType, e.ID, ErrNotFound)
}

// Unwrap returns ErrNotFound so errors.Is traversal finds it.
func (e *NotFoundError) Unwrap() error {
	return ErrNotFound
}

// findProduct uses the combined type.
func findProduct(id int) (string, error) {
	products := map[int]string{10: "Widget", 20: "Gadget"}
	p, ok := products[id]
	if !ok {
		return "", &NotFoundError{ResourceType: "product", ID: id}
	}
	return p, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION 6: The danger of string comparison — why we never do it
// ─────────────────────────────────────────────────────────────────────────────

// badCheck demonstrates the fragile anti-pattern of comparing error messages.
// NEVER do this in real code.
func badCheck(err error) bool {
	// Fragile: breaks if the message ever changes, if the error is wrapped,
	// or if another package happens to produce the same string.
	return err != nil && err.Error() == "record not found"
}

func goodCheck(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== 04: Sentinel Errors ===")
	fmt.Println()

	// ── 1. getUser — sentinel comparison ─────────────────────────────────────
	fmt.Println("── getUser ──")
	for _, id := range []int{1, 5, -1} {
		name, err := getUser(id)
		switch {
		case err == nil:
			fmt.Printf("  id=%d → %q\n", id, name)
		case errors.Is(err, ErrNotFound):
			fmt.Printf("  id=%d → NOT FOUND: %v\n", id, err)
		case errors.Is(err, ErrStoreDown):
			fmt.Printf("  id=%d → STORE DOWN: %v\n", id, err)
		default:
			fmt.Printf("  id=%d → unexpected error: %v\n", id, err)
		}
	}
	fmt.Println()

	// ── 2. createUser — ErrConflict ───────────────────────────────────────────
	fmt.Println("── createUser ──")
	for _, id := range []int{3, 1} { // 3 is new, 1 already exists
		err := createUser(id, "NewUser")
		if err == nil {
			fmt.Printf("  id=%d → created\n", id)
		} else if errors.Is(err, ErrConflict) {
			fmt.Printf("  id=%d → CONFLICT: %v\n", id, err)
		}
	}
	fmt.Println()

	// ── 3. adminAction — ErrUnauthorised ──────────────────────────────────────
	fmt.Println("── adminAction ──")
	for _, role := range []string{"admin", "user", "guest"} {
		err := adminAction(role)
		if err == nil {
			fmt.Printf("  role=%q → action allowed\n", role)
		} else if errors.Is(err, ErrUnauthorised) {
			fmt.Printf("  role=%q → DENIED: %v\n", role, err)
		}
	}
	fmt.Println()

	// ── 4. io.EOF example ────────────────────────────────────────────────────
	fmt.Println("── io.EOF (the canonical sentinel) ──")
	lines := readAll("hello\nworld\nfoo")
	fmt.Printf("  read %d segments from reader\n", len(lines))
	for i, l := range lines {
		fmt.Printf("    [%d] %q\n", i, l)
	}
	fmt.Println()

	// ── 5. errors.Is vs == ───────────────────────────────────────────────────
	fmt.Println("── errors.Is vs == ──")
	demonstrateIs()
	fmt.Println()

	// ── 6. Combined sentinel + typed (NotFoundError) ─────────────────────────
	fmt.Println("── NotFoundError (sentinel + typed) ──")
	for _, id := range []int{10, 99} {
		product, err := findProduct(id)
		if err == nil {
			fmt.Printf("  id=%d → %q\n", id, product)
		} else {
			// Both checks work:
			isNotFound := errors.Is(err, ErrNotFound)
			var nfe *NotFoundError
			asTyped := errors.As(err, &nfe)

			fmt.Printf("  id=%d → err=%q\n", id, err)
			fmt.Printf("    errors.Is(ErrNotFound)=%v  errors.As(*NotFoundError)=%v\n",
				isNotFound, asTyped)
			if asTyped {
				fmt.Printf("    typed fields: ResourceType=%q ID=%d\n",
					nfe.ResourceType, nfe.ID)
			}
		}
	}
	fmt.Println()

	// ── 7. String comparison anti-pattern ────────────────────────────────────
	fmt.Println("── string comparison: bad vs good ──")
	wrappedNF := fmt.Errorf("layer: %w", ErrNotFound)
	fmt.Printf("  badCheck(wrappedNF)  = %v  ← WRONG: string compare misses wrapped\n", badCheck(wrappedNF))
	fmt.Printf("  goodCheck(wrappedNF) = %v  ← CORRECT: errors.Is traverses chain\n", goodCheck(wrappedNF))
	fmt.Println()

	fmt.Println("Key takeaways:")
	fmt.Println("  1. Sentinel errors are package-level vars; identity = their pointer")
	fmt.Println("  2. Name with Err prefix: ErrNotFound, ErrTimeout, etc.")
	fmt.Println("  3. Always compare with errors.Is, never == or string matching")
	fmt.Println("  4. errors.Is traverses the Unwrap() chain — works with wrapped errors")
	fmt.Println("  5. io.EOF is the canonical example: a signal, not really an 'error'")
	fmt.Println("  6. Combine sentinel + typed: typed error's Unwrap returns the sentinel")
}
