// FILE: book/part6_production_engineering/chapter97_security/examples/01_owasp_patterns/main.go
// CHAPTER: 97 — Security for Go Services
// TOPIC: OWASP Top 10 patterns — SQL injection, XSS, path traversal,
//        SSRF, and command injection with Go mitigations.
//
// Run:
//   go run ./book/part6_production_engineering/chapter97_security/examples/01_owasp_patterns

package main

import (
	"errors"
	"fmt"
	"html"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// SQL INJECTION
// ─────────────────────────────────────────────────────────────────────────────

func buildQueryVulnerable(username string) string {
	return "SELECT * FROM users WHERE username = '" + username + "'"
}

func buildQuerySafe(username string) string {
	// In production: db.QueryRow("SELECT * FROM users WHERE username = ?", username)
	return fmt.Sprintf("SELECT * FROM users WHERE username = ? [args: %q]", username)
}

// ─────────────────────────────────────────────────────────────────────────────
// XSS
// ─────────────────────────────────────────────────────────────────────────────

func renderCommentVulnerable(comment string) string {
	return "<div class='comment'>" + comment + "</div>"
}

func renderCommentSafe(comment string) string {
	return "<div class='comment'>" + html.EscapeString(comment) + "</div>"
}

// ─────────────────────────────────────────────────────────────────────────────
// PATH TRAVERSAL
// ─────────────────────────────────────────────────────────────────────────────

var ErrPathTraversal = errors.New("path traversal detected")

func openFileVulnerable(baseDir, filename string) string {
	return filepath.Join(baseDir, filename) // allows ../../../../etc/passwd
}

func openFileSafe(baseDir, filename string) (string, error) {
	clean := filepath.Clean(filepath.Join(baseDir, filename))
	if !strings.HasPrefix(clean, filepath.Clean(baseDir)+"/") {
		return "", ErrPathTraversal
	}
	return clean, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SSRF (Server-Side Request Forgery)
// ─────────────────────────────────────────────────────────────────────────────

var ssrfAllowlist = []*regexp.Regexp{
	regexp.MustCompile(`^https://api\.example\.com/`),
	regexp.MustCompile(`^https://cdn\.example\.com/`),
}

var ErrSSRFBlocked = errors.New("URL not in allowlist")

func fetchURLVulnerable(rawURL string) string {
	return fmt.Sprintf("fetching %s (UNSAFE: no validation)", rawURL)
}

func fetchURLSafe(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return "", fmt.Errorf("%w: only HTTPS allowed", ErrSSRFBlocked)
	}
	for _, pattern := range ssrfAllowlist {
		if pattern.MatchString(rawURL) {
			return fmt.Sprintf("fetching %s (allowed)", rawURL), nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrSSRFBlocked, rawURL)
}

// ─────────────────────────────────────────────────────────────────────────────
// COMMAND INJECTION
// ─────────────────────────────────────────────────────────────────────────────

func pingVulnerable(host string) string {
	// NEVER do this: exec.Command("sh", "-c", "ping -c 1 "+host)
	// host = "google.com; rm -rf /"  → disaster
	return fmt.Sprintf("exec: sh -c 'ping -c 1 %s'  ← DANGEROUS", host)
}

func pingSafe(host string) string {
	// Validate host first, then pass as a separate argument (no shell expansion)
	// In production: exec.Command("ping", "-c", "1", host)  ← shell never involved
	hostRegex := regexp.MustCompile(`^[a-zA-Z0-9.\-]+$`)
	if !hostRegex.MatchString(host) {
		return fmt.Sprintf("rejected invalid host: %q", host)
	}
	return fmt.Sprintf("exec: ping -c 1 %q  ← SAFE (no shell)", host)
}

// ─────────────────────────────────────────────────────────────────────────────
// INPUT VALIDATION
// ─────────────────────────────────────────────────────────────────────────────

var validUsername = regexp.MustCompile(`^[a-zA-Z0-9_\-]{1,32}$`)
var validEmail = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func validateUsername(s string) error {
	if s == "" {
		return errors.New("username: empty")
	}
	if len(s) > 32 {
		return errors.New("username: too long (max 32)")
	}
	if !validUsername.MatchString(s) {
		return errors.New("username: only [a-zA-Z0-9_-] allowed")
	}
	return nil
}

func validateEmail(s string) error {
	if !validEmail.MatchString(s) {
		return errors.New("email: invalid format")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 97: OWASP Security Patterns ===")
	fmt.Println()

	// ── SQL INJECTION ─────────────────────────────────────────────────────────
	fmt.Println("--- SQL Injection ---")
	maliciousInput := "' OR '1'='1'; DROP TABLE users; --"
	fmt.Printf("  Input: %q\n", maliciousInput)
	fmt.Printf("  Vulnerable: %s\n", buildQueryVulnerable(maliciousInput))
	fmt.Printf("  Safe:       %s\n\n", buildQuerySafe(maliciousInput))

	// ── XSS ───────────────────────────────────────────────────────────────────
	fmt.Println("--- Cross-Site Scripting (XSS) ---")
	xssInput := `<script>document.cookie='stolen='+document.cookie</script>`
	fmt.Printf("  Input: %q\n", xssInput)
	fmt.Printf("  Vulnerable: %s\n", renderCommentVulnerable(xssInput))
	fmt.Printf("  Safe:       %s\n\n", renderCommentSafe(xssInput))

	// ── PATH TRAVERSAL ────────────────────────────────────────────────────────
	fmt.Println("--- Path Traversal ---")
	baseDir := "/var/app/uploads"
	traversalInput := "../../../../etc/passwd"
	fmt.Printf("  Base: %q  Input: %q\n", baseDir, traversalInput)
	fmt.Printf("  Vulnerable: %s\n", openFileVulnerable(baseDir, traversalInput))
	if p, err := openFileSafe(baseDir, traversalInput); err != nil {
		fmt.Printf("  Safe:       ERROR: %v\n", err)
	} else {
		fmt.Printf("  Safe:       %s\n", p)
	}
	validFile := "report-2024.pdf"
	if p, err := openFileSafe(baseDir, validFile); err == nil {
		fmt.Printf("  Safe (valid): %s\n\n", p)
	}

	// ── SSRF ──────────────────────────────────────────────────────────────────
	fmt.Println("--- SSRF ---")
	urls := []string{
		"http://169.254.169.254/latest/meta-data/",  // AWS metadata service
		"https://api.example.com/v1/data",           // allowed
		"https://evil.attacker.com/steal",           // blocked
	}
	for _, u := range urls {
		if result, err := fetchURLSafe(u); err != nil {
			fmt.Printf("  BLOCKED: %v\n", err)
		} else {
			fmt.Printf("  ALLOWED: %s\n", result)
		}
	}
	fmt.Println()

	// ── COMMAND INJECTION ─────────────────────────────────────────────────────
	fmt.Println("--- Command Injection ---")
	maliciousHost := "google.com; rm -rf /"
	safeHost := "google.com"
	fmt.Printf("  Malicious host: %q\n", maliciousHost)
	fmt.Printf("  Vulnerable: %s\n", pingVulnerable(maliciousHost))
	fmt.Printf("  Safe:       %s\n", pingSafe(maliciousHost))
	fmt.Printf("  Safe host:  %s\n\n", pingSafe(safeHost))

	// ── INPUT VALIDATION ──────────────────────────────────────────────────────
	fmt.Println("--- Input Validation ---")
	inputs := []string{"alice", "bob_smith", "admin'; DROP TABLE--", "", strings.Repeat("a", 40)}
	for _, u := range inputs {
		if err := validateUsername(u); err != nil {
			fmt.Printf("  %q → INVALID: %v\n", u, err)
		} else {
			fmt.Printf("  %q → valid\n", u)
		}
	}
	fmt.Println()

	// ── SECURITY HEADERS REFERENCE ────────────────────────────────────────────
	fmt.Println("--- Security response headers ---")
	fmt.Println(`  Content-Security-Policy: default-src 'self'
  X-Frame-Options: DENY
  X-Content-Type-Options: nosniff
  Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
  Referrer-Policy: strict-origin-when-cross-origin
  Permissions-Policy: geolocation=(), microphone=(), camera=()`)
}
