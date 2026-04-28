# Chapter 25 — Reflection: Programming the Type System

> **Part II · Core Language** | Estimated reading time: 18 min | Runnable examples: 1 | Exercises: 1

---

## Why this chapter matters

Reflection is how Go programs inspect and manipulate types and values at runtime. It is how `encoding/json`, `database/sql` drivers, dependency injection frameworks, and ORMs work. You will rarely write reflection code directly, but understanding it explains why those packages work, where their costs come from, and how to avoid common mistakes.

---

## 25.1 — The two entry points

```go
t := reflect.TypeOf(v)   // *reflect.rtype — the type descriptor
v := reflect.ValueOf(v)  // reflect.Value  — the value + type
```

Both accept `any`. Both panic if passed a nil interface.

---

## 25.2 — Kind vs Type

`Type` is the static type (`main.Config`, `[]int`, `*os.File`).
`Kind` is the underlying category (`struct`, `slice`, `ptr`, `int`, `string`, ...).

Use `Kind` in switches to handle classes of types; use `Type` for exact matching.

```go
t := reflect.TypeOf(v)
switch t.Kind() {
case reflect.Struct:   ...
case reflect.Slice:    ...
case reflect.Ptr:      ...
}
```

---

## 25.3 — Struct introspection

```go
rt := reflect.TypeOf(Config{})
for i := range rt.NumField() {
    f := rt.Field(i)          // reflect.StructField
    fmt.Println(f.Name)       // "Host"
    fmt.Println(f.Tag.Get("json")) // "host"
}
```

`reflect.StructField.IsExported()` tells you whether the field is accessible.

---

## 25.4 — Setting values

To modify a value via reflection, you must pass a pointer and dereference it:

```go
rv := reflect.ValueOf(&cfg).Elem()   // addressable
rv.FieldByName("Host").SetString("prod.example.com")
```

Setting an unexported field panics. Setting a non-addressable value panics.

---

## 25.5 — reflect.DeepEqual

`reflect.DeepEqual` recursively compares two values of any type:
- Slices: element-wise comparison (nil ≠ empty)
- Maps: key-value pair comparison
- Structs: field-wise comparison

It is primarily used in tests (`assert.Equal` uses it internally). Its recursive nature makes it expensive — avoid in hot paths.

---

## 25.6 — Cost summary

| Operation | Cost |
|---|---|
| `TypeOf`, `ValueOf` | Cheap — single pointer lookup |
| `Value.Field(i)` | Cheap — index into struct |
| Method call via `reflect.Value.Call` | ~10× slower than direct call |
| `reflect.DeepEqual` | Proportional to size of the tree |
| Creating a value via `reflect.New` | Same as `new(T)` + overhead |

Use reflection for: serialisation, ORMs, DI containers, test utilities.
Avoid for: hot paths, simple dispatch (use type switch), known-type operations.

---

## Running the examples

```bash
cd book/part2_core_language/chapter25_reflection

go run ./examples/01_reflect_basics   # TypeOf, ValueOf, struct tags, fillDefaults, DeepEqual

go run ./exercises/01_struct_printer  # recursive struct pretty-printer
```

---

## Key takeaways

1. `reflect.TypeOf` returns the type; `reflect.ValueOf` returns the value.
2. `Kind` categorises types; `Type` identifies them exactly.
3. To set a value via reflection, pass a pointer and call `.Elem()` on the Value.
4. `reflect.DeepEqual` is for tests, not production hot paths.
5. Reflection is the mechanism behind `encoding/json`, `database/sql`, and ORMs.

---

## Cross-references

- **Chapter 10** — Type assertions: the compile-time alternative to runtime type inspection
- **Chapter 22** — Interfaces: the `any` interface is the entry point to reflection
- **Chapter 24** — Generics: often a better alternative to reflection for type-safe code
