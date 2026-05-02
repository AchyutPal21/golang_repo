// FILE: book/part3_designing_software/chapter35_service_layer/examples/02_cross_cutting/main.go
// CHAPTER: 35 — Service Layer
// TOPIC: Cross-cutting concerns in the service layer — transactions, events,
//        idempotency, and service-to-service coordination.
//
// Run (from the chapter folder):
//   go run ./examples/02_cross_cutting

package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN
// ─────────────────────────────────────────────────────────────────────────────

type SubscriptionID string
type UserID string
type PlanID string

type Plan struct {
	ID       PlanID
	Name     string
	PriceCents int64
	Features []string
}

type Subscription struct {
	ID         SubscriptionID
	UserID     UserID
	PlanID     PlanID
	Status     string // "active" | "cancelled" | "past_due"
	StartedAt  time.Time
	CancelledAt *time.Time
}

var (
	ErrSubNotFound    = errors.New("subscription not found")
	ErrAlreadyCancelled = errors.New("subscription already cancelled")
	ErrPlanNotFound   = errors.New("plan not found")
	ErrDuplicateKey   = errors.New("idempotency key already used")
)

// ─────────────────────────────────────────────────────────────────────────────
// PORTS
// ─────────────────────────────────────────────────────────────────────────────

type SubscriptionRepo interface {
	Save(s Subscription) (Subscription, error)
	FindByID(id SubscriptionID) (Subscription, error)
	FindByUser(userID UserID) ([]Subscription, error)
}

type PlanRepo interface {
	FindByID(id PlanID) (Plan, error)
}

type PaymentGateway interface {
	Charge(userID UserID, amountCents int64, idempotencyKey string) (string, error)
	Refund(chargeID string) error
}

type EventPublisher interface {
	Publish(topic string, payload map[string]any)
}

type IdempotencyStore interface {
	Check(key string) (bool, error)   // true = already processed
	Mark(key string) error
}

// ─────────────────────────────────────────────────────────────────────────────
// SUBSCRIPTION SERVICE — shows cross-cutting service patterns
// ─────────────────────────────────────────────────────────────────────────────

type SubscribeRequest struct {
	UserID         UserID
	PlanID         PlanID
	IdempotencyKey string
}

type SubscriptionService struct {
	subs      SubscriptionRepo
	plans     PlanRepo
	payment   PaymentGateway
	events    EventPublisher
	idem      IdempotencyStore
	clock     func() time.Time
	subSeq    int
}

func NewSubscriptionService(
	subs SubscriptionRepo,
	plans PlanRepo,
	payment PaymentGateway,
	events EventPublisher,
	idem IdempotencyStore,
) *SubscriptionService {
	return &SubscriptionService{
		subs:    subs,
		plans:   plans,
		payment: payment,
		events:  events,
		idem:    idem,
		clock:   time.Now,
	}
}

// Subscribe — demonstrates idempotency + event publishing.
func (s *SubscriptionService) Subscribe(req SubscribeRequest) (Subscription, error) {
	// 1. Idempotency check — safe to retry.
	if req.IdempotencyKey != "" {
		used, err := s.idem.Check(req.IdempotencyKey)
		if err != nil {
			return Subscription{}, fmt.Errorf("Subscribe: idempotency check: %w", err)
		}
		if used {
			return Subscription{}, fmt.Errorf("Subscribe: %w", ErrDuplicateKey)
		}
	}

	// 2. Load plan.
	plan, err := s.plans.FindByID(req.PlanID)
	if err != nil {
		return Subscription{}, fmt.Errorf("Subscribe: %w", err)
	}

	// 3. Charge payment.
	s.subSeq++
	chargeKey := fmt.Sprintf("%s-charge", req.IdempotencyKey)
	chargeID, err := s.payment.Charge(req.UserID, plan.PriceCents, chargeKey)
	if err != nil {
		return Subscription{}, fmt.Errorf("Subscribe: payment failed: %w", err)
	}

	// 4. Persist subscription.
	sub := Subscription{
		ID:        SubscriptionID(fmt.Sprintf("SUB-%04d", s.subSeq)),
		UserID:    req.UserID,
		PlanID:    req.PlanID,
		Status:    "active",
		StartedAt: s.clock(),
	}
	saved, err := s.subs.Save(sub)
	if err != nil {
		// Compensate: refund the charge since we couldn't save.
		_ = s.payment.Refund(chargeID)
		return Subscription{}, fmt.Errorf("Subscribe: save failed: %w", err)
	}

	// 5. Mark idempotency key.
	if req.IdempotencyKey != "" {
		_ = s.idem.Mark(req.IdempotencyKey)
	}

	// 6. Publish domain event.
	s.events.Publish("subscription.created", map[string]any{
		"subscription_id": string(saved.ID),
		"user_id":         string(req.UserID),
		"plan_id":         string(req.PlanID),
		"amount_cents":    plan.PriceCents,
	})

	return saved, nil
}

