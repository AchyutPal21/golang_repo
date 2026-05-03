// FILE: book/part5_building_backends/chapter71_redis/examples/02_redis_patterns/main.go
// CHAPTER: 71 — Redis
// TOPIC: Production Redis patterns — session store, rate limiter,
//        distributed lock, pub/sub, and pipelining.
//        Uses miniredis (pure-Go) — no external Redis required.
//
// Run (from the chapter folder):
//   go run ./examples/02_redis_patterns

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newClient() (*redis.Client, *miniredis.Miniredis) {
	mr, _ := miniredis.Run()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return rdb, mr
}

// ─────────────────────────────────────────────────────────────────────────────
// SESSION STORE
// ─────────────────────────────────────────────────────────────────────────────

type Session struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt int64  `json:"created_at"`
}

type SessionStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewSessionStore(rdb *redis.Client, ttl time.Duration) *SessionStore {
	return &SessionStore{rdb: rdb, ttl: ttl}
}

func (s *SessionStore) Create(ctx context.Context, token string, sess Session) error {
	b, _ := json.Marshal(sess)
	return s.rdb.SetEx(ctx, "session:"+token, b, s.ttl).Err()
}

func (s *SessionStore) Get(ctx context.Context, token string) (*Session, error) {
	b, err := s.rdb.Get(ctx, "session:"+token).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, err
	}
	var sess Session
	json.Unmarshal(b, &sess)
	return &sess, nil
}

func (s *SessionStore) Delete(ctx context.Context, token string) error {
	return s.rdb.Del(ctx, "session:"+token).Err()
}

func demoSessionStore(ctx context.Context, rdb *redis.Client) {
	fmt.Println("--- Session Store ---")
	store := NewSessionStore(rdb, 24*time.Hour)

	sess := Session{UserID: "u-123", Email: "alice@example.com", Role: "admin", CreatedAt: time.Now().Unix()}
	store.Create(ctx, "tok-abc", sess)

	got, err := store.Get(ctx, "tok-abc")
	if err == nil {
		fmt.Printf("  ✓ session: user=%s role=%s\n", got.UserID, got.Role)
	}

	store.Delete(ctx, "tok-abc")
	_, err = store.Get(ctx, "tok-abc")
	fmt.Printf("  ✓ after logout: %v\n", err)
}

// ─────────────────────────────────────────────────────────────────────────────
// RATE LIMITER — sliding window counter via INCR + EXPIRE
// ─────────────────────────────────────────────────────────────────────────────

type RateLimiter struct {
	rdb      *redis.Client
	limit    int
	window   time.Duration
}

func NewRateLimiter(rdb *redis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{rdb: rdb, limit: limit, window: window}
}

// Allow returns (allowed bool, current count, remaining).
func (r *RateLimiter) Allow(ctx context.Context, key string) (bool, int, int) {
	rk := "rate:" + key
	pipe := r.rdb.Pipeline()
	incr := pipe.Incr(ctx, rk)
	pipe.Expire(ctx, rk, r.window)
	pipe.Exec(ctx)

	count := int(incr.Val())
	remaining := r.limit - count
	if remaining < 0 {
		remaining = 0
	}
	return count <= r.limit, count, remaining
}

