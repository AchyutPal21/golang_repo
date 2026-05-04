// FILE: book/part6_production_engineering/chapter82_testing_fundamentals/examples/01_table_driven/main.go
// CHAPTER: 82 — Testing Fundamentals
// TOPIC: Table-driven tests, test helpers, golden files, and test coverage.
//        This file is a runnable demo — the real patterns live in _test.go files
//        in production, but here everything is self-contained for go run.
//
// Run:
//   go run ./examples/01_table_driven

package main

import (
	"fmt"
	"math"
	"strings"
	"unicode"
)

// ─────────────────────────────────────────────────────────────────────────────
// FUNCTIONS UNDER TEST
// ─────────────────────────────────────────────────────────────────────────────

func Add(a, b int) int { return a + b }

func Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return a / b, nil
}

func IsPalindrome(s string) bool {
	s = strings.ToLower(s)
	var letters []rune
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			letters = append(letters, r)
		}
	}
	for i, j := 0, len(letters)-1; i < j; i, j = i+1, j-1 {
		if letters[i] != letters[j] {
			return false
		}
	}
	return true
}

func FizzBuzz(n int) string {
	switch {
	case n%15 == 0:
		return "FizzBuzz"
	case n%3 == 0:
		return "Fizz"
	case n%5 == 0:
		return "Buzz"
	default:
		return fmt.Sprintf("%d", n)
	}
}

