// FILE: book/part2_core_language/chapter22_interfaces/examples/03_interface_patterns/main.go
// CHAPTER: 22 — Interfaces: Go's Killer Feature
// TOPIC: Accept interfaces, return structs; small interfaces; interface
//        for testing; io.Reader pipeline; error interface.
//
// Run (from the chapter folder):
//   go run ./examples/03_interface_patterns

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
)

// --- Accept interfaces, return structs ---
//
// Functions should accept interfaces (widest contract caller can satisfy)
// and return concrete types (richer API, no wrapper needed).

type Sizer interface {
	Size() int64
}

type File struct {
	name string
	data []byte
}

func (f *File) Size() int64 { return int64(len(f.data)) }
func (f *File) Name() string { return f.name }

// Takes Sizer (narrow) — caller can pass *File, *Buffer, *S3Object, anything.
func logSize(s Sizer) {
	fmt.Printf("size: %d bytes\n", s.Size())
}

// --- Small interfaces ---
//
// Prefer single-method interfaces. They compose naturally.

type Namer interface{ Name() string }
type Sizer2 interface{ Size() int64 }

// NamerSizer composes two single-method interfaces.
type NamerSizer interface {
	Namer
	Sizer2
}

func printInfo(ns NamerSizer) {
	fmt.Printf("%s: %d bytes\n", ns.Name(), ns.Size())
}

// --- Interface for testing ---

type EmailSender interface {
	Send(to, subject, body string) error
}

type UserService struct {
	email EmailSender
}

func (u *UserService) WelcomeUser(name, address string) error {
	return u.email.Send(address, "Welcome "+name, "Hello, "+name+"!")
}

// fakeEmailSender is a test double — implements EmailSender without SMTP.
type fakeEmailSender struct {
	sent []string
}

func (f *fakeEmailSender) Send(to, subject, body string) error {
	f.sent = append(f.sent, fmt.Sprintf("to=%s subj=%q", to, subject))
	return nil
}

// --- io.Reader pipeline ---

// countingReader wraps an io.Reader and counts bytes read.
type countingReader struct {
	r     io.Reader
	count int64
}

func (c *countingReader) Read(p []byte) (n int, err error) {
	n, err = c.r.Read(p)
	c.count += int64(n)
	return
}

// uppercaseReader wraps an io.Reader and uppercases bytes.
type uppercaseReader struct{ r io.Reader }

func (u *uppercaseReader) Read(p []byte) (n int, err error) {
	n, err = u.r.Read(p)
	for i := range p[:n] {
		if p[i] >= 'a' && p[i] <= 'z' {
			p[i] -= 32
		}
	}
	return
}

// --- sentinel errors and error wrapping ---

var ErrNotFound = errors.New("not found")
var ErrPermission = errors.New("permission denied")

type DBError struct {
	Op  string
	Err error
}

func (e *DBError) Error() string { return fmt.Sprintf("db.%s: %v", e.Op, e.Err) }
func (e *DBError) Unwrap() error { return e.Err }

func getUser(id int) error {
	if id == 0 {
		return &DBError{Op: "getUser", Err: ErrNotFound}
	}
	if id < 0 {
		return &DBError{Op: "getUser", Err: ErrPermission}
	}
	return nil
}

func main() {
	// --- accept interface ---
	f := &File{name: "report.pdf", data: make([]byte, 1024)}
	logSize(f) // accepts Sizer
	printInfo(f) // accepts NamerSizer (File satisfies both Namer and Sizer2)

	fmt.Println()

	// --- testing with interface ---
	fake := &fakeEmailSender{}
	svc := &UserService{email: fake}
	_ = svc.WelcomeUser("Alice", "alice@example.com")
	_ = svc.WelcomeUser("Bob", "bob@example.com")
	fmt.Println("emails sent:")
	for _, s := range fake.sent {
		fmt.Println(" ", s)
	}

	fmt.Println()

	// --- io.Reader pipeline ---
	src := strings.NewReader("hello, world! this is go.")
	cr := &countingReader{r: src}
	ur := &uppercaseReader{r: cr}

	var buf bytes.Buffer
	io.Copy(&buf, ur)
	fmt.Println("uppercased:", buf.String())
	fmt.Println("bytes read:", cr.count)

	fmt.Println()

	// --- error wrapping and unwrapping ---
	err := getUser(0)
	fmt.Println("err:", err)
	fmt.Println("is ErrNotFound:", errors.Is(err, ErrNotFound))

	var dbErr *DBError
	if errors.As(err, &dbErr) {
		fmt.Println("db op:", dbErr.Op)
	}

	err2 := getUser(-1)
	fmt.Println("is ErrPermission:", errors.Is(err2, ErrPermission))

	err3 := getUser(1)
	fmt.Println("success:", err3 == nil)
}
