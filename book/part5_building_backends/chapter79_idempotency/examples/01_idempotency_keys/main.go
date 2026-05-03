// FILE: book/part5_building_backends/chapter79_idempotency/examples/01_idempotency_keys/main.go
// CHAPTER: 79 — Idempotency at the API Boundary
// TOPIC: Idempotency keys — storing results, race conditions on first request,
//        PUT vs POST semantics, and natural idempotency.
//
// Run (from the chapter folder):
//   go run ./examples/01_idempotency_keys

package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// IDEMPOTENCY STORE
// Stores the result of a request keyed by idempotency key.
// On a duplicate request, returns the stored result without re-executing.
// ─────────────────────────────────────────────────────────────────────────────

type IdempResult struct {
	Response any
	Error    error
	At       time.Time
}

type IdempStore struct {
	mu      sync.Mutex
	results map[string]*IdempResult
	inflight map[string]chan struct{} // in-progress requests; others wait
}

func NewIdempStore() *IdempStore {
	return &IdempStore{
		results:  make(map[string]*IdempResult),
		inflight: make(map[string]chan struct{}),
	}
}

// Do executes fn once for a given key; subsequent calls return the cached result.
// Concurrent calls with the same key wait for the first to finish.
func (s *IdempStore) Do(key string, fn func() (any, error)) (any, error) {
	s.mu.Lock()

	// Already stored — return cached result.
	if result, ok := s.results[key]; ok {
		s.mu.Unlock()
		return result.Response, result.Error
	}

	// Another goroutine is currently processing this key — wait.
	if ch, ok := s.inflight[key]; ok {
		s.mu.Unlock()
		<-ch
		// Now the result should be stored.
		s.mu.Lock()
		result := s.results[key]
		s.mu.Unlock()
		if result != nil {
			return result.Response, result.Error
		}
		return nil, fmt.Errorf("inflight request failed")
	}

	// First request for this key — mark as inflight.
	done := make(chan struct{})
	s.inflight[key] = done
	s.mu.Unlock()

	// Execute the operation.
	resp, err := fn()

	// Store result and unblock waiters.
	s.mu.Lock()
	s.results[key] = &IdempResult{Response: resp, Error: err, At: time.Now()}
	delete(s.inflight, key)
	s.mu.Unlock()
	close(done)

	return resp, err
}

func (s *IdempStore) Has(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.results[key]
	return ok
}

// ─────────────────────────────────────────────────────────────────────────────
// PAYMENT SERVICE
// ─────────────────────────────────────────────────────────────────────────────

type Payment struct {
	ID     string
	Amount int
	Status string
}

type PaymentService struct {
	mu       sync.Mutex
	payments map[string]*Payment
	nextID   int
	Charges  atomic.Int32 // actual charge count (should stay 1 per unique payment)
}

func NewPaymentService() *PaymentService {
	return &PaymentService{payments: make(map[string]*Payment)}
}

func (ps *PaymentService) Charge(ctx context.Context, amount int) (*Payment, error) {
	ps.Charges.Add(1)
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.nextID++
	p := &Payment{
		ID:     fmt.Sprintf("pay-%d", ps.nextID),
		Amount: amount,
		Status: "charged",
	}
	ps.payments[p.ID] = p
	return p, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// NATURAL IDEMPOTENCY (PUT semantics)
// ─────────────────────────────────────────────────────────────────────────────

type UserProfile struct {
	UserID string
	Name   string
	Email  string
	Bio    string
}

type ProfileStore struct {
	mu       sync.RWMutex
	profiles map[string]*UserProfile
}

func NewProfileStore() *ProfileStore {
	return &ProfileStore{profiles: make(map[string]*UserProfile)}
}

// Upsert is naturally idempotent — same input produces same state.
func (ps *ProfileStore) Upsert(p *UserProfile) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.profiles[p.UserID] = p
}

func (ps *ProfileStore) Get(userID string) (*UserProfile, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	p, ok := ps.profiles[userID]
	return p, ok
}

// ─────────────────────────────────────────────────────────────────────────────
// CONDITIONAL UPDATE (optimistic locking for idempotency)
// ─────────────────────────────────────────────────────────────────────────────

type Account struct {
	ID      string
	Balance int
	Version int
}

type AccountStore struct {
	mu       sync.Mutex
	accounts map[string]*Account
}

func NewAccountStore() *AccountStore {
	s := &AccountStore{accounts: make(map[string]*Account)}
	s.accounts["acc-1"] = &Account{ID: "acc-1", Balance: 10000, Version: 1}
	return s
}

