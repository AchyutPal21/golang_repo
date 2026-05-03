# Chapter 71 — Redis

## What you'll learn

How to use Redis from Go via `go-redis/v9` for caching, session management, rate limiting, distributed locking, and pub/sub messaging. All examples run against `miniredis` — a pure-Go in-memory Redis server — so no external Redis installation is required.

## Key concepts

| Concept | Redis commands | Use case |
|---|---|---|
| Strings | GET/SET/SETEX/INCR/MGET | Cache single values, counters |
| Lists | LPUSH/RPUSH/LRANGE/LPOP | Job queues, activity feeds |
| Sets | SADD/SMEMBERS/SISMEMBER | Tags, unique visitors |
| Hashes | HSET/HGET/HGETALL | Object fields, user profiles |
| Sorted sets | ZADD/ZRANGE/ZRANGEBYSCORE | Leaderboards, time-sorted events |
| TTL | EXPIRE/TTL/PERSIST | Cache expiry |
| Pipeline | `rdb.Pipeline()` | Batch N commands into 1 round trip |
| TxPipeline | `rdb.TxPipeline()` | Atomic pipeline (MULTI/EXEC) |
| Pub/Sub | Subscribe/Publish | Event fan-out |

## Files

| File | Topic |
|---|---|
| `examples/01_redis_basics/main.go` | Strings, lists, sets, hashes, sorted sets, TTL |
| `examples/02_redis_patterns/main.go` | Session store, rate limiter, distributed lock, pub/sub, pipeline |
| `exercises/01_cache_store/main.go` | Cache interface, Redis + in-memory impls, resilient fallback, prefix invalidation |

## Setup (miniredis — no real Redis needed)

```go
import (
    "github.com/alicebob/miniredis/v2"
    "github.com/redis/go-redis/v9"
)

mr, _ := miniredis.Run()
rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
defer mr.Close()
```

## Common patterns

### Rate limiter

```go
pipe := rdb.Pipeline()
incr := pipe.Incr(ctx, "rate:"+key)
pipe.Expire(ctx, "rate:"+key, window)
pipe.Exec(ctx)
allowed := incr.Val() <= int64(limit)
```

### Distributed lock (SET NX EX)

```go
// Acquire
ok, _ := rdb.SetNX(ctx, "lock:"+resource, workerID, ttl).Result()

// Release (Lua — only delete if we own it)
script := `if redis.call("get",KEYS[1])==ARGV[1] then return redis.call("del",KEYS[1]) else return 0 end`
redis.NewScript(script).Run(ctx, rdb, []string{"lock:"+resource}, workerID)
```

### Session store

```go
rdb.SetEx(ctx, "session:"+token, jsonBytes, 24*time.Hour)
rdb.Get(ctx, "session:"+token).Bytes()
rdb.Del(ctx, "session:"+token)
```

### Pub/Sub

```go
sub := rdb.Subscribe(ctx, "orders")
ch := sub.Channel()
go func() {
    for msg := range ch { process(msg.Payload) }
}()
rdb.Publish(ctx, "orders", payload)
```

## Production tips

- Use `rdb.Pipeline()` for read-heavy pages that query many keys — reduces RTT
- Always set TTL on session keys — `SetEx` not `Set`
- Distributed lock: use Redlock (multi-node) in production for true fault tolerance
- Don't use Redis as primary storage — it's volatile; pair with a persistent DB
- `rdb.Close()` returns connections to the pool; always defer it
- Set `ReadTimeout` and `WriteTimeout` in `redis.Options` to avoid hanging on slow Redis
