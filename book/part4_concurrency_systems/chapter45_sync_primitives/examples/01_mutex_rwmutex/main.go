// FILE: book/part4_concurrency_systems/chapter45_sync_primitives/examples/01_mutex_rwmutex/main.go
// CHAPTER: 45 — sync Primitives
// TOPIC: sync.Mutex, sync.RWMutex, lock ordering, trylock pattern,
//        and mutex embedding for zero-value usability.
//
// Run (from the chapter folder):
//   go run ./examples/01_mutex_rwmutex

package main

import (
	"fmt"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// sync.Mutex — exclusive lock
//
// Zero value is an unlocked mutex — never copy after first use.
// ─────────────────────────────────────────────────────────────────────────────

type SafeCounter struct {
	mu    sync.Mutex
	value int
}

func (c *SafeCounter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
}

func (c *SafeCounter) Add(n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value += n
}

func (c *SafeCounter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

func demoMutex() {
	fmt.Println("=== sync.Mutex ===")

	c := &SafeCounter{}
	var wg sync.WaitGroup
	for range 1000 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Inc()
		}()
	}
	wg.Wait()
	fmt.Printf("  counter = %d  (always 1000)\n", c.Value())
}

// ─────────────────────────────────────────────────────────────────────────────
// sync.RWMutex — multiple readers, exclusive writers
//
// Use when reads are much more frequent than writes.
// ─────────────────────────────────────────────────────────────────────────────

type SafeMap struct {
	mu sync.RWMutex
	m  map[string]string
}

func NewSafeMap() *SafeMap {
	return &SafeMap{m: make(map[string]string)}
}

func (s *SafeMap) Set(k, v string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[k] = v
}

func (s *SafeMap) Get(k string) (string, bool) {
	s.mu.RLock() // multiple goroutines can hold RLock simultaneously
	defer s.mu.RUnlock()
	v, ok := s.m[k]
	return v, ok
}

func (s *SafeMap) Delete(k string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, k)
}

func demoRWMutex() {
	fmt.Println()
	fmt.Println("=== sync.RWMutex ===")

	sm := NewSafeMap()
	sm.Set("name", "Alice")
	sm.Set("role", "admin")

	var wg sync.WaitGroup

	// 10 concurrent readers — all hold RLock at the same time.
	for i := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			name, _ := sm.Get("name")
			_ = name
		}(i)
	}

	// 2 concurrent writers — each blocks until all readers release.
	wg.Add(1)
	go func() {
		defer wg.Done()
		sm.Set("name", "Bob")
	}()

	wg.Wait()
	name, _ := sm.Get("name")
	fmt.Printf("  final name: %s\n", name)
}

// ─────────────────────────────────────────────────────────────────────────────
// LOCK ORDERING — preventing deadlocks with multiple mutexes
//
// Always acquire multiple mutexes in a consistent order across all goroutines.
// ─────────────────────────────────────────────────────────────────────────────

type Account struct {
	id      int
	mu      sync.Mutex
	balance int
}

// transfer acquires both mutexes in ID order to prevent deadlock.
func transfer(from, to *Account, amount int) bool {
	// Always lock the lower-ID account first.
	first, second := from, to
	if from.id > to.id {
		first, second = to, from
	}

	first.mu.Lock()
	defer first.mu.Unlock()
	second.mu.Lock()
	defer second.mu.Unlock()

	if from.balance < amount {
		return false
	}
	from.balance -= amount
	to.balance += amount
	return true
}

func demoLockOrdering() {
	fmt.Println()
	fmt.Println("=== Lock ordering (no deadlock) ===")

	alice := &Account{id: 1, balance: 1000}
	bob := &Account{id: 2, balance: 500}

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			transfer(alice, bob, 10)
		}()
		go func() {
			defer wg.Done()
			transfer(bob, alice, 5)
		}()
	}
	wg.Wait()

	alice.mu.Lock()
	bob.mu.Lock()
	total := alice.balance + bob.balance
	bob.mu.Unlock()
	alice.mu.Unlock()

	fmt.Printf("  alice=%d bob=%d total=%d (always 1500)\n",
		alice.balance, bob.balance, total)
}

// ─────────────────────────────────────────────────────────────────────────────
// TRYLOCK PATTERN — attempt lock without blocking
//
// sync.Mutex.TryLock() was added in Go 1.18.
// ─────────────────────────────────────────────────────────────────────────────

func demoTryLock() {
	fmt.Println()
	fmt.Println("=== TryLock (Go 1.18+) ===")

	var mu sync.Mutex

	// Hold the lock for a moment.
	mu.Lock()
	go func() {
		time.Sleep(30 * time.Millisecond)
		mu.Unlock()
	}()

	// TryLock fails while the goroutine holds it.
	if mu.TryLock() {
		fmt.Println("  trylock: acquired immediately (unexpected)")
		mu.Unlock()
	} else {
		fmt.Println("  trylock: failed — lock is held (expected)")
	}

	// After the goroutine releases, TryLock succeeds.
	time.Sleep(40 * time.Millisecond)
	if mu.TryLock() {
		fmt.Println("  trylock: acquired after release (expected)")
		mu.Unlock()
	}
}

func main() {
	demoMutex()
	demoRWMutex()
	demoLockOrdering()
	demoTryLock()
}
