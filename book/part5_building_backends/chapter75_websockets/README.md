# Chapter 75 — WebSockets, SSE, and Long-Lived Connections

## What you'll learn

How to build long-lived connections in Go: the WebSocket hub pattern, rooms, presence tracking, message history for reconnection, SSE event streaming, heartbeat/ping-pong, and how to scale sticky sessions across multiple server instances.

## Key concepts

| Concept | Description |
|---|---|
| WebSocket | Full-duplex TCP connection over HTTP upgrade; binary + text frames |
| SSE | Half-duplex HTTP chunked stream; text-only; built-in reconnect |
| Hub | Central goroutine that manages connections and fan-out |
| Room | Named subset of connections that receive the same messages |
| Heartbeat | Periodic ping/pong to detect dead connections |
| Graceful close | Client and server agree to close with a status code and reason |
| Last-Event-ID | SSE reconnection header; client sends last seen ID on reconnect |
| Message history | Ring buffer per room; replay missed messages on reconnect |
| Presence | Track which users are online and in which rooms |
| Sticky sessions | Load balancer pins each client to the same instance |

## Files

| File | Topic |
|---|---|
| `examples/01_websocket_basics/main.go` | Connections, message types, ping/pong heartbeat, SSE format, graceful close |
| `examples/02_websocket_patterns/main.go` | Hub, rooms, broadcast, unregister, reconnection, Redis fan-out |
| `exercises/01_chat_hub/main.go` | Rooms, DMs, presence, message history, reconnection |

## Hub pattern

```go
// The hub owns a single goroutine; all map mutations happen inside it.
func (h *Hub) Run(ctx context.Context) {
    for {
        select {
        case c := <-h.register:
            h.clients[c.ID] = c
        case c := <-h.unregister:
            delete(h.clients, c.ID)
            c.Close()
        case msg := <-h.broadcast:
            for _, c := range h.clients {
                c.Send(msg)
            }
        }
    }
}
```

## Real WebSocket with gorilla/websocket

```go
import "github.com/gorilla/websocket"

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer conn.Close()

    // Set ping handler (called when client sends ping).
    conn.SetPingHandler(func(data string) error {
        return conn.WriteControl(websocket.PongMessage, []byte(data), time.Now().Add(time.Second))
    })

    for {
        msgType, msg, err := conn.ReadMessage()
        if err != nil {
            break // connection closed
        }
        conn.WriteMessage(msgType, msg) // echo
    }
}
```

## SSE handler

```go
func sseHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    flusher := w.(http.Flusher)

    lastID, _ := strconv.ParseInt(r.Header.Get("Last-Event-ID"), 10, 64)
    // Replay missed events.
    for _, e := range eventLog.Since(lastID) {
        fmt.Fprintf(w, "id: %d\ndata: %s\n\n", e.ID, e.Data)
        flusher.Flush()
    }

    for {
        select {
        case <-r.Context().Done():
            return
        case event := <-eventStream:
            fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", event.ID, event.Type, event.Data)
            flusher.Flush()
        }
    }
}
```

## Heartbeat

```go
conn.SetReadDeadline(time.Now().Add(60 * time.Second))
conn.SetPongHandler(func(string) error {
    conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    return nil
})
go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    for range ticker.C {
        if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
            return
        }
    }
}()
```

## Scaling with Redis pub/sub

```go
// Each instance subscribes to a shared channel.
sub := rdb.Subscribe(ctx, "hub:broadcast", "hub:room:"+roomID)
go func() {
    for msg := range sub.Channel() {
        hub.localBroadcast(msg.Payload)
    }
}()

// To broadcast across all instances:
rdb.Publish(ctx, "hub:broadcast", payload)
```

## Production notes

- Use `nhooyr.io/websocket` as an alternative to gorilla — it supports `net/http` context natively
- Always set read/write deadlines; a stalled connection holds a goroutine indefinitely
- Slow clients: drop messages with a full inbox warning, or disconnect after N drops
- Sticky sessions (IP hash or cookie) required for stateful WebSocket if using multiple instances without a pub/sub relay
- SSE reconnects automatically on network drops; WebSocket requires client-side reconnect logic
