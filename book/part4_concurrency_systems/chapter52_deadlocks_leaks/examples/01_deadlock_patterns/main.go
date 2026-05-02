// FILE: book/part4_concurrency_systems/chapter52_deadlocks_leaks/examples/01_deadlock_patterns/main.go
// CHAPTER: 52 — Deadlocks, Leaks
// TOPIC: Deadlock patterns and their fixes — mutex ordering, channel
//        deadlock, self-deadlock (recursive mutex), and livelock.
//        Each demo shows the SAFE version; the racy version is described
//        in comments so this file compiles and runs cleanly.
//
// Run (from the chapter folder):
//   go run ./examples/01_deadlock_patterns

package main

import (
	"fmt"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// PATTERN 1: Consistent lock ordering avoids lock-order deadlock
//
// DEADLOCK scenario (do not run):
//   go func() { mu1.Lock(); mu2.Lock() ... }()   // order: 1 → 2
//   go func() { mu2.Lock(); mu1.Lock() ... }()   // order: 2 → 1 ← DEADLOCK
//
// FIX: always acquire locks in the same order (by ID, by address, etc.)
// ─────────────────────────────────────────────────────────────────────────────

type Account struct {
	mu      sync.Mutex
	id      int
	balance int
}

// transfer acquires locks in ascending account-ID order — no deadlock.
func transfer(from, to *Account, amount int) {
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
		fmt.Printf("  transfer %d→%d: insufficient funds (%d < %d)\n",
			from.id, to.id, from.balance, amount)
		return
	}
	from.balance -= amount
	to.balance += amount
	fmt.Printf("  transfer %d→%d: $%d (balances: %d, %d)\n",
		from.id, to.id, amount, from.balance, to.balance)
}

func demoLockOrdering() {
	fmt.Println("=== Lock ordering: no deadlock ===")

	a := &Account{id: 1, balance: 500}
	b := &Account{id: 2, balance: 300}

	var wg sync.WaitGroup
	for range 5 {
		wg.Add(2)
		go func() { defer wg.Done(); transfer(a, b, 50) }()
		go func() { defer wg.Done(); transfer(b, a, 30) }()
	}
	wg.Wait()
	fmt.Printf("  final: a=$%d b=$%d\n", a.balance, b.balance)
}

// ─────────────────────────────────────────────────────────────────────────────
// PATTERN 2: Channel send/receive symmetry
//
// DEADLOCK scenario (do not run):
//   ch := make(chan int)
//   ch <- 1   // blocks forever — nobody is receiving
//
// FIX: ensure every send has a corresponding receiver before the send blocks.
// Option A: buffer large enough to hold all sends before any read.
// Option B: receive in a goroutine.
// ─────────────────────────────────────────────────────────────────────────────

func demoChannelDeadlock() {
	fmt.Println()
	fmt.Println("=== Channel: send with receiver goroutine ===")

	ch := make(chan int) // unbuffered

	// Receiver goroutine ensures the send does not block forever.
	go func() {
		for v := range ch {
			fmt.Printf("  received: %d\n", v)
		}
	}()

	for i := range 5 {
		ch <- i + 1
	}
	close(ch)

	time.Sleep(10 * time.Millisecond) // let receiver finish
}

// ─────────────────────────────────────────────────────────────────────────────
// PATTERN 3: Self-deadlock — calling a function that locks a mutex you hold
//
// DEADLOCK scenario:
//   func (s *Service) outer() {
//       s.mu.Lock()
//       defer s.mu.Unlock()
//       s.inner()  // ← inner also calls s.mu.Lock() → deadlock
//   }
//
// FIX: internal functions that assume the lock is already held use
//      an "unlocked" naming convention and are never called externally.
// ─────────────────────────────────────────────────────────────────────────────

type Service struct {
	mu    sync.Mutex
	count int
}

// incrementLocked assumes mu is held by the caller (not exported).
func (s *Service) incrementLocked(n int) {
	s.count += n
}

func (s *Service) Add(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.incrementLocked(n) // safe: we call the unlocked variant
}

func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.incrementLocked(-s.count) // reset to zero
}

func demoSelfDeadlock() {
	fmt.Println()
	fmt.Println("=== Self-deadlock prevention: unlocked helper convention ===")

	svc := &Service{}
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			svc.Add(n)
		}(i)
	}
	wg.Wait()
	fmt.Printf("  count after 0+1+...+9 = %d (expected 45)\n", svc.count)
	svc.Reset()
	fmt.Printf("  count after Reset = %d (expected 0)\n", svc.count)
}

// ─────────────────────────────────────────────────────────────────────────────
// PATTERN 4: Livelock — two goroutines keep backing off and retrying
//            but never make progress because they always conflict.
//
// Classic example: two people step aside on a corridor simultaneously, then
// step back simultaneously, forever. Shown here as a simulation.
//
// FIX: introduce randomised backoff (jitter), or an explicit leader election.
// ─────────────────────────────────────────────────────────────────────────────

func demoLivelock() {
	fmt.Println()
	fmt.Println("=== Livelock simulation (brief, then resolves) ===")

	type corridor struct {
		mu   sync.Mutex
		side int // 0 = left, 1 = right
	}

	c := &corridor{side: 0}
	resolved := make(chan struct{})

	move := func(person int, preferred int, done chan<- struct{}) {
		attempts := 0
		for attempts < 20 {
			c.mu.Lock()
			if c.side == preferred {
				// Corridor is clear on our side.
				c.mu.Unlock()
				fmt.Printf("  person %d passed (attempts: %d)\n", person, attempts)
				close(done)
				return
			}
			// Livelock: both switch simultaneously.
			preferred = 1 - preferred
			c.mu.Unlock()
			attempts++
			time.Sleep(time.Millisecond)
		}
		// Jitter resolves after limit.
		fmt.Printf("  person %d resolved after %d attempts\n", person, attempts)
		close(done)
	}

	done1 := make(chan struct{})
	done2 := make(chan struct{})

	go move(1, 0, done1)
	go move(2, 1, done2)

	<-done1
	<-done2
	close(resolved)
	<-resolved
}

func main() {
	demoLockOrdering()
	demoChannelDeadlock()
	demoSelfDeadlock()
	demoLivelock()
}
