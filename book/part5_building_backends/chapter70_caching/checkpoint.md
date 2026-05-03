# Chapter 70 Checkpoint — Caching

## Self-assessment questions

1. Why is the LRU cache backed by both a hash map and a doubly-linked list, and what is the time complexity of Get and Put?
2. What is the difference between cache-aside (lazy loading) and write-through caching?
3. Why does TTL cache use lazy expiry (check on access) instead of a background goroutine?
4. What HTTP status code does a server return when a client's ETag matches the cached response, and what is omitted from the response?
5. What does `Cache-Control: no-store` mean, and when should you use it?
6. How does the `Vary: Accept` header affect cache key generation?
7. Why must you invalidate a cache entry on Update and Delete, and what can go wrong if you skip it?

## Checklist

- [ ] Can implement a generic `LRUCache[K, V]` using `container/list` and a map
- [ ] Can wrap LRU with TTL expiry using a `ttlEntry` struct containing an `expiresAt` field
- [ ] Can explain and implement cache-aside, write-through, and write-behind patterns
- [ ] Can write HTTP middleware that caches GET responses by URL path
- [ ] Can generate ETags by hashing the response body with `crypto/sha256`
- [ ] Can handle `If-None-Match` / 304 Not Modified to save bandwidth
- [ ] Can parse `Cache-Control: max-age=N` and honour `no-cache` / `no-store`
- [ ] Can invalidate cache entries on write, update, and delete operations
- [ ] Can expose hit/miss/eviction metrics from a cache layer

## Answers

1. The map provides O(1) lookup by key; the doubly-linked list maintains access order and enables O(1) removal of any node (eviction or move-to-front). Together, both Get and Put are O(1). Without the list, finding the LRU element would be O(n).

2. Cache-aside puts the app in control: the app checks the cache first and, on a miss, loads from the store and populates the cache. Write-through writes to both the store and the cache synchronously on every write, ensuring the cache always holds the latest value. Cache-aside is lazier and simpler; write-through costs a double-write but keeps reads fast after the first write.

3. Lazy expiry avoids a background goroutine that would consume a goroutine forever and complicate shutdown. Expired entries are simply skipped when accessed and then removed. The downside is that the cache can hold expired entries until they are accessed. For a periodic hard sweep, an optional background goroutine can be added.

4. The server returns `304 Not Modified`. The response body is omitted entirely — only headers are sent. The client reuses its locally cached body, saving bandwidth and latency.

5. `Cache-Control: no-store` instructs every cache in the chain (browser, CDN, reverse proxy, application) never to store the response. Use it for sensitive data such as authentication tokens, personal profile pages, or financial data where the risk of a stale or leaked copy outweighs the caching benefit.

6. When `Vary: Accept` is present, the cache must store a separate entry for each distinct `Accept` header value. The cache key becomes `path + "|" + Accept`. A request for `/products` with `Accept: application/json` and another with `Accept: application/xml` hit different cache slots even though the URL is identical.

7. If you skip invalidation, the cache returns the old value after a write. A client that reads a product immediately after a price update sees the stale price. On delete, the cache returns a product that no longer exists. Always invalidate (remove the cache entry) on any mutation. Subsequent reads re-fetch the fresh value from the store.
