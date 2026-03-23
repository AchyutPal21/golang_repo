// FILE: 03_structs_methods_interfaces/09_interface_design.go
// TOPIC: Interface Design Principles — small interfaces, when NOT to use them
//
// Run: go run 03_structs_methods_interfaces/09_interface_design.go

package main

import (
	"fmt"
	"strings"
)

// ── PRINCIPLE 1: Small interfaces ────────────────────────────────────────────
// The Go proverb: "The bigger the interface, the weaker the abstraction."
// The most powerful interfaces in Go's stdlib have 1-2 methods:
//   io.Reader  { Read([]byte) (int, error) }
//   io.Writer  { Write([]byte) (int, error) }
//   fmt.Stringer { String() string }
//   error       { Error() string }
//
// A 10-method interface forces every implementor to implement ALL 10 methods.
// A 1-method interface can be satisfied by many things.

// Good: small, focused interface
type Reader interface {
	Read() (string, error)
}

type Writer interface {
	Write(s string) error
}

// Compose small interfaces when you need both:
type ReadWriter interface {
	Reader
	Writer
}

// ── PRINCIPLE 2: Accept interfaces, return concrete types ────────────────────
// Function parameters: use interfaces → caller can pass anything that satisfies it.
// Return values: return concrete types → caller gets all the methods, not just the interface.

type StringReader struct {
	data string
	pos  int
}

func (r *StringReader) Read() (string, error) {
	if r.pos >= len(r.data) {
		return "", fmt.Errorf("EOF")
	}
	word := strings.Fields(r.data[r.pos:])[0]
	r.pos += len(word) + 1
	return word, nil
}

// Takes an interface (flexible), returns concrete type (rich):
func NewStringReader(s string) *StringReader {
	return &StringReader{data: s}
}

// This function accepts the interface — works with any Reader:
func readAll(r Reader) []string {
	var words []string
	for {
		w, err := r.Read()
		if err != nil {
			break
		}
		words = append(words, w)
	}
	return words
}

// ── PRINCIPLE 3: Don't create interfaces prematurely ─────────────────────────
// Create an interface when you have ≥2 implementations OR need testability.
// One implementation → just use the concrete type.

// BAD: interface with one implementation (interface pollution)
// type UserServiceInterface interface { GetUser(id int) User }
// type UserService struct{}
// func (s UserService) GetUser(id int) User { ... }
//
// GOOD: just use the concrete type directly.
// Only extract to interface when a second impl appears (e.g., mock for testing).

// ── PRINCIPLE 4: Interface for testing (dependency injection) ─────────────────

type EmailSender interface {
	Send(to, subject, body string) error
}

type UserNotifier struct {
	sender EmailSender
}

func NewUserNotifier(s EmailSender) *UserNotifier {
	return &UserNotifier{sender: s}
}

func (n *UserNotifier) NotifyWelcome(email string) error {
	return n.sender.Send(email, "Welcome!", "Thanks for signing up.")
}

// Real implementation:
type SMTPSender struct{ Host string }
func (s SMTPSender) Send(to, subject, body string) error {
	fmt.Printf("  [SMTP:%s] → %s | %s\n", s.Host, to, subject)
	return nil
}

// Test implementation (mock):
type MockSender struct{ Calls []string }
func (m *MockSender) Send(to, subject, body string) error {
	m.Calls = append(m.Calls, fmt.Sprintf("%s|%s", to, subject))
	return nil
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Interface Design Principles")
	fmt.Println("════════════════════════════════════════")

	fmt.Println("\n── Small interfaces + composition ──")
	sr := NewStringReader("hello world go")
	words := readAll(sr)
	fmt.Printf("  readAll result: %v\n", words)

	fmt.Println("\n── Accept interface → dependency injection ──")
	// Production:
	notifier := NewUserNotifier(SMTPSender{Host: "smtp.example.com"})
	_ = notifier.NotifyWelcome("alice@example.com")

	// Test (swap implementation without changing UserNotifier):
	mock := &MockSender{}
	testNotifier := NewUserNotifier(mock)
	_ = testNotifier.NotifyWelcome("bob@example.com")
	fmt.Printf("  Mock recorded calls: %v\n", mock.Calls)

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  Keep interfaces small (1-3 methods)")
	fmt.Println("  Accept interfaces, return concrete types")
	fmt.Println("  Don't create interfaces prematurely — wait for 2nd implementation")
	fmt.Println("  Interface = seam for testing via dependency injection")
	fmt.Println("  Compose interfaces from small ones (io.ReadWriter = Reader+Writer)")
}
