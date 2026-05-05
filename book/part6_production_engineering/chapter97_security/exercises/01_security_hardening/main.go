// FILE: book/part6_production_engineering/chapter97_security/exercises/01_security_hardening/main.go
// CHAPTER: 97 — Security for Go Services
// EXERCISE: Security hardening checklist — input validation pipeline,
//           audit log, secret scanning, security headers middleware.
//
// Run:
//   go run ./book/part6_production_engineering/chapter97_security/exercises/01_security_hardening

package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// INPUT VALIDATION PIPELINE
// ─────────────────────────────────────────────────────────────────────────────

type ValidationRule func(s string) error

type Validator struct {
	Field string
	Rules []ValidationRule
}

func (v Validator) Validate(input string) error {
	for _, rule := range v.Rules {
		if err := rule(input); err != nil {
			return fmt.Errorf("%s: %w", v.Field, err)
		}
	}
	return nil
}

func RuleNotEmpty() ValidationRule {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("must not be empty")
		}
		return nil
	}
}

func RuleMaxLength(n int) ValidationRule {
	return func(s string) error {
		if len(s) > n {
			return fmt.Errorf("too long (max %d chars)", n)
		}
		return nil
	}
}

func RuleRegex(pattern, description string) ValidationRule {
	re := regexp.MustCompile(pattern)
	return func(s string) error {
		if !re.MatchString(s) {
			return fmt.Errorf("must match %s", description)
		}
		return nil
	}
}

