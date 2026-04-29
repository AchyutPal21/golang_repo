// FILE: book/part3_designing_software/chapter31_creational_patterns/examples/01_factory_builder/main.go
// CHAPTER: 31 — Creational Patterns
// TOPIC: Factory Method and Builder patterns in idiomatic Go.
//
// Run (from the chapter folder):
//   go run ./examples/01_factory_builder

package main

import (
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// FACTORY METHOD
//
// A factory function returns an interface. The caller works with the interface;
// the concrete type is encapsulated. In Go, this is a plain constructor that
// returns an interface rather than a concrete pointer.
// ─────────────────────────────────────────────────────────────────────────────

type Logger interface {
	Log(level, msg string)
}

// Three concrete logger types — all returned through the Logger interface.

type consoleLogger struct{ prefix string }

func (c *consoleLogger) Log(level, msg string) {
	fmt.Printf("[%s] %s: %s\n", c.prefix, strings.ToUpper(level), msg)
}

type jsonLogger struct{}

func (j *jsonLogger) Log(level, msg string) {
	fmt.Printf(`{"level":%q,"msg":%q,"ts":%q}`+"\n",
		level, msg, time.Now().UTC().Format(time.RFC3339))
}

type noopLogger struct{}

func (noopLogger) Log(_, _ string) {}

// NewLogger is the factory — callers receive a Logger, never a concrete type.
func NewLogger(format string) Logger {
	switch format {
	case "json":
		return &jsonLogger{}
	case "noop":
		return noopLogger{}
	default:
		return &consoleLogger{prefix: "APP"}
	}
}

// Parameterised factory: returns a family of related objects.
type Notifier interface {
	Notify(recipient, message string) error
}

type emailNotifier struct{ domain string }

func (e *emailNotifier) Notify(to, msg string) error {
	fmt.Printf("  [EMAIL @%s] to=%s  msg=%q\n", e.domain, to, msg)
	return nil
}

type smsNotifier struct{ shortCode string }

func (s *smsNotifier) Notify(to, msg string) error {
	fmt.Printf("  [SMS %s] to=%s  msg=%q\n", s.shortCode, to, msg)
	return nil
}

type slackNotifier struct{ workspace string }

func (s *slackNotifier) Notify(to, msg string) error {
	fmt.Printf("  [SLACK %s] @%s  msg=%q\n", s.workspace, to, msg)
	return nil
}

type NotifierConfig struct {
	Kind      string // "email" | "sms" | "slack"
	Domain    string
	ShortCode string
	Workspace string
}

func NewNotifier(cfg NotifierConfig) (Notifier, error) {
	switch cfg.Kind {
	case "email":
		return &emailNotifier{domain: cfg.Domain}, nil
	case "sms":
		return &smsNotifier{shortCode: cfg.ShortCode}, nil
	case "slack":
		return &slackNotifier{workspace: cfg.Workspace}, nil
	default:
		return nil, fmt.Errorf("unknown notifier kind: %q", cfg.Kind)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// BUILDER
//
// The Builder pattern constructs a complex object step by step.
// In Go: a mutable builder struct with method chaining, validated in Build().
// ─────────────────────────────────────────────────────────────────────────────

// HTTPRequest is the complex object we are building.
type HTTPRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
	Timeout time.Duration
}

func (r HTTPRequest) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s %s (timeout=%s)\n", r.Method, r.URL, r.Timeout)
	for k, v := range r.Headers {
		fmt.Fprintf(&sb, "  %s: %s\n", k, v)
	}
	if r.Body != "" {
		fmt.Fprintf(&sb, "  body: %s\n", r.Body)
	}
	return sb.String()
}

// HTTPRequestBuilder accumulates fields; Build() validates and returns the product.
type HTTPRequestBuilder struct {
	req HTTPRequest
	err error
}

func NewRequest(method, url string) *HTTPRequestBuilder {
	if method == "" {
		return &HTTPRequestBuilder{err: fmt.Errorf("method cannot be empty")}
	}
	if url == "" {
		return &HTTPRequestBuilder{err: fmt.Errorf("url cannot be empty")}
	}
	return &HTTPRequestBuilder{req: HTTPRequest{
		Method:  strings.ToUpper(method),
		URL:     url,
		Headers: make(map[string]string),
		Timeout: 30 * time.Second,
	}}
}

func (b *HTTPRequestBuilder) WithHeader(key, value string) *HTTPRequestBuilder {
	if b.err != nil {
		return b
	}
	b.req.Headers[key] = value
	return b
}

func (b *HTTPRequestBuilder) WithBearerToken(token string) *HTTPRequestBuilder {
	return b.WithHeader("Authorization", "Bearer "+token)
}

func (b *HTTPRequestBuilder) WithJSON(body string) *HTTPRequestBuilder {
	if b.err != nil {
		return b
	}
	b.req.Headers["Content-Type"] = "application/json"
	b.req.Body = body
	return b
}

func (b *HTTPRequestBuilder) WithTimeout(d time.Duration) *HTTPRequestBuilder {
	if b.err != nil {
		return b
	}
	if d <= 0 {
		b.err = fmt.Errorf("timeout must be positive")
		return b
	}
	b.req.Timeout = d
	return b
}

func (b *HTTPRequestBuilder) Build() (HTTPRequest, error) {
	if b.err != nil {
		return HTTPRequest{}, b.err
	}
	return b.req, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// ABSTRACT FACTORY
//
// An interface whose methods are factory methods — creates families of
// related objects without specifying concrete classes.
// ─────────────────────────────────────────────────────────────────────────────

type Button interface {
	Render() string
	OnClick() string
}

type TextInput interface {
	Render() string
	Value() string
}

// UIFactory creates a family of widgets for one theme.
type UIFactory interface {
	NewButton(label string) Button
	NewTextInput(placeholder string) TextInput
}

// Light theme family.
type lightButton struct{ label string }

func (b lightButton) Render() string  { return fmt.Sprintf("[  %s  ]", b.label) }
func (b lightButton) OnClick() string { return fmt.Sprintf("light-click: %s", b.label) }

type lightInput struct{ placeholder string }

func (i lightInput) Render() string { return fmt.Sprintf("____%s____", i.placeholder) }
func (i lightInput) Value() string  { return "" }

type LightUIFactory struct{}

func (LightUIFactory) NewButton(label string) Button         { return lightButton{label} }
func (LightUIFactory) NewTextInput(placeholder string) TextInput { return lightInput{placeholder} }

// Dark theme family.
type darkButton struct{ label string }

func (b darkButton) Render() string  { return fmt.Sprintf("▐  %s  ▌", b.label) }
func (b darkButton) OnClick() string { return fmt.Sprintf("dark-click: %s", b.label) }

type darkInput struct{ placeholder string }

func (i darkInput) Render() string { return fmt.Sprintf("███%s███", i.placeholder) }
func (i darkInput) Value() string  { return "" }

type DarkUIFactory struct{}

func (DarkUIFactory) NewButton(label string) Button          { return darkButton{label} }
func (DarkUIFactory) NewTextInput(placeholder string) TextInput { return darkInput{placeholder} }

func renderLoginForm(factory UIFactory) {
	emailInput := factory.NewTextInput("email")
	passInput := factory.NewTextInput("password")
	submitBtn := factory.NewButton("Sign In")
	fmt.Printf("  %s\n  %s\n  %s\n  → %s\n",
		emailInput.Render(), passInput.Render(), submitBtn.Render(), submitBtn.OnClick())
}

func main() {
	fmt.Println("=== Factory Method: logger ===")
	for _, format := range []string{"console", "json", "noop"} {
		log := NewLogger(format)
		log.Log("info", "application started (format="+format+")")
	}

	fmt.Println()
	fmt.Println("=== Parameterised Factory: notifiers ===")
	configs := []NotifierConfig{
		{Kind: "email", Domain: "example.com"},
		{Kind: "sms", ShortCode: "55555"},
		{Kind: "slack", Workspace: "myteam"},
	}
	for _, cfg := range configs {
		n, err := NewNotifier(cfg)
		if err != nil {
			fmt.Println("error:", err)
			continue
		}
		_ = n.Notify("alice", "hello from "+cfg.Kind)
	}
	_, err := NewNotifier(NotifierConfig{Kind: "unknown"})
	fmt.Println("  unknown kind error:", err)

	fmt.Println()
	fmt.Println("=== Builder: HTTP request ===")
	req, err := NewRequest("POST", "https://api.example.com/orders").
		WithBearerToken("tok_abc123").
		WithJSON(`{"sku":"WIDGET","qty":3}`).
		WithTimeout(10 * time.Second).
		Build()
	if err != nil {
		fmt.Println("build error:", err)
	} else {
		fmt.Print(req)
	}

	_, err = NewRequest("", "https://api.example.com").Build()
	fmt.Println("  empty method error:", err)

	_, err = NewRequest("GET", "https://api.example.com").WithTimeout(-1).Build()
	fmt.Println("  bad timeout error:", err)

	fmt.Println()
	fmt.Println("=== Abstract Factory: UI themes ===")
	fmt.Println("  Light theme:")
	renderLoginForm(LightUIFactory{})
	fmt.Println("  Dark theme:")
	renderLoginForm(DarkUIFactory{})
}
