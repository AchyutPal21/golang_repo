// FILE: book/part3_designing_software/chapter31_creational_patterns/examples/02_singleton_prototype/main.go
// CHAPTER: 31 — Creational Patterns
// TOPIC: Singleton (package-level once) and Prototype (deep copy / Clone) in Go.
//
// Run (from the chapter folder):
//   go run ./examples/02_singleton_prototype

package main

import (
	"fmt"
	"sync"
)

// ─────────────────────────────────────────────────────────────────────────────
// SINGLETON
//
// Go's idiomatic singleton: a package-level variable initialised exactly once
// with sync.Once.  Never use init() for singletons — it runs unconditionally.
//
// Rule: prefer dependency injection over singletons. Use singletons only for
// things that are genuinely global and stateless (loggers, config readers, etc.)
// ─────────────────────────────────────────────────────────────────────────────

type AppConfig struct {
	DatabaseURL string
	Port        int
	Debug       bool
}

var (
	configOnce     sync.Once
	globalConfig   *AppConfig
)

// GetConfig returns the single shared config, initialised exactly once.
func GetConfig() *AppConfig {
	configOnce.Do(func() {
		// In production this would read env vars, files, or flags.
		globalConfig = &AppConfig{
			DatabaseURL: "postgres://localhost/app",
			Port:        8080,
			Debug:       false,
		}
		fmt.Println("  [CONFIG] initialised (runs once)")
	})
	return globalConfig
}

// ── Connection pool: a singleton that manages shared resources ────────────────

type ConnectionPool struct {
	mu      sync.Mutex
	conns   []string // simulated connections
	maxSize int
}

var (
	poolOnce sync.Once
	pool     *ConnectionPool
)

func GetPool() *ConnectionPool {
	poolOnce.Do(func() {
		pool = &ConnectionPool{maxSize: 3}
		for i := 1; i <= 3; i++ {
			pool.conns = append(pool.conns, fmt.Sprintf("conn-%d", i))
		}
		fmt.Println("  [POOL] connection pool created")
	})
	return pool
}

func (p *ConnectionPool) Acquire() (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.conns) == 0 {
		return "", false
	}
	conn := p.conns[0]
	p.conns = p.conns[1:]
	return conn, true
}

func (p *ConnectionPool) Release(conn string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.conns = append(p.conns, conn)
}

func (p *ConnectionPool) Available() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.conns)
}

// ─────────────────────────────────────────────────────────────────────────────
// PROTOTYPE
//
// The Prototype pattern creates new objects by cloning an existing one.
// In Go: implement a Clone() method that returns a deep copy.
// Useful when construction is expensive and many variants start from a base.
// ─────────────────────────────────────────────────────────────────────────────

// DocumentTemplate is the prototype — a base document pre-configured with styles.
type DocumentTemplate struct {
	Title    string
	Author   string
	Sections []string
	Metadata map[string]string
}

// Clone returns a deep copy — safe to modify independently of the original.
func (d *DocumentTemplate) Clone() *DocumentTemplate {
	sections := make([]string, len(d.Sections))
	copy(sections, d.Sections)

	metadata := make(map[string]string, len(d.Metadata))
	for k, v := range d.Metadata {
		metadata[k] = v
	}

	return &DocumentTemplate{
		Title:    d.Title,
		Author:   d.Author,
		Sections: sections,
		Metadata: metadata,
	}
}

func (d *DocumentTemplate) AddSection(s string) { d.Sections = append(d.Sections, s) }
func (d *DocumentTemplate) SetMeta(k, v string)  { d.Metadata[k] = v }

func (d *DocumentTemplate) String() string {
	return fmt.Sprintf("Title=%q Author=%q sections=%v meta=%v",
		d.Title, d.Author, d.Sections, d.Metadata)
}

// ── Object pool (related to prototype): reuse expensive objects ───────────────

type ExpensiveProcessor struct {
	id    int
	state string
}

func newExpensiveProcessor(id int) *ExpensiveProcessor {
	fmt.Printf("  [PROCESSOR] allocating processor-%d (expensive!)\n", id)
	return &ExpensiveProcessor{id: id, state: "idle"}
}

func (p *ExpensiveProcessor) Reset() { p.state = "idle" }
func (p *ExpensiveProcessor) Process(data string) {
	p.state = "busy"
	fmt.Printf("  [PROCESSOR-%d] processing: %q\n", p.id, data)
	p.state = "idle"
}

type ProcessorPool struct {
	mu        sync.Mutex
	available []*ExpensiveProcessor
	nextID    int
}

func (p *ProcessorPool) Get() *ExpensiveProcessor {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.available) > 0 {
		proc := p.available[len(p.available)-1]
		p.available = p.available[:len(p.available)-1]
		return proc
	}
	p.nextID++
	return newExpensiveProcessor(p.nextID)
}

func (p *ProcessorPool) Put(proc *ExpensiveProcessor) {
	proc.Reset()
	p.mu.Lock()
	p.available = append(p.available, proc)
	p.mu.Unlock()
}

func main() {
	fmt.Println("=== Singleton: AppConfig ===")
	cfg1 := GetConfig()
	cfg2 := GetConfig()
	fmt.Printf("  same pointer: %v\n", cfg1 == cfg2)
	fmt.Printf("  port: %d  debug: %v\n", cfg1.Port, cfg1.Debug)

	fmt.Println()
	fmt.Println("=== Singleton: ConnectionPool ===")
	p := GetPool()
	_ = GetPool() // second call — no re-init
	fmt.Printf("  available: %d\n", p.Available())
	conn1, _ := p.Acquire()
	conn2, _ := p.Acquire()
	fmt.Printf("  acquired: %s, %s  remaining: %d\n", conn1, conn2, p.Available())
	p.Release(conn1)
	fmt.Printf("  after release: %d\n", p.Available())

	fmt.Println()
	fmt.Println("=== Prototype: document template cloning ===")
	base := &DocumentTemplate{
		Title:    "Quarterly Report Template",
		Author:   "Template Team",
		Sections: []string{"Executive Summary", "Financial Results"},
		Metadata: map[string]string{"version": "1.0", "confidential": "true"},
	}
	fmt.Println("  base:", base)

	// Clone and customise for Q1
	q1 := base.Clone()
	q1.Title = "Q1 2026 Report"
	q1.Author = "Alice"
	q1.AddSection("Q1 Highlights")
	q1.SetMeta("quarter", "Q1")

	// Clone and customise for Q2
	q2 := base.Clone()
	q2.Title = "Q2 2026 Report"
	q2.Author = "Bob"
	q2.AddSection("Q2 Highlights")
	q2.SetMeta("quarter", "Q2")

	fmt.Println("  base (unchanged):", base)
	fmt.Println("  q1:", q1)
	fmt.Println("  q2:", q2)

	fmt.Println()
	fmt.Println("=== Object Pool: expensive processors ===")
	pool2 := &ProcessorPool{}
	p1 := pool2.Get() // allocates new
	p2 := pool2.Get() // allocates new
	p1.Process("batch-job-1")
	p2.Process("batch-job-2")
	pool2.Put(p1)
	pool2.Put(p2)

	p3 := pool2.Get() // reuses from pool — no allocation message
	fmt.Printf("  reused processor id: %d\n", p3.id)
	p3.Process("batch-job-3")
	pool2.Put(p3)
}