func demoRateLimiter(ctx context.Context, rdb *redis.Client) {
	fmt.Println()
	fmt.Println("--- Rate Limiter (5 req / window) ---")
	rl := NewRateLimiter(rdb, 5, time.Minute)

	for i := 1; i <= 7; i++ {
		allowed, count, remaining := rl.Allow(ctx, "user:alice")
		status := "✓ allowed"
		if !allowed {
			status = "✗ blocked"
		}
		fmt.Printf("  req %d: %s  count=%d remaining=%d\n", i, status, count, remaining)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DISTRIBUTED LOCK — SET NX EX + Lua unlock script
// ─────────────────────────────────────────────────────────────────────────────

const unlockScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end`

type DistLock struct {
	rdb   *redis.Client
	key   string
	value string
	ttl   time.Duration
}

func NewDistLock(rdb *redis.Client, key, value string, ttl time.Duration) *DistLock {
	return &DistLock{rdb: rdb, key: "lock:" + key, value: value, ttl: ttl}
}

func (l *DistLock) Acquire(ctx context.Context) (bool, error) {
	ok, err := l.rdb.SetNX(ctx, l.key, l.value, l.ttl).Result()
	return ok, err
}

func (l *DistLock) Release(ctx context.Context) error {
	script := redis.NewScript(unlockScript)
	_, err := script.Run(ctx, l.rdb, []string{l.key}, l.value).Result()
	return err
}

func demoDistLock(ctx context.Context, rdb *redis.Client) {
	fmt.Println()
	fmt.Println("--- Distributed Lock ---")

	lock1 := NewDistLock(rdb, "payment:ord-1", "worker-a", 10*time.Second)
	lock2 := NewDistLock(rdb, "payment:ord-1", "worker-b", 10*time.Second)

	ok1, _ := lock1.Acquire(ctx)
	ok2, _ := lock2.Acquire(ctx) // should fail — already locked
	fmt.Printf("  worker-a acquired: %v\n", ok1)
	fmt.Printf("  worker-b acquired: %v (lock held by worker-a)\n", ok2)

	lock1.Release(ctx)
	ok3, _ := lock2.Acquire(ctx) // now succeeds
	fmt.Printf("  worker-b after release: %v\n", ok3)
	lock2.Release(ctx)
}

// ─────────────────────────────────────────────────────────────────────────────
// PUB/SUB
// ─────────────────────────────────────────────────────────────────────────────

func demoPubSub(ctx context.Context, rdb *redis.Client) {
	fmt.Println()
	fmt.Println("--- Pub/Sub ---")

	sub := rdb.Subscribe(ctx, "orders")
	ch := sub.Channel()

	received := make([]string, 0, 3)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 3; i++ {
			msg := <-ch
			received = append(received, msg.Payload)
		}
	}()

	// Give subscriber goroutine a moment to start.
	time.Sleep(10 * time.Millisecond)

	events := []string{"order.created:ord-1", "order.shipped:ord-2", "order.delivered:ord-1"}
	for _, e := range events {
		rdb.Publish(ctx, "orders", e)
	}

	wg.Wait()
	sub.Close()

	fmt.Printf("  received %d messages:\n", len(received))
	for _, r := range received {
		fmt.Printf("  → %s\n", r)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// PIPELINE — batch multiple commands in one round trip
// ─────────────────────────────────────────────────────────────────────────────

func demoPipeline(ctx context.Context, rdb *redis.Client) {
	fmt.Println()
	fmt.Println("--- Pipeline (batch commands) ---")

	// Without pipeline: N round trips.
	// With pipeline: 1 round trip.
	pipe := rdb.Pipeline()
	cmds := make([]*redis.IntCmd, 5)
	for i := 0; i < 5; i++ {
		cmds[i] = pipe.Incr(ctx, "pipeline:counter")
	}
	pipe.Exec(ctx)

	vals := make([]int64, 5)
	for i, cmd := range cmds {
		vals[i] = cmd.Val()
	}
	fmt.Printf("  5 INCRs in 1 round trip: %v\n", vals)

	// TxPipeline — atomic: all commands or none.
	txPipe := rdb.TxPipeline()
	txPipe.Set(ctx, "tx:a", 100, 0)
	txPipe.Set(ctx, "tx:b", 200, 0)
	txPipe.Exec(ctx)

	a, _ := rdb.Get(ctx, "tx:a").Result()
	b, _ := rdb.Get(ctx, "tx:b").Result()
	fmt.Printf("  TxPipeline: a=%s b=%s (atomic)\n", a, b)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	rdb, mr := newClient()
	defer mr.Close()
	defer rdb.Close()

	ctx := context.Background()
	fmt.Println("=== Redis Patterns (miniredis) ===")
	fmt.Println()

	demoSessionStore(ctx, rdb)
	demoRateLimiter(ctx, rdb)
	demoDistLock(ctx, rdb)
	demoPubSub(ctx, rdb)
	demoPipeline(ctx, rdb)
}
