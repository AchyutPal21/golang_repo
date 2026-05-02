# Chapter 53 — Networking I: TCP/UDP

## What you will learn

- TCP fundamentals: connection lifecycle, 3-way handshake, graceful close
- `net.Listen` → `Accept` → `Read`/`Write` → `Close` server loop
- Per-connection goroutines with idle timeout via `SetDeadline`
- Graceful server shutdown: close the listener, drain in-flight connections
- Concurrent connection tracking with `sync.WaitGroup` and `sync.Once`
- UDP: connectionless datagrams, `ReadFromUDP`/`WriteToUDP`, `ListenPacket`
- Connected UDP vs connectionless UDP (`net.Dial("udp", ...)`)
- TCP vs UDP trade-off table

---

## TCP server pattern

```go
ln, _ := net.Listen("tcp", ":8080")
for {
    conn, err := ln.Accept()
    if err != nil { return }
    go handleConn(conn)
}
```

Each `handleConn` must:
1. `defer conn.Close()`
2. Set a read deadline: `conn.SetReadDeadline(time.Now().Add(idle))`
3. Handle `io.EOF` (client closed) and `net.Error.Timeout()` (idle timeout)

---

## Graceful shutdown

```go
type Server struct {
    ln   net.Listener
    wg   sync.WaitGroup
    quit chan struct{}
    once sync.Once
}

func (s *Server) Shutdown() {
    s.once.Do(func() {
        close(s.quit)
        s.ln.Close()     // unblocks Accept — returns error
    })
    s.wg.Wait()          // drain all in-flight connections
}
```

Workers call `s.wg.Add(1)` before `go handleConn` and `defer s.wg.Done()` inside.

---

## Connection limits (semaphore)

```go
sem := make(chan struct{}, maxConns)

// In Accept loop:
select {
case sem <- struct{}{}:
    go func() { defer func() { <-sem }(); handleConn(conn) }()
default:
    conn.Close() // reject — at capacity
}
```

---

## UDP: connectionless server

```go
conn, _ := net.ListenPacket("udp", ":9090")
buf := make([]byte, 1500) // MTU
for {
    n, addr, _ := conn.ReadFrom(buf)
    response := process(buf[:n])
    conn.WriteTo(response, addr)
}
```

UDP has no concept of "connection" on the server side. `addr` is the sender's address, returned per-datagram.

---

## Setting deadlines

| Method | Scope |
|---|---|
| `conn.SetDeadline(t)` | Both read and write |
| `conn.SetReadDeadline(t)` | Read only |
| `conn.SetWriteDeadline(t)` | Write only |

Deadlines are absolute times, not durations. Reset them before each operation in a long-lived connection: `conn.SetReadDeadline(time.Now().Add(idle))`.

---

## Examples

| File | Demonstrates |
|---|---|
| `examples/01_tcp_server_client/main.go` | Echo server, concurrent clients, deadlines, context dial |
| `examples/02_udp_patterns/main.go` | UDP echo, fire-and-forget, multi-client, TCP vs UDP comparison |

## Exercise

`exercises/01_echo_server/main.go` — production echo server with idle timeout, max-connections semaphore, graceful shutdown, and atomic stats.
