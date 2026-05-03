// FILE: book/part5_building_backends/chapter80_event_driven/examples/02_event_sourcing/main.go
// CHAPTER: 80 — Event-Driven Architecture in Go
// TOPIC: Event sourcing — append-only event log, aggregate rebuild, snapshots,
//        CQRS read model, and event projection.
//
// Run (from the chapter folder):
//   go run ./examples/02_event_sourcing

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// EVENT STORE — append-only log of domain events
// ─────────────────────────────────────────────────────────────────────────────

type Event struct {
	ID          int64
	AggregateID string
	Type        string
	Version     int // per-aggregate sequence number
	Data        any
	OccurredAt  time.Time
}

type EventStore struct {
	mu     sync.RWMutex
	events []*Event
	seq    atomic.Int64
}

func (es *EventStore) Append(aggregateID, eventType string, version int, data any) *Event {
	es.mu.Lock()
	defer es.mu.Unlock()
	e := &Event{
		ID:          es.seq.Add(1),
		AggregateID: aggregateID,
		Type:        eventType,
		Version:     version,
		Data:        data,
		OccurredAt:  time.Now(),
	}
	es.events = append(es.events, e)
	return e
}

func (es *EventStore) Load(aggregateID string) []*Event {
	es.mu.RLock()
	defer es.mu.RUnlock()
	var out []*Event
	for _, e := range es.events {
		if e.AggregateID == aggregateID {
			out = append(out, e)
		}
	}
	return out
}

func (es *EventStore) LoadFrom(aggregateID string, afterVersion int) []*Event {
	es.mu.RLock()
	defer es.mu.RUnlock()
	var out []*Event
	for _, e := range es.events {
		if e.AggregateID == aggregateID && e.Version > afterVersion {
			out = append(out, e)
		}
	}
	return out
}

func (es *EventStore) All() []*Event {
	es.mu.RLock()
	defer es.mu.RUnlock()
	out := make([]*Event, len(es.events))
	copy(out, es.events)
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// BANK ACCOUNT AGGREGATE — rebuilt from events
// ─────────────────────────────────────────────────────────────────────────────

type DepositedEvent struct{ Amount int }
type WithdrawnEvent struct{ Amount int }
type AccountOpenedEvent struct {
	Owner          string
	InitialBalance int
}

type BankAccount struct {
	ID      string
	Owner   string
	Balance int
	Version int // last applied event version
}

func NewBankAccount(id string) *BankAccount {
	return &BankAccount{ID: id}
}

// Apply rebuilds account state from a single event.
func (a *BankAccount) Apply(e *Event) {
	switch v := e.Data.(type) {
	case AccountOpenedEvent:
		a.Owner = v.Owner
		a.Balance = v.InitialBalance
	case DepositedEvent:
		a.Balance += v.Amount
	case WithdrawnEvent:
		a.Balance -= v.Amount
	}
	a.Version = e.Version
}

// Rebuild loads all events and reconstructs the aggregate.
func Rebuild(id string, store *EventStore) *BankAccount {
	acc := NewBankAccount(id)
	for _, e := range store.Load(id) {
		acc.Apply(e)
	}
	return acc
}

// Command methods record events; they do NOT modify state directly.

func (a *BankAccount) Open(store *EventStore, owner string, initialBalance int) {
	store.Append(a.ID, "account.opened", a.Version+1, AccountOpenedEvent{
		Owner:          owner,
		InitialBalance: initialBalance,
	})
}

func (a *BankAccount) Deposit(store *EventStore, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("deposit amount must be positive")
	}
	store.Append(a.ID, "account.deposited", a.Version+1, DepositedEvent{Amount: amount})
	return nil
}

func (a *BankAccount) Withdraw(store *EventStore, amount int) error {
	if amount > a.Balance {
		return fmt.Errorf("insufficient funds: balance=%d requested=%d", a.Balance, amount)
	}
	store.Append(a.ID, "account.withdrawn", a.Version+1, WithdrawnEvent{Amount: amount})
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SNAPSHOT — avoid replaying from the beginning every time
// ─────────────────────────────────────────────────────────────────────────────

type Snapshot struct {
	AggregateID string
	Version     int
	State       *BankAccount
	TakenAt     time.Time
}

type SnapshotStore struct {
	mu        sync.RWMutex
	snapshots map[string]*Snapshot
}

func NewSnapshotStore() *SnapshotStore {
	return &SnapshotStore{snapshots: make(map[string]*Snapshot)}
}

func (ss *SnapshotStore) Save(s *Snapshot) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.snapshots[s.AggregateID] = s
}

func (ss *SnapshotStore) Load(aggregateID string) (*Snapshot, bool) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	s, ok := ss.snapshots[aggregateID]
	return s, ok
}

// RebuildFromSnapshot uses the latest snapshot then replays only newer events.
func RebuildFromSnapshot(id string, store *EventStore, ss *SnapshotStore) *BankAccount {
	if snap, ok := ss.Load(id); ok {
		// Deep copy the snapshot state.
		acc := *snap.State
		for _, e := range store.LoadFrom(id, snap.Version) {
			acc.Apply(e)
		}
		return &acc
	}
	return Rebuild(id, store)
}

// ─────────────────────────────────────────────────────────────────────────────
// CQRS READ MODEL — projection updated as events flow
// ─────────────────────────────────────────────────────────────────────────────

