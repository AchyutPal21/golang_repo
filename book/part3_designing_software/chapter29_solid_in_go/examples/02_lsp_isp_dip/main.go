// FILE: book/part3_designing_software/chapter29_solid_in_go/examples/02_lsp_isp_dip/main.go
// CHAPTER: 29 — SOLID in Go
// TOPIC: Liskov Substitution Principle, Interface Segregation Principle,
//        and Dependency Inversion Principle in idiomatic Go.
//
// Run (from the chapter folder):
//   go run ./examples/02_lsp_isp_dip

package main

import (
	"errors"
	"fmt"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// L — Liskov Substitution Principle
//
// Subtypes must be substitutable for their base types without altering
// correctness. In Go: any type satisfying an interface must honour the
// full contract the interface implies — not just the method signatures.
// ─────────────────────────────────────────────────────────────────────────────

type ReadWriter interface {
	Read() string
	Write(s string) error
}

// ── BAD: ReadOnlyBuffer claims to implement ReadWriter but panics on Write ───

type badReadOnlyBuffer struct{ content string }

func (b *badReadOnlyBuffer) Read() string { return b.content }
func (b *badReadOnlyBuffer) Write(_ string) error {
	panic("this buffer is read-only") // violates LSP — surprises callers expecting ReadWriter
}

// ── GOOD: split the interface so read-only types never promise Write ──────────

type Reader interface{ Read() string }
type Writer interface{ Write(s string) error }

type readOnlyBuffer struct{ content string }

func (b *readOnlyBuffer) Read() string { return b.content }

type readWriteBuffer struct{ content strings.Builder }

func (b *readWriteBuffer) Read() string      { return b.content.String() }
func (b *readWriteBuffer) Write(s string) error { b.content.WriteString(s); return nil }

// useReader accepts any Reader — readOnlyBuffer and readWriteBuffer both qualify.
func useReader(r Reader) {
	fmt.Println("  read:", r.Read())
}

// useWriter only accepts the writable variant.
func useWriter(w Writer, text string) {
	if err := w.Write(text); err != nil {
		fmt.Println("  write error:", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// I — Interface Segregation Principle
//
// Clients should not be forced to depend on methods they do not use.
// In Go: keep interfaces narrow; compose them when a single type must
// satisfy multiple roles. Consumer-side interfaces naturally enforce ISP.
// ─────────────────────────────────────────────────────────────────────────────

// ── BAD: fat interface forces all implementors to provide every method ────────

type badStorage interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Delete(key string) error
	Flush() error
	Stats() map[string]int
	Backup(dest string) error
}

// Any type that only does reads must provide stub panics for Set/Delete/Flush/Backup.

// ── GOOD: narrow interfaces, each with a single reason to exist ───────────────

type Getter interface {
	Get(key string) (string, error)
}

type Setter interface {
	Set(key, value string) error
}

type Deleter interface {
	Delete(key string) error
}

// Composed only when a consumer genuinely needs both.
type ReadWriteStorage interface {
	Getter
	Setter
}

// memStore satisfies all interfaces — but each consumer only sees its slice.
type memStore struct{ data map[string]string }

func newMemStore() *memStore                    { return &memStore{data: make(map[string]string)} }
func (m *memStore) Get(k string) (string, error) {
	v, ok := m.data[k]
	if !ok {
		return "", fmt.Errorf("key %q not found", k)
	}
	return v, nil
}
func (m *memStore) Set(k, v string) error  { m.data[k] = v; return nil }
func (m *memStore) Delete(k string) error  { delete(m.data, k); return nil }

// CacheService only reads — it receives Getter, not the full store.
type CacheService struct{ cache Getter }

func (c *CacheService) Lookup(key string) string {
	v, err := c.cache.Get(key)
	if err != nil {
		return "<miss>"
	}
	return v
}

// ConfigWriter only writes — it receives Setter.
type ConfigWriter struct{ store Setter }

func (cw *ConfigWriter) Apply(key, value string) {
	if err := cw.store.Set(key, value); err != nil {
		fmt.Println("  config write error:", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// D — Dependency Inversion Principle
//
// High-level modules should not depend on low-level modules.
// Both should depend on abstractions.
// In Go: high-level packages define the interfaces they need; low-level
// packages provide concrete implementations that satisfy those interfaces.
// ─────────────────────────────────────────────────────────────────────────────

// ── BAD: high-level NotificationService imports low-level EmailSender directly ─

type badEmailSender struct{}

func (e *badEmailSender) SendEmail(to, body string) {
	fmt.Printf("  [EMAIL] to=%s body=%q\n", to, body)
}

type badNotificationService struct {
	sender *badEmailSender // concrete dependency — can't swap, can't test
}

func (s *badNotificationService) Notify(user, message string) {
	s.sender.SendEmail(user, message)
}

// ── GOOD: NotificationService depends on its own Notifier interface ───────────

// Notifier is defined in the high-level layer — not in any concrete package.
type Notifier interface {
	Notify(recipient, message string) error
}

// Low-level concrete implementations:

type emailNotifier struct{ from string }

func (e *emailNotifier) Notify(recipient, message string) error {
	fmt.Printf("  [EMAIL] from=%s to=%s msg=%q\n", e.from, recipient, message)
	return nil
}

type smsNotifier struct{ shortCode string }

func (s *smsNotifier) Notify(recipient, message string) error {
	fmt.Printf("  [SMS]   code=%s to=%s msg=%q\n", s.shortCode, recipient, message)
	return nil
}

type slackNotifier struct{ channel string }

func (s *slackNotifier) Notify(recipient, message string) error {
	fmt.Printf("  [SLACK] #%s @%s msg=%q\n", s.channel, recipient, message)
	return nil
}

// fanOutNotifier composes multiple Notifiers — still satisfies Notifier itself.
type fanOutNotifier struct{ notifiers []Notifier }

func (f *fanOutNotifier) Notify(recipient, message string) error {
	var errs []string
	for _, n := range f.notifiers {
		if err := n.Notify(recipient, message); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// AlertService is the high-level module — depends only on Notifier.
type AlertService struct{ notifier Notifier }

func NewAlertService(n Notifier) *AlertService { return &AlertService{notifier: n} }

func (a *AlertService) SendAlert(userID, msg string) error {
	return a.notifier.Notify(userID, msg)
}

func main() {
	// ── LSP ───────────────────────────────────────────────────────────────────
	fmt.Println("=== LSP: bad (panics on Write) ===")
	badBuf := &badReadOnlyBuffer{content: "hello"}
	fmt.Println("  read:", badBuf.Read())
	fmt.Println("  (Write would panic — LSP violated)")

	fmt.Println()
	fmt.Println("=== LSP: good (split interfaces) ===")
	rob := &readOnlyBuffer{content: "read-only data"}
	rwb := &readWriteBuffer{}
	_ = rwb.Write("hello")
	_ = rwb.Write(" world")
	useReader(rob) // both satisfy Reader
	useReader(rwb)
	useWriter(rwb, " — appended") // only readWriteBuffer satisfies Writer
	useReader(rwb)

	// ── ISP ───────────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("=== ISP: fat interface (bad) ===")
	fmt.Println("  (any read-only impl must stub Set/Delete/Flush/Backup — skipped)")

	fmt.Println()
	fmt.Println("=== ISP: narrow interfaces (good) ===")
	store := newMemStore()
	_ = store.Set("theme", "dark")
	_ = store.Set("lang", "en")

	cache := &CacheService{cache: store}   // only sees Getter
	config := &ConfigWriter{store: store}  // only sees Setter

	fmt.Println("  lookup theme:", cache.Lookup("theme"))
	fmt.Println("  lookup missing:", cache.Lookup("missing"))
	config.Apply("lang", "fr")
	fmt.Println("  lookup lang after update:", cache.Lookup("lang"))

	// ── DIP ───────────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("=== DIP: bad (concrete dependency) ===")
	badSvc := &badNotificationService{sender: &badEmailSender{}}
	badSvc.Notify("alice", "hello from bad service")

	fmt.Println()
	fmt.Println("=== DIP: good (depends on abstraction) ===")
	fan := &fanOutNotifier{notifiers: []Notifier{
		&emailNotifier{from: "alerts@example.com"},
		&smsNotifier{shortCode: "12345"},
		&slackNotifier{channel: "ops-alerts"},
	}}
	alertSvc := NewAlertService(fan)
	if err := alertSvc.SendAlert("alice", "disk usage > 90%"); err != nil {
		fmt.Println("alert error:", err)
	}
}
