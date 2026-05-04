// FILE: book/part6_production_engineering/chapter89_logging_strategy/examples/02_pii_levels/main.go
// CHAPTER: 89 — Logging Strategy
// TOPIC: PII scrubbing middleware, log-level-aware field masking,
//        and writing a custom slog.Handler that sanitises sensitive fields.
//
// Run:
//   go run ./part6_production_engineering/chapter89_logging_strategy/examples/02_pii_levels/

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// PII PATTERNS — fields and value regexes to redact
// ─────────────────────────────────────────────────────────────────────────────

// sensitiveKeys are field names whose values must always be masked.
var sensitiveKeys = map[string]bool{
	"password":      true,
	"token":         true,
	"secret":        true,
	"authorization": true,
	"credit_card":   true,
	"ssn":           true,
	"cvv":           true,
}

// piiPatterns are regexes that redact values regardless of field name.
var piiPatterns = []*regexp.Regexp{
	// Email
	regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`),
	// Phone (E.164-ish)
	regexp.MustCompile(`\+?[0-9]{1,3}[-.\s]?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}`),
	// Credit card (Luhn-shape 4×4 groups)
	regexp.MustCompile(`\b(?:\d[ -]?){13,16}\b`),
}

const redacted = "[REDACTED]"

// scrubValue replaces PII in a string value.
func scrubValue(v string) string {
	for _, re := range piiPatterns {
		v = re.ReplaceAllString(v, redacted)
	}
	return v
}

// ─────────────────────────────────────────────────────────────────────────────
// SCRUBBING HANDLER — wraps any slog.Handler
// ─────────────────────────────────────────────────────────────────────────────

type ScrubbingHandler struct {
	inner slog.Handler
}

func NewScrubbingHandler(inner slog.Handler) *ScrubbingHandler {
	return &ScrubbingHandler{inner: inner}
}

func (h *ScrubbingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *ScrubbingHandler) Handle(ctx context.Context, r slog.Record) error {
	safe := slog.NewRecord(r.Time, r.Level, scrubValue(r.Message), r.PC)
	r.Attrs(func(a slog.Attr) bool {
		safe.AddAttrs(h.scrubAttr(a))
		return true
	})
	return h.inner.Handle(ctx, safe)
}

func (h *ScrubbingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	scrubbed := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		scrubbed[i] = h.scrubAttr(a)
	}
	return &ScrubbingHandler{inner: h.inner.WithAttrs(scrubbed)}
}

func (h *ScrubbingHandler) WithGroup(name string) slog.Handler {
	return &ScrubbingHandler{inner: h.inner.WithGroup(name)}
}

func (h *ScrubbingHandler) scrubAttr(a slog.Attr) slog.Attr {
	key := strings.ToLower(a.Key)
	if sensitiveKeys[key] {
		return slog.String(a.Key, redacted)
	}
	if a.Value.Kind() == slog.KindString {
		return slog.String(a.Key, scrubValue(a.Value.String()))
	}
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		scrubbed := make([]any, 0, len(attrs)*2)
		for _, ga := range attrs {
			sa := h.scrubAttr(ga)
			scrubbed = append(scrubbed, sa.Key, sa.Value.Any())
		}
		return slog.Group(a.Key, scrubbed...)
	}
	return a
}

// ─────────────────────────────────────────────────────────────────────────────
// LEVEL-AWARE FIELD MASKING — debug logs include more detail than info
// ─────────────────────────────────────────────────────────────────────────────

// userFields returns log attributes for a user.
// At DEBUG level (e.g., local dev), it includes the email.
// At INFO+ (production), the email is always masked.
func userFields(level slog.Level, userID int, email string) []any {
	fields := []any{"user_id", userID}
	if level <= slog.LevelDebug {
		fields = append(fields, "email", email) // only in debug
	} else {
		fields = append(fields, "email_domain", emailDomain(email))
	}
	return fields
}

func emailDomain(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

// ─────────────────────────────────────────────────────────────────────────────
// LOG AGGREGATION PATTERNS
// ─────────────────────────────────────────────────────────────────────────────

const aggregationRef = `
Log aggregation patterns:

  stdout → Fluentd/Fluent Bit → Elasticsearch/OpenSearch → Kibana
  stdout → Loki → Grafana
  stdout → CloudWatch Logs → CloudWatch Insights
  stdout → Datadog Agent → Datadog Logs

JSON fields that every service should emit:
  time        — RFC3339Nano
  level       — debug/info/warn/error
  msg         — human-readable event name
  service     — "api-gateway", "order-service"
  version     — git SHA or semver
  env         — production/staging/development
  request_id  — propagated from X-Request-ID header
  trace_id    — OpenTelemetry trace ID (16 hex bytes)

Structured query examples (Elasticsearch/Loki):
  level="error" AND service="order-service"
  request_id="abc-123"
  duration_ms > 1000
`

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 89: PII Scrubbing & Log Levels ===")
	fmt.Println()

	// ── RAW vs SCRUBBED ───────────────────────────────────────────────────────
	fmt.Println("--- Raw string scrubbing ---")
	cases := []string{
		"user logged in: alice@example.com",
		"phone: +1 (555) 867-5309",
		"card: 4111 1111 1111 1111",
		"no PII in this string",
		"two PII: bob@test.com and 5551234567",
	}
	for _, c := range cases {
		fmt.Printf("  IN:  %s\n", c)
		fmt.Printf("  OUT: %s\n\n", scrubValue(c))
	}

	// ── SCRUBBING HANDLER ─────────────────────────────────────────────────────
	fmt.Println("--- ScrubbingHandler in action ---")
	inner := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().UTC().Format(time.RFC3339))
			}
			return a
		},
	})
	logger := slog.New(NewScrubbingHandler(inner))

	// These would normally leak PII:
	logger.Info("user registered",
		"email", "carol@example.com",
		"password", "s3cr3tP@ss",
		"token", "eyJhbGciOiJIUzI1NiJ9.xyz",
	)
	logger.Warn("login failed",
		"email", "dave@corp.io",
		"ip", "10.0.0.1",
		"attempt", 3,
	)
	logger.Error("payment declined",
		"credit_card", "4111111111111111",
		"amount", 99.99,
		"user_id", 42,
	)
	fmt.Println()

	// ── LEVEL-AWARE FIELDS ────────────────────────────────────────────────────
	fmt.Println("--- Level-aware field masking ---")
	debugLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	infoLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	email := "frank@example.com"
	debugLogger.Debug("user action (debug — includes email)",
		userFields(slog.LevelDebug, 101, email)...)
	infoLogger.Info("user action (info — domain only)",
		userFields(slog.LevelInfo, 101, email)...)
	fmt.Println()

	// ── SENSITIVE KEY LIST ────────────────────────────────────────────────────
	fmt.Println("--- Sensitive field keys ---")
	fmt.Printf("  %v\n", func() []string {
		var keys []string
		for k := range sensitiveKeys {
			keys = append(keys, k)
		}
		return keys
	}())
	fmt.Println()

	// ── AGGREGATION REFERENCE ─────────────────────────────────────────────────
	fmt.Println("--- Log aggregation patterns ---")
	fmt.Print(aggregationRef)
}
