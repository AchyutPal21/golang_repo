// 06_stringer_error_interfaces.go
//
// THE STRINGER AND ERROR INTERFACES — two of Go's most important interfaces.
//
// Go's standard library defines many small interfaces that enable rich
// behavior through implicit satisfaction. The two most commonly implemented
// are fmt.Stringer and error.
//
// fmt.Stringer:
//   type Stringer interface {
//       String() string
//   }
//   Implemented by types that have a meaningful string representation.
//   The fmt package checks for this interface when printing with %v, %s, println.
//
// error:
//   type error interface {
//       Error() string
//   }
//   The simplest, most important interface in Go. Returned from any function
//   that can fail. Custom error types let you carry structured information.
//
// This file also previews io.Reader and io.Writer — the canonical examples
// of elegant, minimal interface design.

package main

import (
	"fmt"
	"io"
	"math"
	"strings"
)

// ─── 1. fmt.Stringer ──────────────────────────────────────────────────────────
//
// When you pass a value to fmt.Println, fmt.Printf %v, or fmt.Sprintf,
// the fmt package calls the String() method if it exists (via Stringer).
//
// Without Stringer: fmt prints the struct in default format: {field1 field2}
// With Stringer:    fmt calls String() and prints whatever you return.
//
// WHY implement Stringer:
//   - Clean log output
//   - Meaningful debug printing
//   - Consistent representation across the codebase

type Color struct {
	R, G, B uint8
}

// String implements fmt.Stringer.
// Now fmt.Println(c) prints "#RRGGBB" instead of "{R:255 G:128 B:0}"
func (c Color) String() string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

type Point3D struct {
	X, Y, Z float64
}

func (p Point3D) String() string {
	return fmt.Sprintf("(%.2f, %.2f, %.2f)", p.X, p.Y, p.Z)
}

// Card represents a playing card.
type Suit int

const (
	Spades Suit = iota
	Hearts
	Diamonds
	Clubs
)

func (s Suit) String() string {
	switch s {
	case Spades:
		return "♠"
	case Hearts:
		return "♥"
	case Diamonds:
		return "♦"
	case Clubs:
		return "♣"
	default:
		return "?"
	}
}

type Card struct {
	Value int // 1=Ace, 11=Jack, 12=Queen, 13=King
	Suit  Suit
}

func (c Card) String() string {
	valueName := fmt.Sprintf("%d", c.Value)
	switch c.Value {
	case 1:
		valueName = "A"
	case 11:
		valueName = "J"
	case 12:
		valueName = "Q"
	case 13:
		valueName = "K"
	}
	return valueName + c.Suit.String()
}

// ─── 2. The error Interface ───────────────────────────────────────────────────
//
// error is defined in the universe block (always available):
//   type error interface {
//       Error() string
//   }
//
// Any type with an Error() string method satisfies the error interface.
// This means you can carry rich context in errors, not just strings.

// Simple sentinel errors — returned as plain errors, carry no extra data.
// Use errors.New() or fmt.Errorf() for these in real code.

// Custom error type with structured fields.
// WHY: callers can type-assert to get the specific fields (code, operation, etc.)
// and make decisions based on them. A plain string error doesn't allow this.

// ValidationError carries the field name and the constraint that was violated.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field %q: %s", e.Field, e.Message)
}

// NetworkError carries operation, address, and underlying error.
type NetworkError struct {
	Operation string
	Address   string
	Err       error // wrapped underlying error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error during %s to %s: %v", e.Operation, e.Address, e.Err)
}

// Unwrap implements the errors.Unwrapper interface, enabling errors.Is/As
// to traverse the error chain.
func (e *NetworkError) Unwrap() error {
	return e.Err
}

// DatabaseError carries a code for machine-readable classification.
type DatabaseError struct {
	Code    int
	Message string
	Query   string
}

func (e *DatabaseError) Error() string {
	return fmt.Sprintf("DB error [code=%d]: %s (query: %.40s...)", e.Code, e.Message, e.Query)
}

func (e *DatabaseError) IsNotFound() bool  { return e.Code == 404 }
func (e *DatabaseError) IsConflict() bool  { return e.Code == 409 }
func (e *DatabaseError) IsInternal() bool  { return e.Code >= 500 }

