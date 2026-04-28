# Chapter 19 — Revision Checkpoint

## Questions

1. What happens when you write to a nil map?
2. How do you distinguish "key not present" from "key present with zero value"?
3. Is map iteration order guaranteed?
4. What types can be used as map keys?
5. Are maps safe to use from multiple goroutines concurrently?
6. What is the idiomatic Go pattern for implementing a set?

## Answers

1. A runtime panic: `assignment to entry in nil map`. Always initialise a map
   with `make` or a composite literal before writing to it. Reading from a nil
   map is safe and returns the zero value for the value type.

2. Use the comma-ok idiom: `v, ok := m[key]`. If ok is false, the key is absent.
   If ok is true, v is the stored value (which may be the zero value).

3. No. Go deliberately randomises map iteration order on every run to prevent
   code from relying on a specific ordering. Sort the keys explicitly if you need
   a deterministic order.

4. Any comparable type: all numeric types, strings, booleans, pointers, channels,
   arrays of comparable types, and structs whose fields are all comparable.
   Slices, maps, and function values are not comparable and cannot be map keys.

5. No. Concurrent reads are safe, but any concurrent write (including delete) with
   a read or another write causes a fatal race detected at runtime. Use a
   `sync.RWMutex`-protected map or `sync.Map` for concurrent access.

6. `type Set[T comparable] map[T]struct{}`. The `struct{}` value type uses zero
   bytes. Add with `s[v] = struct{}{}`, check with `_, ok := s[v]`, delete with
   `delete(s, v)`.
