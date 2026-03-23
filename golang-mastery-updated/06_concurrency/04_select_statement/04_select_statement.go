// 04_select_statement.go
//
// SELECT STATEMENT — multiplexing channel operations
//
// "select" is to channels what "switch" is to values. It waits on multiple
// channel operations simultaneously and proceeds with whichever one is ready.
//
// WHY SELECT EXISTS
// -----------------
// Without select, if you need to wait on multiple channels you'd have to
// pick one to block on first — and if the other becomes ready first, you miss it.
// select solves this: it blocks until ANY of the listed channel operations
// can proceed, then executes that case.
//
// SELECT MECHANICS (IMPORTANT)
// ----------------------------
// 1. All channel expressions in a select are EVALUATED once before any case is
//    considered (left-to-right, top-to-bottom). This happens atomically.
// 2. If multiple cases are ready SIMULTANEOUSLY, Go picks one UNIFORMLY AT RANDOM.
//    This is guaranteed by the spec — not implementation-defined.
// 3. If no case is ready and there's a default clause, default executes immediately.
// 4. If no case is ready and no default: the goroutine blocks until one is ready.
// 5. A nil channel case never becomes ready — useful for disabling cases.
//
// SYNTAX:
//   select {
//   case v := <-ch1:      // receive from ch1
//       ...
//   case ch2 <- v:        // send to ch2
//       ...
//   case v, ok := <-ch3: // receive with close detection
//       ...
//   default:              // optional: execute if no case ready
//       ...
//   }

package main

import (
	"fmt"
	"time"
)

// =============================================================================
// SECTION 1: Basic select — waiting on multiple channels
// =============================================================================

func demoBasicSelect() {
	fmt.Println("=== Basic Select ===")

	ch1 := make(chan string, 1)
	ch2 := make(chan string, 1)

	// Send to ch2 with a delay, ch1 immediately
	go func() {
		time.Sleep(20 * time.Millisecond)
		ch2 <- "from channel 2"
	}()
	go func() {
		ch1 <- "from channel 1"
	}()

	time.Sleep(5 * time.Millisecond) // let goroutines start

	// Select blocks until one channel is ready.
	// Since ch1 sends immediately, ch1's case should fire first.
	// (But if both are ready simultaneously, selection is random.)
	for i := 0; i < 2; i++ {
		select {
		case msg1 := <-ch1:
			fmt.Printf("  received: %q\n", msg1)
		case msg2 := <-ch2:
			fmt.Printf("  received: %q\n", msg2)
		}
	}
	fmt.Println()
}

// =============================================================================
// SECTION 2: select with default — non-blocking channel operations
// =============================================================================
//
// Adding a "default" case makes the select non-blocking:
// if no channel is ready, default executes immediately.
//
// Use cases:
//   - Try to receive without blocking (poll a channel)
//   - Try to send without blocking (fire-and-forget if buffer full)
//   - Non-blocking check of a done/cancel channel

func demoSelectDefault() {
	fmt.Println("=== Select with Default (non-blocking) ===")

	ch := make(chan int, 3)

	// Non-blocking receive: if ch is empty, default fires
	select {
	case val := <-ch:
		fmt.Printf("  received: %d\n", val)
	default:
		fmt.Println("  channel empty, doing other work (default)")
	}

	// Put something in the channel
	ch <- 42

	// Now the receive case fires
	select {
	case val := <-ch:
		fmt.Printf("  received: %d\n", val)
	default:
		fmt.Println("  default (won't print this time)")
	}

	// Non-blocking send: if buffer full, default fires
	ch <- 1
	ch <- 2
	ch <- 3 // buffer now full (cap=3)
	select {
	case ch <- 99:
		fmt.Println("  sent 99")
	default:
		fmt.Println("  buffer full, could not send (default)")
	}

	// Non-blocking done check — common pattern in loops:
	done := make(chan struct{})
	go func() {
		time.Sleep(50 * time.Millisecond)
		close(done)
	}()

	fmt.Println("  polling for cancellation (non-blocking):")
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			fmt.Println("  cancelled! stopping loop")
			goto doneLabel
		default:
			fmt.Printf("  iteration %d: not cancelled yet\n", i)
			time.Sleep(10 * time.Millisecond)
		}
	}
doneLabel:
	fmt.Println()
}

// =============================================================================
// SECTION 3: select with timeout
// =============================================================================
//
// time.After(d) returns a <-chan time.Time that receives after duration d.
// Use it in a select case to impose a deadline on a channel operation.
//
// This is the single most common select pattern in real Go code.

func fetchData(delay time.Duration) <-chan string {
	ch := make(chan string, 1)
	go func() {
		time.Sleep(delay)
		ch <- "data from server"
	}()
	return ch
}

