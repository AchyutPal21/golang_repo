// FILE: book/part5_building_backends/chapter73_kafka/examples/01_kafka_concepts/main.go
// CHAPTER: 73 — Kafka
// TOPIC: Core Kafka concepts simulated in-process: topics, partitions, offsets,
//        consumer groups, at-least-once delivery, and partition assignment.
//
// No real Kafka broker needed — the simulation is pure Go to illustrate semantics.
//
// Run (from the chapter folder):
//   go run ./examples/01_kafka_concepts

package main

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// RECORD — smallest unit of data in Kafka
// ─────────────────────────────────────────────────────────────────────────────

type Record struct {
	Key       string
	Value     []byte
	Partition int
	Offset    int64
	Timestamp time.Time
}

// ─────────────────────────────────────────────────────────────────────────────
// PARTITION — ordered, immutable log of records
// ─────────────────────────────────────────────────────────────────────────────

type Partition struct {
	mu      sync.RWMutex
	records []*Record
}

func (p *Partition) append(r *Record) int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	r.Offset = int64(len(p.records))
	p.records = append(p.records, r)
	return r.Offset
}

// ReadFrom returns records starting at offset (inclusive).
func (p *Partition) ReadFrom(offset int64) []*Record {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if offset >= int64(len(p.records)) {
		return nil
	}
	out := make([]*Record, len(p.records)-int(offset))
	copy(out, p.records[offset:])
	return out
}

func (p *Partition) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.records)
}

// ─────────────────────────────────────────────────────────────────────────────
// TOPIC — collection of partitions
// ─────────────────────────────────────────────────────────────────────────────

type Topic struct {
	name       string
	partitions []*Partition
}

func NewTopic(name string, numPartitions int) *Topic {
	parts := make([]*Partition, numPartitions)
	for i := range parts {
		parts[i] = &Partition{}
	}
	return &Topic{name: name, partitions: parts}
}

// partition selects partition by key hash (round-robin if key is empty).
var rrCounter atomic.Int64

func (t *Topic) partition(key string) int {
	if key == "" {
		return int(rrCounter.Add(1)-1) % len(t.partitions)
	}
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32()) % len(t.partitions)
}

func (t *Topic) Publish(key string, value []byte) *Record {
	idx := t.partition(key)
	r := &Record{
		Key:       key,
		Value:     value,
		Partition: idx,
		Timestamp: time.Now(),
	}
	t.partitions[idx].append(r)
	return r
}

// ─────────────────────────────────────────────────────────────────────────────
// CONSUMER GROUP — shared offset tracking across consumers
// ─────────────────────────────────────────────────────────────────────────────

type ConsumerGroup struct {
	name      string
	topic     *Topic
	mu        sync.Mutex
	committed []int64 // last committed (processed) offset per partition
}

func NewConsumerGroup(name string, topic *Topic) *ConsumerGroup {
	return &ConsumerGroup{
		name:      name,
		topic:     topic,
		committed: make([]int64, len(topic.partitions)),
	}
}

// Poll returns unread records starting from the committed offset.
// Offsets are only advanced on an explicit Commit call.
func (cg *ConsumerGroup) Poll(partitions []int) []*Record {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	var out []*Record
	for _, p := range partitions {
		records := cg.topic.partitions[p].ReadFrom(cg.committed[p])
		out = append(out, records...)
	}
	return out
}

// Commit advances the committed offset for a partition after successful processing.
func (cg *ConsumerGroup) Commit(partition int, offset int64) {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	if offset > cg.committed[partition] {
		cg.committed[partition] = offset
	}
}

