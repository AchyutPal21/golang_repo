// FILE: book/part5_building_backends/chapter79_idempotency/exercises/01_payment_service/main.go
// CHAPTER: 79 — Idempotency at the API Boundary
// TOPIC: Idempotent payment service — store-and-replay results, race-safe
//        first-write, outbox publishing, and idempotency TTL expiry.
//
// Run (from the chapter folder):
//   go run ./exercises/01_payment_service

package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// IDEMPOTENCY STORE WITH TTL
// ─────────────────────────────────────────────────────────────────────────────

type StoredResult struct {
	Response any
	Err      error
	At       time.Time
}

type IdempStore struct {
	mu       sync.Mutex
	results  map[string]*StoredResult
	inflight map[string]chan struct{}
	ttl      time.Duration
}

func NewIdempStore(ttl time.Duration) *IdempStore {
	return &IdempStore{
		results:  make(map[string]*StoredResult),
		inflight: make(map[string]chan struct{}),
		ttl:      ttl,
	}
}

func (s *IdempStore) Do(key string, fn func() (any, error)) (any, error) {
	s.mu.Lock()
	if r, ok := s.results[key]; ok {
		if s.ttl > 0 && time.Since(r.At) > s.ttl {
			delete(s.results, key) // expired
		} else {
			s.mu.Unlock()
			return r.Response, r.Err
		}
	}
	if ch, ok := s.inflight[key]; ok {
		s.mu.Unlock()
		<-ch
		s.mu.Lock()
		r := s.results[key]
		s.mu.Unlock()
		if r != nil {
			return r.Response, r.Err
		}
		return nil, fmt.Errorf("concurrent request failed")
	}
	done := make(chan struct{})
	s.inflight[key] = done
	s.mu.Unlock()

	resp, err := fn()

	s.mu.Lock()
	s.results[key] = &StoredResult{Response: resp, Err: err, At: time.Now()}
	delete(s.inflight, key)
	s.mu.Unlock()
	close(done)
	return resp, err
}

func (s *IdempStore) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.results)
}

// ─────────────────────────────────────────────────────────────────────────────
// PAYMENT DOMAIN
// ─────────────────────────────────────────────────────────────────────────────

type PaymentStatus string

const (
	StatusPending  PaymentStatus = "pending"
	StatusCharged  PaymentStatus = "charged"
	StatusRefunded PaymentStatus = "refunded"
	StatusFailed   PaymentStatus = "failed"
)

type Payment struct {
	ID         string
	OrderID    string
	CustomerID string
	Amount     int
	Status     PaymentStatus
	CreatedAt  time.Time
}

type ChargeRequest struct {
	IdempotencyKey string
	OrderID        string
	CustomerID     string
	Amount         int
}

type ChargeResponse struct {
	Payment *Payment
	Cached  bool
}

// ─────────────────────────────────────────────────────────────────────────────
// PAYMENT PROCESSOR
// ─────────────────────────────────────────────────────────────────────────────

type PaymentProcessor struct {
	mu       sync.Mutex
	payments map[string]*Payment
	seq      atomic.Int64
	Charges  atomic.Int64 // total actual charges (not replays)

	store  *IdempStore
	outbox []*OutboxEvent
	outMu  sync.Mutex
}

type OutboxEvent struct {
	ID        string
	EventType string
	Payload   string
	At        time.Time
	Published bool
}

func NewPaymentProcessor() *PaymentProcessor {
	return &PaymentProcessor{
		payments: make(map[string]*Payment),
		store:    NewIdempStore(24 * time.Hour),
	}
}

