# Chapter 25 — Exercises

## 25.1 — Struct pretty-printer

Run [`exercises/01_struct_printer`](exercises/01_struct_printer/main.go).

`PrintStruct` uses reflection to walk a struct recursively, printing each
exported field with name, type, and value.

Try:
- Handle slices and maps specially — print their length and element type.
- Add a `depth` limit parameter to prevent infinite recursion on self-referential structs.
- Add colour output using ANSI codes: field names in cyan, types in yellow, values in white.

## 25.2 ★ — Environment variable config loader

Implement `LoadFromEnv(cfg any) error` that reads struct fields tagged with
`env:"VAR_NAME"` and populates them from `os.Getenv`. Support `string`, `int`,
`bool`, and `[]string` (comma-separated). Return an error if a required field
(tagged `required:"true"`) is missing.

## 25.3 ★ — Simple JSON marshaller

Implement `Marshal(v any) ([]byte, error)` that serialises a struct to JSON
using only the `reflect` package (no `encoding/json`). Handle `string`, `int`,
`float64`, `bool`, nested structs, and `[]T` slices. Respect `json:"name"` tags
and skip fields tagged `json:"-"`.
