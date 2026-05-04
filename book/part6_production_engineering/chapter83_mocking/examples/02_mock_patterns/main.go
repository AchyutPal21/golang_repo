// FILE: book/part6_production_engineering/chapter83_mocking/examples/02_mock_patterns/main.go
// CHAPTER: 83 — Mocking
// TOPIC: Advanced mock patterns — call ordering, argument capture, call count
//        expectations, partial mocks, and avoiding over-mocking.
//
// Run:
//   go run ./examples/02_mock_patterns

package main

import (
	"context"
	"fmt"
	"sync"
)

// ─────────────────────────────────────────────────────────────────────────────
// INTERFACES
// ─────────────────────────────────────────────────────────────────────────────

type Cache interface {
	Get(key string) (string, bool)
	Set(key, value string)
	Delete(key string)
}

type Notifier interface {
	Notify(ctx context.Context, userID, message string) error
}

type AuditLog interface {
	Record(action, entity, entityID string)
}

// ─────────────────────────────────────────────────────────────────────────────
// CALL-ORDER-AWARE MOCK
// ─────────────────────────────────────────────────────────────────────────────

type CacheCall struct {
	Method string
	Args   []string
}

type OrderedCacheMock struct {
	mu    sync.Mutex
	Calls []CacheCall
	store map[string]string
}

func NewOrderedCacheMock() *OrderedCacheMock {
	return &OrderedCacheMock{store: make(map[string]string)}
}

func (m *OrderedCacheMock) Get(key string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, CacheCall{"Get", []string{key}})
	v, ok := m.store[key]
	return v, ok
}

func (m *OrderedCacheMock) Set(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, CacheCall{"Set", []string{key, value}})
	m.store[key] = value
}

func (m *OrderedCacheMock) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, CacheCall{"Delete", []string{key}})
	delete(m.store, key)
}

func (m *OrderedCacheMock) MethodSequence() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	seq := make([]string, len(m.Calls))
	for i, c := range m.Calls {
		seq[i] = c.Method
	}
	return seq
}

// ─────────────────────────────────────────────────────────────────────────────
// ARGUMENT CAPTURE MOCK
// ─────────────────────────────────────────────────────────────────────────────

type NotifyCall struct {
	UserID  string
	Message string
}

type CapturingNotifier struct {
	mu    sync.Mutex
	Calls []NotifyCall
	Err   error
}

func (n *CapturingNotifier) Notify(_ context.Context, userID, message string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Calls = append(n.Calls, NotifyCall{UserID: userID, Message: message})
	return n.Err
}