// ─── 3. Implementing io.Reader ────────────────────────────────────────────────
//
// io.Reader is defined in the standard library:
//   type Reader interface {
//       Read(p []byte) (n int, err error)
//   }
//
// The contract:
//   - Fill p with up to len(p) bytes of data.
//   - Return the number of bytes read (n) and any error.
//   - When no more data is available, return 0, io.EOF.
//   - n > 0 can be returned together with err != nil (e.g., the last block).
//
// io.Writer:
//   type Writer interface {
//       Write(p []byte) (n int, err error)
//   }
//
// WHY these interfaces matter:
//   They are the "lingua franca" of data streaming in Go.
//   Functions that accept io.Reader work with files, HTTP bodies, strings,
//   network connections, buffers, gzip readers, etc. — all interchangeably.

// AlphaReader is a simple io.Reader that strips non-alphabetic characters.
// This shows how easy it is to create custom readers.
type AlphaReader struct {
	source string
	pos    int
}

func NewAlphaReader(s string) *AlphaReader {
	return &AlphaReader{source: s}
}

// Read implements io.Reader.
// Copies only [a-zA-Z] characters from the source string into p.
func (r *AlphaReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.source) {
		return 0, io.EOF // signal: no more data
	}

	n := 0
	for n < len(p) && r.pos < len(r.source) {
		ch := r.source[r.pos]
		r.pos++
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			p[n] = ch
			n++
		}
		// Skip non-alpha characters (digits, spaces, punctuation)
	}

	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

// RepeatReader reads the same string n times before returning EOF.
// Demonstrates a stateful reader.
type RepeatReader struct {
	content    string
	timesLeft  int
	posInLine  int
}

func NewRepeatReader(content string, times int) *RepeatReader {
	return &RepeatReader{content: content, timesLeft: times}
}

