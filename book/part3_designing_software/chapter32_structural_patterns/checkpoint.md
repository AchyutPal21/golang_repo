# Chapter 32 — Revision Checkpoint

## Questions

1. What is the structural difference between an Adapter and a Decorator in Go?
2. How does a Composite differ from a slice of the same interface type?
3. When should you use a Facade rather than just calling subsystem components directly?
4. What does a Proxy add that a plain interface value does not already provide?
5. How does the `Chain` function in the middleware exercise ensure the first middleware in the list runs outermost?

## Answers

1. Both are structs that wrap a value and satisfy an interface. The difference is
   **intent and translation**. An Adapter wraps a type with the *wrong* interface
   and translates calls to match the *target* interface — it bridges two different
   APIs. A Decorator wraps a value with the *correct* interface and adds behaviour
   (logging, retrying, rate-limiting) before or after delegation — no translation,
   just augmentation.

2. A plain `[]FileSystemNode` slice is a flat collection; you iterate it yourself.
   A Composite makes the tree *recursive*: a `Directory` implements `FileSystemNode`
   and holds `[]FileSystemNode`. Calling `Size()` on the root walks the entire tree
   without the caller knowing the structure. New node types slot in without changing
   any consumer. The Composite pattern only makes sense when the hierarchy can be
   arbitrarily deep and consumers should not care about depth.

3. Use a Facade when:
   - Callers need to perform the same multi-step sequence repeatedly.
   - The subsystem has complex setup that is easy to get wrong.
   - You want to decouple callers from subsystem internals so they evolve
     independently.
   Calling components directly is fine for one-off or advanced use; the Facade
   just handles the common path and reduces the chance of mis-ordering steps.

4. A plain interface value provides *polymorphism* — you can hold any type that
   satisfies the interface. A Proxy adds *controlled access*: it intercepts calls
   and decides whether (and how) to forward them. The caching proxy stores results
   and skips the real call on a hit; the ACL proxy denies certain keys entirely.
   Neither of these is possible with a bare interface value — they require the
   intercepting wrapper struct.

5. `Chain` applies middlewares in *reverse order*:
   ```go
   for i := len(middlewares) - 1; i >= 0; i-- {
       h = middlewares[i](h)
   }
   ```
   The last middleware in the slice wraps the handler first (innermost). Then each
   earlier middleware wraps what came before, becoming progressively more outer.
   After the loop, the first middleware in the slice is the outermost wrapper — it
   executes first on every request. This mirrors the HTTP handler pattern where
   middleware listed first runs first.
