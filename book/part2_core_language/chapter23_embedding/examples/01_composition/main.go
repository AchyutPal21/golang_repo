// FILE: book/part2_core_language/chapter23_embedding/examples/01_composition/main.go
// CHAPTER: 23 — Embedding and Composition
// TOPIC: Field promotion, method promotion, diamond problem resolution,
//        embedding to satisfy interfaces, mixin pattern.
//
// Run (from the chapter folder):
//   go run ./examples/01_composition

package main

import (
	"fmt"
	"time"
)

// --- Mixin: reusable behaviour via embedding ---

type Timestamps struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (t *Timestamps) Touch() {
	now := time.Now()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
}

func (t Timestamps) Age() time.Duration {
	return time.Since(t.CreatedAt)
}

// Any model can gain audit timestamps by embedding.
type User struct {
	Timestamps
	ID   int
	Name string
}

type Post struct {
	Timestamps
	ID    int
	Title string
}

// --- Embedding to satisfy interfaces ---

type Logger struct{ prefix string }

func (l *Logger) Log(msg string) {
	fmt.Printf("[%s] %s\n", l.prefix, msg)
}

// Auditor wraps Logger and adds audit-specific behaviour.
type Auditor struct {
	*Logger
	userID int
}

func (a *Auditor) Audit(action string) {
	a.Log(fmt.Sprintf("user=%d action=%s", a.userID, action))
}

// --- Diamond resolution ---

type A struct{ Val int }
func (a A) Hello() string { return fmt.Sprintf("A{%d}", a.Val) }

type B struct{ A }
type C struct{ A }

// D embeds both B and C — both have a promoted Hello() from A.
// Calling d.Hello() is ambiguous: must use d.B.Hello() or d.C.Hello().
type D struct {
	B
	C
}

// D.Hello resolves the diamond by explicitly delegating.
func (d D) Hello() string {
	return fmt.Sprintf("D[B=%s, C=%s]", d.B.Hello(), d.C.Hello())
}

// --- Interface promotion ---

type ReadWriter interface {
	Read() string
	Write(s string)
}

type Buffer struct {
	data string
}

func (b *Buffer) Read() string    { return b.data }
func (b *Buffer) Write(s string)  { b.data += s }

// Service embeds *Buffer, so it also satisfies ReadWriter.
type Service struct {
	*Buffer
	name string
}

func NewService(name string) *Service {
	return &Service{Buffer: &Buffer{}, name: name}
}

func processRW(rw ReadWriter) {
	rw.Write("hello ")
	rw.Write("world")
	fmt.Println("buffer:", rw.Read())
}

func main() {
	// --- timestamps mixin ---
	u := &User{ID: 1, Name: "Alice"}
	u.Touch()
	fmt.Printf("user: %s created=%v\n", u.Name, !u.CreatedAt.IsZero())

	p := &Post{ID: 1, Title: "Hello Go"}
	p.Touch()
	fmt.Printf("post: %s age=%v\n", p.Title, p.Age() < time.Second)

	fmt.Println()

	// --- auditor with embedded *Logger ---
	a := &Auditor{
		Logger: &Logger{prefix: "AUDIT"},
		userID: 42,
	}
	a.Log("direct log")   // promoted from *Logger
	a.Audit("login")

	fmt.Println()

	// --- diamond resolution ---
	d := D{B: B{A: A{Val: 10}}, C: C{A: A{Val: 20}}}
	fmt.Println(d.Hello())
	// d.Val would be ambiguous — must use d.B.Val or d.C.Val
	fmt.Println("B.Val:", d.B.Val, "C.Val:", d.C.Val)

	fmt.Println()

	// --- interface promotion ---
	svc := NewService("myservice")
	processRW(svc) // Service satisfies ReadWriter through *Buffer
}
