# Chapter 90 — Prometheus Metrics

Prometheus is the de-facto standard for metrics in Go services. This chapter
covers the four metric types, the RED/USE observability frameworks, and the
label cardinality discipline that keeps Prometheus healthy.

## Topics

| # | Topic | Key ideas |
|---|-------|-----------|
| 1 | Counter/Gauge/Histogram/Summary | Types, rendering, PromQL |
| 2 | RED/USE methods | Rate/Error/Duration, Utilisation/Saturation/Errors |
| E | Label cardinality | Explosion demo, SafeCounter, overflow policy |

## Examples

### `examples/01_metrics_types`

Pure in-process implementations of all four Prometheus metric types:

- `Counter` — atomic int64, `Inc`/`Add`, text-format rendering
- `Gauge` — float64 with mutex, `Set`/`Inc`/`Dec`
- `Histogram` — bucket array, `Observe`, `Percentile`, `Mean`
- `Summary` — sliding window, exact quantiles
- Prometheus text exposition format reference
- PromQL for each type

### `examples/02_red_use`

RED (services) and USE (resources) frameworks:

- `ServiceMetrics` — requests/errors/durations with concurrent recording
- `ResourceMetrics` — capacity/utilisation/saturation/errors
- Multi-service dashboard with alert indicators
- Label cardinality rules (low vs high cardinality)
- Full PromQL reference for RED and USE

### `exercises/01_cardinality`

Demonstrates the cardinality explosion problem:

- 100k users → 100k series → 97 MB just for one counter
- `LabelPolicy` — enforces per-key distinct value limits
- `SafeCounter` — wraps policy into multi-label counter
- Overflow sentinel (`__overflow__`) keeps series bounded
- Anti-pattern reference: `request_id`, `user_id`, raw error messages

## Key Concepts

**Metric type selection**
| Use | Type |
|-----|------|
| Request count, error count | Counter |
| Queue depth, goroutines, CPU% | Gauge |
| Latency, request size (aggregatable) | Histogram |
| Exact quantiles per instance | Summary |

**RED** (for every user-facing service)
- **R**ate: `rate(requests_total[5m])`
- **E**rrors: `rate(errors_total[5m]) / rate(requests_total[5m])`
- **D**uration: `histogram_quantile(0.95, rate(duration_bucket[5m]))`

**USE** (for every resource)
- **U**tilisation: `in_use / capacity`
- **S**aturation: queue depth > 0, CPU steal > 0
- **E**rrors: timeouts, connection refused

**Cardinality rule**: if a label can have > 100 distinct values, it belongs
in logs/traces, not metrics labels.

## Running

```bash
go run ./part6_production_engineering/chapter90_prometheus/examples/01_metrics_types
go run ./part6_production_engineering/chapter90_prometheus/examples/02_red_use
go run ./part6_production_engineering/chapter90_prometheus/exercises/01_cardinality
```
