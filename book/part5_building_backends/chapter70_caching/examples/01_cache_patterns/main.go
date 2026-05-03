// Chapter 70 — Caching: LRU cache, TTL cache, cache-aside, write-through, write-behind.
package main

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

// ─── doubly-linked node ───────────────────────────────────────────────────────

type entry[K comparable, V any] struct {
	key   K
	value V
}

// ─── LRUCache ─────────────────────────────────────────────────────────────────

// LRUCache is a generic least-recently-used cache backed by a doubly-linked
// list + hash map.  Both operations are O(1).
type LRUCache[K comparable, V any] struct {
	mu       sync.Mutex
	capacity int
	items    map[K]*list.Element // key → list element
	order    *list.List          // front = most-recent, back = least-recent
}

// NewLRUCache creates a new LRU cache with the given capacity.
func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	if capacity <= 0 {
		panic("capacity must be > 0")
	}
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
		order:    list.New(),
	}
}

// Get returns the value for key and marks it as most-recently used.
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		return el.Value.(*entry[K, V]).value, true
	}
	var zero V
	return zero, false
}

// Put inserts or updates key; evicts the LRU entry when over capacity.
func (c *LRUCache[K, V]) Put(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		el.Value.(*entry[K, V]).value = value
		return
	}
	if c.order.Len() >= c.capacity {
		c.evict()
	}
	e := &entry[K, V]{key: key, value: value}
	el := c.order.PushFront(e)
	c.items[key] = el
}

// Delete removes the key from the cache.
func (c *LRUCache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.removeElement(el)
	}
}

