# Chapter 53 — Revision Checkpoint

## Questions

1. What is the difference between `SetDeadline`, `SetReadDeadline`, and `SetWriteDeadline`, and when would you use each?
2. How does the graceful shutdown pattern work, and why do you need both `ln.Close()` and `wg.Wait()`?
3. Why does a UDP server use `ReadFromUDP`/`WriteToUDP` instead of reading/writing directly, and what information would it lose if it didn't?
4. What is the difference between `net.Listen`/`Accept` and `net.Dial` from the perspective of which side initiates the connection?
5. Why is a goroutine-per-connection model practical in Go but problematic in languages without cheap goroutines?

## Answers

1. `SetDeadline(t)` sets both read and write deadlines to `t` simultaneously — useful for an overall per-request timeout. `SetReadDeadline(t)` affects only blocking reads; it expires if the remote side sends nothing after `t`. `SetWriteDeadline(t)` affects only blocking writes; it expires if the OS send buffer stays full (remote side not reading) after `t`. Use `SetReadDeadline` to enforce idle timeouts (close slow clients), `SetWriteDeadline` to prevent slow-write attacks that hold the server goroutine open indefinitely, and `SetDeadline` for an overall per-request deadline when both sides must complete within a fixed window.

2. `ln.Close()` causes the blocking `Accept()` call to return with an error, allowing the accept loop goroutine to exit. Without it, the accept goroutine would block forever even after the server decides to shut down. `wg.Wait()` then blocks until all in-flight connection handler goroutines have finished — they may be in the middle of processing a request. Together, they ensure: (a) no new connections are accepted after shutdown begins, and (b) all existing connections are served to completion before the process exits or the caller returns from `Shutdown()`.

3. UDP is connectionless — each datagram is independent and carries its own source address. `ReadFromUDP` returns a `*net.UDPAddr` alongside the payload, telling the server where to send the reply. Without it, the server would only have the payload and no way to address the response. Each `WriteTo` call must specify the destination address explicitly. This contrasts with TCP where `Accept` returns a `net.Conn` already bound to a single remote address — the connection object itself encodes the destination.

4. `net.Listen` + `Accept`: the server side. `Listen` allocates a local address and port, registers with the OS, and puts the socket in the listening state. `Accept` blocks waiting for incoming connections from clients. The client sends a SYN; `Accept` completes the 3-way handshake and returns a `net.Conn` representing that specific connection. `net.Dial`: the client side. `Dial` initiates the 3-way handshake to a specific server address. After `Dial` returns, both sides have a bidirectional `net.Conn`.

5. A Go goroutine starts with a ~2 KB stack (growable) and is multiplexed by the Go scheduler onto OS threads. Creating and parking 10,000 goroutines is routine. In languages that map threads 1:1 to OS threads (Java without virtual threads, C, C++), each connection requires an OS thread with a ~1 MB default stack — 10,000 connections consume ~10 GB of memory just for stacks, and context-switching thousands of OS threads is expensive. Go's M:N scheduling lets a small pool of OS threads service many goroutines, making goroutine-per-connection practical up to hundreds of thousands of concurrent connections on ordinary hardware.
