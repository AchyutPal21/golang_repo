// FILE: book/part5_building_backends/chapter75_websockets/examples/01_websocket_basics/main.go
// CHAPTER: 75 — WebSockets, SSE, and Long-Lived Connections
// TOPIC: WebSocket fundamentals — full-duplex connection, message types,
//        ping/pong heartbeat, graceful close, and SSE (Server-Sent Events).
//        Simulated in-process (no HTTP server) to focus on the patterns.
//
// Run (from the chapter folder):
//   go run ./examples/01_websocket_basics

package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// SIMULATED WEBSOCKET CONNECTION
// ─────────────────────────────────────────────────────────────────────────────

type MessageType int

const (
	MessageText   MessageType = 1
	MessageBinary MessageType = 2
	MessagePing   MessageType = 9
	MessagePong   MessageType = 10
	MessageClose  MessageType = 8
)

func (t MessageType) String() string {
	switch t {
	case MessageText:
		return "text"
	case MessageBinary:
		return "binary"
	case MessagePing:
		return "ping"
	case MessagePong:
		return "pong"
	case MessageClose:
		return "close"
	default:
		return "unknown"
	}
}

type Message struct {
	Type    MessageType
	Payload []byte
}

type Conn struct {
	id       string
	toServer chan Message
	toClient chan Message
	closed   chan struct{}
	once     sync.Once
}

func newConn(id string) *Conn {
	return &Conn{
		id:       id,
		toServer: make(chan Message, 16),
		toClient: make(chan Message, 16),
		closed:   make(chan struct{}),
	}
}

func (c *Conn) Send(msgType MessageType, payload []byte) error {
	select {
	case <-c.closed:
		return fmt.Errorf("connection closed")
	case c.toClient <- Message{Type: msgType, Payload: payload}:
		return nil
	}
}

func (c *Conn) Receive() (Message, bool) {
	select {
	case <-c.closed:
		return Message{}, false
	case msg := <-c.toServer:
		return msg, true
	}
}

func (c *Conn) ClientSend(msgType MessageType, payload []byte) {
	select {
	case <-c.closed:
	case c.toServer <- Message{Type: msgType, Payload: payload}:
	}
}

func (c *Conn) ClientReceive() (Message, bool) {
	select {
	case <-c.closed:
		return Message{}, false
	case msg := <-c.toClient:
		return msg, true
	}
}

func (c *Conn) Close() {
	c.once.Do(func() { close(c.closed) })
}

// ─────────────────────────────────────────────────────────────────────────────
// SSE (Server-Sent Events)
// One-directional: server pushes to client over HTTP/1.1 chunked or HTTP/2.
// ─────────────────────────────────────────────────────────────────────────────

type SSEEvent struct {
	ID    string
	Event string // event type (optional)
	Data  string
	Retry int // reconnect delay ms (optional)
}

func (e SSEEvent) Format() string {
	var out string
	if e.ID != "" {
		out += fmt.Sprintf("id: %s\n", e.ID)
	}
	if e.Event != "" {
		out += fmt.Sprintf("event: %s\n", e.Event)
	}
	out += fmt.Sprintf("data: %s\n", e.Data)
	if e.Retry > 0 {
		out += fmt.Sprintf("retry: %d\n", e.Retry)
	}
	return out + "\n"
}

type SSEStream struct {
	events chan SSEEvent
	done   chan struct{}
}

func NewSSEStream() *SSEStream {
	return &SSEStream{events: make(chan SSEEvent, 32), done: make(chan struct{})}
}

func (s *SSEStream) Send(e SSEEvent) {
	select {
	case <-s.done:
	case s.events <- e:
	}
}

func (s *SSEStream) Close() { close(s.done) }

func (s *SSEStream) Stream(ctx context.Context, write func(string)) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		case e := <-s.events:
			write(e.Format())
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// HEARTBEAT / PING-PONG
// ─────────────────────────────────────────────────────────────────────────────

type HeartbeatConfig struct {
	Interval     time.Duration
	WriteTimeout time.Duration
}

