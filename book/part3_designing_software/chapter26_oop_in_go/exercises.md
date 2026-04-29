# Chapter 26 — Exercises

## 26.1 — Shape hierarchy refactor

Run [`exercises/01_shape_hierarchy`](exercises/01_shape_hierarchy/main.go).

`Shape` is an interface; `Circle`, `Rectangle`, `RightTriangle` are independent types.
`Renderer` and the functional helpers work on `Shape` — no type switches needed.

Try:
- Add `RegularPolygon{N int; SideLen float64}` with area via `(N * SideLen² * cot(π/N)) / 4`.
- Add a `sortByArea(shapes []Shape)` using `sort.Slice`.
- Why does `RightTriangle` have a `Hypotenuse()` method that is NOT on the `Shape` interface? When would you call it directly?

## 26.2 ★ — ORM-style struct mapper

Write `MapToStruct(data map[string]any, target any) error` that populates
a struct from a map using field names (lowercased) as keys.
Use reflection (Ch 25) for the mapping. Support `string`, `int`, `float64`, `bool`.
This is a simplified version of what `encoding/json` does internally.

## 26.3 ★ — Plugin registry (composition)

Design a `PluginRegistry` that holds a map of `string → Plugin` where
`Plugin` has `Name() string` and `Run(args []string) error`.
Implement three plugins: `echoPlugin`, `reversePlugin`, `countPlugin`.
Register them, list them, and dispatch `Run` by name — no switch statement.
