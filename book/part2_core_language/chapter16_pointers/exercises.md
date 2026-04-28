# Chapter 16 — Exercises

## 16.1 — Singly linked list

Run [`exercises/01_linked_list`](exercises/01_linked_list/main.go).

Study how `*Node` is used for `Next` and how `nil` terminates traversal.

Try:
- Add `Reverse() *List` that returns a new list with elements in reverse order.
- Add `InsertAfter(target, value int) bool` that inserts `value` after the first
  node whose `Value == target`. Return false if target is not found.
- What happens if you change `List` methods to value receivers? Try it and explain
  why `Push` and `Pop` break.

## 16.2 ★ — Escape analysis reading

Run the escape analysis build command:

```bash
go build -gcflags="-m" ./examples/02_escape_analysis
```

For each function, identify:
1. Which variables escape and why.
2. Whether the escape is "necessary" (the value must outlive the frame) or
   "incidental" (interface boxing, passed to `fmt.Println`).

Then modify `interfaceEscape` to print `x` directly without assigning to
`interface{}`. Does the escape go away?

## 16.3 ★ — Optional config with JSON

Define a `ServerConfig` with `*int` and `*bool` fields. Use `encoding/json`
to unmarshal two JSON blobs — one with all fields, one with some absent.
Verify that absent fields remain `nil` while present zero-value fields (`"port": 0`)
are non-nil.