func (pp *PaymentProcessor) Charge(ctx context.Context, req ChargeRequest) (*ChargeResponse, error) {
	resp, err := pp.store.Do(req.IdempotencyKey, func() (any, error) {
		return pp.doCharge(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	cr := resp.(*ChargeResponse)
	// If store was hit, mark as cached.
	if pp.store.Count() > 0 && cr.Payment != nil {
		cr2 := *cr
		cr2.Cached = pp.store.Count() > 0
		return &cr2, nil
	}
	return cr, nil
}

func (pp *PaymentProcessor) doCharge(ctx context.Context, req ChargeRequest) (*ChargeResponse, error) {
	pp.Charges.Add(1)
	pp.mu.Lock()
	defer pp.mu.Unlock()
	id := fmt.Sprintf("pay-%d", pp.seq.Add(1))
	p := &Payment{
		ID:         id,
		OrderID:    req.OrderID,
		CustomerID: req.CustomerID,
		Amount:     req.Amount,
		Status:     StatusCharged,
		CreatedAt:  time.Now(),
	}
	pp.payments[id] = p

	// Write to outbox atomically with the payment record.
	evt := &OutboxEvent{
		ID:        fmt.Sprintf("evt-%d", pp.seq.Load()),
		EventType: "payment.charged",
		Payload:   fmt.Sprintf(`{"paymentID":%q,"orderID":%q,"amount":%d}`, id, req.OrderID, req.Amount),
		At:        time.Now(),
	}
	pp.outMu.Lock()
	pp.outbox = append(pp.outbox, evt)
	pp.outMu.Unlock()

	return &ChargeResponse{Payment: p, Cached: false}, nil
}

func (pp *PaymentProcessor) PublishOutbox() []*OutboxEvent {
	pp.outMu.Lock()
	defer pp.outMu.Unlock()
	var published []*OutboxEvent
	for _, e := range pp.outbox {
		if !e.Published {
			e.Published = true
			published = append(published, e)
		}
	}
	return published
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Idempotent Payment Service ===")
	fmt.Println()

	pp := NewPaymentProcessor()
	ctx := context.Background()

	// ── BASIC IDEMPOTENCY ─────────────────────────────────────────────────────
	fmt.Println("--- Basic idempotency: same key → same result ---")

	req := ChargeRequest{
		IdempotencyKey: "idemp-order-1-charge",
		OrderID:        "ord-1",
		CustomerID:     "cust-1",
		Amount:         9999,
	}

	resp1, _ := pp.Charge(ctx, req)
	fmt.Printf("  call 1: paymentID=%s amount=%d cached=%v\n", resp1.Payment.ID, resp1.Payment.Amount, resp1.Cached)

	resp2, _ := pp.Charge(ctx, req)
	fmt.Printf("  call 2: paymentID=%s amount=%d cached=%v\n", resp2.Payment.ID, resp2.Payment.Amount, resp2.Cached)

	resp3, _ := pp.Charge(ctx, req)
	fmt.Printf("  call 3: paymentID=%s amount=%d cached=%v\n", resp3.Payment.ID, resp3.Payment.Amount, resp3.Cached)

	fmt.Printf("  actual charges: %d (should be 1)\n", pp.Charges.Load())

	// ── DIFFERENT KEYS ────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Different keys → new charges ---")

	pp.Charge(ctx, ChargeRequest{"idemp-order-2", "ord-2", "cust-2", 4999})
	pp.Charge(ctx, ChargeRequest{"idemp-order-3", "ord-3", "cust-3", 2999})
	fmt.Printf("  total charges: %d\n", pp.Charges.Load())

	// ── CONCURRENT REQUESTS WITH SAME KEY ─────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Concurrent requests with same key (race safety) ---")

	pp2 := NewPaymentProcessor()
	concReq := ChargeRequest{
		IdempotencyKey: "concurrent-key-xyz",
		OrderID:        "ord-10",
		CustomerID:     "cust-10",
		Amount:         7777,
	}
	var wg sync.WaitGroup
	results := make([]*ChargeResponse, 5)
	for i := 0; i < 5; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, _ := pp2.Charge(ctx, concReq)
			results[i] = resp
		}()
	}
	wg.Wait()

	paymentIDs := make(map[string]bool)
	for _, r := range results {
		if r != nil {
			paymentIDs[r.Payment.ID] = true
		}
	}
	fmt.Printf("  5 concurrent calls produced %d unique payment ID(s)\n", len(paymentIDs))
	fmt.Printf("  actual charges: %d (should be 1)\n", pp2.Charges.Load())

	// ── TTL EXPIRY ────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- TTL expiry: expired key allows new charge ---")

	shortStore := NewIdempStore(50 * time.Millisecond)
	pp3 := &PaymentProcessor{
		payments: make(map[string]*Payment),
		store:    shortStore,
	}
	ttlReq := ChargeRequest{"ttl-key", "ord-20", "cust-20", 1000}
	r1, _ := pp3.Charge(ctx, ttlReq)
	fmt.Printf("  call 1: %s\n", r1.Payment.ID)

	// Wait for TTL to expire.
	time.Sleep(60 * time.Millisecond)
	r2, _ := pp3.Charge(ctx, ttlReq)
	fmt.Printf("  call 2 (after TTL): %s (new payment — key expired)\n", r2.Payment.ID)
	fmt.Printf("  charges: %d (should be 2 — TTL forced re-execution)\n", pp3.Charges.Load())

	// ── OUTBOX PUBLISHING ─────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Outbox: events published after charge ---")
	published := pp.PublishOutbox()
	for _, e := range published {
		fmt.Printf("  [relay] %s → %s\n", e.EventType, e.Payload)
	}
	fmt.Printf("  published %d events\n", len(published))
	fmt.Printf("  relay re-run: %d to publish (idempotent)\n", len(pp.PublishOutbox()))
}
