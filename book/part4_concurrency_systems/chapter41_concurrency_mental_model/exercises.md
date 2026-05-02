# Chapter 41 — Exercises

## 41.1 — Three-way counter

Run [`exercises/01_concurrent_counter`](exercises/01_concurrent_counter/main.go).

`MutexCounter`, `ActorCounter`, and `AtomicCounter` all satisfy a `Counter` interface and produce identical results under concurrent load.

Try:
- Add a fourth implementation `ChannelCounter` that uses a dedicated goroutine and a `chan int` to receive increments (no mutex, no atomic). Verify it gives the same result.
- Add a `Reset()` method to all three and call it mid-run — confirm the counter continues correctly from zero.
- Run with `-race` (`go run -race ./exercises/01_concurrent_counter`) and confirm no race is reported for any implementation.

## 41.2 ★ — Pipeline word count

Build a three-stage pipeline:
1. `lines(r io.Reader) <-chan string` — emit lines
2. `words(lines <-chan string) <-chan string` — split each line into words
3. `count(words <-chan string) map[string]int` — accumulate frequencies

Wire them together and run on a `strings.NewReader` with 5+ lines of text. Verify the word counts are correct.

## 41.3 ★★ — Fan-out merge

Implement `fanOut(in <-chan int, n int) []<-chan int` that sends each value from `in` to all `n` output channels (broadcast), and `merge(channels ...<-chan int) <-chan int` that merges all channels into one output.

Use them to: generate 10 integers → broadcast to 3 workers that square the value → merge results. Confirm you receive 30 total values (each integer squared 3 times).
