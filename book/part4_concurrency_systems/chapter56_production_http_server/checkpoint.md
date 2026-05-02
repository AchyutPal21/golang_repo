# Chapter 56 — Production HTTP Server

## Questions

1. What is the difference between a liveness probe and a readiness probe, and what happens in Kubernetes when each fails?
2. Why is the recovery middleware placed outermost in the middleware chain?
3. What are the three `http.Server` timeout fields, and what specific attack or failure does each prevent?
4. Why must graceful shutdown listen for `SIGTERM` rather than only `SIGINT`, and what is the typical drain window in Kubernetes?
5. What is a correlation ID, and why is it more useful than a request log that contains only the path and status code?

## Answers

1. A **liveness probe** (`/health`) answers "is the process alive and not deadlocked?" — it should return 200 whenever the process is running and responding, even during startup. A **readiness probe** (`/ready`) answers "is this instance ready to receive traffic?" — it returns 503 during startup initialization (loading config, warming caches, waiting for DB) and 200 only after the server is fully ready. In Kubernetes: a liveness failure causes the container to be restarted; a readiness failure removes the pod from the `Service` endpoints (stops routing traffic to it) but does not restart it. Never fail the liveness probe due to external dependencies — a DB outage should not restart your pod.

2. Recovery is outermost so it wraps all inner middleware. A panic in any middleware layer (logging, rate limiting, timeout, or a handler) propagates up the call stack. If recovery were inner, panics in outer middleware would not be caught and would crash the goroutine serving the connection (and write no response). The outermost position guarantees recovery sees every panic regardless of which layer caused it, can write a `500` response, log the stack trace, and allow the server to continue serving other requests.

3. `ReadTimeout`: maximum time from accepting a TCP connection to reading the complete request body. Prevents a slow-send attack where a client sends the request headers byte-by-byte to hold a goroutine open. `WriteTimeout`: maximum time from the end of the request to the completion of the response write. Prevents a slow-read attack where a client reads the response very slowly, holding the write goroutine and connection open. `IdleTimeout`: maximum time a keep-alive connection may sit idle between requests. Without it, idle connections accumulate indefinitely, consuming file descriptors and memory.

4. `SIGTERM` is the standard Kubernetes shutdown signal. When Kubernetes terminates a pod (during rolling updates, node drain, or manual deletion), it sends `SIGTERM` to PID 1 in the container — not `SIGINT`. `SIGINT` is the interrupt signal typically sent by Ctrl-C in a terminal — useful for local development but not for production. If a server only handles `SIGINT`, Kubernetes will wait `terminationGracePeriodSeconds` (default 30s) and then send `SIGKILL`, killing the process immediately with no chance to drain in-flight requests. The typical drain window is 30 seconds — the server should call `Shutdown(ctx)` with a timeout equal to or less than `terminationGracePeriodSeconds`.

5. A correlation ID is a unique identifier (UUID or random hex string) attached to every request at the entry point, propagated through the system in both the request context and response headers (`X-Correlation-ID`). When a user reports an error, they can provide the correlation ID, and engineers can grep all log files across all services for that single ID to see the entire call chain: API gateway → auth service → order service → payment service. Without correlation IDs, reconstructing a distributed trace from logs requires matching timestamps, user IDs, and request paths across hundreds of log lines — error-prone and slow. With correlation IDs, the entire trace for one user action is retrievable in a single log query.