func (cg *ConsumerGroup) Offsets() []int64 {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	out := make([]int64, len(cg.committed))
	copy(out, cg.committed)
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Kafka Concepts (in-process simulation) ===")
	fmt.Println()

	// ── TOPIC + PARTITIONS ────────────────────────────────────────────────────
	fmt.Println("--- Topic with 3 partitions ---")
	topic := NewTopic("orders", 3)

	orders := []struct{ key, val string }{
		{"cust-A", `{"order":"o-1","total":100}`},
		{"cust-B", `{"order":"o-2","total":200}`},
		{"cust-A", `{"order":"o-3","total":150}`}, // same key → same partition
		{"cust-C", `{"order":"o-4","total":300}`},
		{"cust-B", `{"order":"o-5","total":250}`}, // same key → same partition
		{"cust-A", `{"order":"o-6","total":50}`},
	}
	for _, o := range orders {
		r := topic.Publish(o.key, []byte(o.val))
		fmt.Printf("  published key=%s → partition=%d offset=%d\n", o.key, r.Partition, r.Offset)
	}

	fmt.Println()
	for i, p := range topic.partitions {
		fmt.Printf("  partition %d: %d records\n", i, p.Len())
	}

	// ── ORDERED DELIVERY PER KEY ──────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Ordered delivery per key (same partition) ---")
	// cust-A always hashes to the same partition → its records arrive in order.
	h := fnv.New32a()
	h.Write([]byte("cust-A"))
	pA := int(h.Sum32()) % 3
	records := topic.partitions[pA].ReadFrom(0)
	fmt.Printf("  cust-A partition=%d records:\n", pA)
	for _, r := range records {
		fmt.Printf("    offset=%d value=%s\n", r.Offset, r.Value)
	}

	// ── CONSUMER GROUP ────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Consumer group: two consumers sharing 3 partitions ---")
	group := NewConsumerGroup("order-processors", topic)

	// Consumer 1 owns partitions 0 and 1; Consumer 2 owns partition 2.
	var wg sync.WaitGroup
	var mu sync.Mutex
	processed := map[string]int{}

	consumers := []struct {
		name       string
		partitions []int
	}{
		{"consumer-1", []int{0, 1}},
		{"consumer-2", []int{2}},
	}

	for _, c := range consumers {
		c := c
		wg.Add(1)
		go func() {
			defer wg.Done()
			recs := group.Poll(c.partitions)
			for _, r := range recs {
				fmt.Printf("  [%s] p=%d off=%d key=%s\n", c.name, r.Partition, r.Offset, r.Key)
				mu.Lock()
				processed[c.name]++
				mu.Unlock()
				// Commit after each record (auto-commit pattern).
				group.Commit(r.Partition, r.Offset+1)
			}
		}()
	}
	wg.Wait()

	fmt.Println()
	for _, c := range consumers {
		fmt.Printf("  %s processed %d records\n", c.name, processed[c.name])
	}
	fmt.Printf("  committed offsets: %v\n", group.Offsets())

	// ── AT-LEAST-ONCE DELIVERY ────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- At-least-once delivery: simulated crash before commit ---")

	topic2 := NewTopic("payments", 1)
	for i := 1; i <= 4; i++ {
		topic2.Publish("pay", []byte(fmt.Sprintf(`{"payment":%d}`, i)))
	}

	g2 := NewConsumerGroup("payment-processors", topic2)

	// First poll: consumer receives but "crashes" — never calls Commit.
	batch1 := g2.Poll([]int{0})
	fmt.Printf("  poll 1: got %d records (simulating crash — no Commit)\n", len(batch1))
	_ = batch1 // crash: do not commit

	// Committed offset is still 0 — same records redelivered on next poll.
	batch2 := g2.Poll([]int{0})
	fmt.Printf("  poll 2 (redelivery after crash): got %d records\n", len(batch2))
	for _, r := range batch2 {
		fmt.Printf("    redelivered offset=%d value=%s\n", r.Offset, r.Value)
	}
	// Successfully processed — now commit.
	g2.Commit(0, int64(len(batch2)))
	fmt.Printf("  Commit(partition=0, offset=%d) done\n", len(batch2))

	batch3 := g2.Poll([]int{0})
	fmt.Printf("  poll 3: %d records (nothing new — committed offset advanced)\n", len(batch3))

	// ── PARTITION REBALANCE EXPLANATION ──────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Partition rebalance (conceptual) ---")
	fmt.Println("  When a consumer joins/leaves a group:")
	fmt.Println("  1. All consumers stop polling (rebalance begins)")
	fmt.Println("  2. Group coordinator reassigns partitions")
	fmt.Println("  3. Each consumer resumes from its committed offset")
	fmt.Println("  → committed offsets prevent re-processing after rebalance")
	fmt.Println()
	fmt.Println("  Rule: #partitions >= #consumers for full parallelism")
	fmt.Println("  Extra consumers idle; extra partitions → more parallelism headroom")

	// ── OFFSETS AND RETENTION ─────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Offset management and retention ---")

	_ = context.Background() // illustrate context usage in real producers/consumers
	fmt.Println("  earliest offset: read from beginning of partition (replay)")
	fmt.Println("  latest offset:   read only new records (live tail)")
	fmt.Println("  committed offset: last successfully processed position in group")
	fmt.Println()
	fmt.Println("  retention policy: time-based (7 days) or size-based (50 GB)")
	fmt.Println("  records are NOT deleted on consume — consumers advance their own pointer")
	fmt.Println("  multiple consumer groups can read the same topic independently")
}