func runHeartbeat(ctx context.Context, conn *Conn, cfg HeartbeatConfig, onTimeout func()) {
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := conn.Send(MessagePing, []byte("ping")); err != nil {
				onTimeout()
				return
			}
			fmt.Printf("  [heartbeat] sent ping to %s\n", conn.id)
			// Expect pong within WriteTimeout.
			timer := time.NewTimer(cfg.WriteTimeout)
			select {
			case msg, ok := <-conn.toServer:
				timer.Stop()
				if !ok || msg.Type != MessagePong {
					onTimeout()
					return
				}
				fmt.Printf("  [heartbeat] received pong from %s\n", conn.id)
			case <-timer.C:
				fmt.Printf("  [heartbeat] timeout waiting for pong from %s\n", conn.id)
				onTimeout()
				return
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// GRACEFUL CLOSE
// ─────────────────────────────────────────────────────────────────────────────

type CloseCode int

const (
	CloseNormal         CloseCode = 1000
	CloseGoingAway      CloseCode = 1001
	CloseProtocolError  CloseCode = 1002
	CloseUnsupported    CloseCode = 1003
)

func gracefulClose(conn *Conn, code CloseCode, reason string) {
	payload := []byte(fmt.Sprintf("%d:%s", code, reason))
	_ = conn.Send(MessageClose, payload)
	conn.Close()
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("=== WebSocket Basics (in-process simulation) ===")
	fmt.Println()

	// ── TEXT MESSAGE EXCHANGE ─────────────────────────────────────────────────
	fmt.Println("--- Text message exchange ---")
	conn := newConn("client-1")

	var wg sync.WaitGroup
	// Server goroutine: receive and respond.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			msg, ok := conn.Receive()
			if !ok {
				return
			}
			if msg.Type == MessageClose {
				fmt.Printf("  [server] received close from %s\n", conn.id)
				return
			}
			response := fmt.Sprintf("echo: %s", msg.Payload)
			conn.Send(MessageText, []byte(response))
		}
	}()

	// Client: send messages.
	conn.ClientSend(MessageText, []byte("hello server"))
	conn.ClientSend(MessageText, []byte("how are you?"))

	// Read responses.
	for i := 0; i < 2; i++ {
		msg, _ := conn.ClientReceive()
		fmt.Printf("  [client] received: %s\n", msg.Payload)
	}
	gracefulClose(conn, CloseNormal, "done")
	wg.Wait()

	// ── HEARTBEAT ─────────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- Heartbeat (ping/pong) ---")
	conn2 := newConn("client-2")
	heartbeatCtx, heartbeatCancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer heartbeatCancel()

	// Client auto-responds to pings with pongs.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			msg, ok := conn2.ClientReceive()
			if !ok {
				return
			}
			if msg.Type == MessagePing {
				conn2.ClientSend(MessagePong, []byte("pong"))
			}
		}
	}()

	timedOut := false
	runHeartbeat(heartbeatCtx, conn2, HeartbeatConfig{
		Interval:     60 * time.Millisecond,
		WriteTimeout: 50 * time.Millisecond,
	}, func() { timedOut = true })

	conn2.Close()
	wg.Wait()
	fmt.Printf("  heartbeat timed out: %v\n", timedOut)

	// ── SSE: SERVER-SENT EVENTS ───────────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- SSE: Server-Sent Events ---")
	stream := NewSSEStream()
	sseCtx, sseCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer sseCancel()

	var received []string
	var sseMu sync.Mutex
	wg.Add(1)
	go func() {
		defer wg.Done()
		stream.Stream(sseCtx, func(data string) {
			sseMu.Lock()
			received = append(received, data)
			sseMu.Unlock()
			fmt.Printf("  [client] SSE: %s", data)
		})
	}()

	stream.Send(SSEEvent{ID: "1", Event: "order-placed", Data: `{"orderID":"o-1"}`, Retry: 3000})
	stream.Send(SSEEvent{ID: "2", Event: "order-shipped", Data: `{"orderID":"o-1","tracking":"1Z999"}`})
	stream.Send(SSEEvent{ID: "3", Data: "heartbeat"})

	time.Sleep(20 * time.Millisecond)
	stream.Close()
	wg.Wait()

	// ── WEBSOCKET vs SSE COMPARISON ───────────────────────────────────────────
	fmt.Println()
	fmt.Println("--- WebSocket vs SSE ---")
	fmt.Println(`
  WebSocket:
    • Full-duplex: client and server send messages independently
    • Binary and text frames
    • Requires WebSocket-aware proxies/load balancers
    • Browser: new WebSocket("wss://host/ws")
    • Use for: chat, collaborative editing, multiplayer games, live dashboards

  SSE (Server-Sent Events):
    • Half-duplex: server pushes only; client uses normal HTTP requests to write
    • Text-only (UTF-8); built-in reconnection with Last-Event-ID
    • Works over plain HTTP/1.1 or HTTP/2; passes through all standard proxies
    • Browser: new EventSource("https://host/events")
    • Use for: news feeds, live scores, notification streams, log tailing

  When to use SSE over WebSocket:
    Client only reads from server; reconnection and browser support matter more
    than bidirectional framing. SSE is simpler to implement and more proxy-friendly.`)
}
