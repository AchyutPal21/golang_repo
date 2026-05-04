// FILE: book/part6_production_engineering/chapter82_testing_fundamentals/exercises/01_test_suite/main.go
// CHAPTER: 82 — Testing Fundamentals
// TOPIC: Full test suite for a URL shortener service — table-driven, subtests,
//        fixtures, error cases, and property-based invariants.
//
// Run:
//   go run ./exercises/01_test_suite

package main

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
)

// ─────────────────────────────────────────────────────────────────────────────
// URL SHORTENER — system under test
// ─────────────────────────────────────────────────────────────────────────────

type ShortenerError struct {
	Code    string
	Message string
}

func (e *ShortenerError) Error() string { return fmt.Sprintf("[%s] %s", e.Code, e.Message) }

var (
	ErrInvalidURL  = &ShortenerError{"INVALID_URL", "URL must start with http:// or https://"}
	ErrNotFound    = &ShortenerError{"NOT_FOUND", "short code not found"}
	ErrCodeTaken   = &ShortenerError{"CODE_TAKEN", "custom code already in use"}
)

type Stats struct {
	Clicks atomic.Int64
}

type URLEntry struct {
	Long  string
	Short string
	Stats *Stats
}

type Shortener struct {
	mu      sync.RWMutex
	codes   map[string]*URLEntry // short → entry
	reverse map[string]string    // long → short
	seq     int
}

func NewShortener() *Shortener {
	return &Shortener{
		codes:   make(map[string]*URLEntry),
		reverse: make(map[string]string),
	}
}

func (s *Shortener) Shorten(long string) (string, error) {
	return s.ShortenWithCode(long, "")
}

func (s *Shortener) ShortenWithCode(long, customCode string) (string, error) {
	if !strings.HasPrefix(long, "http://") && !strings.HasPrefix(long, "https://") {
		return "", ErrInvalidURL
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// Return existing short code if already shortened.
	if existing, ok := s.reverse[long]; ok && customCode == "" {
		return existing, nil
	}

	code := customCode
	if code == "" {
		s.seq++
		code = fmt.Sprintf("%06d", s.seq)
	} else if _, taken := s.codes[code]; taken {
		return "", ErrCodeTaken
	}

	entry := &URLEntry{Long: long, Short: code, Stats: &Stats{}}
	s.codes[code] = entry
	s.reverse[long] = code
	return code, nil
}

func (s *Shortener) Resolve(code string) (string, error) {
	s.mu.RLock()
	entry, ok := s.codes[code]
	s.mu.RUnlock()
	if !ok {
		return "", ErrNotFound
	}
	entry.Stats.Clicks.Add(1)
	return entry.Long, nil
}

func (s *Shortener) Clicks(code string) (int64, error) {
	s.mu.RLock()
	entry, ok := s.codes[code]
	s.mu.RUnlock()
	if !ok {
		return 0, ErrNotFound
	}
	return entry.Stats.Clicks.Load(), nil
}

func (s *Shortener) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.codes)
}

// ─────────────────────────────────────────────────────────────────────────────
// MINI TEST FRAMEWORK
// ─────────────────────────────────────────────────────────────────────────────

type T struct{ name string; failed bool; logs []string }

func (t *T) Errorf(f string, a ...any) {
	t.failed = true
	t.logs = append(t.logs, "    FAIL: "+fmt.Sprintf(f, a...))
}

type Suite struct{ passed, failed int }

func (s *Suite) Run(name string, fn func(*T)) {
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
	fmt.Printf("  %d/%d passed\n", s.passed, s.passed+s.failed)
}

// ─────────────────────────────────────────────────────────────────────────────
// FIXTURE
// ─────────────────────────────────────────────────────────────────────────────

type Fixture struct {
	svc   *Shortener
	codes map[string]string // label → short code
}

func newFixture() *Fixture {
	svc := NewShortener()
	codes := make(map[string]string)
	goCode, _ := svc.Shorten("https://golang.org")
	codes["golang"] = goCode
	ghCode, _ := svc.ShortenWithCode("https://github.com", "gh")
	codes["github"] = ghCode
	return &Fixture{svc: svc, codes: codes}
}

// ─────────────────────────────────────────────────────────────────────────────
// TESTS
// ─────────────────────────────────────────────────────────────────────────────

func runShortenTests(s *Suite) {
	cases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid_https", "https://example.com", false},
		{"valid_http", "http://example.com", false},
		{"missing_scheme", "example.com", true},
		{"ftp_scheme", "ftp://example.com", true},
		{"empty", "", true},
	}
	for _, tc := range cases {
		tc := tc
		s.Run("Shorten/"+tc.name, func(t *T) {
			svc := NewShortener()
			_, err := svc.Shorten(tc.url)
			if tc.wantErr && err == nil {
				t.Errorf("Shorten(%q): expected error, got nil", tc.url)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Shorten(%q): unexpected error: %v", tc.url, err)
			}
		})
	}
}