func Sqrt(x float64) (float64, error) {
	if x < 0 {
		return 0, fmt.Errorf("sqrt of negative number: %g", x)
	}
	return math.Sqrt(x), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// MINI TEST FRAMEWORK — simulates testing.T behaviour
// ─────────────────────────────────────────────────────────────────────────────

type T struct {
	name   string
	failed bool
	logs   []string
}

func (t *T) Errorf(format string, args ...any) {
	t.failed = true
	t.logs = append(t.logs, fmt.Sprintf("    FAIL: "+format, args...))
}

func (t *T) Logf(format string, args ...any) {
	t.logs = append(t.logs, fmt.Sprintf("    LOG:  "+format, args...))
}

type Suite struct {
	passed, failed int
}

func (s *Suite) Run(name string, fn func(t *T)) {
	t := &T{name: name}
	fn(t)
	if t.failed {
		s.failed++
		fmt.Printf("  --- FAIL: %s\n", name)
		for _, l := range t.logs {
			fmt.Println(l)
		}
	} else {
		s.passed++
		fmt.Printf("  --- PASS: %s\n", name)
	}
}

func (s *Suite) Report() {
	total := s.passed + s.failed
	fmt.Printf("  Results: %d/%d passed", s.passed, total)
	if s.failed > 0 {
		fmt.Printf(", %d FAILED", s.failed)
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// TABLE-DRIVEN TEST PATTERNS
// ─────────────────────────────────────────────────────────────────────────────

func runAddTests(s *Suite) {
	cases := []struct {
		name string
		a, b int
		want int
	}{
		{"zero", 0, 0, 0},
		{"positive", 2, 3, 5},
		{"negative", -4, 4, 0},
		{"large", 1000, 2000, 3000},
	}
	for _, tc := range cases {
		tc := tc
		s.Run("Add/"+tc.name, func(t *T) {
			got := Add(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("Add(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func runDivideTests(s *Suite) {
	cases := []struct {
		name    string
		a, b    float64
		want    float64
		wantErr bool
	}{
		{"normal", 10, 2, 5, false},
		{"fraction", 1, 3, 1.0 / 3.0, false},
		{"by zero", 5, 0, 0, true},
		{"negative", -6, 2, -3, false},
	}
	for _, tc := range cases {
		tc := tc
		s.Run("Divide/"+tc.name, func(t *T) {
			got, err := Divide(tc.a, tc.b)
			if tc.wantErr {
				if err == nil {
					t.Errorf("Divide(%g, %g): expected error, got nil", tc.a, tc.b)
				}
				return
			}
			if err != nil {
				t.Errorf("Divide(%g, %g): unexpected error: %v", tc.a, tc.b, err)
				return
			}
			if math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("Divide(%g, %g) = %g, want %g", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func runPalindromeTests(s *Suite) {
	cases := []struct {
		input string
		want  bool
	}{
		{"racecar", true},
		{"A man a plan a canal Panama", true},
		{"hello", false},
		{"Was it a car or a cat I saw", true},
		{"", true},
		{"a", true},
		{"ab", false},
	}
	for _, tc := range cases {
		tc := tc
		s.Run("IsPalindrome/"+tc.input, func(t *T) {
			got := IsPalindrome(tc.input)
			if got != tc.want {
				t.Errorf("IsPalindrome(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func runFizzBuzzTests(s *Suite) {
	cases := []struct {
		n    int
		want string
	}{
		{1, "1"}, {3, "Fizz"}, {5, "Buzz"}, {15, "FizzBuzz"},
		{30, "FizzBuzz"}, {7, "7"}, {9, "Fizz"}, {10, "Buzz"},
	}
	for _, tc := range cases {
		tc := tc
		s.Run(fmt.Sprintf("FizzBuzz/%d", tc.n), func(t *T) {
			got := FizzBuzz(tc.n)
			if got != tc.want {
				t.Errorf("FizzBuzz(%d) = %q, want %q", tc.n, got, tc.want)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// TEST HELPER PATTERN
// ─────────────────────────────────────────────────────────────────────────────

// assertNoError is a test helper — in real code this marks the caller, not itself.
func assertNoError(t *T, err error, context string) {
	if err != nil {
		t.Errorf("%s: unexpected error: %v", context, err)
	}
}

func assertFloat(t *T, got, want float64, context string) {
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s: got %g, want %g", context, got, want)
	}
}

func runSqrtTests(s *Suite) {
	s.Run("Sqrt/valid", func(t *T) {
		got, err := Sqrt(9)
		assertNoError(t, err, "Sqrt(9)")
		assertFloat(t, got, 3, "Sqrt(9)")
	})
	s.Run("Sqrt/zero", func(t *T) {
		got, err := Sqrt(0)
		assertNoError(t, err, "Sqrt(0)")
		assertFloat(t, got, 0, "Sqrt(0)")
	})
	s.Run("Sqrt/negative", func(t *T) {
		_, err := Sqrt(-1)
		if err == nil {
			t.Errorf("Sqrt(-1): expected error, got nil")
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Table-Driven Tests ===")
	fmt.Println()

	// ── TABLE-DRIVEN: ADD ─────────────────────────────────────────────────────
	fmt.Println("--- Add ---")
	s1 := &Suite{}
	runAddTests(s1)
	s1.Report()

	// ── TABLE-DRIVEN: DIVIDE ──────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Divide ---")
	s2 := &Suite{}
	runDivideTests(s2)
	s2.Report()

	// ── TABLE-DRIVEN: PALINDROME ──────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- IsPalindrome ---")
	s3 := &Suite{}
	runPalindromeTests(s3)
	s3.Report()

	// ── TABLE-DRIVEN: FIZZBUZZ ────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- FizzBuzz ---")
	s4 := &Suite{}
	runFizzBuzzTests(s4)
	s4.Report()

	// ── TEST HELPERS ──────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Sqrt (with test helpers) ---")
	s5 := &Suite{}
	runSqrtTests(s5)
	s5.Report()

	// ── PATTERNS REFERENCE ────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Table-driven test anatomy ---")
	ref := `  // Standard form:
  cases := []struct {
      name  string
      input string
      want  string
  }{
      {"empty", "", ""},
      {"normal", "hello", "HELLO"},
  }
  for _, tc := range cases {
      tc := tc  // capture range var (required in Go <1.22)
      t.Run(tc.name, func(t *testing.T) {
          t.Parallel()  // run subtests concurrently
          got := ToUpper(tc.input)
          if got != tc.want {
              t.Errorf("ToUpper(input) = got, want expected")
          }
      })
  }

  // Test helper (marks caller, not helper):
  func assertEqual[T comparable](t testing.TB, got, want T, msg string) {
      t.Helper()  // failure points to caller line, not this function
      if got != want {
          t.Errorf("msg: got val, want val")
      }
  }`
	fmt.Println(ref)
}
