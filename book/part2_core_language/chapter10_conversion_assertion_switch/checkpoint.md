# Chapter 10 — Revision Checkpoint

## Questions

1. What's the syntactic difference between conversion and
   assertion?
2. When does a type assertion panic, and how do you avoid it?
3. Why is `var err error = (*MyErr)(nil); err == nil` `false`?
4. Inside `switch v := x.(type) { case int, string: ... }`, what
   is the type of `v`?
5. What does `string(65)` produce, and why is it a problem?
6. When should you prefer `errors.As` over manual error
   assertion?

## Answers

1. **Conversion** is `T(x)` — converts a value's type at compile
   time. **Assertion** is `x.(T)` — extracts a concrete type
   from an interface at runtime.

2. Panics when the panicky form (`v := x.(T)`) is used and `x`'s
   dynamic type is not `T`. Avoid by using the comma-ok form:
   `v, ok := x.(T)`.

3. The interface has two words (type and data). A typed nil
   pointer assigned to an interface produces a non-nil interface
   because the type word is set. Always return literal `nil`
   from functions that return interfaces.

4. The original interface type. When multiple types share a
   case, the compiler can't pick one to retype `v` to.

5. `"A"` — Go interprets the int as a Unicode code point. To
   format a number as text, use `strconv.Itoa(65)` or
   `fmt.Sprint(65)`.

6. Always, in modern code (Go 1.13+). `errors.As` walks wrapped
   errors transparently; manual assertion only inspects the
   outer layer.
