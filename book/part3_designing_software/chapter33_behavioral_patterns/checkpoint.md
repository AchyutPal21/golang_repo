# Chapter 33 — Revision Checkpoint

## Questions

1. How does Strategy differ from simple if/switch dispatch?
2. What information must a Command save to support `Undo()`?
3. Why is a closure the idiomatic Go implementation of Iterator?
4. What is the key rule that makes the State pattern safe?
5. How does the Observer pattern's wildcard subscription work in the event bus example?

## Answers

1. `if/switch` embeds the decision logic in the *calling code*; adding a new case
   requires editing the dispatch site. Strategy encapsulates each algorithm in its
   own type. Adding a new algorithm is a new file — no existing code changes. The
   context (`Sorter`, `PriceCalculator`) is closed for modification; new strategies
   extend it from the outside. Strategy also makes algorithms testable in isolation
   and composable as values (you can store them, pass them, swap them at runtime).

2. A Command must save whatever is needed to **reverse its effect**. An `InsertCommand`
   at position `pos` inserts `text`; to undo, delete `len(text)` bytes at `pos` —
   no extra state needed. A `DeleteCommand` must save the *deleted text* before
   erasing it, because that text is gone after `Execute()` and cannot be recovered
   without the saved copy. The general rule: save pre-conditions, not post-conditions.

3. A Go closure captures its environment across multiple calls. An inorder BST iterator
   needs to remember the stack and current node between calls — exactly what a closure's
   captured variables provide. The returned `func() (T, bool)` has the same signature
   for any traversal (slice, tree, channel, file lines) so callers are uniform. No
   separate `Iterator` struct with `HasNext()`/`Next()` is needed; the function itself
   *is* the iterator. Closures are lighter and more composable for this use case.

4. The key rule: **each state implements only the transitions it can legally perform**.
   A `pendingState.Ship()` returns an error — it does not call `o.setState`. A
   `deliveredState.Cancel()` returns an error — it never reaches the code that sets
   the cancelled state. This means illegal transition sequences are impossible by
   construction: you cannot reach `delivered` without going through `paid → shipped`,
   because the intermediate states refuse the skipping transitions.

5. When `Publish` is called with a topic, the event bus iterates over two slices:
   `observers[e.Type]` (exact-match subscribers) and `observers["*"]` (wildcard
   subscribers). Both groups are notified. The `MetricsCollector` subscribes to `"*"`
   and increments its counter for every event regardless of topic. Topic-specific
   subscribers (`AuditLogger` on `"user.login"`, `EmailAlerter` on `"system.alert"`)
   only receive events matching their exact topic. This allows per-topic and cross-
   cutting observers to coexist without special-casing.
