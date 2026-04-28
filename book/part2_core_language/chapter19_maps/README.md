# Chapter 19 — Maps: Hash Tables Built In

> **Part II · Core Language** | Estimated reading time: 20 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Maps are Go's built-in hash table. They power frequency counters, caches, indices, sets, grouping operations, and routing tables. The gotchas are few but serious: writing to a nil map panics; iteration order is randomised; maps are reference types like slices; and concurrent reads + writes cause a fatal race. Understanding these properties is essential before using maps in any production context.

---

## 19.1 — Declaration and initialisation

```go
// nil map — safe to read, panics on write
var m map[string]int

// make — always use for maps you will write to
m = make(map[string]int)

// composite literal
scores := map[string]int{
    "Alice": 95,
    "Bob":   87,
}
```

The key type must be **comparable** (`==` defined): all numeric types, strings, booleans, arrays of comparable types, structs with only comparable fields. Slices, maps, and functions are not comparable and cannot be map keys.

---

## 19.2 — The nil map trap

```go
var m map[string]int
m["key"] = 1 // panic: assignment to entry in nil map
```

Reading from a nil map is safe and returns the zero value. Writing panics. Always initialise with `make` or a composite literal before writing.

---

## 19.3 — CRUD operations

```go
m["key"] = value      // set (create or update)
v := m["key"]         // get (zero value if missing)
delete(m, "key")      // delete (no-op if missing)
len(m)                // count of entries
```

---

## 19.4 — The comma-ok idiom

Reading a missing key returns the zero value — indistinguishable from an explicit zero. Use comma-ok to tell them apart:

```go
if v, ok := m["key"]; ok {
    // key exists, v is the value
} else {
    // key absent
}
```

This is the only way to distinguish "not present" from "present with zero value".

---

## 19.5 — Iteration order is randomised

The Go runtime deliberately randomises map iteration order on every run. This prevents code from accidentally depending on a specific order. To iterate deterministically:

```go
keys := make([]string, 0, len(m))
for k := range m { keys = append(keys, k) }
sort.Strings(keys)
for _, k := range keys { fmt.Println(k, m[k]) }
```

---

## 19.6 — Maps are reference types

Assignment copies the map header (a pointer), not the data:

```go
a := map[string]int{"x": 1}
b := a       // b and a share the same hash table
b["y"] = 2
// a is now {"x":1, "y":2}
```

To copy a map, iterate:

```go
clone := make(map[string]int, len(a))
for k, v := range a { clone[k] = v }
```

---

## 19.7 — Common patterns

### Frequency counter

```go
freq := make(map[string]int)
for _, w := range strings.Fields(text) {
    freq[w]++   // zero value + 1 on first encounter
}
```

The zero value trick: missing keys return 0, so `freq[w]++` works without checking existence.

### Grouping (map of slices)

```go
groups := make(map[string][]Person)
for _, p := range people {
    groups[p.City] = append(groups[p.City], p)
}
```

### Set

```go
type Set[T comparable] map[T]struct{}
```

`struct{}` occupies zero bytes. Adding, checking, and deleting:

```go
s[v] = struct{}{}     // add
_, ok := s[v]         // check
delete(s, v)          // remove
```

### Inverted index

```go
index := make(map[string][]int) // word → doc IDs
for id, text := range docs {
    for _, w := range strings.Fields(text) {
        index[w] = append(index[w], id)
    }
}
```

---

## 19.8 — Maps are not safe for concurrent use

The Go runtime detects concurrent map access and fatally crashes the program:

```
fatal error: concurrent map read and map write
```

For concurrent access, use one of:
- `sync.Mutex` or `sync.RWMutex` wrapping the map
- `sync.Map` (optimised for high-read, low-write patterns)

```go
var mu sync.RWMutex
var cache = make(map[string]int)

func get(k string) (int, bool) {
    mu.RLock()
    defer mu.RUnlock()
    v, ok := cache[k]
    return v, ok
}

func set(k string, v int) {
    mu.Lock()
    defer mu.Unlock()
    cache[k] = v
}
```

---

## 19.9 — sync.Map

`sync.Map` is a concurrent map optimised for:
- Keys that are written once and read many times
- Keys where different goroutines operate on disjoint sets

Its API differs from a regular map:

```go
var m sync.Map
m.Store("key", value)
v, ok := m.Load("key")
m.Range(func(k, v any) bool {
    // process k, v
    return true // return false to stop
})
```

For general cache workloads, a `sync.RWMutex`-protected `map` is often faster because sync.Map has higher constant overhead.

---

## Running the examples

```bash
cd book/part2_core_language/chapter19_maps

go run ./examples/01_map_basics    # nil trap, CRUD, comma-ok, iteration
go run ./examples/02_map_patterns  # freq, grouping, set, inverted index, sync.Map

go run ./exercises/01_word_count   # TopN word frequency with ranking
```

---

## Exercises

### [exercises.md](exercises.md)

---

## Revision checkpoint

### [checkpoint.md](checkpoint.md)

---

## Key takeaways

1. Always initialise maps with `make` or a composite literal before writing — nil map writes panic.
2. Use **comma-ok** to distinguish "absent" from "present with zero value".
3. **Iteration order is randomised** — sort keys explicitly for deterministic output.
4. Maps are **reference types** — assignment copies a pointer, not the data.
5. Maps are **not concurrent-safe** — protect with `sync.RWMutex` or use `sync.Map`.
6. The zero value trick (`freq[w]++`) is idiomatic for counters and accumulators.

---

## Cross-references

- **Chapter 18** — Slices: map of slices is a common combined pattern
- **Chapter 20** — Structs: structs as map values; struct comparability for map keys
- **Chapter 36** — sync: `sync.Mutex`, `sync.RWMutex`, `sync.Map`