func (r *RepeatReader) Read(p []byte) (int, error) {
	if r.timesLeft <= 0 {
		return 0, io.EOF
	}

	n := 0
	for n < len(p) {
		if r.posInLine >= len(r.content) {
			r.timesLeft--
			r.posInLine = 0
			if r.timesLeft <= 0 {
				break
			}
		}
		p[n] = r.content[r.posInLine]
		r.posInLine++
		n++
	}

	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

// ─── Helper Functions ─────────────────────────────────────────────────────────

func validateAge(age int) error {
	if age < 0 {
		return &ValidationError{Field: "age", Message: "must be non-negative"}
	}
	if age > 150 {
		return &ValidationError{Field: "age", Message: "implausibly large (> 150)"}
	}
	return nil
}

func validateEmail(email string) error {
	if !strings.Contains(email, "@") {
		return &ValidationError{Field: "email", Message: "must contain '@'"}
	}
	return nil
}

func connectToDB(host string) error {
	// Simulate a connection failure
	underlying := fmt.Errorf("connection refused")
	return &NetworkError{
		Operation: "connect",
		Address:   host,
		Err:       underlying,
	}
}

func queryDB(query string) error {
	return &DatabaseError{
		Code:    404,
		Message: "record not found",
		Query:   query,
	}
}

// ─── main ─────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("========================================")
	fmt.Println("  Stringer, error, io.Reader Interfaces")
	fmt.Println("========================================")

	// ── fmt.Stringer ─────────────────────────────────────────────────────────
	fmt.Println("\n── fmt.Stringer ─────────────────────────────────────")

	red := Color{R: 255, G: 0, B: 0}
	green := Color{R: 0, G: 255, B: 0}
	navyBlue := Color{R: 0, G: 0, B: 128}

	// fmt.Println calls String() automatically
	fmt.Println("Colors:", red, green, navyBlue)
	fmt.Printf("Red: %v\n", red)   // %v uses String()
	fmt.Printf("Red: %s\n", red)   // %s also uses String()

	pt := Point3D{X: 1.5, Y: math.Sqrt(2), Z: math.Pi}
	fmt.Println("Point:", pt)

	hand := []Card{
		{1, Spades},
		{13, Hearts},
		{11, Diamonds},
		{10, Clubs},
		{12, Spades},
	}
	fmt.Print("Hand: ")
	for i, card := range hand {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(card) // calls card.String()
	}
	fmt.Println()

	// ── Custom Error Types ───────────────────────────────────────────────────
	fmt.Println("\n── Custom Error Types ───────────────────────────────")

	// ValidationError
	ages := []int{25, -5, 200}
	for _, age := range ages {
		if err := validateAge(age); err != nil {
			// Type-assert to get structured info
			if ve, ok := err.(*ValidationError); ok {
				fmt.Printf("  Field: %q, Problem: %s\n", ve.Field, ve.Message)
			}
		} else {
			fmt.Printf("  Age %d is valid\n", age)
		}
	}

	emails := []string{"alice@example.com", "notanemail"}
	for _, email := range emails {
		if err := validateEmail(email); err != nil {
			fmt.Printf("  %v\n", err) // calls err.Error()
		} else {
			fmt.Printf("  Email %q is valid\n", email)
		}
	}

	// NetworkError
	fmt.Println("\n── NetworkError with Wrapping ───────────────────────")
	if err := connectToDB("db.prod.internal:5432"); err != nil {
		fmt.Printf("  Error: %v\n", err)

		// Type-assert to access network-specific fields
		if ne, ok := err.(*NetworkError); ok {
			fmt.Printf("  Operation: %s\n", ne.Operation)
			fmt.Printf("  Address:   %s\n", ne.Address)
			fmt.Printf("  Underlying: %v\n", ne.Err)
		}
	}

	// DatabaseError
	fmt.Println("\n── DatabaseError with Methods ───────────────────────")
	if err := queryDB("SELECT * FROM users WHERE id = 999"); err != nil {
		fmt.Printf("  Error: %v\n", err)

		if dbe, ok := err.(*DatabaseError); ok {
			fmt.Printf("  IsNotFound: %v\n", dbe.IsNotFound())
			fmt.Printf("  IsInternal: %v\n", dbe.IsInternal())
			// Code 404 is not found → caller can decide to return a 404 HTTP response
		}
	}

	// ── io.Reader ────────────────────────────────────────────────────────────
	fmt.Println("\n── io.Reader Implementation ─────────────────────────")

	// AlphaReader: strips non-alpha chars
	ar := NewAlphaReader("H3ll0, W0rld! Go is #1 in 2024!")
	buf := make([]byte, 32)
	n, err := ar.Read(buf)
	fmt.Printf("AlphaReader result: %q (n=%d, err=%v)\n", string(buf[:n]), n, err)

	// Read again to confirm EOF
	n2, err2 := ar.Read(buf)
	fmt.Printf("Second read: n=%d, err=%v\n", n2, err2) // 0, EOF

	// ── io.ReadAll — consuming a custom reader ────────────────────────────────
	fmt.Println("\n── io.ReadAll with Custom Readers ───────────────────")

	// Use io.ReadAll to slurp everything from a custom Reader
	ar2 := NewAlphaReader("abc123def456ghi")
	all, err := io.ReadAll(ar2)
	fmt.Printf("AlphaReader all: %q (err=%v)\n", string(all), err)

	// RepeatReader
	rr := NewRepeatReader("Go! ", 3)
	allRepeat, err := io.ReadAll(rr)
	fmt.Printf("RepeatReader all: %q (err=%v)\n", string(allRepeat), err)

	// ── strings.NewReader — the stdlib's io.Reader for strings ───────────────
	fmt.Println("\n── strings.NewReader (stdlib io.Reader for strings) ──")

	// Any function that accepts io.Reader works with strings.NewReader.
	// This is why accepting io.Reader is so powerful.
	strReader := strings.NewReader("Hello, io.Reader world!")
	data, _ := io.ReadAll(strReader)
	fmt.Printf("strings.NewReader: %q\n", string(data))

	// ── Stringer on Error Types ───────────────────────────────────────────────
	fmt.Println("\n── When to Implement Stringer ───────────────────────")
	fmt.Println(`
  Implement fmt.Stringer when:
    - The type has a natural string representation (Color, IP address, UUID)
    - You want clean log output without manual formatting
    - The default {Field:value ...} format is noisy or confusing
    - You're building a type for library consumers

  DON'T implement Stringer when:
    - The type is an internal implementation detail
    - The default format is perfectly clear
    - Performance is critical (String() allocates)

  Key: fmt.Printf("%v", x) checks for Stringer. "%+v" bypasses it.
  `)

	// ── Summary ──────────────────────────────────────────────────────────────
	fmt.Println("\n── io.Reader / io.Writer Design Wisdom ──────────────")
	fmt.Println(`
  io.Reader and io.Writer are the gold standard of Go interface design:
    - One method each: Read([]byte) or Write([]byte)
    - Compose into ReadWriter, ReadWriteCloser, etc.
    - Work with files, buffers, network conns, HTTP bodies, compression, etc.
    - Enable streaming: process data without loading it all into memory

  Any function that accepts io.Reader automatically works with:
    os.File, bytes.Buffer, strings.Reader, net.Conn,
    http.Response.Body, gzip.Reader, cipher.StreamReader, ...

  This is the power of accepting interfaces: write once, work everywhere.
  `)
}
