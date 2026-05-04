# Chapter 90 Checkpoint — Prometheus Metrics

## Concept checks

1. A counter's value at restart is 0. A Prometheus `rate()` function uses
   `irate` or `rate` over a time window. Why is it safe to use `rate()` on
   a counter that resets on restart?

2. Explain the difference between a `Summary` and a `Histogram` for latency
   measurement. Which one should you prefer in a microservices fleet and why?

3. You have a counter `http_requests_total{status="200"}`. A teammate adds
   a new label `user_id`. What will happen to Prometheus memory usage if
   the service handles 1 million unique users per day?

4. The USE method says: "Utilisation > 80% is a warning." Why is saturation
   a more urgent signal than utilisation?

5. What is the PromQL expression to compute the 95th-percentile latency from
   a histogram named `rpc_duration_seconds`, aggregated across all instances?

## Expected answers

1. Prometheus `rate()` accounts for counter resets automatically: if the
   current value is less than the previous sample, it assumes a reset and
   adjusts the calculation accordingly.

2. Histogram buckets are defined at instrumentation time and can be
   aggregated across instances with `sum()` before `histogram_quantile`.
   Summary computes exact quantiles per instance and cannot be aggregated.
   In a fleet, always prefer Histogram.

3. Memory usage will grow by approximately 1 KB × 1M users/day ≈ 1 GB per
   day. Prometheus will OOM. Never use `user_id` as a label.

4. Saturation (queue depth > 0, threads waiting) means the resource is
   already causing back-pressure to callers. Utilisation at 90% may still
   be absorbing load; saturation > 0 means latency is already degrading.

5. `histogram_quantile(0.95, sum(rate(rpc_duration_seconds_bucket[5m])) by (le))`
