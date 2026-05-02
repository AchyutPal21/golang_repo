# Chapter 53 — Exercises

## 53.1 — Echo server

Run [`exercises/01_echo_server`](exercises/01_echo_server/main.go).

Production echo server exercised through four scenarios: normal echo, idle timeout, max-connection rejection, graceful shutdown. Atomic stats track accepted connections, active connections, and bytes echoed.

Try:
- Add a `MaxMessageSize` config field; reject messages longer than N bytes.
- Add a `Logger` field (`func(format string, args ...any)`) and log each accepted/closed connection.
- Run with `-race` to confirm no data races.

## 53.2 ★ — Line-framed protocol

Build a server that speaks a simple request/response protocol over TCP:

```
Request:  CMD ARG\n
Response: OK result\n  or  ERR message\n
```

Commands:
- `SET key value` — stores key→value
- `GET key` — returns value or `ERR not found`
- `DEL key` — deletes key, returns `OK deleted`

Handle concurrent clients safely (use `sync.RWMutex` for the store).

## 53.3 ★★ — UDP metrics collector

Build a UDP server that receives metric packets in the format `metric.name:value|type` (simplified StatsD):
- `counter.requests:1|c` — increment counter
- `gauge.memory:512|g` — set gauge
- `timer.latency:45|ms` — record timing (store min/max/avg)

Print a summary every 5 seconds. Use atomic operations or a mutex for the aggregation store.
