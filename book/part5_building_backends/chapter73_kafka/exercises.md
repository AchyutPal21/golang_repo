# Chapter 73 Exercises — Kafka

## Exercise 1 — Order Event Stream (`exercises/01_event_stream`)

Build a mini Kafka simulation for an order lifecycle: a multi-topic broker, a consumer group with rebalance, a projection that rebuilds order state from events, and a dead-letter topic for unprocessable records.

### Broker

Implement a `Broker` that manages named topics:

```go
func (b *Broker) CreateTopic(name string, partitions int) *Topic
func (b *Broker) Topic(name string) *Topic
```

### Topic and Partition

```go
func (t *Topic) Publish(key string, value []byte) *Record   // hash(key) % numPartitions
```

Partition selection must be deterministic — the same key always maps to the same partition.

### ConsumerGroup

```go
func (cg *ConsumerGroup) Poll(partitions []int) []*Record   // reads from committed offset
func (cg *ConsumerGroup) Commit(partition int, nextOffset int64)
func (cg *ConsumerGroup) Offsets() []int64
```

`Poll` must NOT advance the committed offset — only `Commit` does.

### Order Projection

Build a `OrderProjection` that applies records to maintain current state per order:

```go
type OrderStatus struct {
    OrderID string
    Status  string
    Total   int
    Updates int
}

func (op *OrderProjection) Apply(r *Record)
func (op *OrderProjection) Get(orderID string) (*OrderStatus, bool)
func (op *OrderProjection) All() []*OrderStatus
```

The projection reads from `r.Key` (order ID) and parses status/total from `r.Value`.

### Demonstration

1. **Produce events**: publish 8 order events (4 order IDs × multiple status updates) — verify same key → same partition
2. **Consumer group**: two consumers sharing 4 partitions; process all events; commit per record
3. **Rebalance**: add a third consumer, republish 2 new orders, verify new partition assignment is honored
4. **Projection**: verify `ord-1` status = "delivered", `ord-3` status = "shipped"
5. **Dead letter**: publish 3 records to `raw-events` (2 malformed JSON, 1 valid); route malformed to `orders.dlq`; verify DLQ count = 2

### Hints

- FNV-32a hash: `h := fnv.New32a(); h.Write([]byte(key)); partition = int(h.Sum32()) % n`
- `Poll` reads from `committed[p]` without modifying it; `Commit` updates `committed[p]` to `offset+1`
- For JSON parsing without `encoding/json`, use `strings.Index` to extract fields by key
- Dead letter routing: check if `value[0] == '{'` and `value[len-1] == '}'` as a simple validity gate
- The rebalance demo requires publishing new records *after* the rebalance so consumers only receive records they haven't committed yet