func RuleHTTPSOnly() ValidationRule {
	return func(s string) error {
		u, err := url.Parse(s)
		if err != nil || u.Scheme != "https" {
			return errors.New("must be an HTTPS URL")
		}
		return nil
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// AUDIT LOG
// ─────────────────────────────────────────────────────────────────────────────

type AuditEvent struct {
	Timestamp time.Time
	EventType string
	UserID    string
	Resource  string
	Action    string
	Outcome   string // "success" | "failure"
	Details   string
	RequestID string
}

func (e AuditEvent) String() string {
	return fmt.Sprintf("[%s] %s user=%s resource=%s action=%s outcome=%s req=%s %s",
		e.Timestamp.Format(time.RFC3339),
		e.EventType, e.UserID, e.Resource, e.Action, e.Outcome, e.RequestID, e.Details)
}

type AuditLog struct {
	mu     sync.Mutex
	events []AuditEvent
}

func (a *AuditLog) Log(e AuditEvent) {
	if e.RequestID == "" {
		b := make([]byte, 8)
		rand.Read(b)
		e.RequestID = hex.EncodeToString(b)
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	a.mu.Lock()
	a.events = append(a.events, e)
	a.mu.Unlock()
	fmt.Printf("  AUDIT: %s\n", e)
}

func (a *AuditLog) Events() []AuditEvent {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]AuditEvent, len(a.events))
	copy(out, a.events)
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// SECRET SCANNER
// ─────────────────────────────────────────────────────────────────────────────

var secretPatterns = map[string]*regexp.Regexp{
	"AWS key":         regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	"Generic secret":  regexp.MustCompile(`(?i)(secret|password|passwd|token|api.?key)\s*[=:]\s*\S{8,}`),
	"Private key":     regexp.MustCompile(`-----BEGIN (RSA|EC|OPENSSH) PRIVATE KEY-----`),
	"Bearer token":    regexp.MustCompile(`Bearer\s+[A-Za-z0-9._\-]{20,}`),
}

type SecretFinding struct {
	Source  string
	Type    string
	Context string
}

func scanForSecrets(source, content string) []SecretFinding {
	var findings []SecretFinding
	for typ, pattern := range secretPatterns {
		if match := pattern.FindString(content); match != "" {
			findings = append(findings, SecretFinding{
				Source:  source,
				Type:    typ,
				Context: truncate(match, 30),
			})
		}
	}
	return findings
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// ─────────────────────────────────────────────────────────────────────────────
// SECURITY HEADERS
// ─────────────────────────────────────────────────────────────────────────────

type SecurityHeaderConfig struct {
	CSP   string
	HSTS  bool
}

var defaultSecurityHeaders = map[string]string{
	"X-Frame-Options":           "DENY",
	"X-Content-Type-Options":    "nosniff",
	"X-XSS-Protection":          "0", // modern browsers use CSP instead
	"Referrer-Policy":           "strict-origin-when-cross-origin",
	"Content-Security-Policy":   "default-src 'self'; script-src 'self'; style-src 'self'",
	"Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload",
	"Permissions-Policy":        "geolocation=(), microphone=(), camera=()",
}

func printSecurityHeaders() {
	fmt.Printf("  %-35s  %s\n", "Header", "Value")
	fmt.Printf("  %s\n", strings.Repeat("-", 75))
	for k, v := range defaultSecurityHeaders {
		fmt.Printf("  %-35s  %s\n", k, v)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 97 Exercise: Security Hardening ===")
	fmt.Println()

	// ── INPUT VALIDATION ──────────────────────────────────────────────────────
	fmt.Println("--- Input validation pipeline ---")
	usernameValidator := Validator{
		Field: "username",
		Rules: []ValidationRule{
			RuleNotEmpty(),
			RuleMaxLength(32),
			RuleRegex(`^[a-zA-Z0-9_\-]+$`, "[a-zA-Z0-9_-]"),
		},
	}
	urlValidator := Validator{
		Field: "webhook_url",
		Rules: []ValidationRule{
			RuleNotEmpty(),
			RuleHTTPSOnly(),
			RuleMaxLength(512),
		},
	}

	inputs := []struct{ field, value string }{
		{"username", "alice_smith"},
		{"username", "'; DROP TABLE users--"},
		{"username", ""},
		{"webhook_url", "https://api.example.com/hook"},
		{"webhook_url", "http://evil.com/steal"},
	}
	for _, inp := range inputs {
		var v Validator
		if inp.field == "username" {
			v = usernameValidator
		} else {
			v = urlValidator
		}
		if err := v.Validate(inp.value); err != nil {
			fmt.Printf("  %-15q → INVALID: %v\n", inp.value, err)
		} else {
			fmt.Printf("  %-15q → valid\n", inp.value)
		}
	}
	fmt.Println()

	// ── AUDIT LOG ─────────────────────────────────────────────────────────────
	fmt.Println("--- Audit log ---")
	log := &AuditLog{}
	log.Log(AuditEvent{EventType: "auth", UserID: "user-42", Resource: "/api/orders", Action: "list", Outcome: "success"})
	log.Log(AuditEvent{EventType: "auth", UserID: "attacker", Resource: "/admin", Action: "access", Outcome: "failure", Details: "insufficient role"})
	log.Log(AuditEvent{EventType: "data", UserID: "user-42", Resource: "order:99", Action: "delete", Outcome: "success"})
	fmt.Printf("  %d events logged\n\n", len(log.Events()))

	// ── SECRET SCANNING ───────────────────────────────────────────────────────
	fmt.Println("--- Secret scanning ---")
	testContents := []struct {
		source  string
		content string
	}{
		{"env", "DATABASE_URL=postgres://localhost/app"},
		{"env", "SECRET_KEY=mysupersecretkey123"},
		{"config", "api_key: AKIAIOSFODNN7EXAMPLE123456"},
		{"code", `token := "Bearer eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJ1c2VyIn0.abc"`},
	}
	for _, tc := range testContents {
		findings := scanForSecrets(tc.source, tc.content)
		if len(findings) > 0 {
			for _, f := range findings {
				fmt.Printf("  WARNING [%s] %s detected: %q\n", f.Source, f.Type, f.Context)
			}
		} else {
			fmt.Printf("  OK [%s] no secrets detected\n", tc.source)
		}
	}
	fmt.Println()

	// ── ENVIRONMENT SECRET SCAN ───────────────────────────────────────────────
	fmt.Println("--- Environment secret scan ---")
	envCount := 0
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			findings := scanForSecrets("env:"+parts[0], parts[0]+"="+parts[1])
			if len(findings) > 0 {
				envCount++
			}
		}
	}
	if envCount == 0 {
		fmt.Println("  No secret-like environment variables detected.")
	} else {
		fmt.Printf("  WARNING: %d potentially secret env vars found\n", envCount)
	}
	fmt.Println()

	// ── SECURITY HEADERS ──────────────────────────────────────────────────────
	fmt.Println("--- Security headers (add to every HTTP response) ---")
	printSecurityHeaders()
	fmt.Println()

	// ── GOVULNCHECK REFERENCE ─────────────────────────────────────────────────
	fmt.Println("--- govulncheck integration ---")
	fmt.Println(`  Install:  go install golang.org/x/vuln/cmd/govulncheck@latest
  Run:      govulncheck ./...
  CI step:  add after go test, before docker build

  Key difference from go list -m all:
    govulncheck only reports vulnerabilities in code paths YOU ACTUALLY CALL,
    not just imported packages. Fewer false positives.

  Example output:
    Vulnerability #1: GO-2024-2448
    More info: https://pkg.go.dev/vuln/GO-2024-2448
    Found in: net/http@go1.21.5 (call stack: main.handler -> net/http.Get)
    Fixed in: go1.22.0`)
}
