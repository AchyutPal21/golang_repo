// FILE: book/part6_production_engineering/chapter83_mocking/examples/01_interfaces_mocks/main.go
// CHAPTER: 83 — Mocking
// TOPIC: Hand-written mocks via interfaces, spy recorders, stub responses,
//        and the difference between mocks, stubs, and fakes.
//
// Run:
//   go run ./examples/01_interfaces_mocks

package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// ─────────────────────────────────────────────────────────────────────────────
// INTERFACES (production abstractions)
// ─────────────────────────────────────────────────────────────────────────────

type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
}

type UserRepository interface {
	FindByID(ctx context.Context, id string) (*User, error)
	Save(ctx context.Context, u *User) error
}

type PaymentGateway interface {
	Charge(ctx context.Context, amount int, cardToken string) (string, error)
	Refund(ctx context.Context, chargeID string) error
}

type User struct {
	ID    string
	Email string
	Name  string
}

// ─────────────────────────────────────────────────────────────────────────────
// STUB — returns fixed responses, records nothing
// ─────────────────────────────────────────────────────────────────────────────

type StubEmailSender struct {
	err error // if non-nil, Send returns this error
}

func (s *StubEmailSender) Send(_ context.Context, to, subject, body string) error {
	return s.err
}

// ─────────────────────────────────────────────────────────────────────────────
// SPY (mock recorder) — records all calls for later assertion
// ─────────────────────────────────────────────────────────────────────────────

type EmailCall struct {
	To      string
	Subject string
	Body    string
}

type SpyEmailSender struct {
	mu    sync.Mutex
	Calls []EmailCall
	Err   error
}

func (s *SpyEmailSender) Send(_ context.Context, to, subject, body string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Calls = append(s.Calls, EmailCall{To: to, Subject: subject, Body: body})
	return s.Err
}

func (s *SpyEmailSender) CallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.Calls)
}

func (s *SpyEmailSender) LastCall() (EmailCall, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Calls) == 0 {
		return EmailCall{}, false
	}
	return s.Calls[len(s.Calls)-1], true
}

// ─────────────────────────────────────────────────────────────────────────────
// FAKE — working in-memory implementation
// ─────────────────────────────────────────────────────────────────────────────

type FakeUserRepository struct {
	mu    sync.RWMutex
	users map[string]*User
}

func NewFakeUserRepository(seed ...*User) *FakeUserRepository {
	f := &FakeUserRepository{users: make(map[string]*User)}
	for _, u := range seed {
		f.users[u.ID] = u
	}
	return f
}

func (f *FakeUserRepository) FindByID(_ context.Context, id string) (*User, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	u, ok := f.users[id]
	if !ok {
		return nil, fmt.Errorf("user %q not found", id)
	}
	return u, nil
}

