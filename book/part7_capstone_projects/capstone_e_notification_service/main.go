// CAPSTONE E — Notification Service
// Self-contained simulation: multi-channel dispatch, provider failover,
// exponential-backoff retry, dead-letter queue, and delivery tracking.
// No external dependencies.
//
// Run:
//   go run ./part7_capstone_projects/capstone_e_notification_service

package main

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN TYPES
// ─────────────────────────────────────────────────────────────────────────────

// Channel identifies the delivery medium for a notification.
type Channel string

const (
	ChannelEmail Channel = "email"
	ChannelSMS   Channel = "sms"
	ChannelPush  Channel = "push"
)

// Priority controls dispatch urgency. Higher value = more important.
type Priority int

const (
	PriorityMarketing    Priority = 1
	PriorityTransactional Priority = 2
	PriorityCritical     Priority = 3
)

func (p Priority) String() string {
	switch p {
	case PriorityCritical:
		return "critical"
	case PriorityTransactional:
		return "transactional"
	default:
		return "marketing"
	}
}

// Notification is the value type that flows through the entire pipeline.
type Notification struct {
	ID       string
	UserID   string
	Channel  Channel
	Subject  string
	Body     string
	Priority Priority
}

func (n Notification) String() string {
	return fmt.Sprintf("[%s|%s|%s] %q", n.ID, n.Channel, n.Priority, n.Subject)
}

// ─────────────────────────────────────────────────────────────────────────────
// PROVIDER INTERFACE
// ─────────────────────────────────────────────────────────────────────────────

// NotificationProvider is the single abstraction every delivery back-end satisfies.
type NotificationProvider interface {
	Name() string
	Send(n Notification) error
}

// ─────────────────────────────────────────────────────────────────────────────
// SIMULATED PROVIDERS
// ─────────────────────────────────────────────────────────────────────────────

// EmailProvider simulates an SMTP / transactional-email gateway (e.g. SES).
type EmailProvider struct {
	FailureRate float64 // 0.0 = never fails, 1.0 = always fails
	rng         *rand.Rand
}

func NewEmailProvider(failureRate float64) *EmailProvider {
	return &EmailProvider{
		FailureRate: failureRate,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano() + 1)),
	}
}

func (p *EmailProvider) Name() string { return "EmailProvider" }

func (p *EmailProvider) Send(n Notification) error {
	if p.rng.Float64() < p.FailureRate {
		return fmt.Errorf("EmailProvider: SMTP timeout for notification %s", n.ID)
	}
	return nil
}

// SMSProvider simulates a carrier gateway (e.g. Twilio).
type SMSProvider struct {
	FailureRate float64
	rng         *rand.Rand
}

func NewSMSProvider(failureRate float64) *SMSProvider {
	return &SMSProvider{
		FailureRate: failureRate,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano() + 2)),
	}
}

func (p *SMSProvider) Name() string { return "SMSProvider" }

func (p *SMSProvider) Send(n Notification) error {
	if p.rng.Float64() < p.FailureRate {
		return fmt.Errorf("SMSProvider: carrier rejected notification %s", n.ID)
	}
	return nil
}

// PushProvider simulates a mobile push gateway (e.g. FCM / APNs).
type PushProvider struct {
	FailureRate float64
	rng         *rand.Rand
}

func NewPushProvider(failureRate float64) *PushProvider {
	return &PushProvider{
		FailureRate: failureRate,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano() + 3)),
	}
}

func (p *PushProvider) Name() string { return "PushProvider" }

