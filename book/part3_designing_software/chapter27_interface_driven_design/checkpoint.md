# Chapter 27 — Revision Checkpoint

## Questions

1. Where should you define an interface — in the producer or consumer package?
2. What is the import-graph benefit of consumer-side interfaces?
3. Why are narrow (1-method) interfaces preferred?
4. When should you NOT define an interface?
5. How does `io.Reader` demonstrate the consumer-side principle?

## Answers

1. In the **consumer** package — the package that uses the value, not the one
   that produces it. The concrete type satisfies the interface without knowing
   the interface exists.

2. Consumer-side interfaces reverse the import direction. The concrete type
   (e.g., a database driver) has no dependency on the consumers. Consumers
   can evolve their interfaces independently. Adding a method to one consumer's
   interface does not break other consumers or the concrete type.

3. Narrow interfaces are satisfied by more types, are easier to fake in tests,
   and can be composed into larger interfaces when needed. A one-method interface
   (`io.Reader`) is satisfied by files, network connections, buffers, and test
   strings — all without those types knowing about each other.

4. When there is only one implementation and no test fake needed; when you own
   both the consumer and the producer and can use the concrete type directly;
   when the "interface" would just mirror the concrete type's full API.

5. `io.Reader` is defined in the `io` package (a consumer — it *uses* readers).
   Concrete readers live in `os`, `net`, `bytes`, `strings`, `compress/gzip`,
   etc. — none of those packages imports `io` to get a base type; they just
   implement the one-method contract. Every consumer of `io.Reader` can use
   all those types without importing any of them.
