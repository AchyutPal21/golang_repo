// Chapter 70 Exercise — CachedProductStore: cache layer over ProductRepository.
// Demonstrates: cache-aside reads, write-invalidation, metrics, and
// stale-while-revalidate via TTL.
package main

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ─── Sentinel errors ──────────────────────────────────────────────────────────

var ErrNotFound = errors.New("product not found")

// ─── Domain ───────────────────────────────────────────────────────────────────

type Product struct {
	ID       int64
	Name     string
	Category string
	PriceCent int
	Stock    int
}

// ProductRepository is the interface both the real store and the cache implement.
type ProductRepository interface {
	Create(ctx context.Context, p Product) (*Product, error)
	GetByID(ctx context.Context, id int64) (*Product, error)
	Update(ctx context.Context, p Product) error
	Delete(ctx context.Context, id int64) error
}

// ─── In-memory ProductStore ───────────────────────────────────────────────────

type memStore struct {
	mu      sync.RWMutex
	data    map[int64]*Product
	nextID  int64
	callLog []string // track which IDs were actually fetched
}

func newMemStore() *memStore {
	return &memStore{data: make(map[int64]*Product), nextID: 1}
}

func (s *memStore) Create(_ context.Context, p Product) (*Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.ID = s.nextID
	s.nextID++
	cp := p
	s.data[p.ID] = &cp
	return &cp, nil
}

func (s *memStore) GetByID(_ context.Context, id int64) (*Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.callLog = append(s.callLog, fmt.Sprintf("GetByID(%d)", id))
	p, ok := s.data[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (s *memStore) Update(_ context.Context, p Product) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[p.ID]; !ok {
		return ErrNotFound
	}
	cp := p
	s.data[p.ID] = &cp
	return nil
}

func (s *memStore) Delete(_ context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; !ok {
		return ErrNotFound
	}
	delete(s.data, id)
	return nil
}

func (s *memStore) storeCalls() int { return len(s.callLog) }

// ─── LRU internals used by CachedProductStore ─────────────────────────────────

type lruEntry struct {
	id        int64
	product   *Product
	expiresAt time.Time
}

// ─── CacheConfig ──────────────────────────────────────────────────────────────

// CacheConfig configures the cached product store.
type CacheConfig struct {
	MaxSize int
	TTL     time.Duration
}

// ─── CacheMetrics ─────────────────────────────────────────────────────────────

// CacheMetrics holds observable counters for the cache layer.
type CacheMetrics struct {
	Hits        int
	Misses      int
	Evictions   int
	TTLExpiries int
}

// ─── CachedProductStore ───────────────────────────────────────────────────────

// CachedProductStore wraps any ProductRepository with an LRU+TTL cache.
// It implements ProductRepository itself, so callers need not change.
type CachedProductStore struct {
	mu      sync.Mutex
	inner   ProductRepository
	cfg     CacheConfig
	items   map[int64]*list.Element
	order   *list.List
	metrics CacheMetrics
}

// NewCachedProductStore creates a caching wrapper around inner.
func NewCachedProductStore(inner ProductRepository, cfg CacheConfig) *CachedProductStore {
	return &CachedProductStore{
		inner: inner,
		cfg:   cfg,
		items: make(map[int64]*list.Element),
		order: list.New(),
	}
}

// Metrics returns a snapshot of cache performance counters.
func (c *CachedProductStore) Metrics() CacheMetrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.metrics
}

// ── Cache internals ───────────────────────────────────────────────────────────

func (c *CachedProductStore) store(p *Product) {
	if el, ok := c.items[p.ID]; ok {
		c.order.MoveToFront(el)
		el.Value.(*lruEntry).product = p
		el.Value.(*lruEntry).expiresAt = time.Now().Add(c.cfg.TTL)
		return
	}
	if c.order.Len() >= c.cfg.MaxSize {
		c.evict()
	}
	e := &lruEntry{id: p.ID, product: p, expiresAt: time.Now().Add(c.cfg.TTL)}
	el := c.order.PushFront(e)
	c.items[p.ID] = el
}

func (c *CachedProductStore) evict() {
	el := c.order.Back()
	if el == nil {
		return
	}
	c.order.Remove(el)
	delete(c.items, el.Value.(*lruEntry).id)
	c.metrics.Evictions++
}

func (c *CachedProductStore) invalidate(id int64) {
	if el, ok := c.items[id]; ok {
		c.order.Remove(el)
		delete(c.items, id)
	}
}

func (c *CachedProductStore) lookup(id int64) (*Product, bool) {
	el, ok := c.items[id]
	if !ok {
		return nil, false
	}
	e := el.Value.(*lruEntry)
	if time.Now().After(e.expiresAt) {
		// TTL expired — stale-while-revalidate: serve stale, mark expired.
		// For simplicity here we remove and force a fresh fetch.
		c.order.Remove(el)
		delete(c.items, id)
		c.metrics.TTLExpiries++
		return e.product, false // stale — caller should re-fetch
	}
	c.order.MoveToFront(el)
	return e.product, true
}

// ── ProductRepository implementation ─────────────────────────────────────────

