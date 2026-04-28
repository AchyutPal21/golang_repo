// FILE: book/part2_core_language/chapter15_defer_panic_recover/examples/03_patterns/main.go
// CHAPTER: 15 — defer, panic, recover
// TOPIC: Production defer patterns: timer/trace, transaction rollback,
//        error annotation, mutex unlock, span-like tracing.
//
// Run (from the chapter folder):
//   go run ./examples/03_patterns

package main

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// --- Timer / trace ---

// elapsed returns a function that prints elapsed time when called.
// Usage: defer elapsed("operation")()
// Note the double (): the outer call runs at defer registration (records
// start time), the inner func runs at function exit (measures elapsed).
func elapsed(name string) func() {
	start := time.Now()
	return func() {
		fmt.Printf("[trace] %s took %v\n", name, time.Since(start))
	}
}

func expensiveOperation() {
	defer elapsed("expensiveOperation")()
	time.Sleep(10 * time.Millisecond)
}

// --- Transaction rollback ---

type txState int

const (
	txOpen txState = iota
	txCommitted
	txRolledBack
)

type Tx struct {
	mu    sync.Mutex
	state txState
	ops   []string
}

func (t *Tx) Exec(op string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state != txOpen {
		return errors.New("transaction not open")
	}
	t.ops = append(t.ops, op)
	return nil
}

func (t *Tx) Commit() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state != txOpen {
		return errors.New("transaction not open")
	}
	t.state = txCommitted
	fmt.Println("[tx] committed:", t.ops)
	return nil
}

func (t *Tx) Rollback() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state == txOpen {
		t.state = txRolledBack
		fmt.Println("[tx] rolled back:", t.ops)
	}
}

func beginTx() *Tx { return &Tx{state: txOpen} }

// runWithTx demonstrates the defer-rollback pattern:
// defer rollback immediately after begin; only call commit on success.
func runWithTx(fail bool) error {
	tx := beginTx()
	defer tx.Rollback() // safe to call even after Commit (idempotent check)

	_ = tx.Exec("INSERT users ...")
	_ = tx.Exec("UPDATE accounts ...")

	if fail {
		return errors.New("simulated failure")
	}

	return tx.Commit()
}

// --- Error annotation via named return ---

func annotate(op string) func(*error) {
	return func(err *error) {
		if *err != nil {
			*err = fmt.Errorf("%s: %w", op, *err)
		}
	}
}

func loadConfig(path string) (cfg map[string]string, err error) {
	defer annotate("loadConfig")(&err)

	if path == "" {
		return nil, errors.New("path is empty")
	}
	// Simulate: would read file here.
	return map[string]string{"host": "localhost"}, nil
}

// --- Mutex unlock via defer ---

type Cache struct {
	mu   sync.RWMutex
	data map[string]string
}

func (c *Cache) Set(k, v string) {
	c.mu.Lock()
	defer c.mu.Unlock() // always unlocks, even if Set panics
	c.data[k] = v
}

func (c *Cache) Get(k string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.data[k]
	return v, ok
}

func main() {
	// --- timer ---
	fmt.Println("=== timer ===")
	expensiveOperation()

	fmt.Println()

	// --- transaction ---
	fmt.Println("=== transaction rollback ===")
	if err := runWithTx(false); err != nil {
		fmt.Println("tx err:", err)
	} else {
		fmt.Println("tx success")
	}
	if err := runWithTx(true); err != nil {
		fmt.Println("tx err:", err)
	}

	fmt.Println()

	// --- error annotation ---
	fmt.Println("=== error annotation ===")
	_, err := loadConfig("")
	fmt.Println("annotated error:", err)
	_, err = loadConfig("/etc/app.conf")
	fmt.Println("success:", err)

	fmt.Println()

	// --- cache with mutex defer ---
	fmt.Println("=== cache ===")
	c := &Cache{data: make(map[string]string)}
	c.Set("key", "value")
	if v, ok := c.Get("key"); ok {
		fmt.Println("cache get:", v)
	}
	if _, ok := c.Get("missing"); !ok {
		fmt.Println("cache miss: ok")
	}
}
