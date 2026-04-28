# Chapter 20 — Revision Checkpoint

## Questions

1. What is the difference between positional and keyed composite literals?
2. What are field tags and how are they used?
3. What does embedding a type `T` in a struct give you?
4. Is embedding the same as inheritance?
5. When would you embed `*T` instead of `T`?
6. How do you access a shadowed (embedded) method?

## Answers

1. Positional: `Point{1, 2}` — must supply all fields in declaration order.
   Keyed: `Point{X: 1, Y: 2}` — specify fields by name, omit zero-value fields.
   Keyed literals are preferred; positional break silently if fields are added
   or reordered.

2. Field tags are raw string metadata attached to struct fields.
   They are read at runtime via `reflect.StructTag`. Used by `encoding/json`
   to control JSON field names and omitempty behaviour, by database drivers
   to map columns, by validation libraries, and more.

3. Field promotion (outer type's fields can be accessed as if they were its own)
   and method promotion (inner type's methods are callable on the outer type).

4. No. Embedding is composition. The outer type does not become the inner type —
   there is no subtype relationship. You cannot pass a `Dog` where an `Animal`
   is expected unless `Dog` explicitly satisfies the same interface.

5. When the inner struct is large (to avoid copying on assignment), when multiple
   outer instances should share the same inner instance, or when the inner value
   may be nil (optional component).

6. Via the explicit embedded field name: `d.Animal.Describe()`. The embedded
   field name is the unqualified type name (or the package-qualified name for
   imported types).
