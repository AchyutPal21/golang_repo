// FILE: book/part5_building_backends/chapter73_kafka/examples/02_kafka_patterns/main.go
// CHAPTER: 73 — Kafka
// TOPIC: Kafka production patterns: exactly-once semantics (idempotent consumers),
//        event sourcing with replay, compacted topics, and fan-out to multiple
//        consumer groups. All implemented in-process without a real broker.
//
// Run (from the chapter folder):
//   go run ./examples/02_kafka_patterns

package main

import (
	"fmt"
	"hash/fnv"
	"strings"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// CORE TYPES (minimal re-implementation to keep this file self-contained)
// ─────────────────────────────────────────────────────────────────────────────

type Record struct {
	Key       string
	Value     []byte
	Partition int
	Offset    int64
	Timestamp time.Time
	Headers   map[string]string
}

type Partition struct {
	mu      sync.RWMutex
	records []*Record
}

func (p *Partition) Append(r *Record) int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	r.Offset = int64(len(p.records))
	p.records = append(p.records, r)
	return r.Offset
}

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

func (p *Partition) All() []*Record {
	return p.ReadFrom(0)
}

type Topic struct {
	name       string
	partitions []*Partition
}

func NewTopic(name string, n int) *Topic {
	parts := make([]*Partition, n)
	for i := range parts {
		parts[i] = &Partition{}
	}
	return &Topic{name: name, partitions: parts}
}

func (t *Topic) partition(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32()) % len(t.partitions)
}

func (t *Topic) Publish(key string, value []byte, headers map[string]string) *Record {
	idx := t.partition(key)
	r := &Record{Key: key, Value: value, Partition: idx, Timestamp: time.Now(), Headers: headers}
	t.partitions[idx].Append(r)
	return r
}

type ConsumerGroup struct {
	topic     *Topic
	mu        sync.Mutex
	committed []int64
}

func NewConsumerGroup(topic *Topic) *ConsumerGroup {
	return &ConsumerGroup{topic: topic, committed: make([]int64, len(topic.partitions))}
}

func (cg *ConsumerGroup) Poll(partitions []int) []*Record {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	var out []*Record
	for _, p := range partitions {
		out = append(out, cg.topic.partitions[p].ReadFrom(cg.committed[p])...)
	}
	return out
}

