# Chapter 24 — Revision Checkpoint

## Questions

1. What syntax declares a generic function?
2. What does `comparable` mean as a constraint?
3. What does `~int` mean in a constraint?
4. When does the compiler infer type parameters?
5. Name two situations where you should NOT use generics.

## Answers

1. Place the type parameters in `[...]` before the function's parameter list:
   `func F[T Constraint](args ...) returnType { ... }`.
   Generic types use the same syntax: `type Stack[T any] struct { ... }`.

2. `comparable` constrains to types that support `==` and `!=`: numbers, strings,
   booleans, arrays of comparable types, comparable structs, pointers. Slices,
   maps, and functions are not comparable.

3. `~int` (tilde) includes `int` itself and any named type whose underlying type
   is `int` (e.g., `type MyInt int`). Without `~`, only `int` exactly would match.

4. When the type can be inferred from the function's arguments. For
   `Min(3, 5)`, the compiler sees two `int` arguments and infers `T=int`.
   Explicit type parameters are needed when inference is impossible (e.g.,
   the type only appears in the return type).

5. Any two of: the function only operates on one specific type (no need for
   generics); behaviour genuinely differs per type (use interface + type switch);
   runtime type inspection is needed (use reflect); the generic version is harder
   to read than three specific copies; the function already works with `any` /
   `interface{}` and type safety is not critical.
