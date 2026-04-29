// FILE: book/part3_designing_software/chapter26_oop_in_go/examples/02_composition_over_inheritance/main.go
// CHAPTER: 26 — OOP in Go
// TOPIC: Why composition beats inheritance for extensibility.
//        A notification system built without a class hierarchy.
//
// Run (from the chapter folder):
//   go run ./examples/02_composition_over_inheritance

package main

import (
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// PROBLEM: Send notifications via multiple channels.
//
// OOP approach: abstract class Notifier, subclass EmailNotifier,
// SMSNotifier, SlackNotifier — fragile when a notification needs
// to go to multiple channels simultaneously.
//
// Go approach: small interfaces + composition.
// ─────────────────────────────────────────────────────────────────────────────

type Message struct {
	To      string
	Subject string
	Body    string
	SentAt  time.Time
}

// Sender is the single-method interface every channel implements.
type Sender interface {
	Send(m Message) error
}

// ─── Concrete senders ────────────────────────────────────────────────────────

type EmailSender struct{ From string }

func (e *EmailSender) Send(m Message) error {
	fmt.Printf("[EMAIL] from=%s to=%s subj=%q\n", e.From, m.To, m.Subject)
	return nil
}

type SMSSender struct{ ShortCode string }

func (s *SMSSender) Send(m Message) error {
	body := m.Body
	if len(body) > 40 {
		body = body[:40] + "…"
	}
	fmt.Printf("[SMS]   code=%s to=%s body=%q\n", s.ShortCode, m.To, body)
	return nil
}

type SlackSender struct{ Channel string }

func (s *SlackSender) Send(m Message) error {
	fmt.Printf("[SLACK] #%s → %q\n", s.Channel, m.Subject)
	return nil
}

// ─── Composable wrappers ─────────────────────────────────────────────────────

// MultiSender sends via all senders, collecting errors.
type MultiSender struct{ senders []Sender }

func NewMultiSender(senders ...Sender) *MultiSender {
	return &MultiSender{senders: senders}
}

func (ms *MultiSender) Send(m Message) error {
	var errs []string
	for _, s := range ms.senders {
		if err := s.Send(m); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("send errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// LoggingSender wraps any Sender and logs each send.
type LoggingSender struct {
	inner Sender
	log   []string
}

func (l *LoggingSender) Send(m Message) error {
	err := l.inner.Send(m)
	status := "ok"
	if err != nil {
		status = err.Error()
	}
	l.log = append(l.log, fmt.Sprintf("to=%s status=%s", m.To, status))
	return err
}

func (l *LoggingSender) Log() []string { return l.log }

// RetryingSender retries up to maxRetries times on error.
type RetryingSender struct {
	inner      Sender
	maxRetries int
}

func (r *RetryingSender) Send(m Message) error {
	var err error
	for attempt := range r.maxRetries + 1 {
		err = r.inner.Send(m)
		if err == nil {
			return nil
		}
		fmt.Printf("[RETRY] attempt %d failed: %v\n", attempt+1, err)
	}
	return fmt.Errorf("all %d attempts failed: %w", r.maxRetries+1, err)
}

// ─── Notification service ─────────────────────────────────────────────────────

type NotificationService struct {
	sender Sender
}

func NewNotificationService(sender Sender) *NotificationService {
	return &NotificationService{sender: sender}
}

func (n *NotificationService) Notify(to, subject, body string) error {
	return n.sender.Send(Message{
		To:      to,
		Subject: subject,
		Body:    body,
		SentAt:  time.Now(),
	})
}

func main() {
	// ── single sender ──
	fmt.Println("=== single email sender ===")
	svc := NewNotificationService(&EmailSender{From: "noreply@example.com"})
	_ = svc.Notify("alice@example.com", "Welcome!", "Hello Alice")

	fmt.Println()

	// ── multi-channel without any class hierarchy ──
	fmt.Println("=== multi-channel ===")
	multi := NewMultiSender(
		&EmailSender{From: "noreply@example.com"},
		&SMSSender{ShortCode: "12345"},
		&SlackSender{Channel: "alerts"},
	)
	svc2 := NewNotificationService(multi)
	_ = svc2.Notify("+1-555-0100", "Alert", "Server CPU > 90% for 5 minutes")

	fmt.Println()

	// ── composed wrappers: logging around multi ──
	fmt.Println("=== with logging wrapper ===")
	logged := &LoggingSender{inner: multi}
	svc3 := NewNotificationService(logged)
	_ = svc3.Notify("bob@example.com", "Report", "Monthly report attached")
	fmt.Println("audit log:", logged.Log())

	fmt.Println()

	// ── key insight ──
	fmt.Println("Key insight:")
	fmt.Println("  EmailSender, SMSSender, SlackSender know nothing about each other.")
	fmt.Println("  MultiSender, LoggingSender, RetryingSender are reusable with ANY Sender.")
	fmt.Println("  No abstract class. No hierarchy. Add a WebhookSender: just implement Send().")
}
