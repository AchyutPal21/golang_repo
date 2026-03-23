// FILE: 02_functions/09_function_types_interfaces.go
// TOPIC: Function Types — named function types, implementing interfaces, middleware
//
// Run: go run 02_functions/09_function_types_interfaces.go
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS MATTERS:
//   Go functions are first-class values AND they can implement interfaces.
//   This is the basis of http.HandlerFunc, middleware chains, and plugin systems.
//   Understanding function types lets you write extremely flexible APIs where
//   callers can pass either a named type OR an anonymous function.
// ─────────────────────────────────────────────────────────────────────────────

package main

import (
	"fmt"
	"strings"
)

// ── NAMED FUNCTION TYPES ────────────────────────────────────────────────────
// You can give a function signature a name, just like naming a struct.
// This makes code more readable and allows attaching methods to function types.

// Transformer is a function that takes a string and returns a string.
type Transformer func(string) string

// Predicate is a function that tests a condition on a string.
type Predicate func(string) bool

// Handler processes a request string and returns a response string.
// (Modelled loosely on http.Handler for familiarity)
type Handler func(string) string

// ── IMPLEMENTING AN INTERFACE WITH A FUNCTION TYPE ──────────────────────────
// This is the http.HandlerFunc pattern — one of Go's most elegant designs.
//
// Define an interface, then define a function type that implements it.
// This lets callers use EITHER a struct (with a method) OR a plain function.

// Processor is an interface for processing messages.
type Processor interface {
	Process(msg string) string
}

// ProcessorFunc is a function type that implements Processor.
// By defining the Process method on it, any func(string) string
// automatically implements the Processor interface after conversion.
type ProcessorFunc func(string) string

func (f ProcessorFunc) Process(msg string) string {
	return f(msg) // delegate to the underlying function
}

// This is EXACTLY how net/http works:
//   type HandlerFunc func(ResponseWriter, *Request)
//   func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) { f(w, r) }

// runProcessor accepts the interface — works with both structs AND function literals
func runProcessor(p Processor, input string) string {
	return p.Process(input)
}

// ── MIDDLEWARE PATTERN ──────────────────────────────────────────────────────
// Middleware wraps a Handler with additional behavior.
// Each middleware is a function that takes a Handler and returns a Handler.
// This enables composable pipelines.

type Middleware func(Handler) Handler

func loggingMiddleware(next Handler) Handler {
	return func(req string) string {
		fmt.Printf("  [LOG] → request: %q\n", req)
		resp := next(req)
		fmt.Printf("  [LOG] ← response: %q\n", resp)
		return resp
	}
}

func uppercaseMiddleware(next Handler) Handler {
	return func(req string) string {
		return strings.ToUpper(next(req))
	}
}

func prefixMiddleware(prefix string) Middleware {
	return func(next Handler) Handler {
		return func(req string) string {
			return prefix + ": " + next(req)
		}
	}
}

// chain applies middleware in order: chain(h, m1, m2, m3) = m3(m2(m1(h)))
func chain(h Handler, middlewares ...Middleware) Handler {
	// Apply in reverse so the first middleware is outermost
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Function Types & Interfaces")
	fmt.Println("════════════════════════════════════════")

	// ── Named function types ─────────────────────────────────────────────
	fmt.Println("\n── Named function types ──")

	var t Transformer = strings.ToUpper
	fmt.Printf("  Transformer(\"hello\") = %q\n", t("hello"))

	var p Predicate = func(s string) bool { return len(s) > 3 }
	fmt.Printf("  Predicate(\"hi\")   = %v\n", p("hi"))
	fmt.Printf("  Predicate(\"hello\")= %v\n", p("hello"))

	// Slice of transformers — apply in sequence
	pipeline := []Transformer{
		strings.TrimSpace,
		strings.ToLower,
		func(s string) string { return "[" + s + "]" },
	}
	result := "  HELLO WORLD  "
	for _, fn := range pipeline {
		result = fn(result)
	}
	fmt.Printf("  Pipeline result: %q\n", result)

	// ── Function type implementing interface ─────────────────────────────
	fmt.Println("\n── ProcessorFunc implements Processor interface ──")

	// Option 1: use a struct with a Process method (traditional OOP)
	// Option 2: use ProcessorFunc (Go's idiomatic approach)

	reverseProcessor := ProcessorFunc(func(msg string) string {
		runes := []rune(msg)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return string(runes)
	})

	fmt.Printf("  runProcessor(reverseProcessor, \"hello\") = %q\n",
		runProcessor(reverseProcessor, "hello"))

	// Both a struct and a function type satisfy the same interface:
	shoutProcessor := ProcessorFunc(strings.ToUpper)
	fmt.Printf("  runProcessor(shoutProcessor, \"hello\")   = %q\n",
		runProcessor(shoutProcessor, "hello"))

	// ── Middleware chaining ──────────────────────────────────────────────
	fmt.Println("\n── Middleware chain ──")

	// The base handler — the actual logic
	baseHandler := Handler(func(req string) string {
		return "echo: " + req
	})

	// Wrap with middleware (applied outermost-first reading left to right):
	// loggingMiddleware → prefixMiddleware("API") → uppercaseMiddleware → baseHandler
	h := chain(baseHandler,
		loggingMiddleware,
		prefixMiddleware("API"),
		uppercaseMiddleware,
	)

	fmt.Println("  Calling chained handler:")
	response := h("ping")
	fmt.Printf("  Final response: %q\n", response)

	fmt.Println("\n─── SUMMARY ────────────────────────────────")
	fmt.Println("  Named function type: type MyFunc func(T) T")
	fmt.Println("  Add methods to function types → implement interfaces")
	fmt.Println("  The http.HandlerFunc pattern: most flexible API design")
	fmt.Println("  Middleware: func(Handler) Handler — composable wrappers")
	fmt.Println("  chain() applies middleware in order — builds pipelines")
}
