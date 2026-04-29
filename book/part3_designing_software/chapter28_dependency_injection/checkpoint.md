# Chapter 28 — Revision Checkpoint

## Questions

1. What is the composition root and where does it live?
2. Why should dependencies be accepted as interfaces rather than concrete types?
3. What is the signature of a functional option and why does it return an error?
4. When should you prefer a config struct over functional options?
5. Why is `time.Now()` considered a hidden dependency?

## Answers

1. The **composition root** is the single place in the program where all concrete
   types are assembled and wired together. In Go it lives in `main()` (or the
   top-level initialiser of a binary). Everything outside `main()` works only
   with interfaces — it never imports or instantiates concrete types from other
   layers.

2. Accepting interfaces rather than concrete types means:
   - Tests can inject fakes without touching production code.
   - The service is decoupled from the concrete implementation — you can swap
     the database driver, mailer, or clock without changing the service.
   - Import cycles are avoided: the service package does not need to import
     the package that holds the concrete type.

3. `type Option func(*T) error`. It returns an error so that each option can
   validate the value it sets (e.g., reject port 0, reject negative timeouts).
   `NewServer` applies options in order and collects the first error, giving
   callers structured feedback without panicking.

4. Use a **config struct** when:
   - All callers are internal (you own both sides of the call).
   - No option needs validation logic.
   - All options are independent value assignments.
   - Backward compatibility across releases is not a concern.
   Functional options are preferred for public library APIs that need to evolve
   without breaking existing callers.

5. `time.Now()` is a hidden dependency because it is a global function whose
   output changes on every call. Code that calls it directly cannot be tested
   deterministically — a test cannot control what "now" means. Injecting a
   `Clock` interface (`Now() time.Time`) lets tests pass a `fixedClock` whose
   timestamp is known, making time-dependent logic fully deterministic.
