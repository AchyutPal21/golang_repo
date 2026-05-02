// FILE: book/part4_concurrency_systems/chapter50_pubsub_rate_limit/examples/01_pubsub/main.go
// CHAPTER: 50 — Pub/Sub, Rate Limit, Throttle
// TOPIC: In-process pub/sub broker — subscribe, publish, unsubscribe;
//        topic-based routing; broadcast vs single-consumer channels;
//        graceful shutdown.
//
// Run (from the chapter folder):
//   go run ./examples/01_pubsub

package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// BROKER
// ─────────────────────────────────────────────────────────────────────────────

type Message struct {
	Topic   string
	Payload any
}

type Broker struct {
	mu     sync.RWMutex
	subs   map[string][]chan Message
	closed bool
}

func NewBroker() *Broker {
	return &Broker{subs: make(map[string][]chan Message)}
}

// Subscribe returns a receive-only channel that delivers messages for topic.
// bufSize controls how many unread messages can queue before Publish blocks.
func (b *Broker) Subscribe(topic string, bufSize int) (<-chan Message, func()) {
	ch := make(chan Message, bufSize)

	b.mu.Lock()
	b.subs[topic] = append(b.subs[topic], ch)
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		subs := b.subs[topic]
		for i, c := range subs {
			if c == ch {
				b.subs[topic] = append(subs[:i], subs[i+1:]...)
				close(ch)
				return
			}
		}
	}
	return ch, unsubscribe
}

// Publish sends msg to all current subscribers of msg.Topic.
// Non-blocking: if a subscriber's buffer is full, the message is dropped for that subscriber.
func (b *Broker) Publish(msg Message) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return
	}
	for _, ch := range b.subs[msg.Topic] {
		select {
		case ch <- msg:
		default:
			// subscriber too slow — drop rather than block publisher
		}
	}
}

// Close shuts down the broker, closing all subscriber channels.
func (b *Broker) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for _, subs := range b.subs {
		for _, ch := range subs {
			close(ch)
		}
	}
	b.subs = make(map[string][]chan Message)
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 1: basic subscribe / publish / unsubscribe
// ─────────────────────────────────────────────────────────────────────────────

func demoBasicPubSub() {
	fmt.Println("=== Basic pub/sub ===")

	b := NewBroker()

	orders, unsubOrders := b.Subscribe("orders", 10)
	payments, unsubPayments := b.Subscribe("payments", 10)
	ordersAlso, unsubOrdersAlso := b.Subscribe("orders", 10)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for msg := range orders {
			fmt.Printf("  [orders-1]   received: %v\n", msg.Payload)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for msg := range payments {
			fmt.Printf("  [payments]   received: %v\n", msg.Payload)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for msg := range ordersAlso {
			fmt.Printf("  [orders-2]   received: %v\n", msg.Payload)
		}
	}()

	b.Publish(Message{Topic: "orders", Payload: "order #1001"})
	b.Publish(Message{Topic: "payments", Payload: "payment $99"})
	b.Publish(Message{Topic: "orders", Payload: "order #1002"})

	// Unsubscribe orders-2 before last message.
	unsubOrdersAlso()
	b.Publish(Message{Topic: "orders", Payload: "order #1003 (only orders-1 gets this)"})

	unsubOrders()
	unsubPayments()
	wg.Wait()
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 2: high-frequency publisher, slow subscriber — drop behaviour
// ─────────────────────────────────────────────────────────────────────────────

func demoDropOnSlow() {
	fmt.Println()
	fmt.Println("=== Slow subscriber: message drop ===")

	b := NewBroker()
	ch, unsub := b.Subscribe("metrics", 3) // small buffer

	var received int64
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range ch {
			received++
			time.Sleep(5 * time.Millisecond) // slow consumer
		}
	}()

	published := 0
	for i := range 20 {
		b.Publish(Message{Topic: "metrics", Payload: i})
		published++
		time.Sleep(1 * time.Millisecond)
	}

	unsub()
	wg.Wait()

	fmt.Printf("  published: %d  received: %d  dropped: %d\n",
		published, received, int64(published)-received)
}

// ─────────────────────────────────────────────────────────────────────────────
// DEMO 3: context-aware subscriber — stop on context cancel
// ─────────────────────────────────────────────────────────────────────────────

func demoContextAwareSub() {
	fmt.Println()
	fmt.Println("=== Context-aware subscriber ===")

	b := NewBroker()
	ch, unsub := b.Subscribe("events", 10)
	defer unsub()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	received := 0
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				received++
				_ = msg
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				b.Publish(Message{Topic: "events", Payload: i})
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	<-done
	fmt.Printf("  received ~%d events in 50ms before context cancelled\n", received)
}

func main() {
	demoBasicPubSub()
	demoDropOnSlow()
	demoContextAwareSub()
}
