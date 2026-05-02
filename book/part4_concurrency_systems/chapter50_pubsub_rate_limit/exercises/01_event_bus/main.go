// FILE: book/part4_concurrency_systems/chapter50_pubsub_rate_limit/exercises/01_event_bus/main.go
// CHAPTER: 50 — Pub/Sub, Rate Limit, Throttle
// EXERCISE: Event bus with typed events, rate-limited publishers, throttled
//           subscribers, wildcard topic matching, and dead-letter queue for
//           dropped messages.
//
// Run (from the chapter folder):
//   go run ./exercises/01_event_bus

package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// EVENT TYPES
// ─────────────────────────────────────────────────────────────────────────────

type Event struct {
	Topic     string
	Payload   any
	Timestamp time.Time
}

// ─────────────────────────────────────────────────────────────────────────────
// EVENT BUS — wildcard topic support (prefix match with "*")
// ─────────────────────────────────────────────────────────────────────────────

type Subscription struct {
	topic   string // exact or prefix ending in "*"
	ch      chan Event
	dropped atomic.Int64
}

type EventBus struct {
	mu       sync.RWMutex
	subs     []*Subscription
	dlq      chan Event // dead-letter queue
	closed   bool
}

func NewEventBus(dlqSize int) *EventBus {
	return &EventBus{dlq: make(chan Event, dlqSize)}
}

func (eb *EventBus) Subscribe(pattern string, bufSize int) (*Subscription, func()) {
	sub := &Subscription{
		topic: pattern,
		ch:    make(chan Event, bufSize),
	}

	eb.mu.Lock()
	eb.subs = append(eb.subs, sub)
	eb.mu.Unlock()

	cancel := func() {
		eb.mu.Lock()
		defer eb.mu.Unlock()
		for i, s := range eb.subs {
			if s == sub {
				eb.subs = append(eb.subs[:i], eb.subs[i+1:]...)
				close(sub.ch)
				return
			}
		}
	}
	return sub, cancel
}

func (eb *EventBus) matches(pattern, topic string) bool {
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(topic, pattern[:len(pattern)-1])
	}
	return pattern == topic
}

func (eb *EventBus) Publish(e Event) {
	e.Timestamp = time.Now()

	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if eb.closed {
		return
	}

	for _, sub := range eb.subs {
		if !eb.matches(sub.topic, e.Topic) {
			continue
		}
		select {
		case sub.ch <- e:
		default:
			sub.dropped.Add(1)
			// Route to dead-letter queue.
			select {
			case eb.dlq <- e:
			default:
				// DLQ also full — truly lost
			}
		}
	}
}

func (eb *EventBus) DLQ() <-chan Event {
	return eb.dlq
}

func (eb *EventBus) Close() {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	if eb.closed {
		return
	}
	eb.closed = true
	for _, sub := range eb.subs {
		close(sub.ch)
	}
	eb.subs = nil
	close(eb.dlq)
}

// ─────────────────────────────────────────────────────────────────────────────
// RATE-LIMITED PUBLISHER
// ─────────────────────────────────────────────────────────────────────────────

type RateLimitedPublisher struct {
	bus      *EventBus
	interval time.Duration
	last     time.Time
	mu       sync.Mutex
}

func NewRateLimitedPublisher(bus *EventBus, ratePerSec int) *RateLimitedPublisher {
	return &RateLimitedPublisher{
		bus:      bus,
		interval: time.Second / time.Duration(ratePerSec),
	}
}

// Publish sends the event only if the rate limit allows; returns false if throttled.
func (p *RateLimitedPublisher) Publish(e Event) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if time.Since(p.last) < p.interval {
		return false
	}
	p.last = time.Now()
	p.bus.Publish(e)
	return true
}

// ─────────────────────────────────────────────────────────────────────────────
// SCENARIO 1: exact topics and wildcard subscriptions
// ─────────────────────────────────────────────────────────────────────────────

