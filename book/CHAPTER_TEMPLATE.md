# Chapter Template — The 23-Section Structure

> Every chapter in this book follows this structure. The structure is not a
> bureaucratic checklist; it is the shape of how a senior engineer thinks
> about a topic. Each section has a *purpose*. If a section has nothing to
> add for a particular chapter, write **"Not applicable for this chapter
> because …"** — never leave it blank, never delete it, and never paper over
> it with filler.
>
> This file is normative. Reviewers should reject any chapter that does not
> obey it.

---

## File and folder layout per chapter

```
book/partN_<part-name>/chapterNN_<topic>/
├── README.md                        # the chapter prose (this template)
├── examples/
│   ├── 01_<concept>/main.go         # runnable example #1
│   ├── 02_<concept>/main.go         # runnable example #2
│   └── ...
├── exercises.md                     # graded exercises
├── exercises/
│   ├── 01_<name>/main.go            # starter
│   └── ...
├── solutions/
│   ├── 01_<name>/main.go            # reference solution
│   └── ...
├── mini_project/                    # optional, only when warranted
│   └── ...
└── checkpoint.md                    # the revision checkpoint Q&A
```

* **One topic per example folder.** Each `examples/NN_<name>/main.go` is its
  own `package main` — runnable directly with
  `go run ./examples/NN_<name>` from the chapter folder.
* **Why per-folder, not per-file?** Multiple `package main` files in the
  same directory are not legal Go. The per-folder layout keeps each example
  individually runnable while keeping `go vet ./...` and `go build ./...`
  clean for the whole book.
* **Numbered prefixes.** `01_`, `02_`, … so reading order is unambiguous.
* **`README.md` is the book chapter.** The Go files are *the artifacts the
  chapter refers to*, not where the teaching lives.
* **Cross-references** use markdown relative links so the rendered PDF
  remains navigable: `[Chapter 18 — Slices](../chapter18_slices/README.md)`.

---

## The 23 sections

Section headings in the chapter README must use these exact names and order.
Levels are `##` for the section heading and `###` for sub-headings within.

### 1. Concept Introduction
Purpose: state the topic in one paragraph as if to a smart friend who has
never heard of it. No jargon yet. End with a one-line working definition.

### 2. Why This Exists
Purpose: what would the world look like without this feature? What pain
prompted it? Make the reader *feel* the absence so the presence makes sense.

### 3. Problem It Solves
Purpose: concrete, named problems. List 3–6. Each gets one sentence.

### 4. Historical Context
Purpose: a short narrative — who built this, when, in response to what.
Include language-design citations (mailing list threads, design docs, papers)
where they exist. Skip if genuinely irrelevant; do not invent history.

### 5. How Industry Uses It
Purpose: name companies and systems where this feature/pattern shows up in
production. Be specific (e.g., "Cloudflare's HTTP/3 stack uses…"); avoid
vague hand-wavers ("big tech uses this").

### 6. Real-World Production Use Cases
Purpose: 3–5 *scenarios* (not companies). E.g., "A payment service uses X to
guarantee Y under condition Z." Each scenario is a paragraph.

### 7. Beginner-Friendly Explanation
Purpose: explain it again, this time with an analogy and the simplest
possible code example. Aim for "smart 14-year-old" comprehensible.

### 8. Deep Technical Explanation
Purpose: full precision. Specification-level. This is the section that takes
the longest to write and is the reason the book is worth buying.

### 9. Internal Working (How Go Handles It)
Purpose: under the hood. Reference the runtime source where relevant
(`runtime/chan.go`, `runtime/proc.go`). Diagrams in ASCII when they help.

### 10. Syntax Breakdown
Purpose: every form of the syntax, annotated. Do not assume the reader
remembers the surface syntax from the previous section.

### 11. Multiple Practical Examples
Purpose: at least three examples of escalating realism. Toy → realistic →
production-shaped. Each example is a separate `.go` file referenced by
filename.

### 12. Good vs Bad Examples
Purpose: a side-by-side table or two code blocks: the idiomatic form and the
common-bad form. Explain *why* the bad form is bad — beyond style.

### 13. Common Mistakes
Purpose: enumerate. Each mistake gets a name, a description, the failure
mode, and the fix. Aim for 5–10 per chapter.

### 14. Debugging Tips
Purpose: how do you know this feature is going wrong? Symptoms, tools, log
patterns, profile shapes.

### 15. Performance Considerations
Purpose: cost model. Allocations, syscalls, lock contention, cache
behaviour. Numbers when possible (microbenchmarks, big-O).

### 16. Security Considerations
Purpose: where this feature can be turned into a vulnerability. Skip with
explicit "Not applicable" only when it is truly N/A — most features have a
security angle when used wrong.

### 17. Senior Engineer Best Practices
Purpose: the rules a senior engineer would write in a code-review comment.
Include rationale. Avoid "always" and "never" without reasons.

### 18. Interview Questions
Purpose: 5–10 questions ranging from screen-level to staff-level. Tag each
with its level (junior / mid / senior / staff).

### 19. Interview Answers
Purpose: model answers. Not "what to memorize" but "how a strong engineer
would answer in real time," including what they would clarify first and
what tradeoffs they would name.

