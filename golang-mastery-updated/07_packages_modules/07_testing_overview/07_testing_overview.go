// FILE: 07_packages_modules/07_testing_overview.go
// TOPIC: Testing in Go — patterns, table tests, benchmarks, examples
//
// Run: go run 07_packages_modules/07_testing_overview.go
//
// NOTE: This file demonstrates testing CONCEPTS and shows code patterns.
// Actual test files are named *_test.go and run with `go test ./...`
// This file runs as a normal program to explain the patterns interactively.

package main

import (
	"fmt"
	"strings"
)

// ── Function under test ───────────────────────────────────────────────────────
func ReverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func IsPalindrome(s string) bool {
	s = strings.ToLower(s)
	return s == ReverseString(s)
}

func main() {
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  Topic: Testing Overview")
	fmt.Println("════════════════════════════════════════")

	// ── TABLE-DRIVEN TESTS — the Go standard ─────────────────────────────
	// Define test cases as a slice of structs.
	// Each case has: name, input(s), expected output.
	// Loop over cases, run each as a subtest with t.Run().
	// WHY? Easy to add cases, output shows which case failed, subtests run independently.

	fmt.Println("\n── Table-driven test pattern ──")
	fmt.Println(`
  // In a *_test.go file:

  func TestReverseString(t *testing.T) {
      tests := []struct {
          name  string
          input string
          want  string
      }{
          {"empty", "", ""},
          {"single", "a", "a"},
          {"ascii", "hello", "olleh"},
          {"unicode", "héllo", "olléh"},
          {"palindrome", "racecar", "racecar"},
      }
      for _, tt := range tests {
          t.Run(tt.name, func(t *testing.T) {
              got := ReverseString(tt.input)
              if got != tt.want {
                  t.Errorf("ReverseString(%q) = %q, want %q", tt.input, got, tt.want)
              }
          })
      }
  }
`)

	// Simulate the test running:
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"single", "a", "a"},
		{"ascii", "hello", "olleh"},
		{"unicode", "héllo", "olléh"},
	}
	fmt.Println("  Running simulated tests:")
	for _, tt := range tests {
		got := ReverseString(tt.input)
		if got == tt.want {
			fmt.Printf("    PASS: ReverseString(%q) = %q\n", tt.input, got)
		} else {
			fmt.Printf("    FAIL: ReverseString(%q) = %q, want %q\n", tt.input, got, tt.want)
		}
	}

	// ── BENCHMARK PATTERN ─────────────────────────────────────────────────
	fmt.Println(`
── Benchmark pattern ──

  func BenchmarkReverseString(b *testing.B) {
      input := "Hello, 世界"
      b.ResetTimer()           // don't count setup time
      for i := 0; i < b.N; i++ {
          ReverseString(input) // b.N adjusts automatically for stable timing
      }
  }

  Run: go test -bench=. -benchmem ./...
  Output:
    BenchmarkReverseString-8   5000000   234 ns/op   48 B/op   2 allocs/op

  -benchmem: shows allocations per op (allocs/op → 0 is ideal for hot paths)
  -benchtime=5s: run for 5 seconds instead of default 1s
`)

	// ── EXAMPLE FUNCTIONS ─────────────────────────────────────────────────
	fmt.Println(`
── Example functions ──

  func ExampleReverseString() {
      fmt.Println(ReverseString("hello"))
      // Output:
      // olleh
  }

  - Appear in godoc as runnable examples
  - The // Output: comment is verified by go test
  - If output doesn't match: test FAILS
  - Great for documenting and testing simultaneously
`)

	// ── TEST HELPERS ──────────────────────────────────────────────────────
	fmt.Println(`
── Test helpers (t.Helper()) ──

  func assertEqual(t *testing.T, got, want string) {
      t.Helper()  // IMPORTANT: marks this as helper, error points to CALLER
      if got != want {
          t.Errorf("got %q, want %q", got, want)
      }
  }

  Without t.Helper(): error line points to inside assertEqual (useless)
  With    t.Helper(): error line points to the test case that called it (useful)
`)

	// ── KEY COMMANDS ──────────────────────────────────────────────────────
	fmt.Println("── Key testing commands ──")
	fmt.Println(`
  go test ./...                   run all tests
  go test -v ./...                verbose: print each test name
  go test -run TestReverse ./...  run only tests matching regex
  go test -race ./...             enable race detector
  go test -cover ./...            show coverage percentage
  go test -coverprofile=c.out ./... && go tool cover -html=c.out
  go test -bench=. -benchmem ./...  run benchmarks with memory stats
  go test -count=5 ./...          run each test 5 times (detect flakiness)
`)
}
