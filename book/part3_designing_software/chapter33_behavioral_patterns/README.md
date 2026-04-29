# Chapter 33 — Behavioral Patterns

> **Part III · Designing Software** | Estimated reading time: 25 min | Runnable examples: 2 | Exercises: 1

---

## Why this chapter matters

Behavioral patterns define how objects communicate and distribute responsibility. Go's interface system, closures, and first-class functions make most behavioral patterns more concise than their Java counterparts — often collapsing to a handful of lines.

---

## 33.1 — Strategy

Defines a family of algorithms, encapsulates each in a type, and makes them interchangeable. The context delegates to the current strategy; swapping the strategy changes behaviour without modifying the context.

```go
type SortStrategy interface {
    Sort(data []int) []int
}

type Sorter struct{ strategy SortStrategy }

func (s *Sorter) Sort(data []int) []int {
    return s.strategy.Sort(data)
}
```

Concrete strategies: `BubbleSort`, `StdSort`, `ReverseSort`. None knows about the others; `Sorter` knows only the interface.

**In Go, a function type is the minimal Strategy**: `type PriceCalc func(base float64, qty int) float64`. Use a full interface when the strategy needs state or multiple methods.

---

## 33.2 — Observer

Defines a one-to-many dependency: when one object changes state, all registered observers are notified automatically.

```go
type Observer interface { OnEvent(e Event) }

type EventBus struct{ observers map[string][]Observer }

func (b *EventBus) Subscribe(topic string, o Observer)
func (b *EventBus) Publish(e Event)
```

Wildcard subscriptions (`"*"`) allow cross-cutting observers like `MetricsCollector`. Observers are independent; adding one does not affect others.

---

## 33.3 — Command

Encapsulates a request as an object with `Execute()` and `Undo()` methods. A `CommandHistory` stack enables undo/redo:

```go
type Command interface {
    Execute() error
    Undo() error
    Description() string
}
```

Key insight: each command saves the data needed to reverse itself (`DeleteCommand` saves `deleted string`). The history stack is just `[]Command`.

---

## 33.4 — Iterator

Provides sequential access to a collection without exposing its representation.

**Closure-based (idiomatic Go):**

```go
func InorderIterator(root *TreeNode) func() (int, bool) {
    stack := []*TreeNode{}
    // ...
    return func() (int, bool) { /* advance state */ }
}
```

The closure captures the traversal state. Callers call `next()` in a `for v, ok := next(); ok; v, ok = next() {}` loop. This pattern integrates naturally with the upcoming `range func` iterator proposal in Go 1.23+.

---

## 33.5 — State

Allows an object to alter its behaviour when its internal state changes. The object appears to change its class:

```go
type OrderStatus interface {
    Pay(o *Order) error
    Ship(o *Order) error
    Cancel(o *Order) error
}
```

Each state struct implements `OrderStatus` and handles only the transitions that are valid in that state. Invalid transitions return an error. The `Order` delegates all calls to `o.status`; swapping `o.status` changes all behaviour.

---

## 33.6 — Pipeline (Strategy + Iterator combined)

A pipeline composes an iterator source with a chain of processor strategies:

```go
pipeline := NewPipeline(SliceIter(data),
    TrimProcessor{},
    FilterEmptyProcessor{},
    UpperProcessor{},
)
results := pipeline.Collect()
```

Each processor is a Strategy. The source is an Iterator. The pipeline is lazy: it only advances the iterator when pulled.

---

## Running the examples

```bash
cd book/part3_designing_software/chapter33_behavioral_patterns

go run ./examples/01_strategy_observer         # Strategy (sort, pricing), Observer (event bus, stock)
go run ./examples/02_command_iterator_state    # Command (doc undo/redo), Iterator (BST, range), State (order)

go run ./exercises/01_pipeline                 # lazy data pipeline combining Strategy + Iterator
```

---

## Key takeaways

1. **Strategy** — interchangeable algorithms behind an interface; swap without modifying the context.
2. **Observer** — event bus with `Subscribe`/`Publish`; observers are independent and composable.
3. **Command** — requests as objects; `Execute()` + `Undo()` + a history stack = full undo/redo.
4. **Iterator** — closure-based `func() (T, bool)` is Go's idiomatic lazy traversal.
5. **State** — each state handles only its valid transitions; invalid ones return errors.

---

## Cross-references

- **Chapter 14** — Closures: iterator functions are closures that capture traversal state
- **Chapter 32** — Structural Patterns: Strategy composes naturally with Decorator
- **Chapter 36** — Error Handling: State pattern uses sentinel errors for invalid transitions
