# Chapter 6 — Coming From Another Language: Mental Model Transfer

> **Reading time:** ~26 minutes (6,500 words). **Code:** 3 runnable
> programs (~210 lines). **Target Go version:** 1.22+.
>
> Almost no one's first language is Go. You arrive with habits — some
> good, some that will fight you. This chapter is a structured,
> language-by-language transfer guide: what carries over, what
> changes, and what habits to *unlearn*. By the end, you'll know
> which of your existing instincts to trust and which to suspend
> while you absorb idiomatic Go.

---

## 1. Concept Introduction

Different languages teach you different *defaults* for how a program
is shaped: how errors propagate, what concurrency looks like, where
state lives, how types are organized, what counts as ceremony. Most
of your "this is just how programs work" intuitions are
language-specific — and Go's choices differ from every mainstream
language in at least a few ways.

This chapter is a *translation table*: for Java, Python, JavaScript,
C++, and Rust, we walk through the major mental-model shifts and
list the patterns that translate cleanly versus the ones you'll have
to rebuild.

> **Working definition:** "transferring to Go" means keeping the
> habits that survive (clean naming, separation of concerns, tests
> as a first-class output) and surrendering the ones that don't
> (try/catch, classes, decorators, async/await, ownership). The
> faster you let go of the latter, the faster Go feels natural.

---

## 2. Why This Exists

Most Go books treat the reader as a blank slate. That's wasteful:
you arrive with years of language-specific habits, and the *speed*
at which you become productive in Go depends on which of those
habits you reuse versus replace. A direct, opinionated translation
table is the fastest path.

It also helps with the inverse problem: when you can't find a Go
feature you expect (no `try`, no decorators, no `async`), you spend
hours wondering if you missed it. This chapter tells you up front
*there is no equivalent* and shows you what the Go answer is, so you
stop looking.

---

## 3. Problem It Solves

1. **Habit interference.** Coming from a language with classes, you
   instinctively reach for inheritance; Go fights back. This chapter
   tells you to use embedding and interfaces instead.
2. **"Where's the X?" frustration.** No try/catch (Go has explicit
   error returns), no async/await (Go has goroutines), no decorators
   (Go has middleware as a function-composition pattern).
3. **Translation noise.** A Python developer writing Go often writes
   "Python-with-Go-syntax." Knowing what's idiomatic up front saves
   you a year of code reviews.
4. **Concurrency culture shock.** Languages like Python, JavaScript,
   Java teach concurrency as "advanced." Go puts it on day one. The
   adjustment is not technical; it's cultural.
5. **Tooling expectations.** Coming from JS, you expect to pick a
   build tool, a test runner, a formatter, a linter. Go has one of
   each. Knowing this in advance avoids an afternoon of comparison.

---

## 4. Historical Context

Go's design borrows from many places, deliberately. Knowing the
heritage explains some of the surprises:

* **From C.** Syntax (curly braces, `for`, type-after-name in
  declarations), the *small standard library + good tooling* ethos,
  pointers, the `printf` family. Ken Thompson co-designed both.
* **From Pascal.** The "type after the name" declaration order
  (`x int` rather than `int x`), the "no implicit conversions"
  discipline, the explicit package boundaries.
