// FILE: book/part5_building_backends/chapter71_redis/exercises/01_cache_store/main.go
// CHAPTER: 71 — Redis
// EXERCISE: Build a Redis-backed cache store with JSON serialization,
//           prefix-based bulk invalidation, health check with in-memory fallback,
//           and per-key TTL support.
//
// Run (from the chapter folder):
//   go run ./exercises/01_cache_store

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// ─────────────────────────────────────────────────────────────────────────────
// CACHE INTERFACE
// ─────────────────────────────────────────────────────────────────────────────

type Cache interface {
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Get(ctx context.Context, key string, dest any) error  // dest is a pointer
	Delete(ctx context.Context, key string) error
	DeleteByPrefix(ctx context.Context, prefix string) (int, error)
	Ping(ctx context.Context) error
}

var ErrCacheMiss = fmt.Errorf("cache miss")

// ─────────────────────────────────────────────────────────────────────────────
// REDIS CACHE
// ─────────────────────────────────────────────────────────────────────────────

type redisCache struct {
	rdb *redis.Client
}

func NewRedisCache(rdb *redis.Client) Cache {
	return &redisCache{rdb: rdb}
}

func (c *redisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return c.rdb.SetEx(ctx, key, b, ttl).Err()
}

func (c *redisCache) Get(ctx context.Context, key string, dest any) error {
	b, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return ErrCacheMiss
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dest)
}

func (c *redisCache) Delete(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}

func (c *redisCache) DeleteByPrefix(ctx context.Context, prefix string) (int, error) {
	var cursor uint64
	var deleted int
	for {
		keys, nextCursor, err := c.rdb.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return deleted, err
		}
		if len(keys) > 0 {
			n, err := c.rdb.Del(ctx, keys...).Result()
			if err != nil {
				return deleted, err
			}
			deleted += int(n)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return deleted, nil
}

func (c *redisCache) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// IN-MEMORY FALLBACK CACHE (used when Redis is unavailable)
// ─────────────────────────────────────────────────────────────────────────────

type memEntry struct {
	data      []byte
	expiresAt time.Time
}

type memCache struct {
	mu    sync.RWMutex
	store map[string]*memEntry
}

func NewMemCache() Cache {
	return &memCache{store: make(map[string]*memEntry)}
}

func (c *memCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	b, _ := json.Marshal(value)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = &memEntry{data: b, expiresAt: time.Now().Add(ttl)}
	return nil
}

func (c *memCache) Get(ctx context.Context, key string, dest any) error {
	c.mu.RLock()
	e, ok := c.store[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) {
		return ErrCacheMiss
	}
	return json.Unmarshal(e.data, dest)
}

func (c *memCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	delete(c.store, key)
	c.mu.Unlock()
	return nil
}

func (c *memCache) DeleteByPrefix(ctx context.Context, prefix string) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	count := 0
	for k := range c.store {
		if strings.HasPrefix(k, prefix) {
			delete(c.store, k)
			count++
		}
	}
	return count, nil
}

func (c *memCache) Ping(ctx context.Context) error { return nil }

// ─────────────────────────────────────────────────────────────────────────────
// RESILIENT CACHE — tries Redis, falls back to in-memory on failure
// ─────────────────────────────────────────────────────────────────────────────

type resilientCache struct {
	primary  Cache
	fallback Cache
}

func NewResilientCache(primary, fallback Cache) Cache {
	return &resilientCache{primary: primary, fallback: fallback}
}

func (r *resilientCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if err := r.primary.Ping(ctx); err != nil {
		fmt.Printf("  [cache] Redis unavailable, using fallback: %v\n", err)
		return r.fallback.Set(ctx, key, value, ttl)
	}
	return r.primary.Set(ctx, key, value, ttl)
}

func (r *resilientCache) Get(ctx context.Context, key string, dest any) error {
	if err := r.primary.Ping(ctx); err != nil {
		return r.fallback.Get(ctx, key, dest)
	}
	return r.primary.Get(ctx, key, dest)
}

func (r *resilientCache) Delete(ctx context.Context, key string) error {
	if err := r.primary.Ping(ctx); err != nil {
		return r.fallback.Delete(ctx, key)
	}
	return r.primary.Delete(ctx, key)
}

func (r *resilientCache) DeleteByPrefix(ctx context.Context, prefix string) (int, error) {
	if err := r.primary.Ping(ctx); err != nil {
		return r.fallback.DeleteByPrefix(ctx, prefix)
	}
	return r.primary.DeleteByPrefix(ctx, prefix)
}

func (r *resilientCache) Ping(ctx context.Context) error {
	return r.primary.Ping(ctx)
}

