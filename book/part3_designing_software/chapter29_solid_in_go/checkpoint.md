# Chapter 29 — Revision Checkpoint

## Questions

1. What is the Single Responsibility Principle, and how does a coordinator struct fit in?
2. How does Go's implicit interface satisfaction enable the Open/Closed Principle?
3. Give a concrete example of an LSP violation in Go and how to fix it.
4. Why does consumer-side interface definition naturally enforce the Interface Segregation Principle?
5. How does the Dependency Inversion Principle relate to constructor injection?

## Answers

1. **SRP**: a type should have one reason to change — one concern, one team/layer
   that modifies it. A coordinator is the thin glue type that wires focused types
   together without owning any of their logic itself. Changing the email template
   only touches the mailer; changing the tax rate only touches the tax policy;
   the coordinator does not change.

2. In Go, any type that has the required methods satisfies an interface without
   declaring it. This means you can add new behaviour — a new `Exporter`,
   `Discount`, or `Notifier` — by writing a new type in a new file. No existing
   code needs to be edited. The `switch`-on-type anti-pattern forces you to edit
   existing code every time, which violates OCP.

3. `ReadOnlyBuffer` satisfies the `ReadWriter` interface (compiles fine) but panics
   on `Write`. Any caller that depends on the `ReadWriter` contract — "you can call
   both Read and Write" — is broken by this substitution. Fix: define a narrower
   `Reader` interface; `ReadOnlyBuffer` satisfies only that, making the promise it
   can actually keep. `ReadWriter` is reserved for types that genuinely support
   both operations.

4. When a consumer defines its own interface, it lists only the methods it actually
   calls. There is no fat interface to inherit. Each consumer creates its own narrow
   contract independently. Because Go uses implicit satisfaction, the concrete type
   never needs to know about the consumer's interface at all — it cannot be forced
   to implement methods it doesn't provide.

5. DIP says high-level modules should depend on abstractions, not on low-level
   concrete types. Constructor injection is the mechanism: the high-level type's
   constructor accepts an interface (the abstraction); the composition root passes
   in the concrete implementation. The high-level package defines the interface;
   the low-level package provides the implementation; `main()` wires them together.
   This is exactly the DI pattern from Chapter 28, viewed through the SOLID lens.