func runResolveTests(s *Suite) {
	fx := newFixture()

	s.Run("Resolve/found", func(t *T) {
		long, err := fx.svc.Resolve(fx.codes["golang"])
		if err != nil {
			t.Errorf("Resolve: unexpected error: %v", err)
			return
		}
		if long != "https://golang.org" {
			t.Errorf("Resolve: got %q, want https://golang.org", long)
		}
	})

	s.Run("Resolve/custom_code", func(t *T) {
		long, err := fx.svc.Resolve("gh")
		if err != nil {
			t.Errorf("Resolve(gh): %v", err)
			return
		}
		if long != "https://github.com" {
			t.Errorf("Resolve(gh): got %q", long)
		}
	})

	s.Run("Resolve/not_found", func(t *T) {
		_, err := fx.svc.Resolve("xxxxxx")
		if err == nil {
			t.Errorf("Resolve(xxxxxx): expected ErrNotFound, got nil")
		}
	})

	s.Run("Resolve/increments_clicks", func(t *T) {
		code := fx.codes["golang"]
		before, _ := fx.svc.Clicks(code)
		fx.svc.Resolve(code)
		fx.svc.Resolve(code)
		after, _ := fx.svc.Clicks(code)
		if after != before+2 {
			t.Errorf("clicks: got %d, want %d", after, before+2)
		}
	})
}

func runCustomCodeTests(s *Suite) {
	s.Run("CustomCode/accepted", func(t *T) {
		svc := NewShortener()
		code, err := svc.ShortenWithCode("https://example.com", "mycode")
		if err != nil {
			t.Errorf("ShortenWithCode: unexpected error: %v", err)
			return
		}
		if code != "mycode" {
			t.Errorf("ShortenWithCode: got code %q, want mycode", code)
		}
	})

	s.Run("CustomCode/duplicate_rejected", func(t *T) {
		svc := NewShortener()
		svc.ShortenWithCode("https://a.com", "clash")
		_, err := svc.ShortenWithCode("https://b.com", "clash")
		if err == nil {
			t.Errorf("duplicate custom code: expected ErrCodeTaken, got nil")
		}
	})
}

func runIdempotencyTests(s *Suite) {
	s.Run("Idempotency/same_url_same_code", func(t *T) {
		svc := NewShortener()
		c1, _ := svc.Shorten("https://idempotent.com")
		c2, _ := svc.Shorten("https://idempotent.com")
		if c1 != c2 {
			t.Errorf("same URL produced different codes: %q vs %q", c1, c2)
		}
		if svc.Count() != 1 {
			t.Errorf("Count = %d, want 1 (no duplicate entries)", svc.Count())
		}
	})
}

func runPropertyTests(s *Suite) {
	// Property: resolve(shorten(url)) == url for any valid URL
	s.Run("Property/round_trip", func(t *T) {
		svc := NewShortener()
		urls := []string{
			"https://a.com", "https://b.org", "http://c.net",
			"https://long-url-example.com/path?q=1&r=2",
		}
		for _, url := range urls {
			code, err := svc.Shorten(url)
			if err != nil {
				t.Errorf("Shorten(%q): %v", url, err)
				continue
			}
			got, err := svc.Resolve(code)
			if err != nil {
				t.Errorf("Resolve(%q): %v", code, err)
				continue
			}
			if got != url {
				t.Errorf("round-trip: got %q, want %q", got, url)
			}
		}
	})

	// Property: concurrent shortens of the same URL produce the same code
	s.Run("Property/concurrent_idempotency", func(t *T) {
		svc := NewShortener()
		url := "https://concurrent.example.com"
		codes := make([]string, 10)
		var wg sync.WaitGroup
		for i := range codes {
			i := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				c, _ := svc.Shorten(url)
				codes[i] = c
			}()
		}
		wg.Wait()
		for i, c := range codes {
			if c != codes[0] {
				t.Errorf("goroutine %d got code %q, goroutine 0 got %q", i, c, codes[0])
			}
		}
	})

	// Property: click count is exactly the number of Resolve calls
	s.Run("Property/click_count_accurate", func(t *T) {
		svc := NewShortener()
		code, _ := svc.Shorten("https://clicks.example.com")
		n := 5 + rand.Intn(10)
		for i := 0; i < n; i++ {
			svc.Resolve(code)
		}
		clicks, _ := svc.Clicks(code)
		if int(clicks) != n {
			t.Errorf("clicks = %d, want %d", clicks, n)
		}
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== URL Shortener Test Suite ===")
	fmt.Println()

	fmt.Println("--- Shorten validation ---")
	s1 := &Suite{}
	runShortenTests(s1)
	s1.Report()

	fmt.Println()
	fmt.Println("--- Resolve ---")
	s2 := &Suite{}
	runResolveTests(s2)
	s2.Report()

	fmt.Println()
	fmt.Println("--- Custom codes ---")
	s3 := &Suite{}
	runCustomCodeTests(s3)
	s3.Report()

	fmt.Println()
	fmt.Println("--- Idempotency ---")
	s4 := &Suite{}
	runIdempotencyTests(s4)
	s4.Report()

	fmt.Println()
	fmt.Println("--- Property-based ---")
	s5 := &Suite{}
	runPropertyTests(s5)
	s5.Report()
}
