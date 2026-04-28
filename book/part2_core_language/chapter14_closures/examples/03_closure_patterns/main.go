// FILE: book/part2_core_language/chapter14_closures/examples/03_closure_patterns/main.go
// CHAPTER: 14 — Closures and the Capture Model
// TOPIC: Real-world closure patterns: lazy evaluation, once, middleware,
//        option functions, and closure-based iterators.
//
// Run (from the chapter folder):
//   go run ./examples/03_closure_patterns

package main

import (
	"fmt"
	"strings"
	"sync"
)

// --- Lazy evaluation ---

// lazy returns a func() T that computes its value exactly once on first call.
// Subsequent calls return the cached result.
// Not concurrency-safe; see onceSafe below for the safe version.
func lazy[T any](compute func() T) func() T {
	var (
		value    T
		computed bool
	)
	return func() T {
		if !computed {
			value = compute()
			computed = true
		}
		return value
	}
}

// onceSafe wraps compute in sync.Once for concurrency-safe lazy eval.
func onceSafe[T any](compute func() T) func() T {
	var (
		once  sync.Once
		value T
	)
	return func() T {
		once.Do(func() { value = compute() })
		return value
	}
}

// --- Middleware / handler wrapping ---

type Handler func(string) string

// withLogging wraps a Handler to print input and output.
func withLogging(h Handler) Handler {
	return func(input string) string {
		result := h(input)
		fmt.Printf("[log] input=%q output=%q\n", input, result)
		return result
	}
}

// withPrefix wraps a Handler to add a prefix to the output.
func withPrefix(prefix string, h Handler) Handler {
	return func(input string) string {
		return prefix + h(input)
	}
}

// --- Option functions (functional options pattern) ---

type serverConfig struct {
	host    string
	port    int
	timeout int
}

type Option func(*serverConfig)

func WithHost(host string) Option {
	return func(c *serverConfig) { c.host = host }
}

func WithPort(port int) Option {
	return func(c *serverConfig) { c.port = port }
}

func WithTimeout(ms int) Option {
	return func(c *serverConfig) { c.timeout = ms }
}

func newServerConfig(opts ...Option) serverConfig {
	cfg := serverConfig{host: "localhost", port: 8080, timeout: 30}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// --- Closure-based iterator ---

// lines returns a func() (string, bool) iterator over the lines of s.
// Each call returns the next line and true, or ("", false) when exhausted.
func lines(s string) func() (string, bool) {
	parts := strings.Split(s, "\n")
	idx := 0
	return func() (string, bool) {
		for idx < len(parts) {
			line := parts[idx]
			idx++
			if line != "" {
				return line, true
			}
		}
		return "", false
	}
}

// --- Self-referencing closure (recursive lambda) ---

// fibonacci returns a closure that produces successive Fibonacci numbers.
func fibonacci() func() int {
	a, b := 0, 1
	return func() int {
		v := a
		a, b = b, a+b
		return v
	}
}

func main() {
	// --- lazy ---
	expensiveCall := 0
	getValue := lazy(func() int {
		expensiveCall++
		fmt.Println("  [computing expensive value]")
		return 42
	})

	fmt.Println("lazy demo:")
	fmt.Println(" first call:", getValue())  // computes
	fmt.Println(" second call:", getValue()) // cached
	fmt.Println(" computations:", expensiveCall)

	fmt.Println()

	// --- middleware ---
	upper := Handler(strings.ToUpper)
	logged := withLogging(upper)
	prefixed := withPrefix(">> ", withLogging(strings.ToLower))

	fmt.Println("middleware demo:")
	_ = logged("hello world")
	_ = prefixed("HELLO")

	fmt.Println()

	// --- functional options ---
	cfg := newServerConfig(
		WithHost("api.example.com"),
		WithPort(9090),
	)
	fmt.Printf("config: %+v\n", cfg)

	defaults := newServerConfig()
	fmt.Printf("defaults: %+v\n", defaults)

	fmt.Println()

	// --- iterator ---
	text := "alpha\nbeta\n\ngamma\ndelta"
	next := lines(text)
	fmt.Println("lines iterator:")
	for line, ok := next(); ok; line, ok = next() {
		fmt.Println(" ", line)
	}

	fmt.Println()

	// --- fibonacci ---
	fib := fibonacci()
	fmt.Print("fibonacci: ")
	for range 10 {
		fmt.Print(fib(), " ")
	}
	fmt.Println()
}
