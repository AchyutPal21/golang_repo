# Chapter 21 — Revision Checkpoint

## Questions

1. What is the difference between a value receiver and a pointer receiver?
2. What is the method set of `T`? What is the method set of `*T`?
3. You have `var s SomeInterface = MyType{...}` and get a compile error
   "does not implement". What is likely wrong and how do you fix it?
4. Can you call a pointer-receiver method on a nil pointer?
5. What is a method value? How does it differ from a method expression?

## Answers

1. A value receiver receives a copy of the value — mutations do not affect
   the caller's variable. A pointer receiver receives a pointer to the original —
   mutations do affect the caller. Go auto-dereferences in both directions for
   addressable values.

2. `T`'s method set contains only methods with receiver `T`. `*T`'s method set
   contains methods with receiver `T` and methods with receiver `*T`. This means
   `*T` always satisfies at least the same interfaces as `T`, and often more.

3. `MyType` has at least one method with a pointer receiver (`*MyType`). Only
   `*MyType`'s method set includes that method. Fix: `var s SomeInterface = &MyType{...}`.

4. Yes, as long as the method handles the nil case. The method must check
   `if recv == nil { ... }` before dereferencing. This is the standard pattern
   for recursive tree/list types.

5. A method value is a function bound to a specific receiver: `c.Area` captures
   `c` and has type `func() float64`. A method expression has the receiver as
   an explicit first parameter: `Circle.Area` has type `func(Circle) float64`.
   Method values are used as callbacks; method expressions are used when you
   want to pass a method like any other function.
