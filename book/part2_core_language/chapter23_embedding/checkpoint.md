# Chapter 23 — Revision Checkpoint

## Questions

1. What does embedding `T` in a struct give you?
2. Is a struct that embeds `Animal` a subtype of `Animal`?
3. How is the diamond ambiguity resolved in Go?
4. If `Buffer` implements `io.ReadWriter`, does a struct that embeds `*Buffer` also implement `io.ReadWriter`?
5. What is the mixin pattern?

## Answers

1. Field promotion (outer type accesses T's fields directly) and method promotion
   (outer type's method set includes T's methods). Both are accessible via
   the outer type directly or via the explicit embedded field name.

2. No. There is no subtype relationship. `Dog{Animal: Animal{}}` cannot be assigned
   to an `Animal` variable — they are unrelated types. Dog satisfies the same
   interfaces as Animal (through promotion), but that is interface satisfaction,
   not subtyping.

3. If two embedded types both promote a method with the same name, accessing it
   directly on the outer type is ambiguous and causes a compile error. The outer
   type must define its own method with that name, optionally delegating to one or
   both embedded types via the explicit field names.

4. Yes. `*Buffer`'s promoted methods are in the outer struct's method set. The outer
   struct satisfies `io.ReadWriter` as long as `*Buffer` does.

5. The mixin pattern uses embedding to add shared behaviour to multiple unrelated
   types. For example, embedding a `Timestamps` struct adds `CreatedAt`, `UpdatedAt`,
   and `Touch()` to any model struct without code duplication.