func (c *CachedProductStore) Create(ctx context.Context, p Product) (*Product, error) {
	created, err := c.inner.Create(ctx, p)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.store(created) // pre-populate cache on create
	c.mu.Unlock()
	return created, nil
}

func (c *CachedProductStore) GetByID(ctx context.Context, id int64) (*Product, error) {
	c.mu.Lock()
	if p, ok := c.lookup(id); ok {
		c.metrics.Hits++
		cp := *p
		c.mu.Unlock()
		return &cp, nil
	}
	c.metrics.Misses++
	c.mu.Unlock()

	// Cache miss or TTL-expired — fetch from inner store.
	p, err := c.inner.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.store(p)
	c.mu.Unlock()
	return p, nil
}

func (c *CachedProductStore) Update(ctx context.Context, p Product) error {
	if err := c.inner.Update(ctx, p); err != nil {
		return err
	}
	c.mu.Lock()
	c.invalidate(p.ID) // invalidate on write
	c.mu.Unlock()
	return nil
}

func (c *CachedProductStore) Delete(ctx context.Context, id int64) error {
	if err := c.inner.Delete(ctx, id); err != nil {
		return err
	}
	c.mu.Lock()
	c.invalidate(id) // invalidate on delete
	c.mu.Unlock()
	return nil
}

// ─── main ─────────────────────────────────────────────────────────────────────

func main() {
	ctx := context.Background()
	store := newMemStore()
	cached := NewCachedProductStore(store, CacheConfig{MaxSize: 3, TTL: 200 * time.Millisecond})

	fmt.Println("=== CachedProductStore Demo ===")

	// Create products.
	p1, _ := cached.Create(ctx, Product{Name: "Widget", Category: "tools", PriceCent: 999, Stock: 50})
	p2, _ := cached.Create(ctx, Product{Name: "Gadget", Category: "electronics", PriceCent: 2499, Stock: 20})
	p3, _ := cached.Create(ctx, Product{Name: "Doohickey", Category: "tools", PriceCent: 149, Stock: 100})
	fmt.Printf("Created: #%d %s, #%d %s, #%d %s\n", p1.ID, p1.Name, p2.ID, p2.Name, p3.ID, p3.Name)

	// All three should be pre-populated in cache (Create does store()).
	m := cached.Metrics()
	fmt.Printf("After create — store_calls=%d  cache: hits=%d misses=%d\n",
		store.storeCalls(), m.Hits, m.Misses)

	fmt.Println("\n--- Cache hits (all from cache, no store calls) ---")
	for i := 0; i < 3; i++ {
		id := int64(i + 1)
		p, _ := cached.GetByID(ctx, id)
		fmt.Printf("  GetByID(%d) → %s\n", id, p.Name)
	}
	m = cached.Metrics()
	fmt.Printf("store_calls=%d  hits=%d  misses=%d\n", store.storeCalls(), m.Hits, m.Misses)

	fmt.Println("\n--- Eviction: add 4th product (MaxSize=3) ---")
	p4, _ := cached.Create(ctx, Product{Name: "Thingamajig", Category: "tools", PriceCent: 399})
	m = cached.Metrics()
	fmt.Printf("Created #%d  evictions=%d\n", p4.ID, m.Evictions)

	fmt.Println("\n--- Update invalidates cache entry ---")
	p2update := Product{ID: p2.ID, Name: "Gadget Pro", Category: "electronics", PriceCent: 3499, Stock: 15}
	cached.Update(ctx, p2update)
	callsBefore := store.storeCalls()
	p2fetched, _ := cached.GetByID(ctx, p2.ID)
	fmt.Printf("After update: name=%s price=%d  store_calls_delta=%d (re-fetched from store)\n",
		p2fetched.Name, p2fetched.PriceCent, store.storeCalls()-callsBefore)

	fmt.Println("\n--- Delete invalidates cache entry ---")
	cached.Delete(ctx, p3.ID)
	callsBefore = store.storeCalls()
	_, err := cached.GetByID(ctx, p3.ID)
	fmt.Printf("After delete: err=%v  store_calls_delta=%d\n", err, store.storeCalls()-callsBefore)

	fmt.Println("\n--- Stale-while-revalidate: TTL expiry ---")
	fresh, _ := cached.GetByID(ctx, p1.ID) // warm cache
	fmt.Printf("Fresh: %s (from cache)\n", fresh.Name)
	time.Sleep(250 * time.Millisecond) // let TTL expire
	callsBefore = store.storeCalls()
	reloaded, _ := cached.GetByID(ctx, p1.ID) // TTL expired → re-fetch
	fmt.Printf("Reloaded after TTL: %s  store_calls_delta=%d (re-fetched from store)\n",
		reloaded.Name, store.storeCalls()-callsBefore)

	fmt.Println("\n=== Final Metrics ===")
	m = cached.Metrics()
	fmt.Printf("Hits=%d  Misses=%d  Evictions=%d  TTL-Expiries=%d\n",
		m.Hits, m.Misses, m.Evictions, m.TTLExpiries)
}
