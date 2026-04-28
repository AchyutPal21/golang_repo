# Chapter 16 — Revision Checkpoint

## Questions

1. What does `&` do? What does `*` do?
2. What happens when you dereference a nil pointer?
3. What is escape analysis? Who performs it?
4. Name three things that cause a variable to escape to the heap.
5. When must you use a pointer receiver on a method?
6. Why is `*SomeInterface` almost always wrong?
7. How do you represent an "optional int" in Go?

## Answers

1. `&` takes the address of a variable, producing a pointer (`*T`). `*` dereferences
   a pointer, giving the value it points to. On a struct pointer, Go auto-dereferences
   field access: `pt.X` == `(*pt).X`.

2. A runtime panic: `panic: runtime error: invalid memory address or nil pointer
   dereference`. Always check for nil before dereferencing a pointer that might be nil.

3. Escape analysis is a compiler pass that decides whether a variable lives on the
   stack (fast, automatic cleanup) or the heap (GC-managed). The Go compiler performs
   it automatically — you do not annotate variables.

4. Any three of: the variable's address is returned from the function; a closure that
   outlives the function captures it; it is assigned to an interface; it is too large
   for the stack.

5. When the method needs to mutate the receiver's fields, or when the receiver is too
   large to copy efficiently. Consistency rule: if any method needs a pointer receiver,
   all methods on that type should use pointer receivers.

6. Interfaces are already reference types internally. `*SomeInterface` adds an
   unnecessary layer of indirection and breaks most interface-satisfaction checks.
   Pass the interface value directly.

7. Use `*int`. The zero value is `nil` (absent), distinct from `0` (present, zero value).
   Helper `func intPtr(n int) *int { return &n }` makes construction ergonomic.
