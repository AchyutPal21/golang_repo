// FILE: book/part5_building_backends/chapter73_kafka/exercises/01_event_stream/main.go
// CHAPTER: 73 — Kafka
// TOPIC: Order event stream — multi-partition producer, consumer group rebalance,
//        event projection, and a dead-letter topic for unprocessable records.
//
// Run (from the chapter folder):
//   go run ./exercises/01_event_stream

package main

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// BROKER — registry of topics
// ─────────────────────────────────────────────────────────────────────────────

type Record struct {
	Key       string
	Value     []byte
	Topic     string
	Partition int
	Offset    int64
	Timestamp time.Time
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

func (t *Topic) partitionFor(key string) int {
	if key == "" {
		return 0
	}
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32()) % len(t.partitions)
}

func (t *Topic) Publish(key string, value []byte) *Record {
	idx := t.partitionFor(key)
	r := &Record{Key: key, Value: value, Topic: t.name, Partition: idx, Timestamp: time.Now()}
	t.partitions[idx].Append(r)
	return r
}

type Broker struct {
	mu     sync.RWMutex
	topics map[string]*Topic
}

func NewBroker() *Broker { return &Broker{topics: make(map[string]*Topic)} }

func (b *Broker) CreateTopic(name string, partitions int) *Topic {
	b.mu.Lock()
	defer b.mu.Unlock()
	t := NewTopic(name, partitions)
	b.topics[name] = t
	return t
}

func (b *Broker) Topic(name string) *Topic {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.topics[name]
}

// ─────────────────────────────────────────────────────────────────────────────
// CONSUMER GROUP with partition assignment
// ─────────────────────────────────────────────────────────────────────────────

