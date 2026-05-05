// FILE: book/part6_production_engineering/chapter96_distributed_building_blocks/examples/02_distributed_lock/main.go
// CHAPTER: 96 — Distributed Building Blocks
// TOPIC: Distributed lock with TTL, fencing tokens, and lock stealing detection.
//
// Run:
//   go run ./book/part6_production_engineering/chapter96_distributed_building_blocks/examples/02_distributed_lock

package main

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DISTRIBUTED LOCK
// ─────────────────────────────────────────────────────────────────────────────

var (
	ErrLockHeld    = errors.New("lock is held by another owner")
	ErrLockExpired = errors.New("lock has expired")
	ErrStaleFence  = errors.New("fencing token is stale")
)

type LockEntry struct {
	OwnerID      string
	FencingToken int64
	ExpiresAt    time.Time
}

func (l LockEntry) isExpired() bool {
	return time.Now().After(l.ExpiresAt)
}

type DistributedLock struct {
	mu           sync.Mutex
	entries      map[string]*LockEntry
	tokenCounter atomic.Int64
}

func NewDistributedLock() *DistributedLock {
	return &DistributedLock{entries: make(map[string]*LockEntry)}
}

type AcquiredLock struct {
	Key          string
	OwnerID      string
	FencingToken int64
	store        *DistributedLock
}

func (dl *DistributedLock) Acquire(key, ownerID string, ttl time.Duration) (*AcquiredLock, error) {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	existing, ok := dl.entries[key]
	if ok && !existing.isExpired() {
		return nil, fmt.Errorf("%w (held by %s, expires in %v)",
			ErrLockHeld, existing.OwnerID, time.Until(existing.ExpiresAt).Round(time.Millisecond))
	}

	token := dl.tokenCounter.Add(1)
	entry := &LockEntry{
		OwnerID:      ownerID,
		FencingToken: token,
		ExpiresAt:    time.Now().Add(ttl),
	}
	dl.entries[key] = entry
	return &AcquiredLock{Key: key, OwnerID: ownerID, FencingToken: token, store: dl}, nil
}

func (dl *DistributedLock) Release(key, ownerID string, token int64) error {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	existing, ok := dl.entries[key]
	if !ok {
		return ErrLockExpired
	}
	if existing.isExpired() {
		delete(dl.entries, key)
		return ErrLockExpired
	}
	if existing.OwnerID != ownerID || existing.FencingToken != token {
		return fmt.Errorf("%w: expected owner=%s token=%d, got owner=%s token=%d",
			ErrStaleFence, existing.OwnerID, existing.FencingToken, ownerID, token)
	}
	delete(dl.entries, key)
	return nil
}

func (dl *DistributedLock) CheckFencing(key string, token int64) error {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	existing, ok := dl.entries[key]
	if !ok || existing.isExpired() {
		return ErrLockExpired
	}
	if token < existing.FencingToken {
		return fmt.Errorf("%w: token %d < current %d", ErrStaleFence, token, existing.FencingToken)
	}
	return nil
}

func (dl *DistributedLock) Status(key string) string {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	e, ok := dl.entries[key]
	if !ok {
		return fmt.Sprintf("key=%q: FREE", key)
	}
	if e.isExpired() {
		return fmt.Sprintf("key=%q: EXPIRED (was held by %s)", key, e.OwnerID)
	}
	return fmt.Sprintf("key=%q: HELD by %s (token=%d, expires in %v)",
		key, e.OwnerID, e.FencingToken, time.Until(e.ExpiresAt).Round(time.Millisecond))
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Chapter 96: Distributed Lock ===")
	fmt.Println()

	lock := NewDistributedLock()
	key := "job:daily-report"

	// ── BASIC ACQUIRE/RELEASE ─────────────────────────────────────────────────
	fmt.Println("--- Basic acquire and release ---")
	acq, err := lock.Acquire(key, "worker-1", 5*time.Second)
	if err != nil {
		fmt.Printf("  worker-1 acquire: ERROR: %v\n", err)
	} else {
		fmt.Printf("  worker-1 acquired lock (token=%d)\n", acq.FencingToken)
	}
	fmt.Printf("  %s\n", lock.Status(key))

	// Another worker tries to acquire — should fail
	_, err = lock.Acquire(key, "worker-2", 5*time.Second)
	fmt.Printf("  worker-2 acquire: %v\n", err)
	fmt.Println()

	// Release
	if err := lock.Release(key, "worker-1", acq.FencingToken); err != nil {
		fmt.Printf("  worker-1 release: ERROR: %v\n", err)
	} else {
		fmt.Println("  worker-1 released lock")
	}
	fmt.Printf("  %s\n", lock.Status(key))
	fmt.Println()

	// ── TTL EXPIRY ────────────────────────────────────────────────────────────
	fmt.Println("--- TTL expiry (50ms TTL) ---")
	acq2, _ := lock.Acquire(key, "worker-1", 50*time.Millisecond)
	fmt.Printf("  worker-1 acquired (token=%d, TTL=50ms)\n", acq2.FencingToken)
	time.Sleep(80 * time.Millisecond)

	// Now worker-2 can acquire because TTL expired
	acq3, err := lock.Acquire(key, "worker-2", 5*time.Second)
	if err != nil {
		fmt.Printf("  worker-2 acquire after expiry: ERROR: %v\n", err)
	} else {
		fmt.Printf("  worker-2 acquired after TTL expiry (token=%d)\n", acq3.FencingToken)
	}
	fmt.Println()

	// ── FENCING TOKEN: STALE WRITE REJECTED ───────────────────────────────────
	fmt.Println("--- Fencing: stale worker-1 tries to write ---")
	// worker-1 still has old token (acq2.FencingToken)
	oldToken := acq2.FencingToken
	newToken := acq3.FencingToken
	fmt.Printf("  worker-1 old token=%d  worker-2 new token=%d\n", oldToken, newToken)
	if err := lock.CheckFencing(key, oldToken); err != nil {
		fmt.Printf("  worker-1 fencing check: REJECTED — %v\n", err)
	}
	if err := lock.CheckFencing(key, newToken); err != nil {
		fmt.Printf("  worker-2 fencing check: ERROR — %v\n", err)
	} else {
		fmt.Printf("  worker-2 fencing check: ACCEPTED (token=%d)\n", newToken)
	}
	fmt.Println()

	// ── DESIGN NOTES ─────────────────────────────────────────────────────────
	fmt.Println("--- Distributed lock design notes ---")
	fmt.Println(`  Correct usage pattern:
    1. Acquire lock → get fencing token
    2. Do work; send fencing token with every write to storage
    3. Storage rejects writes with token < last seen
    4. Release lock

  Production options:
    etcd:   lease + keepalive, strongly consistent (use for critical locks)
    Redis:  SETNX+EXPIRE (single node), Redlock (multi-node, controversial)
    Postgres: advisory locks (SELECT pg_try_advisory_lock(key))

  Never use:
    - Distributed lock for high-frequency operations (>100/s)
    - Lock without TTL (deadlock if holder crashes)
    - Lock as a serialization mechanism for business logic that should be
      redesigned (use idempotency keys instead)`)
}