### 20. Hands-On Exercises
Purpose: 3–7 exercises with starter code in `exercises/`. Each exercise
states its goal, constraints, and acceptance criteria. Solutions live in a
separate `solutions/` folder, gitignored from print but in the repo.

### 21. Mini Project Tasks
Purpose: a single small project that integrates the chapter's lesson with
prior chapters' material. Optional only for chapters where it would be
contrived; if omitted, write a one-line justification.

### 22. Chapter Summary
Purpose: a half-page recap. Bullets are fine. Include the one-line working
definition from Section 1, evolved by everything since.

### 23. Advanced Follow-up Concepts
Purpose: pointers to deeper material — later chapters, papers, talks, blog
posts. The "if you want more, here's where to go" closing.

---

## Required callouts (use as needed, not on every page)

These are inline blockquote conventions. Use markdown blockquote syntax.

```markdown
> **Senior Architect Note —** Short opinionated take on a tradeoff.

> **Coming From Java —** What translates and what changes for Java devs.

> **Coming From Python —** Same, for Python.

> **Coming From JavaScript —** Same, for JS/TS.

> **Coming From C++ —** Same, for C++.

> **Coming From Rust —** Same, for Rust.

> **FAANG-level Implementation —** How this looks at scale at a top-tier
> shop. Be honest — don't fabricate "Google does X" claims unless you can
> cite them.

> **Startup vs Enterprise —** When the small-company answer differs from
> the big-company answer.

> **What Juniors Do Wrong —** A specific mistake with a specific fix.

> **Production Incident —** A short war story of a real failure mode this
> feature can cause. Mark fictional ones as composite.

> **Architecture Review —** A code or design choice and what a reviewer
> would push back on.
```

---

## Code-file conventions

Every `.go` file in a chapter folder must follow this template:

```go
// FILE: book/partX_<part>/chapterNN_<topic>/NN_<file>.go
// CHAPTER: NN — <Chapter Title>
// TOPIC: <one-line topic>
//
// Run: go run book/partX_<part>/chapterNN_<topic>/NN_<file>.go
//
// ─────────────────────────────────────────────────────────────────────────────
// WHY THIS FILE EXISTS:
//   <2–4 lines on what this file demonstrates and why it earns its place>
// ─────────────────────────────────────────────────────────────────────────────

package main

// ─── 1. <subsection> ────────────────────────────────────────────────────────
//
// <prose explanation, using line comments, not block comments>

// <runnable demonstration>

// ─── 2. <subsection> ────────────────────────────────────────────────────────
// ...

func main() {
    // <demonstration code>
}
```

Conventions:

* **Heavy comments.** Comments are 60–70% of every file. The Go is the
  *demonstration*; the prose is the lesson.
* **`package main` + `func main`.** Every file in a chapter folder is
  individually runnable, except for shared helpers in `internal/` packages.
* **No external dependencies in Parts I–IV.** Only `std`. Parts V–VII may
  use third-party libraries but each is justified in the chapter README.
* **Errors are checked.** No `_ = err`. No silently-ignored returns. The
  book teaches good habits by example.
* **`//nolint` is forbidden.** If `golangci-lint` complains, fix the code
  or explain in a comment why the lint is wrong.

---

## Length targets

| Element | Target | Hard floor | Hard ceiling |
| --- | --- | --- | --- |
| Chapter README total | 5,000–9,000 words | 3,500 words | 14,000 words |
| Code per chapter | 300–1,500 lines | 150 lines | 3,000 lines |
| Sections 1–6 combined | 600–1,200 words | 400 | 2,000 |
| Section 8 (Deep Technical) | 1,500–3,000 words | 1,000 | 5,000 |
| Section 9 (Internal Working) | 600–1,500 words | 300 | 3,000 |
| Section 11 (Examples) | 800–2,500 words across 3+ files | 400 | n/a |
| Section 22 (Summary) | 200–400 words | 150 | 600 |

Floors and ceilings are guardrails; the targets are where most chapters
should land. Foundational chapters (Part I) trend shorter; concurrency and
production chapters trend longer.

---

## Voice and style

* Second person ("you"), present tense, active voice.
* Opinionated. State the recommendation, then the alternatives, then the
  reasoning. Wishy-washy "it depends" without an opinion is a failure.
* American English spelling. Oxford comma.
* Code identifiers in `monospace`. File paths in `monospace`.
* No emojis. No exclamation marks (one per chapter, max). No "Let's…".
* Footnotes are fine for citations, kept short.
* When you cite a paper or talk, include enough that the reader can find it
  without a working URL: author, title, venue, year.

---

## Quality gates before a chapter is "done"

A chapter is not done until all of the following pass:

1. `go run` succeeds for every numbered file in the chapter folder.
2. `go vet ./...` is clean for the chapter module.
3. `golangci-lint run` is clean (project-default config).
4. `go test ./...` passes if the chapter has tests (Part VI onward, always).
5. The README contains all 23 sections with no placeholder text.
6. The README has at least one of each required callout where relevant.
7. The exercises have starter files and the solutions in `solutions/`
   actually solve them and pass any `go test` they ship with.
8. Cross-references resolve (no dead relative links).
9. The chapter's reading time is reported at the top of the README, computed
   at 250 words/minute.
10. The chapter has been read end-to-end out loud at least once.

This is the bar. Anything less is a draft, not a chapter.