// Debit is idempotent when called with the same (accountID, amount, expectedVersion).
// Returns error if version has changed (already updated by someone else).
func (s *AccountStore) Debit(accountID string, amount, expectedVersion int) (*Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, ok := s.accounts[accountID]
	if !ok {
		return nil, fmt.Errorf("account not found")
	}
	if acc.Version != expectedVersion {
		return nil, fmt.Errorf("version mismatch: expected %d got %d", expectedVersion, acc.Version)
	}
	if acc.Balance < amount {
		return nil, fmt.Errorf("insufficient funds")
	}
	updated := &Account{ID: acc.ID, Balance: acc.Balance - amount, Version: acc.Version + 1}
	s.accounts[accountID] = updated
	return updated, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Idempotency Keys ===")
	fmt.Println()

	// ── IDEMPOTENCY STORE ─────────────────────────────────────────────────────
	fmt.Println("--- Idempotency store: charge once, return cached on retry ---")
	store := NewIdempStore()
	svc := NewPaymentService()
	ctx := context.Background()

	chargeOnce := func(idempKey string, amount int) (*Payment, error) {
		resp, err := store.Do(idempKey, func() (any, error) {
			return svc.Charge(ctx, amount)
		})
		if err != nil {
			return nil, err
		}
		return resp.(*Payment), nil
	}

	// First call.
	p1, _ := chargeOnce("idemp-key-abc123", 9999)
	fmt.Printf("  call 1: payment %s amount=%d\n", p1.ID, p1.Amount)

	// Retry with same key — should return same result.
	p2, _ := chargeOnce("idemp-key-abc123", 9999)
	fmt.Printf("  call 2 (retry): payment %s amount=%d (same!)\n", p2.ID, p2.Amount)

	// Different key — new charge.
	p3, _ := chargeOnce("idemp-key-xyz789", 5000)
	fmt.Printf("  call 3 (new key): payment %s amount=%d\n", p3.ID, p3.Amount)

	fmt.Printf("  actual charges made: %d (should be 2)\n", svc.Charges.Load())

	// ── RACE CONDITION: CONCURRENT REQUESTS WITH SAME KEY ─────────────────────
	fmt.Println()
	fmt.Println("--- Race condition: concurrent calls with same key ---")
	store2 := NewIdempStore()
	svc2 := NewPaymentService()

	var wg sync.WaitGroup
	var payments [3]*Payment
	for i := 0; i < 3; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, _ := store2.Do("race-key", func() (any, error) {
				time.Sleep(5 * time.Millisecond) // simulate latency
				return svc2.Charge(ctx, 1000)
			})
			payments[i] = resp.(*Payment)
		}()
	}
	wg.Wait()

	// All 3 should have received the same payment ID.
	fmt.Printf("  goroutine 0 got: %s\n", payments[0].ID)
	fmt.Printf("  goroutine 1 got: %s\n", payments[1].ID)
	fmt.Printf("  goroutine 2 got: %s\n", payments[2].ID)
	fmt.Printf("  actual charges: %d (should be 1)\n", svc2.Charges.Load())

	// ── NATURAL IDEMPOTENCY: PUT ──────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Natural idempotency: PUT (upsert) ---")
	profiles := NewProfileStore()
	profile := &UserProfile{UserID: "u-1", Name: "Alice", Email: "alice@example.com", Bio: "Engineer"}

	// PUT three times — same result each time.
	for i := 0; i < 3; i++ {
		profiles.Upsert(profile)
	}
	p, _ := profiles.Get("u-1")
	fmt.Printf("  profile after 3 PUTs: %s <%s>\n", p.Name, p.Email)
	fmt.Println("  (PUT is naturally idempotent — same input → same state)")

	// ── OPTIMISTIC LOCKING ────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Optimistic locking: version-based debit ---")
	accounts := NewAccountStore()

	// First debit: version=1 → success, version becomes 2.
	acc, err := accounts.Debit("acc-1", 500, 1)
	fmt.Printf("  debit 500 v=1: balance=%d v=%d err=%v\n", acc.Balance, acc.Version, err)

	// Retry with same version (simulated duplicate request) → version mismatch.
	_, err = accounts.Debit("acc-1", 500, 1)
	fmt.Printf("  retry debit 500 v=1: err=%v (correctly rejected)\n", err)

	// Next debit: version=2 → success.
	acc2, err := accounts.Debit("acc-1", 200, 2)
	fmt.Printf("  debit 200 v=2: balance=%d v=%d err=%v\n", acc2.Balance, acc2.Version, err)

	// ── POST vs PUT vs PATCH ──────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- POST vs PUT vs PATCH idempotency ---")
	fmt.Println(`  POST /payments          → NOT idempotent (creates new resource each call)
  POST /payments + Idempotency-Key: xyz  → idempotent via stored result
  PUT  /users/u-1         → naturally idempotent (replace resource)
  PATCH /users/u-1        → NOT idempotent by default (depends on semantics)
  DELETE /resource/id     → idempotent (deleting non-existent = same outcome)
  GET/HEAD                → always idempotent (read-only)`)
}
