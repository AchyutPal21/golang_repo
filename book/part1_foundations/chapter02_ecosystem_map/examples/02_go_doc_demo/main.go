// FILE: book/part1_foundations/chapter02_ecosystem_map/examples/02_go_doc_demo/main.go
// CHAPTER: 02 — A Map of the Go Ecosystem
// TOPIC: Doc comments are a contract — and they're the docs.
//
// Run (from the chapter folder):
//   go run ./examples/02_go_doc_demo
//
// To read this file's docs from the command line:
//   go doc -all ./examples/02_go_doc_demo
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   To demonstrate the Go convention that doc comments are *the*
//   documentation surface, read by `go doc` and pkg.go.dev directly. The
//   convention: a doc comment starts with the identifier name, written in
//   complete sentences. Tools, including `go vet` and `golint`, enforce
//   this softly.
// ─────────────────────────────────────────────────────────────────────────────

// Package main demonstrates the doc-comment convention. In a real package
// this comment would describe the package's purpose; in main packages it
// usually describes the program.
package main

import (
	"fmt"
	"strings"
)

// Greeter holds the configuration for a polite greeting service.
//
// Greeter is intentionally simple: it carries a salutation prefix and a
// formality flag. Real services would wrap a logger, a metrics handle, and
// a configuration struct, but the doc-comment convention is the same.
type Greeter struct {
	// Prefix is prepended to every greeting. Required; the zero value
	// produces output that looks broken.
	Prefix string

	// Formal toggles "hello" vs "Greetings". Optional; defaults to
	// informal because that matches typical UI tone.
	Formal bool
}

// NewGreeter constructs a Greeter with sensible defaults.
//
// The convention "constructor named NewT for type T" is widely followed in
// idiomatic Go. NewGreeter never fails for the cases it accepts; it returns
// (Greeter, error) only because real constructors usually do, and we want
// the example to look like real code.
func NewGreeter(prefix string) (Greeter, error) {
	if strings.TrimSpace(prefix) == "" {
		return Greeter{}, fmt.Errorf("prefix must not be blank")
	}
	return Greeter{Prefix: prefix}, nil
}

// Greet returns a greeting addressed to name.
//
// Greet returns a fresh string on each call; it does not allocate any
// fields on the receiver. The formal/informal toggle is read at call
// time, not at construction time, so a single Greeter can switch tone
// over its lifetime.
func (g Greeter) Greet(name string) string {
	salutation := "hello"
	if g.Formal {
		salutation = "Greetings"
	}
	return fmt.Sprintf("%s %s, %s.", g.Prefix, salutation, name)
}

func main() {
	// We use NewGreeter to show the constructor pattern; in production
	// code this is the only public way callers build a Greeter, which
	// gives the package a place to enforce invariants.
	g, err := NewGreeter("[demo]")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(g.Greet("world"))

	// A second Greeter, this time formal. The doc comment on Formal said
	// "defaults to informal because that matches typical UI tone" — by
	// flipping it explicitly, we demonstrate the override.
	g.Formal = true
	fmt.Println(g.Greet("Reader"))

	// The cool part: from a terminal, this works:
	//
	//   go doc -all ./examples/02_go_doc_demo
	//
	// and prints every comment above. Same content, two surfaces — the
	// CLI and pkg.go.dev. There is no separate doc-build step.
	fmt.Println("(now run: go doc -all ./examples/02_go_doc_demo)")
}
