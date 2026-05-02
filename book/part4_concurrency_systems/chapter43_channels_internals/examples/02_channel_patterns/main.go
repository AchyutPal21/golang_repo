// FILE: book/part4_concurrency_systems/chapter43_channels_internals/examples/02_channel_patterns/main.go
// CHAPTER: 43 — Channels: Internals
// TOPIC: Done channel, semaphore, ownership transfer, one-time signal,
//        and the channel as a mutex pattern.
//
// Run (from the chapter folder):
//   go run ./examples/02_channel_patterns

package main

import (
	"fmt"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DONE CHANNEL — broadcast cancellation to many goroutines
//
// close(done) unblocks all goroutines blocked on <-done simultaneously.
// This is the idiomatic way to cancel a group of goroutines.
// ─────────────────────────────────────────────────────────────────────────────

func demodoneChan() {
	fmt.Println("=== Done channel (broadcast cancel) ===")

	done := make(chan struct{})
	var wg sync.WaitGroup

	worker := func(id int) {
		defer wg.Done()
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		ticks := 0
		for {
			select {
			case <-done:
				fmt.Printf("  worker %d: cancelled after %d ticks\n", id, ticks)
				return
			case <-ticker.C:
				ticks++
			}
		}
	}

	for i := range 3 {
		wg.Add(1)
		go worker(i)
	}

	time.Sleep(35 * time.Millisecond)
	close(done) // broadcasts to all 3 workers simultaneously
	wg.Wait()
}

// ─────────────────────────────────────────────────────────────────────────────
// SEMAPHORE — cap-N buffered channel limits concurrency
// ─────────────────────────────────────────────────────────────────────────────

type semaphore chan struct{}

func newSemaphore(n int) semaphore { return make(chan struct{}, n) }
func (s semaphore) acquire()       { s <- struct{}{} }
func (s semaphore) release()       { <-s }

func demoSemaphore() {
	fmt.Println()
	fmt.Println("=== Semaphore (max 3 concurrent) ===")

	sem := newSemaphore(3)
	var wg sync.WaitGroup
	var mu sync.Mutex
	peak := 0
	active := 0

	for i := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sem.acquire()
			defer sem.release()

			mu.Lock()
			active++
			if active > peak {
				peak = active
			}
			mu.Unlock()

			time.Sleep(10 * time.Millisecond) // simulate work
			mu.Lock()
			active--
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	fmt.Printf("  peak concurrent: %d  (capped at 3)\n", peak)
}

// ─────────────────────────────────────────────────────────────────────────────
// OWNERSHIP TRANSFER — channel carries data AND ownership
//
// When you send a value through a channel you transfer responsibility for it.
// The sender should not touch the value after sending.
// ─────────────────────────────────────────────────────────────────────────────

type Buffer struct {
	data []byte
}

func demoOwnershipTransfer() {
	fmt.Println()
	fmt.Println("=== Ownership transfer ===")

	ch := make(chan *Buffer, 1)

	// Producer creates a buffer and transfers ownership via channel.
	go func() {
		buf := &Buffer{data: []byte("hello from producer")}
		ch <- buf
		// buf must not be used after this point — ownership transferred
	}()

	// Consumer owns the buffer exclusively after receiving.
	buf := <-ch
	fmt.Printf("  consumer received: %q\n", buf.data)
	buf.data = append(buf.data, " (modified by consumer)"...)
	fmt.Printf("  consumer modified: %q\n", buf.data)
}

// ─────────────────────────────────────────────────────────────────────────────
// ONE-TIME SIGNAL — a channel closed exactly once acts as a broadcast event
// ─────────────────────────────────────────────────────────────────────────────

type startGun struct{ ch chan struct{} }

func newStartGun() *startGun { return &startGun{ch: make(chan struct{})} }
func (s *startGun) Ready() <-chan struct{} { return s.ch }
func (s *startGun) Fire() { close(s.ch) } // idempotent close not safe — fire once

func demoOneTimeSignal() {
	fmt.Println()
	fmt.Println("=== One-time signal (start gun) ===")

	gun := newStartGun()
	var wg sync.WaitGroup

	for i := range 5 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-gun.Ready() // all block here until Fire()
			fmt.Printf("  runner %d: started!\n", id)
		}(i)
	}

	time.Sleep(5 * time.Millisecond) // let all goroutines park on Ready()
	fmt.Println("  --- firing start gun ---")
	gun.Fire() // unblocks all 5 goroutines at once
	wg.Wait()
}

// ─────────────────────────────────────────────────────────────────────────────
// CHANNEL AS MUTEX — a cap-1 buffered channel acts like a mutex token
//
// Acquiring = sending a token; releasing = receiving it.
// Whoever holds the token has exclusive access.
// ─────────────────────────────────────────────────────────────────────────────

type tokenMutex chan struct{}

func newTokenMutex() tokenMutex {
	mu := make(tokenMutex, 1)
	mu <- struct{}{} // put the token in
	return mu
}
func (m tokenMutex) Lock()   { <-m }           // take the token
func (m tokenMutex) Unlock() { m <- struct{}{} } // return the token

func demoChannelMutex() {
	fmt.Println()
	fmt.Println("=== Channel as mutex ===")

	mu := newTokenMutex()
	counter := 0
	var wg sync.WaitGroup

	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			counter++
			mu.Unlock()
		}()
	}

	wg.Wait()
	fmt.Printf("  counter = %d  (always 100 — mutex via channel)\n", counter)
}

func main() {
	demodoneChan()
	demoSemaphore()
	demoOwnershipTransfer()
	demoOneTimeSignal()
	demoChannelMutex()
}
