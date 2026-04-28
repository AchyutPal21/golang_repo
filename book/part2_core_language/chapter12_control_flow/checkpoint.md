# Chapter 12 — Revision Checkpoint

## Questions

1. How do you write a `while` loop in Go?
2. What does `for range 5` do?
3. Does Go's switch fall through by default?
4. What changed about for-loop variables in Go 1.22?
5. When would you use a label?
6. Is `goto` legal? When is it idiomatic?

## Answers

1. `for cond { ... }`. Go has one loop keyword.

2. Iterates `i` from 0 to 4 (Go 1.22+). With `for range 5
   { ... }`, the body runs 5 times without binding `i`.

3. No. Cases don't fall through unless you use `fallthrough`
   explicitly. This eliminates the C/Java "forgot to break"
   bug class.

4. Each iteration now creates a fresh loop variable. Pre-1.22,
   closures captured the same variable across iterations,
   which surprised most developers spawning goroutines in
   loops.

5. To target a specific outer construct from `break`,
   `continue`, or `goto`. Most common: escaping nested loops
   without flag variables.

6. Yes, `goto` is part of the language. Rarely idiomatic;
   reserved for retry loops or state machines where `for`
   plus `break`/`continue` would obscure intent. Don't use
   `goto` for ordinary control flow.