func (n *CapturingNotifier) CalledWith(userID string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, c := range n.Calls {
		if c.UserID == userID {
			return true
		}
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// EXPECTATION-BASED MOCK (simple expectations without a framework)
// ─────────────────────────────────────────────────────────────────────────────

type AuditExpectation struct {
	Action   string
	Entity   string
	EntityID string
	satisfied bool
}

type ExpectingAuditLog struct {
	mu           sync.Mutex
	expectations []*AuditExpectation
}

func (e *ExpectingAuditLog) Expect(action, entity, entityID string) *ExpectingAuditLog {
	e.expectations = append(e.expectations, &AuditExpectation{
		Action: action, Entity: entity, EntityID: entityID,
	})
	return e
}

func (e *ExpectingAuditLog) Record(action, entity, entityID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, exp := range e.expectations {
		if !exp.satisfied && exp.Action == action && exp.Entity == entity && exp.EntityID == entityID {
			exp.satisfied = true
			return
		}
	}
}

func (e *ExpectingAuditLog) Verify() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	var unmet []string
	for _, exp := range e.expectations {
		if !exp.satisfied {
			unmet = append(unmet, fmt.Sprintf("Record(%q, %q, %q) never called", exp.Action, exp.Entity, exp.EntityID))
		}
	}
	return unmet
}

// ─────────────────────────────────────────────────────────────────────────────
// SERVICES UNDER TEST
// ─────────────────────────────────────────────────────────────────────────────

type UserService struct {
	cache    Cache
	notifier Notifier
	audit    AuditLog
}

func NewUserService(c Cache, n Notifier, a AuditLog) *UserService {
	return &UserService{cache: c, notifier: n, audit: a}
}

func (s *UserService) UpdateEmail(ctx context.Context, userID, newEmail string) error {
	// Pattern: invalidate cache first, then persist (simulated), then notify.
	s.cache.Delete("user:" + userID)
	// (DB update would happen here in production)
	s.cache.Set("user:"+userID+":email", newEmail)
	s.audit.Record("email_updated", "user", userID)
	return s.notifier.Notify(ctx, userID, "Your email has been updated to "+newEmail)
}

func (s *UserService) GetProfile(ctx context.Context, userID string) (string, error) {
	key := "profile:" + userID
	if v, ok := s.cache.Get(key); ok {
		return v, nil
	}
	// Cache miss — simulate DB lookup.
	profile := fmt.Sprintf("profile_data_for_%s", userID)
	s.cache.Set(key, profile)
	return profile, nil
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
	fmt.Println("=== Advanced Mock Patterns ===")
	fmt.Println()
	ctx := context.Background()
	suite := &Suite{}

	// ── CALL ORDER ────────────────────────────────────────────────────────────
	fmt.Println("--- Call order verification ---")

	suite.Run("UpdateEmail/call_order", func(t *T) {
		cache := NewOrderedCacheMock()
		notifier := &CapturingNotifier{}
		audit := &ExpectingAuditLog{}

		svc := NewUserService(cache, notifier, audit)
		svc.UpdateEmail(ctx, "u1", "new@example.com")

		seq := cache.MethodSequence()
		if len(seq) < 2 {
			t.Errorf("expected at least 2 cache calls, got %d", len(seq))
			return
		}
		// Delete must come before Set (invalidate before write).
		if seq[0] != "Delete" {
			t.Errorf("first cache op = %q, want Delete", seq[0])
		}
		if seq[1] != "Set" {
			t.Errorf("second cache op = %q, want Set", seq[1])
		}
	})

	// ── ARGUMENT CAPTURE ──────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Argument capture ---")

	suite.Run("UpdateEmail/notification_sent", func(t *T) {
		notifier := &CapturingNotifier{}
		svc := NewUserService(NewOrderedCacheMock(), notifier, &ExpectingAuditLog{})
		svc.UpdateEmail(ctx, "u2", "updated@example.com")

		if len(notifier.Calls) != 1 {
			t.Errorf("notification calls = %d, want 1", len(notifier.Calls))
			return
		}
		call := notifier.Calls[0]
		if call.UserID != "u2" {
			t.Errorf("notification userID = %q, want u2", call.UserID)
		}
		if call.Message == "" {
			t.Errorf("notification message is empty")
		}
	})

	// ── EXPECTATION-BASED ─────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Expectation-based audit log ---")

	suite.Run("UpdateEmail/audit_recorded", func(t *T) {
		audit := (&ExpectingAuditLog{}).
			Expect("email_updated", "user", "u3")

		svc := NewUserService(NewOrderedCacheMock(), &CapturingNotifier{}, audit)
		svc.UpdateEmail(ctx, "u3", "x@y.com")

		unmet := audit.Verify()
		for _, u := range unmet {
			t.Errorf("unmet expectation: %s", u)
		}
	})

	suite.Run("UpdateEmail/audit_missing_shows_failure", func(t *T) {
		audit := (&ExpectingAuditLog{}).
			Expect("email_updated", "user", "u-NEVER-CALLED")
		// Don't call UpdateEmail — expectation should be unmet.
		unmet := audit.Verify()
		if len(unmet) == 0 {
			t.Errorf("expected unmet expectation, Verify() returned none")
		}
	})

	// ── CACHE READ-THROUGH ────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Cache read-through ---")

	suite.Run("GetProfile/cache_miss_then_hit", func(t *T) {
		cache := NewOrderedCacheMock()
		svc := NewUserService(cache, &CapturingNotifier{}, &ExpectingAuditLog{})

		// First call: miss → set.
		p1, _ := svc.GetProfile(ctx, "u4")
		// Second call: hit (no additional Set).
		p2, _ := svc.GetProfile(ctx, "u4")

		if p1 != p2 {
			t.Errorf("profiles differ: %q vs %q", p1, p2)
		}
		seq := cache.MethodSequence()
		// Expected: Get (miss), Set, Get (hit)
		if len(seq) != 3 {
			t.Errorf("cache call sequence len = %d, want 3: %v", len(seq), seq)
			return
		}
		if seq[0] != "Get" || seq[1] != "Set" || seq[2] != "Get" {
			t.Errorf("unexpected sequence: %v", seq)
		}
	})

	suite.Report()

	// ── OVER-MOCKING ANTI-PATTERN ─────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Over-mocking anti-pattern ---")
	fmt.Println(`  Problem: mocking everything creates tests that mirror the implementation.
  Any refactor breaks tests even when behaviour is unchanged.

  Signs of over-mocking:
    - Test has more mock setup than assertions
    - Test breaks when you rename a private function
    - Mock expectations are a copy of the production code

  Better approach:
    - Mock only external boundaries (DB, HTTP, email, time)
    - Use Fakes for internal collaborators with real logic
    - Test observable behaviour (return values, state), not internal call sequences
    - Prefer integration tests for wiring; unit tests for logic`)
}
