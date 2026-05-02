# Chapter 43 — Revision Checkpoint

## Questions

1. Describe the key fields of `hchan` and explain what happens when a goroutine sends to a full buffered channel.
2. What are the two panic conditions for channels and how do you avoid each?
3. What does a receive from a closed, drained channel return, and how do you use that in a `for` loop?
4. What is a nil channel and how is it useful inside a `select` statement?
5. What is the difference between a one-time signal (`close`) and a repeated signal (sending values) and when do you use each?

## Answers

1. `hchan` contains a circular ring buffer (`buf`), two wait queues (`sendq` for blocked senders, `recvq` for blocked receivers), the current item count, the capacity, a `closed` flag, and an internal mutex. When a goroutine sends to a full buffered channel: (1) the runtime locks `hchan`, (2) since `qcount == dataqsiz`, the goroutine cannot deposit its value, (3) the goroutine wraps itself in a `sudog` struct containing a pointer to its value and appends itself to `sendq`, (4) the goroutine is parked (removed from the P's run queue). When a receiver later drains one slot, it picks the first `sudog` from `sendq`, copies its value into the freed buffer slot, and unparks the waiting sender.

2. The two panic conditions are: (1) **sending to a closed channel** — guard with a done-channel pattern so producers stop before the channel is closed, or use a `sync.Once` to close exactly once; (2) **closing an already-closed channel** — avoid by designating exactly one goroutine (the sole producer or a coordinator) as the one responsible for closing. Never close from multiple goroutines without a Once. Multiple senders must coordinate through a `sync.WaitGroup` + a closing goroutine rather than each calling `close`.

3. A receive from a closed, empty (fully drained) channel returns the zero value for the element type and `false` for the `ok` boolean: `v, ok := <-ch // ok == false`. In a `for range ch` loop, the range automatically checks `ok` and exits when the channel is closed and drained — no manual `ok` check is needed. This is the idiomatic way to consume all items from a channel that a producer will eventually close.

4. A nil channel (declared but not initialised, or explicitly set to `nil`) blocks forever on both send and receive. Inside a `select`, a case whose channel is nil is never selected — it is effectively disabled. This is useful for conditionally disabling a select case without restructuring the statement: after consuming the first value from a channel, set it to nil to prevent that case from firing again in subsequent iterations of the select loop.

5. A **one-time signal** uses `close(ch)` — closing the channel unblocks all current and future receivers simultaneously and permanently. It is appropriate when signalling "this event has occurred, everyone who needs to know should know." Examples: cancellation (done channel), a start gun, a server-ready event. A **repeated signal** sends values through the channel one at a time, each going to exactly one receiver. Use it when you want to distribute discrete work items or results, or when the signal carries data. `close` is broadcast (one-to-many, one-time); value send is unicast (one-to-one, repeatable).
