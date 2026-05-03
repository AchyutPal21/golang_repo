# Chapter 73 Checkpoint — Kafka

## Self-assessment questions

1. Why does Kafka guarantee ordered delivery only within a partition, not across partitions?
2. A consumer in group "payments" polls partition 2 and receives records at offsets 10–14. It processes offsets 10–12 successfully, then crashes. What offsets does the next consumer in the group receive, and why?
3. What is the difference between at-least-once and exactly-once delivery? What does the consumer need to implement to achieve exactly-once?
4. Why can't you reduce the number of partitions in a topic after creation?
5. What is a tombstone record in a compacted topic? When would you publish one?
6. How does a new consumer group that joins an existing topic backfill historical data without disrupting existing consumers?

## Checklist

- [ ] Can explain topics, partitions, and offsets and how they relate
- [ ] Can implement partition selection by key hash for ordered delivery
- [ ] Can implement a ConsumerGroup with Poll and Commit that models Kafka semantics
- [ ] Can simulate at-least-once redelivery by not committing before crash
- [ ] Can implement an idempotent consumer with a dedup store (partition:offset key)
- [ ] Can set up multiple consumer groups on the same topic for fan-out
- [ ] Can replay a topic partition from offset 0 to rebuild a projection
- [ ] Can implement a compacted topic with tombstone support
- [ ] Can route unprocessable records to a dead-letter topic

## Answers

1. Kafka appends records to a partition in arrival order and each consumer reads that partition sequentially — so within a partition, ordering is absolute. Across partitions there is no coordination: partition 0 and partition 1 are written and read independently, so there is no global ordering guarantee. Choosing the right partitioning key (e.g., orderID) ensures all events for one entity land in one partition and stay ordered.

2. The consumer committed offset 10 (no commits after that on partitions 2). The next consumer in the group picks up from offset 10, receiving records 10–14 again. This is at-least-once delivery: records 10–12 will be processed a second time. The consumer must be idempotent (dedup by partition:offset) to avoid duplicate side effects.

3. At-least-once: records may be delivered more than once; the consumer must handle duplicates or accept them. Exactly-once: the consumer tracks which (partition, offset) pairs it has already applied (in a persistent dedup store), skips re-seen records, and commits offset only after successful processing. In practice, storing the dedup key in the same database as the business action (in one transaction) gives true exactly-once semantics.

4. Partition count controls key distribution: `partition = hash(key) % numPartitions`. Reducing partitions would remap keys to different partitions, breaking the ordering guarantee — all records for a given key would scatter across partitions. Increasing partitions is allowed because existing keys remap to new partitions and consumers rebalance; reducing would corrupt historical ordering.

5. A tombstone is a record with a nil value published to a compacted topic. The log compactor treats it as a deletion marker: the key is removed from the compacted view. After the next compaction cycle, both the tombstone and all previous records for that key are purged. Use it when an entity is deleted (e.g., a user account closes, a config key is removed).

6. A new consumer group starts with no committed offsets. Depending on the `auto.offset.reset` config, it can start from `earliest` (offset 0) to replay the full history, or `latest` to read only future records. Starting from `earliest` allows the new group to backfill a projection without any impact on existing groups — each group owns its offset independently.
