# Chapter 41 — Revision Checkpoint

## Questions

1. What is the difference between concurrency and parallelism, and how does `GOMAXPROCS` relate to each?
2. State the CSP rule and explain why it simplifies reasoning about concurrent programs.
3. What is a happens-before relationship and why do channels provide one?
4. Why is an unbuffered channel a synchronisation point but a buffered channel is not (until full)?
5. What is a data race and how do you detect one in Go?

## Answers

1. Concurrency is a structural property: the program is composed of independent tasks that can overlap in time. Parallelism is an execution property: tasks literally execute simultaneously on separate CPUs. A program can be concurrent without being parallel (one CPU, time-sliced) or parallel without being concurrent (trivially vectorised loop). `GOMAXPROCS` sets the number of OS threads the Go runtime uses to run goroutines simultaneously — it controls the degree of parallelism. Setting `GOMAXPROCS=1` makes the program concurrent but not parallel; setting it to `runtime.NumCPU()` (the default since Go 1.5) allows full parallelism.

2. The CSP rule is: "don't communicate by sharing memory — share memory by communicating." Instead of multiple goroutines locking and unlocking access to a shared variable, one goroutine owns the variable exclusively and all others send messages to it through a channel. This simplifies reasoning because the owning goroutine processes messages sequentially — there are no concurrent writes to protect, no lock ordering problems, and no partial-update visibility issues. The code that mutates state is single-threaded even though the overall program is concurrent.

3. A happens-before relationship is a guarantee about memory visibility across goroutines: if operation A happens-before operation B, then everything written before A is guaranteed to be visible after B. The Go memory model specifies that a send on a channel happens-before the corresponding receive completes. This means: if goroutine 1 writes to a variable and then sends on a channel, and goroutine 2 receives from that channel, goroutine 2 is guaranteed to see the write. Without a happens-before guarantee, writes in one goroutine may not be visible to another goroutine due to CPU caches and compiler reordering.

4. An unbuffered channel has no capacity — a send blocks until a receiver is ready, and a receive blocks until a sender is ready. The two goroutines must rendezvous: neither proceeds past the channel operation until the other side is present. That rendezvous is the synchronisation point and establishes happens-before. A buffered channel has capacity N: a send only blocks when the buffer is full, and a receive only blocks when the buffer is empty. Below capacity, the sender and receiver never need to meet — the sender deposits a value and continues; the receiver picks it up later. No rendezvous, no happens-before (unless the buffer fills). Buffered channels trade synchronisation for throughput.

5. A data race occurs when two goroutines access the same memory location concurrently, at least one of the accesses is a write, and there is no synchronisation (happens-before relationship) ordering them. Data races are undefined behaviour: the program may produce wrong results, crash with a confusing error, or appear to work on one machine but fail on another. Detect races with Go's built-in race detector: run `go run -race` or `go test -race`. The detector instruments memory accesses at compile time and reports races at runtime with a full stack trace for both the concurrent accesses. Run it in CI against every test suite.