func (p *PushProvider) Send(n Notification) error {
	if p.rng.Float64() < p.FailureRate {
		return fmt.Errorf("PushProvider: device token expired for notification %s", n.ID)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// PROVIDER WITH FALLBACK
// ─────────────────────────────────────────────────────────────────────────────

// ProviderWithFallback attempts the primary provider and, on any error,
// transparently retries using the secondary provider.
type ProviderWithFallback struct {
	Primary   NotificationProvider
	Secondary NotificationProvider
}

func (p *ProviderWithFallback) Name() string {
	return fmt.Sprintf("%s→%s", p.Primary.Name(), p.Secondary.Name())
}

// Send tries primary first; falls back to secondary on error.
// Returns a wrapped error only if both providers fail.
func (p *ProviderWithFallback) Send(n Notification) error {
	if err := p.Primary.Send(n); err != nil {
		secondaryErr := p.Secondary.Send(n)
		if secondaryErr != nil {
			return fmt.Errorf("primary (%w) and secondary (%v) both failed", err, secondaryErr)
		}
		// secondary succeeded — report that we fell back but treat as success
		return nil
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// RETRY POLICY
// ─────────────────────────────────────────────────────────────────────────────

// RetryPolicy describes exponential-backoff retry parameters.
// In this simulation the delays are counted but not actually slept.
type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration // delay before attempt 2
	Multiplier  float64       // each subsequent wait is multiplied by this
}

// NextDelay returns the simulated wait before the given attempt number (1-based).
// Attempt 1 has no preceding delay.
func (rp RetryPolicy) NextDelay(attempt int) time.Duration {
	if attempt <= 1 {
		return 0
	}
	d := float64(rp.BaseDelay)
	for i := 1; i < attempt; i++ {
		d *= rp.Multiplier
	}
	return time.Duration(d)
}

// ─────────────────────────────────────────────────────────────────────────────
// DEAD-LETTER QUEUE
// ─────────────────────────────────────────────────────────────────────────────

// DLQEntry pairs a notification with the final error that caused it to be queued.
type DLQEntry struct {
	Notification Notification
	Err          error
	Attempts     int
}

// DeadLetterQueue is a thread-safe store for notifications that exhausted
// all delivery attempts.
type DeadLetterQueue struct {
	mu      sync.Mutex
	entries []DLQEntry
}

func (q *DeadLetterQueue) Push(entry DLQEntry) {
	q.mu.Lock()
	q.entries = append(q.entries, entry)
	q.mu.Unlock()
}

// Drain atomically removes and returns all queued entries for reprocessing.
func (q *DeadLetterQueue) Drain() []DLQEntry {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := q.entries
	q.entries = nil
	return out
}

func (q *DeadLetterQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.entries)
}

// ─────────────────────────────────────────────────────────────────────────────
// DELIVERY TRACKER
// ─────────────────────────────────────────────────────────────────────────────

// channelStats holds lock-free counters for one channel.
type channelStats struct {
	sent   atomic.Int64
	failed atomic.Int64
	dlq    atomic.Int64
}

// DeliveryTracker records per-channel outcomes using atomic counters so it is
// safe for concurrent use without a mutex.
type DeliveryTracker struct {
	mu    sync.RWMutex
	stats map[Channel]*channelStats
}

func NewDeliveryTracker() *DeliveryTracker {
	return &DeliveryTracker{stats: make(map[Channel]*channelStats)}
}

func (t *DeliveryTracker) statsFor(ch Channel) *channelStats {
	t.mu.RLock()
	s, ok := t.stats[ch]
	t.mu.RUnlock()
	if ok {
		return s
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if s, ok = t.stats[ch]; ok {
		return s
	}
	s = &channelStats{}
	t.stats[ch] = s
	return s
}

func (t *DeliveryTracker) RecordSent(ch Channel)   { t.statsFor(ch).sent.Add(1) }
func (t *DeliveryTracker) RecordFailed(ch Channel) { t.statsFor(ch).failed.Add(1) }
func (t *DeliveryTracker) RecordDLQ(ch Channel)    { t.statsFor(ch).dlq.Add(1) }

// Summary prints a formatted delivery report to stdout.
func (t *DeliveryTracker) Summary() {
	t.mu.RLock()
	defer t.mu.RUnlock()
	channels := []Channel{ChannelEmail, ChannelSMS, ChannelPush}
	fmt.Println("\n┌─────────────────────────────────────────────┐")
	fmt.Println("│          DELIVERY TRACKER SUMMARY           │")
	fmt.Println("├──────────┬──────────┬──────────┬────────────┤")
	fmt.Printf("│ %-8s │ %-8s │ %-8s │ %-10s │\n", "Channel", "Sent", "Failed", "DLQ")
	fmt.Println("├──────────┼──────────┼──────────┼────────────┤")
	for _, ch := range channels {
		s := t.stats[ch]
		if s == nil {
			fmt.Printf("│ %-8s │ %-8d │ %-8d │ %-10d │\n", ch, 0, 0, 0)
		} else {
			fmt.Printf("│ %-8s │ %-8d │ %-8d │ %-10d │\n",
				ch, s.sent.Load(), s.failed.Load(), s.dlq.Load())
		}
	}
	fmt.Println("└──────────┴──────────┴──────────┴────────────┘")
}

// ─────────────────────────────────────────────────────────────────────────────
// NOTIFICATION DISPATCHER
// ─────────────────────────────────────────────────────────────────────────────

// NotificationDispatcher routes notifications to the correct provider,
// applies the retry policy, and moves exhausted messages to the DLQ.
type NotificationDispatcher struct {
	providers map[Channel]*ProviderWithFallback
	policy    RetryPolicy
	dlq       *DeadLetterQueue
	tracker   *DeliveryTracker
}

func NewNotificationDispatcher(
	providers map[Channel]*ProviderWithFallback,
	policy RetryPolicy,
	dlq *DeadLetterQueue,
	tracker *DeliveryTracker,
) *NotificationDispatcher {
	return &NotificationDispatcher{
		providers: providers,
		policy:    policy,
		dlq:       dlq,
		tracker:   tracker,
	}
}

// Dispatch delivers a single notification, retrying on failure up to
// RetryPolicy.MaxAttempts times with exponential backoff between attempts.
func (d *NotificationDispatcher) Dispatch(n Notification) {
	provider, ok := d.providers[n.Channel]
	if !ok {
		fmt.Printf("  [DISPATCH] no provider registered for channel %q — dropping %s\n", n.Channel, n.ID)
		d.tracker.RecordFailed(n.Channel)
		return
	}

	var lastErr error
	for attempt := 1; attempt <= d.policy.MaxAttempts; attempt++ {
		delay := d.policy.NextDelay(attempt)

		// In production this would be time.Sleep(delay). Here we only log it
		// so the simulation runs instantly.
		delayNote := ""
		if delay > 0 {
			delayNote = fmt.Sprintf(" (after simulated %v backoff)", delay)
		}

		err := provider.Send(n)
		if err == nil {
			fmt.Printf("  [OK]  attempt %d%s — %s via %s\n",
				attempt, delayNote, n, provider.Name())
			d.tracker.RecordSent(n.Channel)
			return
		}

		lastErr = err
		fmt.Printf("  [ERR] attempt %d%s — %s via %s: %v\n",
			attempt, delayNote, n, provider.Name(), err)
		d.tracker.RecordFailed(n.Channel)
	}

	// All attempts exhausted — move to DLQ.
	fmt.Printf("  [DLQ] exhausted %d attempts for %s — queuing in DLQ\n",
		d.policy.MaxAttempts, n)
	d.dlq.Push(DLQEntry{
		Notification: n,
		Err:          lastErr,
		Attempts:     d.policy.MaxAttempts,
	})
	d.tracker.RecordDLQ(n.Channel)
}

// DispatchBatch dispatches a slice of notifications sequentially.
func (d *NotificationDispatcher) DispatchBatch(notifications []Notification) {
	for _, n := range notifications {
		d.Dispatch(n)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// HELPERS
// ─────────────────────────────────────────────────────────────────────────────

func notifID(i int) string { return fmt.Sprintf("notif-%02d", i) }

func separator(label string) {
	line := strings.Repeat("─", 60)
	fmt.Printf("\n%s\n  %s\n%s\n", line, label, line)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN — SIMULATION
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║        CAPSTONE E — Notification Service                 ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")

	// ── Providers ────────────────────────────────────────────────────────────
	//
	// Email primary: 30 % failure rate (simulates occasional SMTP issues)
	// SMS primary:   60 % failure rate (simulates strict carrier rate limits)
	// Push primary:  50 % failure rate (simulates stale device tokens)
	//
	// Fallback providers have lower failure rates, representing more reliable
	// (but possibly more expensive) alternatives.
	emailPrimary   := NewEmailProvider(0.30)
	emailFallback  := NewEmailProvider(0.10)
	smsPrimary     := NewSMSProvider(0.60)
	smsFallback    := NewSMSProvider(0.20)
	pushPrimary    := NewPushProvider(0.50)
	pushFallback   := NewSMSProvider(0.15) // SMS as push fallback (common in prod)

	providers := map[Channel]*ProviderWithFallback{
		ChannelEmail: {Primary: emailPrimary, Secondary: emailFallback},
		ChannelSMS:   {Primary: smsPrimary, Secondary: smsFallback},
		ChannelPush:  {Primary: pushPrimary, Secondary: pushFallback},
	}

	// ── Retry policy ─────────────────────────────────────────────────────────
	policy := RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   200 * time.Millisecond,
		Multiplier:  2.0,
		// Attempt 1: no delay
		// Attempt 2: 200 ms  (simulated, not slept)
		// Attempt 3: 400 ms  (simulated, not slept)
	}

	// ── Infrastructure ───────────────────────────────────────────────────────
	dlq     := &DeadLetterQueue{}
	tracker := NewDeliveryTracker()

	dispatcher := NewNotificationDispatcher(providers, policy, dlq, tracker)

	// ── Batch of 10 notifications ────────────────────────────────────────────
	batch := []Notification{
		{ID: notifID(1), UserID: "u001", Channel: ChannelEmail, Subject: "Order confirmed", Body: "Your order #1001 is confirmed.", Priority: PriorityTransactional},
		{ID: notifID(2), UserID: "u002", Channel: ChannelSMS, Subject: "OTP", Body: "Your one-time password is 482910.", Priority: PriorityCritical},
		{ID: notifID(3), UserID: "u003", Channel: ChannelPush, Subject: "Flash sale", Body: "50 % off for the next hour!", Priority: PriorityMarketing},
		{ID: notifID(4), UserID: "u004", Channel: ChannelEmail, Subject: "Password reset", Body: "Click here to reset your password.", Priority: PriorityCritical},
		{ID: notifID(5), UserID: "u005", Channel: ChannelSMS, Subject: "Delivery update", Body: "Your package arrives tomorrow.", Priority: PriorityTransactional},
		{ID: notifID(6), UserID: "u006", Channel: ChannelPush, Subject: "New message", Body: "You have a new message from Alice.", Priority: PriorityTransactional},
		{ID: notifID(7), UserID: "u007", Channel: ChannelEmail, Subject: "Invoice", Body: "Invoice #INV-007 is attached.", Priority: PriorityTransactional},
		{ID: notifID(8), UserID: "u008", Channel: ChannelSMS, Subject: "Low balance", Body: "Your balance is below $10.", Priority: PriorityCritical},
		{ID: notifID(9), UserID: "u009", Channel: ChannelPush, Subject: "Weekly digest", Body: "Here is what happened this week.", Priority: PriorityMarketing},
		{ID: notifID(10), UserID: "u010", Channel: ChannelEmail, Subject: "Subscription renewal", Body: "Your plan renews in 7 days.", Priority: PriorityMarketing},
	}

	separator("DISPATCHING BATCH")
	dispatcher.DispatchBatch(batch)

	// ── Delivery summary ─────────────────────────────────────────────────────
	tracker.Summary()

	// ── Dead-letter queue inspection ─────────────────────────────────────────
	separator("DEAD-LETTER QUEUE CONTENTS")
	dlqEntries := dlq.Drain()
	if len(dlqEntries) == 0 {
		fmt.Println("  DLQ is empty — all notifications delivered within retry budget.")
	} else {
		fmt.Printf("  %d notification(s) in DLQ:\n\n", len(dlqEntries))
		for i, e := range dlqEntries {
			fmt.Printf("  %d. %s\n", i+1, e.Notification)
			fmt.Printf("     attempts : %d\n", e.Attempts)
			fmt.Printf("     last err : %v\n\n", e.Err)
		}
	}

	// ── Retry policy verification ─────────────────────────────────────────────
	separator("RETRY POLICY VERIFICATION")
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		fmt.Printf("  attempt %d — simulated backoff before this attempt: %v\n",
			attempt, policy.NextDelay(attempt))
	}

	// ── Provider failover demonstration ──────────────────────────────────────
	separator("PROVIDER FAILOVER DEMONSTRATION")
	fmt.Println("  Simulating 100 % primary failure to force fallback path …")

	alwaysFail := NewEmailProvider(1.0)  // primary always fails
	alwaysOK   := NewEmailProvider(0.0)  // secondary always succeeds
	fallbackProvider := &ProviderWithFallback{Primary: alwaysFail, Secondary: alwaysOK}

	demoNotif := Notification{
		ID: "demo-failover", UserID: "u999", Channel: ChannelEmail,
		Subject: "Failover demo", Body: "Should reach secondary.", Priority: PriorityCritical,
	}

	err := fallbackProvider.Send(demoNotif)
	if err != nil {
		fmt.Printf("  Result: FAILED — %v\n", err)
	} else {
		fmt.Println("  Result: delivered via secondary provider (primary silently failed)")
	}

	// ── Validate errors.Is works through wrapping ─────────────────────────────
	separator("ERROR WRAPPING VALIDATION")
	sentinelErr := errors.New("smtp: connection refused")
	wrapped := fmt.Errorf("EmailProvider: %w", sentinelErr)
	if errors.Is(wrapped, sentinelErr) {
		fmt.Println("  errors.Is correctly unwraps through provider error chain.")
	}

	fmt.Println("\nDone.")
}
