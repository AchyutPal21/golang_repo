# Chapter 1 — Exercises

Three exercises. All are runnable Go programs. Solutions live in
`solutions/`; try yours first, peek only after a real attempt.

---

## Exercise 1.1 — Verify your install

**Goal.** Confirm your toolchain works end-to-end and you can read the
output of a Go program.

**Task.** Run the starter file (from the chapter folder):

```bash
go run ./exercises/01_verify
```

The program prints:

* the Go version it was compiled with
* the operating system and architecture
* the number of CPUs the runtime sees
* the time it took the program to start

If any of these look wrong (e.g. `GOOS=windows` on a Linux machine, or
`NumCPU=1` when you have a 16-core machine), your install is misconfigured.
Check `go env` to debug.

**Acceptance criteria.** The program runs without error and the values
match your machine.

---

## Exercise 1.2 — Three concurrent goroutines

**Goal.** Internalize the goroutine + channel + select pattern by
extending it from two workers to three.

**Task.** Copy `examples/02_concurrent_clock/main.go` to a new folder
`exercises/02_three_workers/main.go` and extend it so *three* goroutines
run concurrently:

* `clock` prints the wall-clock time every 200 ms.
* `counter` prints an incrementing integer every 500 ms.
* `tick` prints the literal word "tick!" every second.

All three must terminate cleanly when a single shared `done` channel is
closed, after 5 seconds. The main goroutine must wait long enough for
all three shutdown messages to print before it exits.

**Acceptance criteria.**

* All three workers visibly run concurrently in the output.
* Output ends with three "shutting down" lines (one per worker), then
  "bye".
* No goroutine leaks: there should be no work happening after main
  returns. (You can check this with the race detector: `go run -race`.)

**Stretch.** Use `sync.WaitGroup` to make `main` wait for the workers
deterministically instead of `time.Sleep`. (You'll meet `WaitGroup`
formally in Chapter 45; if you're impatient, `go doc sync.WaitGroup`.)

---

## Exercise 1.3 ★ — Add an `/uptime` route

**Goal.** Touch the standard library's `encoding/json` package, the
HTTP server's request lifecycle, and Go's idiom for returning JSON.

**Task.** Copy `examples/03_http_server/main.go` to a new folder
`exercises/03_uptime_server/main.go` and add a route `GET /uptime` that
returns a JSON object:

```json
{"uptime":"1m23.456s"}
```

Constraints:

* Use `encoding/json` to marshal — do not build the JSON string by
  hand. (Hand-built JSON is the source of half of all CVE-class bugs in
  hand-rolled servers.)
* Set the `Content-Type` header to `application/json`.
* The duration string format is `time.Duration.String()` — `1m23.456s`,
  not `0:01:23.456`.
* The package-level `start` time variable should be the source of truth.

**Acceptance criteria.**

```bash
$ curl -s http://localhost:8080/uptime
{"uptime":"4.221s"}
```

The response must validate as JSON (`echo $output | jq .` should not
error).

**Stretch.** Add an `/info` route that returns:

```json
{"go_version":"go1.22.5","goos":"linux","goarch":"amd64","start":"2026-04-25T..."}
```

Use `runtime.Version()`, `runtime.GOOS`, `runtime.GOARCH`, and a
`time.Time` formatted as RFC 3339 (`time.RFC3339`).