func demoSelectTimeout() {
	fmt.Println("=== Select with Timeout ===")

	// Fast response — completes before timeout
	data := fetchData(30 * time.Millisecond)
	select {
	case result := <-data:
		fmt.Printf("  fast fetch: got %q\n", result)
	case <-time.After(100 * time.Millisecond):
		fmt.Println("  fast fetch: timed out")
	}

	// Slow response — timeout fires first
	data = fetchData(200 * time.Millisecond)
	select {
	case result := <-data:
		fmt.Printf("  slow fetch: got %q\n", result)
	case <-time.After(50 * time.Millisecond):
		fmt.Println("  slow fetch: timed out (expected)")
	}

	fmt.Println()
}

// =============================================================================
// SECTION 4: Priority select — checking one channel before others
// =============================================================================
//
// Go's select chooses randomly among ready cases. But sometimes you want
// to give priority to one channel (e.g., always check done/cancel first).
//
// The pattern: use a nested select or check with default to implement priority.

func demoPrioritySelect() {
	fmt.Println("=== Priority Select ===")

	highPriority := make(chan string, 10)
	lowPriority := make(chan string, 10)
	done := make(chan struct{})

	// Fill both channels
	for i := 0; i < 3; i++ {
		highPriority <- fmt.Sprintf("HIGH-%d", i)
		lowPriority <- fmt.Sprintf("low-%d", i)
	}
	close(done)

	// Pattern: try high-priority channel first (with default), then regular select.
	// This gives high-priority channel preference when both are ready.
	drain := func() {
		for {
			// STEP 1: Non-blocking check of high-priority first
			select {
			case msg := <-highPriority:
				fmt.Printf("  [priority] %s\n", msg)
				continue // loop back and check high-priority again
			default:
				// high-priority is empty, fall through
			}

			// STEP 2: Normal select for other cases
			select {
			case msg := <-highPriority: // also check here in case of race
				fmt.Printf("  [priority] %s\n", msg)
			case msg := <-lowPriority:
				fmt.Printf("  [low] %s\n", msg)
			case <-done:
				fmt.Println("  done signal (but draining first)")
				// After receiving done, drain remaining high-priority
				for {
					select {
					case msg := <-highPriority:
						fmt.Printf("  [priority-drain] %s\n", msg)
					default:
						fmt.Println("  high-priority drained, exiting")
						return
					}
				}
			}
		}
	}

	// Reset done for cleaner demo
	done2 := make(chan struct{})
	close(done2)

	// Simple demo: both channels ready, check high first
	hi := make(chan int, 5)
	lo := make(chan int, 5)
	for i := 0; i < 5; i++ {
		hi <- i
		lo <- i * 10
	}

	fmt.Println("  Priority-ordered drain (high before low):")
	for i := 0; i < 10; i++ {
		select {
		case v := <-hi:
			fmt.Printf("  HIGH: %d\n", v)
		default:
			select {
			case v := <-lo:
				fmt.Printf("  low:  %d\n", v)
			default:
				goto done3
			}
		}
	}
done3:
	_ = drain
	_ = done2
	fmt.Println()
}

// =============================================================================
// SECTION 5: select on nil channel — disabling a case
// =============================================================================
//
// A nil channel in a select case NEVER becomes ready.
// The Go spec guarantees this. It effectively disables that case.
//
// This is a powerful pattern: conditionally include/exclude a channel from
// a select by setting it to nil when you don't want it active.
//
// Use case: merge two streams, but one stream is "paused" or "done".

func demoNilChannelSelect() {
	fmt.Println("=== Nil Channel in Select (disabling a case) ===")

	ch1 := make(chan int, 3)
	ch2 := make(chan int, 3)

	ch1 <- 1
	ch1 <- 2
	ch1 <- 3
	ch2 <- 10
	ch2 <- 20
	ch2 <- 30

	// Merge ch1 and ch2. When one is exhausted, set it to nil
	// so the select no longer waits on it.
	var a, b chan int = ch1, ch2 // local vars we can nil out

	for a != nil || b != nil {
		select {
		case v, ok := <-a:
			if !ok {
				fmt.Println("  ch1 closed, disabling case a")
				a = nil // nil channel case never fires — effectively removed
				continue
			}
			fmt.Printf("  from ch1: %d\n", v)
			if len(ch1) == 0 {
				a = nil // drain exhausted — stop selecting on it
			}
		case v, ok := <-b:
			if !ok {
				fmt.Println("  ch2 closed, disabling case b")
				b = nil
				continue
			}
			fmt.Printf("  from ch2: %d\n", v)
			if len(ch2) == 0 {
				b = nil
			}
		}
	}
	fmt.Println("  both channels exhausted")
	fmt.Println()
}

// =============================================================================
// SECTION 6: for-select loop — the fundamental concurrency loop
// =============================================================================
//
// The for-select loop is the idiomatic way to write a goroutine that:
//   - Continuously processes events from channels
//   - Can be cancelled via a done channel
//   - Handles multiple event types
//
// This pattern appears in almost every real-world concurrent Go program.
// It's the foundation of actors, event loops, state machines, etc.

