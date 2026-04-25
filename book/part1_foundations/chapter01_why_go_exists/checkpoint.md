# Chapter 1 — Revision Checkpoint

Answer each question in your own words *before* peeking at the answer.
If you can't answer, re-read the relevant section of the chapter.

---

## Questions

1. Who designed Go, in what year, and at what company?
2. Name three pain points the Go authors were responding to when they
   designed the language.
3. Why is a Go "hello world" binary about 2 MB on disk?
4. What does it mean that Go is "small on purpose"? Give two specific
   features Go *does not* have, and the design rationale for each
   omission.
5. What is a goroutine, and how does it differ from an OS thread?
6. What is the relationship between channels and the CSP model?
7. Name three components of the Go runtime that ship inside every
   binary.
8. Why is the standard library considered an unusually opinionated and
   capable part of the language?
9. What is the role of `gofmt` in Go culture?
10. In which kinds of systems would you *not* choose Go, and why?

---

## Answers

1. Go was designed at Google starting in late 2007 by **Robert
   Griesemer**, **Rob Pike**, and **Ken Thompson**. It was open-sourced
   in November 2009; Go 1.0 shipped in March 2012.

2. Any three of: slow C++ build times at Google scale; concurrency that
   was a library afterthought; ceremonial verbosity in Java; manual
   memory management's security cost in C/C++; lack of standard tooling
   for large teams; no standard answer to dependency versioning.

3. Because the Go runtime — scheduler, garbage collector, memory
   allocator, channel/select machinery, reflection metadata — is
   **statically linked into every binary**. The trade is that you ship
   a single file with no runtime dependency on the host machine. You
   can shrink with `-ldflags '-s -w'` and `upx`, but the floor is the
   runtime.

4. The Go specification is roughly 90 pages; C++ is over 1,500. The
   philosophy is *team-scale codebase health > individual-developer
   ergonomics*. Examples: **no classes/inheritance** (composition via
   embedding plus interfaces is judged sufficient and avoids fragile
   base-class problems); **no exceptions** (explicit `(value, error)`
   returns make error paths visible at the type level).

5. A goroutine is a function scheduled by the **Go runtime** rather
   than the OS. Goroutines start with a 2 KB growable stack
   (vs. 1–8 MB for an OS thread), are multiplexed M:N onto OS threads
   by the Go scheduler, and cost ~100 ns to spawn. You can comfortably
   have hundreds of thousands of them; you cannot do that with threads.

6. Channels are the implementation of CSP — Tony Hoare's 1978
   *Communicating Sequential Processes* — in Go. The model: independent
   sequential processes that communicate by passing typed messages
   through synchronization channels, rather than by sharing memory.
   Go's slogan: *do not communicate by sharing memory; share memory by
   communicating.* Chapter 41 walks through the paper.

7. Any three of: the **goroutine scheduler** (M:N threading), the
   **garbage collector** (concurrent tri-color mark-sweep), the
   **memory allocator** (tcmalloc-derived with size classes), the
   **channel runtime** (`runtime/chan.go`), the **defer/panic/recover
   machinery**, the **reflection metadata reader**.

8. Out of the box you get a production HTTP/1.1+HTTP/2 server, full
   crypto, JSON/XML/CSV encoders, a SQL interface, a test framework
   with benchmarking and fuzzing, structured logging (`log/slog`),
   text and HTML templates, RE2 regex, the concurrency primitives, and
   `context`. The opinion is that this is enough for servers and
   tools without a framework — and for most Go services in production,
   it is.

9. `gofmt` removes formatting opinions from code review. There is one
   official answer to "how should this be indented and spaced?", and
   it is enforced by a tool. The cultural effect is that Go codebases
   across companies look recognizably similar, which makes it easier
   to read code you didn't write.

10. **Hot-loop numerical code** (C++/Rust still wins on tight CPU
    loops). **Embedded firmware** (the GC and runtime are too heavy).
    **Browser-side code** (JavaScript owns the browser). **Data
    science notebooks** (Python's ecosystem is the product). **Sub-ms
    startup CLI filters** (Go's runtime initialization adds a few ms).
    These are domain mismatches, not language failings.
