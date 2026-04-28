# Chapter 25 — Revision Checkpoint

## Questions

1. What is the difference between `reflect.TypeOf` and `reflect.ValueOf`?
2. What is the difference between `Kind` and `Type`?
3. How do you set a struct field via reflection?
4. What does `reflect.DeepEqual` do? When should you use it?
5. Name three things in the standard library that use reflection internally.

## Answers

1. `reflect.TypeOf(v)` returns `reflect.Type` — information about the type itself
   (name, kind, fields, methods). `reflect.ValueOf(v)` returns `reflect.Value` —
   a handle to both the value and its type, allowing reading and (if addressable)
   writing.

2. `Kind` is the fundamental category: `reflect.Struct`, `reflect.Slice`,
   `reflect.Int`, `reflect.Ptr`, etc. There are ~26 kinds. `Type` is the exact
   type: `main.Config`, `[]int`, `*os.File`. Use `Kind` in switches over
   categories; use `Type` for exact comparisons.

3. Pass a pointer to `reflect.ValueOf`, call `.Elem()` to get the addressable
   struct value, then call `.FieldByName(name).SetString(...)` (or SetInt, SetBool,
   etc.). Setting unexported fields or non-addressable values panics.

4. `reflect.DeepEqual` recursively compares two values of any type for equality,
   handling slices, maps, pointers, and structs. Use it in tests where `==` is not
   defined (slices, maps) or where you need deep structural equality. Avoid in
   production hot paths — it is proportional to the size of the data structure.

5. Any three of: `encoding/json` (struct field tags, marshalling), `encoding/xml`,
   `database/sql` drivers (column scanning), `text/template` and `html/template`,
   `fmt` (for `%v`, `%+v`), dependency injection frameworks, ORMs.
