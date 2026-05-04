// FILE: book/part6_production_engineering/chapter83_mocking/exercises/01_service_mocks/main.go
// CHAPTER: 83 — Mocking
// TOPIC: Full mock suite for a notification service — stubs, spies, fakes,
//        and configurable mocks for multi-provider dispatch.
//
// Run:
//   go run ./exercises/01_service_mocks

package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

// ─────────────────────────────────────────────────────────────────────────────
// INTERFACES
// ─────────────────────────────────────────────────────────────────────────────

type SMSProvider interface {
	SendSMS(ctx context.Context, phone, message string) error
}

type PushProvider interface {
	SendPush(ctx context.Context, deviceToken, title, body string) error
}

type UserStore interface {
	GetUser(ctx context.Context, userID string) (*UserPrefs, error)
}

type UserPrefs struct {
	ID          string
	Phone       string
	DeviceToken string
	Channels    []string // "sms", "push"
}

// ─────────────────────────────────────────────────────────────────────────────
// NOTIFICATION SERVICE
// ─────────────────────────────────────────────────────────────────────────────

type NotificationService struct {
	users UserStore
	sms   SMSProvider
	push  PushProvider
	Sent  atomic.Int64
}

func NewNotificationService(u UserStore, sms SMSProvider, push PushProvider) *NotificationService {
	return &NotificationService{users: u, sms: sms, push: push}
}

type SendResult struct {
	Channels []string
	Errors   []error
}

func (ns *NotificationService) Notify(ctx context.Context, userID, title, message string) (*SendResult, error) {
	user, err := ns.users.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	result := &SendResult{}
	for _, ch := range user.Channels {
		switch ch {
		case "sms":
			if err := ns.sms.SendSMS(ctx, user.Phone, message); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("sms: %w", err))
			} else {
				result.Channels = append(result.Channels, "sms")
				ns.Sent.Add(1)
			}
		case "push":
			if err := ns.push.SendPush(ctx, user.DeviceToken, title, message); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("push: %w", err))
			} else {
				result.Channels = append(result.Channels, "push")
				ns.Sent.Add(1)
			}
		}
	}
	return result, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// FAKES
// ─────────────────────────────────────────────────────────────────────────────

type FakeUserStore struct {
	mu    sync.RWMutex
	users map[string]*UserPrefs
}

func NewFakeUserStore(users ...*UserPrefs) *FakeUserStore {
	f := &FakeUserStore{users: make(map[string]*UserPrefs)}
	for _, u := range users {
		f.users[u.ID] = u
	}
	return f
}

