# Chapter 22 — Revision Checkpoint

## Questions

1. How does Go determine whether a type satisfies an interface?
2. What are the two words of an interface value?
3. What is a typed nil? Why is it dangerous?
4. What is the rule "accept interfaces, return structs"?
5. Why are small (1-2 method) interfaces preferred?
6. How do you use interfaces to make code testable without a mocking framework?

## Answers

1. Implicitly: if the type has all the methods in the interface's method set,
   it satisfies the interface. No declaration or keyword is needed. The compiler
   checks this at every point where the type is used as the interface.

2. (1) A pointer to the itab, which contains the concrete type and a method
   dispatch table. (2) A pointer to (or inline copy of) the concrete value.
   Both are nil for a nil interface; only the data pointer is nil for a typed nil.

3. A typed nil is an interface value whose itab is set (type is known) but whose
   data pointer is nil. It is dangerous because `err == nil` returns false even
   though no error occurred — the `if err != nil` guard does not fire. Always
   return `nil` directly through the interface type, never via a typed variable.

4. Function parameters should be interface types (accept the narrowest contract
   the function needs, so callers can pass any conforming type). Return values
   should be concrete types (give callers the full API without type assertions).

5. Small interfaces are easy to satisfy — a type with one method is much easier
   to implement than one with ten. They compose into larger interfaces. They
   are easy to fake in tests. `io.Reader` (one method) works for 50+ different
   source types in the standard library.

6. Define an interface for each external dependency (database, email, HTTP client).
   Accept the interface in your service constructor. In tests, pass a struct that
   implements the interface with predictable behaviour. In production, pass the
   real implementation. No reflection, no mocking framework required.
