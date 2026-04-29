# Chapter 33 — Exercises

## 33.1 — Data processing pipeline

Run [`exercises/01_pipeline`](exercises/01_pipeline/main.go).

A lazy pipeline combining Iterator source with Strategy processor chain.

Try:
- Add a `MapProcessor` that applies a `func(string) string` to each item. Use it to replace all spaces with underscores.
- Add a `TakeProcessor` that stops the pipeline after N items (return `false` on the N+1th call).
- Add a `DeduplicateProcessor` that filters already-seen strings using a `map[string]bool`.

## 33.2 ★ — Undo stack with macros

Extend the `CommandHistory` from example 02:
- Add `BeginMacro()` / `EndMacro()` — group multiple commands into a single undoable unit.
- When `Undo()` is called, the entire macro undoes as one step.
- Add a `HistorySnapshot()` method that returns a `[]string` of all command descriptions in order.

## 33.3 ★★ — Traffic light (State machine)

Build a `TrafficLight` with states: `Red → Green → Yellow → Red`.
Each state has `Next() *TrafficLight` and `Duration() time.Duration`.
Add a `Controller` that runs through 3 full cycles, printing state and duration at each transition.
Ensure that invalid transitions (e.g., calling `Next()` while in an intermediate state) are impossible by design.
