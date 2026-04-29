# Chapter 26 — Revision Checkpoint

## Questions

1. What Go construct replaces a Java `abstract class`?
2. How do you achieve polymorphism in Go without inheritance?
3. What is the canonical constructor pattern in Go?
4. At what boundary does Go enforce encapsulation?
5. What method should every new type implement first?
6. Why does Go omit method overloading?

## Answers

1. An interface. The interface defines the contract (method set); concrete types
   satisfy it implicitly by implementing the methods.

2. Through interface satisfaction. Any type that has the required methods can be
   used where the interface is expected — regardless of its type hierarchy
   (which doesn't exist in Go).

3. A `NewXxx(params) (*T, error)` function that validates invariants and returns
   a pointer to the initialised value, plus an error if construction fails.

4. At the package level. Lowercase (unexported) identifiers are invisible outside
   the package. All code within the same package has full access.

5. `String() string` (the `fmt.Stringer` interface). It controls how the type
   prints in `fmt.Println`, `fmt.Printf`, logs, and error messages.

6. Overloaded functions make call sites ambiguous — the reader must know all
   overloads to understand which one is called. Go favours explicit, readable
   call sites. Variadic parameters and interface parameters cover the common
   use cases without ambiguity.