// Len returns the current number of items in the cache.
func (c *LRUCache[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}

// evict removes the least-recently-used entry.  Must be called with mu held.
func (c *LRUCache[K, V]) evict() {
	el := c.order.Back()
	if el != nil {
		c.removeElement(el)
	}
}

// removeElement is a helper that deletes a list element and its map entry.
func (c *LRUCache[K, V]) removeElement(el *list.Element) {
	c.order.Remove(el)
	delete(c.items, el.Value.(*entry[K, V]).key)
}

// ─── TTLCache ─────────────────────────────────────────────────────────────────

type ttlEntry[V any] struct {
	value     V
	expiresAt time.Time
}

// TTLCache wraps an LRU cache and adds per-item expiry.
type TTLCache[K comparable, V any] struct {
	inner *LRUCache[K, ttlEntry[V]]
	ttl   time.Duration
}

// NewTTLCache creates a TTL cache with the given capacity and default TTL.
func NewTTLCache[K comparable, V any](capacity int, ttl time.Duration) *TTLCache[K, V] {
	return &TTLCache[K, V]{
		inner: NewLRUCache[K, ttlEntry[V]](capacity),
		ttl:   ttl,
	}
}

// Get returns the value only if it exists and has not expired.
func (c *TTLCache[K, V]) Get(key K) (V, bool) {
	te, ok := c.inner.Get(key)
	if !ok {
		var zero V
		return zero, false
	}
	if time.Now().After(te.expiresAt) {
		c.inner.Delete(key)
		var zero V
		return zero, false
	}
	return te.value, true
}

// Put inserts key with the cache's default TTL.
func (c *TTLCache[K, V]) Put(key K, value V) {
	c.inner.Put(key, ttlEntry[V]{value: value, expiresAt: time.Now().Add(c.ttl)})
}

// ─── Cache-Aside (Lazy Loading) ───────────────────────────────────────────────

// DataStore simulates a slow backend (database/API).
type DataStore struct {
	data  map[string]string
	calls int
}

func NewDataStore() *DataStore {
	return &DataStore{data: map[string]string{
		"user:1": `{"id":1,"name":"Alice","role":"admin"}`,
		"user:2": `{"id":2,"name":"Bob","role":"viewer"}`,
		"user:3": `{"id":3,"name":"Carol","role":"editor"}`,
	}}
}

// Load simulates a slow database read.
func (s *DataStore) Load(key string) (string, bool) {
	s.calls++
	v, ok := s.data[key]
	return v, ok
}

// CacheAsideService demonstrates the cache-aside (lazy-loading) pattern.
type CacheAsideService struct {
	cache *LRUCache[string, string]
	store *DataStore
}

// Get checks the cache first; on miss, loads from store and populates cache.
func (s *CacheAsideService) Get(key string) (string, bool) {
	if v, ok := s.cache.Get(key); ok {
		return v, true // cache hit
	}
	v, ok := s.store.Load(key)
	if ok {
		s.cache.Put(key, v) // populate cache (cache-aside)
	}
	return v, ok
}

// ─── Write-Through ────────────────────────────────────────────────────────────

// WriteThroughService writes to both cache and store synchronously.
type WriteThroughService struct {
	cache *LRUCache[string, string]
	store *DataStore
}

// Set writes to the store first, then updates the cache atomically.
func (s *WriteThroughService) Set(key, value string) {
	s.store.data[key] = value // write to store
	s.cache.Put(key, value)  // write to cache
}

// Get reads from cache; store is source-of-truth on miss (should not happen in WT).
func (s *WriteThroughService) Get(key string) (string, bool) {
	return s.cache.Get(key)
}

// ─── Write-Behind (Write-Back) ────────────────────────────────────────────────

// WriteBehindService writes immediately to cache, flushes to store asynchronously.
type WriteBehindService struct {
	mu    sync.Mutex
	cache *LRUCache[string, string]
	store *DataStore
	dirty map[string]string
}

func NewWriteBehindService(store *DataStore) *WriteBehindService {
	svc := &WriteBehindService{
		cache: NewLRUCache[string, string](100),
		store: store,
		dirty: make(map[string]string),
	}
	return svc
}

// Set writes immediately to cache and marks the key dirty.
func (s *WriteBehindService) Set(key, value string) {
	s.cache.Put(key, value)
	s.mu.Lock()
	s.dirty[key] = value
	s.mu.Unlock()
}

// Flush simulates the async background flush of dirty writes to the store.
func (s *WriteBehindService) Flush() int {
	s.mu.Lock()
	toFlush := s.dirty
	s.dirty = make(map[string]string)
	s.mu.Unlock()
	for k, v := range toFlush {
		s.store.data[k] = v
	}
	return len(toFlush)
}

// ─── main ─────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== LRU Cache Demo ===")
	lru := NewLRUCache[string, int](3)
	lru.Put("a", 1)
	lru.Put("b", 2)
	lru.Put("c", 3)
	fmt.Printf("Len=%d\n", lru.Len())

	// Access "a" to make it most-recent; "b" will be evicted next.
	v, _ := lru.Get("a")
	fmt.Printf("Get a=%d\n", v)
	lru.Put("d", 4) // evicts "b" (LRU)
	_, bOk := lru.Get("b")
	_, dOk := lru.Get("d")
	fmt.Printf("b evicted=%v  d present=%v  Len=%d\n", !bOk, dOk, lru.Len())

	fmt.Println("\n=== TTL Cache Demo ===")
	ttl := NewTTLCache[string, string](10, 100*time.Millisecond)
	ttl.Put("token:xyz", "user:42")
	if v, ok := ttl.Get("token:xyz"); ok {
		fmt.Printf("Before expiry: %s\n", v)
	}
	time.Sleep(150 * time.Millisecond)
	if _, ok := ttl.Get("token:xyz"); !ok {
		fmt.Println("After expiry: key gone (TTL expired)")
	}

	fmt.Println("\n=== Cache-Aside Pattern ===")
	store := NewDataStore()
	cas := &CacheAsideService{
		cache: NewLRUCache[string, string](10),
		store: store,
	}
	// First call: miss → load from store.
	u1, _ := cas.Get("user:1")
	fmt.Printf("1st call (miss) → %s  store_calls=%d\n", u1, store.calls)
	// Second call: hit → no store access.
	u1, _ = cas.Get("user:1")
	fmt.Printf("2nd call (hit)  → %s  store_calls=%d\n", u1, store.calls)
	u2, _ := cas.Get("user:2")
	fmt.Printf("3rd call (miss) → %s  store_calls=%d\n", u2, store.calls)

	fmt.Println("\n=== Write-Through Pattern ===")
	store2 := NewDataStore()
	wt := &WriteThroughService{
		cache: NewLRUCache[string, string](10),
		store: store2,
	}
	wt.Set("user:10", `{"id":10,"name":"Dave"}`)
	v2, inCache := wt.Get("user:10")
	_, inStore := store2.data["user:10"]
	fmt.Printf("Cache hit=%v  Store hit=%v  value=%s\n", inCache, inStore, v2)

	fmt.Println("\n=== Write-Behind Pattern ===")
	store3 := NewDataStore()
	wb := NewWriteBehindService(store3)
	wb.Set("user:20", `{"id":20,"name":"Eve"}`)
	wb.Set("user:21", `{"id":21,"name":"Frank"}`)
	_, inStoreBefore := store3.data["user:20"]
	fmt.Printf("In cache immediately=%v  In store before flush=%v\n",
		func() bool { _, ok := wb.cache.Get("user:20"); return ok }(),
		inStoreBefore)
	n := wb.Flush()
	_, inStoreAfter := store3.data["user:20"]
	fmt.Printf("Flushed %d dirty keys  In store after flush=%v\n", n, inStoreAfter)
}
