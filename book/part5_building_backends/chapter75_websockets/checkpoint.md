# Chapter 75 Checkpoint — WebSockets, SSE, and Long-Lived Connections

## Self-assessment questions

1. What are the key differences between WebSocket and SSE? When would you choose SSE?
2. Why does the hub pattern use a single goroutine to manage all map mutations? What problem does this solve?
3. What is the purpose of the ping/pong heartbeat? What happens without it?
4. How does SSE reconnection work? What does the `Last-Event-ID` header do?
5. Why do WebSockets create a scaling challenge with multiple server instances? How do you solve it?

## Checklist

- [ ] Can implement a WebSocket connection with send and receive goroutines
- [ ] Can implement a Hub with register, unregister, and broadcast channels
- [ ] Can implement room-based broadcast (fan-out only to clients in the same room)
- [ ] Can implement a ping/pong heartbeat that disconnects dead clients
- [ ] Can implement graceful close with a status code and reason
- [ ] Can implement SSE with proper headers and event format
- [ ] Can implement SSE reconnection using Last-Event-ID and an event log
- [ ] Can implement presence tracking (online users per room)
- [ ] Can implement message history for reconnecting clients

## Answers

1. WebSocket: full-duplex, both sides send frames freely, binary and text, requires WebSocket-aware proxies. SSE: server-to-client only, text-only, works through any HTTP proxy, has built-in reconnection with `Last-Event-ID`. Choose SSE when: the client only reads from the server (notifications, live scores, log tails), you want browser reconnection without custom JavaScript, or you need CDN/proxy compatibility.

2. The hub's goroutine serialises all `register`, `unregister`, and `broadcast` operations through channels. Without this, concurrent writes and reads to `map[string]*Client` would cause a data race (Go maps are not thread-safe). A mutex works too, but a channel-based hub avoids holding a lock during potentially slow sends, and makes the control flow explicit and readable.

3. Ping/pong detects connections that are silently dead — the TCP connection looks alive but the remote host is unreachable or crashed. Without a heartbeat, the server's goroutine and file descriptor for that connection would leak indefinitely. Setting a read deadline and resetting it on each pong means the connection is automatically cleaned up if no pong arrives within the timeout.

4. When the SSE connection drops, the browser automatically reconnects and sends the `Last-Event-ID: N` header. The server reads this and replays all events with ID > N from its event log. This ensures the client never misses events that were published during the disconnection window. Always include an `id:` line in SSE responses to enable this.

5. Each WebSocket connection is pinned to one server instance. If the load balancer routes a broadcast originating on instance-1 to instance-2, instance-2 has no connection to the target client. Solution: use a shared pub/sub channel (Redis `PUBLISH`/`SUBSCRIBE`). Each instance subscribes; when any instance needs to broadcast, it publishes to Redis and all instances forward to their local clients. Alternatively, use sticky sessions (IP hash or a cookie) to keep each client on the same instance — simpler but limits rebalancing.
