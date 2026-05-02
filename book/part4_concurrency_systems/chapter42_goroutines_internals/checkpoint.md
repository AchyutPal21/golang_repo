# Chapter 42 — Revision Checkpoint

## Questions

1. Describe the G/M/P model in one sentence each for G, M, and P. What does `GOMAXPROCS` control?
2. Why does a blocking syscall not stall all other goroutines, and what does the runtime do to prevent it?
3. What is work stealing and why does it matter for throughput?
4. What changed about goroutine preemption in Go 1.14, and what problem did it solve?
5. What is the loop-variable capture bug, how did Go 1.22 fix it, and what is the workaround for older versions?

## Answers

1. **G** (Goroutine) is the unit of work: a struct containing the goroutine's state, program counter, and a dynamically-sized stack. **M** (Machine) is an OS thread — it executes Go code and must hold a P to do so. **P** (Processor) is a scheduling context that holds a local run queue of Gs waiting to be executed. `GOMAXPROCS` sets the number of Ps, which equals the maximum number of Goroutines that can execute Go code simultaneously (the degree of parallelism).

2. When a goroutine makes a blocking syscall (e.g., reading from a file or sleeping), the M executing it blocks inside the kernel. The Go runtime detects this and detaches M from its P. Another idle M (or a newly created one) picks up that P and continues running other goroutines on it. When the blocked syscall returns, the original M tries to reclaim a P — either the one it released or another idle one. If no P is available, the goroutine is placed on the global run queue until a P frees up. This ensures that one blocking M never starves the other `GOMAXPROCS-1` Ps.

3. Work stealing is a load-balancing mechanism: when a P's local run queue is empty, it steals half the goroutines from another P's run queue. This keeps all Ps busy without requiring a global lock on a shared queue. Without work stealing, a P that finishes its local work would sit idle while other Ps have a backlog — wasting CPU. Work stealing achieves near-linear scaling with GOMAXPROCS for workloads with many goroutines.

4. Before Go 1.14, the Go scheduler was cooperative with respect to CPU-bound code: a goroutine would only yield at function calls, channel operations, or explicit `runtime.Gosched()` calls. A tight loop with no function calls (e.g., a busy CPU computation) could hold a P indefinitely, starving other goroutines on that P. Go 1.14 introduced asynchronous preemption: the runtime sends a signal to the OS thread running the goroutine, interrupting it at a safe point (identified by metadata from the compiler). This ensures no goroutine can monopolise a P regardless of what code it executes.

5. In Go < 1.22, the `for` loop uses a single variable address for the loop counter across all iterations. A goroutine that captures that variable by reference (closure) reads whatever value the variable holds when the goroutine runs — which is typically the final value after the loop ends, because the goroutines are scheduled after the loop completes. Result: all goroutines print the same (final) value. Go 1.22 fixed this by giving each loop iteration its own copy of the loop variable — closures now capture distinct addresses. The pre-1.22 workaround is to pass the loop variable as an argument to the goroutine function: `go func(n int) { ... }(i)`, which creates a copy bound to that invocation.