type Event struct {
	Type string
	Data interface{}
}

// eventLoop processes events until cancelled.
// It demonstrates the canonical for-select pattern.
func eventLoop(events <-chan Event, done <-chan struct{}) {
	fmt.Println("  eventLoop started")
	for {
		select {
		case evt, ok := <-events:
			if !ok {
				// Channel closed: producer is done sending
				fmt.Println("  eventLoop: events channel closed, exiting")
				return
			}
			// Process the event
			switch evt.Type {
			case "message":
				fmt.Printf("  eventLoop: message: %v\n", evt.Data)
			case "command":
				fmt.Printf("  eventLoop: command: %v\n", evt.Data)
			default:
				fmt.Printf("  eventLoop: unknown event: %v\n", evt.Type)
			}

		case <-done:
			// External cancellation signal
			fmt.Println("  eventLoop: cancelled via done channel")
			return

		case <-time.After(500 * time.Millisecond):
			// Heartbeat / idle timeout
			// This creates a NEW time.After on every iteration — fine here,
			// but for production use time.NewTimer and reset it.
			fmt.Println("  eventLoop: idle timeout (no events for 500ms)")
			return
		}
	}
}

func demoForSelectLoop() {
	fmt.Println("=== For-Select Loop (canonical event loop pattern) ===")

	events := make(chan Event, 5)
	done := make(chan struct{})

	// Send some events
	go func() {
		events <- Event{"message", "hello world"}
		time.Sleep(10 * time.Millisecond)
		events <- Event{"command", "run task"}
		time.Sleep(10 * time.Millisecond)
		events <- Event{"message", "second message"}
		close(events) // signal end of stream
	}()

	eventLoop(events, done)

	fmt.Println()
	fmt.Println("  Demo 2: for-select with explicit cancel")

	events2 := make(chan Event, 10)
	done2 := make(chan struct{})

	// Send events indefinitely
	go func() {
		i := 0
		for {
			select {
			case <-done2:
				return
			default:
				events2 <- Event{"tick", i}
				i++
				time.Sleep(15 * time.Millisecond)
			}
		}
	}()

	// Cancel after 50ms
	go func() {
		time.Sleep(50 * time.Millisecond)
		close(done2)
	}()

	// for-select with cancel
	for {
		select {
		case evt := <-events2:
			fmt.Printf("  tick %v received\n", evt.Data)
		case <-done2:
			fmt.Println("  for-select loop cancelled")
			goto end
		}
	}
end:
	fmt.Println()
}

// =============================================================================
// SECTION 7: select — evaluates all channel expressions once
// =============================================================================
//
// An important subtlety: the channel expressions and values to send in each
// case are evaluated exactly once, before the select blocks.
//
// "case myChan() <- computeValue()": both myChan() and computeValue() are
// called once when the select is entered, regardless of which case runs.

func demoSelectEvaluation() {
	fmt.Println("=== Select Expression Evaluation Order ===")

	callCount := 0
	getChan := func() chan int {
		callCount++
		fmt.Printf("  getChan() called (count=%d)\n", callCount)
		ch := make(chan int, 1)
		return ch
	}

	getValue := func() int {
		fmt.Println("  getValue() called")
		return 42
	}

	// Both getChan() and getValue() are called once when select is entered,
	// even though we might not use that channel/value:
	select {
	case getChan() <- getValue():
		fmt.Println("  sent successfully")
	default:
		// If we hit default, getValue() was still called above
	}
	// Output shows both functions were called before determining which case runs.

	fmt.Println()
}

// =============================================================================
// MAIN
// =============================================================================

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════╗")
	fmt.Println("║         SELECT STATEMENT — Deep Dive                 ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	fmt.Println()

	demoBasicSelect()
	demoSelectDefault()
	demoSelectTimeout()
	demoPrioritySelect()
	demoNilChannelSelect()
	demoForSelectLoop()
	demoSelectEvaluation()

	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("KEY TAKEAWAYS:")
	fmt.Println("  1. select waits on multiple channel ops; picks one that's ready")
	fmt.Println("  2. Multiple ready cases: one is chosen UNIFORMLY AT RANDOM")
	fmt.Println("  3. default: makes select non-blocking (executes if none ready)")
	fmt.Println("  4. Timeout: use time.After in a case for deadline enforcement")
	fmt.Println("  5. Nil channel: never fires in select — use to disable a case")
	fmt.Println("  6. Priority: check preferred channel with nested select+default")
	fmt.Println("  7. for-select: canonical pattern for event-driven goroutines")
	fmt.Println("  8. Case expressions evaluated once on select entry (before block)")
	fmt.Println("═══════════════════════════════════════════════════════")
}