// ─────────────────────────────────────────────────────────────────────────────
// DOMAIN
// ─────────────────────────────────────────────────────────────────────────────

type Product struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Category string  `json:"category"`
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	mr, _ := miniredis.Run()
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	ctx := context.Background()
	cache := NewRedisCache(rdb)

	fmt.Println("=== Redis Cache Store ===")
	fmt.Println()

	// ── SET / GET ────────────────────────────────────────────────────────────
	fmt.Println("--- Set / Get ---")
	products := []Product{
		{ID: 1, Name: "Keyboard", Price: 129.99, Category: "electronics"},
		{ID: 2, Name: "Mouse", Price: 49.99, Category: "electronics"},
		{ID: 3, Name: "Desk Chair", Price: 299.00, Category: "furniture"},
	}
	for _, p := range products {
		cache.Set(ctx, fmt.Sprintf("product:%d", p.ID), p, 5*time.Minute)
	}

	var got Product
	err := cache.Get(ctx, "product:1", &got)
	if err == nil {
		fmt.Printf("  ✓ get product:1 → %s ($%.2f)\n", got.Name, got.Price)
	}

	err = cache.Get(ctx, "product:999", &got)
	fmt.Printf("  ✓ cache miss: %v\n", err)

	// ── PREFIX INVALIDATION ──────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- DeleteByPrefix (invalidate all products) ---")
	n, err := cache.DeleteByPrefix(ctx, "product:")
	if err == nil {
		fmt.Printf("  ✓ deleted %d keys with prefix 'product:'\n", n)
	}
	err = cache.Get(ctx, "product:1", &got)
	fmt.Printf("  ✓ product:1 after delete: %v\n", err)

	// ── TTL EXPIRY ──────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- TTL expiry ---")
	cache.Set(ctx, "ephemeral:key", map[string]string{"data": "temporary"}, 2*time.Second)
	var data map[string]string
	cache.Get(ctx, "ephemeral:key", &data)
	fmt.Printf("  before expiry: data=%v\n", data)

	// Fast-forward miniredis clock.
	mr.FastForward(3 * time.Second)
	err = cache.Get(ctx, "ephemeral:key", &data)
	fmt.Printf("  after expiry: %v\n", err)

	// ── RESILIENT CACHE ──────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Resilient cache (fallback to in-memory) ---")

	// Point a new client at a port with nothing listening to simulate Redis outage.
	deadRdb := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1", // nothing listening
		DialTimeout: 50 * time.Millisecond,
		ReadTimeout: 50 * time.Millisecond,
	})
	defer deadRdb.Close()
	deadCache := NewRedisCache(deadRdb)

	fallback := NewMemCache()
	resilient := NewResilientCache(deadCache, fallback)

	// Write should fall through to in-memory.
	resilient.Set(ctx, "product:10", Product{ID: 10, Name: "Fallback Widget", Price: 9.99, Category: "test"}, time.Minute)

	var fallbackProduct Product
	resilient.Get(ctx, "product:10", &fallbackProduct)
	fmt.Printf("  ✓ from fallback: %s ($%.2f)\n", fallbackProduct.Name, fallbackProduct.Price)

	// ── COMPLEX VALUES ───────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Complex JSON values in in-memory cache ---")
	mem := NewMemCache()
	type UserProfile struct {
		ID    int64    `json:"id"`
		Name  string   `json:"name"`
		Tags  []string `json:"tags"`
		Score float64  `json:"score"`
	}

	profile := UserProfile{ID: 42, Name: "Alice", Tags: []string{"admin", "power-user"}, Score: 9.8}
	mem.Set(ctx, "user:42:profile", profile, time.Hour)

	var gotProfile UserProfile
	mem.Get(ctx, "user:42:profile", &gotProfile)
	fmt.Printf("  ✓ complex value: id=%d name=%s tags=%v score=%.1f\n",
		gotProfile.ID, gotProfile.Name, gotProfile.Tags, gotProfile.Score)

	// ── PREFIX DELETE IN MEMORY ──────────────────────────────────────────────
	mem.Set(ctx, "user:1:profile", profile, time.Hour)
	mem.Set(ctx, "user:2:profile", profile, time.Hour)
	mem.Set(ctx, "session:abc", "tok", time.Hour)
	deleted, _ := mem.DeleteByPrefix(ctx, "user:")
	fmt.Printf("  ✓ deleted %d user:* keys\n", deleted)
	var check UserProfile
	fmt.Printf("  ✓ user:1 after delete: %v\n", mem.Get(ctx, "user:1:profile", &check))
}