func (f *FakeUserRepository) Save(_ context.Context, u *User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users[u.ID] = u
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// CONFIGURABLE MOCK — returns per-call programmed responses
// ─────────────────────────────────────────────────────────────────────────────

type MockPaymentGateway struct {
	mu            sync.Mutex
	chargeResults []struct {
		id  string
		err error
	}
	chargeIdx   int
	refundErr   error
	ChargeCalls []struct{ Amount int; Token string }
	RefundCalls []string
}

func (m *MockPaymentGateway) OnCharge(id string, err error) *MockPaymentGateway {
	m.chargeResults = append(m.chargeResults, struct {
		id  string
		err error
	}{id, err})
	return m
}

func (m *MockPaymentGateway) OnRefund(err error) *MockPaymentGateway {
	m.refundErr = err
	return m
}

func (m *MockPaymentGateway) Charge(_ context.Context, amount int, token string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ChargeCalls = append(m.ChargeCalls, struct{ Amount int; Token string }{amount, token})
	if m.chargeIdx >= len(m.chargeResults) {
		return "", fmt.Errorf("no more programmed charge results")
	}
	r := m.chargeResults[m.chargeIdx]
	m.chargeIdx++
	return r.id, r.err
}

func (m *MockPaymentGateway) Refund(_ context.Context, chargeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RefundCalls = append(m.RefundCalls, chargeID)
	return m.refundErr
}

// ─────────────────────────────────────────────────────────────────────────────
// SERVICE UNDER TEST — uses all three dependencies via interfaces
// ─────────────────────────────────────────────────────────────────────────────

type OrderService struct {
	users    UserRepository
	payments PaymentGateway
	email    EmailSender
}

func NewOrderService(u UserRepository, p PaymentGateway, e EmailSender) *OrderService {
	return &OrderService{users: u, payments: p, email: e}
}

type PlaceOrderResult struct {
	ChargeID string
}

func (s *OrderService) PlaceOrder(ctx context.Context, userID, cardToken string, amount int) (*PlaceOrderResult, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user lookup: %w", err)
	}

	chargeID, err := s.payments.Charge(ctx, amount, cardToken)
	if err != nil {
		return nil, fmt.Errorf("charge failed: %w", err)
	}

	subject := "Order confirmed"
	body := fmt.Sprintf("Hi %s, your payment of %d¢ was charged.", user.Name, amount)
	if err := s.email.Send(ctx, user.Email, subject, body); err != nil {
		// Email failure is non-fatal — log in production
		fmt.Printf("  [warn] email failed: %v\n", err)
	}

	return &PlaceOrderResult{ChargeID: chargeID}, nil
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

func (s *Suite) Report() { fmt.Printf("  %d/%d passed\n", s.passed, s.passed+s.failed) }

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Interfaces & Mocks ===")
	fmt.Println()
	ctx := context.Background()

	// ── STUB ─────────────────────────────────────────────────────────────────
	fmt.Println("--- Stub: fixed response, no recording ---")
	suite := &Suite{}

	suite.Run("Stub/email_success", func(t *T) {
		stub := &StubEmailSender{}
		svc := NewOrderService(
			NewFakeUserRepository(&User{"u1", "alice@x.com", "Alice"}),
			(&MockPaymentGateway{}).OnCharge("ch-1", nil),
			stub,
		)
		res, err := svc.PlaceOrder(ctx, "u1", "tok_1", 1000)
		if err != nil {
			t.Errorf("PlaceOrder: %v", err)
			return
		}
		if res.ChargeID != "ch-1" {
			t.Errorf("chargeID = %q, want ch-1", res.ChargeID)
		}
	})

	suite.Run("Stub/email_failure_non_fatal", func(t *T) {
		stub := &StubEmailSender{err: fmt.Errorf("smtp down")}
		svc := NewOrderService(
			NewFakeUserRepository(&User{"u2", "bob@x.com", "Bob"}),
			(&MockPaymentGateway{}).OnCharge("ch-2", nil),
			stub,
		)
		// Email failure should not fail the order
		_, err := svc.PlaceOrder(ctx, "u2", "tok_2", 500)
		if err != nil {
			t.Errorf("PlaceOrder: expected success despite email failure, got: %v", err)
		}
	})

	suite.Report()

	// ── SPY ───────────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Spy: records calls for assertion ---")
	s2 := &Suite{}

	s2.Run("Spy/email_called_with_correct_fields", func(t *T) {
		spy := &SpyEmailSender{}
		svc := NewOrderService(
			NewFakeUserRepository(&User{"u3", "carol@x.com", "Carol"}),
			(&MockPaymentGateway{}).OnCharge("ch-3", nil),
			spy,
		)
		svc.PlaceOrder(ctx, "u3", "tok_3", 2000)

		if spy.CallCount() != 1 {
			t.Errorf("email calls = %d, want 1", spy.CallCount())
			return
		}
		call, _ := spy.LastCall()
		if call.To != "carol@x.com" {
			t.Errorf("email To = %q, want carol@x.com", call.To)
		}
		if !strings.Contains(call.Body, "Carol") {
			t.Errorf("email body missing name: %q", call.Body)
		}
	})

	s2.Run("Spy/no_email_on_payment_failure", func(t *T) {
		spy := &SpyEmailSender{}
		svc := NewOrderService(
			NewFakeUserRepository(&User{"u4", "dave@x.com", "Dave"}),
			(&MockPaymentGateway{}).OnCharge("", fmt.Errorf("card declined")),
			spy,
		)
		svc.PlaceOrder(ctx, "u4", "tok_4", 999)
		if spy.CallCount() != 0 {
			t.Errorf("email called %d times after payment failure, want 0", spy.CallCount())
		}
	})

	s2.Report()

	// ── FAKE ─────────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Fake: in-memory working implementation ---")
	s3 := &Suite{}

	s3.Run("Fake/user_not_found", func(t *T) {
		repo := NewFakeUserRepository() // empty
		svc := NewOrderService(repo, &MockPaymentGateway{}, &StubEmailSender{})
		_, err := svc.PlaceOrder(ctx, "missing", "tok", 100)
		if err == nil {
			t.Errorf("expected error for missing user, got nil")
		}
	})

	s3.Run("Fake/user_found_after_save", func(t *T) {
		repo := NewFakeUserRepository()
		repo.Save(ctx, &User{"u5", "eve@x.com", "Eve"})
		svc := NewOrderService(repo,
			(&MockPaymentGateway{}).OnCharge("ch-5", nil),
			&StubEmailSender{},
		)
		res, err := svc.PlaceOrder(ctx, "u5", "tok_5", 100)
		if err != nil {
			t.Errorf("PlaceOrder: %v", err)
			return
		}
		if res.ChargeID != "ch-5" {
			t.Errorf("chargeID = %q, want ch-5", res.ChargeID)
		}
	})

	s3.Report()

	// ── CONFIGURABLE MOCK ─────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Configurable mock: programmed responses ---")
	s4 := &Suite{}

	s4.Run("Mock/retry_after_transient_failure", func(t *T) {
		// First call fails, second succeeds (simulating retry logic elsewhere)
		mock := (&MockPaymentGateway{}).
			OnCharge("", fmt.Errorf("timeout")).
			OnCharge("ch-6", nil)

		repo := NewFakeUserRepository(&User{"u6", "frank@x.com", "Frank"})
		svc := NewOrderService(repo, mock, &StubEmailSender{})

		// First attempt fails.
		_, err := svc.PlaceOrder(ctx, "u6", "tok_6", 500)
		if err == nil {
			t.Errorf("first attempt: expected error, got nil")
		}

		// Second attempt succeeds.
		res, err := svc.PlaceOrder(ctx, "u6", "tok_6", 500)
		if err != nil {
			t.Errorf("second attempt: %v", err)
			return
		}
		if res.ChargeID != "ch-6" {
			t.Errorf("chargeID = %q, want ch-6", res.ChargeID)
		}
		if len(mock.ChargeCalls) != 2 {
			t.Errorf("charge calls = %d, want 2", len(mock.ChargeCalls))
		}
	})

	s4.Report()

	// ── SUMMARY ───────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Test double taxonomy ---")
	fmt.Println(`  Dummy   — passed but never used (satisfies interface requirement)
  Stub    — returns fixed responses; no call recording
  Spy     — records calls; you assert against them after the fact
  Mock    — pre-programmed expectations; verifies behaviour automatically
  Fake    — working implementation, lighter than production (in-memory DB)

  Rule of thumb:
    Use Fake when you need real logic (e.g. a DB with query semantics).
    Use Spy when you need to assert on side effects (emails sent, events fired).
    Use Stub when you just need a dependency to "not fail".`)
}
