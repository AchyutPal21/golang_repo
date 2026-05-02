# Chapter 44 — Revision Checkpoint

## Questions

1. When multiple `select` cases are ready simultaneously, which one runs?
2. What does the `default` case do in a `select`, and when should you avoid it?
3. Why does `time.After` leak memory in a loop and what is the correct alternative?
4. What is the safe reset procedure for `time.NewTimer`, and why is each step necessary?
5. How do you implement an overall deadline across multiple sequential operations using `select`?

## Answers

1. When multiple cases are ready at the same time, Go's runtime chooses one **uniformly at random**. The order in which cases are written in the source file has no effect on which one is selected. This is by design — it prevents subtle priority bugs where a case listed first would always win and starve cases listed later. If you need priority, implement it explicitly with a nested select and a `default` (see section 44.3).

2. The `default` case runs immediately if no other case in the `select` is ready, making the statement non-blocking. It is useful for try-send/try-receive operations where you want to proceed rather than block. Avoid `default` when your intent is to wait for the channel: adding `default` to a blocking select turns it into a spin loop that burns CPU, because the `default` will fire on every iteration until a channel is ready.

3. `time.After(d)` creates a new `time.Timer` internally and returns its channel. That `Timer` exists in the Go runtime's timer heap until it fires — it cannot be garbage collected earlier even if you no longer hold a reference to the channel. In a loop that runs frequently (e.g., a polling loop or a request handler), each iteration creates a new `Timer`, and all of them accumulate in the heap until they fire, consuming memory and scheduler overhead. The correct alternative is `time.NewTimer` created once outside the loop, then safely reset for each iteration with Stop + drain + Reset.

4. The safe reset procedure is: (1) call `timer.Stop()` to prevent a concurrent fire, (2) drain the channel with a non-blocking receive (`select { case <-timer.C: default: }`) to remove any value that was already sent before `Stop` took effect (there is a race between Stop and a send that was already in flight), (3) call `timer.Reset(d)` to arm the timer for the next duration. Skipping step 2 can leave a stale value in the channel that fires immediately the next time the timer is selected, causing a spurious timeout.

5. Create the deadline channel **once** before the loop with `deadline := time.After(overallTimeout)`, then include `case <-deadline` in every `select` inside the loop. Because channels are values, the same channel reference is checked on every iteration. Once `overallTimeout` elapses the channel delivers one value, and whichever iteration's `select` happens to be active at that moment selects the deadline case and can abort. All subsequent iterations also see the deadline case as permanently ready (the value stays in the buffered channel), so they can also abort promptly.
