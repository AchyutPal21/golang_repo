# The Complete Golang Engineering Bible

> A self-taught path from "I have never written code" to "I architect
> production Go systems at scale." Written in book style, organized as a
> runnable repository, designed to be exported to a premium PDF.

---

## What this is

This is a book you can run. Every chapter is a folder with a long-form
`README.md` (the prose), runnable example programs (`examples/*`),
graded exercises (`exercises/*`), reference solutions (`solutions/*`),
and a self-test checkpoint (`checkpoint.md`). Open the `README.md` and
read; type the example commands; do the exercises; check yourself.
That's the loop, ninety-times-and-some.

The book is structured into **seven parts**, **102 numbered chapters**,
and **ten capstone projects**. The full table of contents lives in
[`BOOK.md`](BOOK.md). The chapter authoring contract lives in
[`CHAPTER_TEMPLATE.md`](CHAPTER_TEMPLATE.md). The build plan and per-
chapter status lives in [`BUILD_MANIFEST.md`](BUILD_MANIFEST.md).

---

## Status

This is a long-term build. Chapters move from `▢ planned` →
`◐ drafting` → `◑ review` → `■ done`. One chapter is marked
`★ canonical` — that's the reference exemplar against which every
other chapter is measured.

| Section | Status |
| --- | --- |
| `BOOK.md` (full TOC) | ■ done |
| `CHAPTER_TEMPLATE.md` (the 23-section authoring contract) | ■ done |
| `BUILD_MANIFEST.md` (chapter status & source-asset map) | ■ done |
| Chapter 1 — Why Go Exists | ★ canonical |
| Chapters 2–102 | ▢ planned |
| Capstone projects A–J | ▢ planned |

For per-chapter status, see [`BUILD_MANIFEST.md`](BUILD_MANIFEST.md).
Existing material under
[`../golang-mastery-updated/`](../golang-mastery-updated/) is the
source asset for many chapters; nothing there is deleted until the
matching new chapter passes the quality gates in
`CHAPTER_TEMPLATE.md`.

---

## How to read

Three legitimate paths:

1. **Linear** — every chapter, in order, every exercise. Zero-to-senior
   in 4–6 months of evening study.
2. **Bridge** — skim Part I, read the "Coming From X" callouts, go deep
   on Parts III–VI. For engineers from Java/Python/JS/C++/Rust. 4–6
   weeks.
3. **Interview** — concept intro, common mistakes, senior best
   practices, interview Q&A, summary in every chapter. Skip exercises.
   2–3 weeks.

Every chapter ends with a **revision checkpoint**. If you can't answer
the questions without looking, re-read the chapter.

---

## How to run any code

The book is a single Go module rooted at this directory:

```bash
cd book
```

Each chapter's runnable programs live under
`partN_<name>/chapterNN_<topic>/examples/NN_<name>/main.go`. From the
chapter folder you can always run the next example with:

```bash
go run ./examples/NN_<name>
```

Or from the book root:

```bash
go run ./part1_foundations/chapter01_why_go_exists/examples/01_hello
```

To verify the whole book builds clean:

```bash
go build ./...
go vet ./...
```

---

## Repository layout

```
book/
├── README.md                        # this file
├── BOOK.md                          # the full table of contents
├── CHAPTER_TEMPLATE.md              # the 23-section authoring contract
├── BUILD_MANIFEST.md                # chapter status + source-asset map
├── go.mod
├── part1_foundations/
│   ├── chapter01_why_go_exists/     # ★ canonical
│   │   ├── README.md
│   │   ├── examples/
│   │   │   ├── 01_hello/main.go
│   │   │   ├── 02_concurrent_clock/main.go
│   │   │   └── 03_http_server/main.go
│   │   ├── exercises.md
│   │   ├── exercises/
│   │   │   └── 01_verify/main.go
│   │   └── checkpoint.md
│   ├── chapter02_ecosystem_map/     # planned
│   └── …
├── part2_core_language/             # planned
├── part3_designing_software/        # planned
├── part4_concurrency_systems/       # planned
├── part5_building_backends/         # planned
├── part6_production_engineering/    # planned
└── part7_capstones/                 # planned
```

The seven parts mirror the curriculum spec's thirteen sections, grouped
by pedagogical theme (see `BOOK.md`).

---

## Voice and depth

The book is written in second-person present tense, opinionated, and
deeply technical. Each chapter follows the 23-section structure laid
out in `CHAPTER_TEMPLATE.md`:

```
1.  Concept Introduction
2.  Why This Exists
3.  Problem It Solves
4.  Historical Context
5.  How Industry Uses It
6.  Real-World Production Use Cases
7.  Beginner-Friendly Explanation
8.  Deep Technical Explanation
9.  Internal Working (How Go Handles It)
10. Syntax Breakdown
11. Multiple Practical Examples
12. Good vs Bad Examples
13. Common Mistakes
14. Debugging Tips
15. Performance Considerations
16. Security Considerations
17. Senior Engineer Best Practices
18. Interview Questions
19. Interview Answers
20. Hands-On Exercises
21. Mini Project Tasks
22. Chapter Summary
23. Advanced Follow-up Concepts
```

Plus required callouts where relevant: *Senior Architect Note*,
*Coming From Java/Python/JavaScript/C++/Rust*, *FAANG-level
Implementation*, *Startup vs Enterprise*, *What Juniors Do Wrong*,
*Production Incident*, *Architecture Review*.

The reference exemplar for all of this is
[`part1_foundations/chapter01_why_go_exists/README.md`](part1_foundations/chapter01_why_go_exists/README.md).
Read it. Anything new contributed to the book must match its depth and
voice.

---

## Quality gates per chapter

A chapter is not "done" until all ten of these pass — see
`CHAPTER_TEMPLATE.md` for the full contract:

1. `go run` succeeds for every example folder.
2. `go vet ./...` is clean.
3. `golangci-lint run` is clean (project-default config).
4. `go test ./...` passes for any chapter with tests.
5. The README contains all 23 sections, no placeholders.
6. The README has the appropriate callouts where relevant.
7. Exercises have starter files and reference solutions that pass any
   tests they ship with.
8. Cross-references resolve (no dead relative links).
9. The chapter's reading time is reported at the top of the README.
10. The chapter has been read end-to-end out loud at least once.

---

## What's next

The next chapters to land, in order, are 2 (Ecosystem map), 3 (Install
& setup), 4 (Workspace), 5 (`go run`/`build`/`install`), 6 (Coming
from another language), 7 (First real CLI). After Part I lands, Part
II (chapters 8–25) walks the core language at the same depth.

See [`BUILD_MANIFEST.md`](BUILD_MANIFEST.md) for the full plan and
status table.

---

## License

To be set. The intent is dual-licensed: source code under the MIT
License, prose under Creative Commons BY-NC 4.0 (so the book can be
sold as a PDF without losing the open source code samples).