func (f *FakeUserStore) GetUser(_ context.Context, id string) (*UserPrefs, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	u, ok := f.users[id]
	if !ok {
		return nil, fmt.Errorf("user %q not found", id)
	}
	return u, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SPIES
// ─────────────────────────────────────────────────────────────────────────────

type SMSCall struct{ Phone, Message string }

type SpySMSProvider struct {
	mu    sync.Mutex
	Calls []SMSCall
	Errs  []error // per-call errors; if shorter than Calls, last error repeats
}

func (s *SpySMSProvider) SendSMS(_ context.Context, phone, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Calls = append(s.Calls, SMSCall{phone, message})
	idx := len(s.Calls) - 1
	if idx < len(s.Errs) {
		return s.Errs[idx]
	}
	return nil
}

type PushCall struct{ Token, Title, Body string }

type SpyPushProvider struct {
	mu    sync.Mutex
	Calls []PushCall
	Err   error
}

func (s *SpyPushProvider) SendPush(_ context.Context, token, title, body string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Calls = append(s.Calls, PushCall{token, title, body})
	return s.Err
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
// TESTS
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Notification Service Mock Suite ===")
	fmt.Println()
	ctx := context.Background()

	fmt.Println("--- User not found ---")
	s1 := &Suite{}

	s1.Run("Notify/user_not_found", func(t *T) {
		svc := NewNotificationService(NewFakeUserStore(), &SpySMSProvider{}, &SpyPushProvider{})
		_, err := svc.Notify(ctx, "ghost", "Hello", "world")
		if err == nil {
			t.Errorf("expected error for unknown user, got nil")
		}
	})
	s1.Report()

	fmt.Println()
	fmt.Println("--- SMS-only user ---")
	s2 := &Suite{}

	alice := &UserPrefs{ID: "alice", Phone: "+1555001", Channels: []string{"sms"}}
	store := NewFakeUserStore(alice)

	s2.Run("Notify/sms_channel", func(t *T) {
		sms := &SpySMSProvider{}
		push := &SpyPushProvider{}
		svc := NewNotificationService(store, sms, push)

		res, err := svc.Notify(ctx, "alice", "Alert", "Your order shipped")
		if err != nil {
			t.Errorf("Notify: %v", err)
			return
		}
		if len(sms.Calls) != 1 {
			t.Errorf("SMS calls = %d, want 1", len(sms.Calls))
			return
		}
		if sms.Calls[0].Phone != "+1555001" {
			t.Errorf("SMS phone = %q, want +1555001", sms.Calls[0].Phone)
		}
		if !strings.Contains(sms.Calls[0].Message, "shipped") {
			t.Errorf("SMS message missing content: %q", sms.Calls[0].Message)
		}
		if len(push.Calls) != 0 {
			t.Errorf("push calls = %d, want 0 (no push channel)", len(push.Calls))
		}
		if len(res.Channels) != 1 || res.Channels[0] != "sms" {
			t.Errorf("result channels = %v, want [sms]", res.Channels)
		}
	})
	s2.Report()

	fmt.Println()
	fmt.Println("--- Multi-channel user ---")
	s3 := &Suite{}

	bob := &UserPrefs{ID: "bob", Phone: "+1555002", DeviceToken: "tok-bob", Channels: []string{"sms", "push"}}
	store2 := NewFakeUserStore(bob)

	s3.Run("Notify/both_channels", func(t *T) {
		sms := &SpySMSProvider{}
		push := &SpyPushProvider{}
		svc := NewNotificationService(store2, sms, push)

		res, err := svc.Notify(ctx, "bob", "News", "New message")
		if err != nil {
			t.Errorf("Notify: %v", err)
			return
		}
		if len(sms.Calls) != 1 {
			t.Errorf("SMS calls = %d, want 1", len(sms.Calls))
		}
		if len(push.Calls) != 1 {
			t.Errorf("Push calls = %d, want 1", len(push.Calls))
		}
		if len(res.Channels) != 2 {
			t.Errorf("channels = %v, want [sms push]", res.Channels)
		}
		if svc.Sent.Load() != 2 {
			t.Errorf("Sent = %d, want 2", svc.Sent.Load())
		}
	})

	s3.Run("Notify/sms_failure_push_succeeds", func(t *T) {
		sms := &SpySMSProvider{Errs: []error{fmt.Errorf("network error")}}
		push := &SpyPushProvider{}
		svc := NewNotificationService(store2, sms, push)

		res, err := svc.Notify(ctx, "bob", "News", "Degraded")
		if err != nil {
			t.Errorf("Notify: unexpected top-level error: %v", err)
			return
		}
		if len(res.Errors) != 1 {
			t.Errorf("errors = %d, want 1 (sms failed)", len(res.Errors))
		}
		if len(res.Channels) != 1 || res.Channels[0] != "push" {
			t.Errorf("channels = %v, want [push] (sms failed)", res.Channels)
		}
	})
	s3.Report()

	fmt.Println()
	fmt.Println("--- Push argument capture ---")
	s4 := &Suite{}

	carol := &UserPrefs{ID: "carol", DeviceToken: "tok-carol", Channels: []string{"push"}}
	store3 := NewFakeUserStore(carol)

	s4.Run("Notify/push_title_and_body", func(t *T) {
		push := &SpyPushProvider{}
		svc := NewNotificationService(store3, &SpySMSProvider{}, push)

		svc.Notify(ctx, "carol", "Flash Sale", "50% off today only")

		if len(push.Calls) != 1 {
			t.Errorf("push calls = %d, want 1", len(push.Calls))
			return
		}
		call := push.Calls[0]
		if call.Token != "tok-carol" {
			t.Errorf("push token = %q, want tok-carol", call.Token)
		}
		if call.Title != "Flash Sale" {
			t.Errorf("push title = %q, want Flash Sale", call.Title)
		}
		if !strings.Contains(call.Body, "50%") {
			t.Errorf("push body missing content: %q", call.Body)
		}
	})
	s4.Report()

	fmt.Println()
	fmt.Println("--- Concurrent notifications ---")
	s5 := &Suite{}

	s5.Run("Notify/concurrent_safe", func(t *T) {
		users := make([]*UserPrefs, 5)
		for i := range users {
			users[i] = &UserPrefs{
				ID:      fmt.Sprintf("u%d", i),
				Phone:   fmt.Sprintf("+155500%d", i),
				Channels: []string{"sms"},
			}
		}
		store4 := NewFakeUserStore(users...)
		sms := &SpySMSProvider{}
		svc := NewNotificationService(store4, sms, &SpyPushProvider{})

		var wg sync.WaitGroup
		for _, u := range users {
			u := u
			wg.Add(1)
			go func() {
				defer wg.Done()
				svc.Notify(ctx, u.ID, "Hi", "concurrent")
			}()
		}
		wg.Wait()

		if len(sms.Calls) != 5 {
			t.Errorf("SMS calls = %d, want 5 (one per user)", len(sms.Calls))
		}
		if svc.Sent.Load() != 5 {
			t.Errorf("Sent = %d, want 5", svc.Sent.Load())
		}
	})
	s5.Report()
}