func (cg *ConsumerGroup) Commit(partition int, nextOffset int64) {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	if nextOffset > cg.committed[partition] {
		cg.committed[partition] = nextOffset
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// IDEMPOTENT CONSUMER — exactly-once via deduplication store
// ─────────────────────────────────────────────────────────────────────────────

type IdempotentConsumer struct {
	mu      sync.Mutex
	seen    map[string]struct{} // key: "partition:offset"
	Applied int
	Skipped int
}

func NewIdempotentConsumer() *IdempotentConsumer {
	return &IdempotentConsumer{seen: make(map[string]struct{})}
}

func (ic *IdempotentConsumer) Process(r *Record, fn func(*Record)) {
	key := fmt.Sprintf("%d:%d", r.Partition, r.Offset)
	ic.mu.Lock()
	defer ic.mu.Unlock()
	if _, ok := ic.seen[key]; ok {
		ic.Skipped++
		return
	}
	ic.seen[key] = struct{}{}
	fn(r)
	ic.Applied++
}

// ─────────────────────────────────────────────────────────────────────────────
// COMPACTED TOPIC — retains only the latest value per key (tombstone on nil)
// ─────────────────────────────────────────────────────────────────────────────

type CompactedTopic struct {
	mu     sync.Mutex
	log    []*Record      // all records in append order
	latest map[string]int // key → index of latest record in log
	offset int64
}

func NewCompactedTopic() *CompactedTopic {
	return &CompactedTopic{latest: make(map[string]int)}
}

func (ct *CompactedTopic) Write(key string, value []byte) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	r := &Record{Key: key, Value: value, Offset: ct.offset, Timestamp: time.Now()}
	ct.offset++
	ct.log = append(ct.log, r)
	if value == nil {
		// Tombstone: delete the key from compacted view.
		delete(ct.latest, key)
	} else {
		ct.latest[key] = len(ct.log) - 1
	}
}

// Snapshot returns the compacted view (latest value per surviving key).
func (ct *CompactedTopic) Snapshot() map[string][]byte {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	out := make(map[string][]byte, len(ct.latest))
	for k, idx := range ct.latest {
		out[k] = ct.log[idx].Value
	}
	return out
}

func (ct *CompactedTopic) LogLen() int {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return len(ct.log)
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Kafka Patterns (in-process simulation) ===")
	fmt.Println()

	// ── IDEMPOTENT CONSUMER ───────────────────────────────────────────────────
	fmt.Println("--- Idempotent consumer (exactly-once via dedup) ---")

	topic := NewTopic("payments", 2)
	topic.Publish("pay-A", []byte(`{"amount":100}`), nil)
	topic.Publish("pay-B", []byte(`{"amount":200}`), nil)
	topic.Publish("pay-A", []byte(`{"amount":300}`), nil) // same key, different record

	group := NewConsumerGroup(topic)
	ic := NewIdempotentConsumer()

	// First processing pass — all records are new.
	batch := group.Poll([]int{0, 1})
	for _, r := range batch {
		ic.Process(r, func(r *Record) {
			fmt.Printf("  applied: p=%d off=%d key=%s val=%s\n",
				r.Partition, r.Offset, r.Key, r.Value)
		})
		group.Commit(r.Partition, r.Offset+1)
	}

	// Simulate redelivery: reset committed offsets to 0 (crash replay).
	group2 := NewConsumerGroup(topic)
	batch2 := group2.Poll([]int{0, 1})
	fmt.Printf("  redelivery: %d records re-received\n", len(batch2))
	for _, r := range batch2 {
		ic.Process(r, func(r *Record) {
			fmt.Printf("  applied (should not print): %s\n", r.Key)
		})
	}
	fmt.Printf("  applied=%d skipped(dedup)=%d\n", ic.Applied, ic.Skipped)

	// ── FAN-OUT TO MULTIPLE CONSUMER GROUPS ──────────────────────────────────
	fmt.Println()
	fmt.Println("--- Fan-out: same topic, independent consumer groups ---")

	orderTopic := NewTopic("orders", 3)
	events := []struct{ key, val string }{
		{"ord-1", `{"id":"ord-1","total":150}`},
		{"ord-2", `{"id":"ord-2","total":250}`},
		{"ord-3", `{"id":"ord-3","total":350}`},
	}
	for _, e := range events {
		orderTopic.Publish(e.key, []byte(e.val), nil)
	}

	// Three independent consumer groups — each reads all events.
	groups := map[string]*ConsumerGroup{
		"inventory-service":  NewConsumerGroup(orderTopic),
		"email-service":      NewConsumerGroup(orderTopic),
		"analytics-service":  NewConsumerGroup(orderTopic),
	}

	allPartitions := []int{0, 1, 2}
	for name, g := range groups {
		recs := g.Poll(allPartitions)
		for _, r := range recs {
			g.Commit(r.Partition, r.Offset+1)
		}
		fmt.Printf("  [%s] consumed %d records independently\n", name, len(recs))
	}

	// ── EVENT SOURCING + REPLAY ───────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Event sourcing: replay from offset 0 to rebuild state ---")

	accountTopic := NewTopic("account-events", 1)
	accountTopic.Publish("acc-1", []byte(`{"type":"opened","balance":0}`), nil)
	accountTopic.Publish("acc-1", []byte(`{"type":"deposit","amount":500}`), nil)
	accountTopic.Publish("acc-1", []byte(`{"type":"withdraw","amount":100}`), nil)
	accountTopic.Publish("acc-1", []byte(`{"type":"deposit","amount":200}`), nil)

	// Rebuild state by replaying from offset 0 (earliest).
	balance := 0
	replayed := 0
	for _, r := range accountTopic.partitions[0].All() {
		val := string(r.Value)
		switch {
		case strings.Contains(val, `"opened"`):
			balance = 0
		case strings.Contains(val, `"deposit"`):
			var amt int
			fmt.Sscanf(val, `{"type":"deposit","amount":%d}`, &amt)
			balance += amt
		case strings.Contains(val, `"withdraw"`):
			var amt int
			fmt.Sscanf(val, `{"type":"withdraw","amount":%d}`, &amt)
			balance -= amt
		}
		replayed++
		fmt.Printf("  [replay] offset=%d event=%s balance=%d\n", r.Offset, val, balance)
	}
	fmt.Printf("  final balance after replaying %d events: %d\n", replayed, balance)

	// ── COMPACTED TOPIC ───────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Compacted topic: latest value per key ---")

	ct := NewCompactedTopic()
	ct.Write("user:1", []byte(`{"name":"Alice","email":"a@x.com"}`))
	ct.Write("user:2", []byte(`{"name":"Bob","email":"b@x.com"}`))
	ct.Write("user:1", []byte(`{"name":"Alice","email":"alice@new.com"}`)) // update
	ct.Write("user:3", []byte(`{"name":"Carol","email":"c@x.com"}`))
	ct.Write("user:2", nil) // tombstone: delete Bob

	fmt.Printf("  log has %d total records\n", ct.LogLen())
	fmt.Println("  compacted snapshot (surviving keys):")
	for k, v := range ct.Snapshot() {
		fmt.Printf("    %s → %s\n", k, v)
	}
	fmt.Println("  (user:2 deleted by tombstone)")

	// ── HEADERS AND SCHEMA VERSIONING ─────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Record headers for schema versioning ---")

	schemaTopic := NewTopic("events", 1)
	schemaTopic.Publish("key-1", []byte(`{"v":1,"name":"Alice"}`), map[string]string{
		"content-type":   "application/json",
		"schema-version": "1",
	})
	schemaTopic.Publish("key-2", []byte(`{"v":2,"full_name":"Bob Smith","age":30}`), map[string]string{
		"content-type":   "application/json",
		"schema-version": "2",
	})

	for _, r := range schemaTopic.partitions[0].All() {
		fmt.Printf("  offset=%d schema-version=%s payload=%s\n",
			r.Offset, r.Headers["schema-version"], r.Value)
	}
}
