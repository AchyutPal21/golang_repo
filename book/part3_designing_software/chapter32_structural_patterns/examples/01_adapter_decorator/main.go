// FILE: book/part3_designing_software/chapter32_structural_patterns/examples/01_adapter_decorator/main.go
// CHAPTER: 32 — Structural Patterns
// TOPIC: Adapter (bridge incompatible interfaces) and Decorator (add behaviour
//        without modifying the wrapped type) in idiomatic Go.
//
// Run (from the chapter folder):
//   go run ./examples/01_adapter_decorator

package main

import (
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// ADAPTER
//
// Converts the interface of one type into the interface expected by a consumer.
// In Go: wrap the incompatible type in a new struct that satisfies the target
// interface. The adapter does translation, not business logic.
// ─────────────────────────────────────────────────────────────────────────────

// Target interface — what our application expects.
type MessageSender interface {
	Send(to, subject, body string) error
}

// ── Adapt an external SMS library ─────────────────────────────────────────────

// ExternalSMSClient — simulates a third-party library with its own API.
type ExternalSMSClient struct{ apiKey string }

func (c *ExternalSMSClient) SendSMS(phoneNumber, text string) error {
	fmt.Printf("  [SMS API] key=%s phone=%s text=%q\n", c.apiKey[:4]+"…", phoneNumber, text)
	return nil
}

// SMSAdapter makes ExternalSMSClient satisfy MessageSender.
type SMSAdapter struct {
	client *ExternalSMSClient
}

func (a *SMSAdapter) Send(to, subject, body string) error {
	text := fmt.Sprintf("%s: %s", subject, body)
	return a.client.SendSMS(to, text)
}

// ── Adapt a legacy email system ────────────────────────────────────────────────

// LegacyEmailSystem — old API with a different signature.
type LegacyEmailSystem struct{ smtpHost string }

func (l *LegacyEmailSystem) PostMail(fromAddr, toAddr, rawMessage string) bool {
	fmt.Printf("  [LEGACY SMTP %s] from=%s to=%s msg=%q\n",
		l.smtpHost, fromAddr, toAddr, rawMessage[:min(len(rawMessage), 40)])
	return true
}

// LegacyEmailAdapter adapts the old PostMail API to MessageSender.
type LegacyEmailAdapter struct {
	legacy *LegacyEmailSystem
	from   string
}

func (a *LegacyEmailAdapter) Send(to, subject, body string) error {
	raw := fmt.Sprintf("Subject: %s\r\n\r\n%s", subject, body)
	if !a.legacy.PostMail(a.from, to, raw) {
		return fmt.Errorf("legacy email failed for %s", to)
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─────────────────────────────────────────────────────────────────────────────
// DECORATOR
//
// Wraps an interface value and adds behaviour before/after delegation.
// In Go: a struct that holds the wrapped interface value and satisfies the
// same interface. Chain decorators to compose cross-cutting concerns.
// ─────────────────────────────────────────────────────────────────────────────

// ── Logging decorator ──────────────────────────────────────────────────────────

type loggingMessageSender struct {
	inner  MessageSender
	prefix string
}

func WithLogging(inner MessageSender, prefix string) MessageSender {
	return &loggingMessageSender{inner: inner, prefix: prefix}
}

func (l *loggingMessageSender) Send(to, subject, body string) error {
	start := time.Now()
	err := l.inner.Send(to, subject, body)
	elapsed := time.Since(start).Truncate(time.Microsecond)
	status := "ok"
	if err != nil {
		status = "err=" + err.Error()
	}
	fmt.Printf("  [%s LOG] to=%s subj=%q elapsed=%s status=%s\n",
		l.prefix, to, subject, elapsed, status)
	return err
}

// ── Retry decorator ────────────────────────────────────────────────────────────

type retryMessageSender struct {
	inner   MessageSender
	maxTries int
}

func WithRetry(inner MessageSender, maxTries int) MessageSender {
	return &retryMessageSender{inner: inner, maxTries: maxTries}
}

func (r *retryMessageSender) Send(to, subject, body string) error {
	var err error
	for attempt := 1; attempt <= r.maxTries; attempt++ {
		err = r.inner.Send(to, subject, body)
		if err == nil {
			return nil
		}
		fmt.Printf("  [RETRY] attempt %d/%d failed: %v\n", attempt, r.maxTries, err)
	}
	return fmt.Errorf("all %d attempts failed: %w", r.maxTries, err)
}

// ── Rate-limit decorator ───────────────────────────────────────────────────────

type rateLimitedSender struct {
	inner   MessageSender
	sent    int
	maxRate int
}

func WithRateLimit(inner MessageSender, maxPerBatch int) MessageSender {
	return &rateLimitedSender{inner: inner, maxRate: maxPerBatch}
}

func (r *rateLimitedSender) Send(to, subject, body string) error {
	if r.sent >= r.maxRate {
		return fmt.Errorf("rate limit exceeded (%d/%d)", r.sent, r.maxRate)
	}
	r.sent++
	return r.inner.Send(to, subject, body)
}

// ── io.Writer decorators ───────────────────────────────────────────────────────

type Writer interface {
	Write(p []byte) (n int, err error)
}

type uppercaseWriter struct{ inner Writer }

func (u *uppercaseWriter) Write(p []byte) (int, error) {
	return u.inner.Write([]byte(strings.ToUpper(string(p))))
}

type prefixWriter struct {
	inner  Writer
	prefix string
}

func (pw *prefixWriter) Write(p []byte) (int, error) {
	line := fmt.Sprintf("%s%s", pw.prefix, string(p))
	return pw.inner.Write([]byte(line))
}

type stdoutWriter struct{}

func (s stdoutWriter) Write(p []byte) (int, error) {
	fmt.Print(string(p))
	return len(p), nil
}

func main() {
	fmt.Println("=== Adapter: unified MessageSender ===")
	senders := []MessageSender{
		&SMSAdapter{client: &ExternalSMSClient{apiKey: "sk_live_abc123"}},
		&LegacyEmailAdapter{
			legacy: &LegacyEmailSystem{smtpHost: "mail.example.com"},
			from:   "no-reply@example.com",
		},
	}
	for _, s := range senders {
		_ = s.Send("+15551234567", "Alert", "Server CPU > 90%")
	}

	fmt.Println()
	fmt.Println("=== Decorator: logging + retry chain ===")
	base := &SMSAdapter{client: &ExternalSMSClient{apiKey: "sk_live_abc123"}}
	decorated := WithLogging(WithRetry(base, 3), "SMS")
	_ = decorated.Send("+15559876543", "Welcome", "Your account is ready.")

	fmt.Println()
	fmt.Println("=== Decorator: rate limiter ===")
	limited := WithRateLimit(base, 2)
	for i := 1; i <= 4; i++ {
		err := limited.Send("+15550000001", fmt.Sprintf("msg %d", i), "body")
		if err != nil {
			fmt.Printf("  send %d: %v\n", i, err)
		}
	}

	fmt.Println()
	fmt.Println("=== Decorator: io.Writer chain ===")
	var w Writer = stdoutWriter{}
	w = &prefixWriter{inner: w, prefix: ">> "}
	w = &uppercaseWriter{inner: w}
	_, _ = w.Write([]byte("hello from decorated writer\n"))
	_, _ = w.Write([]byte("each write gets prefixed and uppercased\n"))
}