func scenarioWildcard() {
	fmt.Println("=== Scenario 1: wildcard topic routing ===")

	bus := NewEventBus(20)

	// Exact subscription.
	orderSub, cancelOrder := bus.Subscribe("orders.created", 10)
	// Wildcard — matches any topic starting with "orders."
	allOrdersSub, cancelAll := bus.Subscribe("orders.*", 10)
	// Different topic.
	paymentSub, cancelPayment := bus.Subscribe("payments.confirmed", 10)

	var wg sync.WaitGroup
	collect := func(name string, sub *Subscription) []string {
		var msgs []string
		wg.Add(1)
		go func() {
			defer wg.Done()
			for e := range sub.ch {
				msgs = append(msgs, fmt.Sprintf("%s:%v", e.Topic, e.Payload))
			}
			fmt.Printf("  [%s] received: %v\n", name, msgs)
		}()
		return msgs
	}

	collect("orders.created", orderSub)
	collect("orders.*", allOrdersSub)
	collect("payments.confirmed", paymentSub)

	bus.Publish(Event{Topic: "orders.created", Payload: "order-1"})
	bus.Publish(Event{Topic: "orders.shipped", Payload: "order-1"})
	bus.Publish(Event{Topic: "payments.confirmed", Payload: "pay-1"})
	bus.Publish(Event{Topic: "inventory.updated", Payload: "sku-42"}) // no subscriber

	cancelOrder()
	cancelAll()
	cancelPayment()
	wg.Wait()
}

// ─────────────────────────────────────────────────────────────────────────────
// SCENARIO 2: rate-limited publisher + DLQ
// ─────────────────────────────────────────────────────────────────────────────

func scenarioRateAndDLQ() {
	fmt.Println()
	fmt.Println("=== Scenario 2: rate-limited publisher + dead-letter queue ===")

	bus := NewEventBus(5)

	// Slow subscriber — small buffer so messages pile up.
	sub, cancel := bus.Subscribe("metrics", 2)

	dlqCount := atomic.Int64{}
	go func() {
		for range bus.DLQ() {
			dlqCount.Add(1)
		}
	}()

	pub := NewRateLimitedPublisher(bus, 10) // 10 rps = publish every 100ms

	ctx, ctxCancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer ctxCancel()

	var received int64
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for e := range sub.ch {
			received++
			_ = e
			time.Sleep(40 * time.Millisecond) // slower than publish rate
		}
	}()

	published := 0
	throttled := 0
	for {
		select {
		case <-ctx.Done():
			cancel()
			wg.Wait()
			bus.Close()
			fmt.Printf("  published: %d  throttled: %d  received: %d  dropped-to-dlq: %d\n",
				published, throttled, received, dlqCount.Load())
			return
		default:
			e := Event{Topic: "metrics", Payload: fmt.Sprintf("m%d", published)}
			if pub.Publish(e) {
				published++
			} else {
				throttled++
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// SCENARIO 3: fan-out with per-subscriber dropped stats
// ─────────────────────────────────────────────────────────────────────────────

func scenarioDropStats() {
	fmt.Println()
	fmt.Println("=== Scenario 3: per-subscriber drop tracking ===")

	bus := NewEventBus(50)
	subs := make([]*Subscription, 3)
	cancels := make([]func(), 3)
	speeds := []time.Duration{1, 10, 30} // ms processing delay

	var wg sync.WaitGroup
	for i := range 3 {
		subs[i], cancels[i] = bus.Subscribe("stream", 4)
		id := i
		speed := speeds[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range subs[id].ch {
				time.Sleep(speed * time.Millisecond)
			}
		}()
	}

	// Burst publish 30 events.
	for i := range 30 {
		bus.Publish(Event{Topic: "stream", Payload: i})
		time.Sleep(5 * time.Millisecond)
	}

	for _, cancel := range cancels {
		cancel()
	}
	wg.Wait()

	for i, sub := range subs {
		fmt.Printf("  subscriber %d (delay %s): dropped %d\n",
			i, speeds[i]*time.Millisecond, sub.dropped.Load())
	}
}

func main() {
	scenarioWildcard()
	scenarioRateAndDLQ()
	scenarioDropStats()
}