type ConsumerGroup struct {
	mu        sync.Mutex
	topic     *Topic
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

func (cg *ConsumerGroup) Offsets() []int64 {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	out := make([]int64, len(cg.committed))
	copy(out, cg.committed)
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// PROJECTION — materialized view rebuilt from events
// ─────────────────────────────────────────────────────────────────────────────

type OrderStatus struct {
	OrderID string
	Status  string
	Total   int
	Updates int
}

type OrderProjection struct {
	mu     sync.RWMutex
	orders map[string]*OrderStatus
}

func NewOrderProjection() *OrderProjection {
	return &OrderProjection{orders: make(map[string]*OrderStatus)}
}

// jsonString extracts `"key":"value"` from a simple flat JSON string.
func jsonString(src, key string) string {
	needle := `"` + key + `":"`
	idx := strings.Index(src, needle)
	if idx < 0 {
		return ""
	}
	rest := src[idx+len(needle):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return rest
	}
	return rest[:end]
}

// jsonInt extracts `"key":number` from a simple flat JSON string.
func jsonInt(src, key string) int {
	needle := `"` + key + `":`
	idx := strings.Index(src, needle)
	if idx < 0 {
		return 0
	}
	rest := src[idx+len(needle):]
	end := strings.IndexAny(rest, ",}")
	if end < 0 {
		end = len(rest)
	}
	n, _ := strconv.Atoi(rest[:end])
	return n
}

func (op *OrderProjection) Apply(r *Record) {
	op.mu.Lock()
	defer op.mu.Unlock()

	val := string(r.Value)
	status := jsonString(val, "status")
	total := jsonInt(val, "total")

	order, ok := op.orders[r.Key]
	if !ok {
		order = &OrderStatus{OrderID: r.Key}
		op.orders[r.Key] = order
	}
	if status != "" {
		order.Status = status
	}
	if total > 0 {
		order.Total = total
	}
	order.Updates++
}

func (op *OrderProjection) Get(orderID string) (*OrderStatus, bool) {
	op.mu.RLock()
	defer op.mu.RUnlock()
	o, ok := op.orders[orderID]
	return o, ok
}

func (op *OrderProjection) All() []*OrderStatus {
	op.mu.RLock()
	defer op.mu.RUnlock()
	out := make([]*OrderStatus, 0, len(op.orders))
	for _, o := range op.orders {
		out = append(out, o)
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== Order Event Stream Exercise ===")
	fmt.Println()

	broker := NewBroker()
	ordersTopic := broker.CreateTopic("orders", 4)
	dlqTopic := broker.CreateTopic("orders.dlq", 1)

	// ── PRODUCE EVENTS ────────────────────────────────────────────────────────
	fmt.Println("--- Producing order events (key = orderID → deterministic partition) ---")
	events := []struct{ key, val string }{
		{"ord-1", `{"status":"placed","total":1000}`},
		{"ord-2", `{"status":"placed","total":2000}`},
		{"ord-3", `{"status":"placed","total":3000}`},
		{"ord-4", `{"status":"placed","total":4000}`},
		{"ord-1", `{"status":"confirmed","total":0}`},
		{"ord-2", `{"status":"confirmed","total":0}`},
		{"ord-3", `{"status":"shipped","total":0}`},
		{"ord-1", `{"status":"delivered","total":0}`},
	}
	for _, e := range events {
		r := ordersTopic.Publish(e.key, []byte(e.val))
		fmt.Printf("  produced key=%s → p=%d off=%d\n", e.key, r.Partition, r.Offset)
	}

	// ── CONSUMER GROUP WITH SIMULATED REBALANCE ────────────────────────────────
	fmt.Println()
	fmt.Println("--- Consumer group rebalance: 2 consumers → 3 consumers ---")

	group := NewConsumerGroup(ordersTopic)

	// Initial assignment: consumer-A gets p0,p1; consumer-B gets p2,p3.
	assignment := map[string][]int{
		"consumer-A": {0, 1},
		"consumer-B": {2, 3},
	}

	var wg sync.WaitGroup
	var processedMu sync.Mutex
	processed := map[string]int{}
	projection := NewOrderProjection()

	for name, parts := range assignment {
		name, parts := name, parts
		wg.Add(1)
		go func() {
			defer wg.Done()
			recs := group.Poll(parts)
			for _, r := range recs {
				projection.Apply(r)
				group.Commit(r.Partition, r.Offset+1)
				processedMu.Lock()
				processed[name]++
				processedMu.Unlock()
			}
			fmt.Printf("  [%s] processed %d records (partitions %v)\n", name, len(recs), parts)
		}()
	}
	wg.Wait()

	fmt.Printf("  committed offsets: %v\n", group.Offsets())

	// Rebalance: add consumer-C, redistribute partitions.
	fmt.Println()
	fmt.Println("  [rebalance] consumer-C joins — reassigning partitions...")

	newAssignment := map[string][]int{
		"consumer-A": {0, 1},
		"consumer-B": {2},
		"consumer-C": {3},
	}

	// Publish new events to trigger rebalance processing.
	ordersTopic.Publish("ord-5", []byte(`{"status":"placed","total":5000}`))
	ordersTopic.Publish("ord-6", []byte(`{"status":"placed","total":6000}`))

	for name, parts := range newAssignment {
		name, parts := name, parts
		wg.Add(1)
		go func() {
			defer wg.Done()
			recs := group.Poll(parts)
			for _, r := range recs {
				projection.Apply(r)
				group.Commit(r.Partition, r.Offset+1)
				processedMu.Lock()
				processed[name]++
				processedMu.Unlock()
			}
			if len(recs) > 0 {
				fmt.Printf("  [%s] processed %d new records after rebalance\n", name, len(recs))
			}
		}()
	}
	wg.Wait()

	// ── DEAD LETTER QUEUE FOR UNPROCESSABLE RECORDS ────────────────────────────
	fmt.Println()
	fmt.Println("--- Dead letter topic for unprocessable records ---")

	var dlqCount atomic.Int32
	malformedTopic := broker.CreateTopic("raw-events", 1)
	malformedTopic.Publish("bad-1", []byte(`not valid json`))
	malformedTopic.Publish("ok-1", []byte(`{"status":"placed","total":100}`))
	malformedTopic.Publish("bad-2", []byte(`{"broken`))

	rawGroup := NewConsumerGroup(malformedTopic)
	recs := rawGroup.Poll([]int{0})
	for _, r := range recs {
		rawGroup.Commit(r.Partition, r.Offset+1)
		val := string(r.Value)
		if len(val) == 0 || val[0] != '{' || val[len(val)-1] != '}' {
			// Route unprocessable record to DLQ.
			dlqTopic.Publish(r.Key, r.Value)
			dlqCount.Add(1)
			fmt.Printf("  [DLQ] routed key=%s val=%q\n", r.Key, val)
		} else {
			fmt.Printf("  [ok] processed key=%s val=%s\n", r.Key, val)
		}
	}
	fmt.Printf("  total DLQ records: %d\n", dlqCount.Load())

	// ── PROJECTION RESULTS ─────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Order projection (current state per order) ---")
	for _, o := range projection.All() {
		fmt.Printf("  order=%s status=%s total=%d updates=%d\n",
			o.OrderID, o.Status, o.Total, o.Updates)
	}

	// ── THROUGHPUT STATS ──────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Records per partition ---")
	for i, p := range ordersTopic.partitions {
		recs := p.ReadFrom(0)
		fmt.Printf("  partition %d: %d records\n", i, len(recs))
	}
}
