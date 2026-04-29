# Chapter 27 — Exercises

## 27.1 — Storage abstraction

Run [`exercises/01_storage_abstraction`](exercises/01_storage_abstraction/main.go).

`TodoService` depends on `TodoStore` — a composed consumer-side interface.
`MemoryStore` satisfies it without knowing `TodoService` exists.

Try:
- Add a `FileStore` stub that prints "would write to file:" for each Save.
  Swap it in as the backend without changing `TodoService`.
- Add `TodoUpdater` (just `Update(t Todo) error`) to the interface composition.
- Write a test fake that records all calls and verifies them.

## 27.2 ★ — Read-through cache

Implement `ReadThroughCache` that satisfies `Getter` but falls back to a
`Loader` interface (`Load(key string) ([]byte, error)`) on cache miss,
then populates the cache. The `Loader` can be any source — file, HTTP, DB.

## 27.3 ★ — Interface discovery

Pick three packages from the standard library (`net/http`, `database/sql`,
`os`). For each, identify:
1. The key interfaces it exports.
2. Which are consumer-side and which are producer-side.
3. Which you could satisfy in a test with a 10-line struct.