// Cancel — demonstrates compensation / rollback pattern.
func (s *SubscriptionService) Cancel(subID SubscriptionID) (Subscription, error) {
	sub, err := s.subs.FindByID(subID)
	if err != nil {
		return Subscription{}, fmt.Errorf("Cancel: %w", err)
	}
	if sub.Status == "cancelled" {
		return Subscription{}, fmt.Errorf("Cancel: %w", ErrAlreadyCancelled)
	}

	now := s.clock()
	sub.Status = "cancelled"
	sub.CancelledAt = &now

	saved, err := s.subs.Save(sub)
	if err != nil {
		return Subscription{}, fmt.Errorf("Cancel: save failed: %w", err)
	}

	s.events.Publish("subscription.cancelled", map[string]any{
		"subscription_id": string(saved.ID),
		"user_id":         string(saved.UserID),
		"cancelled_at":    now.Format(time.RFC3339),
	})

	return saved, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// INFRASTRUCTURE
// ─────────────────────────────────────────────────────────────────────────────

type memSubRepo struct{ data map[SubscriptionID]Subscription }

func newMemSubRepo() *memSubRepo { return &memSubRepo{data: make(map[SubscriptionID]Subscription)} }
func (r *memSubRepo) Save(s Subscription) (Subscription, error) {
	r.data[s.ID] = s
	return s, nil
}
func (r *memSubRepo) FindByID(id SubscriptionID) (Subscription, error) {
	s, ok := r.data[id]
	if !ok {
		return Subscription{}, ErrSubNotFound
	}
	return s, nil
}
func (r *memSubRepo) FindByUser(userID UserID) ([]Subscription, error) {
	var result []Subscription
	for _, s := range r.data {
		if s.UserID == userID {
			result = append(result, s)
		}
	}
	return result, nil
}

type memPlanRepo struct{ plans map[PlanID]Plan }

func newMemPlanRepo(plans ...Plan) *memPlanRepo {
	r := &memPlanRepo{plans: make(map[PlanID]Plan)}
	for _, p := range plans {
		r.plans[p.ID] = p
	}
	return r
}
func (r *memPlanRepo) FindByID(id PlanID) (Plan, error) {
	p, ok := r.plans[id]
	if !ok {
		return Plan{}, ErrPlanNotFound
	}
	return p, nil
}

type fakePaymentGateway struct {
	charged []string
	failOn  string
	txSeq   int
}

func (g *fakePaymentGateway) Charge(userID UserID, amountCents int64, key string) (string, error) {
	if string(userID) == g.failOn {
		return "", fmt.Errorf("card declined for %s", userID)
	}
	g.txSeq++
	txID := fmt.Sprintf("TX-%04d", g.txSeq)
	g.charged = append(g.charged, txID)
	fmt.Printf("  [PAYMENT] charged %s $%.2f → %s\n", userID, float64(amountCents)/100, txID)
	return txID, nil
}
func (g *fakePaymentGateway) Refund(chargeID string) error {
	fmt.Printf("  [PAYMENT] refunded %s\n", chargeID)
	return nil
}

type stdoutEventPublisher struct{}

func (stdoutEventPublisher) Publish(topic string, payload map[string]any) {
	var parts []string
	for k, v := range payload {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	fmt.Printf("  [EVENT] %s  {%s}\n", topic, strings.Join(parts, " "))
}

type memIdempotencyStore struct{ used map[string]bool }

func newMemIdempotencyStore() *memIdempotencyStore {
	return &memIdempotencyStore{used: make(map[string]bool)}
}
func (s *memIdempotencyStore) Check(key string) (bool, error) { return s.used[key], nil }
func (s *memIdempotencyStore) Mark(key string) error          { s.used[key] = true; return nil }

func main() {
	plans := newMemPlanRepo(
		Plan{"starter", "Starter", 999, []string{"5 projects"}},
		Plan{"pro", "Pro", 2999, []string{"unlimited projects", "priority support"}},
	)
	payment := &fakePaymentGateway{}
	svc := NewSubscriptionService(
		newMemSubRepo(),
		plans,
		payment,
		stdoutEventPublisher{},
		newMemIdempotencyStore(),
	)

	fmt.Println("=== Subscribe ===")
	sub1, err := svc.Subscribe(SubscribeRequest{
		UserID:         "user-alice",
		PlanID:         "pro",
		IdempotencyKey: "req-001",
	})
	if err != nil {
		fmt.Println("  error:", err)
	} else {
		fmt.Printf("  created %s  status=%s\n", sub1.ID, sub1.Status)
	}

	fmt.Println()
	fmt.Println("=== Idempotent retry (same key) ===")
	_, err = svc.Subscribe(SubscribeRequest{
		UserID:         "user-alice",
		PlanID:         "pro",
		IdempotencyKey: "req-001",
	})
	fmt.Println("  expected duplicate error:", err)

	fmt.Println()
	fmt.Println("=== Payment failure → compensating refund ===")
	payment.failOn = "user-bad"
	_, err = svc.Subscribe(SubscribeRequest{
		UserID: "user-bad",
		PlanID: "starter",
	})
	fmt.Println("  error:", err)

	fmt.Println()
	fmt.Println("=== Cancel subscription ===")
	cancelled, err := svc.Cancel(sub1.ID)
	if err != nil {
		fmt.Println("  error:", err)
	} else {
		fmt.Printf("  %s  status=%s  cancelledAt=%s\n",
			cancelled.ID, cancelled.Status, cancelled.CancelledAt.Format(time.RFC3339))
	}

	fmt.Println()
	fmt.Println("=== Cancel already-cancelled ===")
	_, err = svc.Cancel(sub1.ID)
	fmt.Println("  expected error:", err)
	fmt.Println("  is ErrAlreadyCancelled:", errors.Is(err, ErrAlreadyCancelled))
}