* **From Modula and Oberon.** Module system, package-level
  visibility by capitalization (Niklaus Wirth's invention),
  composition over inheritance.
* **From CSP (Hoare, 1978).** Goroutines + channels. The language's
  concurrency model is almost a direct implementation of the CSP
  paper.
* **From Plan 9.** The toolchain ancestry, the assembly conventions,
  the build cache idea.
* **From Java.** Lessons learned in the negative — what *not* to do
  (deep hierarchies, checked exceptions, abstract factories).
* **From Erlang.** "Lots of cheap concurrent processes" as a
  first-class workload.

Go is *not* descended from Lisp, ML, or Haskell, and it shows: no
algebraic data types (until generics-as-they-exist-now, partially),
no pattern matching, no first-class macros, no immutable-by-default
values. If you're coming from a functional language, expect more
adjustment than your Java/Python peers.

---

## 5. How Industry Uses It

Senior engineers transferring to Go hit predictable inflection
points:

* **Week 1:** "Why won't it let me have unused imports?" → realize
  this is a feature, configure editor to auto-organize.
* **Week 2:** "Where are the exceptions?" → start writing
  `if err != nil` everywhere, complain.
* **Week 3:** Write the first goroutine + channel pattern, feel
  unreasonably proud.
* **Week 4:** Try to build a deep class hierarchy with embedded
  structs, fail, learn about interface-driven design.
* **Month 2:** Stop trying to write Java/Python/JS-with-Go-syntax;
  start writing Go.
* **Month 3:** Discover `pkg.go.dev` and stop reaching for third-
  party libraries reflexively.
* **Month 6:** Become opinionated about layout, errors, and
  concurrency patterns. Now contributing reviews to teammates.

This is the typical curve at companies that hire experienced
engineers from other stacks. The "sit with the differences" step at
month 2 is the one most people skip; this chapter exists to
short-circuit it.

---

## 6. Real-World Production Use Cases

**Java team migrates a microservice to Go.** A six-engineer team
rewrites a Spring Boot service in Go. They:

* Replace the dependency-injection framework (Spring) with manual
  constructor injection — the resulting code is shorter and the
  startup is faster (50ms vs 8 seconds).
* Replace Spring Web with `net/http` + `chi`. Lines of code drop
  significantly.
* Replace Maven with `go.mod`. Build times go from 4 minutes to
  20 seconds.
* Replace JUnit with `testing` + `testify`. They keep `testify`
  for assertions; everything else they let go.

The hardest adjustment, by their own account, was *not having
exceptions*. The team's senior architect spent two weeks walking
juniors through error wrapping idioms.

**Python team adds a Go service to a Python codebase.** A team
running a Django monolith adds a high-throughput proxy in Go because
the Python service can't keep up with concurrent connection load.
They:

* Write the proxy in 600 lines of Go.
* Reuse the team's Postgres schema unchanged.
* Add a `--migrate` flag that runs schema migrations from the same
  binary.
* Deploy as a single static binary on the existing fleet.

Total time from "let's try Go" to "in production": 3 weeks.

**JavaScript team builds a CLI in Go.** A team running a Node
monorepo builds a CLI tool in Go for ops engineers. Reasons: the
JavaScript version requires installing Node; the Go version is one
binary. Cross-platform releases (macOS Intel + Apple Silicon, Linux
amd64 + arm64, Windows) ship from one build server with `goreleaser`.
The team's TypeScript skills transferred fully on type-system
intuition; the goroutine model was the surprise.

**C++ team writes a control plane in Go.** A team running a C++
data-plane writes the control plane in Go: lower-throughput, but
much higher rate of feature delivery. The team retains C++ in the
hot path; uses Go where developer velocity matters more than the
last 20% of single-thread performance.

**Rust team adopts Go for service tooling.** A team using Rust for
their high-performance core writes operational tooling (CLI, admin
UI backend, ops automation) in Go. Reason: their team can be
productive in Go in days; Rust takes weeks for the same task. They
get the right language for each job without retraining everyone.

---

## 7. Beginner-Friendly Explanation

If you're moving from any other language, the *biggest* mental shift
is this:

> **Go is small on purpose.** Almost every "Go doesn't have X" you'll
> notice is a deliberate omission, not a bug. The language designers
> believe the absence of X improves team-scale codebase health. Your
> first instinct will be to look for X. Suppress that instinct for
> three months and you'll start to understand why.

The second-biggest shift is concurrency:

> **Goroutines are cheap and meant to be used.** You'll come from a
> language where concurrency means careful, infrequent threads. In
> Go, spawning a goroutine for every connection, every request,
> every job is fine. The runtime is built for it.

The third-biggest shift is errors:

> **Errors are values; you handle them inline.** No exceptions. No
> stack-unwinding control flow. Every function that can fail
> returns `(value, error)`, and every caller checks the error
> *right there, on the next line.* This feels verbose at first;
> it becomes clarity.

The fourth-biggest shift is types:

> **Types are simple; interfaces are implicit.** No generics-
> everywhere ceremony, no `extends`, no `implements`. A type
> "satisfies" an interface by having the methods. The compiler
> figures it out.

If you internalize those four shifts, you're 80% of the way there.

---

## 8. Deep Technical Explanation

The rest of this chapter is the language-by-language transfer table.
Pick the section for your background; skim the others for context.

### 8.1. Coming from Java

Things that translate cleanly:

* Strong static typing.
* Method receivers (in Go, you write `func (u *User) Save()` — same
  intent as Java's `class User { void save() {...} }`).
* Test-first culture.
* Build tooling that respects modules (`go.mod` ≈ `pom.xml`).
* Standard library coverage of HTTP, JSON, crypto, time.

Things that change:

* **No classes.** Use struct + methods + interfaces. Composition is
  *embedding* — a struct that includes another struct gets its
  methods promoted, but it's *not* inheritance.
* **No exceptions.** Functions that can fail return `(T, error)`.
  `panic` exists, but it's for genuinely-unrecoverable errors and
  is recovered only at process boundaries.
* **No constructors.** Convention: a function `NewT()` that returns
  `*T` or `(T, error)`. The function lives in the same package as
  `T`.
* **No annotations.** No `@Override`, no `@Autowired`, no
  `@RestController`. Wiring is by hand or by code generation.
* **No generics-everywhere.** Generics exist (since 1.18) but are
  used sparingly. Most Go code is non-generic.
* **No `final`, no `private`, no `public`, no `protected`.**
  Capitalization decides visibility. `Save` exported; `save`
  unexported. Field-level same rule.
* **No checked exceptions.** Whether a function returns `error` is
  part of its signature; the compiler enforces you handle the
  error or explicitly discard it.
* **No DI frameworks.** Constructor injection by hand. For larger
  graphs, code generation (Wire, Dig) is occasionally used. Most
  teams don't need it.
* **No reflection-based ORMs by default.** ORMs exist (GORM) but
  most production Go uses raw SQL or `sqlc` (code generation
  instead of reflection).

Java idiom → Go translation table:

| Java | Go |
| --- | --- |
| `class User { ... }` | `type User struct { ... }` |
| `extends` | embed: `type Admin struct { User; ... }` |
| `implements` | implicit; just have the methods |
| `try { ... } catch (E e) { ... }` | `if err != nil { ... }` |
| `throw new IllegalArgumentException(...)` | `return fmt.Errorf("invalid: %s", arg)` |
| `Optional<T>` | `*T`, with `nil` for absence; or `(T, bool)` |
| `List<T>`, `Map<K,V>` | `[]T`, `map[K]V` |
| `final` field | unexported field, set only by constructor |
| `static` member | package-level variable or function |
| `synchronized` | `sync.Mutex` (or, better, a channel) |
| `CompletableFuture<T>` | a goroutine + channel |
| `@Nullable` / `@NotNull` | nil-checks; no annotations |

The single biggest *cultural* difference: idiomatic Go is anti-
ceremonial. A 200-line Spring controller compresses to a 30-line
Go handler. If your Go code looks like Java with `{}` and `:=`,
you're doing it wrong.

### 8.2. Coming from Python

Things that translate cleanly:

* "Indentation/syntax is shallow; the structure is what matters."
* Module-as-directory thinking (Python's `mymod/` ≈ Go's package).
* Strong stdlib culture.
* Test-first culture.

Things that change:

* **Static typing, checked at compile time.** No more
  `TypeError: 'NoneType' object has no attribute 'foo'` — the
  compiler catches it.
* **No exceptions.** Functions return `(value, error)`. The Python
  `try/except` reflex must die; replace with `if err != nil`.
* **No decorators.** Same effect via function wrapping (HTTP
  middleware, e.g. `withAuth(handler)`).
* **No `async/await`.** Concurrency is goroutines + channels.
  There's no event loop; the scheduler is preemptive.
* **No list comprehensions.** Use a for loop with `append`. It's
  fine; the compiler optimizes well.
* **No `**kwargs`.** Use functional options (a pattern we'll cover
  in Chapter 28) or a config struct.
* **No duck typing at runtime.** Duck typing exists, but it's
  *static* — interfaces are checked at compile time, not runtime.
* **No REPL.** `go run` is the closest thing. There is `gore` (a
  third-party REPL) but it's rarely used.
* **No `pip install` runtime cost.** Modules are downloaded once
  per version and cached forever; subsequent builds use the cache.
* **No GIL.** Multiple goroutines genuinely run in parallel on
  multiple CPUs by default.

Python idiom → Go translation:

| Python | Go |
| --- | --- |
| `def f(*args, **kwargs):` | `func F(opts ...Option) { ... }` |
| `try: ... except: ...` | `if err := f(); err != nil { ... }` |
| `[x*2 for x in xs]` | `for i, x := range xs { result[i] = x*2 }` |
| `with open(f) as fh:` | `f, err := os.Open(...); defer f.Close()` |
| `class Animal: ...` | `type Animal struct { ... }` |
| `@dataclass` | a struct (no decorator needed) |
| `@property` | unexported field + getter method |
| `async def f():` | `func F() {} ` and `go F()` |
| `asyncio.gather` | `errgroup.Group` (Chapter 48) |
| `if x is None:` | `if x == nil:` (for pointers, slices, maps, interfaces) |
| `requests.get(url)` | `http.Get(url)` (stdlib) |

The cultural shift: Go is more verbose at the line level than
Python, less verbose at the architecture level. A "Pythonic" Go
program — one-liners, dunder methods, `lambda`s — is awkward Go.
Lean into the slight verbosity; it pays off in code review.

### 8.3. Coming from JavaScript / TypeScript

Things that translate cleanly:

* TypeScript-style static typing (Go's is somewhat simpler).
* Module-per-folder thinking.
* "Async at boundaries" mentality.
* CLI-tooling culture.
* Standard test runner (TS has Jest/Vitest; Go has `testing`).

Things that change:

* **No `null`/`undefined` distinction.** Go has *zero values* for
  every type (0, "", false, nil). For pointers/slices/maps/channels
  /interfaces, the zero value is `nil`. There is no separate
  `undefined`.
* **No truthy/falsy nonsense.** `if x` doesn't compile if `x` is
  not `bool`. Be explicit: `if x != ""`, `if x != 0`, `if x != nil`.
* **No `==` vs `===` debate.** `==` does what `===` does in JS.
* **No prototype chains.** Composition + interfaces.
* **No hoisting.** Declarations must precede uses (well — at
  package level Go is order-independent for declarations, but
  within a function it's strictly top-to-bottom).
* **No `await`.** Goroutines + channels. The model is different;
  it'll take a couple of weeks to retrain.
* **No `Promise`-like type.** A function that takes a callback or
  returns through a channel is the closest analogue.
* **No JSX or templating-in-string-literal magic.** Templates use
  the `text/template` and `html/template` stdlib packages.
* **No NPM.** Modules come from Git URLs via the module proxy.
  No central registry. No `package.json` lockfile drift.
* **One language, no transpilation.** What you write is what runs.
  No Babel, no webpack, no `tsconfig.json`.

JavaScript/TS idiom → Go translation:

| JavaScript / TypeScript | Go |
| --- | --- |
| `class Foo { ... }` | `type Foo struct { ... }` |
| `interface Foo { ... }` | `type Foo interface { ... }` |
| `async function f() { ... }` | `func F() { ... }` and `go F()` |
| `Promise<T>` | a channel of `T` (or `(T, error)` return + goroutine) |
| `await fetch(url)` | `http.Get(url)` (synchronous; goroutine wraps it) |
| `try { ... } catch (e) { ... }` | `if err != nil { ... }` |
| `[1, 2, 3].map(x => x*2)` | `for i, x := range xs { ys[i] = x*2 }` |
| `Object.assign({}, src)` | struct copy assignment, or explicit `for k, v := range src` |
| `null \| undefined` | `nil` (one concept) |
| `import x from "y"` | `import "y"` |
| `npm install` | `go mod tidy` |
| `package.json` | `go.mod` |
| `package-lock.json` | `go.sum` |
| `.eslintrc` | `.golangci.yml` |

The cultural shift: Go has no front-end / back-end split; it's a
back-end language. Concepts like "the bundler," "the transpiler,"
"the polyfill" don't exist.

### 8.4. Coming from C++

Things that translate cleanly:

* Pointers (without arithmetic).
* Manual memory layout intuition (struct field ordering matters).
* Stack vs heap thinking (escape analysis decides for you, but
  the intuition transfers).
* "Compile to a binary, ship it" mental model.
* Static linking instinct.

Things that change:

* **GC.** No `new`/`delete`. No RAII. The runtime cleans up.
* **No templates.** Generics (1.18+) exist but are intentionally
  far less expressive than C++ templates. No template
  metaprogramming. No SFINAE. No partial specialization.
* **No multiple inheritance.** Embedding gives you most of what
  you actually wanted; the rest you do without.
* **No move semantics.** Go values are either copied (`value`)
  or shared via pointer. No `std::move`, no rvalue references.
* **No const-correctness.** Go has no `const` for variables (only
  for compile-time constants). Convention: don't mutate something
  that conceptually belongs to a caller.
* **No header files.** Imports + capitalization. No forward
  declarations. No `#ifdef`. (Build tags handle conditional compile.)
* **No undefined behavior in normal code.** Race conditions
  exist but are caught by `go test -race`. Out-of-bounds slice
  access panics; it doesn't corrupt memory.
* **No operator overloading.** `+` only does what the language
  says it does for `int`/`float`/`string`. No `__add__`.
* **No deterministic destructors.** `defer` is the closest, and
  it fires at function exit, not scope exit. RAII patterns become
  explicit `defer`.
* **No cgo by default.** You can call C, but it costs you the
  static-binary story. Most Go avoids C entirely.

C++ idiom → Go translation:

| C++ | Go |
| --- | --- |
| `class Foo { public: ... };` | `type Foo struct { ... }` |
| `virtual void f() = 0;` | `interface { F() }` |
| `std::unique_ptr<T>` | `*T` with explicit ownership convention |
| `std::shared_ptr<T>` | `*T` (GC handles refcounts implicitly) |
| `std::string` | `string` (immutable in Go) |
| `std::vector<T>` | `[]T` (slice) |
| `std::map<K,V>` | `map[K]V` (hash, not ordered) |
| `RAII destructor` | `defer cleanup()` |
| `template<typename T>` | `func F[T any](x T) ...` (1.18+) |
| `const T&` parameter | `T` (value semantics) |
| `T&` (out param) | `*T` parameter |
| `std::thread` | `go f()` |
| `std::mutex` | `sync.Mutex` |
| `std::condition_variable` | channel + select |

The cultural shift: idiomatic Go is much less concerned with the
last 20% of perf. You'll have a reflex to optimize allocations and
cache locality; relax it. Profile first; most "obvious"
optimizations don't move the needle in a typical service.

### 8.5. Coming from Rust

Things that translate cleanly:

* "Static binaries are good, runtimes are bad" instinct.
* Pattern of "small focused crates"/packages.
* "Errors are values, not exceptions" thinking.
* Test-first, doc-first culture.
* "Compile-time safety > runtime checks" disposition.

Things that change:

* **GC instead of ownership.** No borrow checker. No lifetimes.
  No `Rc`/`Arc` distinction. The runtime tracks references for
  you; `nil` is a real possibility you have to reason about.
* **No `Result<T,E>` enum.** Errors are an interface (`error`),
  returned as a separate value. No `?` operator; explicit `if err
  != nil` everywhere. `errors.Is`/`errors.As` give you typed
  matching.
* **No `Option<T>`.** Pointers (`*T`) double as optionality with
  `nil`, or use a `(T, bool)` "comma-ok" return.
* **No `match`.** Use `switch`. It's similar but less expressive
  (no destructuring, no exhaustiveness check).
* **No traits with default methods.** Interfaces are smaller.
* **No `trait` object orphan rule.** Go interfaces are implicit;
  any type satisfies any interface whose methods it has, anywhere.
* **No `unsafe`-as-feature.** Go has `unsafe`, but it's a small
  package, not the gateway to a different dialect of the language.
* **No async ecosystem.** No `tokio`/`async-std` decision. Just
  goroutines and the scheduler.
* **No macros.** `go generate` (code generation invoked via
  comments) is the closest thing. Less powerful, less footgun.
* **No build profiles.** No `--release` vs debug split. One Go
  build is the build; flags choose stripped vs not.

Rust idiom → Go translation:

| Rust | Go |
| --- | --- |
| `struct Foo { ... }` | `type Foo struct { ... }` |
| `impl Foo { fn bar(&self) ... }` | `func (f *Foo) Bar() { ... }` |
| `trait Foo { ... }` | `type Foo interface { ... }` |
| `Result<T, E>` | `(T, error)` |
| `Option<T>` | `*T` with nil, or `(T, bool)` |
| `?` operator | `if err != nil { return err }` |
| `match x { ... }` | `switch x { ... }` |
| `tokio::spawn(async { ... })` | `go func() { ... }()` |
| `Arc<Mutex<T>>` | `*T` + `sync.Mutex` (or use a channel) |
| `Box<dyn Trait>` | `interface{ ... }` value |
| `Cargo.toml` | `go.mod` |
| `cargo build` | `go build` |
| `clippy` | `golangci-lint` |
| derive macros | `go:generate` directives |

The cultural shift: idiomatic Go takes shortcuts Rust would not.
A `nil` pointer is fine if invariants are clear. A goroutine leak
is a code smell, not a compile error. The Go answer to "could this
fail at runtime?" is often "yes, and we have a good story for
recovering" rather than "we made it impossible at compile time."

This is the trade Go made: less safety than Rust, much more
productivity. The right answer depends on the system. Don't think
of it as a regression — think of it as a different point on the
safety/productivity Pareto frontier.

---

## 9. Internal Working (How Go Handles It)

Not directly applicable — this chapter is about translation, not
internals. The relevant internals (runtime, GC, scheduler) are
covered in Chapters 9, 42, 89.

---

## 10. Syntax Breakdown

Three side-by-side translation snippets follow as examples in
Section 11. Read those rather than learning syntax abstractly.

---

## 11. Multiple Practical Examples

### Example 1 — `examples/01_translation_table`

```bash
go run ./examples/01_translation_table
```

A program that prints the translation tables from this chapter, in
your terminal, for a chosen source language. Filter with an arg:
`go run ./examples/01_translation_table python`.

### Example 2 — `examples/02_python_to_go`

```bash
go run ./examples/02_python_to_go
```

A small "WordCounter" written in Go that mirrors a hypothetical
Python implementation. The file's comments show, line-by-line, the
Python original and the Go translation. Read top to bottom; the
mental shift is in the diff.

### Example 3 — `examples/03_javascript_to_go`

```bash
go run ./examples/03_javascript_to_go
```

A "fetch-N-URLs-concurrently" program. The file's comments show
the JavaScript `Promise.all` version next to the Go goroutine +
errgroup version. Notice that the Go version is *more* lines but
*less* clever — the kind of trade Go consistently asks you to make.

---

## 12. Good vs Bad Examples

**Good Java → Go:** keep your domain modeling, drop the ceremony.

```java
// Java
@RestController
@RequestMapping("/users")
public class UserController {
    @Autowired private UserService service;

    @GetMapping("/{id}")
    public ResponseEntity<User> get(@PathVariable Long id) {
        Optional<User> u = service.find(id);
        return u.map(ResponseEntity::ok)
                .orElseGet(() -> ResponseEntity.notFound().build());
    }
}
```

```go
// Go
type UserHandler struct {
    svc *UserService
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
    if err != nil {
        http.Error(w, "bad id", http.StatusBadRequest)
        return
    }
    u, err := h.svc.Find(r.Context(), id)
    if errors.Is(err, ErrNotFound) {
        http.NotFound(w, r)
        return
    }
    if err != nil {
        http.Error(w, "internal", http.StatusInternalServerError)
        return
    }
    _ = json.NewEncoder(w).Encode(u)
}
```

The Go version is shorter once you remove the framework. It's also
explicit about every error path.

**Bad Java → Go:** Java with Go syntax.

```go
// Don't do this:
type AbstractBaseUserService struct{}
type UserServiceImpl struct {
    AbstractBaseUserService
    repo UserRepository
}
type UserServiceFactory struct{}
func (f *UserServiceFactory) Create() *UserServiceImpl { ... }
```

This is Spring-with-curly-braces. Idiomatic Go would be:

```go
type UserService struct {
    repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
    return &UserService{repo: repo}
}
```

No factory. No abstract. No "Impl" suffix.

---

## 13. Common Mistakes

1. **Translating `try/except` literally as `if err != nil { panic }`.**
   Go's panic is for unrecoverable invariant violations. Real
   errors return `error`.
2. **Reaching for inheritance via embedding.** Embedding *looks*
   like inheritance but isn't — there's no virtual dispatch on
   embedded methods. Use interfaces.
3. **Building a deep package hierarchy.** Coming from Java, you'll
   instinctively make `com.example.app.service.user.repository.impl`-
   shaped paths. Don't. Flatten.
4. **Naming interfaces with `I` prefix or `able` suffix.** Java
   convention. Go convention: name interfaces after their *behavior*.
   `io.Reader`, not `IReader`. `fmt.Stringer`, not `Stringable`.
5. **Returning `interface{}` (or `any`) by default.** Don't.
   Return concrete types where you can; let callers up-cast to
   interfaces if they need to.
6. **Writing `getX`/`setX` accessor methods reflexively.** Most Go
   uses public fields. Add an accessor only when there's a real
   reason (validation, lazy init).
7. **Using `nil` checks in places where the zero value would do.**
   In Go, `""`, `0`, `false`, empty slice, and `nil` pointer are
   all *distinct*. You usually want one specific check.
8. **Making everything a goroutine.** Just because they're cheap
   doesn't mean every function should run in one. Spawn when you
   need concurrency, not as a default.
9. **Using channels for everything.** Channels are for
   communication between goroutines. For mutable shared state
   that's only used by one goroutine, plain values are fine. For
   shared state used by many, a `sync.Mutex` is sometimes simpler
   than channels.
10. **Reaching for a framework.** Coming from Java/Python/JS,
    "find a framework" is muscle memory. Don't. The standard
    library does most of the framework's job.

---

## 14. Debugging Tips

* When you find yourself thinking "Go won't let me do X," check
  the FAQ first. `go.dev/doc/faq` has answers for most common
  surprises.
* When the compiler error doesn't make sense, run `go vet ./...` —
  often it surfaces a clearer related issue.
* When you can't figure out the idiomatic form, search
  `pkg.go.dev` for how the standard library solves the same shape
  of problem. Stdlib is the Rosetta Stone.
* When error handling feels too verbose, look up `errors.Join`
  and `fmt.Errorf("...: %w", err)`. The right idiom is shorter
  than you think.

---

## 15. Performance Considerations

Cross-language performance comparisons are rarely useful in the
abstract. Two specific points:

* **Go is faster than Python and Ruby**, typically 10–100x,
  because Go is compiled and statically typed. If you're moving
  *to* Go from a dynamic language, expect a perf win essentially
  for free.
* **Go is slower than C/C++/Rust on tight CPU loops**, typically
  by 1.5–3x, because Go has GC and less aggressive optimization.
  If you're moving *from* C++, expect to give up some single-
  thread perf for a lot of productivity.

For most production services (network-bound, database-bound), the
language is not the bottleneck. Pick for productivity, not for the
benchmark.

---

## 16. Security Considerations

Translating code, *not* translating security assumptions:

* **Coming from Python/JS:** Go's static binary plus the standard
  library's curated crypto is a security upgrade. Most "implicit"
  Python/JS vulns (eval, prototype pollution, pickle) don't have
  Go analogues.
* **Coming from Java:** Go's smaller dep tree is a smaller supply-
  chain surface. Spring's deserialization CVEs don't have Go
  equivalents at the same scale. The trade: less mature security
  scanning ecosystem (though `govulncheck` is good).
* **Coming from C/C++:** Memory safety is a genuine win. Most
  CVE-class memory bugs (buffer overflow, UAF) don't apply to Go.
  Race conditions still do.
* **Coming from Rust:** You're giving up some compile-time safety
  for productivity. Audit your concurrent code more carefully than
  Rust would force you to.

---

## 17. Senior Engineer Best Practices

1. **Spend the first month writing pure Go**, not your-old-language-
   in-Go. Reject "but in X we did it like…" reflexively for 30
   days.
2. **Read 1,000 lines of standard-library source** before reaching
   for third-party libraries. The stdlib is the canonical idiom.
3. **Pair-review for the first three PRs** with someone fluent in
   Go. The patterns reveal themselves faster with feedback.
4. **Don't recreate frameworks.** The first time you reach for
   "Spring for Go" or "Express for Go," recognize it as a habit
   to suppress.
5. **Embrace verbosity at the line level.** `if err != nil` is
   the price of clarity at the architecture level.
6. **Translate big-picture first**, syntax second. The shape of a
   Go service isn't the shape of a Spring service even if every
   line maps cleanly.
7. **Read the Go FAQ** end to end. Most "why doesn't Go have X"
   is answered there.
8. **Use `golangci-lint` from day one.** The linter is your
   second mentor when the first one isn't around.
9. **Write at least one goroutine + channel program in your first
   week.** The mental model only solidifies when you write it.
10. **Suspend judgment for 90 days.** Your first month of opinions
    will be wrong; revisit them after three months of writing
    real Go.

---

## 18. Interview Questions

1. *(junior)* Java has classes; Go doesn't. How do you express
   "a thing with state and behavior" in Go?
2. *(junior)* Python has try/except; Go doesn't. How do you
   handle errors?
3. *(mid)* JavaScript has `async/await`; Go doesn't. How does Go
   express asynchrony?
4. *(mid)* C++ has templates; Go has generics — what's
   intentionally less expressive about Go's?
5. *(senior)* Rust has `Result<T, E>` and `?`; Go has `(T,
   error)` and explicit checks. Trade-offs?
6. *(senior)* You're leading a team migrating a Spring Boot
   service to Go. What are the three biggest pitfalls you warn
   them about?
7. *(staff)* When does it make sense to keep something in
   another language and write *only the surrounding* in Go?

---

## 19. Interview Answers

1. A `struct` (state) plus methods on that struct (behavior).
   Composition via embedding when one type "is a kind of"
   another. Polymorphism via interfaces, which are satisfied
   implicitly. There's no `extends`, no `implements`, no
   abstract base class.

2. Functions return `(value, error)`. Callers check `err != nil`
   immediately. Errors propagate by being returned (not unwound).
   `errors.Is`/`errors.As` give typed matching;
   `fmt.Errorf("...: %w", err)` wraps for context. `panic` exists
   for genuinely-unrecoverable errors and is rare in normal code.

3. Goroutines + channels. `go func() { ... }()` schedules a
   function to run concurrently. Coordination via channels, the
   `sync` package, or `errgroup` for "wait for these N tasks." No
   event loop; the scheduler is preemptive. The mental model is
   different — you write what looks like sync code, the runtime
   makes it concurrent.

4. No variance, no covariant return types, no method-level
   generics, no negative type-set constraints, no template
   metaprogramming. The design rejects what the C++ template
   system became. Go generics are deliberately the *minimum*
   feature that lets you write a type-safe `Min`, `Max`, `Slices`,
   and a few generic data structures — and stop there.

5. Rust's `?` is more concise; Go's explicit checks are more
   visible. Rust's `Result` is exhaustively typed; Go's `error`
   is an interface, more flexible but less precise. Rust forces
   you to reason about the failure case; Go encourages it but
   doesn't force it (`_ = err` compiles). The trade is concision
   vs. ceremony for productivity vs. enforced correctness; both
   defensible.

6. (a) **No exceptions** — drill the team on `if err != nil`
   patterns and `errors.Is/As`. (b) **No DI framework** — accept
   that wiring is by hand; the resulting code is shorter. (c)
   **No deep package hierarchy** — flatten; `cmd/` plus
   `internal/` is enough.

7. When you're keeping the *hot path* in a language with better
   single-thread perf or memory control, and writing the
   *control plane* in Go for productivity. Common pattern: C++
   data plane + Go control plane. Or: Rust core + Go ops tooling.
   The decision is "where is developer velocity worth more than
   perf," and that's usually true everywhere except the inner
   loop of a small number of CPU-bound services.

---

## 20. Hands-On Exercises

**Exercise 6.1 — Translate one of your old programs.**

**Goal.** Force the mental shift by doing the work.

**Task.** Pick a 50–200 line program you've written in your
previous language. Translate it to Go *without keeping the
structure of the original.* Don't translate line-by-line; rewrite
the program as you'd write it natively.

**Acceptance.** The Go version compiles, passes the same tests,
and is *not* obviously a translation. Code review yourself: does
it use `error` as values, structs and methods, packages
appropriately, idiomatic naming?

**Exercise 6.2 — Read translation table tool.**

```bash
go run ./examples/01_translation_table python
go run ./examples/01_translation_table java
go run ./examples/01_translation_table javascript
go run ./examples/01_translation_table cpp
go run ./examples/01_translation_table rust
```

For your background, read the table and pick three rows that
*surprise* you. Write a one-paragraph note on why the surprise
exists.

**Exercise 6.3 ★ — Implement a Promise-like primitive.**

**Goal.** Internalize the difference between Go's concurrency
model and JS's.

**Task.** Implement a `Future[T]` type in Go with `Get(ctx
context.Context) (T, error)` and `Then(fn func(T) T) Future[T]`
methods. Start it as a goroutine; coordinate via channels. Test
it with a chain of three transformations.

**Acceptance.** A working generic future type, ~50–100 lines.

(You'll know everything you need by Chapter 47. This is a
landmark exercise to come back to.)

---

## 21. Mini Project Tasks

**Task — Document your team's conventions.**

For your team (current or future), write a CONVENTIONS.md that
captures the decisions a senior Go engineer would make
implicitly: error wrapping style, package naming, layout choice,
linting rules, doc-comment expectations, test layout. The
exercise of writing it forces explicit decisions; the artifact
helps onboarding.

---

## 22. Chapter Summary

* Go is small on purpose; many "missing" features are deliberate
  omissions designed to improve team-scale codebase health.
* The four biggest mental shifts: explicit errors as values,
  goroutines + channels for concurrency, structs + interfaces for
  modeling, and small focused packages instead of frameworks.
* Java translates cleanly at the type level; the shift is in
  visibility, error handling, and DI patterns.
* Python translates cleanly at the module level; the shift is in
  static typing, explicit errors, and concurrency.
* JavaScript translates cleanly at the package-per-folder level;
  the shift is in static types, the absence of `null`/`undefined`
  distinction, and the goroutine model.
* C++ translates cleanly at the static-binary level; the shift is
  to GC, no templates, no operator overloading, no deterministic
  destructors.
* Rust translates cleanly at the "errors as values" level; the
  shift is to GC, no ownership, less compile-time safety in
  exchange for productivity.
* The right disposition for the first 90 days: write Go, not
  X-with-Go-syntax. Suspend judgment.

Updated working definition: *"transferring to Go" means keeping
the habits that survive (clean naming, separation of concerns,
test culture) and surrendering the ones that don't (try/catch,
classes, decorators, async/await, ownership). The faster you let
go of the latter, the faster Go feels natural.*

---

## 23. Advanced Follow-up Concepts

* **Effective Go** (`go.dev/doc/effective_go`) — the canonical
  short style guide. Read it before writing any production Go.
* **The Go FAQ** (`go.dev/doc/faq`) — every "why doesn't Go..."
  question has an answer here.
* **Rob Pike, "Less is exponentially more"** — the philosophy in
  one talk.
* **William Kennedy, "Ultimate Go"** — book and talks. A senior-
  practitioner's translation guide.
* **Jaana Dogan, "Five things that make Go fast"** — the
  performance-focused intro that helps C++ refugees adjust.
* **The Go Style Guide** (Google's internal style guide,
  published 2022) — the most opinionated style reference. Long;
  worth reading once.

> **End of Chapter 6.** Move on to [Chapter 7 — Your First Real
> Program (CLI Word Counter)](../chapter07_first_real_program/README.md).
