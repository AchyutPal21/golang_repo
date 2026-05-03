# Chapter 73 — Kafka

## What you'll learn

How Apache Kafka works: topics, partitions, offsets, consumer groups, and the guarantee properties (at-least-once, exactly-once). All concepts are demonstrated in-process without a real broker, so you can focus on the semantics before connecting to real infrastructure.

## Key concepts

| Concept | Description |
|---|---|
| Topic | Named channel of records; split into partitions |
| Partition | Ordered, immutable log; unit of parallelism |
| Offset | Per-partition record position; consumers own their position |
| Consumer group | Set of consumers sharing a topic; each partition assigned to one member |
| Rebalance | Partition reassignment when group membership changes |
| At-least-once | Records can be redelivered; consumer must be idempotent |
| Exactly-once | At-least-once + idempotent consumer (dedup by partition:offset) |
| Compacted topic | Retains only the latest value per key; tombstone (nil) deletes |
| Event sourcing | Replay the topic log to rebuild any past or present state |
| Dead-letter topic | Separate topic for unprocessable records |

## Files

| File | Topic |
|---|---|
| `examples/01_kafka_concepts/main.go` | Topics, partitions, offsets, consumer groups, rebalance, at-least-once |
| `examples/02_kafka_patterns/main.go` | Idempotent consumer, fan-out groups, event sourcing, compaction, headers |
| `exercises/01_event_stream/main.go` | Broker registry, projection, DLQ topic, rebalance simulation |

## Core semantics

### Why keys matter

A record published with the same key always lands on the same partition (consistent hash). This guarantees ordered delivery for all events belonging to one entity (e.g., all events for `order-123`). Records with different keys may share a partition.

```
Publish("ord-1", ...) → partition = fnv32("ord-1") % numPartitions
```

### Offset management

- Each consumer group maintains its own committed offset per partition.
- `Poll` returns records from the committed offset onward.
- `Commit` advances the committed offset after successful processing.
- If a consumer crashes before committing, the same records are re-delivered on the next poll (at-least-once).

```go
records := group.Poll(myPartitions)
for _, r := range records {
    process(r)
    group.Commit(r.Partition, r.Offset+1)  // advance only after success
}
```

### Idempotent consumer (exactly-once)

```go
seen := make(map[string]struct{}) // persisted in Redis or DB in production

for _, r := range records {
    key := fmt.Sprintf("%d:%d", r.Partition, r.Offset)
    if _, ok := seen[key]; ok {
        continue // duplicate — skip
    }
    seen[key] = struct{}{}
    process(r)
    group.Commit(r.Partition, r.Offset+1)
}
```

### Fan-out to multiple consumer groups

```go
// All three groups read the same topic independently.
inventoryGroup  := NewConsumerGroup(ordersTopic)
emailGroup      := NewConsumerGroup(ordersTopic)
analyticsGroup  := NewConsumerGroup(ordersTopic)
```

Each group has its own committed offset — adding a new group does not affect existing ones, and it can replay from offset 0 to backfill.

### Event sourcing with replay

```go
// Replay from offset 0 to rebuild any materialized view.
for _, r := range partition.ReadFrom(0) {
    projection.Apply(r)
}
```

### Compacted topic

```go
ct.Write("user:1", []byte(`{"name":"Alice"}`)) // upsert
ct.Write("user:2", nil)                         // tombstone: delete
snapshot := ct.Snapshot()                        // latest value per surviving key
```

## Connecting to real Kafka (go-kafka / sarama)

```go
// Using github.com/segmentio/kafka-go
w := kafka.NewWriter(kafka.WriterConfig{
    Brokers: []string{"localhost:9092"},
    Topic:   "orders",
})
w.WriteMessages(ctx, kafka.Message{Key: []byte("ord-1"), Value: payload})

r := kafka.NewReader(kafka.ReaderConfig{
    Brokers: []string{"localhost:9092"},
    Topic:   "orders",
    GroupID: "inventory-service",
})
msg, _ := r.FetchMessage(ctx)
r.CommitMessages(ctx, msg)
```

## Production notes

- Set `NumPartitions` to match your target parallelism; you can't reduce partitions after creation
- Always commit *after* processing, not before — pre-commit causes message loss on crash
- Monitor consumer lag (committed offset vs log-end offset) to detect slow consumers
- Use a dead-letter topic rather than discarding unprocessable records
- Compacted topics work well for config, user profiles, and other "current state" data
- Kafka retains records even after consumption; retention is controlled by time or bytes
