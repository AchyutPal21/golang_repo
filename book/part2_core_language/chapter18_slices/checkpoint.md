# Chapter 18 — Revision Checkpoint

## Questions

1. What are the three fields of a slice header?
2. What is the difference between a nil slice and an empty slice?
3. Why must you always assign the return value of `append`?
4. What is the aliasing trap? When does it occur?
5. What does `s[1:4:4]` do differently from `s[1:4]`?
6. How do you clone a slice?
7. A function receives `s []int` and calls `append(s, 99)` without returning.
   What does the caller see?

## Answers

1. Pointer to the first element, length (number of visible elements), capacity
   (elements from the pointer to the end of the backing array).

2. Both have len=0. A nil slice has a nil pointer and equals nil. An empty slice
   has a non-nil pointer and does not equal nil. Both are safe to range over and
   append to. The distinction matters for JSON (`null` vs `[]`) and `reflect.DeepEqual`.

3. append may need to allocate a new backing array. When it does, the original
   backing array is abandoned and the new slice header (with a new pointer) is
   returned. If you discard the return value, you lose the new allocation.

4. Two slice headers pointing to the same backing array. Writing through one
   slice modifies the data visible through the other. Occurs when sub-slicing:
   `alias := original[1:4]` — alias and original share the same array.

5. `s[1:4]` creates a slice with cap extending to the end of the backing array.
   Appending to it can overwrite elements beyond index 4 in the original.
   `s[1:4:4]` limits cap to 3 (4-1), so any append will allocate a new array,
   protecting the original data.

6. `clone := append([]int(nil), s...)` or `clone := make([]int, len(s)); copy(clone, s)`.

7. The caller's slice is unchanged. The function received a copy of the slice
   header. If append grew beyond cap, it returned a new header pointing to a new
   array — but the caller still holds the original header. The appended value is
   silently lost.
