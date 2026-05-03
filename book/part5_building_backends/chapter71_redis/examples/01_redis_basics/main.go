// Chapter 71 — Redis basics: strings, lists, sets, hashes, sorted sets, TTL.
// Uses miniredis (pure-Go in-memory Redis) — no external server required.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Start an in-process Redis-compatible server.
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	ctx := context.Background()

	// ── Strings ───────────────────────────────────────────────────────────────
	fmt.Println("=== Strings ===")

	rdb.Set(ctx, "greeting", "hello, redis", 0)
	v, _ := rdb.Get(ctx, "greeting").Result()
	fmt.Printf("GET greeting = %q\n", v)

	// SETEX: set with TTL.
	rdb.Set(ctx, "session:abc", "user:42", 30*time.Second)
	ttl, _ := rdb.TTL(ctx, "session:abc").Result()
	fmt.Printf("SETEX session:abc TTL = %v\n", ttl.Round(time.Second))

	// INCR: atomic counter.
	rdb.Set(ctx, "counter", "10", 0)
	newVal, _ := rdb.Incr(ctx, "counter").Result()
	fmt.Printf("INCR counter = %d\n", newVal)
	rdb.IncrBy(ctx, "counter", 5)
	v2, _ := rdb.Get(ctx, "counter").Result()
	fmt.Printf("INCRBY counter 5 = %s\n", v2)

	// MGET: fetch multiple keys in one round trip.
	rdb.Set(ctx, "k1", "alpha", 0)
	rdb.Set(ctx, "k2", "beta", 0)
	rdb.Set(ctx, "k3", "gamma", 0)
	vals, _ := rdb.MGet(ctx, "k1", "k2", "k3", "k_missing").Result()
	fmt.Printf("MGET = %v\n", vals)

	// ── Lists ─────────────────────────────────────────────────────────────────
	fmt.Println("\n=== Lists ===")

	rdb.RPush(ctx, "queue", "task1", "task2", "task3")
	rdb.LPush(ctx, "queue", "task0") // prepend — becomes first element
	length, _ := rdb.LLen(ctx, "queue").Result()
	fmt.Printf("LLEN queue = %d\n", length)

	items, _ := rdb.LRange(ctx, "queue", 0, -1).Result()
	fmt.Printf("LRANGE queue 0 -1 = %v\n", items)

	popped, _ := rdb.LPop(ctx, "queue").Result()
	fmt.Printf("LPOP queue = %q  remaining=%d\n", popped,
		func() int64 { n, _ := rdb.LLen(ctx, "queue").Result(); return n }())

	// ── Sets ──────────────────────────────────────────────────────────────────
	fmt.Println("\n=== Sets ===")

	rdb.SAdd(ctx, "tags:go", "backend", "concurrency", "performance", "backend") // dup ignored
	members, _ := rdb.SMembers(ctx, "tags:go").Result()
	fmt.Printf("SMEMBERS tags:go = %v  (len=%d)\n", members, len(members))

	isMember, _ := rdb.SIsMember(ctx, "tags:go", "backend").Result()
	notMember, _ := rdb.SIsMember(ctx, "tags:go", "rust").Result()
	fmt.Printf("SISMEMBER backend=%v  rust=%v\n", isMember, notMember)

	// SINTER: intersection of two sets.
	rdb.SAdd(ctx, "tags:rust", "performance", "systems", "memory-safe")
	inter, _ := rdb.SInter(ctx, "tags:go", "tags:rust").Result()
	fmt.Printf("SINTER tags:go ∩ tags:rust = %v\n", inter)

	// ── Hashes ────────────────────────────────────────────────────────────────
	fmt.Println("\n=== Hashes ===")

	rdb.HSet(ctx, "user:1", "name", "Alice", "email", "alice@example.com", "role", "admin")
	name, _ := rdb.HGet(ctx, "user:1", "name").Result()
	fmt.Printf("HGET user:1 name = %q\n", name)

	all, _ := rdb.HGetAll(ctx, "user:1").Result()
	fmt.Printf("HGETALL user:1 = %v\n", all)

	fields, _ := rdb.HMGet(ctx, "user:1", "name", "role", "missing").Result()
	fmt.Printf("HMGET name,role,missing = %v\n", fields)

	rdb.HIncrBy(ctx, "user:1", "login_count", 1)
	loginCount, _ := rdb.HGet(ctx, "user:1", "login_count").Result()
	fmt.Printf("HINCRBY login_count = %s\n", loginCount)

	// ── Sorted Sets ───────────────────────────────────────────────────────────
	fmt.Println("\n=== Sorted Sets ===")

	rdb.ZAdd(ctx, "leaderboard",
		redis.Z{Score: 1200, Member: "alice"},
		redis.Z{Score: 950, Member: "bob"},
		redis.Z{Score: 1450, Member: "carol"},
		redis.Z{Score: 1100, Member: "dave"},
	)

	// ZRANGE: ascending by score.
	top, _ := rdb.ZRangeWithScores(ctx, "leaderboard", 0, -1).Result()
	fmt.Println("ZRANGE leaderboard (asc):")
	for _, z := range top {
		fmt.Printf("  %-8s  %.0f\n", z.Member, z.Score)
	}

	// ZRANGEBYSCORE: players with score >= 1000.
	highScorers, _ := rdb.ZRangeByScoreWithScores(ctx, "leaderboard",
		&redis.ZRangeBy{Min: "1000", Max: "+inf"}).Result()
	fmt.Printf("ZRANGEBYSCORE >= 1000: %d players\n", len(highScorers))

	rank, _ := rdb.ZRevRank(ctx, "leaderboard", "carol").Result()
	fmt.Printf("ZREVRANK carol = %d (0=first place)\n", rank)

	// ── Key TTL / Expiry ──────────────────────────────────────────────────────
	fmt.Println("\n=== TTL / EXPIRE / PERSIST ===")

	rdb.Set(ctx, "ephemeral", "gone soon", 5*time.Second)
	ttl2, _ := rdb.TTL(ctx, "ephemeral").Result()
	fmt.Printf("TTL ephemeral = %v\n", ttl2.Round(time.Second))

	// PERSIST: remove TTL, make key permanent.
	rdb.Persist(ctx, "ephemeral")
	ttl3, _ := rdb.TTL(ctx, "ephemeral").Result()
	fmt.Printf("After PERSIST TTL = %v  (−1 means no expiry)\n", ttl3)

	// Simulate TTL expiry by fast-forwarding miniredis clock.
	rdb.Set(ctx, "temp", "value", 2*time.Second)
	mr.FastForward(3 * time.Second)
	_, errGet := rdb.Get(ctx, "temp").Result()
	fmt.Printf("After TTL expiry: err=%v\n", errGet)
}