type AccountSummary struct {
	AccountID  string
	Owner      string
	Balance    int
	TxCount    int
	LastUpdate time.Time
}

type AccountReadModel struct {
	mu       sync.RWMutex
	accounts map[string]*AccountSummary
}

func NewAccountReadModel() *AccountReadModel {
	return &AccountReadModel{accounts: make(map[string]*AccountSummary)}
}

func (rm *AccountReadModel) Project(e *Event) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	summary := rm.accounts[e.AggregateID]
	if summary == nil {
		summary = &AccountSummary{AccountID: e.AggregateID}
		rm.accounts[e.AggregateID] = summary
	}
	switch v := e.Data.(type) {
	case AccountOpenedEvent:
		summary.Owner = v.Owner
		summary.Balance = v.InitialBalance
	case DepositedEvent:
		summary.Balance += v.Amount
		summary.TxCount++
	case WithdrawnEvent:
		summary.Balance -= v.Amount
		summary.TxCount++
	}
	summary.LastUpdate = e.OccurredAt
}

func (rm *AccountReadModel) Get(id string) (*AccountSummary, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	s, ok := rm.accounts[id]
	return s, ok
}

// Rebuild replays all events from the store into the read model.
func (rm *AccountReadModel) Rebuild(store *EventStore) {
	for _, e := range store.All() {
		rm.Project(e)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Event Sourcing + CQRS ===")
	fmt.Println()

	store := &EventStore{}
	snapshots := NewSnapshotStore()
	readModel := NewAccountReadModel()

	// ── COMMAND SIDE: record events ───────────────────────────────────────────
	fmt.Println("--- Command side: record events ---")

	acc := NewBankAccount("acc-1")
	acc.Open(store, "Alice", 1000)
	acc.Apply(store.Load("acc-1")[0]) // apply opened event so balance is known

	acc.Deposit(store, 500)
	acc.Apply(store.Load("acc-1")[1])

	acc.Deposit(store, 300)
	acc.Apply(store.Load("acc-1")[2])

	err := acc.Withdraw(store, 200)
	if err != nil {
		fmt.Println("  withdraw error:", err)
	}
	acc.Apply(store.Load("acc-1")[3])

	// Attempt overdraft.
	err = acc.Withdraw(store, 9999)
	fmt.Printf("  overdraft attempt: %v\n", err)

	fmt.Printf("  events recorded: %d\n", len(store.Load("acc-1")))
	fmt.Printf("  current balance (in-memory): %d\n", acc.Balance)

	// ── QUERY SIDE: rebuild from events ──────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Query side: rebuild aggregate from event log ---")

	rebuilt := Rebuild("acc-1", store)
	fmt.Printf("  rebuilt owner=%s balance=%d version=%d\n", rebuilt.Owner, rebuilt.Balance, rebuilt.Version)

	// ── SNAPSHOT ──────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Snapshot: avoid replaying from the beginning ---")

	// Take snapshot at current version (deep-copy by value).
	snapCopy := *rebuilt
	snapshots.Save(&Snapshot{
		AggregateID: "acc-1",
		Version:     snapCopy.Version,
		State:       &snapCopy,
		TakenAt:     time.Now(),
	})
	snapVersion := snapCopy.Version

	// Add two more events after the snapshot.
	// Append directly so we don't re-apply to the snapshotted copy.
	store.Append("acc-1", "account.deposited", snapVersion+1, DepositedEvent{Amount: 100})
	store.Append("acc-1", "account.deposited", snapVersion+2, DepositedEvent{Amount: 50})

	// Rebuild using snapshot + delta.
	fast := RebuildFromSnapshot("acc-1", store, snapshots)
	full := Rebuild("acc-1", store)
	fmt.Printf("  snapshot at version %d; 2 new events replayed\n", snapVersion)
	fmt.Printf("  fast-rebuilt balance=%d version=%d\n", fast.Balance, fast.Version)
	fmt.Printf("  full-rebuild   balance=%d version=%d (should match)\n", full.Balance, full.Version)

	// ── CQRS READ MODEL ───────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- CQRS: read model updated by projections ---")

	// Start a second account.
	acc3 := NewBankAccount("acc-2")
	acc3.Open(store, "Bob", 500)
	acc3.Apply(store.Load("acc-2")[0])
	acc3.Deposit(store, 200)

	// Rebuild read model from the full event log.
	readModel.Rebuild(store)

	if s, ok := readModel.Get("acc-1"); ok {
		fmt.Printf("  acc-1: owner=%s balance=%d txns=%d\n", s.Owner, s.Balance, s.TxCount)
	}
	if s, ok := readModel.Get("acc-2"); ok {
		fmt.Printf("  acc-2: owner=%s balance=%d txns=%d\n", s.Owner, s.Balance, s.TxCount)
	}

	// ── EVENT LOG AUDIT ───────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Full audit log ---")
	for _, e := range store.All() {
		fmt.Printf("  [%d] %s v%d %s\n", e.ID, e.AggregateID, e.Version, e.Type)
	}

	// ── CQRS SUMMARY ─────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- CQRS / Event Sourcing summary ---")
	fmt.Println(`  Write side:  commands mutate aggregates by appending events.
  Read side:   projections listen to events and update read models.
  Rebuilding:  replay events from position 0 (or from snapshot) to restore state.
  Benefits:    full audit trail, temporal queries, independent read/write scaling.
  Trade-offs:  eventual consistency between write and read models.`)
}
