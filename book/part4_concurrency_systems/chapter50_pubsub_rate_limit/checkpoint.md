# Chapter 50 — Revision Checkpoint

## Questions

1. What is the key trade-off between blocking and non-blocking fan-out in a pub/sub broker?
2. How does a token bucket differ from a sliding-window rate limiter, and when would you prefer each?
3. What distinguishes a throttle from a debounce, and which does the leading-edge implementation in this chapter represent?
4. Why does the leaky bucket guarantee a constant output rate while the token bucket does not?
5. When a subscriber's channel is full and a message is dropped, what information is lost and how does a dead-letter queue help?

## Answers

1. **Blocking fan-out** (no `default`) pushes back-pressure from slow subscribers all the way to the publisher: a slow subscriber blocks the publish call, which blocks the caller. This preserves every message but can stall the whole system if any one subscriber is slow. **Non-blocking fan-out** (with `default`) keeps the publisher fast — it never waits — but slow subscribers silently miss messages. The right choice depends on whether delivery guarantees or publisher availability matter more. Most production brokers use non-blocking with a configurable per-subscriber buffer and metrics on drops.

2. A **token bucket** models a physical bucket filled at a constant rate up to a capacity. It explicitly allows bursts: if no requests arrived for a while, accumulated tokens let a burst of requests through immediately. A **sliding window** counts exact requests in a rolling time window and denies any request that would exceed the limit — there is no burst credit. Use the token bucket when downstream can handle short spikes; use the sliding window for hard legal or SLA limits (e.g., "never more than 1000 requests per minute").

3. **Throttle** fires at most once per interval, discarding subsequent calls during the interval: the first call wins (leading-edge). **Debounce** fires after a quiet period — it waits until calls stop arriving, then fires once for the trailing edge. The implementation in this chapter is a throttle (leading-edge): the call that arrives after `interval` has elapsed executes immediately, and further calls within the next interval are dropped. Classic use of debounce: search-as-you-type (wait for typing to stop). Classic use of throttle: scroll events, metrics aggregation.

4. The leaky bucket drains at a fixed tick rate driven by a `time.Ticker` — the ticker controls the output precisely regardless of how fast items arrive. The token bucket, by contrast, permits multiple tokens to accumulate when the bucket is under-used, allowing a burst of requests to all be admitted in rapid succession. The leaky bucket's drain goroutine can emit at most one item per tick interval; there is no accumulation mechanism.

5. When a message is dropped because a subscriber's buffer is full, you lose: (a) the message payload, (b) the delivery guarantee to that subscriber, and (c) any ordering continuity if subsequent messages do arrive. A dead-letter queue preserves dropped messages in a separate channel for later inspection, retry, alerting, or replay. It converts a silent loss into an auditable event — operators can see how many messages piled up and re-process them once the subscriber recovers. The DLQ itself has a finite capacity; if it also fills, messages are truly lost — monitoring the DLQ fill level is a key operational metric.
