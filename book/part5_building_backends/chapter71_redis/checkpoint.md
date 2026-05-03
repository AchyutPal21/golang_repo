# Chapter 71 Checkpoint — Redis

## Self-assessment questions

1. Name five Redis data structures and a concrete use case for each.
2. What is the difference between `SET key val` and `SETEX key 300 val`?
3. How does `rdb.Pipeline()` improve performance, and when should you use it?
4. How does the distributed lock pattern work using `SET NX EX`? Why is a Lua script needed to release it?
5. What is the difference between pub/sub and a message queue in Redis?
6. What are TTL and EXPIRE, and how do you check remaining TTL?

## Checklist

- [ ] Can connect to Redis using `redis.NewClient` and ping successfully
- [ ] Can use GET/SET/SETEX/INCR/MGET for string operations
- [ ] Can use LPUSH/LRANGE/LPOP for list/queue operations
- [ ] Can use SADD/SMEMBERS/SISMEMBER/SINTER for set operations
- [ ] Can use HSET/HGET/HGETALL for hash (object) operations
- [ ] Can use ZADD/ZRANGE/ZRANGEBYSCORE for sorted set/leaderboard operations
- [ ] Can implement a session store with JSON marshaling and TTL
- [ ] Can implement a sliding-window rate limiter using INCR + EXPIRE
- [ ] Can acquire and release a distributed lock with SET NX EX + Lua
- [ ] Can publish and subscribe to a Redis channel
- [ ] Can batch commands with Pipeline and TxPipeline

## Answers

1. **Strings**: cache a user profile (SETEX key json 300). **Lists**: job queue (RPUSH queue job, LPOP queue worker). **Sets**: tag membership (SADD user:1:tags go backend; SISMEMBER). **Hashes**: user fields (HSET user:1 name Alice email ...). **Sorted sets**: leaderboard (ZADD scores 1200 alice; ZRANGE scores 0 -1 WITHSCORES).

2. `SET` creates a key with no expiry — it lives until deleted. `SETEX` creates a key that expires after N seconds. In production always use `SETEX` (or `SET key val EX n`) for cache entries and sessions — a `SET` without expiry can fill memory indefinitely.

3. `Pipeline()` buffers commands client-side and sends them in one TCP round trip, then reads all responses together. Use it when executing N commands where results don't depend on each other (e.g. setting 100 cache keys at startup, or incrementing multiple counters). Reduces latency from N×RTT to 1×RTT.

4. `SET lock_key worker_id NX EX 10` atomically sets the key only if it doesn't exist. Release must use a Lua script (`if get(key)==worker_id then del(key)`), not a plain DEL — otherwise, if your lock expired while you were working, you'd delete another worker's lock. The Lua script is atomic at the Redis level.

5. Pub/sub delivers messages only to currently connected subscribers — messages sent while a subscriber is disconnected are lost. A message queue (LIST with BRPOP/BLPOP or Redis Streams) persists messages until a consumer acknowledges them. Use pub/sub for real-time events; use lists/streams for reliable task delivery.

6. EXPIRE sets a key to expire after N seconds. TTL returns the remaining seconds (-1 = no expiry, -2 = key doesn't exist). In Go: `rdb.Expire(ctx, key, 5*time.Minute)`, `rdb.TTL(ctx, key).Result()`. PERSIST removes a TTL, making the key permanent.
