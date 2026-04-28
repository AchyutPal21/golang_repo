# Chapter 17 — Revision Checkpoint

## Questions

1. What is the type of `[3]int{1,2,3}`? Is it the same type as `[4]int{...}`?
2. What happens when you assign one array to another in Go?
3. Are arrays comparable? Can they be used as map keys?
4. Name two practical uses for fixed-size array types.
5. What is the relationship between an array and a slice?

## Answers

1. `[3]int`. No — `[3]int` and `[4]int` are distinct, incompatible types. You
   cannot assign one to the other or pass one where the other is expected.

2. The entire array is copied. This is value semantics: after `b := a`, `b` is
   an independent copy. Modifying `b` does not affect `a`.

3. Yes, if the element type is comparable. `[3]int{1,2,3} == [3]int{1,2,3}` is
   true. Arrays of comparable element types can be used as map keys.

4. Any two of: cryptographic digest types (`[32]byte` for SHA-256), IP addresses
   (`[4]byte` for IPv4), fixed-size protocol frames, look-up table indices, pixel
   types (`[3]uint8` for RGB).

5. A slice is a three-field header (pointer, length, capacity) pointing into an
   underlying array. Slicing an array (`arr[1:4]`) creates a slice that shares
   memory with the array. Every `make([]T, n)` allocates an underlying array and
   returns a slice header over it.
