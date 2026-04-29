# Chapter 31 — Exercises

## 31.1 — SQL query builder

Run [`exercises/01_query_builder`](exercises/01_query_builder/main.go).

A fluent `QueryBuilder` constructs `SELECT` statements with method chaining and validates in `Build()`.

Try:
- Add `Join(table, condition string) *QueryBuilder` support: `JOIN table ON condition`.
- Add `GroupBy(column string)` and `Having(condition string)` — `Having` should error if `GroupBy` was not called first.
- Add `Offset(n int)` — error if called without `Limit`.

## 31.2 ★ — Typed factory registry

Build a `PluginRegistry` that maps string keys to factory functions:

```go
type Factory func(config map[string]string) (Plugin, error)

type Registry struct{ factories map[string]Factory }

func (r *Registry) Register(name string, f Factory)
func (r *Registry) Create(name string, config map[string]string) (Plugin, error)
```

Register three plugins (`compressor`, `encryptor`, `logger`). Create them by name from a config map. Verify that an unknown name returns a descriptive error.

## 31.3 ★★ — Connection pool with timeout

Extend the `ProcessorPool` from example 02 with a `GetWithTimeout(d time.Duration)` method. If no processor is available within `d`, return an error. Use `time.After` and a channel rather than a spin loop.
